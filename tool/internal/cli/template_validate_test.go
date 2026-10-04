package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"threehillpath.com/marvin-sdd/tool/internal/cli"
	"threehillpath.com/marvin-sdd/tool/internal/exectest"
)

const phaseDraftOK = `title: "[PLAN-00112-5] Phase title"
metadata:
  Implementation Plan: "#132 ([PLAN-00112])"
  Plan Number: "PLAN-00112"
  Status: "upcoming"
sections:
  objective: |
    Do the thing.
  scope: |
    In scope.
  components: |
    Some component.
  verification: |
    go test ./...
  success_criteria: |
    - [ ] It works.
`

// phaseDraftNoVerification drops the required verification section.
var phaseDraftNoVerification = strings.Replace(phaseDraftOK, "  verification: |\n    go test ./...\n", "", 1)

// runCLI runs marvin with args and returns stdout, stderr and the error.
func runCLI(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs(args)
	err := root.Execute()
	return stdout.String(), stderr.String(), err
}

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func wantCode(t *testing.T, err error, code int) *cli.CLIError {
	t.Helper()
	var cliErr *cli.CLIError
	if !errors.As(err, &cliErr) || cliErr.Code != code {
		t.Fatalf("want a CLIError with code %d, got %T: %v", code, err, err)
	}
	return cliErr
}

// TestTemplateValidateMissingSectionExits3 is the phase entry point: a draft
// without the verification section exits 3, with the schema line first and an
// error finding naming the section on stdout.
func TestTemplateValidateMissingSectionExits3(t *testing.T) {
	draft := writeTemp(t, "d.yml", phaseDraftNoVerification)
	stdout, _, err := runCLI(t, "template", "validate", "impl-phase", "--draft", draft)
	wantCode(t, err, 3)
	lines := strings.Split(stdout, "\n")
	if lines[0] != "schema: impl-phase (built-in)" {
		t.Errorf("first line = %q, want the schema line", lines[0])
	}
	found := false
	for _, l := range lines[1:] {
		if strings.HasPrefix(l, "error section:verification") {
			found = true
		}
	}
	if !found {
		t.Errorf("no \"error section:verification\" line in:\n%s", stdout)
	}
}

// TestTemplateValidateLoaderFindingsExit3 verifies a draft that will not
// load is a finding (exit 3), not an operational error.
func TestTemplateValidateLoaderFindingsExit3(t *testing.T) {
	draft := writeTemp(t, "d.yml", strings.Replace(phaseDraftOK, "    Do the thing.", "    Do the thing.\n  Note: stray", 1))
	stdout, _, err := runCLI(t, "template", "validate", "impl-phase", "--draft", draft)
	wantCode(t, err, 3)
	if !strings.Contains(stdout, "error draft line ") {
		t.Errorf("want a loader finding on stdout, got:\n%s", stdout)
	}
}

// TestTemplateValidateConformingExits0 verifies a good draft exits 0 with the
// schema line and no error.
func TestTemplateValidateConformingExits0(t *testing.T) {
	draft := writeTemp(t, "d.yml", phaseDraftOK)
	stdout, _, err := runCLI(t, "template", "validate", "impl-phase", "--draft", draft)
	if err != nil {
		t.Fatalf("want exit 0, got %v\n%s", err, stdout)
	}
	if !strings.HasPrefix(stdout, "schema: impl-phase (built-in)\n") {
		t.Errorf("stdout = %q", stdout)
	}
}

// TestTemplateValidateOperationalErrorsExit1 verifies unknown type,
// unreadable input, and a malformed override exit 1 (never 3, never a
// silent fallback to the built-in).
func TestTemplateValidateOperationalErrorsExit1(t *testing.T) {
	draft := writeTemp(t, "d.yml", phaseDraftOK)
	if _, _, err := runCLI(t, "template", "validate", "nosuch", "--draft", draft); err != nil {
		wantCode(t, err, 1)
	} else {
		t.Error("unknown type: want an error")
	}
	_, _, err := runCLI(t, "template", "validate", "impl-phase", "--draft", filepath.Join(t.TempDir(), "missing.yml"))
	wantCode(t, err, 1)
	_, _, err = runCLI(t, "template", "validate", "impl-phase", "--body-file", filepath.Join(t.TempDir(), "missing.md"))
	wantCode(t, err, 1)

	dir := t.TempDir()
	od := filepath.Join(dir, ".claude", "plan-workflow-templates")
	if err := os.MkdirAll(od, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(od, "impl-phase.yml"), []byte("type: [not, a, schema\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	chdir(t, dir)
	stdout, _, err := runCLI(t, "template", "validate", "impl-phase", "--draft", draft)
	wantCode(t, err, 1)
	if stdout != "" {
		t.Errorf("want no stdout on a malformed override, got %q", stdout)
	}
}

// TestTemplateValidateWarningsAloneExit0 verifies a Result with only warnings
// exits 0 and still prints them.
func TestTemplateValidateWarningsAloneExit0(t *testing.T) {
	// An optional section left empty is a warning.
	draft := writeTemp(t, "d.yml", strings.Replace(phaseDraftOK, "  components: |", "  tdd_entry_point: |\n  components: |", 1))
	stdout, _, err := runCLI(t, "template", "validate", "impl-phase", "--draft", draft)
	if err != nil {
		t.Fatalf("want exit 0, got %v\n%s", err, stdout)
	}
	if !strings.Contains(stdout, "warning section:tdd_entry_point") {
		t.Errorf("want the warning printed, got:\n%s", stdout)
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
}

// TestTemplateValidateReportsOverridePath verifies the origin names the
// override file that was used.
func TestTemplateValidateReportsOverridePath(t *testing.T) {
	dir := t.TempDir()
	od := filepath.Join(dir, ".claude", "plan-workflow-templates")
	if err := os.MkdirAll(od, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(od, "quick-task.yml")
	if err := os.WriteFile(path, []byte(overrideSchemaFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	chdir(t, dir)
	draft := writeTemp(t, "d.yml", "title: \"[TASK-00001] x\"\nmetadata:\n  Source Issue: \"#1\"\nsections:\n  override_marker: |\n    hi\n")
	stdout, _, _ := runCLI(t, "template", "validate", "quick-task", "--draft", draft)
	first := strings.SplitN(stdout, "\n", 2)[0]
	// t.TempDir may sit behind a symlink (macOS /var), so compare the suffix.
	if !strings.HasPrefix(first, "schema: quick-task (project override: ") || !strings.HasSuffix(first, filepath.Join(".claude", "plan-workflow-templates", "quick-task.yml")+")") {
		t.Errorf("first line = %q", first)
	}
}

// TestTemplateRenderDraft verifies render --draft prints markdown on success
// and, on an error, exits 3 with the findings on stderr and nothing on stdout.
func TestTemplateRenderDraft(t *testing.T) {
	stdout, stderr, err := runCLI(t, "template", "render", "impl-phase", "--draft", writeTemp(t, "ok.yml", phaseDraftOK))
	if err != nil {
		t.Fatalf("want success, got %v\n%s", err, stderr)
	}
	for _, w := range []string{"**Plan Number:** PLAN-00112", "## Objective", "## Verification"} {
		if !strings.Contains(stdout, w) {
			t.Errorf("markdown missing %q:\n%s", w, stdout)
		}
	}
	if strings.Contains(stdout, "schema:") {
		t.Errorf("stdout must be the body only:\n%s", stdout)
	}

	stdout, stderr, err = runCLI(t, "template", "render", "impl-phase", "--draft", writeTemp(t, "bad.yml", phaseDraftNoVerification))
	wantCode(t, err, 3)
	if stdout != "" {
		t.Errorf("want no stdout on error, got %q", stdout)
	}
	if !strings.Contains(stderr, "error section:verification") {
		t.Errorf("findings missing from stderr:\n%s", stderr)
	}
}

// TestTemplateRenderDraftWarningsGoToStderr verifies warnings do not stop the
// render and are not mixed into the markdown.
func TestTemplateRenderDraftWarningsGoToStderr(t *testing.T) {
	draft := writeTemp(t, "d.yml", strings.Replace(phaseDraftOK, "  components: |", "  tdd_entry_point: |\n  components: |", 1))
	stdout, stderr, err := runCLI(t, "template", "render", "impl-phase", "--draft", draft)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout, "warning") || !strings.Contains(stderr, "warning section:tdd_entry_point") {
		t.Errorf("stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
}

// TestTemplateHelpListsEmbeddedTypes verifies the Use strings of render and
// validate name every embedded schema type, quick-task included.
func TestTemplateHelpListsEmbeddedTypes(t *testing.T) {
	for _, cmd := range []string{"render", "validate"} {
		stdout, _, err := runCLI(t, "template", cmd, "--help")
		if err != nil {
			t.Fatal(err)
		}
		for _, typ := range []string{"arch-plan", "impl-plan", "impl-phase", "quick-task"} {
			if !strings.Contains(stdout, typ) {
				t.Errorf("%s --help does not list %q:\n%s", cmd, typ, stdout)
			}
		}
	}
}

// TestTemplateValidateInputFlagsAreExclusive verifies neither or both of
// --draft/--body-file is a usage error (exit 1) naming both flags.
func TestTemplateValidateInputFlagsAreExclusive(t *testing.T) {
	f := writeTemp(t, "x", "x")
	for _, args := range [][]string{
		{"template", "validate", "impl-phase"},
		{"template", "validate", "impl-phase", "--draft", f, "--body-file", f},
	} {
		stdout, _, err := runCLI(t, args...)
		ce := wantCode(t, err, 1)
		if !strings.Contains(ce.Msg, "--draft") || !strings.Contains(ce.Msg, "--body-file") || stdout != "" {
			t.Errorf("%v: msg %q stdout %q", args, ce.Msg, stdout)
		}
	}
}

// TestTemplateValidateBodyFile verifies the markdown path: a conforming
// body exits 0, a missing --title is a title error (exit 3), and a missing
// section is reported.
func TestTemplateValidateBodyFile(t *testing.T) {
	body := "**Implementation Plan:** #132 ([PLAN-00112])\n**Plan Number:** PLAN-00112\n**Status:** upcoming\n\n## Objective\n\nDo it.\n\n## Scope\n\nIn.\n\n## Components\n\nC.\n\n## Verification\n\nV.\n\n## Success Criteria\n\n- [ ] ok\n"
	f := writeTemp(t, "b.md", body)
	stdout, _, err := runCLI(t, "template", "validate", "impl-phase", "--body-file", f, "--title", "[PLAN-00112-5] Phase title")
	if err != nil {
		t.Fatalf("want exit 0, got %v\n%s", err, stdout)
	}
	stdout, _, err = runCLI(t, "template", "validate", "impl-phase", "--body-file", f)
	wantCode(t, err, 3)
	if !strings.Contains(stdout, "error title") {
		t.Errorf("want a title error without --title:\n%s", stdout)
	}
}

// TestTemplateValidateJSON verifies --json gives schema, origin and findings.
func TestTemplateValidateJSON(t *testing.T) {
	draft := writeTemp(t, "d.yml", phaseDraftNoVerification)
	stdout, _, err := runCLI(t, "template", "validate", "impl-phase", "--draft", draft, "--json")
	wantCode(t, err, 3)
	var out struct {
		Schema   string `json:"schema"`
		Origin   string `json:"origin"`
		Findings []struct {
			Severity string `json:"severity"`
			Location string `json:"location"`
			Line     int    `json:"line"`
			Message  string `json:"message"`
		} `json:"findings"`
	}
	if jerr := json.Unmarshal([]byte(stdout), &out); jerr != nil {
		t.Fatalf("not JSON: %v\n%s", jerr, stdout)
	}
	if out.Schema != "impl-phase" || out.Origin != "built-in" || len(out.Findings) == 0 || out.Findings[0].Severity != "error" || out.Findings[0].Location != "section:verification" {
		t.Errorf("unexpected JSON: %+v", out)
	}
}

// conformingPhaseBody is a markdown body that conforms to impl-phase.
const conformingPhaseBody = "**Implementation Plan:** #132 ([PLAN-00112])\n**Plan Number:** PLAN-00112\n**Status:** upcoming\n\n## Objective\n\nDo it.\n\n## Scope\n\nIn.\n\n## Components\n\nC.\n\n## Verification\n\nV.\n\n## Success Criteria\n\n- [ ] ok\n"

// TestTemplateValidateGoldmarkBackstopOnDraft verifies a draft whose only
// problem is an HTML block in Scope (which Check alone accepts) is refused.
func TestTemplateValidateGoldmarkBackstopOnDraft(t *testing.T) {
	draft := writeTemp(t, "d.yml", strings.Replace(phaseDraftOK, "    In scope.\n", "    In scope.\n\n    <div>\n    hidden\n    </div>\n", 1))
	stdout, _, err := runCLI(t, "template", "validate", "impl-phase", "--draft", draft)
	wantCode(t, err, 3)
	assertScopeHTMLBlock(t, stdout)
}

// TestTemplateValidateGoldmarkBackstopOnBody is the same for --body-file.
func TestTemplateValidateGoldmarkBackstopOnBody(t *testing.T) {
	body := strings.Replace(conformingPhaseBody, "In.\n", "In.\n\n<div>\nhidden\n</div>\n", 1)
	stdout, _, err := runCLI(t, "template", "validate", "impl-phase", "--body-file", writeTemp(t, "b.md", body), "--title", "[PLAN-00112-5] Phase title")
	wantCode(t, err, 3)
	assertScopeHTMLBlock(t, stdout)
}

func assertScopeHTMLBlock(t *testing.T, stdout string) {
	t.Helper()
	for _, l := range strings.Split(stdout, "\n") {
		if strings.HasPrefix(l, "error section:scope") && strings.Contains(l, "HTML block") {
			return
		}
	}
	t.Errorf("no \"error section:scope ... HTML block\" line in:\n%s", stdout)
}
