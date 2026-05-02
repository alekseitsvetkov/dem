package output

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestWriteErrorJSONWritesEnvelope(t *testing.T) {
	var buf bytes.Buffer

	err := WriteErrorJSON(&buf, "validation_error", "invalid input", map[string]any{"field": "match_id"})
	if err != nil {
		t.Fatalf("WriteErrorJSON returned error: %v", err)
	}

	var payload ErrorResponse
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if payload.Error.Code != "validation_error" {
		t.Fatalf("code = %q, want validation_error", payload.Error.Code)
	}
	if payload.Error.Message == "" {
		t.Fatalf("message is empty")
	}
	if payload.Error.Details == nil {
		t.Fatalf("details was null, want object")
	}
}

func TestWriteErrorJSONUsesEmptyObjectForNilDetails(t *testing.T) {
	var buf bytes.Buffer

	err := WriteErrorJSON(&buf, "validation_error", "invalid input", nil)
	if err != nil {
		t.Fatalf("WriteErrorJSON returned error: %v", err)
	}

	var payload ErrorResponse
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if payload.Error.Details == nil {
		t.Fatalf("details was null, want empty object")
	}
	if len(payload.Error.Details) != 0 {
		t.Fatalf("details = %v, want empty object", payload.Error.Details)
	}
}
