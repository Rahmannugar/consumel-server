package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	resendEmailsEndpoint = "https://api.resend.com/emails"
	maximumErrorBody     = 64 << 10
	// V1 has one worker instance. Keep its request starts below Resend's default
	// account-wide limit; add shared pacing before horizontally scaling workers.
	resendRequestSpacing = 125 * time.Millisecond
)

type ResendEmailClient struct {
	httpClient *http.Client
	apiKey     string
	from       string
	endpoint   string
	rateMu     sync.Mutex
	nextSendAt time.Time
}

func NewResendEmailClient(httpClient *http.Client, apiKey, from string) (*ResendEmailClient, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("Resend HTTP client is required")
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("Resend API key is required")
	}
	from = strings.TrimSpace(from)
	if _, err := mail.ParseAddress(from); err != nil {
		return nil, fmt.Errorf("parse Resend sender: %w", err)
	}
	return &ResendEmailClient{
		httpClient: httpClient, apiKey: apiKey, from: from, endpoint: resendEmailsEndpoint,
	}, nil
}

func (client *ResendEmailClient) Send(
	ctx context.Context,
	recipient, subject, textBody, htmlBody, idempotencyKey string,
) (string, error) {
	if err := client.waitForSendSlot(ctx); err != nil {
		return "", newResendFailure(err, true, 0)
	}
	payload, err := json.Marshal(map[string]any{
		"from": client.from, "to": []string{recipient}, "subject": subject,
		"text": textBody, "html": htmlBody,
	})
	if err != nil {
		return "", newResendFailure(fmt.Errorf("encode Resend email: %w", err), false, 0)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", newResendFailure(fmt.Errorf("create Resend request: %w", err), false, 0)
	}
	request.Header.Set("Authorization", "Bearer "+client.apiKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", idempotencyKey)

	response, err := client.httpClient.Do(request)
	if err != nil {
		return "", newResendFailure(fmt.Errorf("send email through Resend: %w", err), true, 0)
	}
	defer response.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(response.Body, maximumErrorBody))
	if readErr != nil {
		return "", newResendFailure(fmt.Errorf("read Resend response: %w", readErr), true, 0)
	}
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		var sent struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(body, &sent); err != nil || strings.TrimSpace(sent.ID) == "" {
			return "", newResendFailure(fmt.Errorf("decode successful Resend response"), true, 0)
		}
		return sent.ID, nil
	}

	var providerError struct {
		Name    string `json:"name"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(body, &providerError)
	retryable := response.StatusCode == http.StatusRequestTimeout ||
		response.StatusCode == http.StatusTooEarly ||
		response.StatusCode == http.StatusTooManyRequests ||
		response.StatusCode >= http.StatusInternalServerError ||
		(response.StatusCode == http.StatusConflict && providerError.Name == "concurrent_idempotent_requests")
	retryAfter := parseRetryAfter(response.Header.Get("Retry-After"), time.Now().UTC())
	return "", newResendFailure(
		fmt.Errorf("Resend rejected email: status=%d code=%s", response.StatusCode, boundedCode(providerError.Name)),
		retryable,
		retryAfter,
	)
}

func (client *ResendEmailClient) waitForSendSlot(ctx context.Context) error {
	client.rateMu.Lock()
	now := time.Now()
	reservedAt := client.nextSendAt
	if reservedAt.Before(now) {
		reservedAt = now
	}
	client.nextSendAt = reservedAt.Add(resendRequestSpacing)
	client.rateMu.Unlock()
	if delay := time.Until(reservedAt); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

type resendFailure struct {
	err        error
	retryable  bool
	retryAfter time.Duration
}

func newResendFailure(err error, retryable bool, retryAfter time.Duration) *resendFailure {
	return &resendFailure{err: err, retryable: retryable, retryAfter: retryAfter}
}

func (failure *resendFailure) Error() string             { return failure.err.Error() }
func (failure *resendFailure) Unwrap() error             { return failure.err }
func (failure *resendFailure) Retryable() bool           { return failure.retryable }
func (failure *resendFailure) RetryAfter() time.Duration { return failure.retryAfter }

func parseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(value); err == nil && at.After(now) {
		return at.Sub(now)
	}
	return 0
}

func boundedCode(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	if len(value) > 100 {
		return value[:100]
	}
	return value
}

var _ interface {
	Retryable() bool
	RetryAfter() time.Duration
} = (*resendFailure)(nil)
