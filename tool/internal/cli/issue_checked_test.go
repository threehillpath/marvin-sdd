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
