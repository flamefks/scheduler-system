package domain

import (
	"encoding/json"
	"time"
)

// =========================
// Patch
// =========================

type PatchScheduleModel struct {
	RepeatIntervalSec *int32
	TargetRuns        *int32
	NextRunAt         *time.Time
	Status            *string
}

type PatchJSONField struct {
	Set   bool
	Value json.RawMessage
}

type PatchIOConfig struct {
	Payload    PatchJSONField
	Headers    PatchJSONField
	JsonSchema PatchJSONField
	TargetUrl  *string
	Method     *string
}

type PatchJobModel struct {
	Name *string

	Schedule *PatchScheduleModel

	FetcherConfig *PatchIOConfig
	DeliverConfig *PatchIOConfig
}
