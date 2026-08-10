package gh_test

import (
	"context"
	"strings"
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

// TestGraphQLPartialFailureExtractsErrorsFromStdout verifies that, on a non-zero
// exit whose stdout carries a {data, errors} GraphQL envelope, GraphQL builds its
// error from errors[].message rather than the stderr summary.
func TestGraphQLPartialFailureExtractsErrorsFromStdout(t *testing.T) {
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{
		Stdout:   []byte(`{"data":{"a":{"name":"x"},"b":null},"errors":[{"message":"boom"}]}`),
		Stderr:   []byte("gh: boom"),
		ExitCode: 1,
	})

	_, err := gh.New(fake).GraphQL(context.Background(), "query { a { name } b { name } }", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("err = %q, want it to contain %q", err.Error(), "boom")
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(fake.Calls))
	}
	wantArgs := []string{"api", "graphql", "-f", "query=query { a { name } b { name } }"}
	args := fake.Calls[0].Args
	if len(args) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", args, wantArgs)
	}
	for i, a := range args {
		if a != wantArgs[i] {
			t.Errorf("arg[%d] = %q, want %q", i, a, wantArgs[i])
		}
	}
}

// TestIssueRefReturnsRefAndNodeID verifies that IssueRef wraps
// `gh issue view <number> --repo <repo> --json id,number,title,state` and
// returns both the GraphQL node ID and the ref (number/title/state).
func TestIssueRefReturnsRefAndNodeID(t *testing.T) {
	fake := &exectest.FakeRunner{}
	payload := []byte(`{"id":"I_kwDOAbc123","number":58,"title":"[PLAN-00041-1] GraphQL foundation","state":"OPEN"}`)
	fake.Enqueue(exectest.FakeResponse{Stdout: payload})

	ref, nodeID, err := gh.New(fake).IssueRef(context.Background(), "owner/repo", 58)
	if err != nil {
		t.Fatalf("IssueRef returned error: %v", err)
	}
	if nodeID != "I_kwDOAbc123" {
		t.Errorf("nodeID = %q, want %q", nodeID, "I_kwDOAbc123")
	}
	if ref.Number != 58 || ref.Title != "[PLAN-00041-1] GraphQL foundation" || ref.State != "OPEN" {
		t.Errorf("ref = %+v, want Number=58 Title=%q State=OPEN", ref, "[PLAN-00041-1] GraphQL foundation")
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(fake.Calls))
	}
	wantArgs := []string{"issue", "view", "58", "--repo", "owner/repo", "--json", "id,number,title,state"}
	args := fake.Calls[0].Args
	if len(args) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", args, wantArgs)
	}
	for i, a := range args {
		if a != wantArgs[i] {
			t.Errorf("arg[%d] = %q, want %q", i, a, wantArgs[i])
		}
	}
}

// TestAddSubIssueCallsGraphQLMutation verifies that AddSubIssue calls
// GraphQL with the addSubIssue mutation, passing the given parent/child node
// IDs through as GraphQL variables.
func TestAddSubIssueCallsGraphQLMutation(t *testing.T) {
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"data":{"addSubIssue":{"subIssue":{"id":"I_child"}}}}`), ExitCode: 0})

	err := gh.New(fake).AddSubIssue(context.Background(), "I_parent", "I_child")
	if err != nil {
		t.Fatalf("AddSubIssue returned error: %v", err)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(fake.Calls))
	}
	args := fake.Calls[0].Args
	if args[0] != "api" || args[1] != "graphql" {
		t.Fatalf("args = %v, want to start with [api graphql]", args)
	}
	if !argsContainFlagValue(args, "parentId=I_parent") {
		t.Errorf("args = %v, want a -f flag containing parentId=I_parent", args)
	}
	if !argsContainFlagValue(args, "childId=I_child") {
		t.Errorf("args = %v, want a -f flag containing childId=I_child", args)
	}
}

// argsContainFlagValue reports whether args contains want as a standalone
// element (used to check for a specific -f <key>=<value> pair regardless of
// its position among other -f flags).
func argsContainFlagValue(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}

// TestSubIssuesReturnsPhasesRegardlessOfTitlePrefix verifies that SubIssues
// resolves the target's node ID via IssueRef, then queries its subIssues
// field and returns the results — including phase titles that are not
// true string-prefixes of the parent's title (the PLAN-00036 regression case
// that motivates this whole component).
func TestSubIssuesReturnsPhasesRegardlessOfTitlePrefix(t *testing.T) {
	fake := &exectest.FakeRunner{}
	// Call 1: IssueRef(target) — resolves the impl plan's node ID.
	fake.Enqueue(exectest.FakeResponse{
		Stdout: []byte(`{"id":"I_impl36","number":36,"title":"[PLAN-00036] Something","state":"OPEN"}`),
	})
	// Call 2: GraphQL subIssues query.
	fake.Enqueue(exectest.FakeResponse{
		Stdout: []byte(`{"data":{"node":{"subIssues":{"nodes":[` +
			`{"number":39,"title":"Domain Model","state":"OPEN"},` +
			`{"number":40,"title":"API Layer","state":"CLOSED"}` +
			`]}}}}`),
	})

	items, err := gh.New(fake).SubIssues(context.Background(), "owner/repo", 36)
	if err != nil {
		t.Fatalf("SubIssues returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 sub-issues, got %d: %+v", len(items), items)
	}
	if items[0].Number != 39 || items[0].Title != "Domain Model" || items[0].State != "OPEN" {
		t.Errorf("items[0] = %+v, want {39 Domain Model OPEN}", items[0])
	}
	if items[1].Number != 40 || items[1].Title != "API Layer" || items[1].State != "CLOSED" {
		t.Errorf("items[1] = %+v, want {40 API Layer CLOSED}", items[1])
	}

	if len(fake.Calls) != 2 {
		t.Fatalf("expected 2 calls, got %d: %+v", len(fake.Calls), fake.Calls)
	}
	if fake.Calls[0].Args[0] != "issue" || fake.Calls[0].Args[1] != "view" {
		t.Errorf("call 0 = %v, want IssueRef (issue view ...)", fake.Calls[0].Args)
	}
	if fake.Calls[1].Args[0] != "api" || fake.Calls[1].Args[1] != "graphql" {
		t.Errorf("call 1 = %v, want GraphQL (api graphql ...)", fake.Calls[1].Args)
	}
	if !argsContainFlagValue(fake.Calls[1].Args, "id=I_impl36") {
		t.Errorf("call 1 args = %v, want a -f flag containing id=I_impl36", fake.Calls[1].Args)
	}
}

// TestParentIssueFound verifies that ParentIssue resolves the target's node
// ID via IssueRef, queries its parent field, and returns (ref, true, nil)
// when a parent exists.
func TestParentIssueFound(t *testing.T) {
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{
		Stdout: []byte(`{"id":"I_phase39","number":39,"title":"[PLAN-00036-1] Domain Model","state":"OPEN"}`),
	})
	fake.Enqueue(exectest.FakeResponse{
		Stdout: []byte(`{"data":{"node":{"parent":{"number":36,"title":"[PLAN-00036] Something","state":"OPEN"}}}}`),
	})

	ref, ok, err := gh.New(fake).ParentIssue(context.Background(), "owner/repo", 39)
	if err != nil {
		t.Fatalf("ParentIssue returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true, got false")
	}
	if ref.Number != 36 || ref.Title != "[PLAN-00036] Something" || ref.State != "OPEN" {
		t.Errorf("ref = %+v, want {36 [PLAN-00036] Something OPEN}", ref)
	}
}

// TestParentIssueNone verifies that ParentIssue returns ok=false with no
// error when the target has no parent (a nil parent field on the response).
func TestParentIssueNone(t *testing.T) {
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{
		Stdout: []byte(`{"id":"I_arch56","number":56,"title":"[PLAN-00041-ARCH] Something","state":"OPEN"}`),
	})
	fake.Enqueue(exectest.FakeResponse{
		Stdout: []byte(`{"data":{"node":{"parent":null}}}`),
	})

	ref, ok, err := gh.New(fake).ParentIssue(context.Background(), "owner/repo", 56)
	if err != nil {
		t.Fatalf("ParentIssue returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false, got true with ref=%+v", ref)
	}
}

// TestGraphQLNonGraphQLFailureFallsBackToStderr verifies that a non-zero exit
// whose stdout is not a {data, errors} envelope (e.g. an auth failure) still
// produces the stderr-based error message, unchanged from the existing idiom.
func TestGraphQLNonGraphQLFailureFallsBackToStderr(t *testing.T) {
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{
		Stdout:   []byte(""),
		Stderr:   []byte("gh: not authenticated"),
		ExitCode: 1,
	})

	_, err := gh.New(fake).GraphQL(context.Background(), "query { viewer { login } }", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not authenticated") {
		t.Errorf("err = %q, want it to contain %q", err.Error(), "not authenticated")
	}

	wantAuthArgs := []string{"api", "graphql", "-f", "query=query { viewer { login } }"}
	authArgs := fake.Calls[0].Args
	if len(authArgs) != len(wantAuthArgs) {
		t.Fatalf("args = %v, want %v", authArgs, wantAuthArgs)
	}
	for i, a := range authArgs {
		if a != wantAuthArgs[i] {
			t.Errorf("arg[%d] = %q, want %q", i, a, wantAuthArgs[i])
		}
	}
}

// TestGraphQLSuccessReturnsStdoutAndSortsVars verifies that on exit 0, GraphQL
// returns stdout unchanged (no envelope check) and appends -f <key>=<value>
// pairs for each vars entry in sorted key order.
func TestGraphQLSuccessReturnsStdoutAndSortsVars(t *testing.T) {
	fake := &exectest.FakeRunner{}
	payload := []byte(`{"data":{"node":{"id":"I_1"}}}`)
	fake.Enqueue(exectest.FakeResponse{Stdout: payload, ExitCode: 0})

	got, err := gh.New(fake).GraphQL(context.Background(), "query($id: ID!) { node(id: $id) { id } }", map[string]string{
		"zeta":  "z",
		"alpha": "a",
	})
	if err != nil {
		t.Fatalf("GraphQL returned error: %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("GraphQL returned %q, want %q", got, payload)
	}

	wantArgs := []string{
		"api", "graphql",
		"-f", "query=query($id: ID!) { node(id: $id) { id } }",
		"-f", "alpha=a",
		"-f", "zeta=z",
	}
	args := fake.Calls[0].Args
	if len(args) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", args, wantArgs)
	}
	for i, a := range args {
		if a != wantArgs[i] {
			t.Errorf("arg[%d] = %q, want %q", i, a, wantArgs[i])
		}
	}
}
