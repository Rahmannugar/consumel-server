package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

const StreamName = "consumel:events"

type Event struct {
	ID            uuid.UUID
	Type          string
	AggregateType string
	AggregateID   uuid.UUID
	Payload       json.RawMessage
	TraceContext  propagation.MapCarrier
	OccurredAt    time.Time
}

type Execer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func Insert(ctx context.Context, execer Execer, event Event) error {
	if !json.Valid(event.Payload) {
		return fmt.Errorf("event payload must be valid JSON")
	}
	traceContext := propagation.MapCarrier{}
	// Persist W3C propagation fields with the business transaction so worker
	// spans remain connected even when delivery happens much later.
	otel.GetTextMapPropagator().Inject(ctx, traceContext)
	_, err := execer.Exec(ctx, `INSERT INTO outbox_events
		(id, event_type, aggregate_type, aggregate_id, payload, trace_context, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		event.ID, event.Type, event.AggregateType, event.AggregateID, event.Payload, traceContext, event.OccurredAt,
	)
	if err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	return nil
}
