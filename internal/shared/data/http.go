package data

import (
	"encoding/json"
	"net/http"
)

type Request struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    json.RawMessage
}

type ExternalResponse struct {
	StatusCode int
	Headers    http.Header
	Body       json.RawMessage
}

type BasicResonse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
