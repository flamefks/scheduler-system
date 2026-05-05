package http

import (
	"encoding/json"
	"time"

	"github.com/flamefks/scheduler-system/internal/shared/data"
)

// =========================
// CREATE
// =========================
// Requests - CreateJob
type ScheduleBlockCreateJobRequest struct {
	NextRunAt         time.Time `json:"next_run_at"`
	RepeatIntervalSec int32     `json:"repeat_interval_sec"`
	TargetRuns        int32     `json:"target_runs"`
}

type IOBlockCreateJobRequest struct {
	TargetURL string          `json:"target_url"`
	Method    string          `json:"method"`
	Payload   json.RawMessage `json:"payload"`
	Headers   json.RawMessage `json:"headers"`
}

type CreateJobRequest struct {
	Name     string                        `json:"name"`
	Schedule ScheduleBlockCreateJobRequest `json:"schedule"`

	FetcherConfig IOBlockCreateJobRequest `json:"fetcher_config"`
	DeliverConfig IOBlockCreateJobRequest `json:"deliver_config"`
}

// =========================
// Patch
// =========================
// Requests - PatchJob

type ScheduleBlockPatchJobRequest struct {
	NextRunAt         *time.Time `json:"next_run_at"`
	RepeatIntervalSec *int32     `json:"repeat_interval_sec"`
	TargetRuns        *int32     `json:"target_runs"`
}

type IOBlockPatchJobRequest struct {
	Payload *json.RawMessage `json:"payload"`
	Headers *json.RawMessage `json:"headers"`
}

type PatchJobRequest struct {
	Name     *string                       `json:"name"`
	Schedule *ScheduleBlockPatchJobRequest `json:"schedule"`

	FetcherConfig *IOBlockPatchJobRequest `json:"fetcher_config"`
	DeliverConfig *IOBlockPatchJobRequest `json:"deliver_config"`
}

// UpdateStatusJob
type UpdateStatusRequest struct {
	Status string `json:"status"`
}

// =========================
// Responses
// =========================
type GetJobResponse struct {
	Status string    `json:"status"`
	Data   *data.Job `json:"data"`
}

type ResponseWithUUID struct {
	Status string `json:"status"`
}
