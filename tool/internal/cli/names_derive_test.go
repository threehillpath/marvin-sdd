package cli_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"threehillpath.com/marvin-sdd/tool/internal/cli"
	"threehillpath.com/marvin-sdd/tool/internal/exectest"
)

// TestNamesDeriveType verifies that 'marvin names derive <issue> --type bug --phase N'
// emits JSON with main_branch/phase_branch in the new nested shape and echoes
// the resolved type field.
func TestNamesDeriveType(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"names", "derive", "42", "--type", "bug", "--phase", "3", "--worktree-base", ".worktrees", "--json"})

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
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"names", "derive", "42", "--json"})

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

// TestNamesDeriveDefaultPlainText is this phase's TDD entry point: `names derive`
// without --json must print one key:value line per populated field, in
// struct-field order, with the nested title_prefix flattened to three lines.
func TestNamesDeriveDefaultPlainText(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"names", "derive", "42", "--type", "bug", "--phase", "3", "--worktree-base", ".worktrees"})

	if err := root.Execute(); err != nil {
		t.Fatalf("names derive returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := "plan_number: plan-00042\n" +
		"type: bug\n" +
		"main_branch: bug/PLAN-00042/main\n" +
		"phase_branch: bug/PLAN-00042/phase-3\n" +
		"worktree_path: .worktrees/phase-00042-3\n" +
		"title_prefix_arch: [PLAN-00042-ARCH]\n" +
		"title_prefix_impl: [PLAN-00042]\n" +
		"title_prefix_phase: [PLAN-00042-3]\n"

	if stdout.String() != want {
		t.Errorf("plain-text output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

// TestNamesDeriveNoPhasePlainText verifies the plain-text omission path for
// the invocation with no --phase: phase_branch, worktree_path, and
// title_prefix_phase must all be absent (not printed as empty values), which
// is the form impl-plan/start-impl call.
func TestNamesDeriveNoPhasePlainText(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"names", "derive", "42"})

	if err := root.Execute(); err != nil {
		t.Fatalf("names derive returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := "plan_number: plan-00042\n" +
		"type: feature\n" +
		"main_branch: feature/PLAN-00042/main\n" +
		"title_prefix_arch: [PLAN-00042-ARCH]\n" +
		"title_prefix_impl: [PLAN-00042]\n"

	if stdout.String() != want {
		t.Errorf("plain-text output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

// TestNamesDeriveJSONUnchanged verifies that --json for the same input as
// TestNamesDeriveDefaultPlainText reproduces the exact pre-Component-5 JSON
// shape, byte-identical, unaffected by the new plain-text default.
func TestNamesDeriveJSONUnchanged(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"names", "derive", "42", "--type", "bug", "--phase", "3", "--worktree-base", ".worktrees", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("names derive --json returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := `{
  "plan_number": "plan-00042",
  "type": "bug",
  "main_branch": "bug/PLAN-00042/main",
  "phase_branch": "bug/PLAN-00042/phase-3",
  "worktree_path": ".worktrees/phase-00042-3",
  "title_prefix": {
    "arch": "[PLAN-00042-ARCH]",
    "impl": "[PLAN-00042]",
    "phase": "[PLAN-00042-3]"
  }
}
`
	if stdout.String() != want {
		t.Errorf("--json output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

// TestNamesDeriveTaskJSON verifies that 'marvin names derive <n> --task
// --type bug --json' emits exactly the five-key task-shaped field set,
// pretty-printed, with all PLAN-shaped fields absent.
func TestNamesDeriveTaskJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"names", "derive", "91", "--task", "--type", "bug", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("names derive --task returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := `{
  "task_number": "TASK-00091",
  "type": "bug",
  "task_branch": "bug/TASK-00091",
  "worktree_path": ".worktrees/task-00091",
  "title_prefix": {
    "task": "[TASK-00091]"
  }
}
`
	if stdout.String() != want {
		t.Errorf("--task --json output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

// TestNamesDeriveTaskPlainText verifies the plain-text mirror of the --task
// --json field set: the same five keys, in the same order, with every
// PLAN-shaped line omitted.
func TestNamesDeriveTaskPlainText(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"names", "derive", "91", "--task", "--type", "bug", "--worktree-base", ".worktrees"})

	if err := root.Execute(); err != nil {
		t.Fatalf("names derive --task returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := "task_number: TASK-00091\n" +
		"type: bug\n" +
		"task_branch: bug/TASK-00091\n" +
		"worktree_path: .worktrees/task-00091\n" +
		"title_prefix_task: [TASK-00091]\n"

	if stdout.String() != want {
		t.Errorf("--task plain-text output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

// TestNamesDeriveTaskPhaseConflict verifies that --task combined with a
// non-zero --phase is rejected with a Code 1 error naming both flags.
func TestNamesDeriveTaskPhaseConflict(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"names", "derive", "91", "--task", "--phase", "2"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for --task with --phase, got nil")
	}
	ce, ok := err.(*cli.CLIError)
	if !ok {
		t.Fatalf("expected *cli.CLIError, got %T: %v", err, err)
	}
	if ce.Code != 1 {
		t.Errorf("expected exit code 1, got %d", ce.Code)
	}
	if !strings.Contains(ce.Msg, "--task") || !strings.Contains(ce.Msg, "--phase") {
		t.Errorf("expected error message to name both --task and --phase, got %q", ce.Msg)
	}
}

// TestNamesDeriveInvalidType verifies that an invalid --type value exits 1
// with a clear error message.
func TestNamesDeriveInvalidType(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
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
