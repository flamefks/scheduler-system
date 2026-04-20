package data

import "net/http"

type Request struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    []byte
}

type ExternalResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

type BasicResonse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
