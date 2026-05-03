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

type fakeResultsProvider struct {
	results []domain.Result
	err     error
}

func (f *fakeResultsProvider) GetResults(ctx context.Context, eventID int, limit int) ([]domain.Result, error) {
	return f.results, f.err
}

func TestResultsCommandSuccess(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeProvider := &fakeResultsProvider{
		results: []domain.Result{
			{MatchID: "333", Team1: "Team Alpha", Team2: "Team Beta", Score: "2-1"},
			{MatchID: "444", Team1: "Team Gamma", Team2: "Team Delta", Score: "16-10"},
		},
	}

	root := &cobra.Command{Use: "dem", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.AddCommand(newResultsCommand(&stdout, &stderr, fakeProvider))
	root.SetArgs([]string{"results"})

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
		t.Fatalf("got %d results, want 2", len(data))
	}

	first := data[0].(map[string]any)
	if first["match_id"] != "333" {
		t.Fatalf("first result match_id = %v, want 333", first["match_id"])
	}
	if first["team1"] != "Team Alpha" {
		t.Fatalf("first result team1 = %v, want Team Alpha", first["team1"])
	}
}

func TestResultsCommandWithLimit(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeProvider := &fakeResultsProvider{
		results: []domain.Result{
			{MatchID: "333", Team1: "Team Alpha", Team2: "Team Beta", Score: "2-1"},
			{MatchID: "444", Team1: "Team Gamma", Team2: "Team Delta", Score: "16-10"},
		},
	}

	root := &cobra.Command{Use: "dem", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.AddCommand(newResultsCommand(&stdout, &stderr, fakeProvider))
	root.SetArgs([]string{"results", "--limit", "1"})

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
	if len(data) != 2 {
		t.Fatalf("got %d results, want 2", len(data))
	}
}

func TestResultsCommandValidationError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeProvider := &fakeResultsProvider{}

	root := &cobra.Command{Use: "dem", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.AddCommand(newResultsCommand(&stdout, &stderr, fakeProvider))
	root.SetArgs([]string{"results", "--limit", "-1"})

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

func TestResultsCommandProviderError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeProvider := &fakeResultsProvider{
		err: &hltv.ProviderError{Code: hltv.ErrorCodeNetwork, Message: "failed"},
	}

	root := &cobra.Command{Use: "dem", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.AddCommand(newResultsCommand(&stdout, &stderr, fakeProvider))
	root.SetArgs([]string{"results"})

	err := root.Execute()
	if err == nil {
		t.Fatalf("Execute returned nil error for provider error")
	}

	if !strings.Contains(stderr.String(), "network_error") {
		t.Fatalf("stderr = %q, want network_error", stderr.String())
	}
}

func TestResultsCommandParseError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeProvider := &fakeResultsProvider{
		err: &parser.ParseError{Code: parser.ErrorCodeParse, Area: "results"},
	}

	root := &cobra.Command{Use: "dem", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.AddCommand(newResultsCommand(&stdout, &stderr, fakeProvider))
	root.SetArgs([]string{"results"})

	err := root.Execute()
	if err == nil {
		t.Fatalf("Execute returned nil error for parse error")
	}

	if !strings.Contains(stderr.String(), "parse_error") {
		t.Fatalf("stderr = %q, want parse_error", stderr.String())
	}
}

func TestResultsCommandNilResults(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeProvider := &fakeResultsProvider{
		results: nil,
	}

	root := &cobra.Command{Use: "dem", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.AddCommand(newResultsCommand(&stdout, &stderr, fakeProvider))
	root.SetArgs([]string{"results"})

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
		t.Fatalf("got %d results, want 0 for nil input", len(data))
	}
}
