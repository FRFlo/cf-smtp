package smtpbridge

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/FRFlo/cf-smtp/pkg/emailsending/entities"
	cloudflare "github.com/cloudflare/cloudflare-go/v6"
	gosmtp "github.com/emersion/go-smtp"
	"github.com/rs/zerolog"
)

type Sender interface {
	SendRaw(ctx context.Context, from string, recipients []string, mimeMessage []byte) (*entities.SendResult, error)
}

type Config struct {
	ListenAddr      string
	Hostname        string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	RequestTimeout  time.Duration
	MaxMessageBytes int64
}

type Server struct {
	config Config
	sender Sender
	logger zerolog.Logger
}

func New(config Config, sender Sender, logger zerolog.Logger) *Server {
	return &Server{config: config, sender: sender, logger: logger}
}

func (s *Server) Serve(ctx context.Context) error {
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", s.config.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.config.ListenAddr, err)
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			s.logger.Error().Err(err).Msg("error closing listener")
		}
	}(listener)

	backend := &backend{parentCtx: ctx, server: s}
	smtpServer := gosmtp.NewServer(backend)
	smtpServer.Addr = s.config.ListenAddr
	smtpServer.Domain = s.config.Hostname
	smtpServer.ReadTimeout = s.config.ReadTimeout
	smtpServer.WriteTimeout = s.config.WriteTimeout
	smtpServer.MaxMessageBytes = s.config.MaxMessageBytes
	smtpServer.AllowInsecureAuth = false

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout())
		defer cancel()
		if err := smtpServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			s.logger.Error().Err(err).Msg("smtp shutdown error")
		}
	}()

	err = smtpServer.Serve(listener)
	if err == nil || ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
		return nil
	}

	return fmt.Errorf("serve smtp bridge: %w", err)
}

func (s *Server) shutdownTimeout() time.Duration {
	timeout := s.config.RequestTimeout
	if timeout < s.config.ReadTimeout {
		timeout = s.config.ReadTimeout
	}
	if timeout < s.config.WriteTimeout {
		timeout = s.config.WriteTimeout
	}
	if timeout <= 0 {
		return 5 * time.Second
	}
	return timeout
}

type backend struct {
	parentCtx context.Context
	server    *Server
}

func (b *backend) NewSession(_ *gosmtp.Conn) (gosmtp.Session, error) {
	return &session{
		parentCtx: b.parentCtx,
		server:    b.server,
	}, nil
}

type session struct {
	parentCtx  context.Context
	server     *Server
	from       string
	recipients []string
}

func (s *session) Mail(from string, _ *gosmtp.MailOptions) error {
	s.Reset()
	if strings.TrimSpace(from) == "" {
		return smtpError(553, 5, 1, 7, "Null reverse-path is not supported by this relay")
	}

	address, err := mail.ParseAddress(from)
	if err != nil {
		return smtpError(501, 5, 1, 7, "Bad sender address syntax")
	}

	s.from = address.Address
	return nil
}

func (s *session) Rcpt(to string, _ *gosmtp.RcptOptions) error {
	address, err := mail.ParseAddress(to)
	if err != nil {
		return smtpError(501, 5, 1, 3, "Bad recipient address syntax")
	}

	recipient := address.Address
	for _, existing := range s.recipients {
		if strings.EqualFold(existing, recipient) {
			return nil
		}
	}

	s.recipients = append(s.recipients, recipient)
	return nil
}

func (s *session) Data(r io.Reader) error {
	defer s.Reset()

	rawMessage, err := io.ReadAll(r)
	if err != nil {
		return smtpError(451, 4, 3, 0, "Failed to read message data")
	}
	if !utf8.Valid(rawMessage) {
		return smtpError(554, 5, 6, 0, "Message contains non-UTF-8 data unsupported by this relay")
	}
	if _, err := mail.ReadMessage(bytes.NewReader(rawMessage)); err != nil {
		return smtpError(554, 5, 6, 0, "Message content rejected: invalid RFC 5322 headers")
	}

	requestCtx, cancel := context.WithTimeout(s.parentCtx, s.server.config.RequestTimeout)
	result, err := s.server.sender.SendRaw(requestCtx, s.from, append([]string(nil), s.recipients...), rawMessage)
	cancel()
	if err != nil {
		smtpErr, logMessage := smtpErrorFromSendFailure(err)
		s.server.logger.Warn().Int("recipient_count", len(s.recipients)).Str("details", logMessage).Msg("cloudflare send failed")
		return smtpErr
	}
	if len(result.Delivered) == 0 && len(result.Queued) == 0 && len(result.PermanentBounces) > 0 {
		s.server.logger.Warn().Int("bounced_count", len(result.PermanentBounces)).Msg("cloudflare rejected all recipients")
		return smtpError(550, 5, 1, 1, "Cloudflare permanently rejected all recipients")
	}

	// Cloudflare don't return Delivered when it's succesful, idk why.
	//if len(result.Delivered) == 0 && len(result.Queued) == 0 && len(result.PermanentBounces) == 0 {
	//	s.server.logger.Warn().Int("recipient_count", len(s.recipients)).Msg("cloudflare send returned no delivery outcome")
	//	return smtpError(451, 4, 4, 0, "Cloudflare email send produced no delivery outcome")
	//}

	s.server.logger.Info().Int("delivered", len(result.Delivered)).Int("queued", len(result.Queued)).Int("bounced", len(result.PermanentBounces)).Msg("cloudflare accepted message")
	return nil
}

func (s *session) Reset() {
	s.from = ""
	s.recipients = nil
}

func (s *session) Logout() error {
	return nil
}

func smtpError(code int, class, subject, detail int, message string) error {
	return &gosmtp.SMTPError{
		Code:         code,
		EnhancedCode: gosmtp.EnhancedCode{class, subject, detail},
		Message:      message,
	}
}

func smtpErrorFromSendFailure(err error) (error, string) {
	var apiErr *cloudflare.Error
	if errors.As(err, &apiErr) {
		logMessage := fmt.Sprintf("status=%d api_error_count=%d", apiErr.StatusCode, len(apiErr.Errors))
		if apiErr.StatusCode >= 500 || apiErr.StatusCode == 408 || apiErr.StatusCode == 409 || apiErr.StatusCode == 429 {
			return smtpError(451, 4, 4, 0, "Cloudflare email send failed"), logMessage
		}
		if apiErr.StatusCode >= 400 {
			return smtpError(550, 5, 7, 1, "Cloudflare permanently rejected the message"), logMessage
		}
	}

	return smtpError(451, 4, 4, 0, "Cloudflare email send failed"), "transport_error=true"
}
