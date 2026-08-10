package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"threehillpath.com/claude-plan-workflow/tool/internal/cli"
	"threehillpath.com/claude-plan-workflow/tool/internal/exectest"
)

// The remaining tests in this file exercise the --json flag on the 6 of 7
// Phase 3 commands not covered by TestBoardListJSONFidelity (board_json_test.go):
// names derive, parse title, parse phase-list, pr find, pr base, issue list.
// Output is unconditionally JSON today (plain-text-by-default lands in a
// later phase), so each test asserts --json parses without error and the
// command's normal JSON output is still produced.

func TestNamesDeriveJSONFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"names", "derive", "42", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("names derive --json returned error: %v\nstderr: %s", err, stderr.String())
	}

	var out struct {
		PlanNumber string `json:"plan_number"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal stdout: %v\nstdout: %s", err, stdout.String())
	}
	if out.PlanNumber != "plan-00042" {
		t.Errorf("plan_number = %q, want %q", out.PlanNumber, "plan-00042")
	}
}

func TestParseTitleJSONFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"parse", "title", "[PLAN-00042-3] Some title", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("parse title --json returned error: %v\nstderr: %s", err, stderr.String())
	}

	var out struct {
		Found bool `json:"found"`
		Plan  int  `json:"plan"`
		Phase int  `json:"phase"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal stdout: %v\nstdout: %s", err, stdout.String())
	}
	if !out.Found || out.Plan != 42 || out.Phase != 3 {
		t.Errorf("got found=%v plan=%d phase=%d, want found=true plan=42 phase=3", out.Found, out.Plan, out.Phase)
	}
}

func TestParsePhaseListJSONFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("Phases created:\n- #12 [PLAN-00042-1] Real\n")
	root := cli.NewRootCmd(stdin, &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"parse", "phase-list", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("parse phase-list --json returned error: %v\nstderr: %s", err, stderr.String())
	}

	var out struct {
		Found  bool  `json:"found"`
		Issues []int `json:"issues"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal stdout: %v\nstdout: %s", err, stdout.String())
	}
	if !out.Found || len(out.Issues) != 1 || out.Issues[0] != 12 {
		t.Errorf("got found=%v issues=%v, want found=true issues=[12]", out.Found, out.Issues)
	}
}

func TestPRBaseJSONFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"pr", "base", "feature/PLAN-00042/phase-3", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("pr base --json returned error: %v\nstderr: %s", err, stderr.String())
	}

	var out struct {
		Base string `json:"base"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal stdout: %v\nstdout: %s", err, stdout.String())
	}
	if out.Base != "feature/PLAN-00042/main" {
		t.Errorf("base = %q, want %q", out.Base, "feature/PLAN-00042/main")
	}
}

func TestPRFindJSONFlag(t *testing.T) {
	withConfigFixture(t)

	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`[{"number":68,"title":"[PLAN-00042-3] Some phase","url":"https://github.com/threehillpath/claude-plan-workflow/pull/68","headRefName":"feature/PLAN-00042/phase-3","baseRefName":"feature/PLAN-00042/main","state":"OPEN"}]`)})

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	root.SetArgs([]string{"pr", "find", "[PLAN-00042-3]", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("pr find --json returned error: %v\nstderr: %s", err, stderr.String())
	}

	var out struct {
		Found bool   `json:"found"`
		Head  string `json:"head"`
		Base  string `json:"base"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal stdout: %v\nstdout: %s", err, stdout.String())
	}
	if !out.Found || out.Head != "feature/PLAN-00042/phase-3" || out.Base != "feature/PLAN-00042/main" {
		t.Errorf("got found=%v head=%q base=%q, want found=true head=feature/PLAN-00042/phase-3 base=feature/PLAN-00042/main", out.Found, out.Head, out.Base)
	}
}

func TestIssueListJSONFlag(t *testing.T) {
	withConfigFixture(t)

	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`[{"number":59,"title":"[PLAN-00041-2] Phase 2","state":"OPEN","labels":[{"name":"plan:phase"}]}]`)})

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	root.SetArgs([]string{"issue", "list", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("issue list --json returned error: %v\nstderr: %s", err, stderr.String())
	}

	var out []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal stdout: %v\nstdout: %s", err, stdout.String())
	}
	if len(out) != 1 || out[0].Number != 59 {
		t.Errorf("got %v, want 1 item with number=59", out)
	}
}

// withConfigFixture chdirs the test into a temp directory containing a
// minimal .claude/plan-workflow-config.yml (reusing boardJSONConfigFixture
// from board_json_test.go) and restores the original cwd on cleanup.
func withConfigFixture(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "plan-workflow-config.yml"), []byte(boardJSONConfigFixture), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
}
