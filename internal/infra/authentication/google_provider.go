package authentication

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/Rahmannugar/authlier/googleoauth"
	"golang.org/x/oauth2"
)

type observedGoogleProvider struct {
	provider googleoauth.Provider
	logger   *slog.Logger
}

func NewObservedGoogleProvider(
	provider googleoauth.Provider,
	logger *slog.Logger,
) googleoauth.Provider {
	return &observedGoogleProvider{provider: provider, logger: logger}
}

func (provider *observedGoogleProvider) Exchange(
	ctx context.Context,
	input googleoauth.ExchangeInput,
) (googleoauth.Identity, error) {
	startedAt := time.Now()
	identity, err := provider.provider.Exchange(ctx, input)
	if err == nil {
		return identity, nil
	}

	// OAuth inputs and provider response bodies can contain credentials. Record
	// only a bounded category so the failure is diagnosable without leaking them.
	provider.logger.ErrorContext(ctx, "Google identity exchange failed",
		"event", "authentication.google.exchange_failed",
		"operation", "authentication.google.exchange",
		"outcome", "error",
		"error_category", googleExchangeErrorCategory(err),
		"duration_ms", time.Since(startedAt).Milliseconds(),
	)
	return googleoauth.Identity{}, err
}

func googleExchangeErrorCategory(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "request_canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "provider_timeout"
	case errors.Is(err, googleoauth.ErrInvalidRecord):
		return "invalid_provider_identity"
	}

	var oauthError *oauth2.RetrieveError
	if errors.As(err, &oauthError) {
		if oauthError.Response != nil && oauthError.Response.StatusCode == http.StatusTooManyRequests {
			return "provider_rate_limited"
		}
		return "authorization_code_rejected"
	}

	var networkError net.Error
	if errors.As(err, &networkError) {
		if networkError.Timeout() {
			return "provider_timeout"
		}
		return "provider_unavailable"
	}
	return "provider_exchange_failed"
}

var _ googleoauth.Provider = (*observedGoogleProvider)(nil)
