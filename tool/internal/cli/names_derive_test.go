package cli_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"threehillpath.com/claude-plan-workflow/tool/internal/cli"
)

// TestNamesDeriveType verifies that 'marvin names derive <issue> --type bug --phase N'
// emits JSON with main_branch/phase_branch in the new nested shape and echoes
// the resolved type field.
func TestNamesDeriveType(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(&stdout, &stderr)
	root.SetArgs([]string{"names", "derive", "42", "--type", "bug", "--phase", "3", "--worktree-base", ".worktrees"})

	if err := root.Execute(); err != nil {
		t.Fatalf("names derive returned error: %v\nstderr: %s", err, stderr.String())
	}

	var out struct {
		Type        string `json:"type"`
		MainBranch  string `json:"main_branch"`
		PhaseBranch string `json:"phase_branch"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal stdout: %v\nstdout: %s", err, stdout.String())
	}

	if out.Type != "bug" {
		t.Errorf("type = %q, want %q", out.Type, "bug")
	}
	if out.MainBranch != "bug/PLAN-00042/main" {
		t.Errorf("main_branch = %q, want %q", out.MainBranch, "bug/PLAN-00042/main")
	}
	if out.PhaseBranch != "bug/PLAN-00042/phase-3" {
		t.Errorf("phase_branch = %q, want %q", out.PhaseBranch, "bug/PLAN-00042/phase-3")
	}
}

// TestNamesDeriveDefaultType verifies that omitting --type defaults to "feature".
func TestNamesDeriveDefaultType(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(&stdout, &stderr)
	root.SetArgs([]string{"names", "derive", "42"})

	if err := root.Execute(); err != nil {
		t.Fatalf("names derive returned error: %v\nstderr: %s", err, stderr.String())
	}

	var out struct {
		Type       string `json:"type"`
		MainBranch string `json:"main_branch"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal stdout: %v\nstdout: %s", err, stdout.String())
	}
	if out.Type != "feature" {
		t.Errorf("type = %q, want %q", out.Type, "feature")
	}
	if out.MainBranch != "feature/PLAN-00042/main" {
		t.Errorf("main_branch = %q, want %q", out.MainBranch, "feature/PLAN-00042/main")
	}
}

// TestNamesDeriveInvalidType verifies that an invalid --type value exits 1
// with a clear error message.
func TestNamesDeriveInvalidType(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(&stdout, &stderr)
	root.SetArgs([]string{"names", "derive", "42", "--type", "bogus"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for invalid --type, got nil")
	}
	ce, ok := err.(*cli.CLIError)
	if !ok {
		t.Fatalf("expected *cli.CLIError, got %T: %v", err, err)
	}
	if ce.Code != 1 {
		t.Errorf("expected exit code 1, got %d", ce.Code)
	}
}
