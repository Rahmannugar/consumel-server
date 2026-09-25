package unit_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Rahmannugar/consumel-server/internal/authentication/templates"
)

func TestVerificationEmailUsesConsumelLayoutAndContext(t *testing.T) {
	now := time.Date(2026, time.September, 24, 12, 0, 0, 0, time.UTC)
	email, err := templates.RenderVerificationEmail("482913", now.Add(10*time.Minute), now)
	if err != nil {
		t.Fatalf("render verification email: %v", err)
	}

	for _, expected := range []string{
		"border:1px solid #dce3e8",
		"https://consumel.com/assets/logo.png",
		"Verify your email",
		"482913",
		"10 minutes",
		"© 2026 Consumel",
	} {
		if !strings.Contains(email.HTML, expected) {
			t.Fatalf("rendered HTML does not contain %q", expected)
		}
	}
	if !strings.Contains(email.Text, "482913") || !strings.Contains(email.Text, "10 minutes") {
		t.Fatalf("plain text does not contain contextual code and expiry: %q", email.Text)
	}
}
