package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"threehillpath.com/claude-plan-workflow/tool/internal/cli"
	"threehillpath.com/claude-plan-workflow/tool/internal/exectest"
)

// TestParseTitlePlainTextFound verifies the default (non-JSON) output for a
// title that matches: one key:value line per populated field, in
// struct-field order.
func TestParseTitlePlainTextFound(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"parse", "title", "[PLAN-00042-A-3] Some title"})

	if err := root.Execute(); err != nil {
		t.Fatalf("parse title returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := "found: true\n" +
		"plan: 42\n" +
		"plan_number: plan-00042\n" +
		"suffix: A\n" +
		"phase: 3\n"

	if stdout.String() != want {
		t.Errorf("plain-text output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

// TestParseTitlePlainTextNotFound verifies that found: false is always
// printed, never omitted, and no other lines appear when nothing matched.
func TestParseTitlePlainTextNotFound(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"parse", "title", "no bracket token here"})

	if err := root.Execute(); err != nil {
		t.Fatalf("parse title returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := "found: false\n"
	if stdout.String() != want {
		t.Errorf("plain-text output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

// TestParseTitleJSONFidelityFound verifies --json for a matched title is
// byte-identical to the pre-Component-5 JSON shape.
func TestParseTitleJSONFidelityFound(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"parse", "title", "[PLAN-00042-A-3] Some title", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("parse title --json returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := `{
  "found": true,
  "plan": 42,
  "plan_number": "plan-00042",
  "suffix": "A",
  "phase": 3
}
`
	if stdout.String() != want {
		t.Errorf("--json output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

// TestParseTitleJSONFidelityNotFound verifies --json for a non-matching
// title is byte-identical to the pre-Component-5 JSON shape (found: false,
// all omitempty fields absent).
func TestParseTitleJSONFidelityNotFound(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"parse", "title", "no bracket token here", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("parse title --json returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := "{\n  \"found\": false\n}\n"
	if stdout.String() != want {
		t.Errorf("--json output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}
