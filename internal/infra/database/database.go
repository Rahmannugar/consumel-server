package database

import (
	"context"
	"fmt"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	minimumConnections int32 = 2
)

// Open builds an instrumented pool capped at maxConns for this process.
func Open(ctx context.Context, connectionString string, maxConns int32) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		return nil, fmt.Errorf("parse connection string: %w", err)
	}
	config.MaxConns = maxConns
	config.MinConns = minimumConnections
	config.ConnConfig.Tracer = otelpgx.NewTracer(otelpgx.WithDisableSQLStatementInAttributes())
	// PostgreSQL normally switches a cached prepared statement to a generic
	// plan after a few executions. Generic plans cannot see the caller's range
	// values, so a parameterized time window is estimated with a fixed 0.5%
	// selectivity and PostgreSQL can replace a whole-table aggregate with a
	// nested loop over the operation table. Consumel keeps prepared statements
	// but always plans with the current parameters.
	config.ConnConfig.RuntimeParams["plan_cache_mode"] = "force_custom_plan"

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	if err := otelpgx.RecordStats(pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("record database pool metrics: %w", err)
	}

	return pool, nil
}
