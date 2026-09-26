package emaildelivery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	authenticationtemplates "github.com/Rahmannugar/consumel-server/internal/authentication/templates"
	"github.com/Rahmannugar/consumel-server/internal/infra/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	emailConsumerGroup = "email-deliveries"
	maximumAttempts    = 10
	claimIdle          = 10 * time.Second
	processingLease    = 2 * time.Minute
	maximumErrorLength = 1000
)

type Sender interface {
	Send(context.Context, string, string, string, string, string) (string, error)
}

type Consumer struct {
	pool   *pgxpool.Pool
	redis  *redis.Client
	queue  *Queue
	sender Sender
	name   string
	logger *slog.Logger
}

func NewConsumer(
	pool *pgxpool.Pool,
	redisClient *redis.Client,
	queue *Queue,
	sender Sender,
	name string,
	logger *slog.Logger,
) *Consumer {
	return &Consumer{pool: pool, redis: redisClient, queue: queue, sender: sender, name: name, logger: logger}
}

func (consumer *Consumer) Run(ctx context.Context) error {
	if err := consumer.redis.XGroupCreateMkStream(ctx, events.StreamName, emailConsumerGroup, "0").Err(); err != nil &&
		!strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("create email consumer group: %w", err)
	}
	for {
		if ctx.Err() != nil {
			return nil
		}
		consumer.reclaim(ctx)
		streams, err := consumer.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group: emailConsumerGroup, Consumer: consumer.name,
			Streams: []string{events.StreamName, ">"}, Count: 20, Block: 5 * time.Second,
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
				consumer.process(ctx, message)
			}
		}
	}
}

func (consumer *Consumer) reclaim(ctx context.Context) {
	start := "0-0"
	for {
		messages, next, err := consumer.redis.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream: events.StreamName, Group: emailConsumerGroup, Consumer: consumer.name,
			MinIdle: claimIdle, Start: start, Count: 20,
		}).Result()
		if err != nil && !errors.Is(err, redis.Nil) && !errors.Is(err, context.Canceled) {
			consumer.logger.WarnContext(ctx, "Could not reclaim email delivery events",
				"event", "email.worker.reclaim.failed", "operation", "email.delivery.reclaim", "error", err)
			return
		}
		for _, message := range messages {
			consumer.process(ctx, message)
		}
		if next == "0-0" || next == start || ctx.Err() != nil {
			return
		}
		start = next
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
	if claim == claimTerminal {
		consumer.acknowledge(ctx, message.ID)
		return
	}
	if claim == claimUnavailable {
		// The message stays pending in Redis. XAUTOCLAIM revisits it after the
		// database-owned retry time or processing lease becomes available.
		return
	}

	started := time.Now()
	consumer.logger.InfoContext(ctx, "Email delivery started",
		"event", "email.delivery.started", "operation", "email.delivery.send",
		"message_id", message.ID, "event_id", event.ID,
		"delivery_id", delivery.ID, "attempt", delivery.Attempts)
	payload, err := consumer.queue.Decrypt(delivery.ID, delivery.Nonce, delivery.Ciphertext)
	if err == nil {
		var rendered renderedEmail
		rendered, err = render(delivery.Template, payload, time.Now().UTC())
		if err == nil {
			var providerID string
			providerID, err = consumer.sender.Send(
				ctx, payload.Recipient, rendered.Subject, rendered.Text, rendered.HTML,
				// A retry after Resend accepted the request but before PostgreSQL was
				// updated reuses this key instead of sending a second email.
				"email-delivery/"+delivery.ID.String(),
			)
			if err == nil {
				err = consumer.markDelivered(ctx, delivery.ID, providerID)
			}
		}
	}
	if err != nil {
		span.RecordError(err)
		exhausted := delivery.Attempts >= maximumAttempts || time.Now().After(delivery.ExpiresAt)
		if recordErr := consumer.recordFailure(ctx, delivery.ID, delivery.Attempts, err, exhausted); recordErr != nil {
			consumer.logger.ErrorContext(ctx, "Could not record email delivery failure",
				"event", "email.delivery.failure_record.failed", "operation", "email.delivery.send",
				"delivery_id", delivery.ID, "error", recordErr)
			return
		}
		if exhausted {
			consumer.acknowledge(ctx, message.ID)
		}
		consumer.logger.Log(ctx, failureLevel(exhausted), "Email delivery failed",
			"event", "email.delivery.failed", "operation", "email.delivery.send",
			"message_id", message.ID, "event_id", event.ID, "delivery_id", delivery.ID,
			"attempt", delivery.Attempts, "duration_ms", time.Since(started).Milliseconds(),
			"exhausted", exhausted, "error", err)
		return
	}
	consumer.acknowledge(ctx, message.ID)
	consumer.logger.InfoContext(ctx, "Email delivered",
		"event", "email.delivery.completed", "operation", "email.delivery.send",
		"message_id", message.ID, "event_id", event.ID, "delivery_id", delivery.ID,
		"attempt", delivery.Attempts, "duration_ms", time.Since(started).Milliseconds())
}

type claimResult int

const (
	claimUnavailable claimResult = iota
	claimAcquired
	claimTerminal
)

type delivery struct {
	ID         uuid.UUID
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
	now := time.Now()
	if now.After(item.ExpiresAt) {
		_, err = tx.Exec(ctx, `UPDATE email_deliveries
			SET status = 'expired', processing_started_at = NULL, updated_at = now()
			WHERE id = $1`, id)
		if err != nil {
			return delivery{}, claimUnavailable, err
		}
		return item, claimTerminal, tx.Commit(ctx)
	}
	if status == "delivered" || status == "failed" || status == "expired" {
		return item, claimTerminal, tx.Commit(ctx)
	}
	if (status == "processing" && startedAt != nil && startedAt.Add(processingLease).After(now)) ||
		(status == "retrying" && nextAttemptAt.After(now)) {
		return item, claimUnavailable, tx.Commit(ctx)
	}
	item.Attempts++
	_, err = tx.Exec(ctx, `UPDATE email_deliveries
		SET status = 'processing', attempts = $2, processing_started_at = now(), updated_at = now()
		WHERE id = $1`, id, item.Attempts)
	if err != nil {
		return delivery{}, claimUnavailable, err
	}
	return item, claimAcquired, tx.Commit(ctx)
}

func (consumer *Consumer) markDelivered(ctx context.Context, id uuid.UUID, providerID string) error {
	_, err := consumer.pool.Exec(ctx, `UPDATE email_deliveries
		SET status = 'delivered', delivered_at = now(), provider_message_id = $2,
			processing_started_at = NULL, encrypted_payload = '\x00'::bytea,
			payload_nonce = decode('000000000000000000000000', 'hex'), last_error = NULL, updated_at = now()
		WHERE id = $1`, id, providerID)
	return err
}

func (consumer *Consumer) recordFailure(
	ctx context.Context,
	id uuid.UUID,
	attempt int,
	deliveryError error,
	exhausted bool,
) error {
	status := "retrying"
	if exhausted {
		status = "failed"
	}
	delay := time.Duration(math.Min(math.Pow(2, float64(attempt-1))*10, 300)) * time.Second
	message := deliveryError.Error()
	if len(message) > maximumErrorLength {
		message = message[:maximumErrorLength]
	}
	_, err := consumer.pool.Exec(ctx, `UPDATE email_deliveries
		SET status = $2, processing_started_at = NULL, next_attempt_at = now() + $3,
			last_error = $4, updated_at = now()
		WHERE id = $1`, id, status, delay, message)
	return err
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

func failureLevel(exhausted bool) slog.Level {
	if exhausted {
		return slog.LevelError
	}
	return slog.LevelWarn
}
