package authentication

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Rahmannugar/authlier/emailverification"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const replaceVerificationScript = `
if redis.call("EXISTS", KEYS[2]) == 1 then
    return 0
end

local previous_proof_key = redis.call("GET", KEYS[1])
if previous_proof_key then
    redis.call("DEL", previous_proof_key)
end

redis.call("SET", KEYS[1], KEYS[2], "PX", ARGV[2])
redis.call("SET", KEYS[2], ARGV[1], "PX", ARGV[2])
return 1
`

const clearVerificationOwnerScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
    redis.call("DEL", KEYS[1])
end
return 1
`

type verificationChallenge struct {
	UserID    string    `json:"userId"`
	EmailHash string    `json:"emailHash"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type RedisEmailVerificationStore struct {
	pool   *pgxpool.Pool
	redis  redis.UniversalClient
	secret []byte
	prefix string
	now    func() time.Time
}

func NewRedisEmailVerificationStore(
	pool *pgxpool.Pool,
	redisClient redis.UniversalClient,
	secret []byte,
	keyPrefix string,
) (*RedisEmailVerificationStore, error) {
	if pool == nil || redisClient == nil {
		return nil, fmt.Errorf("PostgreSQL pool and Redis client are required")
	}
	if len(secret) < 32 {
		return nil, fmt.Errorf("email verification key secret must contain at least 32 bytes")
	}
	keyPrefix = strings.TrimSpace(keyPrefix)
	if keyPrefix == "" {
		keyPrefix = "consumel:auth:email-verification"
	}
	return &RedisEmailVerificationStore{
		pool:   pool,
		redis:  redisClient,
		secret: append([]byte(nil), secret...),
		prefix: "{" + keyPrefix + "}",
		now:    time.Now,
	}, nil
}

func (store *RedisEmailVerificationStore) FindUserByEmail(
	ctx context.Context,
	normalizedEmail string,
) (emailverification.User, error) {
	var user emailverification.User
	err := store.pool.QueryRow(ctx,
		"SELECT id, email, email_verified FROM authlier_users WHERE email = $1",
		normalizedEmail,
	).Scan(&user.ID, &user.Email, &user.Verified)
	if errors.Is(err, pgx.ErrNoRows) {
		return emailverification.User{}, emailverification.ErrNotFound
	}
	return user, err
}

func (store *RedisEmailVerificationStore) Issue(
	ctx context.Context,
	record emailverification.Record,
) error {
	ttl := record.ExpiresAt.Sub(store.now().UTC())
	if ttl <= 0 {
		return emailverification.ErrInactiveToken
	}
	challenge, err := json.Marshal(verificationChallenge{
		UserID:    record.UserID,
		EmailHash: store.emailHash(record.Email),
		ExpiresAt: record.ExpiresAt,
	})
	if err != nil {
		return fmt.Errorf("encode email verification challenge: %w", err)
	}

	proofKey := store.proofKey(record.TokenHash)
	replaced, err := store.redis.Eval(
		ctx,
		replaceVerificationScript,
		[]string{store.userKey(record.UserID), proofKey},
		challenge,
		ttl.Milliseconds(),
	).Int64()
	if err != nil {
		return fmt.Errorf("store email verification challenge: %w", err)
	}
	if replaced != 1 {
		return emailverification.ErrConflict
	}
	return nil
}

func (store *RedisEmailVerificationStore) Verify(
	ctx context.Context,
	tokenHash emailverification.TokenHash,
	verifiedAt time.Time,
) (emailverification.User, error) {
	proofKey := store.proofKey(tokenHash)
	encoded, err := store.redis.GetDel(ctx, proofKey).Bytes()
	if errors.Is(err, redis.Nil) {
		return emailverification.User{}, emailverification.ErrNotFound
	}
	if err != nil {
		return emailverification.User{}, fmt.Errorf("consume email verification challenge: %w", err)
	}

	var challenge verificationChallenge
	if err := json.Unmarshal(encoded, &challenge); err != nil {
		return emailverification.User{}, emailverification.ErrInvalidRecord
	}
	_, _ = store.redis.Eval(
		ctx,
		clearVerificationOwnerScript,
		[]string{store.userKey(challenge.UserID)},
		proofKey,
	).Result()

	if challenge.UserID == "" || challenge.EmailHash == "" ||
		!verifiedAt.Before(challenge.ExpiresAt) {
		return emailverification.User{}, emailverification.ErrInactiveToken
	}

	var user emailverification.User
	err = store.pool.QueryRow(ctx,
		"SELECT id, email, email_verified FROM authlier_users WHERE id = $1",
		challenge.UserID,
	).Scan(&user.ID, &user.Email, &user.Verified)
	if errors.Is(err, pgx.ErrNoRows) {
		return emailverification.User{}, emailverification.ErrNotFound
	}
	if err != nil {
		return emailverification.User{}, fmt.Errorf("load user for email verification: %w", err)
	}
	if user.Verified || store.emailHash(user.Email) != challenge.EmailHash {
		return emailverification.User{}, emailverification.ErrInactiveToken
	}

	command, err := store.pool.Exec(ctx,
		"UPDATE authlier_users SET email_verified = true WHERE id = $1 AND email = $2 AND email_verified = false",
		user.ID,
		user.Email,
	)
	if err != nil {
		return emailverification.User{}, fmt.Errorf("mark Authlier email verified: %w", err)
	}
	if command.RowsAffected() != 1 {
		return emailverification.User{}, emailverification.ErrInactiveToken
	}
	user.Verified = true
	return user, nil
}

func (store *RedisEmailVerificationStore) userKey(userID string) string {
	mac := hmac.New(sha256.New, store.secret)
	_, _ = mac.Write([]byte("user\x00" + userID))
	return store.prefix + ":user:" + hex.EncodeToString(mac.Sum(nil))
}

func (store *RedisEmailVerificationStore) proofKey(tokenHash emailverification.TokenHash) string {
	return store.prefix + ":proof:" + hex.EncodeToString(tokenHash[:])
}

func (store *RedisEmailVerificationStore) emailHash(email string) string {
	mac := hmac.New(sha256.New, store.secret)
	_, _ = mac.Write([]byte("email\x00" + email))
	return hex.EncodeToString(mac.Sum(nil))
}

var _ emailverification.Store = (*RedisEmailVerificationStore)(nil)
