//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Rahmannugar/authlier/emailverification"
	"github.com/Rahmannugar/authlier/passwordreset"
	"github.com/Rahmannugar/consumel-server/internal/infra/database/testdb"
	"github.com/Rahmannugar/consumel-server/internal/infra/emaildelivery"
	"github.com/Rahmannugar/consumel-server/internal/infra/events"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestAuthenticationEmailsFlowThroughOutboxAndRedisReliably(t *testing.T) {
	pool := testdb.OpenMigratedDatabase(t)
	redisClient := openRedis(t)
	queue, err := emaildelivery.NewQueue(pool, bytes.Repeat([]byte{0x31}, 32))
	if err != nil {
		t.Fatalf("create email delivery queue: %v", err)
	}

	expiresAt := time.Now().UTC().Add(time.Hour)
	if err := queue.SendVerification(t.Context(), emailverification.Message{
		UserID: "user-verification", Email: "verify@example.com", Code: "482193", ExpiresAt: expiresAt,
	}); err != nil {
		t.Fatalf("queue verification email: %v", err)
	}
	if err := queue.SendPasswordReset(t.Context(), passwordreset.Message{
		UserID: "user-reset", Email: "reset@example.com", URL: "https://app.example.com/reset?token=secret", ExpiresAt: expiresAt,
	}); err != nil {
		t.Fatalf("queue password-reset email: %v", err)
	}

	assertEncryptedAtRest(t, pool)
	sender := newRecordingSender()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	relay, err := events.NewRelay(pool, redisClient, 4, logger)
	if err != nil {
		t.Fatalf("create outbox relay: %v", err)
	}
	consumer, err := emaildelivery.NewConsumer(pool, redisClient, queue, sender, "integration-test", 2, logger)
	if err != nil {
		t.Fatalf("create email delivery consumer: %v", err)
	}
	errorsChannel := make(chan error, 2)
	go func() { errorsChannel <- relay.Run(ctx) }()
	go func() { errorsChannel <- consumer.Run(ctx) }()

	// Both sends must enter the provider boundary before either is released.
	// This proves the configured worker slots execute deliveries concurrently.
	for range 2 {
		select {
		case <-sender.started:
		case <-time.After(10 * time.Second):
			t.Fatal("email deliveries did not execute concurrently")
		}
	}
	if maximum := sender.maximumActive(); maximum != 2 {
		t.Fatalf("maximum concurrent sends = %d, want 2", maximum)
	}
	close(sender.release)

	waitFor(t, 10*time.Second, func() bool {
		var delivered int
		if err := pool.QueryRow(t.Context(),
			"SELECT count(*) FROM email_deliveries WHERE status = 'delivered'",
		).Scan(&delivered); err != nil {
			t.Fatalf("count delivered email: %v", err)
		}
		return delivered == 2 && sender.count() == 2
	})

	// Re-publishing a stable event is possible when Redis accepted XADD but the
	// relay's PostgreSQL transaction did not commit. The delivery claim must make
	// that at-least-once transport duplicate harmless.
	duplicateOutboxEvent(t, pool, redisClient)
	waitFor(t, 5*time.Second, func() bool {
		pending, err := redisClient.XPending(t.Context(), events.StreamName, "email-deliveries").Result()
		if err != nil {
			t.Fatalf("inspect email consumer pending entries: %v", err)
		}
		return pending.Count == 0
	})
	if count := sender.count(); count != 2 {
		t.Fatalf("email send count after duplicate event = %d, want 2", count)
	}
	for _, call := range sender.callsSnapshot() {
		if !strings.HasPrefix(call.idempotencyKey, "email-delivery/") {
			t.Fatalf("idempotency key = %q", call.idempotencyKey)
		}
	}

	sender.fail("retry@example.com", testSendFailure{retryable: true, retryAfter: time.Minute})
	sender.fail("permanent@example.com", testSendFailure{retryable: false})
	if err := queue.SendVerification(t.Context(), emailverification.Message{
		UserID: "user-retry", Email: "retry@example.com", Code: "812763", ExpiresAt: expiresAt,
	}); err != nil {
		t.Fatalf("queue retryable email: %v", err)
	}
	if err := queue.SendVerification(t.Context(), emailverification.Message{
		UserID: "user-permanent", Email: "permanent@example.com", Code: "192834", ExpiresAt: expiresAt,
	}); err != nil {
		t.Fatalf("queue permanently failing email: %v", err)
	}
	waitFor(t, 15*time.Second, func() bool {
		var retrying, failed int
		if err := pool.QueryRow(t.Context(), `SELECT
			count(*) FILTER (WHERE status = 'retrying'),
			count(*) FILTER (WHERE status = 'failed')
			FROM email_deliveries`).Scan(&retrying, &failed); err != nil {
			t.Fatalf("count failed email outcomes: %v", err)
		}
		return retrying == 1 && failed == 1
	})

	var scheduledRetries, terminalRetries int
	if err := pool.QueryRow(t.Context(), `SELECT
		count(*) FILTER (WHERE delivery.status = 'retrying' AND event.available_at > now()),
		count(*) FILTER (WHERE delivery.status = 'failed')
		FROM outbox_events AS event
		JOIN email_deliveries AS delivery ON delivery.id = event.aggregate_id
		WHERE event.published_at IS NULL`).Scan(&scheduledRetries, &terminalRetries); err != nil {
		t.Fatalf("inspect scheduled email retries: %v", err)
	}
	if scheduledRetries != 1 {
		t.Fatalf("scheduled retry events = %d, want 1", scheduledRetries)
	}
	if terminalRetries != 0 {
		t.Fatalf("terminal retry events = %d, want 0", terminalRetries)
	}

	cancel()
	for range 2 {
		select {
		case err := <-errorsChannel:
			if err != nil {
				t.Fatalf("stop async email component: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("async email component did not stop")
		}
	}
}

type sendCall struct {
	recipient      string
	idempotencyKey string
}

type recordingSender struct {
	mu       sync.Mutex
	calls    []sendCall
	failures map[string]error
	active   int
	maximum  int
	started  chan struct{}
	release  chan struct{}
}

func newRecordingSender() *recordingSender {
	return &recordingSender{
		failures: make(map[string]error),
		started:  make(chan struct{}, 8),
		release:  make(chan struct{}),
	}
}

func (sender *recordingSender) Send(
	ctx context.Context,
	recipient, _, _, _, idempotencyKey string,
) (string, error) {
	sender.mu.Lock()
	sender.active++
	if sender.active > sender.maximum {
		sender.maximum = sender.active
	}
	failure := sender.failures[recipient]
	sender.mu.Unlock()

	sender.started <- struct{}{}
	select {
	case <-sender.release:
	case <-ctx.Done():
		return "", ctx.Err()
	}

	sender.mu.Lock()
	defer sender.mu.Unlock()
	sender.active--
	sender.calls = append(sender.calls, sendCall{recipient: recipient, idempotencyKey: idempotencyKey})
	if failure != nil {
		return "", failure
	}
	return fmt.Sprintf("provider-%d", len(sender.calls)), nil
}

func (sender *recordingSender) fail(recipient string, err error) {
	sender.mu.Lock()
	defer sender.mu.Unlock()
	sender.failures[recipient] = err
}

func (sender *recordingSender) maximumActive() int {
	sender.mu.Lock()
	defer sender.mu.Unlock()
	return sender.maximum
}

func (sender *recordingSender) count() int {
	return len(sender.callsSnapshot())
}

func (sender *recordingSender) callsSnapshot() []sendCall {
	sender.mu.Lock()
	defer sender.mu.Unlock()
	return append([]sendCall(nil), sender.calls...)
}

func assertEncryptedAtRest(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	rows, err := pool.Query(t.Context(), "SELECT encrypted_payload FROM email_deliveries")
	if err != nil {
		t.Fatalf("query encrypted email deliveries: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var ciphertext []byte
		if err := rows.Scan(&ciphertext); err != nil {
			t.Fatalf("scan encrypted email delivery: %v", err)
		}
		for _, secret := range []string{"verify@example.com", "482193", "reset@example.com", "token=secret"} {
			if bytes.Contains(ciphertext, []byte(secret)) {
				t.Fatalf("encrypted payload contains plaintext %q", secret)
			}
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate encrypted email deliveries: %v", err)
	}
}

func duplicateOutboxEvent(t *testing.T, pool *pgxpool.Pool, redisClient *redis.Client) {
	t.Helper()
	var event events.Event
	err := pool.QueryRow(t.Context(), `SELECT id, event_type, aggregate_type, aggregate_id, payload, trace_context, occurred_at
		FROM outbox_events ORDER BY occurred_at LIMIT 1`).Scan(
		&event.ID, &event.Type, &event.AggregateType, &event.AggregateID,
		&event.Payload, &event.TraceContext, &event.OccurredAt,
	)
	if err != nil {
		t.Fatalf("load outbox event for duplicate: %v", err)
	}
	if err := redisClient.XAdd(t.Context(), &redis.XAddArgs{
		Stream: events.StreamName,
		Values: map[string]any{
			"event_id": event.ID.String(), "event_type": event.Type,
			"aggregate_type": event.AggregateType, "aggregate_id": event.AggregateID.String(),
			"payload": string(event.Payload), "occurred_at": event.OccurredAt.Format(time.RFC3339Nano),
			"trace_context": "{}",
		},
	}).Err(); err != nil {
		t.Fatalf("publish duplicate event: %v", err)
	}
}

func openRedis(t *testing.T) *redis.Client {
	t.Helper()
	container, err := testcontainers.Run(
		t.Context(),
		"redis:8.2-alpine",
		testcontainers.WithExposedPorts("6379/tcp"),
		testcontainers.WithWaitStrategy(wait.ForLog("Ready to accept connections")),
	)
	if err != nil {
		t.Fatalf("start Redis container: %v", err)
	}
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Errorf("terminate Redis container: %v", err)
		}
	})
	host, err := container.Host(t.Context())
	if err != nil {
		t.Fatalf("get Redis host: %v", err)
	}
	port, err := container.MappedPort(t.Context(), "6379/tcp")
	if err != nil {
		t.Fatalf("get Redis port: %v", err)
	}
	client := redis.NewClient(&redis.Options{Addr: net.JoinHostPort(host, port.Port())})
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close Redis client: %v", err)
		}
	})
	return client
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("timed out waiting for asynchronous email delivery")
}

var _ emaildelivery.Sender = (*recordingSender)(nil)

type testSendFailure struct {
	retryable  bool
	retryAfter time.Duration
}

func (failure testSendFailure) Error() string             { return "test delivery failure" }
func (failure testSendFailure) Retryable() bool           { return failure.retryable }
func (failure testSendFailure) RetryAfter() time.Duration { return failure.retryAfter }

var _ emaildelivery.SendFailure = testSendFailure{}
