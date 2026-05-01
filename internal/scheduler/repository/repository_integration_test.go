//go:build integration

package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	db "github.com/flamefks/scheduler-system/internal/postgres/queries"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

type testDB struct {
	Repo      *SchedulerRepository
	Pool      *pgxpool.Pool
	Queries   *db.Queries
	Container testcontainers.Container
}

func setupTestRepo(t *testing.T) *testDB {
	t.Helper()

	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(
		ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("scheduler_test"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = pgContainer.Terminate(ctx)
		t.Fatalf("failed to get connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		_ = pgContainer.Terminate(ctx)
		t.Fatalf("failed to create pgx pool: %v", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		_ = pgContainer.Terminate(ctx)
		t.Fatalf("failed to ping db: %v", err)
	}

	if err := runMigrations(pool); err != nil {
		pool.Close()
		_ = pgContainer.Terminate(ctx)
		t.Fatalf("failed to run migrations: %v", err)
	}

	q := db.New(pool)
	repo := NewSchedulerRepository(pool, q)

	t.Cleanup(func() {
		pool.Close()
		_ = pgContainer.Terminate(context.Background())
	})

	return &testDB{
		Repo:      repo,
		Pool:      pool,
		Queries:   q,
		Container: pgContainer,
	}
}

func runMigrations(pool *pgxpool.Pool) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}

	projectRoot := filepath.Clean(filepath.Join(wd, "../../.."))
	migrationsPath := filepath.Join(projectRoot, "migrations")

	driver, err := migratepgx.WithInstance(pool, &migratepgx.Config{})
	if err != nil {
		return fmt.Errorf("migrate driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("new migrate: %w", err)
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}

	return nil
}
