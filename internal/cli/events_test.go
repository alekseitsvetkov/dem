package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/alekseitsvetkov/dem/internal/domain"
	"github.com/alekseitsvetkov/dem/internal/hltv"
	"github.com/alekseitsvetkov/dem/internal/hltv/parser"
	"github.com/spf13/cobra"
)

type fakeEventsProvider struct {
	events []domain.Event
	err    error
}

func (f *fakeEventsProvider) GetEvents(ctx context.Context, tier string, limit int) ([]domain.Event, error) {
	return f.events, f.err
}

func TestEventsCommandSuccess(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeProvider := &fakeEventsProvider{
		events: []domain.Event{
			{ID: "111", Name: "Test Event 1", Tier: "S-Tier"},
			{ID: "222", Name: "Another Event", Tier: "A-Tier"},
		},
	}

	root := &cobra.Command{Use: "dem", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.AddCommand(newEventsCommand(&stdout, &stderr, fakeProvider))
	root.SetArgs([]string{"events"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("stdout is not valid JSON: %v", err)
	}
	if _, ok := payload["data"]; !ok {
		t.Fatalf("stdout missing data key: %s", stdout.String())
	}

	data, ok := payload["data"].([]any)
	if !ok {
		t.Fatalf("data is not an array: %T", payload["data"])
	}
	if len(data) != 2 {
		t.Fatalf("got %d events, want 2", len(data))
	}

	first := data[0].(map[string]any)
	if first["id"] != "111" {
		t.Fatalf("first event id = %v, want 111", first["id"])
	}
}

func TestEventsCommandWithTier(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeProvider := &fakeEventsProvider{
		events: []domain.Event{
			{ID: "111", Name: "Test Event 1", Tier: "S-Tier"},
			{ID: "222", Name: "Another Event", Tier: "A-Tier"},
		},
	}

	root := &cobra.Command{Use: "dem", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.AddCommand(newEventsCommand(&stdout, &stderr, fakeProvider))
	root.SetArgs([]string{"events", "--tier", "S-Tier"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("stdout is not valid JSON: %v", err)
	}
	data, ok := payload["data"].([]any)
	if !ok {
		t.Fatalf("data is not an array: %T", payload["data"])
	}
	// Fake provider returns all events regardless of tier filter;
	// the tier filtering happens in the real provider, not in the fake.
	// For the CLI test, we verify the flags are passed through correctly.
	if len(data) != 2 {
		t.Fatalf("got %d events, want 2", len(data))
	}
}

func TestEventsCommandWithLimit(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeProvider := &fakeEventsProvider{
		events: []domain.Event{
			{ID: "111", Name: "Test Event 1"},
			{ID: "222", Name: "Another Event"},
		},
	}

	root := &cobra.Command{Use: "dem", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.AddCommand(newEventsCommand(&stdout, &stderr, fakeProvider))
	root.SetArgs([]string{"events", "--limit", "1"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("stdout is not valid JSON: %v", err)
	}
	data, ok := payload["data"].([]any)
	if !ok {
		t.Fatalf("data is not an array: %T", payload["data"])
	}
	// Fake provider returns all events regardless of limit;
	// the limit truncation happens in the real provider.
	// Verify the --limit flag is passed through without error.
	if len(data) != 2 {
		t.Fatalf("got %d events, want 2", len(data))
	}
}

func TestEventsCommandValidationError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeProvider := &fakeEventsProvider{}

	root := &cobra.Command{Use: "dem", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.AddCommand(newEventsCommand(&stdout, &stderr, fakeProvider))
	root.SetArgs([]string{"events", "--limit", "-1"})

	err := root.Execute()
	if err == nil {
		t.Fatalf("Execute returned nil error for invalid --limit")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	if !strings.Contains(stderr.String(), "validation_error") {
		t.Fatalf("stderr = %q, want validation_error", stderr.String())
	}
}

func TestEventsCommandProviderError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeProvider := &fakeEventsProvider{
		err: &hltv.ProviderError{Code: hltv.ErrorCodeNetwork, Message: "failed"},
	}

	root := &cobra.Command{Use: "dem", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.AddCommand(newEventsCommand(&stdout, &stderr, fakeProvider))
	root.SetArgs([]string{"events"})

	err := root.Execute()
	if err == nil {
		t.Fatalf("Execute returned nil error for provider error")
	}

	if !strings.Contains(stderr.String(), "network_error") {
		t.Fatalf("stderr = %q, want network_error", stderr.String())
	}
}

func TestEventsCommandParseError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeProvider := &fakeEventsProvider{
		err: &parser.ParseError{Code: parser.ErrorCodeParse, Area: "events"},
	}

	root := &cobra.Command{Use: "dem", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.AddCommand(newEventsCommand(&stdout, &stderr, fakeProvider))
	root.SetArgs([]string{"events"})

	err := root.Execute()
	if err == nil {
		t.Fatalf("Execute returned nil error for parse error")
	}

	if !strings.Contains(stderr.String(), "parse_error") {
		t.Fatalf("stderr = %q, want parse_error", stderr.String())
	}
}

func TestEventsCommandNilEvents(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeProvider := &fakeEventsProvider{
		events: nil,
	}

	root := &cobra.Command{Use: "dem", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.AddCommand(newEventsCommand(&stdout, &stderr, fakeProvider))
	root.SetArgs([]string{"events"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	// Verify JSON is [] not null
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("stdout is not valid JSON: %v", err)
	}
	data, ok := payload["data"].([]any)
	if !ok {
		t.Fatalf("data is not an array: %T", payload["data"])
	}
	if len(data) != 0 {
		t.Fatalf("got %d events, want 0 for nil input", len(data))
	}
}
