package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const (
	defaultRelayBatchSize = 100
	defaultPollInterval   = 5 * time.Second
	defaultPublishLease   = 30 * time.Second
	streamMaximumLength   = 50_000
	maximumPublishError   = 1000
)

type Relay struct {
	pool         *pgxpool.Pool
	redis        *redis.Client
	logger       *slog.Logger
	pollInterval time.Duration
	publishLease time.Duration
	batchSize    int
	concurrency  int
}

func NewRelay(
	pool *pgxpool.Pool,
	redisClient *redis.Client,
	concurrency int,
	logger *slog.Logger,
) (*Relay, error) {
	if concurrency < 1 {
		return nil, fmt.Errorf("outbox publication concurrency must be positive")
	}
	return &Relay{
		pool: pool, redis: redisClient, logger: logger,
		pollInterval: defaultPollInterval, publishLease: defaultPublishLease,
		batchSize: defaultRelayBatchSize, concurrency: concurrency,
	}, nil
}

func (relay *Relay) Run(ctx context.Context) error {
	wake := make(chan struct{}, 1)
	listenerErrors := make(chan error, 1)
	go relay.listen(ctx, wake, listenerErrors)

	ticker := time.NewTicker(relay.pollInterval)
	defer ticker.Stop()
	relay.drain(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-listenerErrors:
			if err != nil && !errors.Is(err, context.Canceled) {
				relay.logger.WarnContext(ctx, "Outbox notification listener stopped; polling remains active",
					"event", "outbox.listener.failed", "operation", "outbox.listen", "error", err)
			}
			listenerErrors = nil
		case <-wake:
			relay.drain(ctx)
		case <-ticker.C:
			relay.drain(ctx)
		}
	}
}

func (relay *Relay) listen(ctx context.Context, wake chan<- struct{}, failures chan<- error) {
	connection, err := relay.pool.Acquire(ctx)
	if err != nil {
		failures <- fmt.Errorf("acquire outbox listener connection: %w", err)
		return
	}
	defer connection.Release()
	if _, err := connection.Exec(ctx, "LISTEN outbox_events"); err != nil {
		failures <- fmt.Errorf("listen for outbox events: %w", err)
		return
	}
	for {
		if _, err := connection.Conn().WaitForNotification(ctx); err != nil {
			failures <- err
			return
		}
		select {
		case wake <- struct{}{}:
		default:
		}
	}
}

func (relay *Relay) drain(ctx context.Context) {
	for {
		published, err := relay.publishBatch(ctx)
		if err != nil {
			relay.logger.ErrorContext(ctx, "Could not publish outbox events",
				"event", "outbox.publish.failed", "operation", "outbox.publish", "error", err)
			return
		}
		if published < relay.batchSize {
			return
		}
	}
}

func (relay *Relay) publishBatch(ctx context.Context) (int, error) {
	claimID := uuid.New()
	events, err := relay.claimBatch(ctx, claimID)
	if err != nil || len(events) == 0 {
		return 0, err
	}

	outcomes := make([]publishOutcome, len(events))
	var publications errgroup.Group
	publications.SetLimit(relay.concurrency)
	for index, event := range events {
		publications.Go(func() error {
			outcomes[index] = relay.publishEvent(ctx, event)
			return nil
		})
	}
	_ = publications.Wait()

	published := 0
	for _, outcome := range outcomes {
		if outcome.err == nil {
			published++
		}
	}

	// Redis publication happens outside database transactions. This short
	// checkpoint transaction releases successful claims and makes failures
	// immediately claimable; an expired claim recovers a crashed relay.
	tx, err := relay.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin outbox checkpoint transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	for _, outcome := range outcomes {
		if outcome.err != nil {
			message := outcome.err.Error()
			if len(message) > maximumPublishError {
				message = message[:maximumPublishError]
			}
			if _, updateErr := tx.Exec(ctx, `UPDATE outbox_events SET
				publish_attempts = publish_attempts + 1, last_publish_error = $3,
				publish_claim_id = NULL, publish_claimed_at = NULL
				WHERE id = $1 AND publish_claim_id = $2 AND published_at IS NULL`,
				outcome.event.ID, claimID, message); updateErr != nil {
				return published, fmt.Errorf("record outbox publication failure: %w", updateErr)
			}
			continue
		}
		if _, err := tx.Exec(ctx, `UPDATE outbox_events SET
			published_at = now(), publish_attempts = publish_attempts + 1,
			last_publish_error = NULL, publish_claim_id = NULL, publish_claimed_at = NULL
			WHERE id = $1 AND publish_claim_id = $2 AND published_at IS NULL`,
			outcome.event.ID, claimID); err != nil {
			return published, fmt.Errorf("mark outbox event published: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return published, fmt.Errorf("commit outbox checkpoint transaction: %w", err)
	}
	return published, nil
}

type publishOutcome struct {
	event Event
	err   error
}

func (relay *Relay) publishEvent(ctx context.Context, event Event) publishOutcome {
	messageContext := otel.GetTextMapPropagator().Extract(ctx, event.TraceContext)
	messageContext, span := otel.Tracer("github.com/Rahmannugar/consumel-server/internal/infra/events").Start(
		messageContext, "outbox.publish", trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(attribute.String("messaging.message.id", event.ID.String())),
	)
	defer span.End()
	started := time.Now()
	relay.logger.InfoContext(messageContext, "Outbox publication started",
		"event", "outbox.publish.started", "operation", "outbox.publish",
		"event_id", event.ID, "event_type", event.Type,
		"aggregate_type", event.AggregateType, "aggregate_id", event.AggregateID)
	publishErr := relay.publish(messageContext, event)
	if publishErr != nil {
		relay.logger.WarnContext(messageContext, "Outbox publication will retry",
			"event", "outbox.publish.retry", "operation", "outbox.publish",
			"event_id", event.ID, "event_type", event.Type,
			"aggregate_type", event.AggregateType, "aggregate_id", event.AggregateID,
			"duration_ms", time.Since(started).Milliseconds(), "error", publishErr)
		span.RecordError(publishErr)
		return publishOutcome{event: event, err: publishErr}
	}
	relay.logger.InfoContext(messageContext, "Outbox publication completed",
		"event", "outbox.publish.completed", "operation", "outbox.publish",
		"event_id", event.ID, "event_type", event.Type,
		"aggregate_type", event.AggregateType, "aggregate_id", event.AggregateID,
		"duration_ms", time.Since(started).Milliseconds())
	return publishOutcome{event: event}
}

func (relay *Relay) claimBatch(ctx context.Context, claimID uuid.UUID) ([]Event, error) {
	// SKIP LOCKED coordinates relay instances only for this short claim. The
	// lease, rather than a database lock, owns the rows during Redis I/O.
	rows, err := relay.pool.Query(ctx, `WITH candidates AS (
		SELECT id FROM outbox_events
		WHERE published_at IS NULL AND available_at <= now()
			AND (publish_claimed_at IS NULL OR publish_claimed_at < now() - $3::interval)
		ORDER BY available_at, id
		FOR UPDATE SKIP LOCKED
		LIMIT $2
	)
	UPDATE outbox_events AS event
	SET publish_claim_id = $1, publish_claimed_at = now()
	FROM candidates
	WHERE event.id = candidates.id
	RETURNING event.id, event.event_type, event.aggregate_type, event.aggregate_id,
		event.payload, event.trace_context, event.occurred_at`,
		claimID, relay.batchSize, relay.publishLease.String())
	if err != nil {
		return nil, fmt.Errorf("claim outbox events: %w", err)
	}
	defer rows.Close()
	events := make([]Event, 0, relay.batchSize)
	for rows.Next() {
		var event Event
		if err := rows.Scan(
			&event.ID, &event.Type, &event.AggregateType, &event.AggregateID,
			&event.Payload, &event.TraceContext, &event.OccurredAt,
		); err != nil {
			return nil, fmt.Errorf("scan claimed outbox event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate claimed outbox events: %w", err)
	}
	return events, nil
}

func (relay *Relay) publish(ctx context.Context, event Event) error {
	if !json.Valid(event.Payload) {
		return fmt.Errorf("event %s has invalid JSON payload", event.ID)
	}
	_, err := relay.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamName, Mode: "ACKED", MaxLen: streamMaximumLength, Approx: true,
		Values: map[string]any{
			"event_id": event.ID.String(), "event_type": event.Type,
			"aggregate_type": event.AggregateType, "aggregate_id": event.AggregateID.String(),
			"payload": string(event.Payload), "occurred_at": event.OccurredAt.UTC().Format(time.RFC3339Nano),
			"trace_context": encodeTraceContext(event.TraceContext),
		},
	}).Result()
	if err != nil {
		return fmt.Errorf("publish event %s to Redis Streams: %w", event.ID, err)
	}
	return nil
}

func ParseStreamEvent(message redis.XMessage) (Event, error) {
	value := func(name string) (string, error) {
		raw, ok := message.Values[name]
		if !ok {
			return "", fmt.Errorf("stream message %s is missing %s", message.ID, name)
		}
		text, ok := raw.(string)
		if !ok {
			return "", fmt.Errorf("stream message %s has invalid %s", message.ID, name)
		}
		return text, nil
	}
	idText, err := value("event_id")
	if err != nil {
		return Event{}, err
	}
	id, err := uuid.Parse(idText)
	if err != nil {
		return Event{}, fmt.Errorf("parse event ID: %w", err)
	}
	aggregateIDText, err := value("aggregate_id")
	if err != nil {
		return Event{}, err
	}
	aggregateID, err := uuid.Parse(aggregateIDText)
	if err != nil {
		return Event{}, fmt.Errorf("parse aggregate ID: %w", err)
	}
	eventType, err := value("event_type")
	if err != nil {
		return Event{}, err
	}
	aggregateType, err := value("aggregate_type")
	if err != nil {
		return Event{}, err
	}
	payloadText, err := value("payload")
	if err != nil {
		return Event{}, err
	}
	payload := json.RawMessage(payloadText)
	if !json.Valid(payload) {
		return Event{}, fmt.Errorf("stream message %s has invalid JSON payload", message.ID)
	}
	occurredText, err := value("occurred_at")
	if err != nil {
		return Event{}, err
	}
	occurredAt, err := time.Parse(time.RFC3339Nano, occurredText)
	if err != nil {
		return Event{}, fmt.Errorf("parse event occurrence time: %w", err)
	}
	traceContext := propagation.MapCarrier{}
	if encoded, ok := message.Values["trace_context"].(string); ok && encoded != "" {
		if err := json.Unmarshal([]byte(encoded), &traceContext); err != nil {
			return Event{}, fmt.Errorf("decode stream trace context: %w", err)
		}
	}
	return Event{
		ID: id, Type: eventType, AggregateType: aggregateType,
		AggregateID: aggregateID, Payload: payload, TraceContext: traceContext, OccurredAt: occurredAt,
	}, nil
}

func encodeTraceContext(carrier propagation.MapCarrier) string {
	encoded, err := json.Marshal(carrier)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}
