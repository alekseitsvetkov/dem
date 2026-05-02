package output

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestWriteJSONWritesEnvelope(t *testing.T) {
	var buf bytes.Buffer

	err := WriteJSON(&buf, map[string]string{"name": "dem"}, map[string]any{"source": "test"})
	if err != nil {
		t.Fatalf("WriteJSON returned error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if _, ok := payload["data"]; !ok {
		t.Fatalf("output missing data key: %s", buf.String())
	}
	if _, ok := payload["meta"]; !ok {
		t.Fatalf("output missing meta key: %s", buf.String())
	}
}

func TestWriteJSONUsesEmptyObjectForNilMeta(t *testing.T) {
	var buf bytes.Buffer

	err := WriteJSON(&buf, map[string]string{"name": "dem"}, nil)
	if err != nil {
		t.Fatalf("WriteJSON returned error: %v", err)
	}

	var payload struct {
		Meta map[string]any `json:"meta"`
	}
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if payload.Meta == nil {
		t.Fatalf("meta was null, want empty object: %s", buf.String())
	}
	if len(payload.Meta) != 0 {
		t.Fatalf("meta = %v, want empty object", payload.Meta)
	}
}
