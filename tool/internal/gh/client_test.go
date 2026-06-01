package gh_test

import (
	"context"
	"testing"

	"threehillpath.com/claude-plan-workflow/tool/internal/exectest"
	"threehillpath.com/claude-plan-workflow/tool/internal/gh"
)

// TestIssueJSONArgs verifies that IssueJSON builds the correct gh arg vector
// and returns the runner's stdout bytes unchanged.
func TestIssueJSONArgs(t *testing.T) {
	fake := &exectest.FakeRunner{}
	payload := []byte(`{"title":"Phase 2","body":"..."}`)
	fake.Enqueue(exectest.FakeResponse{Stdout: payload})

	got, err := gh.New(fake).IssueJSON(context.Background(), 42, "title", "body")
	if err != nil {
		t.Fatalf("IssueJSON returned error: %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("IssueJSON returned %q, want %q", got, payload)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(fake.Calls))
	}
	want := []string{"issue", "view", "42", "--json", "title,body"}
	args := fake.Calls[0].Args
	if len(args) != len(want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
	for i, a := range args {
		if a != want[i] {
			t.Errorf("arg[%d] = %q, want %q", i, a, want[i])
		}
	}
}

// TestIssueJSONNonZeroExitReturnsError verifies that a non-zero exit code
// from gh is surfaced as an error.
func TestIssueJSONNonZeroExitReturnsError(t *testing.T) {
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stderr: []byte("issue not found"), ExitCode: 1})

	_, err := gh.New(fake).IssueJSON(context.Background(), 99, "title")
	if err == nil {
		t.Fatal("expected error for non-zero exit, got nil")
	}
}
