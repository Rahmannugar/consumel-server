package templates

import (
	"bytes"
	"fmt"
	"html/template"
	"math"
	"net/url"
	"strings"
	"time"
)

var parsedPasswordReset = template.Must(
	template.New("password-reset").ParseFS(
		emailTemplates,
		"email_layout.html",
		"password_reset.html",
	),
)

type PasswordResetEmail struct {
	Subject string
	Text    string
	HTML    string
}

type passwordResetData struct {
	Title     string
	Preview   string
	ResetURL  string
	ExpiresIn string
	Year      int
}

func RenderPasswordResetEmail(resetURL string, expiresAt, now time.Time) (PasswordResetEmail, error) {
	resetURL = strings.TrimSpace(resetURL)
	parsedURL, err := url.Parse(resetURL)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return PasswordResetEmail{}, fmt.Errorf("password reset URL must be an HTTP or HTTPS URL")
	}
	remaining := expiresAt.Sub(now)
	if remaining <= 0 {
		return PasswordResetEmail{}, fmt.Errorf("password reset expiry must be in the future")
	}
	hours := int(math.Ceil(remaining.Hours()))
	expiresIn := fmt.Sprintf("%d hours", hours)
	if hours == 1 {
		expiresIn = "1 hour"
	}

	data := passwordResetData{
		Title:     "Reset your Consumel password",
		Preview:   "Choose a new password for your Consumel account.",
		ResetURL:  resetURL,
		ExpiresIn: expiresIn,
		Year:      now.UTC().Year(),
	}
	var rendered bytes.Buffer
	if err := parsedPasswordReset.ExecuteTemplate(&rendered, "email-layout", data); err != nil {
		return PasswordResetEmail{}, fmt.Errorf("render password reset email: %w", err)
	}

	return PasswordResetEmail{
		Subject: "Reset your Consumel password",
		Text: fmt.Sprintf(
			"Reset your Consumel password using this secure link: %s\n\nThis link expires in %s and can be used once. If you didn't request it, you can safely ignore this email.",
			resetURL,
			expiresIn,
		),
		HTML: rendered.String(),
	}, nil
}
