package templates

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"math"
	"strings"
	"time"
)

//go:embed email_layout.html email_verification.html
var emailTemplates embed.FS

var parsedEmailVerification = template.Must(
	template.New("email-verification").ParseFS(
		emailTemplates,
		"email_layout.html",
		"email_verification.html",
	),
)

type VerificationEmail struct {
	Subject string
	Text    string
	HTML    string
}

type emailVerificationData struct {
	Title     string
	Preview   string
	Code      string
	ExpiresIn string
	Year      int
}

func RenderVerificationEmail(code string, expiresAt, now time.Time) (VerificationEmail, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return VerificationEmail{}, fmt.Errorf("verification code is required")
	}
	remaining := expiresAt.Sub(now)
	if remaining <= 0 {
		return VerificationEmail{}, fmt.Errorf("verification code expiry must be in the future")
	}
	minutes := int(math.Ceil(remaining.Minutes()))
	expiresIn := fmt.Sprintf("%d minutes", minutes)
	if minutes == 1 {
		expiresIn = "1 minute"
	}

	data := emailVerificationData{
		Title:     "Verify your Consumel account",
		Preview:   "Use this code to verify your Consumel account.",
		Code:      code,
		ExpiresIn: expiresIn,
		Year:      now.UTC().Year(),
	}
	var rendered bytes.Buffer
	if err := parsedEmailVerification.ExecuteTemplate(&rendered, "email-layout", data); err != nil {
		return VerificationEmail{}, fmt.Errorf("render verification email: %w", err)
	}

	return VerificationEmail{
		Subject: "Verify your Consumel account",
		Text: fmt.Sprintf(
			"Your Consumel verification code is %s.\n\nThis code expires in %s. Never share it; Consumel staff will not ask for it.",
			code,
			expiresIn,
		),
		HTML: rendered.String(),
	}, nil
}
