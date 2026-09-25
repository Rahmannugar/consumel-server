package clients

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/Rahmannugar/authlier/emailverification"
	"github.com/Rahmannugar/authlier/passwordreset"
	authenticationtemplates "github.com/Rahmannugar/consumel-server/internal/authentication/templates"
	"github.com/resend/resend-go/v2"
)

type ResendEmailService interface {
	SendWithContext(context.Context, *resend.SendEmailRequest) (*resend.SendEmailResponse, error)
}

type ResendAuthenticationEmailSender struct {
	emails ResendEmailService
	from   string
	now    func() time.Time
}

func NewResendAuthenticationEmailSender(
	emails ResendEmailService,
	from string,
) (*ResendAuthenticationEmailSender, error) {
	if emails == nil {
		return nil, fmt.Errorf("Resend email service is required")
	}
	from = strings.TrimSpace(from)
	if _, err := mail.ParseAddress(from); err != nil {
		return nil, fmt.Errorf("parse Resend sender: %w", err)
	}
	return &ResendAuthenticationEmailSender{emails: emails, from: from, now: time.Now}, nil
}

func (sender *ResendAuthenticationEmailSender) SendVerification(
	ctx context.Context,
	message emailverification.Message,
) error {
	email, err := authenticationtemplates.RenderVerificationEmail(
		message.Code,
		message.ExpiresAt,
		sender.now().UTC(),
	)
	if err != nil {
		return err
	}

	_, err = sender.emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    sender.from,
		To:      []string{message.Email},
		Subject: email.Subject,
		Text:    email.Text,
		Html:    email.HTML,
	})
	if err != nil {
		return fmt.Errorf("send authentication email through Resend: %w", err)
	}
	return nil
}

func (sender *ResendAuthenticationEmailSender) SendPasswordReset(
	ctx context.Context,
	message passwordreset.Message,
) error {
	email, err := authenticationtemplates.RenderPasswordResetEmail(
		message.URL,
		message.ExpiresAt,
		sender.now().UTC(),
	)
	if err != nil {
		return err
	}

	_, err = sender.emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    sender.from,
		To:      []string{message.Email},
		Subject: email.Subject,
		Text:    email.Text,
		Html:    email.HTML,
	})
	if err != nil {
		return fmt.Errorf("send authentication email through Resend: %w", err)
	}
	return nil
}

var _ emailverification.Sender = (*ResendAuthenticationEmailSender)(nil)
var _ passwordreset.Sender = (*ResendAuthenticationEmailSender)(nil)
