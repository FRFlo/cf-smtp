package main

import (
	"context"
	"os"

	cfsmtpcmd "github.com/FRFlo/cf-smtp/cmd/cfsmtp"
	"github.com/FRFlo/cf-smtp/pkg/cfsmtp"
)

func main() {
	if err := cfsmtpcmd.Execute(context.Background(), os.Args); err != nil {
		logger := cfsmtp.NewLogger(false)
		logger.Error().Err(err).Msg("command failed")
		os.Exit(1)
	}
}
