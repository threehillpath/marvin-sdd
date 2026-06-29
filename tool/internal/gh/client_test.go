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

// TestProjectItemListArgs verifies the gh arg vector and JSON parsing.
func TestProjectItemListArgs(t *testing.T) {
	fake := &exectest.FakeRunner{}
	payload := []byte(`{"items":[{"id":"PVTI_1","title":"Phase 1","status":"In Progress","content":{"number":10,"title":"Phase 1","url":"https://github.com/o/r/issues/10"}}]}`)
	fake.Enqueue(exectest.FakeResponse{Stdout: payload})

	items, err := gh.New(fake).ProjectItemList(context.Background(), 4, "owner", 50)
	if err != nil {
		t.Fatalf("ProjectItemList returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Status != "In Progress" {
		t.Errorf("Status = %q, want In Progress", items[0].Status)
	}
	if items[0].Content.Number != 10 {
		t.Errorf("Content.Number = %d, want 10", items[0].Content.Number)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(fake.Calls))
	}
	args := fake.Calls[0].Args
	wantArgs := []string{"project", "item-list", "4", "--owner", "owner", "--format", "json", "--limit", "50"}
	if len(args) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", args, wantArgs)
	}
	for i, a := range args {
		if a != wantArgs[i] {
			t.Errorf("arg[%d] = %q, want %q", i, a, wantArgs[i])
		}
	}
}

// TestProjectItemListDefaultLimit verifies that limit=0 sends --limit 100.
func TestProjectItemListDefaultLimit(t *testing.T) {
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"items":[]}`)})

	if _, err := gh.New(fake).ProjectItemList(context.Background(), 4, "owner", 0); err != nil {
		t.Fatalf("ProjectItemList error: %v", err)
	}
	args := fake.Calls[0].Args
	for i, a := range args {
		if a == "--limit" && i+1 < len(args) && args[i+1] != "100" {
			t.Errorf("expected --limit 100, got --limit %s", args[i+1])
		}
	}
}

// TestIssueListArgs verifies the gh arg vector and JSON parsing for IssueList.
func TestIssueListArgs(t *testing.T) {
	fake := &exectest.FakeRunner{}
	payload := []byte(`[{"number":5,"title":"Fix bug","state":"OPEN","labels":[{"name":"plan:phase"}]}]`)
	fake.Enqueue(exectest.FakeResponse{Stdout: payload})

	items, err := gh.New(fake).IssueList(context.Background(), "owner/repo", "plan:phase", "open", 25)
	if err != nil {
		t.Fatalf("IssueList returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Number != 5 {
		t.Errorf("Number = %d, want 5", items[0].Number)
	}
	if len(items[0].Labels) != 1 || items[0].Labels[0].Name != "plan:phase" {
		t.Errorf("Labels = %v, want [{plan:phase}]", items[0].Labels)
	}

	args := fake.Calls[0].Args
	wantArgs := []string{"issue", "list", "--repo", "owner/repo", "--state", "open", "--json", "number,title,state,labels", "--limit", "25", "--label", "plan:phase"}
	if len(args) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", args, wantArgs)
	}
	for i, a := range args {
		if a != wantArgs[i] {
			t.Errorf("arg[%d] = %q, want %q", i, a, wantArgs[i])
		}
	}
}

// TestIssueListNoLabelSkipsFlag verifies that an empty label omits --label.
func TestIssueListNoLabelSkipsFlag(t *testing.T) {
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`[]`)})

	if _, err := gh.New(fake).IssueList(context.Background(), "owner/repo", "", "open", 10); err != nil {
		t.Fatalf("IssueList error: %v", err)
	}
	for _, a := range fake.Calls[0].Args {
		if a == "--label" {
			t.Error("expected no --label flag when label is empty")
		}
	}
}
