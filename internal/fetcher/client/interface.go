package client

import "context"

type Client interface {
	Do(ctx context.Context, req Request) (*Response, error)
}
