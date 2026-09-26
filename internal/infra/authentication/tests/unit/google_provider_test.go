package unit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/Rahmannugar/authlier/googleoauth"
	infraauthentication "github.com/Rahmannugar/consumel-server/internal/infra/authentication"
)

func TestObservedGoogleProviderLogsBoundedFailureWithoutOAuthInput(t *testing.T) {
	var output bytes.Buffer
	provider := infraauthentication.NewObservedGoogleProvider(
		failingGoogleProvider{err: context.DeadlineExceeded},
		slog.New(slog.NewJSONHandler(&output, nil)),
	)

	_, err := provider.Exchange(t.Context(), googleoauth.ExchangeInput{
		Code:         "sensitive-authorization-code",
		CodeVerifier: "sensitive-code-verifier",
		Audience:     "sensitive-client-id",
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("exchange error = %v, want context deadline exceeded", err)
	}

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("decode log entry: %v", err)
	}
	if entry["error_category"] != "provider_timeout" {
		t.Fatalf("error category = %v, want provider_timeout", entry["error_category"])
	}
	for _, sensitive := range []string{
		"sensitive-authorization-code",
		"sensitive-code-verifier",
		"sensitive-client-id",
	} {
		if bytes.Contains(output.Bytes(), []byte(sensitive)) {
			t.Fatalf("log contains sensitive OAuth input %q", sensitive)
		}
	}
}

type failingGoogleProvider struct {
	err error
}

func (provider failingGoogleProvider) Exchange(
	context.Context,
	googleoauth.ExchangeInput,
) (googleoauth.Identity, error) {
	return googleoauth.Identity{}, provider.err
}
