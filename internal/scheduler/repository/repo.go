package repository

import (
	"context"

	"github.com/google/uuid"
)

type PostgresRepo interface {
	ClaimNextJob(ctx context.Context) (uuid.UUID, error)
}
