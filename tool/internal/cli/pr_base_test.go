package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"threehillpath.com/claude-plan-workflow/tool/internal/cli"
	"threehillpath.com/claude-plan-workflow/tool/internal/exectest"
)

// TestPRBasePlainText verifies the default (non-JSON) output collapses to a
// bare value with no label, matching the board status/config get precedent.
func TestPRBasePlainText(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"pr", "base", "feature/PLAN-00042/phase-3"})

	if err := root.Execute(); err != nil {
		t.Fatalf("pr base returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := "feature/PLAN-00042/main\n"
	if stdout.String() != want {
		t.Errorf("plain-text output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

// TestPRBaseJSONFidelity verifies --json is byte-identical to the
// pre-Component-5 JSON shape.
func TestPRBaseJSONFidelity(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"pr", "base", "feature/PLAN-00042/phase-3", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("pr base --json returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := "{\n  \"base\": \"feature/PLAN-00042/main\"\n}\n"
	if stdout.String() != want {
		t.Errorf("--json output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}
