package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"threehillpath.com/marvin-sdd/tool/internal/cli"
	"threehillpath.com/marvin-sdd/tool/internal/exectest"
)

// runIssue runs marvin with args against fake and returns stdout, stderr and
// the error. The caller supplies the fake so it can inspect fake.Calls.
func runIssue(fake *exectest.FakeRunner, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	root.SetArgs(args)
	err := root.Execute()
	return stdout.String(), stderr.String(), err
}

// TestIssueCreateBadDraftExits3WithoutCalls is the phase entry point: a draft
// that fails the schema check exits 3 and makes no gh call at all.
func TestIssueCreateBadDraftExits3WithoutCalls(t *testing.T) {
	withConfigFixture(t)
	draft := writeTemp(t, "d.yml", phaseDraftNoVerification)
	fake := &exectest.FakeRunner{}

	stdout, stderr, err := runIssue(fake, "issue", "create", "--template", "impl-phase", "--draft", draft, "--label", "x")

	wantCode(t, err, 3)
	if len(fake.Calls) != 0 {
		t.Errorf("want zero gh calls, got %v", fake.Calls)
	}
	if stdout != "" {
		t.Errorf("want empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "error section:verification") {
		t.Errorf("want the verification finding on stderr:\n%s", stderr)
	}
}

// TestIssueCreateConformingDraftCreatesOnce verifies a conforming draft makes
// exactly one gh issue create call whose --title is the draft's title and
// whose --body is the rendered markdown, with number then URL on stdout and
// the schema line on stderr.
func TestIssueCreateConformingDraftCreatesOnce(t *testing.T) {
	withConfigFixture(t)
	draft := writeTemp(t, "d.yml", phaseDraftOK)
	rendered, _, err := runCLI(t, "template", "render", "impl-phase", "--draft", draft)
	if err != nil {
		t.Fatal(err)
	}
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("https://github.com/threehillpath/marvin-sdd/issues/55\n")})

	stdout, stderr, err := runIssue(fake, "issue", "create", "--template", "impl-phase", "--draft", draft, "--label", "plan:phase")
	if err != nil {
		t.Fatalf("create returned %v\nstderr: %s", err, stderr)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("want 1 gh call, got %v", fake.Calls)
	}
	want := []string{"issue", "create", "--repo", "threehillpath/marvin-sdd", "--title", "[PLAN-00112-5] Phase title", "--body", rendered, "--label", "plan:phase"}
	if got := fake.Calls[0].Args; strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Errorf("args = %q\nwant   %q", got, want)
	}
	if stdout != "55\nhttps://github.com/threehillpath/marvin-sdd/issues/55\n" {
		t.Errorf("stdout = %q", stdout)
	}
	if !strings.HasPrefix(stderr, "schema: impl-phase (built-in)\n") {
		t.Errorf("want the schema line on stderr, got %q", stderr)
	}
}

// TestIssueEditDraftEditsBodyAndTitle verifies a conforming draft makes
// exactly one gh issue edit call with the rendered body and the draft title,
// and prints nothing on stdout.
func TestIssueEditDraftEditsBodyAndTitle(t *testing.T) {
	withConfigFixture(t)
	draft := writeTemp(t, "d.yml", phaseDraftOK)
	rendered, _, err := runCLI(t, "template", "render", "impl-phase", "--draft", draft)
	if err != nil {
		t.Fatal(err)
	}
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("https://github.com/threehillpath/marvin-sdd/issues/7\n")})

	stdout, stderr, err := runIssue(fake, "issue", "edit", "7", "--template", "impl-phase", "--draft", draft)
	if err != nil {
		t.Fatalf("edit returned %v\nstderr: %s", err, stderr)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("want 1 gh call, got %v", fake.Calls)
	}
	want := []string{"issue", "edit", "7", "--repo", "threehillpath/marvin-sdd", "--body", rendered, "--title", "[PLAN-00112-5] Phase title"}
	if got := fake.Calls[0].Args; strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Errorf("args = %q\nwant   %q", got, want)
	}
	if stdout != "" {
		t.Errorf("want empty stdout, got %q", stdout)
	}
	if !strings.HasPrefix(stderr, "schema: impl-phase (built-in)\n") {
		t.Errorf("want the schema line on stderr, got %q", stderr)
	}
}
