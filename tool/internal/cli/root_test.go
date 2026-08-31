package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"threehillpath.com/marvin-sdd/tool/internal/cli"
	"threehillpath.com/marvin-sdd/tool/internal/exectest"
)

// TestVersionCmdPrintsCurrentVersion verifies that 'marvin version' reports
// the current marvin CLI version string.
func TestVersionCmdPrintsCurrentVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("version returned error: %v\nstderr: %s", err, stderr.String())
	}

	got := strings.TrimSpace(stdout.String())
	if !strings.Contains(got, "v0.2.0") {
		t.Errorf("version output = %q, want it to contain v0.2.0", got)
	}
}
