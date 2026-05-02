package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
)

func TestVersionWritesJSONEnvelope(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRootCommand(&stdout, &stderr)
	root.SetArgs([]string{"version"})

	if err := root.Execute(); err != nil {
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
	if _, ok := payload["meta"]; !ok {
		t.Fatalf("stdout missing meta key: %s", stdout.String())
	}
}

func TestHelpDoesNotWriteError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRootCommand(&stdout, &stderr)
	root.SetArgs([]string{"--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestUnknownCommandReturnsError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRootCommand(&stdout, &stderr)
	root.SetArgs([]string{"does-not-exist"})

	if err := root.Execute(); err == nil {
		t.Fatalf("Execute returned nil error for unknown command")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty because centralized Execute writes JSON errors", stderr.String())
	}
}

func TestAdditionalCommandCanBeRegistered(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRootCommand(&stdout, &stderr)
	root.AddCommand(&cobra.Command{
		Use: "probe",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := cmd.OutOrStdout().Write([]byte(`{"data":{"ok":true},"meta":{}}` + "\n"))
			return err
		},
	})
	root.SetArgs([]string{"probe"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"ok":true`)) {
		t.Fatalf("stdout = %q, want registered command output", stdout.String())
	}
}
