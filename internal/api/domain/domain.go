package domain

import (
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

type PatchIOConfig struct {
	Payload   *[]byte
	Headers   *[]byte
	TargetUrl *string
	Method    *string
}

type PatchJobModel struct {
	Name *string

	Schedule *PatchScheduleModel

	FetcherConfig *PatchIOConfig
	DeliverConfig *PatchIOConfig
}
