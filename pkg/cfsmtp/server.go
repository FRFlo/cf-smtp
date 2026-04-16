package cfsmtp

import (
	"context"
	"errors"
	"fmt"

	"github.com/FRFlo/cf-smtp/pkg/cloudflareemail"
	"github.com/FRFlo/cf-smtp/pkg/smtpbridge"
	"github.com/rs/zerolog"
)

func Run(ctx context.Context, config Config, logger zerolog.Logger) error {
	bridgeLogger := logger.With().Str("component", "smtp-bridge").Logger()
	sender := cloudflareemail.New(config.CloudflareAccountID, config.CloudflareToken)
	server := smtpbridge.New(smtpbridge.Config{
		ListenAddr:      config.ListenAddr,
		Hostname:        config.Hostname,
		ReadTimeout:     config.ReadTimeout,
		WriteTimeout:    config.WriteTimeout,
		RequestTimeout:  config.RequestTimeout,
		MaxMessageBytes: config.MaxMessageBytes,
	}, sender, bridgeLogger)

	logger.Info().Str("listen_addr", config.ListenAddr).Str("hostname", config.Hostname).Msg("starting smtp bridge")
	if err := server.Serve(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("serve smtp bridge: %w", err)
	}
	logger.Info().Msg("smtp bridge stopped")

	return nil
}
