package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"threehillpath.com/claude-plan-workflow/tool/internal/cli"
	"threehillpath.com/claude-plan-workflow/tool/internal/exectest"
)

// TestIssueListPlainText verifies the default (non-JSON) output is
// pipe-delimited with no header row:
// <number> | <state> | <labels-comma-joined> | <title>, title last.
func TestIssueListPlainText(t *testing.T) {
	withConfigFixture(t)

	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`[{"number":59,"title":"[PLAN-00041-2] Phase 2","state":"OPEN","labels":[{"name":"plan:phase"},{"name":"status:in-progress"}]},{"number":60,"title":"[PLAN-00041-3] Phase 3","state":"CLOSED","labels":[]}]`)})

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	root.SetArgs([]string{"issue", "list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("issue list returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := "59 | OPEN | plan:phase,status:in-progress | [PLAN-00041-2] Phase 2\n" +
		"60 | CLOSED |  | [PLAN-00041-3] Phase 3\n"
	if stdout.String() != want {
		t.Errorf("plain-text output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

// TestIssueListPlainTextEmpty verifies an empty result prints nothing.
func TestIssueListPlainTextEmpty(t *testing.T) {
	withConfigFixture(t)

	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`[]`)})

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	root.SetArgs([]string{"issue", "list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("issue list returned error: %v\nstderr: %s", err, stderr.String())
	}

	if stdout.String() != "" {
		t.Errorf("expected empty output, got:\n%s", stdout.String())
	}
}

// TestIssueListJSONFidelity verifies --json is byte-identical to the
// pre-Component-5 JSON shape.
func TestIssueListJSONFidelity(t *testing.T) {
	withConfigFixture(t)

	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`[{"number":59,"title":"[PLAN-00041-2] Phase 2","state":"OPEN","labels":[{"name":"plan:phase"}]}]`)})

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	root.SetArgs([]string{"issue", "list", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("issue list --json returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := `[
  {
    "number": 59,
    "title": "[PLAN-00041-2] Phase 2",
    "state": "OPEN",
    "labels": [
      "plan:phase"
    ]
  }
]
`
	if stdout.String() != want {
		t.Errorf("--json output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}
