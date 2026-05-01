package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/flamefks/scheduler-system/internal/api/domain"
	"github.com/flamefks/scheduler-system/internal/shared/data"
	"github.com/google/uuid"
)

type mockRepo struct {
	createJobFn            func(ctx context.Context, job *data.Job) (uuid.UUID, error)
	deleteJobFn            func(ctx context.Context, id uuid.UUID) error
	getJobByIDFn           func(ctx context.Context, id uuid.UUID) (*data.Job, error)
	patchJobFn             func(ctx context.Context, patch *domain.PatchJobModel, id uuid.UUID) error
	updateScheduleStatusFn func(ctx context.Context, id uuid.UUID, status string) error
}

func (m *mockRepo) CreateJob(ctx context.Context, job *data.Job) (uuid.UUID, error) {
	return m.createJobFn(ctx, job)
}

func (m *mockRepo) DeleteJob(ctx context.Context, id uuid.UUID) error {
	return m.deleteJobFn(ctx, id)
}

func (m *mockRepo) GetJobByID(ctx context.Context, id uuid.UUID) (*data.Job, error) {
	return m.getJobByIDFn(ctx, id)
}

func (m *mockRepo) PatchJob(ctx context.Context, patch *domain.PatchJobModel, id uuid.UUID) error {
	return m.patchJobFn(ctx, patch, id)
}

func (m *mockRepo) UpdateScheduleStatus(ctx context.Context, id uuid.UUID, status string) error {
	return m.updateScheduleStatusFn(ctx, id, status)
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNewApiService(t *testing.T) {
	repo := &mockRepo{}
	svc := NewApiService(testLogger(), repo)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.repo == nil {
		t.Fatal("expected repo to be set")
	}
	if svc.logger == nil {
		t.Fatal("expected logger to be set")
	}
}

func TestApiService_CreateJob(t *testing.T) {
	job := &data.Job{Name: "job-1"}
	expectedID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{
			createJobFn: func(ctx context.Context, got *data.Job) (uuid.UUID, error) {
				if got != job {
					t.Fatal("unexpected job pointer")
				}
				return expectedID, nil
			},
		}

		svc := NewApiService(testLogger(), repo)

		gotID, err := svc.CreateJob(context.Background(), job)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotID != expectedID {
			t.Fatalf("expected %s, got %s", expectedID, gotID)
		}
	})

	t.Run("repo error", func(t *testing.T) {
		repoErr := errors.New("insert failed")
		repo := &mockRepo{
			createJobFn: func(ctx context.Context, got *data.Job) (uuid.UUID, error) {
				return uuid.Nil, repoErr
			},
		}

		svc := NewApiService(testLogger(), repo)

		gotID, err := svc.CreateJob(context.Background(), job)
		if err == nil {
			t.Fatal("expected error")
		}
		if gotID != uuid.Nil {
			t.Fatalf("expected nil uuid, got %s", gotID)
		}
		if !strings.Contains(err.Error(), "Error on creating job") {
			t.Fatalf("unexpected error text: %v", err)
		}
		if !errors.Is(err, repoErr) {
			t.Fatal("expected wrapped repo error")
		}
	})
}

func TestApiService_DeleteJob(t *testing.T) {
	jobID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{
			deleteJobFn: func(ctx context.Context, id uuid.UUID) error {
				if id != jobID {
					t.Fatalf("expected %s, got %s", jobID, id)
				}
				return nil
			},
		}

		svc := NewApiService(testLogger(), repo)

		if err := svc.DeleteJob(context.Background(), jobID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("repo error", func(t *testing.T) {
		repoErr := errors.New("delete failed")
		repo := &mockRepo{
			deleteJobFn: func(ctx context.Context, id uuid.UUID) error {
				return repoErr
			},
		}

		svc := NewApiService(testLogger(), repo)

		err := svc.DeleteJob(context.Background(), jobID)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "Error on removing job") {
			t.Fatalf("unexpected error text: %v", err)
		}
		if !errors.Is(err, repoErr) {
			t.Fatal("expected wrapped repo error")
		}
	})
}

func TestApiService_GetJobByID(t *testing.T) {
	jobID := uuid.New()
	expectedJob := &data.Job{Name: "job-1"}

	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{
			getJobByIDFn: func(ctx context.Context, id uuid.UUID) (*data.Job, error) {
				if id != jobID {
					t.Fatalf("expected %s, got %s", jobID, id)
				}
				return expectedJob, nil
			},
		}

		svc := NewApiService(testLogger(), repo)

		got, err := svc.GetJobByID(context.Background(), jobID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != expectedJob {
			t.Fatal("unexpected job pointer")
		}
	})

	t.Run("repo error", func(t *testing.T) {
		repoErr := errors.New("get failed")
		repo := &mockRepo{
			getJobByIDFn: func(ctx context.Context, id uuid.UUID) (*data.Job, error) {
				return nil, repoErr
			},
		}

		svc := NewApiService(testLogger(), repo)

		got, err := svc.GetJobByID(context.Background(), jobID)
		if err == nil {
			t.Fatal("expected error")
		}
		if got != nil {
			t.Fatal("expected nil job")
		}
		if !strings.Contains(err.Error(), "error getting job by id") {
			t.Fatalf("unexpected error text: %v", err)
		}
	})
}

func TestApiService_PatchJob(t *testing.T) {
	jobID := uuid.New()
	patch := &domain.PatchJobModel{}

	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{
			patchJobFn: func(ctx context.Context, gotPatch *domain.PatchJobModel, id uuid.UUID) error {
				if gotPatch != patch {
					t.Fatal("unexpected patch pointer")
				}
				if id != jobID {
					t.Fatalf("expected %s, got %s", jobID, id)
				}
				return nil
			},
		}

		svc := NewApiService(testLogger(), repo)

		if err := svc.PatchJob(context.Background(), patch, jobID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("repo error", func(t *testing.T) {
		repoErr := errors.New("patch failed")
		repo := &mockRepo{
			patchJobFn: func(ctx context.Context, gotPatch *domain.PatchJobModel, id uuid.UUID) error {
				return repoErr
			},
		}

		svc := NewApiService(testLogger(), repo)

		err := svc.PatchJob(context.Background(), patch, jobID)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "error patching job by id") {
			t.Fatalf("unexpected error text: %v", err)
		}
	})
}

func TestApiService_UpdateJobStatus(t *testing.T) {
	jobID := uuid.New()
	status := "running"

	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{
			updateScheduleStatusFn: func(ctx context.Context, id uuid.UUID, gotStatus string) error {
				if id != jobID {
					t.Fatalf("expected %s, got %s", jobID, id)
				}
				if gotStatus != status {
					t.Fatalf("expected %s, got %s", status, gotStatus)
				}
				return nil
			},
		}

		svc := NewApiService(testLogger(), repo)

		if err := svc.UpdateJobStatus(context.Background(), jobID, status); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("repo error", func(t *testing.T) {
		repoErr := errors.New("status update failed")
		repo := &mockRepo{
			updateScheduleStatusFn: func(ctx context.Context, id uuid.UUID, gotStatus string) error {
				return repoErr
			},
		}

		svc := NewApiService(testLogger(), repo)

		err := svc.UpdateJobStatus(context.Background(), jobID, status)
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, repoErr) {
			t.Fatal("expected original repo error")
		}
	})
}
