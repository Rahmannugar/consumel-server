package authentication

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Rahmannugar/authlier"
	"github.com/Rahmannugar/authlier/emailverification"
	"github.com/Rahmannugar/authlier/sessiontoken"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/singleflight"
)

const sessionAdvisoryLockNamespace = 639904427

type AuthlierDatabase struct {
	database           authlier.Database
	sessions           sessiontoken.Store
	emailVerifications emailverification.Store
}

func NewAuthlierDatabase(
	database authlier.Database,
	pool *pgxpool.Pool,
	cache sessiontoken.Cache,
	emailVerifications emailverification.Store,
	maximumActiveSessions int,
	logger *slog.Logger,
) (*AuthlierDatabase, error) {
	if database == nil || pool == nil || cache == nil || emailVerifications == nil || logger == nil {
		return nil, fmt.Errorf("Authlier database, PostgreSQL pool, session cache, email verification store, and logger are required")
	}
	if maximumActiveSessions < 1 {
		return nil, fmt.Errorf("maximum active sessions must be positive")
	}
	stores := database.Stores()
	if stores.Sessions == nil {
		return nil, fmt.Errorf("Authlier session store is required")
	}

	return &AuthlierDatabase{
		database:           database,
		emailVerifications: emailVerifications,
		sessions: &postgresSessionStore{
			store:                 stores.Sessions,
			pool:                  pool,
			cache:                 cache,
			maximumActiveSessions: maximumActiveSessions,
			logger:                logger,
		},
	}, nil
}

func (database *AuthlierDatabase) Migrate(ctx context.Context) error {
	return database.database.Migrate(ctx)
}

func (database *AuthlierDatabase) Stores() authlier.Stores {
	stores := database.database.Stores()
	stores.Sessions = database.sessions
	stores.EmailVerification = database.emailVerifications
	return stores
}

type postgresSessionStore struct {
	store                 sessiontoken.Store
	pool                  *pgxpool.Pool
	cache                 sessiontoken.Cache
	maximumActiveSessions int
	logger                *slog.Logger
	lookups               singleflight.Group
}

func (store *postgresSessionStore) Create(
	ctx context.Context,
	record sessiontoken.Record,
) error {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin capped session creation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Serialize each subject's session creation across API instances so the
	// three-session limit cannot race during concurrent sign-ins.
	if _, err := tx.Exec(
		ctx,
		"SELECT pg_advisory_xact_lock($1, hashtext($2))",
		sessionAdvisoryLockNamespace,
		record.SubjectID,
	); err != nil {
		return fmt.Errorf("lock Authlier subject sessions: %w", err)
	}

	if _, err := tx.Exec(ctx, `INSERT INTO authlier_sessions
		(id, token_hash, subject_id, created_at, expires_at, extended_at, revoked_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		record.ID,
		record.TokenHash[:],
		record.SubjectID,
		record.CreatedAt,
		record.ExpiresAt,
		record.ExtendedAt,
		record.RevokedAt,
	); err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) && postgresError.Code == "23505" {
			return sessiontoken.ErrConflict
		}
		return fmt.Errorf("insert Authlier session: %w", err)
	}

	revokedRows, err := tx.Query(ctx, `UPDATE authlier_sessions
		SET revoked_at = $1
		WHERE token_hash IN (
			SELECT token_hash
			FROM authlier_sessions
			WHERE subject_id = $2
			  AND revoked_at IS NULL
			  AND expires_at > $1
			ORDER BY created_at DESC, id DESC
			OFFSET $3
		)
		RETURNING token_hash`, record.CreatedAt, record.SubjectID, store.maximumActiveSessions)
	if err != nil {
		return fmt.Errorf("revoke oldest Authlier sessions: %w", err)
	}
	revoked := make([]sessiontoken.TokenHash, 0, 1)
	for revokedRows.Next() {
		var hashBytes []byte
		if err := revokedRows.Scan(&hashBytes); err != nil {
			revokedRows.Close()
			return fmt.Errorf("read revoked Authlier session: %w", err)
		}
		var tokenHash sessiontoken.TokenHash
		copy(tokenHash[:], hashBytes)
		revoked = append(revoked, tokenHash)
	}
	if err := revokedRows.Err(); err != nil {
		revokedRows.Close()
		return fmt.Errorf("iterate revoked Authlier sessions: %w", err)
	}
	revokedRows.Close()

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit capped session creation: %w", err)
	}

	for _, tokenHash := range revoked {
		if err := store.cache.Delete(ctx, tokenHash); err != nil {
			store.logger.WarnContext(ctx, "session cache invalidation deferred to expiry",
				"event", "authentication.session.cache_invalidation_failed",
				"outcome", "degraded",
				"error", err,
			)
		}
	}
	return nil
}

func (store *postgresSessionStore) FindByTokenHash(
	ctx context.Context,
	tokenHash sessiontoken.TokenHash,
) (sessiontoken.Record, error) {
	key := hex.EncodeToString(tokenHash[:])
	resolved, err, _ := store.lookups.Do(key, func() (any, error) {
		return store.store.FindByTokenHash(ctx, tokenHash)
	})
	if err != nil {
		return sessiontoken.Record{}, err
	}
	return resolved.(sessiontoken.Record), nil
}

func (store *postgresSessionStore) ListBySubject(
	ctx context.Context,
	subjectID string,
) ([]sessiontoken.Record, error) {
	return store.store.ListBySubject(ctx, subjectID)
}

func (store *postgresSessionStore) Extend(
	ctx context.Context,
	tokenHash sessiontoken.TokenHash,
	extendedAt time.Time,
	expiresAt time.Time,
) (sessiontoken.Record, error) {
	return store.store.Extend(ctx, tokenHash, extendedAt, expiresAt)
}

func (store *postgresSessionStore) Rotate(
	ctx context.Context,
	current sessiontoken.TokenHash,
	replacement sessiontoken.Record,
	rotatedAt time.Time,
) error {
	return store.store.Rotate(ctx, current, replacement, rotatedAt)
}

func (store *postgresSessionStore) Revoke(
	ctx context.Context,
	tokenHash sessiontoken.TokenHash,
	revokedAt time.Time,
) error {
	return store.store.Revoke(ctx, tokenHash, revokedAt)
}

func (store *postgresSessionStore) RevokeAll(
	ctx context.Context,
	subjectID string,
	revokedAt time.Time,
) ([]sessiontoken.Record, error) {
	return store.store.RevokeAll(ctx, subjectID, revokedAt)
}

var _ authlier.Database = (*AuthlierDatabase)(nil)
var _ sessiontoken.Store = (*postgresSessionStore)(nil)
