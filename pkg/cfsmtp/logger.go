package cfsmtp

import (
	"os"

	"github.com/rs/zerolog"
)

func NewLogger(debug bool) zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs

	level := zerolog.InfoLevel
	if debug {
		level = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(level)

	return zerolog.New(os.Stdout).With().Timestamp().Str("service", "cf-smtp").Logger()
}
