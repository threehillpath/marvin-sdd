package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"threehillpath.com/marvin-sdd/tool/internal/cli"
	"threehillpath.com/marvin-sdd/tool/internal/exectest"
)

// TestVersionJSON verifies that `version --json` prints the version as a
// pretty-printed JSON object (two-space indent, trailing newline), matching
// this CLI's established JSON convention.
func TestVersionJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"version", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("version --json returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := "{\n  \"version\": \"v0.1.0\"\n}\n"
	if stdout.String() != want {
		t.Errorf("--json output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

// TestVersionPlainTextUnchanged verifies that `version` (no flag) still
// prints the exact plain-text banner from before --json support was added.
func TestVersionPlainTextUnchanged(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("version returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := "marvin v0.1.0 (plan-workflow deterministic CLI)\n"
	if stdout.String() != want {
		t.Errorf("plain-text output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}
