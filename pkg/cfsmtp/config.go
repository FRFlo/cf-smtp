package cfsmtp

import (
	"errors"
	"os"
	"strings"
	"time"
)

type Config struct {
	ListenAddr          string
	Hostname            string
	CloudflareToken     string
	CloudflareAccountID string
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	RequestTimeout      time.Duration
	MaxMessageBytes     int64
	Debug               bool
}

type HelloConfig struct {
	SMTPAddr string
	From     string
	To       string
	Subject  string
	TextBody string
	HTMLBody string
}

func DefaultHostname() string {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		return "localhost"
	}

	return hostname
}

func NewConfig(listenAddr, hostname, cloudflareToken, cloudflareAccountID string, readTimeout, writeTimeout, requestTimeout time.Duration, maxMessageMB int, debug bool) (Config, error) {
	config := Config{
		ListenAddr:          strings.TrimSpace(listenAddr),
		Hostname:            strings.TrimSpace(hostname),
		CloudflareToken:     strings.TrimSpace(cloudflareToken),
		CloudflareAccountID: strings.TrimSpace(cloudflareAccountID),
		ReadTimeout:         readTimeout,
		WriteTimeout:        writeTimeout,
		RequestTimeout:      requestTimeout,
		MaxMessageBytes:     int64(maxMessageMB) * 1024 * 1024,
		Debug:               debug,
	}

	if config.ListenAddr == "" {
		return Config{}, errors.New("listen address is required")
	}
	if config.Hostname == "" {
		config.Hostname = DefaultHostname()
	}
	if config.CloudflareToken == "" {
		return Config{}, errors.New("cloudflare API token is required")
	}
	if config.CloudflareAccountID == "" {
		return Config{}, errors.New("cloudflare account ID is required")
	}
	if config.ReadTimeout <= 0 {
		return Config{}, errors.New("read timeout must be greater than zero")
	}
	if config.WriteTimeout <= 0 {
		return Config{}, errors.New("write timeout must be greater than zero")
	}
	if config.RequestTimeout <= 0 {
		return Config{}, errors.New("request timeout must be greater than zero")
	}
	if config.MaxMessageBytes <= 0 {
		return Config{}, errors.New("max message size must be greater than zero")
	}

	return config, nil
}

func NewHelloConfig(smtpAddr, from, to, subject, textBody, htmlBody string) (HelloConfig, error) {
	config := HelloConfig{
		SMTPAddr: strings.TrimSpace(smtpAddr),
		From:     strings.TrimSpace(from),
		To:       strings.TrimSpace(to),
		Subject:  strings.TrimSpace(subject),
		TextBody: textBody,
		HTMLBody: htmlBody,
	}

	if config.SMTPAddr == "" {
		return HelloConfig{}, errors.New("SMTP relay address is required")
	}
	if config.From == "" {
		return HelloConfig{}, errors.New("from header is required")
	}
	if config.To == "" {
		return HelloConfig{}, errors.New("recipient address is required")
	}
	if config.Subject == "" {
		return HelloConfig{}, errors.New("subject is required")
	}
	if config.TextBody == "" && config.HTMLBody == "" {
		return HelloConfig{}, errors.New("at least one message body must be provided")
	}

	return config, nil
}
