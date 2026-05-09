package http

import (
	"encoding/json"
	"testing"
)

func TestOptionalRawMessage_UnmarshalJSON(t *testing.T) {
	t.Run("missing field", func(t *testing.T) {
		var req IOBlockPatchJobRequest
		if err := json.Unmarshal([]byte(`{}`), &req); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if req.Headers.Set {
			t.Fatal("expected missing headers to keep Set=false")
		}
		if req.Headers.Value != nil {
			t.Fatalf("expected nil value, got %s", string(req.Headers.Value))
		}
	})

	t.Run("explicit null", func(t *testing.T) {
		var req IOBlockPatchJobRequest
		if err := json.Unmarshal([]byte(`{"headers":null}`), &req); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !req.Headers.Set {
			t.Fatal("expected explicit null headers to set Set=true")
		}
		if req.Headers.Value != nil {
			t.Fatalf("expected nil value, got %s", string(req.Headers.Value))
		}
	})

	t.Run("value", func(t *testing.T) {
		var req IOBlockPatchJobRequest
		if err := json.Unmarshal([]byte(`{"headers":{"Authorization":"token"}}`), &req); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !req.Headers.Set {
			t.Fatal("expected headers to set Set=true")
		}
		if string(req.Headers.Value) != `{"Authorization":"token"}` {
			t.Fatalf("unexpected value: %s", string(req.Headers.Value))
		}
	})
}
