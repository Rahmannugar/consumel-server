//go:build integration

package testdb

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Rahmannugar/consumel-server/internal/infra/database"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/tern/v2/migrate"
	"github.com/testcontainers/testcontainers-go"
	containerpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func OpenMigratedDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := t.Context()
	container, err := containerpostgres.Run(
		ctx,
		"postgres:17-alpine",
		containerpostgres.WithDatabase("consumel_test"),
		containerpostgres.WithUsername("consumel"),
		containerpostgres.WithPassword("consumel"),
		containerpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start PostgreSQL container: %v", err)
	}
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Errorf("terminate PostgreSQL container: %v", err)
		}
	})

	connectionString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get PostgreSQL connection string: %v", err)
	}

	pool, err := database.Open(ctx, connectionString)
	if err != nil {
		t.Fatalf("open PostgreSQL pool: %v", err)
	}
	t.Cleanup(pool.Close)

	connection, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire migration connection: %v", err)
	}
	defer connection.Release()

	migrator, err := migrate.NewMigrator(ctx, connection.Conn(), "public.schema_version")
	if err != nil {
		t.Fatalf("create migrator: %v", err)
	}
	if err := migrator.LoadMigrations(os.DirFS(migrationsPath(t))); err != nil {
		t.Fatalf("load migrations: %v", err)
	}
	if err := migrator.Migrate(ctx); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	return pool
}

func migrationsPath(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve database test path")
	}
	return filepath.Join(filepath.Dir(filename), "..", "migrations")
}
