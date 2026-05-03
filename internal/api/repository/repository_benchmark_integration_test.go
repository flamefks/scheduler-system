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

	"github.com/flamefks/scheduler-system/internal/api/domain"
	db "github.com/flamefks/scheduler-system/internal/postgres/queries"
	"github.com/flamefks/scheduler-system/internal/shared/data"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

type benchDB struct {
	Repo      *Repository
	Pool      *pgxpool.Pool
	Queries   *db.Queries
	Container testcontainers.Container
}

func setupBenchRepo(b *testing.B) *benchDB {
	b.Helper()

	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(
		ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("scheduler_bench"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		b.Fatalf("failed to start postgres container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = pgContainer.Terminate(ctx)
		b.Fatalf("failed to get connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		_ = pgContainer.Terminate(ctx)
		b.Fatalf("failed to create pgx pool: %v", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		_ = pgContainer.Terminate(ctx)
		b.Fatalf("failed to ping db: %v", err)
	}

	if err := runBenchMigrations(connStr); err != nil {
		pool.Close()
		_ = pgContainer.Terminate(ctx)
		b.Fatalf("failed to run migrations: %v", err)
	}

	q := db.New(pool)
	repo := NewRepository(pool, q)

	b.Cleanup(func() {
		pool.Close()
		_ = pgContainer.Terminate(context.Background())
	})

	return &benchDB{
		Repo:      repo,
		Pool:      pool,
		Queries:   q,
		Container: pgContainer,
	}
}

func runBenchMigrations(connStr string) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}

	projectRoot := filepath.Clean(filepath.Join(wd, "../../.."))
	migrationsPath := filepath.Join(projectRoot, "sql", "migrations")

	m, err := migrate.New(
		"file://"+migrationsPath,
		connStr,
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

func seedJobForBench(b *testing.B, repo *Repository, name string) uuid.UUID {
	b.Helper()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	job := &data.Job{
		Name: name,
		Schedule: data.Schedule{
			RepeatIntervalSec: int32(60),
			TargetRuns:        int32(5),
			NextRunAt:         now.Add(1 * time.Hour),
		},
		FetcherConfig: data.IOConfig{
			Payload:   []byte(`{"kind":"fetch"}`),
			Headers:   []byte(`{"Authorization":"Bearer fetch-token"}`),
			TargetUrl: "https://fetch.example/api",
			Method:    "POST",
		},
		DeliverConfig: data.IOConfig{
			Payload:   []byte(`{"kind":"deliver"}`),
			Headers:   []byte(`{"Authorization":"Bearer deliver-token"}`),
			TargetUrl: "https://deliver.example/api",
			Method:    "POST",
		},
	}

	id, err := repo.CreateJob(ctx, job)
	if err != nil {
		b.Fatalf("seed CreateJob failed: %v", err)
	}
	if id == uuid.Nil {
		b.Fatal("seed returned nil uuid")
	}

	return id
}

func strPtrBench(s string) *string        { return &s }
func i32PtrBench(v int32) *int32          { return &v }
func timePtrBench(v time.Time) *time.Time { return &v }
func bytesPtrBench(v []byte) *[]byte      { return &v }

func BenchmarkRepository_CreateJob_Integration(b *testing.B) {
	tdb := setupBenchRepo(b)
	repo := tdb.Repo
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		now := time.Now().UTC().Truncate(time.Second)

		job := &data.Job{
			Name: fmt.Sprintf("bench-create-%d", i),
			Schedule: data.Schedule{
				RepeatIntervalSec: int32(60),
				TargetRuns:        int32(3),
				NextRunAt:         now.Add(1 * time.Hour),
			},
			FetcherConfig: data.IOConfig{
				Payload:   []byte(`{"fetch":1}`),
				Headers:   []byte(`{"Authorization":"Bearer fetch"}`),
				TargetUrl: "https://fetch.example",
				Method:    "POST",
			},
			DeliverConfig: data.IOConfig{
				Payload:   []byte(`{"deliver":1}`),
				Headers:   []byte(`{"Authorization":"Bearer deliver"}`),
				TargetUrl: "https://deliver.example",
				Method:    "POST",
			},
		}

		_, err := repo.CreateJob(ctx, job)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRepository_GetJobByID_Integration(b *testing.B) {
	tdb := setupBenchRepo(b)
	repo := tdb.Repo
	ctx := context.Background()

	id := seedJobForBench(b, repo, "bench-get")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := repo.GetJobByID(ctx, id)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRepository_UpdateScheduleStatus_Integration(b *testing.B) {
	tdb := setupBenchRepo(b)
	repo := tdb.Repo
	ctx := context.Background()

	id := seedJobForBench(b, repo, "bench-status")
	statuses := []string{"running", "idle"}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := repo.UpdateScheduleStatus(ctx, id, statuses[i%2])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRepository_PatchJob_NameSchedule_Integration(b *testing.B) {
	tdb := setupBenchRepo(b)
	repo := tdb.Repo
	ctx := context.Background()

	id := seedJobForBench(b, repo, "bench-patch-name-schedule")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("patched-%d", i)
		nextRunAt := time.Now().UTC().Truncate(time.Second).Add(time.Duration(i+1) * time.Minute)
		status := "running"
		if i%2 == 0 {
			status = "idle"
		}

		patch := &domain.PatchJobModel{
			Name: strPtrBench(name),
			Schedule: &domain.PatchScheduleModel{
				RepeatIntervalSec: i32PtrBench(int32(60 + i%100)),
				TargetRuns:        i32PtrBench(int32(10 + i%50)),
				NextRunAt:         timePtrBench(nextRunAt),
				Status:            strPtrBench(status),
			},
		}

		if err := repo.PatchJob(ctx, patch, id); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRepository_PatchJob_FetcherConfig_Integration(b *testing.B) {
	tdb := setupBenchRepo(b)
	repo := tdb.Repo
	ctx := context.Background()

	id := seedJobForBench(b, repo, "bench-patch-fetcher")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		payload := []byte(fmt.Sprintf(`{"fetch":"%d"}`, i))
		header := []byte(fmt.Sprintf(`{"Authorization":"Bearer fetch-%d"}`, i))
		url := fmt.Sprintf("https://fetch.example/%d", i)
		method := "PATCH"

		patch := &domain.PatchJobModel{
			FetcherConfig: &domain.PatchIOConfig{
				Payload:   bytesPtrBench(payload),
				Headers:   bytesPtrBench(header),
				TargetUrl: strPtrBench(url),
				Method:    strPtrBench(method),
			},
		}

		if err := repo.PatchJob(ctx, patch, id); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRepository_DeleteJob_Integration(b *testing.B) {
	tdb := setupBenchRepo(b)
	repo := tdb.Repo
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		now := time.Now().UTC().Truncate(time.Second)

		job := &data.Job{
			Name: fmt.Sprintf("bench-delete-%d", i),
			Schedule: data.Schedule{
				RepeatIntervalSec: int32(60),
				TargetRuns:        int32(1),
				NextRunAt:         now.Add(1 * time.Hour),
			},
			FetcherConfig: data.IOConfig{
				Payload:   []byte(`{"fetch":1}`),
				Headers:   []byte(`{"Authorization":"Bearer fetch"}`),
				TargetUrl: "https://fetch.example",
				Method:    "POST",
			},
			DeliverConfig: data.IOConfig{
				Payload:   []byte(`{"deliver":1}`),
				Headers:   []byte(`{"Authorization":"Bearer deliver"}`),
				TargetUrl: "https://deliver.example",
				Method:    "POST",
			},
		}

		id, err := repo.CreateJob(ctx, job)
		if err != nil {
			b.Fatal(err)
		}

		if err := repo.DeleteJob(ctx, id); err != nil {
			b.Fatal(err)
		}
	}
}
