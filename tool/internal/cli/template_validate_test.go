package cli_test

import (
	"bytes"
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
