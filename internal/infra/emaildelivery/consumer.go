package emaildelivery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	authenticationtemplates "github.com/Rahmannugar/consumel-server/internal/authentication/templates"
	"github.com/Rahmannugar/consumel-server/internal/infra/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const (
	emailConsumerGroup = "email-deliveries"
	maximumAttempts    = 5
	claimIdle          = 30 * time.Second
	processingLease    = 30 * time.Second
	readBlock          = 5 * time.Second
	maximumRetryDelay  = 5 * time.Minute
	maximumErrorLength = 1000
)

type Sender interface {
	Send(context.Context, string, string, string, string, string) (string, error)
}

// SendFailure distinguishes temporary provider or network failures from
// requests that cannot succeed without changing their input or configuration.
type SendFailure interface {
	error
	Retryable() bool
	RetryAfter() time.Duration
}

type Consumer struct {
	pool        *pgxpool.Pool
	redis       *redis.Client
	queue       *Queue
	sender      Sender
	name        string
	concurrency int
	logger      *slog.Logger
}

func NewConsumer(
	pool *pgxpool.Pool,
	redisClient *redis.Client,
	queue *Queue,
	sender Sender,
	name string,
	concurrency int,
	logger *slog.Logger,
) (*Consumer, error) {
	if concurrency < 1 {
		return nil, fmt.Errorf("email delivery concurrency must be positive")
	}
	return &Consumer{
		pool: pool, redis: redisClient, queue: queue, sender: sender,
		name: name, concurrency: concurrency, logger: logger,
	}, nil
}

func (consumer *Consumer) Run(ctx context.Context) error {
	if err := consumer.redis.XGroupCreateMkStream(ctx, events.StreamName, emailConsumerGroup, "0").Err(); err != nil &&
		!strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("create email consumer group: %w", err)
	}

	group, groupContext := errgroup.WithContext(ctx)
	messages := make(chan redis.XMessage)
	group.Go(func() error {
		defer close(messages)
		consumer.read(groupContext, messages)
		return nil
	})
	for range consumer.concurrency {
		group.Go(func() error {
			for message := range messages {
				consumer.process(groupContext, message)
			}
			return nil
		})
	}
	return group.Wait()
}

func (consumer *Consumer) read(ctx context.Context, messages chan<- redis.XMessage) {
	for ctx.Err() == nil {
		consumer.reclaim(ctx, messages)
		streams, err := consumer.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group: emailConsumerGroup, Consumer: consumer.name,
			Streams: []string{events.StreamName, ">"}, Count: int64(consumer.concurrency), Block: readBlock,
		}).Result()
		if errors.Is(err, redis.Nil) || errors.Is(err, context.Canceled) {
			continue
		}
		if err != nil {
			consumer.logger.ErrorContext(ctx, "Could not read email delivery events",
				"event", "email.worker.read.failed", "operation", "email.delivery.consume", "error", err)
			continue
		}
		for _, stream := range streams {
			for _, message := range stream.Messages {
				if !sendMessage(ctx, messages, message) {
					return
				}
			}
		}
	}
}

func (consumer *Consumer) reclaim(ctx context.Context, messages chan<- redis.XMessage) {
	start := "0-0"
	for ctx.Err() == nil {
		claimed, next, err := consumer.redis.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream: events.StreamName, Group: emailConsumerGroup, Consumer: consumer.name,
			MinIdle: claimIdle, Start: start, Count: int64(consumer.concurrency),
		}).Result()
		if err != nil {
			if !errors.Is(err, redis.Nil) && !errors.Is(err, context.Canceled) {
				consumer.logger.WarnContext(ctx, "Could not reclaim email delivery events",
					"event", "email.worker.reclaim.failed", "operation", "email.delivery.reclaim", "error", err)
			}
			return
		}
		for _, message := range claimed {
			if !sendMessage(ctx, messages, message) {
				return
			}
		}
		if next == "0-0" || next == start {
			return
		}
		start = next
	}
}

func sendMessage(ctx context.Context, messages chan<- redis.XMessage, message redis.XMessage) bool {
	select {
	case messages <- message:
		return true
	case <-ctx.Done():
		return false
	}
}

func (consumer *Consumer) process(ctx context.Context, message redis.XMessage) {
	event, err := events.ParseStreamEvent(message)
	if err != nil {
		consumer.logger.ErrorContext(ctx, "Discarding malformed email event",
			"event", "email.worker.message.invalid", "operation", "email.delivery.consume",
			"message_id", message.ID, "error", err)
		consumer.acknowledge(ctx, message.ID)
		return
	}
	if event.Type != EventTypeQueued || event.AggregateType != "email_delivery" {
		consumer.acknowledge(ctx, message.ID)
		return
	}
	ctx = otel.GetTextMapPropagator().Extract(ctx, event.TraceContext)
	ctx, span := otel.Tracer("github.com/Rahmannugar/consumel-server/internal/infra/emaildelivery").Start(
		ctx, "email.delivery.send", trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.message.id", event.ID.String()),
			attribute.String("email.delivery.id", event.AggregateID.String()),
		),
	)
	defer span.End()

	delivery, claim, err := consumer.claim(ctx, event.AggregateID)
	if err != nil {
		span.RecordError(err)
		consumer.logger.ErrorContext(ctx, "Could not claim email delivery",
			"event", "email.delivery.claim.failed", "operation", "email.delivery.send",
			"message_id", message.ID, "event_id", event.ID, "delivery_id", event.AggregateID, "error", err)
		return
	}
	if claim == claimTerminal || claim == claimSuperseded {
		consumer.acknowledge(ctx, message.ID)
		return
	}
	if claim == claimUnavailable {
		return
	}

	started := time.Now()
	consumer.logger.InfoContext(ctx, "Email delivery started",
		"event", "email.delivery.started", "operation", "email.delivery.send",
		"message_id", message.ID, "event_id", event.ID,
		"delivery_id", delivery.ID, "attempt", delivery.Attempts)

	payload, err := consumer.queue.Decrypt(delivery.ID, delivery.Nonce, delivery.Ciphertext)
	if err != nil {
		consumer.finishFailure(ctx, span, message, event, delivery, err, false, 0, started)
		return
	}
	rendered, err := render(delivery.Template, payload, time.Now().UTC())
	if err != nil {
		consumer.finishFailure(ctx, span, message, event, delivery, err, false, 0, started)
		return
	}
	providerID, err := consumer.sender.Send(
		ctx, payload.Recipient, rendered.Subject, rendered.Text, rendered.HTML,
		// Resend retains idempotency keys for 24 hours, which covers this
		// authentication-email delivery's complete retry window.
		"email-delivery/"+delivery.ID.String(),
	)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		retryable, retryAfter := sendFailurePolicy(err)
		consumer.finishFailure(ctx, span, message, event, delivery, err, retryable, retryAfter, started)
		return
	}

	updated, err := consumer.markDelivered(ctx, delivery, providerID)
	if err != nil {
		span.RecordError(err)
		consumer.logger.ErrorContext(ctx, "Could not record delivered email",
			"event", "email.delivery.completion_record.failed", "operation", "email.delivery.send",
			"message_id", message.ID, "event_id", event.ID, "delivery_id", delivery.ID, "error", err)
		return
	}
	if !updated {
		return
	}
	consumer.acknowledge(ctx, message.ID)
	consumer.logger.InfoContext(ctx, "Email delivered",
		"event", "email.delivery.completed", "operation", "email.delivery.send",
		"message_id", message.ID, "event_id", event.ID, "delivery_id", delivery.ID,
		"attempt", delivery.Attempts, "duration_ms", time.Since(started).Milliseconds())
}

func (consumer *Consumer) finishFailure(
	ctx context.Context,
	span trace.Span,
	message redis.XMessage,
	event events.Event,
	delivery delivery,
	deliveryError error,
	retryable bool,
	retryAfter time.Duration,
	started time.Time,
) {
	span.RecordError(deliveryError)
	terminal, updated, err := consumer.recordFailure(ctx, delivery, deliveryError, retryable, retryAfter)
	if err != nil {
		consumer.logger.ErrorContext(ctx, "Could not record email delivery failure",
			"event", "email.delivery.failure_record.failed", "operation", "email.delivery.send",
			"delivery_id", delivery.ID, "error", err)
		return
	}
	if !updated {
		return
	}
	// A retry owns a new, transactionally scheduled outbox event, so the
	// current Redis entry is complete even though the delivery is not.
	consumer.acknowledge(ctx, message.ID)
	consumer.logger.Log(ctx, failureLevel(terminal), "Email delivery failed",
		"event", "email.delivery.failed", "operation", "email.delivery.send",
		"message_id", message.ID, "event_id", event.ID, "delivery_id", delivery.ID,
		"attempt", delivery.Attempts, "duration_ms", time.Since(started).Milliseconds(),
		"terminal", terminal, "retryable", retryable, "error", deliveryError)
}

type claimResult int

const (
	claimUnavailable claimResult = iota
	claimAcquired
	claimTerminal
	claimSuperseded
)

type delivery struct {
	ID         uuid.UUID
	ClaimID    uuid.UUID
	Template   string
	Ciphertext []byte
	Nonce      []byte
	ExpiresAt  time.Time
	Attempts   int
}

func (consumer *Consumer) claim(ctx context.Context, id uuid.UUID) (delivery, claimResult, error) {
	tx, err := consumer.pool.Begin(ctx)
	if err != nil {
		return delivery{}, claimUnavailable, err
	}
	defer tx.Rollback(ctx)
	var item delivery
	var status string
	var startedAt *time.Time
	var nextAttemptAt time.Time
	err = tx.QueryRow(ctx, `SELECT id, template, encrypted_payload, payload_nonce, expires_at,
		status, attempts, processing_started_at, next_attempt_at
		FROM email_deliveries WHERE id = $1 FOR UPDATE`, id,
	).Scan(&item.ID, &item.Template, &item.Ciphertext, &item.Nonce, &item.ExpiresAt,
		&status, &item.Attempts, &startedAt, &nextAttemptAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return delivery{}, claimTerminal, tx.Commit(ctx)
	}
	if err != nil {
		return delivery{}, claimUnavailable, err
	}
	now := time.Now().UTC()
	if !now.Before(item.ExpiresAt) {
		_, err = tx.Exec(ctx, `UPDATE email_deliveries SET
			status = 'expired', processing_started_at = NULL, processing_claim_id = NULL,
			encrypted_payload = '\x00'::bytea,
			payload_nonce = decode('000000000000000000000000', 'hex'), updated_at = now()
			WHERE id = $1`, id)
		if err != nil {
			return delivery{}, claimUnavailable, err
		}
		return item, claimTerminal, tx.Commit(ctx)
	}
	if status == "delivered" || status == "failed" || status == "expired" {
		return item, claimTerminal, tx.Commit(ctx)
	}
	if status == "processing" && startedAt != nil && startedAt.Add(processingLease).After(now) {
		return item, claimUnavailable, tx.Commit(ctx)
	}
	if status == "retrying" && nextAttemptAt.After(now) {
		// A future outbox event owns the retry. This is an unacknowledged copy
		// of the completed earlier attempt.
		return item, claimSuperseded, tx.Commit(ctx)
	}
	item.Attempts++
	item.ClaimID = uuid.New()
	_, err = tx.Exec(ctx, `UPDATE email_deliveries SET
		status = 'processing', attempts = $2, processing_started_at = now(),
		processing_claim_id = $3, updated_at = now()
		WHERE id = $1`, id, item.Attempts, item.ClaimID)
	if err != nil {
		return delivery{}, claimUnavailable, err
	}
	return item, claimAcquired, tx.Commit(ctx)
}

func (consumer *Consumer) markDelivered(ctx context.Context, item delivery, providerID string) (bool, error) {
	tag, err := consumer.pool.Exec(ctx, `UPDATE email_deliveries SET
		status = 'delivered', delivered_at = now(), provider_message_id = $3,
		processing_started_at = NULL, processing_claim_id = NULL,
		encrypted_payload = '\x00'::bytea,
		payload_nonce = decode('000000000000000000000000', 'hex'),
		last_error = NULL, updated_at = now()
		WHERE id = $1 AND processing_claim_id = $2 AND status = 'processing'`,
		item.ID, item.ClaimID, providerID)
	return tag.RowsAffected() == 1, err
}

func (consumer *Consumer) recordFailure(
	ctx context.Context,
	item delivery,
	deliveryError error,
	retryable bool,
	retryAfter time.Duration,
) (terminal bool, updated bool, resultError error) {
	now := time.Now().UTC()
	delay := retryDelay(item.Attempts)
	if retryAfter > delay {
		delay = min(retryAfter, maximumRetryDelay)
	}
	nextAttemptAt := now.Add(delay)
	terminal = !retryable || item.Attempts >= maximumAttempts || !nextAttemptAt.Before(item.ExpiresAt)
	status := "retrying"
	if terminal {
		status = "failed"
	}
	message := deliveryError.Error()
	if len(message) > maximumErrorLength {
		message = message[:maximumErrorLength]
	}

	tx, err := consumer.pool.Begin(ctx)
	if err != nil {
		return terminal, false, err
	}
	defer tx.Rollback(ctx)
	var tag pgconn.CommandTag
	if terminal {
		tag, err = tx.Exec(ctx, `UPDATE email_deliveries SET
			status = $3, processing_started_at = NULL, processing_claim_id = NULL,
			next_attempt_at = $4, last_error = $5,
			encrypted_payload = '\x00'::bytea,
			payload_nonce = decode('000000000000000000000000', 'hex'), updated_at = now()
			WHERE id = $1 AND processing_claim_id = $2 AND status = 'processing'`,
			item.ID, item.ClaimID, status, nextAttemptAt, message)
	} else {
		tag, err = tx.Exec(ctx, `UPDATE email_deliveries SET
			status = $3, processing_started_at = NULL, processing_claim_id = NULL,
			next_attempt_at = $4, last_error = $5, updated_at = now()
			WHERE id = $1 AND processing_claim_id = $2 AND status = 'processing'`,
			item.ID, item.ClaimID, status, nextAttemptAt, message)
	}
	if err != nil {
		return terminal, false, err
	}
	if tag.RowsAffected() == 0 {
		return terminal, false, tx.Commit(ctx)
	}
	if !terminal {
		if err := consumer.queue.insertEvent(ctx, tx, item.ID, nextAttemptAt); err != nil {
			return terminal, false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return terminal, false, err
	}
	return terminal, true, nil
}

func sendFailurePolicy(err error) (bool, time.Duration) {
	var failure SendFailure
	if !errors.As(err, &failure) {
		return false, 0
	}
	return failure.Retryable(), failure.RetryAfter()
}

func retryDelay(attempt int) time.Duration {
	delay := 10 * time.Second
	for current := 1; current < attempt; current++ {
		delay *= 2
	}
	return min(delay, maximumRetryDelay)
}

func (consumer *Consumer) acknowledge(ctx context.Context, messageID string) {
	if err := consumer.redis.XAck(ctx, events.StreamName, emailConsumerGroup, messageID).Err(); err != nil {
		consumer.logger.ErrorContext(ctx, "Could not acknowledge email delivery event",
			"event", "email.worker.ack.failed", "operation", "email.delivery.acknowledge",
			"message_id", messageID, "error", err)
	}
}

type renderedEmail struct {
	Subject string
	Text    string
	HTML    string
}

func render(templateName string, payload Payload, now time.Time) (renderedEmail, error) {
	switch templateName {
	case TemplateVerification:
		email, err := authenticationtemplates.RenderVerificationEmail(payload.Code, payload.ExpiresAt, now)
		return renderedEmail{Subject: email.Subject, Text: email.Text, HTML: email.HTML}, err
	case TemplatePasswordReset:
		email, err := authenticationtemplates.RenderPasswordResetEmail(payload.URL, payload.ExpiresAt, now)
		return renderedEmail{Subject: email.Subject, Text: email.Text, HTML: email.HTML}, err
	default:
		return renderedEmail{}, fmt.Errorf("unsupported email template %q", templateName)
	}
}

func failureLevel(terminal bool) slog.Level {
	if terminal {
		return slog.LevelError
	}
	return slog.LevelWarn
}
