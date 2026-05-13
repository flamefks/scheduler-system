package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/flamefks/scheduler-system/internal/shared/data"
	natsqueue "github.com/flamefks/scheduler-system/internal/shared/queue/nats"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

type mockFetcherRepo struct {
	getConfigFn    func(ctx context.Context, kind string, jobId uuid.UUID) (*data.IOConfig, error)
	setJobStatusFn func(ctx context.Context, status string, jobId uuid.UUID) error
}

func (m *mockFetcherRepo) GetConfig(ctx context.Context, kind string, jobId uuid.UUID) (*data.IOConfig, error) {
	return m.getConfigFn(ctx, kind, jobId)
}

func (m *mockFetcherRepo) SetJobStatus(ctx context.Context, status string, jobId uuid.UUID) error {
	if m.setJobStatusFn == nil {
		return nil
	}
	return m.setJobStatusFn(ctx, status, jobId)
}

type mockFetcherPublisher struct {
	publishFn func(ctx context.Context, subject string, payload []byte, headers map[string]string) error
}

func (m *mockFetcherPublisher) Publish(ctx context.Context, subject string, payload []byte, headers map[string]string) error {
	return m.publishFn(ctx, subject, payload, headers)
}

type mockFetcherHTTPClient struct {
	doFn func(ctx context.Context, req *data.Request) (*data.ExternalResponse, error)
}

func (m *mockFetcherHTTPClient) Do(ctx context.Context, req *data.Request) (*data.ExternalResponse, error) {
	return m.doFn(ctx, req)
}

func fetcherTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func headerWithJobID(jobID uuid.UUID) nats.Header {
	header := nats.Header{}
	header.Set("job-id", jobID.String())
	return header
}

func TestNewFetcherService(t *testing.T) {
	repo := &mockFetcherRepo{}
	publisher := &mockFetcherPublisher{}

	svc := NewFetcherService(fetcherTestLogger(), publisher, repo)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.repo == nil {
		t.Fatal("expected repo to be set")
	}
	if svc.publisher == nil {
		t.Fatal("expected publisher to be set")
	}
	if svc.httpClient == nil {
		t.Fatal("expected http client to be set")
	}
}

func TestFetcherService_Handle(t *testing.T) {
	jobID := uuid.New()
	headersJSON := json.RawMessage(`{"X-Token":"secret"}`)
	payloadJSON := json.RawMessage(`{"hello":"world"}`)

	t.Run("success", func(t *testing.T) {
		repo := &mockFetcherRepo{
			getConfigFn: func(ctx context.Context, kind string, gotJobID uuid.UUID) (*data.IOConfig, error) {
				if kind != data.FetcherKindName {
					t.Fatalf("expected kind %s, got %s", data.FetcherKindName, kind)
				}
				if gotJobID != jobID {
					t.Fatalf("expected job id %s, got %s", jobID, gotJobID)
				}
				return &data.IOConfig{
					TargetUrl: "https://example.test/fetch",
					Method:    http.MethodPost,
					Payload:   payloadJSON,
					Headers:   headersJSON,
				}, nil
			},
		}

		publisher := &mockFetcherPublisher{
			publishFn: func(ctx context.Context, subject string, payload []byte, headers map[string]string) error {
				if subject != data.JobsSubjectDeliver {
					t.Fatalf("expected subject %s, got %s", data.JobsSubjectDeliver, subject)
				}
				if headers["job-id"] != jobID.String() {
					t.Fatalf("expected job header %s, got %s", jobID, headers["job-id"])
				}

				if string(payload) != `{"accepted":true}` {
					t.Fatalf("unexpected payload: %s", string(payload))
				}
				return nil
			},
		}

		svc := NewFetcherService(fetcherTestLogger(), publisher, repo)
		svc.httpClient = &mockFetcherHTTPClient{
			doFn: func(ctx context.Context, req *data.Request) (*data.ExternalResponse, error) {
				if req.Method != http.MethodPost {
					t.Fatalf("expected method POST, got %s", req.Method)
				}
				if req.URL != "https://example.test/fetch" {
					t.Fatalf("unexpected url: %s", req.URL)
				}
				if string(req.Body) != string(payloadJSON) {
					t.Fatalf("unexpected body: %s", string(req.Body))
				}
				if req.Headers["X-Token"] != "secret" {
					t.Fatalf("unexpected headers: %#v", req.Headers)
				}
				return &data.ExternalResponse{
					StatusCode: http.StatusAccepted,
					Body:       json.RawMessage(`{"accepted":true}`),
				}, nil
			},
		}

		needSetDbStatus := true
		err, statusCode := svc.Handle(context.Background(), nil, headerWithJobID(jobID), &needSetDbStatus)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status %d, got %d", http.StatusAccepted, statusCode)
		}
	})

	t.Run("invalid job id header", func(t *testing.T) {
		svc := NewFetcherService(fetcherTestLogger(), &mockFetcherPublisher{}, &mockFetcherRepo{})

		needSetDbStatus := true
		err, statusCode := svc.Handle(context.Background(), nil, nats.Header{}, &needSetDbStatus)
		if !errors.Is(err, natsqueue.TermError) {
			t.Fatalf("expected term error, got %v", err)
		}
		if statusCode != 0 {
			t.Fatalf("expected status 0, got %d", statusCode)
		}
	})

	t.Run("repo error", func(t *testing.T) {
		repoErr := errors.New("config failed")
		repo := &mockFetcherRepo{
			getConfigFn: func(ctx context.Context, kind string, gotJobID uuid.UUID) (*data.IOConfig, error) {
				return nil, repoErr
			},
		}
		svc := NewFetcherService(fetcherTestLogger(), &mockFetcherPublisher{}, repo)

		needSetDbStatus := true
		err, statusCode := svc.Handle(context.Background(), nil, headerWithJobID(jobID), &needSetDbStatus)
		if !errors.Is(err, natsqueue.NakError) {
			t.Fatalf("expected nak error, got %v", err)
		}
		if statusCode != 0 {
			t.Fatalf("expected status 0, got %d", statusCode)
		}
	})

	t.Run("invalid config headers", func(t *testing.T) {
		repo := &mockFetcherRepo{
			getConfigFn: func(ctx context.Context, kind string, gotJobID uuid.UUID) (*data.IOConfig, error) {
				return &data.IOConfig{Headers: json.RawMessage(`{bad}`)}, nil
			},
		}
		svc := NewFetcherService(fetcherTestLogger(), &mockFetcherPublisher{}, repo)

		needSetDbStatus := true
		err, statusCode := svc.Handle(context.Background(), nil, headerWithJobID(jobID), &needSetDbStatus)
		if !errors.Is(err, natsqueue.TermError) {
			t.Fatalf("expected term error, got %v", err)
		}
		if statusCode != 0 {
			t.Fatalf("expected status 0, got %d", statusCode)
		}
	})

	t.Run("nil config headers are treated as empty", func(t *testing.T) {
		repo := &mockFetcherRepo{
			getConfigFn: func(ctx context.Context, kind string, gotJobID uuid.UUID) (*data.IOConfig, error) {
				return &data.IOConfig{
					TargetUrl: "https://example.test/fetch",
					Method:    http.MethodGet,
					Headers:   nil,
				}, nil
			},
		}
		publisher := &mockFetcherPublisher{
			publishFn: func(ctx context.Context, subject string, payload []byte, headers map[string]string) error {
				return nil
			},
		}
		svc := NewFetcherService(fetcherTestLogger(), publisher, repo)
		svc.httpClient = &mockFetcherHTTPClient{
			doFn: func(ctx context.Context, req *data.Request) (*data.ExternalResponse, error) {
				if len(req.Headers) != 0 {
					t.Fatalf("expected empty headers, got %#v", req.Headers)
				}
				return &data.ExternalResponse{StatusCode: http.StatusOK}, nil
			},
		}

		needSetDbStatus := true
		err, statusCode := svc.Handle(context.Background(), nil, headerWithJobID(jobID), &needSetDbStatus)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if statusCode != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, statusCode)
		}
	})

	t.Run("http error returns response status", func(t *testing.T) {
		repo := &mockFetcherRepo{
			getConfigFn: func(ctx context.Context, kind string, gotJobID uuid.UUID) (*data.IOConfig, error) {
				return &data.IOConfig{
					TargetUrl: "https://example.test/fetch",
					Method:    http.MethodGet,
					Headers:   json.RawMessage(`{}`),
				}, nil
			},
		}
		svc := NewFetcherService(fetcherTestLogger(), &mockFetcherPublisher{}, repo)
		httpErr := errors.New("temporary failure")
		svc.httpClient = &mockFetcherHTTPClient{
			doFn: func(ctx context.Context, req *data.Request) (*data.ExternalResponse, error) {
				return &data.ExternalResponse{StatusCode: http.StatusBadGateway}, httpErr
			},
		}

		needSetDbStatus := true
		err, statusCode := svc.Handle(context.Background(), nil, headerWithJobID(jobID), &needSetDbStatus)
		if !errors.Is(err, natsqueue.NakError) {
			t.Fatalf("expected nak error, got %v", err)
		}
		if statusCode != http.StatusBadGateway {
			t.Fatalf("expected status %d, got %d", http.StatusBadGateway, statusCode)
		}
	})

	t.Run("publish error", func(t *testing.T) {
		repo := &mockFetcherRepo{
			getConfigFn: func(ctx context.Context, kind string, gotJobID uuid.UUID) (*data.IOConfig, error) {
				return &data.IOConfig{
					TargetUrl: "https://example.test/fetch",
					Method:    http.MethodGet,
					Headers:   json.RawMessage(`{}`),
				}, nil
			},
		}
		publishErr := errors.New("publish failed")
		publisher := &mockFetcherPublisher{
			publishFn: func(ctx context.Context, subject string, payload []byte, headers map[string]string) error {
				return publishErr
			},
		}
		svc := NewFetcherService(fetcherTestLogger(), publisher, repo)
		svc.httpClient = &mockFetcherHTTPClient{
			doFn: func(ctx context.Context, req *data.Request) (*data.ExternalResponse, error) {
				return &data.ExternalResponse{StatusCode: http.StatusOK}, nil
			},
		}

		needSetDbStatus := true
		err, statusCode := svc.Handle(context.Background(), nil, headerWithJobID(jobID), &needSetDbStatus)
		if !errors.Is(err, natsqueue.NakError) {
			t.Fatalf("expected nak error, got %v", err)
		}
		if statusCode != 0 {
			t.Fatalf("expected status 0, got %d", statusCode)
		}
	})
}

func TestFetcherService_ErrorHandler(t *testing.T) {
	jobID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &mockFetcherRepo{
			setJobStatusFn: func(ctx context.Context, status string, gotJobID uuid.UUID) error {
				if status != "error" {
					t.Fatalf("expected error status, got %s", status)
				}
				if gotJobID != jobID {
					t.Fatalf("expected job id %s, got %s", jobID, gotJobID)
				}
				return nil
			},
		}
		svc := NewFetcherService(fetcherTestLogger(), &mockFetcherPublisher{}, repo)

		svc.ErrorHandler(context.Background(), nil, headerWithJobID(jobID))
	})

	t.Run("invalid job id header", func(t *testing.T) {
		svc := NewFetcherService(fetcherTestLogger(), &mockFetcherPublisher{}, &mockFetcherRepo{})

		svc.ErrorHandler(context.Background(), nil, nats.Header{})
	})

	t.Run("repo error", func(t *testing.T) {
		called := false
		repo := &mockFetcherRepo{
			setJobStatusFn: func(ctx context.Context, status string, gotJobID uuid.UUID) error {
				called = true
				return errors.New("set status failed")
			},
		}
		svc := NewFetcherService(fetcherTestLogger(), &mockFetcherPublisher{}, repo)

		svc.ErrorHandler(context.Background(), nil, headerWithJobID(jobID))
		if !called {
			t.Fatal("expected repo set status to be called")
		}
	})
}
