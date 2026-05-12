package shared

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/flamefks/scheduler-system/internal/shared/data"
)

const (
	maxResponseBytesSize = 100 << 20
)

type HTTPClient struct {
	client *http.Client
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{},
	}
}

func (c *HTTPClient) Do(ctx context.Context, req *data.Request) (*data.ExternalResponse, error) {
	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		req.Method,
		parsedURL.String(),
		bytes.NewReader(req.Body),
	)
	if err != nil {
		return nil, err
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	limitedBody := io.LimitReader(resp.Body, maxResponseBytesSize+1)

	body, err := io.ReadAll(limitedBody)
	if err != nil {
		return nil, err
	}

	if int64(len(body)) > maxResponseBytesSize {
		return nil, fmt.Errorf("response body too large: limit %d bytes", maxResponseBytesSize)
	}

	return &data.ExternalResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header.Clone(),
		Body:       body,
	}, nil
}
