package unit_test

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Rahmannugar/consumel-server/internal/infra/clients"
	"github.com/Rahmannugar/consumel-server/internal/infra/emaildelivery"
)

func TestResendEmailClientClassifiesProviderResponses(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		body          string
		retryAfter    string
		wantRetryable bool
		wantDelay     time.Duration
	}{
		{
			name: "rate limit", status: http.StatusTooManyRequests,
			body:       `{"name":"rate_limit_exceeded","message":"slow down"}`,
			retryAfter: "3", wantRetryable: true, wantDelay: 3 * time.Second,
		},
		{
			name: "invalid recipient", status: http.StatusUnprocessableEntity,
			body:          `{"name":"validation_error","message":"invalid recipient"}`,
			wantRetryable: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if got := request.Header.Get("Idempotency-Key"); got != "delivery-1" {
					t.Fatalf("idempotency key = %q", got)
				}
				return &http.Response{
					StatusCode: test.status,
					Header:     http.Header{"Retry-After": []string{test.retryAfter}},
					Body:       io.NopCloser(strings.NewReader(test.body)),
					Request:    request,
				}, nil
			})
			client, err := clients.NewResendEmailClient(
				&http.Client{Transport: transport}, "re_test", "Consumel <noreply@consumel.com>",
			)
			if err != nil {
				t.Fatalf("create Resend client: %v", err)
			}
			_, err = client.Send(t.Context(), "developer@example.com", "Subject", "Text", "<p>HTML</p>", "delivery-1")
			if err == nil {
				t.Fatal("send email succeeded unexpectedly")
			}
			var failure emaildelivery.SendFailure
			if !errors.As(err, &failure) {
				t.Fatalf("error does not expose delivery policy: %v", err)
			}
			if got := failure.Retryable(); got != test.wantRetryable {
				t.Fatalf("retryable = %t, want %t", got, test.wantRetryable)
			}
			if got := failure.RetryAfter(); got != test.wantDelay {
				t.Fatalf("retry delay = %s, want %s", got, test.wantDelay)
			}
		})
	}
}

func TestResendEmailClientReturnsProviderMessageID(t *testing.T) {
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"id":"email_123"}`)),
			Header:     make(http.Header),
			Request:    request,
		}, nil
	})
	client, err := clients.NewResendEmailClient(
		&http.Client{Transport: transport}, "re_test", "Consumel <noreply@consumel.com>",
	)
	if err != nil {
		t.Fatalf("create Resend client: %v", err)
	}
	providerID, err := client.Send(t.Context(), "developer@example.com", "Subject", "Text", "<p>HTML</p>", "delivery-1")
	if err != nil {
		t.Fatalf("send email: %v", err)
	}
	if providerID != "email_123" {
		t.Fatalf("provider message ID = %q", providerID)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

var _ http.RoundTripper = roundTripFunc(nil)
