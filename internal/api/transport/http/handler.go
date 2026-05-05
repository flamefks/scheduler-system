package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/flamefks/scheduler-system/internal/api/domain"
	"github.com/flamefks/scheduler-system/internal/api/service"
	"github.com/flamefks/scheduler-system/internal/shared/data"
	httputils "github.com/flamefks/scheduler-system/internal/shared/utils"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ApiHandler struct {
	apiService *service.ApiService
}

func NewApiHandler(service *service.ApiService) *ApiHandler {
	return &ApiHandler{
		apiService: service,
	}
}

func (h *ApiHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var req CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.apiService.Logger.Warn(
			"error_create_job",
			slog.Any("state", "decode_request"),
			slog.Any("error", err),
		)
		httputils.WriteJSON(w, http.StatusBadRequest, data.BasicResonse{
			Status:  "error",
			Message: "error decoding request body",
		})
		return
	}

	jobDomain := &data.Job{
		Name: req.Name,
		Schedule: data.Schedule{
			RepeatIntervalSec: req.Schedule.RepeatIntervalSec,
			TargetRuns:        req.Schedule.TargetRuns,
			ScheduledRuns:     0,
			NextRunAt:         req.Schedule.NextRunAt,
			LastRunAt:         nil,
		},
		FetcherConfig: data.IOConfig{
			TargetUrl: req.FetcherConfig.TargetURL,
			Method:    req.FetcherConfig.Method,
			Payload:   req.FetcherConfig.Payload,
			Headers:   req.FetcherConfig.Headers,
		},
		DeliverConfig: data.IOConfig{
			TargetUrl: req.DeliverConfig.TargetURL,
			Method:    req.DeliverConfig.Method,
			Payload:   req.DeliverConfig.Payload,
			Headers:   req.DeliverConfig.Headers,
		},
	}

	jobID, err := h.apiService.CreateJob(r.Context(), jobDomain)
	if err != nil {
		httputils.WriteJSON(w, http.StatusInternalServerError, data.BasicResonse{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	httputils.WriteJSON(w, http.StatusCreated, map[string]any{
		"status": "success",
		"data": map[string]string{
			"id": jobID.String(),
		},
	})
}

func (h *ApiHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	id, err := CheckUUID(chi.URLParam(r, "id"), w)
	if err != nil {
		return
	}

	job, err := h.apiService.GetJobByID(r.Context(), id)
	if err != nil {
		httputils.WriteJSON(w, http.StatusInternalServerError, data.BasicResonse{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	httputils.WriteJSON(w, http.StatusOK, GetJobResponse{
		Status: "success",
		Data:   job,
	})
}

func (h *ApiHandler) UpdateJob(w http.ResponseWriter, r *http.Request) {
	id, err := CheckUUID(chi.URLParam(r, "id"), w)
	if err != nil {
		return
	}

	var req PatchJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputils.WriteJSON(w, http.StatusBadRequest, data.BasicResonse{
			Status:  "error",
			Message: "error decoding request body",
		})
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
			Payload: req.FetcherConfig.Payload,
			Headers: req.FetcherConfig.Headers,
		}
	}

	if req.DeliverConfig != nil {
		patch.DeliverConfig = &domain.PatchIOConfig{
			Payload: req.DeliverConfig.Payload,
			Headers: req.DeliverConfig.Headers,
		}
	}

	if err := h.apiService.PatchJob(r.Context(), patch, id); err != nil {
		httputils.WriteJSON(w, http.StatusInternalServerError, data.BasicResonse{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	httputils.WriteJSON(w, http.StatusOK, data.BasicResonse{
		Status:  "success",
		Message: "Job successfully updated",
	})
}

func (h *ApiHandler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	id, err := CheckUUID(chi.URLParam(r, "id"), w)
	if err != nil {
		return
	}

	if err := h.apiService.DeleteJob(r.Context(), id); err != nil {
		httputils.WriteJSON(w, http.StatusInternalServerError, data.BasicResonse{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ApiHandler) UpdateJobStatus(w http.ResponseWriter, r *http.Request) {
	id, err := CheckUUID(chi.URLParam(r, "id"), w)
	if err != nil {
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputils.WriteJSON(w, http.StatusBadRequest, data.BasicResonse{
			Status:  "error",
			Message: "Error body decode",
		})
		return
	}

	if err := h.apiService.UpdateJobStatus(r.Context(), id, req.Status); err != nil {
		httputils.WriteJSON(w, http.StatusBadRequest, data.BasicResonse{
			Status:  "error",
			Message: "Failed job status update",
		})
		return
	}

	httputils.WriteJSON(w, http.StatusOK, data.BasicResonse{
		Status:  "success",
		Message: "",
	})
}

// helper
func CheckUUID(strId string, w http.ResponseWriter) (uuid.UUID, error) {
	id, err := uuid.Parse(strId)
	if err != nil {
		httputils.WriteJSON(w, 400, data.BasicResonse{
			Status:  "error",
			Message: "Incorrect id type",
		})
		return uuid.Nil, err
	}
	return id, nil
}
