package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"threehillpath.com/marvin-sdd/tool/internal/cli"
	"threehillpath.com/marvin-sdd/tool/internal/exectest"
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
		"phase: 3\n" +
		"slug: some-title\n"

	if stdout.String() != want {
		t.Errorf("plain-text output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

// TestParseTitlePlainTextNotFound verifies that found: false is always
// printed, never omitted. slug is still populated — it strips any leading
// "[...]" and slugifies the rest independently of whether a plan ident matched.
func TestParseTitlePlainTextNotFound(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"parse", "title", "no bracket token here"})

	if err := root.Execute(); err != nil {
		t.Fatalf("parse title returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := "found: false\n" +
		"slug: no-bracket-token-here\n"
	if stdout.String() != want {
		t.Errorf("plain-text output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

// TestParseTitleJSONFidelityFound verifies --json for a matched title,
// including the slug field added for per-phase doc filenames.
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
  "phase": 3,
  "slug": "some-title"
}
`
	if stdout.String() != want {
		t.Errorf("--json output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

// TestParseTitleJSONFidelityNotFound verifies --json for a non-matching
// title: found: false, all plan-ident fields omitted, but slug still present.
func TestParseTitleJSONFidelityNotFound(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"parse", "title", "no bracket token here", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("parse title --json returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := "{\n  \"found\": false,\n  \"slug\": \"no-bracket-token-here\"\n}\n"
	if stdout.String() != want {
		t.Errorf("--json output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

// TestParseTitleSlugCollapsesPunctuation verifies the slug used to name a
// per-phase doc file (docs/stories/<plan>/phase-NN-<slug>.md) matches
// regardless of which skill invocation (wrap-phase or finish-impl) computes
// it, since both go through this same command.
func TestParseTitleSlugCollapsesPunctuation(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"parse", "title", "[PLAN-00042-1] Person & Role Rendering Logic"})

	if err := root.Execute(); err != nil {
		t.Fatalf("parse title returned error: %v\nstderr: %s", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "slug: person-role-rendering-logic\n") {
		t.Errorf("expected slug line in output, got:\n%s", stdout.String())
	}
}
