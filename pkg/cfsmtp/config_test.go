package cfsmtp

import (
	"testing"
	"time"
)

func TestNewConfigBuildsValidatedConfig(t *testing.T) {
	t.Parallel()

	config, err := NewConfig(
		"127.0.0.1:2525",
		"relay.local",
		"token",
		"account",
		30*time.Second,
		30*time.Second,
		20*time.Second,
		10,
		true,
	)
	if err != nil {
		t.Fatalf("NewConfig() error = %v", err)
	}
	if config.MaxMessageBytes != 10*1024*1024 {
		t.Fatalf("MaxMessageBytes = %d", config.MaxMessageBytes)
	}
	if !config.Debug {
		t.Fatalf("Debug = false, want true")
	}
}

func TestNewConfigRejectsMissingCloudflareToken(t *testing.T) {
	t.Parallel()

	_, err := NewConfig(
		"127.0.0.1:2525",
		"relay.local",
		"",
		"account",
		30*time.Second,
		30*time.Second,
		20*time.Second,
		10,
		false,
	)
	if err == nil {
		t.Fatalf("expected error for missing Cloudflare token")
	}
}

func TestHealthcheckReportsExpectedShape(t *testing.T) {
	t.Parallel()

	status := Healthcheck(Config{
		ListenAddr:      "127.0.0.1:2525",
		Hostname:        "relay.local",
		MaxMessageBytes: 15 * 1024 * 1024,
	})

	if status.Status != "ok" {
		t.Fatalf("status = %q", status.Status)
	}
	if status.Service != "cf-smtp" {
		t.Fatalf("service = %q", status.Service)
	}
	if status.MaxMessageMB != 15 {
		t.Fatalf("max message MB = %d", status.MaxMessageMB)
	}
}
