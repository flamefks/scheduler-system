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

type testDB struct {
	Repo      *Repository
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

	if err := runMigrations(connStr); err != nil {
		pool.Close()
		_ = pgContainer.Terminate(ctx)
		t.Fatalf("failed to run migrations: %v", err)
	}

	q := db.New(pool)
	repo := NewRepository(pool, q)

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

func runMigrations(connStr string) error {
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

func seedJob(t *testing.T, repo *Repository, name string) uuid.UUID {
	t.Helper()

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
		t.Fatalf("seed CreateJob failed: %v", err)
	}
	if id == uuid.Nil {
		t.Fatal("seed returned nil uuid")
	}

	return id
}

func strPtr(s string) *string        { return &s }
func i32Ptr(v int32) *int32          { return &v }
func timePtr(v time.Time) *time.Time { return &v }
func bytesPtr(v []byte) *[]byte      { return &v }

func TestRepository_CreateJob_GetJobByID_DeleteJob_Integration(t *testing.T) {
	tdb := setupTestRepo(t)
	repo := tdb.Repo
	ctx := context.Background()

	nextRunAt := time.Now().UTC().Truncate(time.Second).Add(2 * time.Hour)

	job := &data.Job{
		Name: "job-create-get-delete",
		Schedule: data.Schedule{
			RepeatIntervalSec: int32(120),
			TargetRuns:        int32(10),
			NextRunAt:         nextRunAt,
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
			Method:    "PUT",
		},
	}

	id, err := repo.CreateJob(ctx, job)
	if err != nil {
		t.Fatalf("CreateJob error: %v", err)
	}
	if id == uuid.Nil {
		t.Fatal("expected non-nil uuid")
	}

	got, err := repo.GetJobByID(ctx, id)
	if err != nil {
		t.Fatalf("GetJobByID error: %v", err)
	}

	if got.ID != id {
		t.Fatalf("expected id %s, got %s", id, got.ID)
	}
	if got.Name != job.Name {
		t.Fatalf("expected name %q, got %q", job.Name, got.Name)
	}
	if got.Schedule.RepeatIntervalSec != job.Schedule.RepeatIntervalSec {
		t.Fatalf("expected repeat_interval_sec %d, got %d", job.Schedule.RepeatIntervalSec, got.Schedule.RepeatIntervalSec)
	}
	if got.Schedule.TargetRuns != job.Schedule.TargetRuns {
		t.Fatalf("expected target_runs %d, got %d", job.Schedule.TargetRuns, got.Schedule.TargetRuns)
	}
	if !got.Schedule.NextRunAt.Equal(job.Schedule.NextRunAt) {
		t.Fatalf("expected next_run_at %v, got %v", job.Schedule.NextRunAt, got.Schedule.NextRunAt)
	}

	if string(got.FetcherConfig.Payload) != string(job.FetcherConfig.Payload) {
		t.Fatalf("unexpected fetcher payload: %s", string(got.FetcherConfig.Payload))
	}
	if string(got.DeliverConfig.Payload) != string(job.DeliverConfig.Payload) {
		t.Fatalf("unexpected deliver payload: %s", string(got.DeliverConfig.Payload))
	}
	if got.FetcherConfig.TargetUrl != job.FetcherConfig.TargetUrl {
		t.Fatalf("unexpected fetcher target url: %s", got.FetcherConfig.TargetUrl)
	}
	if got.DeliverConfig.TargetUrl != job.DeliverConfig.TargetUrl {
		t.Fatalf("unexpected deliver target url: %s", got.DeliverConfig.TargetUrl)
	}

	if err := repo.DeleteJob(ctx, id); err != nil {
		t.Fatalf("DeleteJob error: %v", err)
	}

	_, err = repo.GetJobByID(ctx, id)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestRepository_UpdateScheduleStatus_Integration(t *testing.T) {
	tdb := setupTestRepo(t)
	repo := tdb.Repo
	ctx := context.Background()

	id := seedJob(t, repo, "job-status-update")

	if err := repo.UpdateScheduleStatus(ctx, id, "running"); err != nil {
		t.Fatalf("UpdateScheduleStatus error: %v", err)
	}

	got, err := repo.GetJobByID(ctx, id)
	if err != nil {
		t.Fatalf("GetJobByID error: %v", err)
	}

	if got.Schedule.Status != "running" {
		t.Fatalf("expected status running, got %s", got.Schedule.Status)
	}
}

func TestRepository_UpdateScheduleStatus_InvalidStatus_Integration(t *testing.T) {
	tdb := setupTestRepo(t)
	repo := tdb.Repo
	ctx := context.Background()

	id := seedJob(t, repo, "job-invalid-status")

	err := repo.UpdateScheduleStatus(ctx, id, "lolwat")
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestRepository_PatchJob_NameScheduleFetcher_Integration(t *testing.T) {
	tdb := setupTestRepo(t)
	repo := tdb.Repo
	ctx := context.Background()

	id := seedJob(t, repo, "job-before-patch")

	newName := "job-after-patch"
	newNextRunAt := time.Now().UTC().Truncate(time.Second).Add(5 * time.Hour)
	newStatus := "disabled"
	newFetcherPayload := []byte(`{"fetch":"patched"}`)
	newFetcherHeader := []byte(`{"Authorization":"Bearer patched-fetch"}`)
	newFetcherURL := "https://fetch.example/patched"
	newFetcherMethod := "PATCH"

	patch := &domain.PatchJobModel{
		Name: strPtr(newName),
		Schedule: &domain.PatchScheduleModel{
			TargetRuns:        i32Ptr(99),
			RepeatIntervalSec: i32Ptr(777),
			NextRunAt:         timePtr(newNextRunAt),
			Status:            strPtr(newStatus),
		},
		FetcherConfig: &domain.PatchIOConfig{
			Payload:   bytesPtr(newFetcherPayload),
			Headers:   bytesPtr(newFetcherHeader),
			TargetUrl: strPtr(newFetcherURL),
			Method:    strPtr(newFetcherMethod),
		},
	}

	if err := repo.PatchJob(ctx, patch, id); err != nil {
		t.Fatalf("PatchJob error: %v", err)
	}

	got, err := repo.GetJobByID(ctx, id)
	if err != nil {
		t.Fatalf("GetJobByID error: %v", err)
	}

	if got.Name != newName {
		t.Fatalf("expected name %q, got %q", newName, got.Name)
	}
	if got.Schedule.TargetRuns != 99 {
		t.Fatalf("expected target_runs 99, got %d", got.Schedule.TargetRuns)
	}
	if got.Schedule.RepeatIntervalSec != 777 {
		t.Fatalf("expected repeat_interval_sec 777, got %d", got.Schedule.RepeatIntervalSec)
	}
	if got.Schedule.Status != newStatus {
		t.Fatalf("expected status %q, got %q", newStatus, got.Schedule.Status)
	}
	if !got.Schedule.NextRunAt.Equal(newNextRunAt) {
		t.Fatalf("expected next_run_at %v, got %v", newNextRunAt, got.Schedule.NextRunAt)
	}

	if string(got.FetcherConfig.Payload) != string(newFetcherPayload) {
		t.Fatalf("expected fetcher payload %s, got %s", string(newFetcherPayload), string(got.FetcherConfig.Payload))
	}
	if string(got.FetcherConfig.Headers) != string(newFetcherHeader) {
		t.Fatalf("expected fetcher header %s, got %s", string(newFetcherHeader), string(got.FetcherConfig.Headers))
	}
	if got.FetcherConfig.TargetUrl != newFetcherURL {
		t.Fatalf("expected fetcher target url %s, got %s", newFetcherURL, got.FetcherConfig.TargetUrl)
	}
	if got.FetcherConfig.Method != newFetcherMethod {
		t.Fatalf("expected fetcher method %s, got %s", newFetcherMethod, got.FetcherConfig.Method)
	}
}

func TestRepository_PatchJob_DeliverConfig_Integration(t *testing.T) {
	tdb := setupTestRepo(t)
	repo := tdb.Repo
	ctx := context.Background()

	id := seedJob(t, repo, "job-deliver-patch")

	newDeliverPayload := []byte(`{"deliver":"patched"}`)
	newDeliverHeader := []byte(`{"Authorization":"Bearer patched-deliver"}`)
	newDeliverURL := "https://deliver.example/patched"
	newDeliverMethod := "PATCH"

	patch := &domain.PatchJobModel{
		DeliverConfig: &domain.PatchIOConfig{
			Payload:   bytesPtr(newDeliverPayload),
			Headers:   bytesPtr(newDeliverHeader),
			TargetUrl: strPtr(newDeliverURL),
			Method:    strPtr(newDeliverMethod),
		},
	}

	if err := repo.PatchJob(ctx, patch, id); err != nil {
		t.Fatalf("PatchJob error: %v", err)
	}

	got, err := repo.GetJobByID(ctx, id)
	if err != nil {
		t.Fatalf("GetJobByID error: %v", err)
	}

	if string(got.DeliverConfig.Payload) != string(newDeliverPayload) {
		t.Fatalf("expected deliver payload %s, got %s", string(newDeliverPayload), string(got.DeliverConfig.Payload))
	}
	if string(got.DeliverConfig.Headers) != string(newDeliverHeader) {
		t.Fatalf("expected deliver header %s, got %s", string(newDeliverHeader), string(got.DeliverConfig.Headers))
	}
	if got.DeliverConfig.TargetUrl != newDeliverURL {
		t.Fatalf("expected deliver target url %s, got %s", newDeliverURL, got.DeliverConfig.TargetUrl)
	}
	if got.DeliverConfig.Method != newDeliverMethod {
		t.Fatalf("expected deliver method %s, got %s", newDeliverMethod, got.DeliverConfig.Method)
	}
}
