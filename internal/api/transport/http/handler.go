package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/flamefks/scheduler-system/internal/api/apperrors"
	"github.com/flamefks/scheduler-system/internal/api/domain"
	apimetrics "github.com/flamefks/scheduler-system/internal/api/metrics"
	"github.com/flamefks/scheduler-system/internal/api/service"
	"github.com/flamefks/scheduler-system/internal/shared/data"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ApiHandler struct {
	apiService *service.ApiService
	metrics    *apimetrics.ApiMetrics
}

func NewApiHandler(service *service.ApiService, metrics *apimetrics.ApiMetrics) *ApiHandler {
	return &ApiHandler{
		apiService: service,
		metrics:    metrics,
	}
}

func (h *ApiHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()

	var req CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.apiService.Logger.Warn(
			"create_job",
			slog.Any("state", "decode_request"),
			slog.Any("error", err),
		)
		h.metrics.RecordHTTP(r.Context(), apimetrics.OperationCreateJob, http.StatusBadRequest, time.Since(startedAt))
		writeError(w, apperrors.ErrInvalidJSON)
		return
	}

	if err := ValidateCreateJobRequest(&req); err != nil {
		h.apiService.Logger.Warn(
			"create_job",
			slog.Any("state", "validate_request"),
			slog.Any("error", err),
		)
		h.metrics.RecordHTTP(r.Context(), apimetrics.OperationCreateJob, http.StatusBadRequest, time.Since(startedAt))
		writeError(w, apperrors.ErrInvalidRequest)
		return
	}

	jobDomain := &data.Job{
		Name: req.Name,
		Schedule: data.Schedule{
			RepeatIntervalSec: req.Schedule.RepeatIntervalSec,
			TargetRuns:        req.Schedule.TargetRuns,
			DoneRuns:          0,
			NextRunAt:         req.Schedule.NextRunAt,
			LastScheduledAt:   nil,
			LastRunTakenAt:    nil,
		},
		FetcherConfig: data.IOConfig{
			TargetUrl:  req.FetcherConfig.TargetURL,
			Method:     req.FetcherConfig.Method,
			Payload:    req.FetcherConfig.Payload,
			Headers:    req.FetcherConfig.Headers,
			JsonSchema: req.FetcherConfig.JsonSchema,
		},
		DeliverConfig: data.IOConfig{
			TargetUrl:  req.DeliverConfig.TargetURL,
			Method:     req.DeliverConfig.Method,
			Payload:    req.DeliverConfig.Payload,
			Headers:    req.DeliverConfig.Headers,
			JsonSchema: req.DeliverConfig.JsonSchema,
		},
	}

	jobID, err := h.apiService.CreateJob(r.Context(), jobDomain)
	if err != nil {
		h.metrics.RecordHTTP(r.Context(), apimetrics.OperationCreateJob, statusCodeFromError(err), time.Since(startedAt))
		writeError(w, err)
		return
	}

	h.metrics.RecordHTTP(r.Context(), apimetrics.OperationCreateJob, http.StatusCreated, time.Since(startedAt))
	writeJSON(w, http.StatusCreated, map[string]any{
		"status": "success",
		"data": map[string]string{
			"id": jobID.String(),
		},
	})
}

func (h *ApiHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()

	id, err := CheckUUID(chi.URLParam(r, "id"), w)
	if err != nil {
		h.apiService.Logger.Warn(
			"get_job",
			slog.Any("state", "parse_uuid"),
			slog.Any("error", err),
		)
		h.metrics.RecordHTTP(r.Context(), apimetrics.OperationGetJob, http.StatusUnprocessableEntity, time.Since(startedAt))
		writeError(w, err)
		return
	}

	job, err := h.apiService.GetJobByID(r.Context(), id)
	if err != nil {
		h.metrics.RecordHTTP(r.Context(), apimetrics.OperationGetJob, statusCodeFromError(err), time.Since(startedAt))
		writeError(w, err)
		return
	}

	h.metrics.RecordHTTP(r.Context(), apimetrics.OperationGetJob, http.StatusOK, time.Since(startedAt))
	writeJSON(w, http.StatusOK, GetJobResponse{
		Status: "success",
		Data:   job,
	})
}

func (h *ApiHandler) UpdateJob(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()

	id, err := CheckUUID(chi.URLParam(r, "id"), w)
	if err != nil {
		h.apiService.Logger.Warn(
			"patch_job",
			slog.Any("state", "parse_uuid"),
			slog.Any("error", err),
		)
		h.metrics.RecordHTTP(r.Context(), apimetrics.OperationPatchJob, http.StatusUnprocessableEntity, time.Since(startedAt))
		writeError(w, err)
		return
	}

	var req PatchJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.apiService.Logger.Warn(
			"patch_job",
			slog.Any("state", "decode_request"),
			slog.Any("error", err),
		)
		h.metrics.RecordHTTP(r.Context(), apimetrics.OperationPatchJob, http.StatusBadRequest, time.Since(startedAt))
		writeError(w, apperrors.ErrInvalidJSON)
		return
	}

	if err := ValidatePatchJobRequest(&req); err != nil {
		h.apiService.Logger.Warn(
			"patch_job",
			slog.Any("state", "validate_request"),
			slog.Any("error", err),
		)
		h.metrics.RecordHTTP(r.Context(), apimetrics.OperationPatchJob, http.StatusBadRequest, time.Since(startedAt))
		writeError(w, apperrors.ErrInvalidRequest)
		return
	}

	patch := &domain.PatchJobModel{
		Name: req.Name,
	}

	if req.Schedule != nil {
		patch.Schedule = &domain.PatchScheduleModel{
			RepeatIntervalSec: req.Schedule.RepeatIntervalSec,
			TargetRuns:        req.Schedule.TargetRuns,
			NextRunAt:         req.Schedule.NextRunAt,
		}
	}

	if req.FetcherConfig != nil {
		patch.FetcherConfig = &domain.PatchIOConfig{
			TargetUrl: req.FetcherConfig.TargetURL,
			Method:    req.FetcherConfig.Method,
			Payload: domain.PatchJSONField{
				Set:   req.FetcherConfig.Payload.Set,
				Value: req.FetcherConfig.Payload.Value,
			},
			Headers: domain.PatchJSONField{
				Set:   req.FetcherConfig.Headers.Set,
				Value: req.FetcherConfig.Headers.Value,
			},
			JsonSchema: domain.PatchJSONField{
				Set:   req.FetcherConfig.JsonSchema.Set,
				Value: req.FetcherConfig.JsonSchema.Value,
			},
		}
	}

	if req.DeliverConfig != nil {
		patch.DeliverConfig = &domain.PatchIOConfig{
			TargetUrl: req.DeliverConfig.TargetURL,
			Method:    req.DeliverConfig.Method,
			Payload: domain.PatchJSONField{
				Set:   req.DeliverConfig.Payload.Set,
				Value: req.DeliverConfig.Payload.Value,
			},
			Headers: domain.PatchJSONField{
				Set:   req.DeliverConfig.Headers.Set,
				Value: req.DeliverConfig.Headers.Value,
			},
			JsonSchema: domain.PatchJSONField{
				Set:   req.DeliverConfig.JsonSchema.Set,
				Value: req.DeliverConfig.JsonSchema.Value,
			},
		}
	}

	if err := h.apiService.PatchJob(r.Context(), patch, id); err != nil {
		h.metrics.RecordHTTP(r.Context(), apimetrics.OperationPatchJob, statusCodeFromError(err), time.Since(startedAt))
		writeError(w, err)
		return
	}

	h.metrics.RecordHTTP(r.Context(), apimetrics.OperationPatchJob, http.StatusOK, time.Since(startedAt))
	writeJSON(w, http.StatusOK, data.BasicResonse{
		Status:  "success",
		Message: "Job successfully updated",
	})
}

func (h *ApiHandler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()

	id, err := CheckUUID(chi.URLParam(r, "id"), w)
	if err != nil {
		h.apiService.Logger.Warn(
			"delete_job",
			slog.Any("state", "parse_uuid"),
			slog.Any("error", err),
		)
		h.metrics.RecordHTTP(r.Context(), apimetrics.OperationDeleteJob, http.StatusUnprocessableEntity, time.Since(startedAt))
		writeError(w, err)
		return
	}

	if err := h.apiService.DeleteJob(r.Context(), id); err != nil {
		h.metrics.RecordHTTP(r.Context(), apimetrics.OperationDeleteJob, statusCodeFromError(err), time.Since(startedAt))
		writeError(w, err)
		return
	}

	h.metrics.RecordHTTP(r.Context(), apimetrics.OperationDeleteJob, http.StatusNoContent, time.Since(startedAt))
	w.WriteHeader(http.StatusNoContent)
}

func (h *ApiHandler) ActivateJob(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()

	id, err := CheckUUID(chi.URLParam(r, "id"), w)
	if err != nil {
		h.apiService.Logger.Warn(
			"activate_job",
			slog.Any("state", "parse_uuid"),
			slog.Any("error", err),
		)
		h.metrics.RecordHTTP(r.Context(), apimetrics.OperationActivateJob, http.StatusUnprocessableEntity, time.Since(startedAt))
		writeError(w, err)
		return
	}

	if err := h.apiService.ActivateJob(r.Context(), id); err != nil {
		h.metrics.RecordHTTP(r.Context(), apimetrics.OperationActivateJob, statusCodeFromError(err), time.Since(startedAt))
		writeError(w, err)
		return
	}

	h.metrics.RecordHTTP(r.Context(), apimetrics.OperationActivateJob, http.StatusOK, time.Since(startedAt))
	writeJSON(w, http.StatusOK, data.BasicResonse{
		Status:  "success",
		Message: "Job successfully activated",
	})
}

func (h *ApiHandler) DeactivateJob(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()

	id, err := CheckUUID(chi.URLParam(r, "id"), w)
	if err != nil {
		h.apiService.Logger.Warn(
			"deactivate_job",
			slog.Any("state", "parse_uuid"),
			slog.Any("error", err),
		)
		h.metrics.RecordHTTP(r.Context(), apimetrics.OperationDeactivateJob, http.StatusUnprocessableEntity, time.Since(startedAt))
		writeError(w, err)
		return
	}

	if err := h.apiService.DeactivateJob(r.Context(), id); err != nil {
		h.metrics.RecordHTTP(r.Context(), apimetrics.OperationDeactivateJob, statusCodeFromError(err), time.Since(startedAt))
		writeError(w, err)
		return
	}

	h.metrics.RecordHTTP(r.Context(), apimetrics.OperationDeactivateJob, http.StatusOK, time.Since(startedAt))
	writeJSON(w, http.StatusOK, data.BasicResonse{
		Status:  "success",
		Message: "Job successfully deactivated",
	})
}

// helper
func CheckUUID(strId string, w http.ResponseWriter) (uuid.UUID, error) {
	id, err := uuid.Parse(strId)
	if err != nil {
		return uuid.Nil, apperrors.ErrInvalidUUID
	}
	return id, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func statusCodeFromError(err error) int {
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, apperrors.ErrInvalidStatus):
		return http.StatusBadRequest
	case errors.Is(err, apperrors.ErrStatusConflict):
		return http.StatusConflict
	case errors.Is(err, apperrors.ErrInvalidJSON):
		return http.StatusBadRequest
	case errors.Is(err, apperrors.ErrInvalidRequest):
		return http.StatusBadRequest
	case errors.Is(err, apperrors.ErrInvalidUUID):
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		writeJSON(w, http.StatusNotFound, data.BasicResonse{
			Status:  "error",
			Message: "resource not found",
		})
	case errors.Is(err, apperrors.ErrInvalidStatus):
		writeJSON(w, http.StatusBadRequest, data.BasicResonse{
			Status:  "error",
			Message: "invalid status",
		})
	case errors.Is(err, apperrors.ErrStatusConflict):
		writeJSON(w, http.StatusConflict, data.BasicResonse{
			Status:  "error",
			Message: "status transition is not allowed",
		})
	case errors.Is(err, apperrors.ErrInvalidJSON):
		writeJSON(w, http.StatusBadRequest, data.BasicResonse{
			Status:  "error",
			Message: "error decoding request body",
		})
	case errors.Is(err, apperrors.ErrInvalidRequest):
		writeJSON(w, http.StatusBadRequest, data.BasicResonse{
			Status:  "error",
			Message: "invalid request body",
		})
	case errors.Is(err, apperrors.ErrInvalidUUID):
		writeJSON(w, http.StatusUnprocessableEntity, data.BasicResonse{
			Status:  "error",
			Message: "incorrect id type",
		})
	default:
		writeJSON(w, http.StatusInternalServerError, data.BasicResonse{
			Status:  "error",
			Message: "internal server error",
		})
	}
}
