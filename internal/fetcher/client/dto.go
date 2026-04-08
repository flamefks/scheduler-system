package client

import "net/http"

type Request struct {
	Method      string
	URL         string
	Headers     map[string]string
	QueryParams map[string]string
	Body        []byte
}

type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}
