package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
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

	stdout, stderr, err := runIssue(fake, "issue", "edit", "7", "--template", "impl-phase", "--body-file", writeTemp(t, "b.md", bad))

	wantCode(t, err, 3)
	if len(fake.Calls) != 1 || fake.Calls[0].Args[1] != "view" || stdout != "" {
		t.Errorf("want only the gh issue view call and empty stdout, got %v / %q", fake.Calls, stdout)
	}
	if !strings.Contains(stderr, "error section:verification") {
		t.Errorf("want the verification finding on stderr:\n%s", stderr)
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
			wantCode(t, err, 1)
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
			if name == "body-file with title" {
				want := []string{"issue", "create", "--repo", "threehillpath/marvin-sdd", "--title", "[PLAN-00112-5] Phase title", "--body", conformingPhaseBody}
				if got := fake.Calls[0].Args; strings.Join(got, "\x00") != strings.Join(want, "\x00") {
					t.Errorf("args = %q\nwant   %q", got, want)
				}
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
	stdout, stderr, err := runIssue(fake, "issue", "create", "--template", "impl-phase", "--body-file", writeTemp(t, "b.md", bad), "--title", "[PLAN-00112-5] Phase title")
	wantCode(t, err, 3)
	if len(fake.Calls) != 0 || stdout != "" {
		t.Errorf("want zero gh calls and empty stdout, got %v / %q", fake.Calls, stdout)
	}
	if !strings.Contains(stderr, "error section:verification") {
		t.Errorf("want the verification finding on stderr:\n%s", stderr)
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

// assertScopeHTMLBlockLine asserts stderr carries the goldmark backstop's
// "error section:scope ... HTML block" finding.
func assertScopeHTMLBlockLine(t *testing.T, stderr string) {
	t.Helper()
	for _, l := range strings.Split(stderr, "\n") {
		if strings.HasPrefix(l, "error section:scope") && strings.Contains(l, "HTML block") {
			return
		}
	}
	t.Errorf("no \"error section:scope ... HTML block\" line on stderr:\n%s", stderr)
}

// TestIssueCreateGoldmarkBackstopOnDraft verifies a draft whose only problem
// is an HTML block in Scope (which Check alone accepts) is refused on create.
func TestIssueCreateGoldmarkBackstopOnDraft(t *testing.T) {
	withConfigFixture(t)
	draft := writeTemp(t, "d.yml", strings.Replace(phaseDraftOK, "    In scope.\n", "    In scope.\n\n    <div>\n    hidden\n    </div>\n", 1))
	fake := &exectest.FakeRunner{}

	stdout, stderr, err := runIssue(fake, "issue", "create", "--template", "impl-phase", "--draft", draft)

	wantCode(t, err, 3)
	if len(fake.Calls) != 0 || stdout != "" {
		t.Errorf("want zero gh calls and empty stdout, got %v / %q", fake.Calls, stdout)
	}
	assertScopeHTMLBlockLine(t, stderr)
}

// TestIssueEditGoldmarkBackstopOnBodyFile verifies a body whose only problem
// is an HTML block under Scope is refused on edit, after only the read call.
func TestIssueEditGoldmarkBackstopOnBodyFile(t *testing.T) {
	withConfigFixture(t)
	body := strings.Replace(conformingPhaseBody, "In.\n", "In.\n\n<div>\nhidden\n</div>\n", 1)
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"id":"I_1","number":7,"title":"[PLAN-00112-5] Phase title","state":"OPEN"}`)})

	stdout, stderr, err := runIssue(fake, "issue", "edit", "7", "--template", "impl-phase", "--body-file", writeTemp(t, "b.md", body))

	wantCode(t, err, 3)
	if len(fake.Calls) != 1 || fake.Calls[0].Args[1] != "view" || stdout != "" {
		t.Errorf("want only the gh issue view call and empty stdout, got %v / %q", fake.Calls, stdout)
	}
	assertScopeHTMLBlockLine(t, stderr)
}

// TestIssueCreateAndEditWithoutConfigExit2 verifies a missing config exits 2
// with zero gh calls on both checked commands.
func TestIssueCreateAndEditWithoutConfigExit2(t *testing.T) {
	draft := writeTemp(t, "d.yml", phaseDraftOK)
	chdir(t, t.TempDir())

	// A missing config is an environment problem that blocks everything, so
	// it is reported alone (by design): usage problems in the same
	// invocation are not collected ahead of it.
	for name, args := range map[string][]string{
		"create":                {"issue", "create", "--template", "impl-phase", "--draft", draft},
		"edit":                  {"issue", "edit", "7", "--template", "impl-phase", "--draft", draft},
		"create with bad flags": {"issue", "create", "--template", "bogus", "--draft", "", "--body", "x"},
		"edit with bad flags":   {"issue", "edit", "seven", "--template", "bogus", "--draft", ""},
	} {
		t.Run(name, func(t *testing.T) {
			fake := &exectest.FakeRunner{}
			code, stdout, stderr := runIssueExit(fake, args...)
			if code != 2 {
				t.Errorf("exit code = %d, want 2\nstderr: %s", code, stderr)
			}
			if strings.Contains(stderr, "problems") || strings.Contains(stderr, "unknown schema") || strings.Contains(stderr, "empty value") {
				t.Errorf("the config error must appear alone:\n%s", stderr)
			}
			if len(fake.Calls) != 0 || stdout != "" {
				t.Errorf("want zero gh calls and empty stdout, got %v / %q", fake.Calls, stdout)
			}
		})
	}
}

// TestIssueEditMissingBodyFileMakesNoCalls verifies an unreadable --body-file
// is an exit-1 input error found before any network call, not even the
// read of the issue's title.
func TestIssueEditMissingBodyFileMakesNoCalls(t *testing.T) {
	withConfigFixture(t)
	fake := &exectest.FakeRunner{}

	_, _, err := runIssue(fake, "issue", "edit", "7", "--template", "impl-phase", "--body-file", filepath.Join(t.TempDir(), "nope.md"))

	wantCode(t, err, 1)
	if len(fake.Calls) != 0 {
		t.Errorf("want zero gh calls, got %v", fake.Calls)
	}
}

// runIssueExit runs marvin through RunWithStreams and returns the exit code,
// stdout and stderr, so a test can see exactly what the caller sees.
func runIssueExit(fake *exectest.FakeRunner, args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	root.SetArgs(args)
	code := cli.RunWithStreams(&stdout, &stderr, root.Execute)
	return code, stdout.String(), stderr.String()
}

// TestSeveralUsageProblemsAreReportedTogether verifies one invocation with
// several usage problems reports all of them in a single exit-1 error, one
// problem per line, with zero gh calls and empty stdout.
func TestSeveralUsageProblemsAreReportedTogether(t *testing.T) {
	withConfigFixture(t)
	draft := writeTemp(t, "d.yml", phaseDraftOK)
	body := writeTemp(t, "b.md", conformingPhaseBody)
	for name, tc := range map[string]struct {
		args []string
		want []string
	}{
		"create: unknown type, empty draft, inline body": {
			[]string{"issue", "create", "--template", "bogus", "--draft", "", "--body", "x"},
			[]string{`unknown schema "bogus"`, "--draft was given an empty value", "inline --body"},
		},
		"create: draft with body-file and inline body": {
			[]string{"issue", "create", "--template", "impl-phase", "--draft", draft, "--body-file", body, "--body", "x"},
			[]string{"--draft and --body-file", "inline --body"},
		},
		"create: draft without template and empty body-file": {
			[]string{"issue", "create", "--draft", draft, "--body-file", ""},
			[]string{"--draft requires --template", "--body-file was given an empty value"},
		},
		"create: empty template and empty body-file": {
			[]string{"issue", "create", "--template", "", "--body-file", ""},
			[]string{"--template needs one of", "--body-file was given an empty value"},
		},
		"edit: bad number, unknown type, empty draft": {
			[]string{"issue", "edit", "seven", "--template", "bogus", "--draft", ""},
			[]string{`invalid issue number "seven"`, `unknown schema "bogus"`, "--draft was given an empty value"},
		},
		"edit: no template and both inputs": {
			[]string{"issue", "edit", "7", "--draft", draft, "--body-file", body},
			[]string{"requires --template", "--draft and --body-file"},
		},
		"validate: unknown type, both inputs, title with draft": {
			[]string{"template", "validate", "bogus", "--draft", draft, "--body-file", body, "--title", "T"},
			[]string{`unknown schema "bogus"`, "--draft and --body-file", "--title"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			fake := &exectest.FakeRunner{}
			code, stdout, stderr := runIssueExit(fake, tc.args...)
			if code != 1 {
				t.Errorf("exit code = %d, want 1\nstderr: %s", code, stderr)
			}
			for _, w := range tc.want {
				if !strings.Contains(stderr, w) {
					t.Errorf("stderr should mention %q:\n%s", w, stderr)
				}
			}
			bullets := 0
			for _, l := range strings.Split(stderr, "\n") {
				if strings.HasPrefix(l, "  - ") {
					bullets++
				}
			}
			if bullets != len(tc.want) {
				t.Errorf("want %d problem lines, got %d:\n%s", len(tc.want), bullets, stderr)
			}
			if len(fake.Calls) != 0 || stdout != "" {
				t.Errorf("want zero gh calls and empty stdout, got %v / %q", fake.Calls, stdout)
			}
		})
	}
}

// TestBadDraftWithMismatchedTitleReportsBoth verifies a bad draft plus a
// mismatched --title prints the conformance findings and the mismatch
// together and exits 1 (the invocation was wrong), with zero gh calls. A
// bad draft with a usage problem other than the title behaves the same.
func TestBadDraftWithMismatchedTitleReportsBoth(t *testing.T) {
	withConfigFixture(t)
	bad := writeTemp(t, "d.yml", phaseDraftNoVerification)
	for name, tc := range map[string]struct {
		args []string
		want string
	}{
		"mismatched title": {[]string{"issue", "create", "--template", "impl-phase", "--draft", bad, "--title", "Something else"}, "does not match the draft's title"},
		"inline body":      {[]string{"issue", "create", "--template", "impl-phase", "--draft", bad, "--body", "x"}, "inline --body"},
	} {
		t.Run(name, func(t *testing.T) {
			fake := &exectest.FakeRunner{}
			code, stdout, stderr := runIssueExit(fake, tc.args...)
			if code != 1 {
				t.Errorf("exit code = %d, want 1\nstderr: %s", code, stderr)
			}
			if !strings.Contains(stderr, "error section:verification") {
				t.Errorf("want the conformance finding on stderr:\n%s", stderr)
			}
			if !strings.Contains(stderr, tc.want) {
				t.Errorf("want %q on stderr:\n%s", tc.want, stderr)
			}
			if len(fake.Calls) != 0 || stdout != "" {
				t.Errorf("want zero gh calls and empty stdout, got %v / %q", fake.Calls, stdout)
			}
		})
	}
}

// TestTemplateNameMustBeOneOfTheFixedTypes verifies the type name is checked
// against the fixed built-in set before any override path is built or opened:
// a name outside the set is an exit-1 error listing the valid types on every
// command that takes one, even when a file of that name exists under
// .claude/plan-workflow-templates/, and no gh call is made.
func TestTemplateNameMustBeOneOfTheFixedTypes(t *testing.T) {
	withConfigFixture(t)
	// Directories where each name's override file would resolve: opening one
	// as a file fails with a "reading project template override" error, so
	// any attempt to look the name up is visible in the message. "../x" lands
	// at .claude/x.yml and "a/b" at .claude/plan-workflow-templates/a/b.yml.
	for _, dir := range []string{
		filepath.Join(".claude", "plan-workflow-templates", "bogus.yml"),
		filepath.Join(".claude", "x.yml"),
		filepath.Join(".claude", "plan-workflow-templates", "a", "b.yml"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	draft := writeTemp(t, "d.yml", phaseDraftOK)
	const types = "arch-plan, impl-phase, impl-plan, quick-task"
	for _, typ := range []string{"bogus", "../x", "a/b", ""} {
		for cmdName, args := range map[string][]string{
			"create":   {"issue", "create", "--template", typ, "--draft", draft},
			"edit":     {"issue", "edit", "7", "--template", typ, "--draft", draft},
			"validate": {"template", "validate", typ, "--draft", draft},
			"render":   {"template", "render", typ, "--draft", draft},
		} {
			t.Run(cmdName+" "+typ, func(t *testing.T) {
				fake := &exectest.FakeRunner{}
				code, stdout, stderr := runIssueExit(fake, args...)
				if code != 1 {
					t.Errorf("exit code = %d, want 1\nstderr: %s", code, stderr)
				}
				if !strings.Contains(stderr, types) {
					t.Errorf("stderr should list %q:\n%s", types, stderr)
				}
				if strings.Contains(stderr, "override") {
					t.Errorf("the override path must not be looked up:\n%s", stderr)
				}
				if len(fake.Calls) != 0 || stdout != "" {
					t.Errorf("want zero gh calls and empty stdout, got %v / %q", fake.Calls, stdout)
				}
			})
		}
	}
}

// TestValidateReportsInputFindingsWithUsageProblems verifies validate checks
// the input even when the invocation has usage problems, and reports the
// problems and the input's findings or read error together on stderr with
// empty stdout and exit 1.
func TestValidateReportsInputFindingsWithUsageProblems(t *testing.T) {
	chdir(t, t.TempDir())
	bad := writeTemp(t, "d.yml", phaseDraftNoVerification)
	missing := filepath.Join(t.TempDir(), "nope.yml")
	for name, tc := range map[string]struct {
		args []string
		want []string
	}{
		"bad draft and title":     {[]string{"template", "validate", "impl-phase", "--draft", bad, "--title", "T"}, []string{"error section:verification", "--title applies only to --body-file"}},
		"missing draft and title": {[]string{"template", "validate", "impl-phase", "--draft", missing, "--title", "T"}, []string{`reading --draft "` + missing + `"`, "--title applies only to --body-file"}},
		"empty draft and title":   {[]string{"template", "validate", "impl-phase", "--draft", "", "--title", "T"}, []string{"--draft was given an empty value", "--title applies only to --body-file"}},
		"missing body-file":       {[]string{"template", "validate", "impl-phase", "--body-file", missing, "--draft", ""}, []string{`reading --body-file "` + missing + `"`, "--draft was given an empty value"}},
	} {
		t.Run(name, func(t *testing.T) {
			code, stdout, stderr := runIssueExit(&exectest.FakeRunner{}, tc.args...)
			if code != 1 {
				t.Errorf("exit code = %d, want 1\nstderr: %s", code, stderr)
			}
			for _, w := range tc.want {
				if !strings.Contains(stderr, w) {
					t.Errorf("stderr should contain %q:\n%s", w, stderr)
				}
			}
			if stdout != "" {
				t.Errorf("want empty stdout, got %q", stdout)
			}
		})
	}
}

// TestCreateAndEditReportEveryProblem covers the remaining report-everything
// cases: a body file that cannot be read is reported with the missing title,
// an explicitly empty inline --body is still a usage error, the missing title
// of a body file is reported once, edit reports findings with usage problems
// and its cobra-level problems with the rest.
func TestCreateAndEditReportEveryProblem(t *testing.T) {
	withConfigFixture(t)
	ok := writeTemp(t, "d.yml", phaseDraftOK)
	bad := writeTemp(t, "d.yml", phaseDraftNoVerification)
	okBody := writeTemp(t, "b.md", conformingPhaseBody)
	badBody := writeTemp(t, "bad.md", strings.Replace(conformingPhaseBody, "## Verification\n\nV.\n\n", "", 1))
	missing := filepath.Join(t.TempDir(), "nope.md")
	const types = "arch-plan, impl-phase, impl-plan, quick-task"
	for name, tc := range map[string]struct {
		args    []string
		want    []string
		notWant []string
		view    bool // a gh issue view call is expected
	}{
		"create: unreadable body-file and no title": {
			args: []string{"issue", "create", "--body-file", missing},
			want: []string{"requires --title", `reading --body-file "` + missing + `"`},
		},
		"create: explicitly empty inline body": {
			args: []string{"issue", "create", "--template", "impl-phase", "--draft", ok, "--body", ""},
			want: []string{"inline --body"},
		},
		"create: missing title of a body file is reported once": {
			args:    []string{"issue", "create", "--template", "impl-phase", "--body-file", badBody},
			want:    []string{"requires --title", "error section:verification"},
			notWant: []string{"error title"},
		},
		"edit: findings with a usage problem": {
			args: []string{"issue", "edit", "7", "--template", "impl-phase", "--draft", bad, "--body", "x"},
			want: []string{"error section:verification", "there is no inline --body: use --draft <file.yml> or --body-file <file.md>"},
		},
		"edit: body-file with another problem skips the view": {
			args: []string{"issue", "edit", "seven", "--template", "impl-phase", "--body-file", okBody},
			want: []string{`invalid issue number "seven"`},
		},
		"edit: no arguments and empty flags": {
			args: []string{"issue", "edit", "--template", "", "--draft", ""},
			want: []string{"needs exactly one <issue-number>", "--template needs one of " + types, "--draft was given an empty value"},
		},
		"edit: missing template lists the types": {
			args: []string{"issue", "edit", "7", "--draft", ok},
			want: []string{"requires --template <type>", types},
		},
	} {
		t.Run(name, func(t *testing.T) {
			fake := &exectest.FakeRunner{}
			code, stdout, stderr := runIssueExit(fake, tc.args...)
			if code != 1 {
				t.Errorf("exit code = %d, want 1\nstderr: %s", code, stderr)
			}
			for _, w := range tc.want {
				if !strings.Contains(stderr, w) {
					t.Errorf("stderr should contain %q:\n%s", w, stderr)
				}
			}
			for _, w := range tc.notWant {
				if strings.Contains(stderr, w) {
					t.Errorf("stderr should not contain %q:\n%s", w, stderr)
				}
			}
			if len(fake.Calls) != 0 || stdout != "" {
				t.Errorf("want zero gh calls and empty stdout, got %v / %q", fake.Calls, stdout)
			}
		})
	}
}

// TestProblemFormat verifies the shape of the problems message: one problem
// keeps the command prefix and is a bare message, several print the prefix
// once in a header and no prefix on the lines, and an unreadable file has one
// wording across commands.
func TestProblemFormat(t *testing.T) {
	withConfigFixture(t)
	ok := writeTemp(t, "d.yml", phaseDraftOK)
	missing := filepath.Join(t.TempDir(), "nope.md")

	_, _, err := runIssue(&exectest.FakeRunner{}, "issue", "create", "--template", "impl-phase", "--draft", ok, "--body", "x")
	ce := wantCode(t, err, 1)
	if !strings.HasPrefix(ce.Msg, "issue create: ") || strings.Contains(ce.Msg, "problems:") {
		t.Errorf("single problem should be a bare message with the prefix, got %q", ce.Msg)
	}

	_, _, err = runIssue(&exectest.FakeRunner{}, "issue", "create", "--template", "bogus", "--draft", "", "--body", "x")
	ce = wantCode(t, err, 1)
	lines := strings.Split(ce.Msg, "\n")
	if lines[0] != "issue create: 3 problems:" {
		t.Errorf("header = %q, want %q", lines[0], "issue create: 3 problems:")
	}
	for _, l := range lines[1:] {
		if !strings.HasPrefix(l, "  - ") || strings.Contains(l, "issue create:") {
			t.Errorf("problem line %q should be a bullet without the command prefix", l)
		}
	}

	for name, args := range map[string][]string{
		"create":   {"issue", "create", "--template", "impl-phase", "--title", "T", "--body-file", missing},
		"edit":     {"issue", "edit", "7", "--template", "impl-phase", "--body-file", missing},
		"validate": {"template", "validate", "impl-phase", "--body-file", missing, "--title", "T"},
	} {
		_, _, err := runIssue(&exectest.FakeRunner{}, args...)
		ce := wantCode(t, err, 1)
		if !strings.Contains(ce.Msg, `reading --body-file "`+missing+`"`) {
			t.Errorf("%s: unreadable-file wording = %q", name, ce.Msg)
		}
	}
}

const titleChecksNote = "note: checks that need the title (title/metadata cross-references and the rendered-structure check) did not run; they run once --title is given"

// TestCreateBodyFileWithoutTitleSaysTitleChecksDidNotRun verifies that when
// the missing title is reported as a usage problem, the checks that need the
// title are said not to have run, so a clean-looking body is not mistaken for
// a fully checked one.
func TestCreateBodyFileWithoutTitleSaysTitleChecksDidNotRun(t *testing.T) {
	withConfigFixture(t)
	fake := &exectest.FakeRunner{}
	code, stdout, stderr := runIssueExit(fake, "issue", "create", "--template", "impl-phase", "--body-file", writeTemp(t, "b.md", conformingPhaseBody))
	if code != 1 || stdout != "" || len(fake.Calls) != 0 {
		t.Fatalf("code=%d stdout=%q calls=%v", code, stdout, fake.Calls)
	}
	if !strings.Contains(stderr, titleChecksNote) {
		t.Errorf("stderr should carry the note %q:\n%s", titleChecksNote, stderr)
	}
}

// TestEditBodyFileIsCheckedDespiteOtherProblems verifies edit --body-file
// reports the body's findings together with other problems, as create does:
// the body is checked with an empty title, the title findings are dropped, the
// note says the title-dependent checks did not run, and nothing is sent.
func TestEditBodyFileIsCheckedDespiteOtherProblems(t *testing.T) {
	withConfigFixture(t)
	bad := writeTemp(t, "bad.md", strings.Replace(conformingPhaseBody, "## Verification\n\nV.\n\n", "", 1))
	t.Run("number forgotten", func(t *testing.T) {
		fake := &exectest.FakeRunner{}
		code, stdout, stderr := runIssueExit(fake, "issue", "edit", "--template", "impl-phase", "--body-file", bad)
		if code != 1 || stdout != "" || len(fake.Calls) != 0 {
			t.Fatalf("code=%d stdout=%q calls=%v", code, stdout, fake.Calls)
		}
		for _, w := range []string{"needs exactly one <issue-number>", "error section:verification", titleChecksNote} {
			if !strings.Contains(stderr, w) {
				t.Errorf("stderr should contain %q:\n%s", w, stderr)
			}
		}
		if strings.Contains(stderr, "error title") {
			t.Errorf("title findings should be dropped:\n%s", stderr)
		}
	})
	t.Run("title read fails", func(t *testing.T) {
		fake := &exectest.FakeRunner{}
		fake.Enqueue(exectest.FakeResponse{Stderr: []byte("not found"), ExitCode: 1})
		code, stdout, stderr := runIssueExit(fake, "issue", "edit", "7", "--template", "impl-phase", "--body-file", bad)
		if code != 1 || stdout != "" {
			t.Fatalf("code=%d stdout=%q", code, stdout)
		}
		if len(fake.Calls) != 1 || fake.Calls[0].Args[1] != "view" {
			t.Errorf("want only the gh issue view call, got %v", fake.Calls)
		}
		for _, w := range []string{"not found", "error section:verification", titleChecksNote} {
			if !strings.Contains(stderr, w) {
				t.Errorf("stderr should contain %q:\n%s", w, stderr)
			}
		}
	})
}

// TestValidateReadsInputsEvenWithoutASchema verifies validate reports an
// unreadable input together with an unknown schema or both inputs given.
func TestValidateReadsInputsEvenWithoutASchema(t *testing.T) {
	chdir(t, t.TempDir())
	missing := filepath.Join(t.TempDir(), "nope.yml")
	missingBody := filepath.Join(t.TempDir(), "nope.md")
	for name, tc := range map[string]struct {
		args []string
		want []string
	}{
		"unknown schema and missing draft":     {[]string{"template", "validate", "bogus", "--draft", missing}, []string{`unknown schema "bogus"`, `reading --draft "` + missing + `"`}},
		"both inputs and both unreadable":      {[]string{"template", "validate", "impl-phase", "--draft", missing, "--body-file", missingBody}, []string{"mutually exclusive", `reading --draft "` + missing + `"`, `reading --body-file "` + missingBody + `"`}},
		"unknown schema and missing body-file": {[]string{"template", "validate", "bogus", "--body-file", missingBody, "--title", "T"}, []string{`unknown schema "bogus"`, `reading --body-file "` + missingBody + `"`}},
	} {
		t.Run(name, func(t *testing.T) {
			code, stdout, stderr := runIssueExit(&exectest.FakeRunner{}, tc.args...)
			if code != 1 || stdout != "" {
				t.Errorf("code=%d stdout=%q", code, stdout)
			}
			for _, w := range tc.want {
				if !strings.Contains(stderr, w) {
					t.Errorf("stderr should contain %q:\n%s", w, stderr)
				}
			}
		})
	}
}

// TestValidateNoFalseTitleProblemWithBodyFile verifies --title together with
// --body-file is correct even when an empty --draft is also (wrongly) given:
// only the empty draft is reported.
func TestValidateNoFalseTitleProblemWithBodyFile(t *testing.T) {
	chdir(t, t.TempDir())
	body := writeTemp(t, "b.md", conformingPhaseBody)
	code, _, stderr := runIssueExit(&exectest.FakeRunner{}, "template", "validate", "impl-phase", "--draft", "", "--body-file", body, "--title", "T")
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "--draft was given an empty value") {
		t.Errorf("want the empty-draft problem:\n%s", stderr)
	}
	if strings.Contains(stderr, "--title applies only to --body-file") {
		t.Errorf("--title with --body-file is correct and must not be reported:\n%s", stderr)
	}
}

// TestRenderReportsEmptyDraftAndModeConflictTogether verifies an empty
// --draft combined with --skeleton reports both problems, with the
// "template render:" prefix.
func TestRenderReportsEmptyDraftAndModeConflictTogether(t *testing.T) {
	chdir(t, t.TempDir())
	code, stdout, stderr := runIssueExit(&exectest.FakeRunner{}, "template", "render", "impl-phase", "--skeleton", "--draft", "")
	if code != 1 || stdout != "" {
		t.Errorf("code=%d stdout=%q", code, stdout)
	}
	for _, w := range []string{"template render: 2 problems:", "--draft was given an empty value", "cannot be combined"} {
		if !strings.Contains(stderr, w) {
			t.Errorf("stderr should contain %q:\n%s", w, stderr)
		}
	}
}

// TestCreateDraftWithoutTemplateReportsOnlyThat verifies --draft without
// --template is reported alone, naming the valid types, and does not also
// demand --title and --body (which would steer toward the unchecked path).
func TestCreateDraftWithoutTemplateReportsOnlyThat(t *testing.T) {
	withConfigFixture(t)
	fake := &exectest.FakeRunner{}
	_, _, err := runIssue(fake, "issue", "create", "--draft", writeTemp(t, "d.yml", phaseDraftOK))
	ce := wantCode(t, err, 1)
	if strings.Contains(ce.Msg, "problems:") || strings.Contains(ce.Msg, "requires --title") || strings.Contains(ce.Msg, "requires --body") {
		t.Errorf("want only the --draft problem, got %q", ce.Msg)
	}
	for _, w := range []string{"--draft requires --template", "arch-plan, impl-phase, impl-plan, quick-task"} {
		if !strings.Contains(ce.Msg, w) {
			t.Errorf("message should contain %q: %q", w, ce.Msg)
		}
	}
	if len(fake.Calls) != 0 {
		t.Errorf("want zero gh calls, got %v", fake.Calls)
	}
}

// TestEditTitleAndLabelFlagsAreReportedWithAdvice verifies --title and
// --label on edit are reported problems with advice, together with the other
// problems, rather than a bare unknown-flag error.
func TestEditTitleAndLabelFlagsAreReportedWithAdvice(t *testing.T) {
	withConfigFixture(t)
	draft := writeTemp(t, "d.yml", phaseDraftOK)
	fake := &exectest.FakeRunner{}
	code, stdout, stderr := runIssueExit(fake, "issue", "edit", "7", "--template", "impl-phase", "--draft", draft, "--title", "X", "--label", "bug", "--body", "y")
	if code != 1 || stdout != "" || len(fake.Calls) != 0 {
		t.Fatalf("code=%d stdout=%q calls=%v", code, stdout, fake.Calls)
	}
	for _, w := range []string{
		"issue edit: 3 problems:",
		"--title is not supported: --draft sets the title from the draft's \"title:\", --body-file keeps the current title",
		"--label is not supported: issue edit changes the body (and, with --draft, the title) only",
		"there is no inline --body",
	} {
		if !strings.Contains(stderr, w) {
			t.Errorf("stderr should contain %q:\n%s", w, stderr)
		}
	}
}

// TestCreateWordingIsConsistent verifies create uses the same bare wording as
// edit and validate for a missing input, and names an explicitly empty
// --title as an empty value rather than as a missing one.
func TestCreateWordingIsConsistent(t *testing.T) {
	withConfigFixture(t)
	body := writeTemp(t, "b.md", conformingPhaseBody)
	_, _, err := runIssue(&exectest.FakeRunner{}, "issue", "create", "--template", "impl-phase")
	ce := wantCode(t, err, 1)
	if want := "issue create: " + exactlyOneWording; ce.Msg != want {
		t.Errorf("message = %q, want %q", ce.Msg, want)
	}
	for name, args := range map[string][]string{
		"legacy":   {"issue", "create", "--title", "", "--body", "x"},
		"template": {"issue", "create", "--template", "impl-phase", "--body-file", body, "--title", ""},
	} {
		_, _, err := runIssue(&exectest.FakeRunner{}, args...)
		ce := wantCode(t, err, 1)
		if !strings.Contains(ce.Msg, "--title was given an empty value") {
			t.Errorf("%s: message = %q, want it to name the empty --title", name, ce.Msg)
		}
	}
}

const exactlyOneWording = "pass exactly one of --draft <file.yml> or --body-file <file.md>"

// TestGoldmarkBackstopOnCreateBodyFileAndEditDraft covers the two remaining
// input paths: create --body-file and edit --draft refuse an HTML block in
// Scope, with exit 3, findings on stderr and no mutating gh call.
func TestGoldmarkBackstopOnCreateBodyFileAndEditDraft(t *testing.T) {
	withConfigFixture(t)
	div := strings.Replace(conformingPhaseBody, "In.\n", "In.\n\n<div>\nhidden\n</div>\n", 1)
	divDraft := strings.Replace(phaseDraftOK, "    In scope.\n", "    In scope.\n\n    <div>\n    hidden\n    </div>\n", 1)

	fake := &exectest.FakeRunner{}
	stdout, stderr, err := runIssue(fake, "issue", "create", "--template", "impl-phase", "--body-file", writeTemp(t, "b.md", div), "--title", "[PLAN-00112-5] Phase title")
	wantCode(t, err, 3)
	if len(fake.Calls) != 0 || stdout != "" {
		t.Errorf("create: want zero gh calls and empty stdout, got %v / %q", fake.Calls, stdout)
	}
	assertScopeHTMLBlockLine(t, stderr)

	fake = &exectest.FakeRunner{}
	stdout, stderr, err = runIssue(fake, "issue", "edit", "7", "--template", "impl-phase", "--draft", writeTemp(t, "d.yml", divDraft))
	wantCode(t, err, 3)
	if len(fake.Calls) != 0 || stdout != "" {
		t.Errorf("edit: want zero gh calls and empty stdout, got %v / %q", fake.Calls, stdout)
	}
	assertScopeHTMLBlockLine(t, stderr)
}

// TestOverrideMustDeclareTheRequestedType verifies a project override whose
// "type:" differs from its file name is an error on every command, naming the
// file, the declared type and the expected type, never accepted and reported
// as another schema.
func TestOverrideMustDeclareTheRequestedType(t *testing.T) {
	withConfigFixture(t)
	dir := filepath.Join(".claude", "plan-workflow-templates")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	override := strings.Replace(overrideSchemaFixture, "type: quick-task", "type: custom-thing", 1)
	if err := os.WriteFile(filepath.Join(dir, "impl-phase.yml"), []byte(override), 0o644); err != nil {
		t.Fatal(err)
	}
	draft := writeTemp(t, "d.yml", phaseDraftOK)
	for name, args := range map[string][]string{
		"create":   {"issue", "create", "--template", "impl-phase", "--draft", draft},
		"edit":     {"issue", "edit", "7", "--template", "impl-phase", "--draft", draft},
		"validate": {"template", "validate", "impl-phase", "--draft", draft},
		"render":   {"template", "render", "impl-phase", "--draft", draft},
	} {
		t.Run(name, func(t *testing.T) {
			fake := &exectest.FakeRunner{}
			code, stdout, stderr := runIssueExit(fake, args...)
			if code != 1 {
				t.Errorf("exit code = %d, want 1\nstderr: %s", code, stderr)
			}
			for _, w := range []string{"impl-phase.yml", `declares type "custom-thing"`, `expected "impl-phase"`} {
				if !strings.Contains(stderr, w) {
					t.Errorf("stderr should contain %q:\n%s", w, stderr)
				}
			}
			if strings.Contains(stderr, "schema: custom-thing") || stdout != "" || len(fake.Calls) != 0 {
				t.Errorf("want no schema line, empty stdout and zero calls; got stdout=%q calls=%v\n%s", stdout, fake.Calls, stderr)
			}
		})
	}
}

// TestCreateStrayPositionalIsReportedWithOtherProblems verifies a stray
// positional argument on create is one more reported problem, not a cobra
// error that hides the rest.
func TestCreateStrayPositionalIsReportedWithOtherProblems(t *testing.T) {
	withConfigFixture(t)
	fake := &exectest.FakeRunner{}
	code, stdout, stderr := runIssueExit(fake, "issue", "create", "stray", "--template", "bogus", "--draft", "")
	if code != 1 || stdout != "" || len(fake.Calls) != 0 {
		t.Fatalf("code=%d stdout=%q calls=%v", code, stdout, fake.Calls)
	}
	for _, w := range []string{"issue create: 3 problems:", `takes no positional arguments: got "stray"`, `unknown schema "bogus"`, "--draft was given an empty value"} {
		if !strings.Contains(stderr, w) {
			t.Errorf("stderr should contain %q:\n%s", w, stderr)
		}
	}
}

// TestIssueCreateUncheckedExplicitEmptyFlagsAreReported verifies the
// unchecked path tests --body and --label by whether they were passed, not by
// value: an explicitly empty --body or --label is a reported problem, and
// --body with --body-file is mutually exclusive even when --body is empty.
// Each exits 1 with zero gh calls.
func TestIssueCreateUncheckedExplicitEmptyFlagsAreReported(t *testing.T) {
	withConfigFixture(t)
	bodyFile := writeTemp(t, "b.md", "a body\n")
	for name, tc := range map[string]struct {
		args []string
		want []string
	}{
		"empty body alone":            {[]string{"--title", "T", "--body", ""}, []string{"--body was given an empty value: pass the body, or leave the flag out"}},
		"empty body with a body file": {[]string{"--title", "T", "--body", "", "--body-file", bodyFile}, []string{"--body was given an empty value", "--body and --body-file are mutually exclusive"}},
		"body with a body file":       {[]string{"--title", "T", "--body", "x", "--body-file", bodyFile}, []string{"--body and --body-file are mutually exclusive"}},
		"empty label":                 {[]string{"--title", "T", "--body", "x", "--label", ""}, []string{"--label was given an empty value: pass a label name, or leave the flag out"}},
		"empty label with empty body": {[]string{"--title", "T", "--body", "", "--label", ""}, []string{"--body was given an empty value", "--label was given an empty value"}},
	} {
		t.Run(name, func(t *testing.T) {
			fake := &exectest.FakeRunner{}
			code, stdout, stderr := runIssueExit(fake, append([]string{"issue", "create"}, tc.args...)...)
			if code != 1 || stdout != "" || len(fake.Calls) != 0 {
				t.Fatalf("code=%d stdout=%q calls=%v\nstderr: %s", code, stdout, fake.Calls, stderr)
			}
			for _, w := range tc.want {
				if !strings.Contains(stderr, w) {
					t.Errorf("stderr should contain %q:\n%s", w, stderr)
				}
			}
			if strings.Contains(stderr, "requires --body or --body-file") {
				t.Errorf("an explicitly empty --body is not an absent one:\n%s", stderr)
			}
		})
	}
}

// TestIssueCreateTemplatePathEmptyLabelIsReported verifies an explicitly empty
// --label is a reported problem on the checked path too: exit 1, no gh call.
func TestIssueCreateTemplatePathEmptyLabelIsReported(t *testing.T) {
	withConfigFixture(t)
	draft := writeTemp(t, "d.yml", phaseDraftOK)
	fake := &exectest.FakeRunner{}
	code, stdout, stderr := runIssueExit(fake, "issue", "create", "--template", "impl-phase", "--draft", draft, "--label", "")
	if code != 1 || stdout != "" || len(fake.Calls) != 0 {
		t.Fatalf("code=%d stdout=%q calls=%v\nstderr: %s", code, stdout, fake.Calls, stderr)
	}
	if !strings.Contains(stderr, "--label was given an empty value: pass a label name, or leave the flag out") {
		t.Errorf("stderr:\n%s", stderr)
	}
}

// TestIssueCreateEditMalformedOverrideStructureExits1 verifies an override with
// a duplicate section id exits 1 on issue create and issue edit, naming the
// file, with zero gh calls.
func TestIssueCreateEditMalformedOverrideStructureExits1(t *testing.T) {
	withConfigFixture(t) // chdirs into a fresh directory holding the config
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	od := filepath.Join(dir, ".claude", "plan-workflow-templates")
	if err := os.MkdirAll(od, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(od, "impl-phase.yml")
	override := "type: impl-phase\ntitle_prefix: \"[PLAN-XXXXX-N] <T>\"\nsections:\n  - id: a\n    heading: A\n  - id: a\n    heading: B\n"
	if err := os.WriteFile(path, []byte(override), 0o644); err != nil {
		t.Fatal(err)
	}
	draft := writeTemp(t, "d.yml", phaseDraftOK)
	for name, args := range map[string][]string{
		"create": {"issue", "create", "--template", "impl-phase", "--draft", draft},
		"edit":   {"issue", "edit", "7", "--template", "impl-phase", "--draft", draft},
	} {
		t.Run(name, func(t *testing.T) {
			fake := &exectest.FakeRunner{}
			code, stdout, stderr := runIssueExit(fake, args...)
			if code != 1 || stdout != "" || len(fake.Calls) != 0 {
				t.Fatalf("code=%d stdout=%q calls=%v\nstderr: %s", code, stdout, fake.Calls, stderr)
			}
			for _, w := range []string{path, `"id"`, "twice"} {
				if !strings.Contains(stderr, w) {
					t.Errorf("stderr should contain %q:\n%s", w, stderr)
				}
			}
		})
	}
}
