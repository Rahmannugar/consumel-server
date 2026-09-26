package clients

import (
	"context"
	"fmt"
	"net/mail"
	"strings"

	"github.com/resend/resend-go/v2"
)

type ResendEmailService interface {
	SendWithOptions(
		context.Context,
		*resend.SendEmailRequest,
		*resend.SendEmailOptions,
	) (*resend.SendEmailResponse, error)
}

type ResendEmailClient struct {
	emails ResendEmailService
	from   string
}

func NewResendEmailClient(emails ResendEmailService, from string) (*ResendEmailClient, error) {
	if emails == nil {
		return nil, fmt.Errorf("Resend email service is required")
	}
	from = strings.TrimSpace(from)
	if _, err := mail.ParseAddress(from); err != nil {
		return nil, fmt.Errorf("parse Resend sender: %w", err)
	}
	return &ResendEmailClient{emails: emails, from: from}, nil
}

func (client *ResendEmailClient) Send(
	ctx context.Context,
	recipient, subject, textBody, htmlBody, idempotencyKey string,
) (string, error) {
	response, err := client.emails.SendWithOptions(ctx, &resend.SendEmailRequest{
		From: client.from, To: []string{recipient}, Subject: subject, Text: textBody, Html: htmlBody,
	}, &resend.SendEmailOptions{IdempotencyKey: idempotencyKey})
	if err != nil {
		return "", fmt.Errorf("send email through Resend: %w", err)
	}
	return response.Id, nil
}
