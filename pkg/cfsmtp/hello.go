package cfsmtp

import (
	"fmt"
	"net"
	"strconv"

	"github.com/rs/zerolog"
	gomail "github.com/wneessen/go-mail"
)

func SendHello(config HelloConfig, logger zerolog.Logger) error {
	host, portValue, err := net.SplitHostPort(config.SMTPAddr)
	if err != nil {
		return fmt.Errorf("split SMTP relay address: %w", err)
	}

	port, err := strconv.Atoi(portValue)
	if err != nil {
		return fmt.Errorf("parse SMTP relay port: %w", err)
	}

	message := gomail.NewMsg()
	if err := message.From(config.From); err != nil {
		return fmt.Errorf("set From header: %w", err)
	}
	if err := message.To(config.To); err != nil {
		return fmt.Errorf("set To header: %w", err)
	}
	message.Subject(config.Subject)

	if config.TextBody != "" {
		message.SetBodyString(gomail.TypeTextPlain, config.TextBody)
	}
	if config.HTMLBody != "" {
		if config.TextBody == "" {
			message.SetBodyString(gomail.TypeTextHTML, config.HTMLBody)
		} else {
			message.AddAlternativeString(gomail.TypeTextHTML, config.HTMLBody)
		}
	}

	client, err := gomail.NewClient(
		host,
		gomail.WithPort(port),
		gomail.WithTLSPortPolicy(gomail.NoTLS),
		gomail.WithSMTPAuth(gomail.SMTPAuthNoAuth),
	)
	if err != nil {
		return fmt.Errorf("build SMTP client: %w", err)
	}
	defer client.Close()

	if err := client.DialAndSend(message); err != nil {
		return fmt.Errorf("send hello email: %w", err)
	}

	logger.Info().Str("smtp_addr", config.SMTPAddr).Str("to", config.To).Msg("sent hello email")
	return nil
}
