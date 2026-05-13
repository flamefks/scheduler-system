package data

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type IOConfig struct {
	Payload    json.RawMessage
	Headers    json.RawMessage
	JsonSchema json.RawMessage
	TargetUrl  string
	Method     string
}

type Schedule struct {
	Status            string
	RepeatIntervalSec int32
	TargetRuns        int32
	DoneRuns          int32
	NextRunAt         time.Time
	LastScheduledAt   *time.Time
	LastRunTakenAt    *time.Time
}

type Job struct {
	ID   uuid.UUID
	Name string

	Schedule Schedule

	FetcherConfig IOConfig
	DeliverConfig IOConfig

	CreatedAt time.Time
	UpdatedAt time.Time
}
