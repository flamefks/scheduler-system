package domain

import (
	"encoding/json"
	"time"
)

// General
type ScheduleBlock struct {
	RunAt             time.Time `json:"run_at"`
	RepeatIntervalSec int       `json:"repeat_interval_sec"`
	MaxRuns           *int      `json:"max_runs,omitempty"`
}

type RetryPolicyBlock struct {
	MaxRetries         int    `json:"max_retries"`
	Backoff            string `json:"backoff"`
	InitialIntervalSec int    `json:"initial_interval_sec"`
	MaxIntervalSec     int    `json:"max_interval_sec"`
}

type ResponseSchemaBlock struct {
	Type   string          `json:"type"`
	Schema json.RawMessage `json:"schema,omitempty"`
}

// Source
type HTTPRequestSourceBlock struct {
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers,omitempty"`
	QueryParams map[string]string `json:"query_params,omitempty"`
	Body        json.RawMessage   `json:"body,omitempty"`
	TimeoutSec  int               `json:"timeout_sec"`
}

type SourceBlock struct {
	Type    string                 `json:"type"`
	Request HTTPRequestSourceBlock `json:"request"`
}

// Dst
type HTTPDestinationBlock struct {
	URL        string            `json:"url"`
	Method     string            `json:"method"`
	Headers    map[string]string `json:"headers,omitempty"`
	TimeoutSec int               `json:"timeout_sec"`
}

type GRPCDestinationBlock struct {
	Endpoint string            `json:"endpoint"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type DestinationBlock struct {
	Type string                `json:"type"`
	HTTP *HTTPDestinationBlock `json:"http,omitempty"`
	GRPC *GRPCDestinationBlock `json:"grpc,omitempty"`
}

type CreateJobRequest struct {
	Schedule       ScheduleBlock        `json:"schedule"`
	Source         SourceBlock          `json:"source"`
	ResponseSchema *ResponseSchemaBlock `json:"response_schema,omitempty"`
	Destination    DestinationBlock     `json:"destination"`
	RetryPolicy    RetryPolicyBlock     `json:"retry_policy"`
}
