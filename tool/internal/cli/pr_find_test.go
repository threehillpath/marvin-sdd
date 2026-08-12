package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"threehillpath.com/marvin-sdd/tool/internal/cli"
	"threehillpath.com/marvin-sdd/tool/internal/exectest"
)

// TestPRFindPlainTextFound verifies the default (non-JSON) output for a
// matched PR: one key:value line per populated field, in struct-field order,
// with url included like any other populated field.
func TestPRFindPlainTextFound(t *testing.T) {
	withConfigFixture(t)

	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`[{"number":68,"title":"[PLAN-00042-3] Some phase","url":"https://github.com/threehillpath/claude-plan-workflow/pull/68","headRefName":"feature/PLAN-00042/phase-3","baseRefName":"feature/PLAN-00042/main","state":"OPEN"}]`)})

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	root.SetArgs([]string{"pr", "find", "[PLAN-00042-3]"})

	if err := root.Execute(); err != nil {
		t.Fatalf("pr find returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := "found: true\n" +
		"number: 68\n" +
		"title: [PLAN-00042-3] Some phase\n" +
		"url: https://github.com/threehillpath/claude-plan-workflow/pull/68\n" +
		"head: feature/PLAN-00042/phase-3\n" +
		"base: feature/PLAN-00042/main\n" +
		"state: OPEN\n"

	if stdout.String() != want {
		t.Errorf("plain-text output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

// TestPRFindPlainTextNotFound verifies that found: false is always printed,
// never omitted, and no other lines appear when nothing matched.
func TestPRFindPlainTextNotFound(t *testing.T) {
	withConfigFixture(t)

	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`[]`)})

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	root.SetArgs([]string{"pr", "find", "[PLAN-00042-3]"})

	if err := root.Execute(); err != nil {
		t.Fatalf("pr find returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := "found: false\n"
	if stdout.String() != want {
		t.Errorf("plain-text output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

// TestPRFindJSONFidelity verifies --json is byte-identical to the
// pre-Component-5 JSON shape.
func TestPRFindJSONFidelity(t *testing.T) {
	withConfigFixture(t)

	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`[{"number":68,"title":"[PLAN-00042-3] Some phase","url":"https://github.com/threehillpath/claude-plan-workflow/pull/68","headRefName":"feature/PLAN-00042/phase-3","baseRefName":"feature/PLAN-00042/main","state":"OPEN"}]`)})

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	root.SetArgs([]string{"pr", "find", "[PLAN-00042-3]", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("pr find --json returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := `{
  "found": true,
  "number": 68,
  "title": "[PLAN-00042-3] Some phase",
  "url": "https://github.com/threehillpath/claude-plan-workflow/pull/68",
  "head": "feature/PLAN-00042/phase-3",
  "base": "feature/PLAN-00042/main",
  "state": "OPEN"
}
`
	if stdout.String() != want {
		t.Errorf("--json output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}
