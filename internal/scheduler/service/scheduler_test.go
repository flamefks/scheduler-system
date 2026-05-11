package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/flamefks/scheduler-system/internal/shared/data"
	"github.com/google/uuid"
)

type mockSchedulerRepo struct {
	claimNextJobsFn          func(ctx context.Context, jobBatchSize int) ([]uuid.UUID, error)
	resetHungMessageFn       func(ctx context.Context, scheduleJobTimeout int, procJobTimeout int) error
	switchToDisabledIfNeedFn func(ctx context.Context) error
}

func (m *mockSchedulerRepo) ClaimNextJobs(ctx context.Context, jobBatchSize int) ([]uuid.UUID, error) {
	return m.claimNextJobsFn(ctx, jobBatchSize)
}

func (m *mockSchedulerRepo) ResetHungMessage(ctx context.Context, scheduleJobTimeout int, procJobTimeout int) error {
	return m.resetHungMessageFn(ctx, scheduleJobTimeout, procJobTimeout)
}

func (m *mockSchedulerRepo) SwitchToDisabledIfNeed(ctx context.Context) error {
	return m.switchToDisabledIfNeedFn(ctx)
}

type mockSchedulerPublisher struct {
	publishFn func(ctx context.Context, subject string, payload []byte, headers map[string]string) error
}

func (m *mockSchedulerPublisher) Publish(ctx context.Context, subject string, payload []byte, headers map[string]string) error {
	return m.publishFn(ctx, subject, payload, headers)
}

func schedulerTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNewSchedulerService(t *testing.T) {
	repo := &mockSchedulerRepo{}
	publisher := &mockSchedulerPublisher{}

	svc := NewSchedulerService(schedulerTestLogger(), repo, publisher)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.repo == nil {
		t.Fatal("expected repo to be set")
	}
	if svc.publisher == nil {
		t.Fatal("expected publisher to be set")
	}
}

func TestSchedulerService_ClaimNextJobs(t *testing.T) {
	expectedIDs := []uuid.UUID{uuid.New(), uuid.New()}

	t.Run("success", func(t *testing.T) {
		repo := &mockSchedulerRepo{
			claimNextJobsFn: func(ctx context.Context, jobBatchSize int) ([]uuid.UUID, error) {
				if jobBatchSize != 2 {
					t.Fatalf("expected batch size 2, got %d", jobBatchSize)
				}
				return expectedIDs, nil
			},
		}
		svc := NewSchedulerService(schedulerTestLogger(), repo, &mockSchedulerPublisher{})

		got := svc.ClaimNextJobs(context.Background(), 2)
		if len(got) != len(expectedIDs) {
			t.Fatalf("expected %d ids, got %d", len(expectedIDs), len(got))
		}
		for i := range expectedIDs {
			if got[i] != expectedIDs[i] {
				t.Fatalf("expected id %s, got %s", expectedIDs[i], got[i])
			}
		}
	})

	t.Run("repo error", func(t *testing.T) {
		repoErr := errors.New("claim failed")
		repo := &mockSchedulerRepo{
			claimNextJobsFn: func(ctx context.Context, jobBatchSize int) ([]uuid.UUID, error) {
				return nil, repoErr
			},
		}
		svc := NewSchedulerService(schedulerTestLogger(), repo, &mockSchedulerPublisher{})

		got := svc.ClaimNextJobs(context.Background(), 2)
		if got != nil {
			t.Fatalf("expected nil ids, got %v", got)
		}
	})
}

func TestSchedulerService_PublishJobIdToChannel(t *testing.T) {
	jobID := uuid.New()

	t.Run("success", func(t *testing.T) {
		publisher := &mockSchedulerPublisher{
			publishFn: func(ctx context.Context, subject string, payload []byte, headers map[string]string) error {
				if subject != data.JobsSubjectFetcher {
					t.Fatalf("expected subject %s, got %s", data.JobsSubjectFetcher, subject)
				}
				if payload != nil {
					t.Fatalf("expected nil payload, got %s", string(payload))
				}
				if headers["job-id"] != jobID.String() {
					t.Fatalf("expected job header %s, got %s", jobID, headers["job-id"])
				}
				return nil
			},
		}
		svc := NewSchedulerService(schedulerTestLogger(), &mockSchedulerRepo{}, publisher)

		svc.PublishJobIdToChannel(context.Background(), jobID)
	})

	t.Run("publisher error is swallowed", func(t *testing.T) {
		publisher := &mockSchedulerPublisher{
			publishFn: func(ctx context.Context, subject string, payload []byte, headers map[string]string) error {
				return errors.New("publish failed")
			},
		}
		svc := NewSchedulerService(schedulerTestLogger(), &mockSchedulerRepo{}, publisher)

		svc.PublishJobIdToChannel(context.Background(), jobID)
	})
}

func TestSchedulerService_MonitorHungedTasks(t *testing.T) {
	called := make(chan int, 1)
	repo := &mockSchedulerRepo{
		resetHungMessageFn: func(ctx context.Context, scheduleJobTimeout int, procJobTimeout int) error {
			if procJobTimeout != 30 {
				t.Fatalf("expected proc timeout 30, got %d", procJobTimeout)
			}
			called <- scheduleJobTimeout
			return nil
		},
	}
	svc := NewSchedulerService(schedulerTestLogger(), repo, &mockSchedulerPublisher{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go svc.MonitorHungedTasks(ctx, 15, 30, time.Millisecond)

	select {
	case got := <-called:
		if got != 15 {
			t.Fatalf("expected timeout 15, got %d", got)
		}
	case <-time.After(time.Second):
		t.Fatal("expected monitor call")
	}
}

func TestSchedulerService_MonitorDisabledTasks(t *testing.T) {
	called := make(chan struct{}, 1)
	repo := &mockSchedulerRepo{
		switchToDisabledIfNeedFn: func(ctx context.Context) error {
			called <- struct{}{}
			return nil
		},
	}
	svc := NewSchedulerService(schedulerTestLogger(), repo, &mockSchedulerPublisher{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go svc.MonitorDisabledTasks(ctx, time.Millisecond)

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("expected monitor call")
	}
}
