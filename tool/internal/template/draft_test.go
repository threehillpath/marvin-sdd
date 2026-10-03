package template_test

import (
	"strings"
	"testing"

	tmpl "threehillpath.com/marvin-sdd/tool/internal/template"
)

const phaseDraft = `title: "[PLAN-00112-1] Add the thing"
metadata:
  Status: "Upcoming"
  Plan Number: "PLAN-00112"
  Implementation Plan: "#132 ([PLAN-00112])"
sections:
  # guidance comments are fine
  objective: |
    Do the thing.
  scope: |
    Includes:
    - one
    - two
  components: |
    Stuff.
  verification: |
    go test ./...
  success_criteria: |
    - [ ] Done
`

const planDraft = `title: "[PLAN-00112] Schema-checked plan"
metadata:
  Objective: "Make plans conform"
  Architecture Plan: "#131 ([PLAN-00112-ARCH])"
  Source Issue: "#112"
  Author: "Claude"
  Status: "Draft"
  Last Updated: "2026-10-03"
sections:
  scope: |
    Includes things.
  component:
    - name: "First one"
      content: |
        Body one.
    - name: "Second one"
      content: |
        Body two.
  verification_steps:
    - |
      Run the tests.
  design_notes: |
    Notes.
  success_criteria: |
    - [ ] Done
`

func loadDraft(t *testing.T, typ, draft string) (*tmpl.SectionMap, []tmpl.Finding) {
	t.Helper()
	return tmpl.LoadDraft(loadBuiltIn(t, typ), []byte(draft))
}

func TestLoadDraftValidPhase(t *testing.T) {
	m, fs := loadDraft(t, "impl-phase", phaseDraft)
	if len(fs) != 0 || m == nil {
		t.Fatalf("want a map and no findings, got %+v", fs)
	}
	if m.Source != tmpl.SourceYAML || m.Title != "[PLAN-00112-1] Add the thing" || m.TitleLine != 1 {
		t.Errorf("title/source = %v %q line %d", m.Source, m.Title, m.TitleLine)
	}
	if f := m.Metadata["Plan Number"]; f.Value != "PLAN-00112" || f.Line != 4 {
		t.Errorf("Plan Number = %+v", f)
	}
	es := m.Sections["scope"]
	if len(es) != 1 || es[0].Content != "Includes:\n- one\n- two\n" || es[0].Line != 10 {
		t.Errorf("scope = %+v", es)
	}
	if res := tmpl.Check(loadBuiltIn(t, "impl-phase"), builtIn, m); len(res.Findings) != 0 {
		t.Errorf("valid draft should conform:\n%s", res.Format())
	}
}

func TestLoadDraftValidNamedAndNumbered(t *testing.T) {
	m, fs := loadDraft(t, "impl-plan", planDraft)
	if len(fs) != 0 || m == nil {
		t.Fatalf("want a map and no findings, got %+v", fs)
	}
	cs := m.Sections["component"]
	if len(cs) != 2 || cs[0].Name != "First one" || cs[1].Name != "Second one" || !strings.HasPrefix(cs[1].Content, "Body two") {
		t.Fatalf("component = %+v", cs)
	}
	vs := m.Sections["verification_steps"]
	if len(vs) != 1 || vs[0].Name != "" || vs[0].Content != "Run the tests.\n" {
		t.Errorf("verification_steps = %+v", vs)
	}
	if res := tmpl.Check(loadBuiltIn(t, "impl-plan"), builtIn, m); len(res.Findings) != 0 {
		t.Errorf("valid draft should conform:\n%s", res.Format())
	}
}

// wantDraftFinding asserts that LoadDraft rejected the draft with exactly one
// error finding at location "draft" on the given line (0 means the message
// must say "line unknown"), whose message contains every substring in want.
// No section map may be returned.
func wantDraftFinding(t *testing.T, typ, draft string, line int, want ...string) {
	t.Helper()
	m, fs := loadDraft(t, typ, draft)
	if m != nil {
		t.Errorf("a draft with a loader finding must not yield a section map")
	}
	if len(fs) != 1 {
		t.Fatalf("want exactly 1 finding, got %d: %+v", len(fs), fs)
	}
	f := fs[0]
	if f.Severity != tmpl.SeverityError || f.Location != "draft" || f.Line != line {
		t.Fatalf("want error at draft line %d, got %s at %s line %d: %s", line, f.Severity, f.Location, f.Line, f.Message)
	}
	if line == 0 {
		want = append(want, "line unknown")
	}
	for _, w := range want {
		if !strings.Contains(f.Message, w) {
			t.Errorf("message %q missing %q", f.Message, w)
		}
	}
}

// patch replaces the one occurrence of old in draft with new, failing if old
// is absent so a typo cannot make a test pass vacuously.
func patch(t *testing.T, draft, old, new string) string {
	t.Helper()
	if strings.Count(draft, old) != 1 {
		t.Fatalf("patch: %q must occur exactly once in the draft", old)
	}
	return strings.Replace(draft, old, new, 1)
}

func TestLoadDraftCommentCutsAreFindings(t *testing.T) {
	t.Run("metadata value", func(t *testing.T) {
		d := patch(t, phaseDraft, `Implementation Plan: "#132 ([PLAN-00112])"`, `Implementation Plan: #132 ([PLAN-00112])`)
		wantDraftFinding(t, "impl-phase", d, 5, `"Implementation Plan"`, "cut", `"#132 ([PLAN-00112])"`, "double quotes")
	})
	t.Run("quoted value with trailing comment", func(t *testing.T) {
		d := patch(t, phaseDraft, `Status: "Upcoming"`, `Status: "Upcoming" # note`)
		wantDraftFinding(t, "impl-phase", d, 3, `"Status"`, "comment", "# note", "remove")
	})
	t.Run("section content as plain scalar", func(t *testing.T) {
		d := patch(t, phaseDraft, "objective: |\n    Do the thing.", "objective: Fix #112 thing")
		wantDraftFinding(t, "impl-phase", d, 8, `"objective"`, "cut", "#112 thing", "| block")
	})
	t.Run("named entry name", func(t *testing.T) {
		d := patch(t, planDraft, `name: "First one"`, `name: Fix #112 handling`)
		wantDraftFinding(t, "impl-plan", d, 13, `"name"`, "cut", "#112 handling", "double quotes")
	})
	t.Run("title", func(t *testing.T) {
		d := patch(t, phaseDraft, `title: "[PLAN-00112-1] Add the thing"`, `title: Add the thing #1`)
		wantDraftFinding(t, "impl-phase", d, 1, `"title"`, "cut", "#1", "double quotes")
	})
}
