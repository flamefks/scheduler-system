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

type mockDeliverRepo struct {
	getConfigFn    func(ctx context.Context, kind string, jobId uuid.UUID) (*data.IOConfig, error)
	setJobStatusFn func(ctx context.Context, status string, jobId uuid.UUID) error
}

func (m *mockDeliverRepo) GetConfig(ctx context.Context, kind string, jobId uuid.UUID) (*data.IOConfig, error) {
	return m.getConfigFn(ctx, kind, jobId)
}

func (m *mockDeliverRepo) SetJobStatus(ctx context.Context, status string, jobId uuid.UUID) error {
	if m.setJobStatusFn == nil {
		return nil
	}
	return m.setJobStatusFn(ctx, status, jobId)
}

type mockDeliverHTTPClient struct {
	doFn func(ctx context.Context, req *data.Request) (*data.ExternalResponse, error)
}

func (m *mockDeliverHTTPClient) Do(ctx context.Context, req *data.Request) (*data.ExternalResponse, error) {
	return m.doFn(ctx, req)
}

func deliverTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func deliverHeaderWithJobID(jobID uuid.UUID) nats.Header {
	header := nats.Header{}
	header.Set("job-id", jobID.String())
	return header
}

func TestNewDeliverService(t *testing.T) {
	repo := &mockDeliverRepo{}

	svc := NewDeliverService(deliverTestLogger(), repo, nil)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.repo == nil {
		t.Fatal("expected repo to be set")
	}
	if svc.httpClient == nil {
		t.Fatal("expected http client to be set")
	}
}

func TestDeliverService_Handle(t *testing.T) {
	jobID := uuid.New()
	headersJSON := json.RawMessage(`{"X-Delivery":"yes"}`)
	natsPayload := []byte(`{"status_code":200}`)

	t.Run("success", func(t *testing.T) {
		repo := &mockDeliverRepo{
			getConfigFn: func(ctx context.Context, kind string, gotJobID uuid.UUID) (*data.IOConfig, error) {
				if kind != data.DeliverKindName {
					t.Fatalf("expected kind %s, got %s", data.DeliverKindName, kind)
				}
				if gotJobID != jobID {
					t.Fatalf("expected job id %s, got %s", jobID, gotJobID)
				}
				return &data.IOConfig{
					TargetUrl: "https://example.test/deliver",
					Method:    http.MethodPut,
					Headers:   headersJSON,
				}, nil
			},
			setJobStatusFn: func(ctx context.Context, status string, gotJobID uuid.UUID) error {
				if status != "delivering" && status != "idle" {
					t.Fatalf("expected delivering or idle status, got %s", status)
				}
				if gotJobID != jobID {
					t.Fatalf("expected job id %s, got %s", jobID, gotJobID)
				}
				return nil
			},
		}

		svc := NewDeliverService(deliverTestLogger(), repo, nil)
		svc.httpClient = &mockDeliverHTTPClient{
			doFn: func(ctx context.Context, req *data.Request) (*data.ExternalResponse, error) {
				if req.Method != http.MethodPut {
					t.Fatalf("expected method PUT, got %s", req.Method)
				}
				if req.URL != "https://example.test/deliver" {
					t.Fatalf("unexpected url: %s", req.URL)
				}
				if string(req.Body) != string(natsPayload) {
					t.Fatalf("unexpected body: %s", string(req.Body))
				}
				if req.Headers["X-Delivery"] != "yes" {
					t.Fatalf("unexpected headers: %#v", req.Headers)
				}
				return &data.ExternalResponse{StatusCode: http.StatusNoContent}, nil
			},
		}

		needSetDbStatus := true
		err, statusCode := svc.Handle(context.Background(), natsPayload, deliverHeaderWithJobID(jobID), &needSetDbStatus)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if statusCode != http.StatusNoContent {
			t.Fatalf("expected status %d, got %d", http.StatusNoContent, statusCode)
		}
	})

	t.Run("invalid job id header", func(t *testing.T) {
		svc := NewDeliverService(deliverTestLogger(), &mockDeliverRepo{}, nil)

		needSetDbStatus := true
		err, statusCode := svc.Handle(context.Background(), natsPayload, nats.Header{}, &needSetDbStatus)
		if !errors.Is(err, natsqueue.TermError) {
			t.Fatalf("expected term error, got %v", err)
		}
		if statusCode != 0 {
			t.Fatalf("expected status 0, got %d", statusCode)
		}
	})

	t.Run("repo error", func(t *testing.T) {
		repoErr := errors.New("config failed")
		repo := &mockDeliverRepo{
			getConfigFn: func(ctx context.Context, kind string, gotJobID uuid.UUID) (*data.IOConfig, error) {
				return nil, repoErr
			},
		}
		svc := NewDeliverService(deliverTestLogger(), repo, nil)

		needSetDbStatus := true
		err, statusCode := svc.Handle(context.Background(), natsPayload, deliverHeaderWithJobID(jobID), &needSetDbStatus)
		if !errors.Is(err, natsqueue.NakError) {
			t.Fatalf("expected nak error, got %v", err)
		}
		if statusCode != 0 {
			t.Fatalf("expected status 0, got %d", statusCode)
		}
	})

	t.Run("invalid config headers", func(t *testing.T) {
		repo := &mockDeliverRepo{
			getConfigFn: func(ctx context.Context, kind string, gotJobID uuid.UUID) (*data.IOConfig, error) {
				return &data.IOConfig{Headers: json.RawMessage(`{bad}`)}, nil
			},
		}
		svc := NewDeliverService(deliverTestLogger(), repo, nil)

		needSetDbStatus := true
		err, statusCode := svc.Handle(context.Background(), natsPayload, deliverHeaderWithJobID(jobID), &needSetDbStatus)
		if !errors.Is(err, natsqueue.TermError) {
			t.Fatalf("expected term error, got %v", err)
		}
		if statusCode != 0 {
			t.Fatalf("expected status 0, got %d", statusCode)
		}
	})

	t.Run("nil config headers are treated as empty", func(t *testing.T) {
		repo := &mockDeliverRepo{
			getConfigFn: func(ctx context.Context, kind string, gotJobID uuid.UUID) (*data.IOConfig, error) {
				return &data.IOConfig{
					TargetUrl: "https://example.test/deliver",
					Method:    http.MethodPost,
					Headers:   nil,
				}, nil
			},
			setJobStatusFn: func(ctx context.Context, status string, gotJobID uuid.UUID) error {
				return nil
			},
		}
		svc := NewDeliverService(deliverTestLogger(), repo, nil)
		svc.httpClient = &mockDeliverHTTPClient{
			doFn: func(ctx context.Context, req *data.Request) (*data.ExternalResponse, error) {
				if len(req.Headers) != 0 {
					t.Fatalf("expected empty headers, got %#v", req.Headers)
				}
				return &data.ExternalResponse{StatusCode: http.StatusOK}, nil
			},
		}

		needSetDbStatus := true
		err, statusCode := svc.Handle(context.Background(), natsPayload, deliverHeaderWithJobID(jobID), &needSetDbStatus)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if statusCode != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, statusCode)
		}
	})

	t.Run("http error returns response status", func(t *testing.T) {
		repo := &mockDeliverRepo{
			getConfigFn: func(ctx context.Context, kind string, gotJobID uuid.UUID) (*data.IOConfig, error) {
				return &data.IOConfig{
					TargetUrl: "https://example.test/deliver",
					Method:    http.MethodPost,
					Headers:   json.RawMessage(`{}`),
				}, nil
			},
		}
		svc := NewDeliverService(deliverTestLogger(), repo, nil)
		httpErr := errors.New("temporary failure")
		svc.httpClient = &mockDeliverHTTPClient{
			doFn: func(ctx context.Context, req *data.Request) (*data.ExternalResponse, error) {
				return &data.ExternalResponse{StatusCode: http.StatusGatewayTimeout}, httpErr
			},
		}

		needSetDbStatus := true
		err, statusCode := svc.Handle(context.Background(), natsPayload, deliverHeaderWithJobID(jobID), &needSetDbStatus)
		if !errors.Is(err, natsqueue.NakError) {
			t.Fatalf("expected nak error, got %v", err)
		}
		if statusCode != http.StatusGatewayTimeout {
			t.Fatalf("expected status %d, got %d", http.StatusGatewayTimeout, statusCode)
		}
	})

	t.Run("set idle error", func(t *testing.T) {
		statusErr := errors.New("set status failed")
		repo := &mockDeliverRepo{
			getConfigFn: func(ctx context.Context, kind string, gotJobID uuid.UUID) (*data.IOConfig, error) {
				return &data.IOConfig{
					TargetUrl: "https://example.test/deliver",
					Method:    http.MethodPost,
					Headers:   json.RawMessage(`{}`),
				}, nil
			},
			setJobStatusFn: func(ctx context.Context, status string, gotJobID uuid.UUID) error {
				return statusErr
			},
		}
		svc := NewDeliverService(deliverTestLogger(), repo, nil)
		svc.httpClient = &mockDeliverHTTPClient{
			doFn: func(ctx context.Context, req *data.Request) (*data.ExternalResponse, error) {
				return &data.ExternalResponse{StatusCode: http.StatusOK}, nil
			},
		}

		needSetDbStatus := true
		err, statusCode := svc.Handle(context.Background(), natsPayload, deliverHeaderWithJobID(jobID), &needSetDbStatus)
		if !errors.Is(err, natsqueue.NakError) {
			t.Fatalf("expected nak error, got %v", err)
		}
		if statusCode != 0 {
			t.Fatalf("expected status 0, got %d", statusCode)
		}
	})
}

func TestDeliverService_HandleError(t *testing.T) {
	jobID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &mockDeliverRepo{
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
		svc := NewDeliverService(deliverTestLogger(), repo, nil)

		svc.HandleError(context.Background(), nil, deliverHeaderWithJobID(jobID))
	})

	t.Run("invalid job id header", func(t *testing.T) {
		svc := NewDeliverService(deliverTestLogger(), &mockDeliverRepo{}, nil)

		svc.HandleError(context.Background(), nil, nats.Header{})
	})

	t.Run("repo error", func(t *testing.T) {
		called := false
		repo := &mockDeliverRepo{
			setJobStatusFn: func(ctx context.Context, status string, gotJobID uuid.UUID) error {
				called = true
				return errors.New("set status failed")
			},
		}
		svc := NewDeliverService(deliverTestLogger(), repo, nil)

		svc.HandleError(context.Background(), nil, deliverHeaderWithJobID(jobID))
		if !called {
			t.Fatal("expected repo set status to be called")
		}
	})
}
