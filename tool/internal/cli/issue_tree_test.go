package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"threehillpath.com/claude-plan-workflow/tool/internal/cli"
	"threehillpath.com/claude-plan-workflow/tool/internal/exectest"
)

// TestIssueTreePlainText verifies the default (non-JSON) rendering of `issue
// tree`: one pipe-delimited line per node, "<kind> | #<number> | <state> |
// <status> | <title>", title last. This exercises the CLI-level rendering
// path; the underlying hierarchy-walk logic is covered by
// tool/internal/issue/tree_test.go.
func TestIssueTreePlainText(t *testing.T) {
	withConfigFixture(t)

	fake := &exectest.FakeRunner{}
	implTitle := "[PLAN-00041] Marvin JSON overhaul"
	// 1) IssueRef(target=57)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"id":"ID_57","number":57,"title":"` + implTitle + `","state":"OPEN"}`)})
	// 2) ParentIssue(57): internal IssueRef(57)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"id":"ID_57","number":57,"title":"` + implTitle + `","state":"OPEN"}`)})
	// 2) ParentIssue(57): graphql parent query -> no parent
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"data":{"node":{"parent":null}}}`)})
	// 3) SubIssues(archRoot=57): internal IssueRef(57)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"id":"ID_57","number":57,"title":"` + implTitle + `","state":"OPEN"}`)})
	// 3) SubIssues(57): graphql subIssues query -> one phase
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"data":{"node":{"subIssues":{"nodes":[{"number":62,"title":"[PLAN-00041-5] Phase 5","state":"OPEN"}]}}}}`)})
	// 5) board.List
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"items":[{"id":"PVTI_1","title":"Phase 5","status":"In Progress","content":{"number":62,"title":"Phase 5","url":"https://github.com/threehillpath/claude-plan-workflow/issues/62"}}]}`)})

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	root.SetArgs([]string{"issue", "tree", "57"})

	if err := root.Execute(); err != nil {
		t.Fatalf("issue tree returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := "impl | #57 | OPEN |  | " + implTitle + "\n" +
		"phase | #62 | OPEN | In Progress | [PLAN-00041-5] Phase 5\n"
	if stdout.String() != want {
		t.Errorf("plain-text output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

// TestIssueTreeJSON verifies --json emits []{kind, number, title, state, status}.
func TestIssueTreeJSON(t *testing.T) {
	withConfigFixture(t)

	fake := &exectest.FakeRunner{}
	title := "[PLAN-00050] Solo plan"
	// 1) IssueRef(target=50)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"id":"ID_50","number":50,"title":"` + title + `","state":"OPEN"}`)})
	// 2) ParentIssue(50): internal IssueRef(50)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"id":"ID_50","number":50,"title":"` + title + `","state":"OPEN"}`)})
	// 2) ParentIssue(50): graphql parent query -> no parent
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"data":{"node":{"parent":null}}}`)})
	// 3) SubIssues(archRoot=50): internal IssueRef(50)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"id":"ID_50","number":50,"title":"` + title + `","state":"OPEN"}`)})
	// 3) SubIssues(50): graphql subIssues query -> no children
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"data":{"node":{"subIssues":{"nodes":[]}}}}`)})
	// 5) board.List
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"items":[]}`)})

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	root.SetArgs([]string{"issue", "tree", "50", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("issue tree --json returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := `[
  {
    "kind": "impl",
    "number": 50,
    "title": "[PLAN-00050] Solo plan",
    "state": "OPEN",
    "status": ""
  }
]
`
	if stdout.String() != want {
		t.Errorf("--json output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

// TestIssueTreeNotAPlanIssue verifies output is empty when the target's own
// title fails to parse via parse.PlanIdent.
func TestIssueTreeNotAPlanIssue(t *testing.T) {
	withConfigFixture(t)

	fake := &exectest.FakeRunner{}
	// 1) IssueRef(target=41) -- not a plan issue
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"id":"ID_41","number":41,"title":"Some source issue","state":"OPEN"}`)})

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	root.SetArgs([]string{"issue", "tree", "41"})

	if err := root.Execute(); err != nil {
		t.Fatalf("issue tree returned error: %v\nstderr: %s", err, stderr.String())
	}

	if stdout.String() != "" {
		t.Errorf("expected empty output, got:\n%s", stdout.String())
	}
}
