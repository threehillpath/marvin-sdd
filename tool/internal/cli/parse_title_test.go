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

func runParseTitleCLI(t *testing.T, args ...string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs(append([]string{"parse", "title"}, args...))
	if err := root.Execute(); err != nil {
		t.Fatalf("parse title returned error: %v\nstderr: %s", err, stderr.String())
	}
	return stdout.String()
}

// kind: appears only when the leading bracket classifies; found and plan
// fields stay on PlanIdent.
func TestParseTitleKindLines(t *testing.T) {
	tests := []struct{ title, want string }{
		{"[PLAN-00112-ARCH] X", "found: true\nplan: 112\nplan_number: plan-00112\nkind: arch\nslug: x\n"},
		{"[PLAN-00042-a] X", "found: true\nplan: 42\nplan_number: plan-00042\nsuffix: A\nkind: impl\nslug: x\n"},
		{"[PLAN-00042-1] X", "found: true\nplan: 42\nplan_number: plan-00042\nphase: 1\nkind: phase\nslug: x\n"},
		{"Notes on [PLAN-00042-1]", "found: true\nplan: 42\nplan_number: plan-00042\nphase: 1\nslug: notes-on-plan-00042-1\n"},
	}
	for _, tc := range tests {
		if got := runParseTitleCLI(t, tc.title); got != tc.want {
			t.Errorf("parse title %q\ngot:\n%s\nwant:\n%s", tc.title, got, tc.want)
		}
	}
}

func TestParseTitleTask(t *testing.T) {
	got := runParseTitleCLI(t, "[TASK-00091] Fix X")
	want := "found: true\nkind: task\ntask: 91\ntask_number: TASK-00091\nslug: fix-x\n"
	if got != want {
		t.Errorf("task plain text\ngot:\n%s\nwant:\n%s", got, want)
	}
	// A later plan bracket must not leak plan fields into a task title.
	got = runParseTitleCLI(t, "[TASK-00140] Fix regression from [PLAN-00112-3]")
	want = "found: true\nkind: task\ntask: 140\ntask_number: TASK-00140\nslug: fix-regression-from-plan-00112-3\n"
	if got != want {
		t.Errorf("task with later plan bracket\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestParseTitleKindJSON(t *testing.T) {
	got := runParseTitleCLI(t, "[TASK-00091] Fix X", "--json")
	for _, frag := range []string{`"found": true`, `"kind": "task"`, `"task": 91`, `"task_number": "TASK-00091"`} {
		if !strings.Contains(got, frag) {
			t.Errorf("json missing %s:\n%s", frag, got)
		}
	}
	got = runParseTitleCLI(t, "Notes on [PLAN-00042-1]", "--json")
	if strings.Contains(got, `"kind"`) {
		t.Errorf("non-leading bracket must not emit kind:\n%s", got)
	}
}
