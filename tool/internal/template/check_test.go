package template_test

import (
	"strings"
	"testing"

	tmpl "threehillpath.com/marvin-sdd/tool/internal/template"
)

const builtIn = "built-in"

func loadBuiltIn(t *testing.T, name string) *tmpl.Schema {
	t.Helper()
	sc, err := tmpl.LoadSchema(builtIn, schema(t, name))
	if err != nil {
		t.Fatalf("LoadSchema(%s): %v", name, err)
	}
	return sc
}

// phaseMap returns a fully conformant impl-phase section map.
func phaseMap() *tmpl.SectionMap {
	sec := func(c string) []tmpl.Entry { return []tmpl.Entry{{Content: c}} }
	return &tmpl.SectionMap{
		Source:    tmpl.SourceYAML,
		Title:     "[PLAN-00112-1] X",
		TitleLine: 1,
		Metadata: map[string]tmpl.Field{
			"Implementation Plan": {Value: "#132 ([PLAN-00112])", Line: 2},
			"Plan Number":         {Value: "PLAN-00112", Line: 3},
			"Status":              {Value: "Upcoming", Line: 4},
		},
		Sections: map[string][]tmpl.Entry{
			"objective":        sec("Do it."),
			"scope":            sec("Includes things."),
			"components":       sec("Stuff."),
			"verification":     sec("go test ./..."),
			"success_criteria": sec("- [ ] Done"),
		},
	}
}

func check(t *testing.T, name string, m *tmpl.SectionMap) tmpl.Result {
	t.Helper()
	return tmpl.Check(loadBuiltIn(t, name), builtIn, m)
}

// wantOne asserts exactly one finding, at loc, whose message contains every
// substring in want.
func wantOne(t *testing.T, res tmpl.Result, sev tmpl.Severity, loc string, want ...string) {
	t.Helper()
	if len(res.Findings) != 1 {
		t.Fatalf("want exactly 1 finding, got %d:\n%s", len(res.Findings), res.Format())
	}
	f := res.Findings[0]
	if f.Severity != sev || f.Location != loc {
		t.Fatalf("want %s at %s, got %s at %s: %s", sev, loc, f.Severity, f.Location, f.Message)
	}
	for _, w := range want {
		if !strings.Contains(f.Message, w) {
			t.Errorf("message %q missing %q", f.Message, w)
		}
	}
}

func TestCheckPhaseMissingVerification(t *testing.T) {
	m := phaseMap()
	delete(m.Sections, "verification")
	res := check(t, "impl-phase", m)
	wantOne(t, res, tmpl.SeverityError, "section:verification", "Verification", "Add")

	m = phaseMap()
	if res := check(t, "impl-phase", m); len(res.Findings) != 0 {
		t.Fatalf("conformant map should have no findings:\n%s", res.Format())
	}
}

func TestBuiltInSchemasDeriveExpectedKinds(t *testing.T) {
	want := map[string]string{"arch-plan": "arch", "impl-plan": "impl", "impl-phase": "phase", "quick-task": "task"}
	for name, kind := range want {
		sc := loadBuiltIn(t, name)
		if got := sc.ExpectedKind.String(); got != kind {
			t.Errorf("%s: ExpectedKind = %s, want %s", name, got, kind)
		}
	}
}
