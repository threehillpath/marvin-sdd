package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"threehillpath.com/claude-plan-workflow/tool/internal/cli"
	"threehillpath.com/claude-plan-workflow/tool/internal/exectest"
)

// TestIssueLinkParentCallSequence is TDD Entry Point 3: `marvin issue
// link-parent 58 57` issues exactly two `gh issue view --json
// id,number,title,state` calls (child #58 then parent #57) followed by one
// `gh api graphql` call (the addSubIssue mutation), exits 0, and writes
// nothing to stdout — matching board move / label ensure's silent
// convention.
func TestIssueLinkParentCallSequence(t *testing.T) {
	withConfigFixture(t)

	fake := &exectest.FakeRunner{}
	// 1) IssueRef(child=58)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"id":"ID_58","number":58,"title":"[PLAN-00041-5] Phase 5","state":"OPEN"}`)})
	// 2) IssueRef(parent=57)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"id":"ID_57","number":57,"title":"[PLAN-00041] Impl plan","state":"OPEN"}`)})
	// 3) gh api graphql (addSubIssue mutation)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"data":{"addSubIssue":{"subIssue":{"id":"ID_58"}}}}`)})

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	root.SetArgs([]string{"issue", "link-parent", "58", "57"})

	if err := root.Execute(); err != nil {
		t.Fatalf("issue link-parent returned error: %v\nstderr: %s", err, stderr.String())
	}

	if len(fake.Calls) != 3 {
		t.Fatalf("expected 3 calls, got %d: %v", len(fake.Calls), fake.Calls)
	}

	// Calls 0 and 1: gh issue view for child (58) then parent (57).
	if fake.Calls[0].Name != "gh" || !containsArg(fake.Calls[0].Args, "58") || !containsArg(fake.Calls[0].Args, "view") {
		t.Errorf("call[0] = %+v, want gh issue view 58", fake.Calls[0])
	}
	if fake.Calls[1].Name != "gh" || !containsArg(fake.Calls[1].Args, "57") || !containsArg(fake.Calls[1].Args, "view") {
		t.Errorf("call[1] = %+v, want gh issue view 57", fake.Calls[1])
	}
	// Call 2: gh api graphql.
	if fake.Calls[2].Name != "gh" || !containsArg(fake.Calls[2].Args, "graphql") {
		t.Errorf("call[2] = %+v, want gh api graphql", fake.Calls[2])
	}

	if stdout.String() != "" {
		t.Errorf("expected no stdout, got: %q", stdout.String())
	}
}

func containsArg(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}
