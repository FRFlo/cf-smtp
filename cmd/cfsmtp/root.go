package cfsmtpcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/FRFlo/cf-smtp/pkg/cfsmtp"
	"github.com/urfave/cli/v3"
)

func Execute(ctx context.Context, args []string) error {
	return newRootCommand().Run(ctx, args)
}

func newRootCommand() *cli.Command {
	return &cli.Command{
		Name:  "cf-smtp",
		Usage: "Run the Cloudflare SMTP bridge and related utilities",
		Flags: serveFlags(),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runServe(ctx, cmd)
		},
		Commands: []*cli.Command{
			{
				Name:   "serve",
				Usage:  "Start the local SMTP bridge",
				Flags:  serveFlags(),
				Action: runServe,
			},
			{
				Name:   "healthcheck",
				Usage:  "Validate configuration and report service health",
				Flags:  serveFlags(),
				Action: runHealthcheck,
			},
			{
				Name:   "send-hello",
				Usage:  "Send the sample rich Hello World email through a local SMTP relay",
				Flags:  helloFlags(),
				Action: runSendHello,
			},
		},
	}
}

func serveFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    "debug",
			Aliases: []string{"d"},
			Usage:   "Enable debug logging",
			Sources: cli.EnvVars("APP_DEBUG"),
		},
		&cli.StringFlag{
			Name:    "listen-addr",
			Usage:   "Local address the SMTP bridge listens on",
			Value:   "127.0.0.1:2525",
			Sources: cli.EnvVars("SMTP_LISTEN_ADDR"),
		},
		&cli.StringFlag{
			Name:    "hostname",
			Usage:   "Hostname announced in the SMTP greeting",
			Value:   cfsmtp.DefaultHostname(),
			Sources: cli.EnvVars("SMTP_HOSTNAME"),
		},
		&cli.StringFlag{
			Name:    "cloudflare-api-token",
			Usage:   "Cloudflare API token used for outbound email sending",
			Sources: cli.EnvVars("CLOUDFLARE_API_TOKEN"),
		},
		&cli.StringFlag{
			Name:    "cloudflare-account-id",
			Usage:   "Cloudflare account ID used for outbound email sending",
			Sources: cli.EnvVars("CLOUDFLARE_ACCOUNT_ID"),
		},
		&cli.DurationFlag{
			Name:    "read-timeout",
			Usage:   "Socket read timeout for SMTP sessions",
			Value:   30 * time.Second,
			Sources: cli.EnvVars("SMTP_READ_TIMEOUT"),
		},
		&cli.DurationFlag{
			Name:    "write-timeout",
			Usage:   "Socket write timeout for SMTP responses",
			Value:   30 * time.Second,
			Sources: cli.EnvVars("SMTP_WRITE_TIMEOUT"),
		},
		&cli.DurationFlag{
			Name:    "request-timeout",
			Usage:   "Timeout for Cloudflare email sending requests",
			Value:   20 * time.Second,
			Sources: cli.EnvVars("CLOUDFLARE_REQUEST_TIMEOUT"),
		},
		&cli.IntFlag{
			Name:    "max-message-mb",
			Usage:   "Maximum accepted SMTP message size in MiB",
			Value:   10,
			Sources: cli.EnvVars("SMTP_MAX_MESSAGE_MB"),
		},
	}
}

func helloFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    "debug",
			Aliases: []string{"d"},
			Usage:   "Enable debug logging",
			Sources: cli.EnvVars("APP_DEBUG"),
		},
		&cli.StringFlag{
			Name:    "smtp-addr",
			Usage:   "SMTP relay address used for the sample email",
			Value:   "127.0.0.1:2525",
			Sources: cli.EnvVars("SMTP_BRIDGE_ADDR", "SMTP_LISTEN_ADDR"),
		},
		&cli.StringFlag{
			Name:  "from",
			Usage: "Display From header used by the sample email",
			Value: "CF SMTP Example <sender@example.com>",
		},
		&cli.StringFlag{
			Name:  "to",
			Usage: "Recipient email address used by the sample email",
			Value: "recipient@example.com",
		},
		&cli.StringFlag{
			Name:  "subject",
			Usage: "Subject used by the sample email",
			Value: "Hello world",
		},
		&cli.StringFlag{
			Name:  "text-body",
			Usage: "Plain-text body used by the sample email",
			Value: "Hello world !",
		},
		&cli.StringFlag{
			Name:  "html-body",
			Usage: "HTML body used by the sample email",
			Value: "<html><body><h1>Hello world !</h1><p>Hello world !</p></body></html>",
		},
	}
}

func runServe(ctx context.Context, cmd *cli.Command) error {
	logger := cfsmtp.NewLogger(cmd.Bool("debug"))
	config, err := cfsmtp.NewConfig(
		cmd.String("listen-addr"),
		cmd.String("hostname"),
		cmd.String("cloudflare-api-token"),
		cmd.String("cloudflare-account-id"),
		cmd.Duration("read-timeout"),
		cmd.Duration("write-timeout"),
		cmd.Duration("request-timeout"),
		cmd.Int("max-message-mb"),
		cmd.Bool("debug"),
	)
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}

	if err := cfsmtp.Run(ctx, config, logger); err != nil {
		return cli.Exit(err.Error(), 1)
	}

	return nil
}

func runHealthcheck(_ context.Context, cmd *cli.Command) error {
	config, err := cfsmtp.NewConfig(
		cmd.String("listen-addr"),
		cmd.String("hostname"),
		cmd.String("cloudflare-api-token"),
		cmd.String("cloudflare-account-id"),
		cmd.Duration("read-timeout"),
		cmd.Duration("write-timeout"),
		cmd.Duration("request-timeout"),
		cmd.Int("max-message-mb"),
		cmd.Bool("debug"),
	)
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}

	status := cfsmtp.Healthcheck(config)
	if err := json.NewEncoder(cmd.Root().Writer).Encode(status); err != nil {
		return fmt.Errorf("write healthcheck output: %w", err)
	}

	return nil
}

func runSendHello(_ context.Context, cmd *cli.Command) error {
	logger := cfsmtp.NewLogger(cmd.Bool("debug"))
	helloConfig, err := cfsmtp.NewHelloConfig(
		cmd.String("smtp-addr"),
		cmd.String("from"),
		cmd.String("to"),
		cmd.String("subject"),
		cmd.String("text-body"),
		cmd.String("html-body"),
	)
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}

	if err := cfsmtp.SendHello(helloConfig, logger); err != nil {
		return cli.Exit(err.Error(), 1)
	}

	return nil
}
