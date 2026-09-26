package emaildelivery

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Rahmannugar/authlier/emailverification"
	"github.com/Rahmannugar/authlier/passwordreset"
	"github.com/Rahmannugar/consumel-server/internal/common/ids"
	"github.com/Rahmannugar/consumel-server/internal/infra/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	EventTypeQueued       = "email.delivery.queued.v1"
	TemplateVerification  = "email_verification"
	TemplatePasswordReset = "password_reset"
	payloadKeyVersion     = 1
)

type Payload struct {
	SubjectID        string    `json:"subjectId,omitempty"`
	Recipient        string    `json:"recipient"`
	Code             string    `json:"code,omitempty"`
	URL              string    `json:"url,omitempty"`
	OrganizationName string    `json:"organizationName,omitempty"`
	ProjectName      string    `json:"projectName,omitempty"`
	ExpiresAt        time.Time `json:"expiresAt"`
}

type Queue struct {
	pool *pgxpool.Pool
	aead cipher.AEAD
}

func NewQueue(pool *pgxpool.Pool, masterSecret []byte) (*Queue, error) {
	if len(masterSecret) < 32 {
		return nil, fmt.Errorf("email delivery encryption requires at least 32 bytes of key material")
	}
	mac := hmac.New(sha256.New, masterSecret)
	// Derive a purpose-specific encryption key so encrypted delivery payloads do
	// not use the configured authentication secret as raw AES key material.
	_, _ = mac.Write([]byte("consumel/email-delivery/v1"))
	block, err := aes.NewCipher(mac.Sum(nil))
	if err != nil {
		return nil, fmt.Errorf("create email delivery cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create email delivery AEAD: %w", err)
	}
	return &Queue{pool: pool, aead: aead}, nil
}

func (queue *Queue) SendVerification(
	ctx context.Context,
	message emailverification.Message,
) error {
	return queue.Enqueue(ctx, TemplateVerification, Payload{
		SubjectID: message.UserID, Recipient: message.Email, Code: message.Code, ExpiresAt: message.ExpiresAt,
	})
}

func (queue *Queue) SendPasswordReset(
	ctx context.Context,
	message passwordreset.Message,
) error {
	return queue.Enqueue(ctx, TemplatePasswordReset, Payload{
		SubjectID: message.UserID, Recipient: message.Email, URL: message.URL, ExpiresAt: message.ExpiresAt,
	})
}

func (queue *Queue) Enqueue(ctx context.Context, templateName string, payload Payload) error {
	tx, err := queue.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin email delivery transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := queue.EnqueueTx(ctx, tx, templateName, payload); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit email delivery transaction: %w", err)
	}
	return nil
}

func (queue *Queue) EnqueueTx(
	ctx context.Context,
	tx pgx.Tx,
	templateName string,
	payload Payload,
) error {
	deliveryID, err := ids.New()
	if err != nil {
		return fmt.Errorf("generate email delivery ID: %w", err)
	}
	eventID, err := ids.New()
	if err != nil {
		return fmt.Errorf("generate email event ID: %w", err)
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode email delivery payload: %w", err)
	}
	nonce := make([]byte, queue.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("generate email payload nonce: %w", err)
	}
	// Binding ciphertext to its delivery ID prevents one row's payload from
	// being copied into another valid delivery record.
	ciphertext := queue.aead.Seal(nil, nonce, encoded, deliveryID[:])

	// Delivery state and its transport event commit together. PostgreSQL remains
	// authoritative even when LISTEN/NOTIFY or Redis is temporarily unavailable.
	if _, err := tx.Exec(ctx, `INSERT INTO email_deliveries
		(id, template, encrypted_payload, payload_nonce, payload_key_version, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		deliveryID, templateName, ciphertext, nonce, payloadKeyVersion, payload.ExpiresAt,
	); err != nil {
		return fmt.Errorf("insert email delivery: %w", err)
	}
	eventPayload, err := json.Marshal(map[string]string{"deliveryId": deliveryID.String()})
	if err != nil {
		return fmt.Errorf("encode email event payload: %w", err)
	}
	if err := events.Insert(ctx, tx, events.Event{
		ID: eventID, Type: EventTypeQueued, AggregateType: "email_delivery",
		AggregateID: deliveryID, Payload: eventPayload, OccurredAt: time.Now().UTC(),
	}); err != nil {
		return err
	}
	return nil
}

func (queue *Queue) Decrypt(deliveryID uuid.UUID, nonce, ciphertext []byte) (Payload, error) {
	plaintext, err := queue.aead.Open(nil, nonce, ciphertext, deliveryID[:])
	if err != nil {
		return Payload{}, fmt.Errorf("decrypt email delivery payload: %w", err)
	}
	var payload Payload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return Payload{}, fmt.Errorf("decode email delivery payload: %w", err)
	}
	return payload, nil
}

var _ emailverification.Sender = (*Queue)(nil)
var _ passwordreset.Sender = (*Queue)(nil)
