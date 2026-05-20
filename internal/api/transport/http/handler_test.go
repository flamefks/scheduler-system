package http

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/flamefks/scheduler-system/internal/api/apperrors"
	"github.com/flamefks/scheduler-system/internal/api/domain"
	apiservice "github.com/flamefks/scheduler-system/internal/api/service"
	"github.com/flamefks/scheduler-system/internal/shared/data"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type mockRepo struct {
	createJobFn     func(ctx context.Context, job *data.Job) (uuid.UUID, error)
	deleteJobFn     func(ctx context.Context, id uuid.UUID) error
	getJobByIDFn    func(ctx context.Context, id uuid.UUID) (*data.Job, error)
	patchJobFn      func(ctx context.Context, patch *domain.PatchJobModel, id uuid.UUID) error
	activateJobFn   func(ctx context.Context, id uuid.UUID) error
	deactivateJobFn func(ctx context.Context, id uuid.UUID) error
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

func (m *mockRepo) ActivateJob(ctx context.Context, id uuid.UUID) error {
	return m.activateJobFn(ctx, id)
}

func (m *mockRepo) DeactivateJob(ctx context.Context, id uuid.UUID) error {
	return m.deactivateJobFn(ctx, id)
}

func handlerLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newHandler(repo *mockRepo) *ApiHandler {
	svc := apiservice.NewApiService(handlerLogger(), repo, nil)
	return NewApiHandler(svc, nil)
}

func withURLParam(req *stdhttp.Request, key, value string) *stdhttp.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func TestCheckUUID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		id := uuid.New()
		w := httptest.NewRecorder()

		got, err := CheckUUID(id.String(), w)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != id {
			t.Fatalf("expected %s, got %s", id, got)
		}
	})

	t.Run("invalid uuid", func(t *testing.T) {
		w := httptest.NewRecorder()

		_, err := CheckUUID("not-a-uuid", w)
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, apperrors.ErrInvalidUUID) {
			t.Fatalf("expected ErrInvalidUUID, got %v", err)
		}
	})
}

func TestApiHandler_CreateJob(t *testing.T) {
	nextRunAt := time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC)
	expectedID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{
			createJobFn: func(ctx context.Context, job *data.Job) (uuid.UUID, error) {
				if job.Name != "job-1" {
					t.Fatalf("unexpected name: %s", job.Name)
				}
				if job.Schedule.RepeatIntervalSec != 60 {
					t.Fatalf("unexpected repeat interval: %d", job.Schedule.RepeatIntervalSec)
				}
				if job.Schedule.TargetRuns != 3 {
					t.Fatalf("unexpected target runs: %d", job.Schedule.TargetRuns)
				}
				if !job.Schedule.NextRunAt.Equal(nextRunAt) {
					t.Fatalf("unexpected next_run_at: %v", job.Schedule.NextRunAt)
				}
				return expectedID, nil
			},
		}

		h := newHandler(repo)

		body := []byte(`{
			"name":"job-1",
			"schedule":{
				"repeat_interval_sec":60,
				"target_runs":3,
				"next_run_at":"2026-04-22T10:00:00Z"
			},
			"fetcher_config":{
				"target_url":"https://fetcher.example/jobs",
				"method":"POST",
				"payload":{},
				"headers":{"Authorization":"token"}
			},
			"deliver_config":{
				"target_url":"https://deliver.example/jobs",
				"method":"POST",
				"payload":{},
				"headers":{"Authorization":"token"}
			}
		}`)

		req := httptest.NewRequest(stdhttp.MethodPost, "/jobs", bytes.NewReader(body))
		w := httptest.NewRecorder()

		h.CreateJob(w, req)

		if w.Code != stdhttp.StatusCreated {
			t.Fatalf("expected 201, got %d, body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("bad json", func(t *testing.T) {
		h := newHandler(&mockRepo{})
		req := httptest.NewRequest(stdhttp.MethodPost, "/jobs", bytes.NewReader([]byte(`{bad json}`)))
		w := httptest.NewRecorder()

		h.CreateJob(w, req)

		if w.Code != stdhttp.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("service error", func(t *testing.T) {
		repo := &mockRepo{
			createJobFn: func(ctx context.Context, job *data.Job) (uuid.UUID, error) {
				return uuid.Nil, errors.New("insert failed")
			},
		}

		h := newHandler(repo)

		body := []byte(`{
			"name":"job-1",
			"schedule":{
				"repeat_interval_sec":60,
				"target_runs":3,
				"next_run_at":"2026-04-22T10:00:00Z"
			},
			"fetcher_config":{
				"target_url":"https://fetcher.example/jobs",
				"method":"POST",
				"payload":{},
				"headers":{"Authorization":"token"}
			},
			"deliver_config":{
				"target_url":"https://deliver.example/jobs",
				"method":"POST",
				"payload":{},
				"headers":{"Authorization":"token"}
			}
		}`)

		req := httptest.NewRequest(stdhttp.MethodPost, "/jobs", bytes.NewReader(body))
		w := httptest.NewRecorder()

		h.CreateJob(w, req)

		if w.Code != stdhttp.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestApiHandler_GetJob(t *testing.T) {
	jobID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{
			getJobByIDFn: func(ctx context.Context, id uuid.UUID) (*data.Job, error) {
				return &data.Job{
					ID:   id,
					Name: "job-1",
				}, nil
			},
		}

		h := newHandler(repo)
		req := httptest.NewRequest(stdhttp.MethodGet, "/jobs/"+jobID.String(), nil)
		req = withURLParam(req, "id", jobID.String())
		w := httptest.NewRecorder()

		h.GetJob(w, req)

		if w.Code != stdhttp.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("bad uuid", func(t *testing.T) {
		h := newHandler(&mockRepo{})
		req := httptest.NewRequest(stdhttp.MethodGet, "/jobs/bad", nil)
		req = withURLParam(req, "id", "bad-uuid")
		w := httptest.NewRecorder()

		h.GetJob(w, req)

		if w.Code != stdhttp.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockRepo{
			getJobByIDFn: func(ctx context.Context, id uuid.UUID) (*data.Job, error) {
				return nil, apperrors.ErrNotFound
			},
		}

		h := newHandler(repo)
		req := httptest.NewRequest(stdhttp.MethodGet, "/jobs/"+jobID.String(), nil)
		req = withURLParam(req, "id", jobID.String())
		w := httptest.NewRecorder()

		h.GetJob(w, req)

		if w.Code != stdhttp.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("service error", func(t *testing.T) {
		repo := &mockRepo{
			getJobByIDFn: func(ctx context.Context, id uuid.UUID) (*data.Job, error) {
				return nil, errors.New("db failed")
			},
		}

		h := newHandler(repo)
		req := httptest.NewRequest(stdhttp.MethodGet, "/jobs/"+jobID.String(), nil)
		req = withURLParam(req, "id", jobID.String())
		w := httptest.NewRecorder()

		h.GetJob(w, req)

		if w.Code != stdhttp.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestApiHandler_UpdateJob(t *testing.T) {
	jobID := uuid.New()
	nextRunAt := time.Date(2026, 4, 22, 11, 0, 0, 0, time.UTC)

	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{
			patchJobFn: func(ctx context.Context, patch *domain.PatchJobModel, id uuid.UUID) error {
				if id != jobID {
					t.Fatalf("expected %s, got %s", jobID, id)
				}
				if patch.Name == nil || *patch.Name != "updated-job" {
					t.Fatal("unexpected patch name")
				}
				if patch.Schedule == nil {
					t.Fatal("expected schedule patch")
				}
				if patch.Schedule.TargetRuns == nil || *patch.Schedule.TargetRuns != 10 {
					t.Fatal("unexpected target_runs")
				}
				if patch.Schedule.NextRunAt == nil || !patch.Schedule.NextRunAt.Equal(nextRunAt) {
					t.Fatal("unexpected next_run_at")
				}
				return nil
			},
		}

		h := newHandler(repo)

		body := []byte(`{
			"name":"updated-job",
			"schedule":{
				"target_runs":10,
				"next_run_at":"2026-04-22T11:00:00Z"
			}
		}`)

		req := httptest.NewRequest(stdhttp.MethodPatch, "/jobs/"+jobID.String(), bytes.NewReader(body))
		req = withURLParam(req, "id", jobID.String())
		w := httptest.NewRecorder()

		h.UpdateJob(w, req)

		if w.Code != stdhttp.StatusOK {
			t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("explicit null json fields", func(t *testing.T) {
		repo := &mockRepo{
			patchJobFn: func(ctx context.Context, patch *domain.PatchJobModel, id uuid.UUID) error {
				if id != jobID {
					t.Fatalf("expected %s, got %s", jobID, id)
				}
				if patch.FetcherConfig == nil {
					t.Fatal("expected fetcher config patch")
				}
				if !patch.FetcherConfig.Headers.Set {
					t.Fatal("expected headers to be marked as set")
				}
				if patch.FetcherConfig.Headers.Value != nil {
					t.Fatalf("expected nil headers value, got %s", string(patch.FetcherConfig.Headers.Value))
				}
				if !patch.FetcherConfig.JsonSchema.Set {
					t.Fatal("expected json_schema to be marked as set")
				}
				if patch.FetcherConfig.JsonSchema.Value != nil {
					t.Fatalf("expected nil json_schema value, got %s", string(patch.FetcherConfig.JsonSchema.Value))
				}
				if patch.FetcherConfig.Payload.Set {
					t.Fatal("did not expect absent payload to be marked as set")
				}
				return nil
			},
		}

		h := newHandler(repo)

		body := []byte(`{
			"fetcher_config":{
				"headers":null,
				"json_schema":null
			}
		}`)

		req := httptest.NewRequest(stdhttp.MethodPatch, "/jobs/"+jobID.String(), bytes.NewReader(body))
		req = withURLParam(req, "id", jobID.String())
		w := httptest.NewRecorder()

		h.UpdateJob(w, req)

		if w.Code != stdhttp.StatusOK {
			t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("bad uuid", func(t *testing.T) {
		h := newHandler(&mockRepo{})

		req := httptest.NewRequest(stdhttp.MethodPatch, "/jobs/bad", bytes.NewReader([]byte(`{}`)))
		req = withURLParam(req, "id", "bad-uuid")
		w := httptest.NewRecorder()

		h.UpdateJob(w, req)

		if w.Code != stdhttp.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d", w.Code)
		}
	})

	t.Run("bad json", func(t *testing.T) {
		h := newHandler(&mockRepo{})

		req := httptest.NewRequest(stdhttp.MethodPatch, "/jobs/"+jobID.String(), bytes.NewReader([]byte(`{bad}`)))
		req = withURLParam(req, "id", jobID.String())
		w := httptest.NewRecorder()

		h.UpdateJob(w, req)

		if w.Code != stdhttp.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("conflict error", func(t *testing.T) {
		repo := &mockRepo{
			patchJobFn: func(ctx context.Context, patch *domain.PatchJobModel, id uuid.UUID) error {
				return apperrors.ErrStatusConflict
			},
		}

		h := newHandler(repo)

		req := httptest.NewRequest(stdhttp.MethodPatch, "/jobs/"+jobID.String(), bytes.NewReader([]byte(`{"name":"x"}`)))
		req = withURLParam(req, "id", jobID.String())
		w := httptest.NewRecorder()

		h.UpdateJob(w, req)

		if w.Code != stdhttp.StatusConflict {
			t.Fatalf("expected 409, got %d", w.Code)
		}
	})

	t.Run("service error", func(t *testing.T) {
		repo := &mockRepo{
			patchJobFn: func(ctx context.Context, patch *domain.PatchJobModel, id uuid.UUID) error {
				return errors.New("patch failed")
			},
		}

		h := newHandler(repo)

		req := httptest.NewRequest(stdhttp.MethodPatch, "/jobs/"+jobID.String(), bytes.NewReader([]byte(`{"name":"x"}`)))
		req = withURLParam(req, "id", jobID.String())
		w := httptest.NewRecorder()

		h.UpdateJob(w, req)

		if w.Code != stdhttp.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestApiHandler_DeleteJob(t *testing.T) {
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

		h := newHandler(repo)
		req := httptest.NewRequest(stdhttp.MethodDelete, "/jobs/"+jobID.String(), nil)
		req = withURLParam(req, "id", jobID.String())
		w := httptest.NewRecorder()

		h.DeleteJob(w, req)

		if w.Code != stdhttp.StatusNoContent {
			t.Fatalf("expected 204, got %d", w.Code)
		}
	})

	t.Run("service error", func(t *testing.T) {
		repo := &mockRepo{
			deleteJobFn: func(ctx context.Context, id uuid.UUID) error {
				return errors.New("delete failed")
			},
		}

		h := newHandler(repo)
		req := httptest.NewRequest(stdhttp.MethodDelete, "/jobs/"+jobID.String(), nil)
		req = withURLParam(req, "id", jobID.String())
		w := httptest.NewRecorder()

		h.DeleteJob(w, req)

		if w.Code != stdhttp.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestApiHandler_ActivateJob(t *testing.T) {
	jobID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{
			activateJobFn: func(ctx context.Context, id uuid.UUID) error {
				if id != jobID {
					t.Fatalf("expected %s, got %s", jobID, id)
				}
				return nil
			},
		}

		h := newHandler(repo)

		req := httptest.NewRequest(stdhttp.MethodPost, "/jobs/"+jobID.String()+"/activate", nil)
		req = withURLParam(req, "id", jobID.String())
		w := httptest.NewRecorder()

		h.ActivateJob(w, req)

		if w.Code != stdhttp.StatusOK {
			t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("bad uuid", func(t *testing.T) {
		h := newHandler(&mockRepo{})

		req := httptest.NewRequest(stdhttp.MethodPost, "/jobs/bad/activate", nil)
		req = withURLParam(req, "id", "bad-uuid")
		w := httptest.NewRecorder()

		h.ActivateJob(w, req)

		if w.Code != stdhttp.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d", w.Code)
		}
	})

	t.Run("not found error", func(t *testing.T) {
		repo := &mockRepo{
			activateJobFn: func(ctx context.Context, id uuid.UUID) error {
				return apperrors.ErrNotFound
			},
		}

		h := newHandler(repo)

		req := httptest.NewRequest(stdhttp.MethodPost, "/jobs/"+jobID.String()+"/activate", nil)
		req = withURLParam(req, "id", jobID.String())
		w := httptest.NewRecorder()

		h.ActivateJob(w, req)

		if w.Code != stdhttp.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("conflict error", func(t *testing.T) {
		repo := &mockRepo{
			activateJobFn: func(ctx context.Context, id uuid.UUID) error {
				return apperrors.ErrStatusConflict
			},
		}

		h := newHandler(repo)

		req := httptest.NewRequest(stdhttp.MethodPost, "/jobs/"+jobID.String()+"/activate", nil)
		req = withURLParam(req, "id", jobID.String())
		w := httptest.NewRecorder()

		h.ActivateJob(w, req)

		if w.Code != stdhttp.StatusConflict {
			t.Fatalf("expected 409, got %d", w.Code)
		}
	})
}

func TestApiHandler_DeactivateJob(t *testing.T) {
	jobID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{
			deactivateJobFn: func(ctx context.Context, id uuid.UUID) error {
				if id != jobID {
					t.Fatalf("expected %s, got %s", jobID, id)
				}
				return nil
			},
		}

		h := newHandler(repo)

		req := httptest.NewRequest(stdhttp.MethodPost, "/jobs/"+jobID.String()+"/deactivate", nil)
		req = withURLParam(req, "id", jobID.String())
		w := httptest.NewRecorder()

		h.DeactivateJob(w, req)

		if w.Code != stdhttp.StatusOK {
			t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("bad uuid", func(t *testing.T) {
		h := newHandler(&mockRepo{})

		req := httptest.NewRequest(stdhttp.MethodPost, "/jobs/bad/deactivate", nil)
		req = withURLParam(req, "id", "bad-uuid")
		w := httptest.NewRecorder()

		h.DeactivateJob(w, req)

		if w.Code != stdhttp.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d", w.Code)
		}
	})

	t.Run("conflict error", func(t *testing.T) {
		repo := &mockRepo{
			deactivateJobFn: func(ctx context.Context, id uuid.UUID) error {
				return apperrors.ErrStatusConflict
			},
		}

		h := newHandler(repo)

		req := httptest.NewRequest(stdhttp.MethodPost, "/jobs/"+jobID.String()+"/deactivate", nil)
		req = withURLParam(req, "id", jobID.String())
		w := httptest.NewRecorder()

		h.DeactivateJob(w, req)

		if w.Code != stdhttp.StatusConflict {
			t.Fatalf("expected 409, got %d", w.Code)
		}
	})
}
