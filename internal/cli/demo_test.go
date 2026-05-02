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

type fakeDemoProvider struct {
	link domain.DemoLink
	err  error
}

func (f *fakeDemoProvider) GetDemo(ctx context.Context, matchID int) (domain.DemoLink, error) {
	return f.link, f.err
}

func TestDemoCommand_Success(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeProvider := &fakeDemoProvider{
		link: domain.DemoLink{
			MatchID:  "107224",
			MatchURL: "https://www.hltv.org/matches/107224/-",
			DemoURL:  "https://www.hltv.org/download/demo/107224",
		},
	}

	root := &cobra.Command{Use: "dem", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.AddCommand(newDemoCommand(&stdout, &stderr, fakeProvider))
	root.SetArgs([]string{"demo", "107224"})

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
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("data is not an object: %T", payload["data"])
	}
	if data["match_id"] != "107224" {
		t.Fatalf("match_id = %v, want 107224", data["match_id"])
	}
	if data["demo_url"] != "https://www.hltv.org/download/demo/107224" {
		t.Fatalf("demo_url = %v, want https://www.hltv.org/download/demo/107224", data["demo_url"])
	}
}

func TestDemoCommand_Unavailable(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeProvider := &fakeDemoProvider{
		link: domain.DemoLink{
			MatchID:  "99999",
			MatchURL: "https://www.hltv.org/matches/99999/-",
			// DemoURL is zero value (empty) — no demo available
		},
	}

	root := &cobra.Command{Use: "dem", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.AddCommand(newDemoCommand(&stdout, &stderr, fakeProvider))
	root.SetArgs([]string{"demo", "99999"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("Execute returned error for unavailable demo: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("stdout is not valid JSON: %v", err)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("data is not an object: %T", payload["data"])
	}
	if data["match_id"] != "99999" {
		t.Fatalf("match_id = %v, want 99999", data["match_id"])
	}
	// D-04: demo_url key should not exist when demo is unavailable
	if _, exists := data["demo_url"]; exists {
		t.Fatalf("demo_url key exists, want absent for unavailable demo")
	}
}

func TestDemoCommand_ValidationError_NonNumeric(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeProvider := &fakeDemoProvider{}

	root := &cobra.Command{Use: "dem", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.AddCommand(newDemoCommand(&stdout, &stderr, fakeProvider))
	root.SetArgs([]string{"demo", "abc"})

	err := root.Execute()
	if err == nil {
		t.Fatalf("Execute returned nil error for non-numeric match-id")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "validation_error") {
		t.Fatalf("stderr = %q, want validation_error", stderr.String())
	}
}

func TestDemoCommand_ValidationError_Zero(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeProvider := &fakeDemoProvider{}

	root := &cobra.Command{Use: "dem", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.AddCommand(newDemoCommand(&stdout, &stderr, fakeProvider))
	root.SetArgs([]string{"demo", "0"})

	err := root.Execute()
	if err == nil {
		t.Fatalf("Execute returned nil error for zero match-id")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "validation_error") {
		t.Fatalf("stderr = %q, want validation_error", stderr.String())
	}
}

func TestDemoCommand_ValidationError_Negative(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeProvider := &fakeDemoProvider{}

	root := &cobra.Command{Use: "dem", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.AddCommand(newDemoCommand(&stdout, &stderr, fakeProvider))
	root.SetArgs([]string{"demo", "-5"})

	err := root.Execute()
	if err == nil {
		t.Fatalf("Execute returned nil error for negative match-id")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "validation_error") {
		t.Fatalf("stderr = %q, want validation_error", stderr.String())
	}
}

func TestDemoCommand_ProviderError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeProvider := &fakeDemoProvider{
		err: &hltv.ProviderError{Code: hltv.ErrorCodeNetwork, Message: "connection failed"},
	}

	root := &cobra.Command{Use: "dem", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.AddCommand(newDemoCommand(&stdout, &stderr, fakeProvider))
	root.SetArgs([]string{"demo", "107224"})

	err := root.Execute()
	if err == nil {
		t.Fatalf("Execute returned nil error for provider error")
	}
	if !strings.Contains(stderr.String(), "network_error") {
		t.Fatalf("stderr = %q, want network_error", stderr.String())
	}
}

func TestDemoCommand_ParseError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	fakeProvider := &fakeDemoProvider{
		err: &parser.ParseError{Code: parser.ErrorCodeParse, Area: "demo", Message: "failed to parse page"},
	}

	root := &cobra.Command{Use: "dem", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.AddCommand(newDemoCommand(&stdout, &stderr, fakeProvider))
	root.SetArgs([]string{"demo", "107224"})

	err := root.Execute()
	if err == nil {
		t.Fatalf("Execute returned nil error for parse error")
	}
	if !strings.Contains(stderr.String(), "parse_error") {
		t.Fatalf("stderr = %q, want parse_error", stderr.String())
	}
}
