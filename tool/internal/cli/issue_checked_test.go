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

// TestIssueEditBodyFileKeepsTitle verifies --body-file reads the existing
// title with gh issue view --repo, then makes exactly one gh issue edit with
// the unchanged body and no --title.
func TestIssueEditBodyFileKeepsTitle(t *testing.T) {
	withConfigFixture(t)
	bodyPath := writeTemp(t, "b.md", conformingPhaseBody)
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"id":"I_1","number":7,"title":"[PLAN-00112-5] Phase title","state":"OPEN"}`)})
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("https://github.com/threehillpath/marvin-sdd/issues/7\n")})

	stdout, stderr, err := runIssue(fake, "issue", "edit", "7", "--template", "impl-phase", "--body-file", bodyPath)
	if err != nil {
		t.Fatalf("edit returned %v\nstderr: %s", err, stderr)
	}

	if len(fake.Calls) != 2 {
		t.Fatalf("want 2 gh calls (view then edit), got %v", fake.Calls)
	}
	view := fake.Calls[0].Args
	if len(view) < 5 || strings.Join(view[:5], " ") != "issue view 7 --repo threehillpath/marvin-sdd" {
		t.Errorf("first call = %q, want gh issue view 7 --repo threehillpath/marvin-sdd ...", view)
	}
	want := []string{"issue", "edit", "7", "--repo", "threehillpath/marvin-sdd", "--body", conformingPhaseBody}
	if got := fake.Calls[1].Args; strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Errorf("edit args = %q\nwant       %q", got, want)
	}
	if stdout != "" {
		t.Errorf("want empty stdout, got %q", stdout)
	}
}

// TestIssueEditBadDraftExits3WithoutCalls verifies a failing check on edit
// exits 3 with zero gh calls and empty stdout.
func TestIssueEditBadDraftExits3WithoutCalls(t *testing.T) {
	withConfigFixture(t)
	draft := writeTemp(t, "d.yml", phaseDraftNoVerification)
	fake := &exectest.FakeRunner{}

	stdout, stderr, err := runIssue(fake, "issue", "edit", "7", "--template", "impl-phase", "--draft", draft)

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

// TestIssueEditBadBodyFileMakesNoMutatingCall verifies a non-conforming body
// file exits 3 after only the read call.
func TestIssueEditBadBodyFileMakesNoMutatingCall(t *testing.T) {
	withConfigFixture(t)
	bad := strings.Replace(conformingPhaseBody, "## Verification\n\nV.\n\n", "", 1)
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"id":"I_1","number":7,"title":"[PLAN-00112-5] Phase title","state":"OPEN"}`)})

	_, _, err := runIssue(fake, "issue", "edit", "7", "--template", "impl-phase", "--body-file", writeTemp(t, "b.md", bad))

	wantCode(t, err, 3)
	if len(fake.Calls) != 1 || fake.Calls[0].Args[1] != "view" {
		t.Errorf("want only the gh issue view call, got %v", fake.Calls)
	}
}

// TestIssueEditUsageErrorsMakeNoCalls verifies the exit-1 usage errors on
// edit, none of which may touch gh.
func TestIssueEditUsageErrorsMakeNoCalls(t *testing.T) {
	withConfigFixture(t)
	draft := writeTemp(t, "d.yml", phaseDraftOK)
	body := writeTemp(t, "b.md", conformingPhaseBody)
	for name, args := range map[string][]string{
		"no template":      {"issue", "edit", "7", "--draft", draft},
		"no input":         {"issue", "edit", "7", "--template", "impl-phase"},
		"both inputs":      {"issue", "edit", "7", "--template", "impl-phase", "--draft", draft, "--body-file", body},
		"unknown type":     {"issue", "edit", "7", "--template", "nope", "--draft", draft},
		"inline body":      {"issue", "edit", "7", "--template", "impl-phase", "--body", "x"},
		"non-numeric item": {"issue", "edit", "seven", "--template", "impl-phase", "--draft", draft},
	} {
		t.Run(name, func(t *testing.T) {
			fake := &exectest.FakeRunner{}
			_, _, err := runIssue(fake, args...)
			if err == nil {
				t.Fatal("want an error")
			}
			if ce, ok := err.(*cli.CLIError); ok && ce.Code != 1 {
				t.Errorf("code = %d, want 1", ce.Code)
			}
			if len(fake.Calls) != 0 {
				t.Errorf("want zero gh calls, got %v", fake.Calls)
			}
		})
	}
}

// TestIssueCreateTemplateUsageErrorsMakeNoCalls verifies the exit-1 usage
// errors on create --template.
func TestIssueCreateTemplateUsageErrorsMakeNoCalls(t *testing.T) {
	withConfigFixture(t)
	draft := writeTemp(t, "d.yml", phaseDraftOK)
	body := writeTemp(t, "b.md", conformingPhaseBody)
	for name, tc := range map[string]struct {
		args []string
		want string
	}{
		"inline body":            {[]string{"issue", "create", "--template", "impl-phase", "--title", "[PLAN-00112-1] X", "--body", "x"}, "--draft"},
		"draft without template": {[]string{"issue", "create", "--draft", draft}, "--template"},
		"body-file needs title":  {[]string{"issue", "create", "--template", "impl-phase", "--body-file", body}, "--title"},
		"title differs":          {[]string{"issue", "create", "--template", "impl-phase", "--draft", draft, "--title", "Something else"}, "Phase title"},
		"empty title differs":    {[]string{"issue", "create", "--template", "impl-phase", "--draft", draft, "--title", ""}, "Phase title"},
		"draft and body-file":    {[]string{"issue", "create", "--template", "impl-phase", "--draft", draft, "--body-file", body}, "exactly one"},
	} {
		t.Run(name, func(t *testing.T) {
			fake := &exectest.FakeRunner{}
			stdout, _, err := runIssue(fake, tc.args...)
			ce := wantCode(t, err, 1)
			if !strings.Contains(ce.Msg, tc.want) {
				t.Errorf("message %q should mention %q", ce.Msg, tc.want)
			}
			if len(fake.Calls) != 0 || stdout != "" {
				t.Errorf("want zero gh calls and empty stdout, got %v / %q", fake.Calls, stdout)
			}
		})
	}
}

// TestIssueCreateTitleMatchingDraftIsAccepted verifies --title equal to the
// draft's title is accepted, and a body-file create sends the file unchanged
// under --title.
func TestIssueCreateTitleMatchingDraftIsAccepted(t *testing.T) {
	withConfigFixture(t)
	draft := writeTemp(t, "d.yml", phaseDraftOK)
	body := writeTemp(t, "b.md", conformingPhaseBody)
	for name, args := range map[string][]string{
		"draft with equal title": {"issue", "create", "--template", "impl-phase", "--draft", draft, "--title", "[PLAN-00112-5] Phase title"},
		"body-file with title":   {"issue", "create", "--template", "impl-phase", "--body-file", body, "--title", "[PLAN-00112-5] Phase title"},
	} {
		t.Run(name, func(t *testing.T) {
			fake := &exectest.FakeRunner{}
			fake.Enqueue(exectest.FakeResponse{Stdout: []byte("https://github.com/threehillpath/marvin-sdd/issues/56\n")})
			_, stderr, err := runIssue(fake, args...)
			if err != nil {
				t.Fatalf("create returned %v\nstderr: %s", err, stderr)
			}
			if len(fake.Calls) != 1 {
				t.Fatalf("want 1 gh call, got %v", fake.Calls)
			}
		})
	}
}

// TestIssueCreateBadBodyFileExits3WithoutCalls verifies the body-file path
// also stops on a failing check.
func TestIssueCreateBadBodyFileExits3WithoutCalls(t *testing.T) {
	withConfigFixture(t)
	bad := strings.Replace(conformingPhaseBody, "## Verification\n\nV.\n\n", "", 1)
	fake := &exectest.FakeRunner{}
	_, _, err := runIssue(fake, "issue", "create", "--template", "impl-phase", "--body-file", writeTemp(t, "b.md", bad), "--title", "[PLAN-00112-5] Phase title")
	wantCode(t, err, 3)
	if len(fake.Calls) != 0 {
		t.Errorf("want zero gh calls, got %v", fake.Calls)
	}
}

// TestExplicitEmptyFlagsAreErrors verifies a flag that is passed with an
// empty value is an exit-1 usage error naming the flag, never a fallback to
// an unchecked or "neither given" path, and that no gh call is made.
func TestExplicitEmptyFlagsAreErrors(t *testing.T) {
	withConfigFixture(t)
	draft := writeTemp(t, "d.yml", phaseDraftOK)
	body := writeTemp(t, "b.md", conformingPhaseBody)
	const types = "arch-plan, impl-phase, impl-plan, quick-task"
	for name, tc := range map[string]struct {
		args []string
		want string
	}{
		"create empty template, body-file": {[]string{"issue", "create", "--template", "", "--title", "T", "--body-file", body}, "--template needs one of " + types + "; it was given an empty value"},
		"create empty template, draft":     {[]string{"issue", "create", "--template", "", "--draft", draft}, "--template needs one of " + types + "; it was given an empty value"},
		"create unknown template":          {[]string{"issue", "create", "--template", "bogus", "--draft", draft}, types},
		"create empty draft":               {[]string{"issue", "create", "--template", "impl-phase", "--draft", ""}, "--draft was given an empty value"},
		"create empty body-file":           {[]string{"issue", "create", "--template", "impl-phase", "--title", "T", "--body-file", ""}, "--body-file was given an empty value"},
		"create empty draft with body":     {[]string{"issue", "create", "--template", "impl-phase", "--draft", "", "--body-file", body, "--title", "T"}, "--draft was given an empty value"},
		"edit empty template":              {[]string{"issue", "edit", "7", "--template", "", "--draft", draft}, "--template needs one of " + types + "; it was given an empty value"},
		"edit unknown template":            {[]string{"issue", "edit", "7", "--template", "bogus", "--draft", draft}, types},
		"edit empty draft":                 {[]string{"issue", "edit", "7", "--template", "impl-phase", "--draft", ""}, "--draft was given an empty value"},
		"edit empty body-file":             {[]string{"issue", "edit", "7", "--template", "impl-phase", "--body-file", ""}, "--body-file was given an empty value"},
		"edit empty draft with body":       {[]string{"issue", "edit", "7", "--template", "impl-phase", "--draft", "", "--body-file", body}, "--draft was given an empty value"},
		"validate empty draft":             {[]string{"template", "validate", "impl-phase", "--draft", ""}, "--draft was given an empty value"},
		"validate empty body-file":         {[]string{"template", "validate", "impl-phase", "--body-file", "", "--title", "T"}, "--body-file was given an empty value"},
		"validate empty draft with body":   {[]string{"template", "validate", "impl-phase", "--draft", "", "--body-file", body, "--title", "T"}, "--draft was given an empty value"},
		"render empty draft":               {[]string{"template", "render", "impl-phase", "--draft", ""}, "--draft was given an empty value"},
	} {
		t.Run(name, func(t *testing.T) {
			fake := &exectest.FakeRunner{}
			stdout, _, err := runIssue(fake, tc.args...)
			ce := wantCode(t, err, 1)
			if !strings.Contains(ce.Msg, tc.want) {
				t.Errorf("message %q should contain %q", ce.Msg, tc.want)
			}
			if len(fake.Calls) != 0 || stdout != "" {
				t.Errorf("want zero gh calls and empty stdout, got %v / %q", fake.Calls, stdout)
			}
		})
	}
}
