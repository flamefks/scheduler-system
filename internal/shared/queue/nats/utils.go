package nats

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

func GetJobIDFromHeader(strJobId string) (uuid.UUID, error) {
	if strJobId == "" {
		return uuid.Nil, errors.New("missing job-id header")
	}

	jobId, err := uuid.Parse(strJobId)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid job-id: %w", err)
	}

	return jobId, nil
}
