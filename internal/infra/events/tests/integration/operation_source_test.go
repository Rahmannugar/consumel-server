//go:build integration

package integration_test

import (
	"net"
	"testing"
	"time"

	"github.com/Rahmannugar/consumel-server/internal/infra/events"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestOperationSourceFiltersAndAdvancesPastUnrelatedEvents(t *testing.T) {
	client := openRedis(t)
	source := events.NewOperationSource(client)
	environmentID := uuid.MustParse("0199bb00-0000-7000-8000-000000000002")
	operationID := uuid.MustParse("0199bb00-0000-7000-8000-000000000010")

	unrelatedCursor := addStreamEvent(t, client, "email.delivery.queued.v1", uuid.New(), `{}`)
	notices, cursor, err := source.Read(t.Context(), "0-0", time.Millisecond)
	if err != nil {
		t.Fatalf("read unrelated event: %v", err)
	}
	if len(notices) != 0 || cursor != unrelatedCursor {
		t.Fatalf("unrelated read returned %d notices and cursor %q, want 0 and %q", len(notices), cursor, unrelatedCursor)
	}

	payload := `{"projectEnvironmentId":"` + environmentID.String() + `","usageEventId":"` + operationID.String() + `"}`
	usageCursor := addStreamEvent(t, client, "usage.consumed.v1", operationID, payload)
	notices, cursor, err = source.Read(t.Context(), cursor, time.Millisecond)
	if err != nil {
		t.Fatalf("read usage event: %v", err)
	}
	if len(notices) != 1 || cursor != usageCursor {
		t.Fatalf("usage read returned %d notices and cursor %q, want 1 and %q", len(notices), cursor, usageCursor)
	}
	if notices[0].ProjectEnvironmentID != environmentID || notices[0].OperationID != operationID {
		t.Fatalf("usage notice = %#v", notices[0])
	}
}

func addStreamEvent(t *testing.T, client *redis.Client, eventType string, aggregateID uuid.UUID, payload string) string {
	t.Helper()
	cursor, err := client.XAdd(t.Context(), &redis.XAddArgs{
		Stream: events.StreamName,
		Values: map[string]any{
			"event_id": uuid.New().String(), "event_type": eventType,
			"aggregate_type": "test", "aggregate_id": aggregateID.String(),
			"payload": payload, "occurred_at": time.Now().UTC().Format(time.RFC3339Nano),
			"trace_context": "{}",
		},
	}).Result()
	if err != nil {
		t.Fatalf("add stream event: %v", err)
	}
	return cursor
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
