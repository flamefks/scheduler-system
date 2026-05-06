package utils

import (
	"encoding/json"
	"testing"
)

func TestValidateRawMessageWithSchema(t *testing.T) {
	schema := json.RawMessage(`{
		"type": "object",
		"required": ["name", "attempts"],
		"properties": {
			"name": {"type": "string"},
			"attempts": {"type": "integer", "minimum": 1}
		},
		"additionalProperties": false
	}`)

	t.Run("valid payload", func(t *testing.T) {
		payload := json.RawMessage(`{"name":"job-1","attempts":3}`)

		if err := ValidateRawMessageWithSchema(schema, payload); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("schema violation", func(t *testing.T) {
		payload := json.RawMessage(`{"name":"job-1","attempts":0}`)

		if err := ValidateRawMessageWithSchema(schema, payload); err == nil {
			t.Fatal("expected validation error")
		}
	})

	t.Run("invalid schema json", func(t *testing.T) {
		err := ValidateRawMessageWithSchema(json.RawMessage(`{bad}`), json.RawMessage(`{}`))
		if err == nil {
			t.Fatal("expected schema parse error")
		}
	})

	t.Run("invalid payload json", func(t *testing.T) {
		err := ValidateRawMessageWithSchema(schema, json.RawMessage(`{bad}`))
		if err == nil {
			t.Fatal("expected payload parse error")
		}
	})
}
