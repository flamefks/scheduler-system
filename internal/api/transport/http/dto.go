package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/flamefks/scheduler-system/internal/shared/data"
	"github.com/flamefks/scheduler-system/internal/shared/utils"
)

type OptionalRawMessage struct {
	Set   bool
	Value json.RawMessage
}

func (o *OptionalRawMessage) UnmarshalJSON(data []byte) error {
	o.Set = true

	if string(data) == "null" {
		o.Value = nil
		return nil
	}

	o.Value = append(o.Value[:0], data...)
	return nil
}

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
	TargetURL  string          `json:"target_url"`
	Method     string          `json:"method"`
	Payload    json.RawMessage `json:"payload"`
	Headers    json.RawMessage `json:"headers"`
	JsonSchema json.RawMessage `json:"json_schema"`
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
	TargetURL  *string            `json:"target_url"`
	Method     *string            `json:"method"`
	Payload    OptionalRawMessage `json:"payload"`
	Headers    OptionalRawMessage `json:"headers"`
	JsonSchema OptionalRawMessage `json:"json_schema"`
}

type PatchJobRequest struct {
	Name     *string                       `json:"name"`
	Schedule *ScheduleBlockPatchJobRequest `json:"schedule"`

	FetcherConfig *IOBlockPatchJobRequest `json:"fetcher_config"`
	DeliverConfig *IOBlockPatchJobRequest `json:"deliver_config"`
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

// =========================
// Validators
// =========================
func ValidateCreateJobRequest(req *CreateJobRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if err := validateCreateSchedule(req.Schedule); err != nil {
		return fmt.Errorf("schedule: %w", err)
	}
	if err := validateCreateIOConfig(req.FetcherConfig); err != nil {
		return fmt.Errorf("fetcher_config: %w", err)
	}
	if err := validateCreateIOConfig(req.DeliverConfig); err != nil {
		return fmt.Errorf("deliver_config: %w", err)
	}
	return nil
}

func ValidatePatchJobRequest(req *PatchJobRequest) error {
	if req.Name != nil && *req.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if req.Schedule != nil {
		if err := validatePatchSchedule(req.Schedule); err != nil {
			return fmt.Errorf("schedule: %w", err)
		}
	}
	if req.FetcherConfig != nil {
		if err := validatePatchIOConfig(req.FetcherConfig); err != nil {
			return fmt.Errorf("fetcher_config: %w", err)
		}
	}
	if req.DeliverConfig != nil {
		if err := validatePatchIOConfig(req.DeliverConfig); err != nil {
			return fmt.Errorf("deliver_config: %w", err)
		}
	}
	return nil
}

func validateCreateSchedule(schedule ScheduleBlockCreateJobRequest) error {
	if schedule.RepeatIntervalSec < 0 {
		return fmt.Errorf("repeat_interval_sec cannot be negative")
	}
	if schedule.TargetRuns < 0 {
		return fmt.Errorf("target_runs cannot be negative")
	}
	return nil
}

func validatePatchSchedule(schedule *ScheduleBlockPatchJobRequest) error {
	if schedule.RepeatIntervalSec != nil && *schedule.RepeatIntervalSec < 0 {
		return fmt.Errorf("repeat_interval_sec cannot be negative")
	}
	if schedule.TargetRuns != nil && *schedule.TargetRuns < 0 {
		return fmt.Errorf("target_runs cannot be negative")
	}
	return nil
}

func validateCreateIOConfig(cfg IOBlockCreateJobRequest) error {
	if err := validateHTTPUrl(cfg.TargetURL); err != nil {
		return err
	}
	if err := validateHttpMethod(cfg.Method); err != nil {
		return err
	}
	if _, err := validateJSONField("payload", &cfg.Payload); err != nil {
		return err
	}
	if err := validateHeaders(&cfg.Headers); err != nil {
		return err
	}
	if err := validateJsonSchema(&cfg.JsonSchema); err != nil {
		return err
	}
	return nil
}

func validatePatchIOConfig(cfg *IOBlockPatchJobRequest) error {
	if cfg.TargetURL != nil {
		if err := validateHTTPUrl(*cfg.TargetURL); err != nil {
			return err
		}
	}
	if cfg.Method != nil {
		if err := validateHttpMethod(*cfg.Method); err != nil {
			return err
		}
	}
	if cfg.Payload.Set {
		if _, err := validateJSONField("payload", &cfg.Payload.Value); err != nil {
			return err
		}
	}
	if cfg.Headers.Set {
		if err := validateHeaders(&cfg.Headers.Value); err != nil {
			return err
		}
	}
	if cfg.JsonSchema.Set {
		if err := validateJsonSchema(&cfg.JsonSchema.Value); err != nil {
			return err
		}
	}
	return nil
}

func validateJSONField(name string, raw *json.RawMessage) (bool, error) {
	if len(*raw) == 0 {
		return false, nil
	}
	if bytes.Equal(*raw, []byte("null")) {
		*raw = nil
		return false, nil
	}
	if !json.Valid(*raw) {
		return false, fmt.Errorf("%s must be valid json", name)
	}
	return true, nil
}

func validateHeaders(raw *json.RawMessage) error {
	notNill, err := validateJSONField("headers", raw)
	if err != nil {
		return err
	}
	if !notNill {
		return nil
	}
	var obj map[string]string
	if err := json.Unmarshal(*raw, &obj); err != nil {
		return fmt.Errorf("headers must be json object with string values: %w", err)
	}
	return nil
}

func validateJsonSchema(raw *json.RawMessage) error {
	notNill, err := validateJSONField("schema", raw)
	if err != nil {
		return err
	}
	if !notNill {
		return nil
	}
	if _, err := utils.CompileJsonSchema(raw); err != nil {
		return fmt.Errorf("json_schema compile failed: %w", err)
	}
	return nil
}

func validateHTTPUrl(raw string) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return fmt.Errorf("invalid_url: %w", err)
	}

	switch u.Scheme {
	case "http", "https":
	default:
		return fmt.Errorf("invalid_scheme: %s", u.Scheme)
	}

	if u.Host == "" {
		return fmt.Errorf("host is required")
	}

	return nil
}

func validateHttpMethod(method string) error {
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
	default:
		return fmt.Errorf("Invalid http method: %s", method)
	}
	return nil
}
