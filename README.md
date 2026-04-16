# cf-smtp

`cf-smtp` is a tiny local SMTP bridge for Cloudflare Email Service. It uses [`emersion/go-smtp`](https://github.com/emersion/go-smtp) for SMTP protocol handling, keeps the RFC 5322 message emitted by the client intact for UTF-8-safe payloads, and forwards that MIME payload to Cloudflare with the official Go SDK via `EmailSending.SendRaw`.

This matches the flow described in Cloudflare's [Email for Agents announcement](https://blog.cloudflare.com/email-for-agents/): email becomes a general interface, while Cloudflare Email Service handles actual outbound delivery.

## Repository layout

This repository now follows the APICube-style baseline from `ARCHITECTURE.md`:

```text
./
├── cmd/
│   └── cfsmtp/
│       └── root.go
├── data/
├── pkg/
│   ├── cfsmtp/
│   ├── cloudflareemail/
│   ├── emailsending/
│   │   └── entities/
│   └── smtpbridge/
├── main.go
├── go.mod
└── README.md
```

- `main.go` is now a thin launcher.
- `cmd/cfsmtp/root.go` owns CLI parsing, env-backed flags, and command dispatch.
- `pkg/cfsmtp` holds app-level bootstrap concerns such as config, logging, runtime wiring, health output, and the sample sender utility.
- `pkg/cloudflareemail` and `pkg/smtpbridge` contain the outbound Cloudflare integration and inbound SMTP bridge runtime.

## How to run the bridge

1. Enable Cloudflare Email Sending for a domain or subdomain in your account.
2. Create an API token that can use Email Service for the target account.
3. Export the required environment variables:

   ```bash
   export CLOUDFLARE_API_TOKEN="..."
   export CLOUDFLARE_ACCOUNT_ID="..."
   export SMTP_LISTEN_ADDR="127.0.0.1:2525"
   ```

4. Start the server:

   ```bash
    go run .
   ```

   The root command defaults to `serve`, so these are equivalent:

   ```bash
   go run .
   go run . serve
   ```

5. Point a local client at the bridge and send mail through it. For example:

   ```bash
   swaks --server 127.0.0.1:2525 \
     --from agent@your-domain.com \
     --to user@example.com \
     --header "Subject: Hello from cf-smtp" \
     --body "Sent through Cloudflare Email Service"
   ```

By default the bridge only listens on `127.0.0.1:2525`, which keeps it from becoming an accidental open relay on your network.

## Configuration reference

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `CLOUDFLARE_API_TOKEN` | Yes | — | API token passed to the official Cloudflare Go SDK |
| `CLOUDFLARE_ACCOUNT_ID` | Yes | — | Cloudflare account ID used for `email/sending/send_raw` |
| `SMTP_LISTEN_ADDR` | No | `127.0.0.1:2525` | Address the local SMTP server binds to |
| `SMTP_HOSTNAME` | No | machine hostname | Hostname announced in SMTP greetings |
| `SMTP_READ_TIMEOUT` | No | `30s` | Per-command socket read timeout |
| `SMTP_WRITE_TIMEOUT` | No | `30s` | Per-response socket write timeout |
| `CLOUDFLARE_REQUEST_TIMEOUT` | No | `20s` | Timeout for the Cloudflare API request |
| `SMTP_MAX_MESSAGE_MB` | No | `10` | Maximum accepted message size in MiB |
| `APP_DEBUG` | No | `false` | Enables debug-level zerolog output |

The CLI is implemented with `urfave/cli/v3`, so every daemon flag can also be provided explicitly. For example:

```bash
go run . serve \
  --listen-addr 127.0.0.1:2525 \
  --hostname relay.local \
  --cloudflare-api-token "$CLOUDFLARE_API_TOKEN" \
  --cloudflare-account-id "$CLOUDFLARE_ACCOUNT_ID"
```

## Utility commands

- Validate configuration and print structured health output:

  ```bash
  go run . healthcheck
  ```

- Send the sample rich Hello World email through the local relay:

  ```bash
  go run . send-hello
  ```

  The sample sender uses neutral defaults (`sender@example.com` -> `recipient@example.com`) so you must opt in explicitly before pointing it at real mailboxes.

## SMTP behavior

The bridge intentionally exposes a small local relay profile for agent and automation use:

- Uses `go-smtp` for SMTP session handling instead of a custom protocol loop
- Requires a valid RFC 5322 message during `DATA`
- Forwards UTF-8-safe raw MIME content to Cloudflare using `SendRaw`
- Returns `451` for transient Cloudflare send failures and `550` for permanent rejection paths
- Enforces the configured message-size limit before forwarding
- Rejects null reverse-path senders and non-UTF-8 message payloads

## Logging

The service now uses `zerolog` as its single structured logger stack. Logs are written as JSON to stdout, timestamps use Unix milliseconds, and `APP_DEBUG=true` (or `--debug`) raises the global log level for troubleshooting.

It does **not** currently support SMTP AUTH, STARTTLS, or remote relay hardening. Keep it local unless you add authentication and transport security in front of it.
