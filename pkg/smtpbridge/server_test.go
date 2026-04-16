package smtpbridge

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/FRFlo/cf-smtp/pkg/emailsending/entities"
	cloudflare "github.com/cloudflare/cloudflare-go/v6"
	gosmtp "github.com/emersion/go-smtp"
	"github.com/rs/zerolog"
)

type fakeSender struct {
	result     *entities.SendResult
	err        error
	from       string
	recipients []string
	payload    []byte
}

func (s *fakeSender) SendRaw(_ context.Context, from string, recipients []string, mimeMessage []byte) (*entities.SendResult, error) {
	s.from = from
	s.recipients = append([]string(nil), recipients...)
	s.payload = append([]byte(nil), mimeMessage...)
	if s.err != nil {
		return nil, s.err
	}
	if s.result == nil {
		return &entities.SendResult{}, nil
	}
	return s.result, nil
}

func TestSessionMailRejectsNullReversePath(t *testing.T) {
	t.Parallel()

	sess := newTestSession(t, &fakeSender{})
	err := sess.Mail("", nil)
	assertSMTPError(t, err, 553, [3]int{5, 1, 7})
}

func TestSessionRcptDeduplicatesRecipients(t *testing.T) {
	t.Parallel()

	sess := newTestSession(t, &fakeSender{})
	if err := sess.Mail("sender@example.com", nil); err != nil {
		t.Fatalf("Mail() error = %v", err)
	}
	if err := sess.Rcpt("recipient@example.com", nil); err != nil {
		t.Fatalf("Rcpt() first error = %v", err)
	}
	if err := sess.Rcpt("Recipient@Example.com", nil); err != nil {
		t.Fatalf("Rcpt() duplicate error = %v", err)
	}

	if len(sess.recipients) != 1 {
		t.Fatalf("recipient count = %d, want 1", len(sess.recipients))
	}
}

func TestSessionDataRejectsNonUTF8(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{}
	sess := newTestSession(t, sender)
	prepareEnvelope(t, sess)

	err := sess.Data(bytes.NewReader([]byte{0xff, 0xfe, 0xfd}))
	assertSMTPError(t, err, 554, [3]int{5, 6, 0})
	if sender.from != "" {
		t.Fatalf("sender should not have been called")
	}
	if sess.from != "" || len(sess.recipients) != 0 {
		t.Fatalf("session state was not reset after DATA failure")
	}
}

func TestSessionDataReturnsPermanentBounceFailure(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{result: &entities.SendResult{PermanentBounces: []string{"recipient@example.com"}}}
	sess := newTestSession(t, sender)
	prepareEnvelope(t, sess)

	err := sess.Data(bytes.NewReader(validMessage()))
	assertSMTPError(t, err, 550, [3]int{5, 1, 1})
	if sender.from != "sender@example.com" {
		t.Fatalf("from = %q, want %q", sender.from, "sender@example.com")
	}
	if len(sender.recipients) != 1 || sender.recipients[0] != "recipient@example.com" {
		t.Fatalf("recipients = %#v", sender.recipients)
	}
}

func TestSessionDataForwardsMessage(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{result: &entities.SendResult{Delivered: []string{"recipient@example.com"}}}
	sess := newTestSession(t, sender)
	prepareEnvelope(t, sess)

	message := validMessage()
	if err := sess.Data(bytes.NewReader(message)); err != nil {
		t.Fatalf("Data() error = %v", err)
	}
	if sender.from != "sender@example.com" {
		t.Fatalf("from = %q, want %q", sender.from, "sender@example.com")
	}
	if len(sender.recipients) != 1 || sender.recipients[0] != "recipient@example.com" {
		t.Fatalf("recipients = %#v", sender.recipients)
	}
	if !bytes.Equal(sender.payload, message) {
		t.Fatalf("payload = %q, want %q", string(sender.payload), string(message))
	}
	if sess.from != "" || len(sess.recipients) != 0 {
		t.Fatalf("session state was not reset after successful DATA")
	}
}

func TestSessionDataMapsPermanentCloudflareAPIError(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{err: &cloudflare.Error{StatusCode: http.StatusBadRequest}}
	sess := newTestSession(t, sender)
	prepareEnvelope(t, sess)

	err := sess.Data(bytes.NewReader(validMessage()))
	assertSMTPError(t, err, 550, [3]int{5, 7, 1})
}

func TestSessionDataMapsTransientCloudflareAPIError(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{err: &cloudflare.Error{StatusCode: http.StatusTooManyRequests}}
	sess := newTestSession(t, sender)
	prepareEnvelope(t, sess)

	err := sess.Data(bytes.NewReader(validMessage()))
	assertSMTPError(t, err, 451, [3]int{4, 4, 0})
}

func TestSessionDataRejectsMissingCloudflareOutcome(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{result: &entities.SendResult{}}
	sess := newTestSession(t, sender)
	prepareEnvelope(t, sess)

	err := sess.Data(bytes.NewReader(validMessage()))
	assertSMTPError(t, err, 451, [3]int{4, 4, 0})
}

func newTestSession(t *testing.T, sender Sender) *session {
	t.Helper()

	return &session{
		parentCtx: context.Background(),
		server: &Server{
			config: Config{RequestTimeout: time.Second},
			sender: sender,
			logger: zerolog.New(io.Discard),
		},
	}
}

func prepareEnvelope(t *testing.T, sess *session) {
	t.Helper()

	if err := sess.Mail("sender@example.com", &gosmtp.MailOptions{}); err != nil {
		t.Fatalf("Mail() error = %v", err)
	}
	if err := sess.Rcpt("recipient@example.com", &gosmtp.RcptOptions{}); err != nil {
		t.Fatalf("Rcpt() error = %v", err)
	}
}

func validMessage() []byte {
	return []byte("From: Sender <sender@example.com>\r\nTo: Recipient <recipient@example.com>\r\nSubject: Test\r\n\r\nHello\r\n")
}

func assertSMTPError(t *testing.T, err error, wantCode int, wantEnhanced [3]int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected SMTP error, got nil")
	}

	var smtpErr *gosmtp.SMTPError
	if !errors.As(err, &smtpErr) {
		t.Fatalf("error = %T, want *gosmtp.SMTPError", err)
	}
	if smtpErr.Code != wantCode {
		t.Fatalf("smtp code = %d, want %d", smtpErr.Code, wantCode)
	}
	gotEnhanced := [3]int{smtpErr.EnhancedCode[0], smtpErr.EnhancedCode[1], smtpErr.EnhancedCode[2]}
	if gotEnhanced != wantEnhanced {
		t.Fatalf("enhanced code = %v, want %v", gotEnhanced, wantEnhanced)
	}
}
