package client

import (
	"context"

	"github.com/flamefks/scheduler-system/internal/shared"
)

type Client interface {
	Do(ctx context.Context, req *shared.Request) (*shared.Response, error)
}
