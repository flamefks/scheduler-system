package client

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
)

type HTTPClient struct {
	client *http.Client
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{},
	}
}

func (c *HTTPClient) Do(ctx context.Context, req Request) (*Response, error) {
	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		return nil, err
	}

	if len(req.QueryParams) > 0 {
		q := parsedURL.Query()
		for k, v := range req.QueryParams {
			q.Set(k, v)
		}
		parsedURL.RawQuery = q.Encode()
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header.Clone(),
		Body:       body,
	}, nil
}
