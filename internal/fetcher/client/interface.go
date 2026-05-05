package client

import (
	"context"

	"github.com/flamefks/scheduler-system/internal/shared/data"
)

type Client interface {
	Do(ctx context.Context, req *data.Request) (*data.ExternalResponse, error)
}
