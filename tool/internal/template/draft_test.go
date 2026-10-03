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
