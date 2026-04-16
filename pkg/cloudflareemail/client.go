package cloudflareemail

import (
	"context"
	"fmt"

	"github.com/FRFlo/cf-smtp/pkg/emailsending/entities"
	cloudflare "github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/email_sending"
	"github.com/cloudflare/cloudflare-go/v6/option"
)

type Client struct {
	accountID string
	client    *cloudflare.Client
}

func New(accountID, apiToken string) *Client {
	return &Client{
		accountID: accountID,
		client: cloudflare.NewClient(
			option.WithAPIToken(apiToken),
			option.WithMaxRetries(0),
		),
	}
}

func (c *Client) SendRaw(ctx context.Context, from string, recipients []string, mimeMessage []byte) (*entities.SendResult, error) {
	response, err := c.client.EmailSending.SendRaw(ctx, email_sending.EmailSendingSendRawParams{
		AccountID:   cloudflare.F(c.accountID),
		From:        cloudflare.F(from),
		MimeMessage: cloudflare.F(string(mimeMessage)),
		Recipients:  cloudflare.F(recipients),
	})
	if err != nil {
		return nil, fmt.Errorf("send raw email through cloudflare: %w", err)
	}

	return &entities.SendResult{
		Delivered:        response.Delivered,
		Queued:           response.Queued,
		PermanentBounces: response.PermanentBounces,
	}, nil
}
