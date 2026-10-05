package template_test

import (
	"fmt"
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

const overrideOrigin = "project override: /p/.claude/plan-workflow-templates/x.yml"

func TestLoadSchemaMalformedOverrides(t *testing.T) {
	cases := []struct {
		name string
		yaml string
		want []string
	}{
		{"missing title_prefix", "type: quick-task\nmetadata: [A]\nsections: []\n",
			[]string{overrideOrigin, "title_prefix", "Add", `"[TASK-XXXXX] <Title>"`}},
		{"missing title_prefix, custom type", "type: my-type\nmetadata: [A]\n",
			[]string{overrideOrigin, "title_prefix", "[PLAN-XXXXX-ARCH]", "[PLAN-XXXXX] ", "[PLAN-XXXXX-N]", "[TASK-XXXXX]"}},
		{"missing type", "title_prefix: \"[TASK-XXXXX] <T>\"\n",
			[]string{overrideOrigin, `"type"`, "Add", "type:", "file's base name", "arch-plan", "impl-plan", "impl-phase", "quick-task"}},
		{"numbered section missing named", `type: impl-plan
title_prefix: "[PLAN-XXXXX] <T>"
sections:
  - id: component
    heading: C
    required: true
    repeatable: true
    numbered: true
`, []string{overrideOrigin, `"component"`, `"named"`, "named: true", "named: false"}},
		{"unclassifiable prefix", "type: x\ntitle_prefix: \"[WHAT-XXXXX] <T>\"\n",
			[]string{overrideOrigin, "title_prefix", "[WHAT-XXXXX] <T>"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := tmpl.LoadSchema(overrideOrigin, []byte(c.yaml))
			if err == nil {
				t.Fatal("want malformed-schema error, got nil")
			}
			for _, w := range c.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q missing %q", err, w)
				}
			}
			if c.name == "missing type" && strings.Contains(err.Error(), "type: quick-task") {
				t.Errorf("error must not suggest a specific type: %q", err)
			}
		})
	}
}

// TestLoadSchemaRejectsMalformedStructure has one case per structural rule
// LoadSchema enforces beyond the required fields. Each error must name the
// origin, the offending field and the fix.
func TestLoadSchemaRejectsMalformedStructure(t *testing.T) {
	const head = "type: impl-plan\ntitle_prefix: \"[PLAN-XXXXX] <T>\"\n"
	const sec = "  - id: %s\n    heading: %s\n    required: true\n"
	section := func(id, heading string) string { return fmt.Sprintf(sec, id, heading) }
	cases := []struct {
		name string
		yaml string
		want []string
	}{
		{"unknown top-level field", head + "requried: true\n",
			[]string{overrideOrigin, "requried", "field"}},
		{"unknown section field", head + "sections:\n" + section("a", "A") + "    requried: true\n",
			[]string{overrideOrigin, "requried"}},
		{"empty section id", head + "sections:\n" + section(`""`, "A"),
			[]string{overrideOrigin, "section 1", `"id"`, "empty", "Set"}},
		{"empty section heading", head + "sections:\n" + section("a", `""`),
			[]string{overrideOrigin, `"a"`, `"heading"`, "empty", "Set"}},
		{"duplicate section id", head + "sections:\n" + section("a", "A") + section("a", "B"),
			[]string{overrideOrigin, `"id"`, `"a"`, "twice", "Rename"}},
		{"duplicate section heading", head + "sections:\n" + section("a", "Same") + section("b", "Same"),
			[]string{overrideOrigin, `"heading"`, `"Same"`, "twice", "Rename"}},
		{"duplicate metadata key", head + "metadata: [Author, Status, Author]\n",
			[]string{overrideOrigin, "metadata", `"Author"`, "twice", "Remove"}},
		{"two named numbered sections", head + "sections:\n" + section("a", "A") + "    repeatable: true\n    numbered: true\n    named: true\n" + section("b", "B") + "    repeatable: true\n    numbered: true\n    named: true\n",
			[]string{overrideOrigin, "named: true", `"a"`, `"b"`, "at most one", "named: false"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := tmpl.LoadSchema(overrideOrigin, []byte(c.yaml))
			if err == nil {
				t.Fatal("want a schema error, got nil")
			}
			for _, w := range c.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q missing %q", err, w)
				}
			}
		})
	}
}

// TestLoadSchemaAcceptsBuiltIns verifies the stricter loader still accepts
// every embedded schema.
func TestLoadSchemaAcceptsBuiltIns(t *testing.T) {
	for _, name := range tmpl.DefaultSchemaNames() {
		loadBuiltIn(t, name)
	}
}

func TestBuiltInImplPlanNamedFlags(t *testing.T) {
	sc := loadBuiltIn(t, "impl-plan")
	got := map[string]bool{}
	for _, s := range sc.Sections {
		if s.Numbered {
			if s.Named == nil {
				t.Fatalf("section %s has no named", s.ID)
			}
			got[s.ID] = *s.Named
		}
	}
	if len(got) != 2 || !got["component"] || got["verification_steps"] {
		t.Errorf("named flags = %v, want component:true verification_steps:false", got)
	}
}

func archMap(t *testing.T) *tmpl.SectionMap {
	t.Helper()
	sec := func(c string) []tmpl.Entry { return []tmpl.Entry{{Content: c}} }
	m := &tmpl.SectionMap{
		Source:    tmpl.SourceYAML,
		Title:     "[PLAN-00112-ARCH] X",
		TitleLine: 1,
		Metadata: map[string]tmpl.Field{
			"Source Issue": {Value: "#112"}, "Plan Number": {Value: "PLAN-00112"},
			"Author": {Value: "Claude"}, "Status": {Value: "Draft"}, "Date": {Value: "2026-10-02"},
		},
		Sections: map[string][]tmpl.Entry{},
	}
	for _, s := range loadBuiltIn(t, "arch-plan").Sections {
		m.Sections[s.ID] = sec("content")
	}
	return m
}

func TestCheckConformantArchMap(t *testing.T) {
	if res := check(t, "arch-plan", archMap(t)); len(res.Findings) != 0 {
		t.Fatalf("want no findings:\n%s", res.Format())
	}
}

func TestCheckArchPlaceholderTitleFailsAtTitle(t *testing.T) {
	m := archMap(t)
	m.Title = "[PLAN-XXXXX-ARCH] X"
	wantOne(t, check(t, "arch-plan", m), tmpl.SeverityError, "title", `"[PLAN-XXXXX-ARCH] X"`, "identifier", "[PLAN-00112-ARCH]")
}

func TestCheckTitleMissing(t *testing.T) {
	m := phaseMap()
	m.Title = "  "
	wantOne(t, check(t, "impl-phase", m), tmpl.SeverityError, "title", "missing", `"[PLAN-XXXXX-N] <Phase Title>"`, `Set "title:" in the draft`)

	m = mdPhaseMap()
	m.Title = ""
	res := check(t, "impl-phase", m)
	wantOne(t, res, tmpl.SeverityError, "title", "missing", `"[PLAN-XXXXX-N] <Phase Title>"`, "--title")
	if strings.Contains(res.Findings[0].Message, `"title:"`) {
		t.Errorf("markdown fix must not mention the draft key: %s", res.Findings[0].Message)
	}
}

func TestCheckTitleNoIdentifier(t *testing.T) {
	m := phaseMap()
	m.Title = "Just a title"
	wantOne(t, check(t, "impl-phase", m), tmpl.SeverityError, "title", `"Just a title"`, "identifier", "[PLAN-XXXXX-N] <Phase Title>")
}

func TestCheckTitleKindMismatch(t *testing.T) {
	m := phaseMap()
	m.Title = "[PLAN-00112-ARCH] X"
	m.Metadata["Plan Number"] = tmpl.Field{Value: "PLAN-00112", Line: 3}
	wantOne(t, check(t, "impl-phase", m), tmpl.SeverityError, "title",
		`"[PLAN-00112-ARCH] X"`, "an arch title", "a phase title", "[PLAN-XXXXX-N] <Phase Title>",
		"Change the title's identifier to match", `e.g. "[PLAN-00112-1] <Phase Title>"`)
}

func TestCheckTitleMultiLine(t *testing.T) {
	m := phaseMap()
	m.Title = "[PLAN-00112-1] X\nsecond line"
	wantOne(t, check(t, "impl-phase", m), tmpl.SeverityError, "title",
		`"[PLAN-00112-1] X\nsecond line"`, "more than one line", "Use a single line")
}

func TestCheckMetadataKeyAbsent(t *testing.T) {
	m := phaseMap()
	delete(m.Metadata, "Status")
	wantOne(t, check(t, "impl-phase", m), tmpl.SeverityError, "metadata:Status", `"Status"`, "Add")
}

func TestCheckMetadataValueEmpty(t *testing.T) {
	m := phaseMap()
	m.Metadata["Status"] = tmpl.Field{Value: "  ", Line: 4}
	res := check(t, "impl-phase", m)
	wantOne(t, res, tmpl.SeverityError, "metadata:Status", `"Status"`, "empty", "Set")
	if res.Findings[0].Line != 4 {
		t.Errorf("line = %d, want 4", res.Findings[0].Line)
	}
}

func TestCheckMetadataValueMultiLine(t *testing.T) {
	m := phaseMap()
	m.Metadata["Status"] = tmpl.Field{Value: "Upcoming\nsoon", Line: 4}
	wantOne(t, check(t, "impl-phase", m), tmpl.SeverityError, "metadata:Status", `"Upcoming\nsoon"`, "more than one line", "single line")
}

func TestCheckRequiredSectionEmpty(t *testing.T) {
	m := phaseMap()
	m.Sections["scope"] = []tmpl.Entry{{Content: " \n\t", Line: 9}}
	res := check(t, "impl-phase", m)
	wantOne(t, res, tmpl.SeverityError, "section:scope", `"Scope"`, "empty", "Fill")
	if res.Findings[0].Line != 9 {
		t.Errorf("line = %d, want 9", res.Findings[0].Line)
	}
}

func TestCheckNonRepeatableSectionDuplicated(t *testing.T) {
	m := phaseMap()
	m.Sections["scope"] = []tmpl.Entry{{Content: "a"}, {Content: "b"}}
	wantOne(t, check(t, "impl-phase", m), tmpl.SeverityError, "section:scope", `"Scope"`, "2 entries", "Merge")
}

func implPlanMap() *tmpl.SectionMap {
	e := func(c string) []tmpl.Entry { return []tmpl.Entry{{Content: c}} }
	return &tmpl.SectionMap{
		Source: tmpl.SourceYAML, Title: "[PLAN-00112] X", TitleLine: 1,
		Metadata: map[string]tmpl.Field{
			"Objective": {Value: "o"}, "Architecture Plan": {Value: "#131 ([PLAN-00112-ARCH])"},
			"Source Issue": {Value: "#112"}, "Author": {Value: "a"}, "Status": {Value: "Draft"}, "Last Updated": {Value: "d"},
		},
		Sections: map[string][]tmpl.Entry{
			"scope":              e("s"),
			"component":          {{Name: "First", Content: "c1"}, {Name: "Second", Content: "c2"}},
			"verification_steps": e("v"),
			"design_notes":       e("d"),
			"success_criteria":   e("- [ ] x"),
		},
	}
}

func TestCheckConformantImplPlan(t *testing.T) {
	if res := check(t, "impl-plan", implPlanMap()); len(res.Findings) != 0 {
		t.Fatalf("want no findings:\n%s", res.Format())
	}
}

func TestCheckNamedEntryEmptyName(t *testing.T) {
	m := implPlanMap()
	m.Sections["component"] = []tmpl.Entry{{Name: " ", Content: "c1", Line: 12}}
	wantOne(t, check(t, "impl-plan", m), tmpl.SeverityError, "section:component", "name", "Give")
}

func TestCheckUnnamedNumberedEntryNeedsNoName(t *testing.T) {
	m := implPlanMap()
	m.Sections["verification_steps"] = []tmpl.Entry{{Content: "v1"}, {Content: "v2"}}
	if res := check(t, "impl-plan", m); len(res.Findings) != 0 {
		t.Fatalf("want no findings:\n%s", res.Format())
	}
}

func TestCheckSectionContentH2OutsideFence(t *testing.T) {
	m := phaseMap()
	m.Sections["scope"] = []tmpl.Entry{{Content: "ok\n## Sneaky\nmore", Line: 9}}
	wantOne(t, check(t, "impl-phase", m), tmpl.SeverityError, "section:scope", `"## Sneaky"`, "###")
}

func TestCheckSectionContentH2InsideFenceAllowed(t *testing.T) {
	m := phaseMap()
	m.Sections["scope"] = []tmpl.Entry{{Content: "```md\n## fine\n```\n~~~\n## also fine\n~~~\n### ok"}}
	if res := check(t, "impl-phase", m); len(res.Findings) != 0 {
		t.Fatalf("want no findings:\n%s", res.Format())
	}
}

func TestCheckPlanNumberMismatch(t *testing.T) {
	m := phaseMap()
	m.Metadata["Plan Number"] = tmpl.Field{Value: "PLAN-00113", Line: 3}
	res := check(t, "impl-phase", m)
	wantOne(t, res, tmpl.SeverityError, "metadata:Plan Number", `"PLAN-00113"`, "PLAN-00112", "Set")
	if res.Findings[0].Line != 3 {
		t.Errorf("line = %d, want 3", res.Findings[0].Line)
	}
}

func TestCheckPlanNumberBadForm(t *testing.T) {
	m := phaseMap()
	m.Metadata["Plan Number"] = tmpl.Field{Value: "112", Line: 3}
	wantOne(t, check(t, "impl-phase", m), tmpl.SeverityError, "metadata:Plan Number", `"112"`, "PLAN-00112", "Set")
}

func taskMap() *tmpl.SectionMap {
	e := func(c string) []tmpl.Entry { return []tmpl.Entry{{Content: c}} }
	return &tmpl.SectionMap{
		Source: tmpl.SourceYAML, Title: "[TASK-00091] X", TitleLine: 1,
		Metadata: map[string]tmpl.Field{
			"Source Issue": {Value: "#91"}, "Task Number": {Value: "TASK-00091", Line: 3},
			"Author": {Value: "a"}, "Status": {Value: "s"}, "Date": {Value: "d"},
		},
		Sections: map[string][]tmpl.Entry{
			"problem_statement": e("p"), "scope": e("s"), "technical_analysis": e("t"),
			"tdd_entry_point": e("t"), "implementation_notes": e("i"), "success_criteria": e("- [ ] x"),
		},
	}
}

func TestCheckConformantTaskMap(t *testing.T) {
	if res := check(t, "quick-task", taskMap()); len(res.Findings) != 0 {
		t.Fatalf("want no findings:\n%s", res.Format())
	}
}

func TestCheckTaskNumberMismatch(t *testing.T) {
	m := taskMap()
	m.Metadata["Task Number"] = tmpl.Field{Value: "TASK-00092", Line: 3}
	wantOne(t, check(t, "quick-task", m), tmpl.SeverityError, "metadata:Task Number", `"TASK-00092"`, "TASK-00091", "Set")
}

func TestCheckReferenceMustStartWithIssueRef(t *testing.T) {
	m := phaseMap()
	m.Metadata["Implementation Plan"] = tmpl.Field{Value: "the plan", Line: 2}
	wantOne(t, check(t, "impl-phase", m), tmpl.SeverityError, "metadata:Implementation Plan", `"the plan"`, "#<n>", "Set")
}

func TestCheckReferencePlanIdentMismatch(t *testing.T) {
	m := phaseMap()
	m.Metadata["Implementation Plan"] = tmpl.Field{Value: "#132 ([PLAN-00099])", Line: 2}
	wantOne(t, check(t, "impl-phase", m), tmpl.SeverityError, "metadata:Implementation Plan", `"#132 ([PLAN-00099])"`, "PLAN-00099", "PLAN-00112", "Set")
}

func TestCheckReferenceTrailingTextAllowed(t *testing.T) {
	m := phaseMap()
	m.Metadata["Implementation Plan"] = tmpl.Field{Value: "#132 (some note)", Line: 2}
	if res := check(t, "impl-phase", m); len(res.Findings) != 0 {
		t.Fatalf("want no findings:\n%s", res.Format())
	}
}

func TestResultFormatMissingVerification(t *testing.T) {
	m := phaseMap()
	delete(m.Sections, "verification")
	out := check(t, "impl-phase", m).Format()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got:\n%s", out)
	}
	if lines[0] != "schema: impl-phase (built-in)" {
		t.Errorf("first line = %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "error section:verification: ") {
		t.Errorf("second line = %q", lines[1])
	}
}

func TestResultFormatOrdersErrorsBeforeWarningsInSourceOrder(t *testing.T) {
	r := tmpl.Result{Type: "impl-phase", Origin: "project override: /p/x.yml", Findings: []tmpl.Finding{
		{Severity: tmpl.SeverityWarning, Location: "section:a", Line: 1, Message: "w1"},
		{Severity: tmpl.SeverityError, Location: "section:b", Line: 7, Message: "e7"},
		{Severity: tmpl.SeverityError, Location: "title", Line: 3, Message: "e3"},
		{Severity: tmpl.SeverityWarning, Location: "metadata:K", Message: "w0"},
		{Severity: tmpl.SeverityError, Location: "section:c", Message: "e0"},
	}}
	want := "schema: impl-phase (project override: /p/x.yml)\n" +
		"error title line 3: e3\n" +
		"error section:b line 7: e7\n" +
		"error section:c: e0\n" +
		"warning section:a line 1: w1\n" +
		"warning metadata:K: w0\n"
	if got := r.Format(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestResultFormatNoFindingsIsHeaderOnly(t *testing.T) {
	r := tmpl.Result{Type: "quick-task", Origin: "built-in"}
	if got := r.Format(); got != "schema: quick-task (built-in)\n" {
		t.Errorf("got %q", got)
	}
}

func TestCheckOptionalSectionEmptyIsWarning(t *testing.T) {
	m := phaseMap()
	m.Sections["tdd_entry_point"] = []tmpl.Entry{{Content: "", Line: 20}}
	wantOne(t, check(t, "impl-phase", m), tmpl.SeverityWarning, "section:tdd_entry_point", `"TDD Entry Point"`, "empty", "Fill")
}

func mdPhaseMap() *tmpl.SectionMap {
	m := phaseMap()
	m.Source = tmpl.SourceMarkdown
	for i, id := range []string{"objective", "scope", "components", "verification", "success_criteria"} {
		m.Sections[id] = []tmpl.Entry{{Content: "c", Line: 10 + i*5}}
	}
	return m
}

func TestCheckMarkdownConformant(t *testing.T) {
	if res := check(t, "impl-phase", mdPhaseMap()); len(res.Findings) != 0 {
		t.Fatalf("want no findings:\n%s", res.Format())
	}
}

func TestCheckMarkdownUnknownHeadingWarns(t *testing.T) {
	m := mdPhaseMap()
	m.UnknownHeadings = []tmpl.Heading{{Text: "Extras", Line: 60}}
	res := check(t, "impl-phase", m)
	wantOne(t, res, tmpl.SeverityWarning, "draft", `"## Extras"`, "Rename",
		`"## Objective"`, `"## Scope"`, `"## Verification"`, `"## Success Criteria"`)
	if res.Findings[0].Line != 60 {
		t.Errorf("line = %d, want 60", res.Findings[0].Line)
	}
}

func TestCheckMarkdownUnknownMetadataWarns(t *testing.T) {
	m := mdPhaseMap()
	m.Metadata["Reviewer"] = tmpl.Field{Value: "x", Line: 5}
	wantOne(t, check(t, "impl-phase", m), tmpl.SeverityWarning, "metadata:Reviewer", `"Reviewer"`, "Remove")
}

func TestCheckMarkdownSectionsOutOfOrderWarns(t *testing.T) {
	m := mdPhaseMap()
	m.Sections["objective"] = []tmpl.Entry{{Content: "c", Line: 70}}
	wantOne(t, check(t, "impl-phase", m), tmpl.SeverityWarning, "section:objective", `"## Objective"`, "Move", `before "## Scope"`)

	m = mdPhaseMap()
	m.Sections["scope"] = []tmpl.Entry{{Content: "c", Line: 70}}
	wantOne(t, check(t, "impl-phase", m), tmpl.SeverityWarning, "section:scope", `"## Scope"`, "Move", `after "## Objective"`)
}

func TestCheckYAMLOrderIsIrrelevant(t *testing.T) {
	m := phaseMap()
	m.Sections["objective"] = []tmpl.Entry{{Content: "c", Line: 70}}
	m.Sections["scope"] = []tmpl.Entry{{Content: "c", Line: 5}}
	if res := check(t, "impl-phase", m); len(res.Findings) != 0 {
		t.Fatalf("want no findings:\n%s", res.Format())
	}
}

func TestCheckMarkdownNumberingNotConsecutiveWarns(t *testing.T) {
	m := implPlanMap()
	m.Source = tmpl.SourceMarkdown
	m.Sections["component"] = []tmpl.Entry{{Name: "A", Content: "c", Line: 10, Number: 1}, {Name: "B", Content: "c", Line: 20, Number: 3}}
	m.Sections["verification_steps"] = []tmpl.Entry{{Content: "c", Line: 30, Number: 4}}
	// Lines keep the schema order monotonic so only numbering is reported.
	m.Sections["scope"] = []tmpl.Entry{{Content: "c", Line: 5}}
	m.Sections["design_notes"] = []tmpl.Entry{{Content: "c", Line: 40}}
	m.Sections["success_criteria"] = []tmpl.Entry{{Content: "c", Line: 50}}
	res := check(t, "impl-plan", m)
	wantOne(t, res, tmpl.SeverityWarning, "section:component", `"## 3. B"`, "## 2.")
	if res.Findings[0].Line != 20 {
		t.Errorf("line = %d, want 20", res.Findings[0].Line)
	}
}

func mdImplPlanMap() *tmpl.SectionMap {
	m := implPlanMap()
	m.Source = tmpl.SourceMarkdown
	for i, id := range []string{"scope", "component", "verification_steps", "design_notes", "success_criteria"} {
		es := m.Sections[id]
		for j := range es {
			es[j].Line = 10 + i*10 + j
		}
	}
	m.Sections["component"][0].Number, m.Sections["component"][1].Number = 1, 2
	m.Sections["verification_steps"][0].Number = 3
	return m
}

func TestCheckMissingRepeatableNamedSectionFix(t *testing.T) {
	for _, src := range []tmpl.Source{tmpl.SourceYAML, tmpl.SourceMarkdown} {
		m := implPlanMap()
		if src == tmpl.SourceMarkdown {
			m = mdImplPlanMap()
		}
		delete(m.Sections, "component")
		m.Sections["verification_steps"][0].Number = 1 // keep numbering valid so only the missing section is reported
		var want []string
		if src == tmpl.SourceYAML {
			want = []string{`"component"`, `"component:"`, `"name:"`, `"content: |"`}
		} else {
			want = []string{`"component"`, `"## <n>. <Name>"`, "consecutive"}
		}
		wantOne(t, check(t, "impl-plan", m), tmpl.SeverityError, "section:component", want...)
		if msg := check(t, "impl-plan", m).Findings[0].Message; strings.Contains(msg, "<Component or Layer Name>") {
			t.Errorf("message must not quote the placeholder heading: %s", msg)
		}
	}
}

func TestCheckMissingRepeatableUnnamedSectionFix(t *testing.T) {
	m := implPlanMap()
	delete(m.Sections, "verification_steps")
	wantOne(t, check(t, "impl-plan", m), tmpl.SeverityError, "section:verification_steps",
		`"Verification Steps"`, `"verification_steps:"`, `"- |"`)

	m = mdImplPlanMap()
	delete(m.Sections, "verification_steps")
	wantOne(t, check(t, "impl-plan", m), tmpl.SeverityError, "section:verification_steps",
		`"Verification Steps"`, `"## <n>. Verification Steps"`)
}

func TestCheckEmptyEntryNamesEntryAndOmitsRemoveWhenRequired(t *testing.T) {
	m := implPlanMap()
	m.Sections["component"] = []tmpl.Entry{{Name: "First", Content: " ", Line: 12}}
	res := check(t, "impl-plan", m)
	wantOne(t, res, tmpl.SeverityError, "section:component", `"First"`, "empty", "content: |")
	if strings.Contains(res.Findings[0].Message, "remove") {
		t.Errorf("required section must not suggest removing: %s", res.Findings[0].Message)
	}

	m = mdImplPlanMap()
	m.Sections["component"][0].Content = ""
	wantOne(t, check(t, "impl-plan", m), tmpl.SeverityError, "section:component", `"First"`, "empty", `"## 1. First"`)

	m = implPlanMap()
	m.Sections["verification_steps"] = []tmpl.Entry{{Content: "", Line: 30}}
	wantOne(t, check(t, "impl-plan", m), tmpl.SeverityError, "section:verification_steps", "empty", `"- |"`)
}

func TestCheckEmptyOptionalSectionOffersRemove(t *testing.T) {
	m := phaseMap()
	m.Sections["tdd_entry_point"] = []tmpl.Entry{{Content: "", Line: 20}}
	wantOne(t, check(t, "impl-phase", m), tmpl.SeverityWarning, "section:tdd_entry_point", "or remove it")
}

func TestCheckMarkdownUnknownHeadingListsNumberedForms(t *testing.T) {
	m := mdImplPlanMap()
	m.UnknownHeadings = []tmpl.Heading{{Text: "Extras", Line: 90}}
	wantOne(t, check(t, "impl-plan", m), tmpl.SeverityWarning, "draft", `"## Extras"`,
		`"## Scope"`, `"## <n>. <Name>"`, `"## <n>. Verification Steps"`, `"## Design Notes"`)
}

func TestCheckMarkdownInterleavedEntriesOutOfOrderWarns(t *testing.T) {
	m := mdImplPlanMap()
	// Lines: A(11) Verification(25) B(30): the second component follows Verification Steps.
	m.Sections["component"] = []tmpl.Entry{{Name: "A", Content: "c", Line: 11, Number: 1}, {Name: "B", Content: "c", Line: 30, Number: 3}}
	m.Sections["verification_steps"] = []tmpl.Entry{{Content: "c", Line: 25, Number: 2}}
	m.Sections["design_notes"][0].Line = 40
	m.Sections["success_criteria"][0].Line = 50
	res := check(t, "impl-plan", m)
	wantOne(t, res, tmpl.SeverityWarning, "section:component", `"## 3. B"`, "Move", `after "## 1. A"`)
	if res.Findings[0].Line != 30 {
		t.Errorf("line = %d, want 30", res.Findings[0].Line)
	}
}

func TestCheckSectionContentFenceRules(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string // offending line, "" when content is fine
	}{
		{"longer fence holds a shorter fence line", "````\n```\n## inside\n```\n````\n### ok", ""},
		{"info string does not close", "```python\n## inside\n```", ""},
		{"fence with info string inside a fence does not close it", "```\n```python\n## still inside\n```", ""},
		{"fence closes only on the same char", "```\n~~~\n## inside\n~~~\n```", ""},
		{"heading after a properly closed fence", "```\ncode\n```\n## real", "## real"},
		{"indented H2 is a heading", "text\n  ## Indented", "## Indented"},
		{"four spaces is code, not a heading", "text\n\n    ## code", ""},
		{"hash run without space is not an H2", "##tag", ""},
		{"H3 is fine", "### ok", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := phaseMap()
			m.Sections["scope"] = []tmpl.Entry{{Content: c.content, Line: 9}}
			res := check(t, "impl-phase", m)
			if c.want == "" {
				if len(res.Findings) != 0 {
					t.Fatalf("want no findings:\n%s", res.Format())
				}
				return
			}
			wantOne(t, res, tmpl.SeverityError, "section:scope", "###", `"`+c.want+`"`)
		})
	}
}

func TestCheckUnknownSourceIsReported(t *testing.T) {
	m := phaseMap()
	m.Source = tmpl.SourceUnknown
	wantOne(t, check(t, "impl-phase", m), tmpl.SeverityError, "draft",
		"SectionMap.Source", "SourceYAML", "SourceMarkdown")
}

func TestCheckZeroSourceIsUnknown(t *testing.T) {
	m := phaseMap()
	m.Source = 0
	if m.Source != tmpl.SourceUnknown {
		t.Fatalf("zero Source = %v, want SourceUnknown", m.Source)
	}
}

func TestCheckRejectsSchemaNotFromLoadSchema(t *testing.T) {
	sc := &tmpl.Schema{Type: "impl-phase", TitlePrefix: "[PLAN-XXXXX-N] <Phase Title>"}
	res := tmpl.Check(sc, builtIn, phaseMap())
	wantOne(t, res, tmpl.SeverityError, "draft", "LoadSchema", "Schema")
}

func TestCheckEmptyNamedEntryWithNoNameOrContent(t *testing.T) {
	entry := []tmpl.Entry{{Name: "", Content: "", Line: 12}}

	m := implPlanMap()
	m.Sections["component"] = entry
	res := check(t, "impl-plan", m)
	var sawEmpty bool
	for _, f := range res.Findings {
		if f.Location != "section:component" {
			continue
		}
		if strings.Contains(f.Message, "- |") {
			t.Errorf("named section must never mention \"- |\": %s", f.Message)
		}
		if strings.Contains(f.Message, "is empty") {
			sawEmpty = true
			for _, w := range []string{"content: |", "name:"} {
				if !strings.Contains(f.Message, w) {
					t.Errorf("empty finding missing %q: %s", w, f.Message)
				}
			}
		}
	}
	if !sawEmpty {
		t.Fatalf("want an empty-entry finding:\n%s", res.Format())
	}

	m = mdImplPlanMap()
	m.Sections["component"] = []tmpl.Entry{{Name: "", Content: "", Line: 12, Number: 1}}
	res = check(t, "impl-plan", m)
	var sawMDEmpty bool
	for _, f := range res.Findings {
		if strings.Contains(f.Message, "<Component or Layer Name>") {
			t.Errorf("placeholder heading leaked: %s", f.Message)
		}
		if f.Location == "section:component" && strings.Contains(f.Message, "is empty") {
			sawMDEmpty = true
			if !strings.Contains(f.Message, `"## 1. <Name>"`) {
				t.Errorf("markdown empty finding must name %q: %s", "## 1. <Name>", f.Message)
			}
		}
	}
	if !sawMDEmpty {
		t.Fatalf("want a section:component finding containing \"is empty\" on the markdown path:\n%s", res.Format())
	}
}

// An unnamed component entry that is out of sequence must be reported by the
// heading the author would write, never the schema's placeholder heading.
func TestCheckMarkdownNumberingUnnamedComponentNoPlaceholder(t *testing.T) {
	m := mdImplPlanMap()
	m.Sections["component"] = []tmpl.Entry{{Name: "", Content: "c", Line: 20, Number: 2}, {Name: "B", Content: "c", Line: 21, Number: 2}}
	res := check(t, "impl-plan", m)
	var sawNumbering bool
	for _, f := range res.Findings {
		if strings.Contains(f.Message, "<Component or Layer Name>") {
			t.Errorf("placeholder heading leaked: %s", f.Message)
		}
		if strings.Contains(f.Message, "breaks the sequence") {
			sawNumbering = true
			if !strings.Contains(f.Message, `"## 2. <Name>"`) || !strings.Contains(f.Message, "## 1.") {
				t.Errorf("numbering finding must quote the author's heading and the renumber target: %s", f.Message)
			}
		}
	}
	if !sawNumbering {
		t.Fatalf("want a numbering warning:\n%s", res.Format())
	}
}

// The content-H2 error must quote the actual line and say where in the
// section it is.
func TestCheckSectionContentH2ReportsLineWithinSection(t *testing.T) {
	m := phaseMap()
	m.Sections["scope"] = []tmpl.Entry{{Content: "first\nsecond\n  ##   Foo", Line: 9}}
	wantOne(t, check(t, "impl-phase", m), tmpl.SeverityError, "section:scope", `"##   Foo"`, "line 3 of the section", "###")
}

// TestHeadingLinesAreOneBasedAndStripMarker guards what FindH2Lines used to:
// headings outside fences are found with one-based lines and the "##" marker
// and surrounding spaces stripped, and a fenced "## " line is not a heading.
// It asserts it through the markdown path production uses (UnknownHeadings
// from the parser, and the findings CheckMarkdown reports).
func TestHeadingLinesAreOneBasedAndStripMarker(t *testing.T) {
	sc := loadBuiltIn(t, "impl-phase")
	body := "## First\ntext\n```\n## fenced\n```\n  ##   Spaced  \n##\n"
	got := tmpl.ParseMarkdown(sc, "[PLAN-00112-1] X", body).UnknownHeadings
	want := []tmpl.Heading{{Text: "First", Line: 1}, {Text: "Spaced", Line: 6}, {Text: "", Line: 7}}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i].Text != want[i].Text || got[i].Line != want[i].Line {
			t.Errorf("heading %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	res := tmpl.CheckMarkdown(sc, builtIn, "[PLAN-00112-1] X", body)
	if strings.Contains(res.Format(), "fenced") {
		t.Errorf("the fenced line must not be reported as a heading:\n%s", res.Format())
	}
}

func TestCheckUnclosedFenceIsError(t *testing.T) {
	cases := []struct {
		name    string
		content string
		fence   string // want quoted in the message; "" when content is fine
	}{
		{"unclosed backticks", "text\n```go\ncode\n", "```"},
		{"unclosed tildes", "~~~~\ncode", "~~~~"},
		{"properly closed", "```\ncode\n```", ""},
		{"longer closing fence closes", "```\ncode\n`````", ""},
		{"shorter closing fence does not close", "````\ncode\n```", "````"},
		{"unclosed fence hides later headings", "```\n## inside", "```"},
	}
	for _, src := range []tmpl.Source{tmpl.SourceYAML, tmpl.SourceMarkdown} {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				m := phaseMap()
				m.Source = src
				m.Sections["scope"] = []tmpl.Entry{{Content: c.content, Line: 9}}
				res := check(t, "impl-phase", m)
				if c.fence == "" {
					if len(res.Findings) != 0 {
						t.Fatalf("want no findings:\n%s", res.Format())
					}
					return
				}
				wantOne(t, res, tmpl.SeverityError, "section:scope",
					`"Scope"`, `"`+c.fence+`"`, "never closed", "every later section", "matching fence line")
				if res.Findings[0].Line != 9 {
					t.Errorf("line = %d, want 9", res.Findings[0].Line)
				}
				where := "inside the \"scope\" block"
				if src == tmpl.SourceMarkdown {
					where = "under that heading"
				}
				if !strings.Contains(res.Findings[0].Message, where) {
					t.Errorf("message missing %q: %s", where, res.Findings[0].Message)
				}
			})
		}
	}
}

// guardCase checks one structure guard on both input paths: the content must
// yield exactly one error at section:scope naming the section by heading,
// the line within the section, and a fix phrased for the path.
func guardCase(t *testing.T, content string, line int, want ...string) {
	t.Helper()
	for _, src := range []tmpl.Source{tmpl.SourceYAML, tmpl.SourceMarkdown} {
		m := phaseMap()
		m.Source = src
		m.Sections["scope"] = []tmpl.Entry{{Content: content, Line: 9}}
		res := check(t, "impl-phase", m)
		w := append([]string{`"Scope"`, fmt.Sprintf("line %d of the section", line)}, want...)
		if src == tmpl.SourceMarkdown {
			w = append(w, "under that heading")
		} else {
			w = append(w, `inside the "scope" block`)
		}
		wantOne(t, res, tmpl.SeverityError, "section:scope", w...)
	}
}

func TestCheckSetextHeadingIsError(t *testing.T) {
	// guardCase's line is the text line that turns into a heading.
	guardCase(t, "ok\n\nTitle text\n---\nmore", 3, `"Title text"`, "---", "horizontal rule", "blank line", "### Title text")
	guardCase(t, "first\nsecond\n--", 2, "--")

	// Conservative rule: an underline directly after ANY non-blank line is an
	// error, whatever that line contains.
	for name, content := range map[string]string{
		"hash-number paragraph": "#112 tracks this\n---",
		"inline kbd tag":        "<kbd>Ctrl</kbd>+C copies\n---",
		"autolink":              "<https://example.com> has docs\n---",
		"indented continuation": "Title\n    continued\n---",
		"non-1 ordered item":    "Text\n2. item\n---",
		"list item":             "- item\n---",
		"quote":                 "> quote\n---",
		"heading":               "### H\n---",
		"hr then hr":            "---\n---",
		"indented code":         "    code\n---",
	} {
		t.Run(name, func(t *testing.T) {
			for _, src := range []tmpl.Source{tmpl.SourceYAML, tmpl.SourceMarkdown} {
				m := phaseMap()
				m.Source = src
				m.Sections["scope"] = []tmpl.Entry{{Content: content, Line: 9}}
				res := check(t, "impl-phase", m)
				wantOne(t, res, tmpl.SeverityError, "section:scope", `"Scope"`, "---", "horizontal rule", "blank line", "###")
			}
		})
	}
}

// An "=" underline is never a thematic break, so its fix must not offer a
// horizontal rule.
func TestCheckSetextEqualsFixHasNoHorizontalRule(t *testing.T) {
	m := phaseMap()
	m.Sections["scope"] = []tmpl.Entry{{Content: "Big title\n=====", Line: 9}}
	res := check(t, "impl-phase", m)
	wantOne(t, res, tmpl.SeverityError, "section:scope", `"Big title"`, "=====", "line 1 of the section", "Remove", "### Big title")
	if strings.Contains(res.Findings[0].Message, "horizontal rule") {
		t.Errorf("an = underline is not a thematic break: %s", res.Findings[0].Message)
	}
}

func TestCheckSetextLookalikesAreFine(t *testing.T) {
	for name, ok := range map[string]string{
		"hr after blank line": "text\n\n---\nmore",
		"hr at start":         "---\ntext",
		"fenced underline":    "```\nTitle\n---\n```",
		"equals after blank":  "text\n\n===\n",
		"table delimiter":     "| a | b |\n|---|---|",
		"mixed chars":         "text\n-=-",
	} {
		m := phaseMap()
		m.Sections["scope"] = []tmpl.Entry{{Content: ok}}
		if res := check(t, "impl-phase", m); len(res.Findings) != 0 {
			t.Errorf("%s: content %q: want no findings:\n%s", name, ok, res.Format())
		}
	}
}

func TestCheckLineBreaksInSingleLineFields(t *testing.T) {
	t.Run("named entry name with newline", func(t *testing.T) {
		m := implPlanMap()
		m.Sections["component"] = []tmpl.Entry{{Name: "Foo\n## Injected", Content: "body", Line: 12}}
		res := check(t, "impl-plan", m)
		wantOne(t, res, tmpl.SeverityError, "section:component", `"Foo\n## Injected"`, "line break", "single line", `"name:"`)
		if res.Findings[0].Line != 12 {
			t.Errorf("line = %d, want 12", res.Findings[0].Line)
		}
	})
	t.Run("name with bare CR on the markdown path", func(t *testing.T) {
		m := mdImplPlanMap()
		m.Sections["component"][0].Name = "First\r## one"
		wantOne(t, check(t, "impl-plan", m), tmpl.SeverityError, "section:component", "line break", "single line")
	})
	t.Run("metadata value with bare CR", func(t *testing.T) {
		m := phaseMap()
		m.Metadata["Status"] = tmpl.Field{Value: "Up\r## Injected", Line: 4}
		wantOne(t, check(t, "impl-phase", m), tmpl.SeverityError, "metadata:Status", "more than one line", "single line")
	})
	t.Run("title with bare CR", func(t *testing.T) {
		m := phaseMap()
		m.Title = "[PLAN-00112-1] A\rB"
		wantOne(t, check(t, "impl-phase", m), tmpl.SeverityError, "title", "more than one line", "single line")
	})
}

const banFix = "backticks"

// Raw HTML that could open a comment, a collapsible or a raw block is
// banned outright outside code, whether or not it is closed.
func TestCheckRawHTMLIsBanned(t *testing.T) {
	cases := []struct {
		name    string
		content string
		line    int
		tag     string
	}{
		{"unclosed comment", "text\n<!-- note to self\nmore", 2, "<!--"},
		{"closed comment", "<!-- x -->\ntext", 1, "<!--"},
		{"comment after text", "a <!-- x --> b", 1, "<!--"},
		{"closed details", "<details>\n<summary>S</summary>\nbody\n</details>", 1, "<details>"},
		{"details with attributes", "x\n<details open>\nbody", 2, "<details"},
		{"closing details alone", "a\nb\n</details>", 3, "</details>"},
		{"uppercase pre", "<PRE class=\"x\">\nstuff", 1, "<PRE"},
		{"closed pre", "<pre>one line</pre>", 1, "<pre>"},
		{"script", "intro\n<script>\nx\n", 2, "<script>"},
		{"style", "<style>a{}</style>", 1, "<style>"},
		{"textarea", "t\n<textarea>\nx", 2, "<textarea>"},
		{"comment after an inline tag, then details", "<kbd>x</kbd> <!-- a\n<details>\n-->", 1, "<!--"},
		{"after a closed pre on the same line", "</pre> <!-- c", 1, "</pre>"},
		{"unmatched triple backtick run before a real tag", "Use ``` for fences and <details> blocks, see `scanContent`.", 1, "<details>"},
		{"escaped backticks do not make a code span", "ok\nUse \\`<details>\\` literally", 2, "<details>"},
		{"a run of the wrong length does not match", "Use `` here <details> and ``` there", 1, "<details>"},
		{"unmatched run does not carry over a blank line", "A ` tick\n\nthen <details> and ` tock", 3, "<details>"},
		{"processing instruction", "see <?php echo 1; ?> here", 1, "<?"},
		{"unclosed processing instruction", "a\nb <? start\nc", 2, "<?"},
		{"cdata", "x <![CDATA[ hidden", 1, "<![CDATA["},
		{"declaration", "see <!DOCTYPE html> here", 1, "<!"},
		{"lowercase declaration", "see <!doctype x", 1, "<!"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			guardCase(t, c.content, c.line, "raw HTML", `"`+c.tag, "swallow", banFix, "remove")
		})
	}
}

func TestCheckRawHTMLInCodeOrHarmlessTagsIsFine(t *testing.T) {
	for name, ok := range map[string]string{
		"inline code":                        "Mention `<details>` and `<!--` in prose.",
		"double backticks":                   "Use ``<details>`` here.",
		"double backticks around a backtick": "Use `` `<pre>` `` here.",
		"span that wraps lines":              "A `code span\nthat wraps <details> lines` ok",
		"span over three lines":              "x ``a\nb <pre>\nc`` y",
		"fenced block":                       "```html\n<details>\n<!-- x -->\n<pre>\n```",
		"kbd":                                "Press <kbd>Ctrl</kbd>+C.",
		"autolink":                           "See <https://example.com/docs> for more.",
		"similar tag name":                   "<prefix>x</prefix> and <stylesheet>",
		"br":                                 "line<br>break",
		"a comment-like text":                "<! not a comment",
		"a lone question mark":               "Is it fine? <3 and < ? too",
	} {
		m := phaseMap()
		m.Sections["scope"] = []tmpl.Entry{{Content: ok}}
		if res := check(t, "impl-phase", m); len(res.Findings) != 0 {
			t.Errorf("%s: content %q: want no findings:\n%s", name, ok, res.Format())
		}
	}
}

func TestCheckRawHTMLInNamedEntryContent(t *testing.T) {
	m := implPlanMap()
	m.Sections["component"] = []tmpl.Entry{{Name: "A", Content: "x\n<details>\ny", Line: 12}}
	wantOne(t, check(t, "impl-plan", m), tmpl.SeverityError, "section:component", "raw HTML", `"<details>"`, "line 2 of the section", banFix)
}

func TestCheckRawHTMLInMetadataNameAndTitle(t *testing.T) {
	t.Run("metadata value", func(t *testing.T) {
		m := implPlanMap()
		m.Metadata["Objective"] = tmpl.Field{Value: "Collapse verbose logs into <details> blocks", Line: 3}
		res := check(t, "impl-plan", m)
		wantOne(t, res, tmpl.SeverityError, "metadata:Objective", "raw HTML", `"<details>"`, "swallow", banFix, `"Objective"`, `edit "Objective" under "metadata:"`)
		if res.Findings[0].Line != 3 {
			t.Errorf("line = %d, want 3", res.Findings[0].Line)
		}
	})
	t.Run("metadata value, markdown", func(t *testing.T) {
		m := mdPhaseMap()
		m.Metadata["Status"] = tmpl.Field{Value: "<!-- hidden", Line: 4}
		wantOne(t, check(t, "impl-phase", m), tmpl.SeverityError, "metadata:Status", "raw HTML", `"<!--"`, `"**Status:**"`)
	})
	t.Run("entry name", func(t *testing.T) {
		m := implPlanMap()
		m.Sections["component"] = []tmpl.Entry{{Name: "Use <details> here", Content: "x", Line: 12}}
		wantOne(t, check(t, "impl-plan", m), tmpl.SeverityError, "section:component", "raw HTML", `"<details>"`, "name", banFix)
	})
	t.Run("title", func(t *testing.T) {
		m := phaseMap()
		m.Title = "[PLAN-00112-1] Add <script> support"
		wantOne(t, check(t, "impl-phase", m), tmpl.SeverityError, "title", "raw HTML", `"<script>"`, banFix)
	})
	t.Run("inline code is fine in metadata", func(t *testing.T) {
		m := implPlanMap()
		m.Metadata["Objective"] = tmpl.Field{Value: "Collapse logs into `<details>` blocks", Line: 3}
		if res := check(t, "impl-plan", m); len(res.Findings) != 0 {
			t.Fatalf("want no findings:\n%s", res.Format())
		}
	})
}

// The setext rule runs on every line of content now that no HTML state can
// skip lines.
func TestCheckSetextAppliesAfterInlineHTML(t *testing.T) {
	guardCase(t, "<kbd>x</kbd> <https://example.com>\n---", 1, "---", "horizontal rule")
}

func TestCheckRawHTMLInUnknownMarkdownMetadata(t *testing.T) {
	m := mdPhaseMap()
	m.Metadata["Notes"] = tmpl.Field{Value: "wrap logs in <details>", Line: 5}
	res := check(t, "impl-phase", m)
	var sawErr, sawWarn bool
	for _, f := range res.Findings {
		if f.Location != "metadata:Notes" {
			t.Errorf("unexpected finding: %+v", f)
		}
		switch f.Severity {
		case tmpl.SeverityError:
			sawErr = true
			for _, w := range []string{"raw HTML", `"<details>"`, `"Notes"`, "backticks", `"**Notes:**"`} {
				if !strings.Contains(f.Message, w) {
					t.Errorf("error %q missing %q", f.Message, w)
				}
			}
			if f.Line != 5 {
				t.Errorf("line = %d, want 5", f.Line)
			}
		case tmpl.SeverityWarning:
			sawWarn = true
		}
	}
	if !sawErr || !sawWarn {
		t.Fatalf("want the raw-HTML error and the not-in-schema warning:\n%s", res.Format())
	}
}

// TestCheckInlineHTMLOpenersInMetadata covers the cross-line case: a
// processing instruction or declaration opened in one metadata value would
// swallow the following metadata lines, and GitHub then shows
// "raw HTML omitted" where the parser saw values.
func TestCheckInlineHTMLOpenersInMetadata(t *testing.T) {
	for _, c := range []struct{ value, tag string }{
		{"<?php start", "<?"},
		{"<![CDATA[ start", "<![CDATA["},
		{"<!DOCTYPE start", "<!"},
	} {
		m := implPlanMap()
		m.Metadata["Objective"] = tmpl.Field{Value: c.value, Line: 3}
		m.Metadata["Author"] = tmpl.Field{Value: "end ?>", Line: 4}
		wantOne(t, check(t, "impl-plan", m), tmpl.SeverityError, "metadata:Objective", "raw HTML", `"`+c.tag, banFix)
	}
}

// TestCheckMarkdownInlineHTMLOpenerAcrossMetadataLines verifies the markdown
// path refuses a body whose metadata lines are joined into one raw HTML
// construct.
func TestCheckMarkdownInlineHTMLOpenerAcrossMetadataLines(t *testing.T) {
	sc := loadBuiltIn(t, "impl-phase")
	body := "**Implementation Plan:** <?x\n**Plan Number:** ?>\n**Status:** upcoming\n\n## Objective\n\nc\n\n## Scope\n\nc\n\n## Components\n\nc\n\n## Verification\n\nc\n\n## Success Criteria\n\n- [ ] c\n"
	res := tmpl.CheckMarkdown(sc, builtIn, "[PLAN-00112-1] T", body)
	if !res.HasErrors() || !strings.Contains(res.Format(), "raw HTML") || !strings.Contains(res.Format(), "Implementation Plan") {
		t.Errorf("want a raw HTML error naming the Implementation Plan metadata value:\n%s", res.Format())
	}
}

// TestLoadSchemaOnlyUnknownFieldErrorsGetTheUnknownFieldHint verifies a YAML
// syntax error or a wrong value type is reported as the YAML error alone: the
// "misspelt key" advice is for unknown fields, and would send the reader
// hunting for a key that does not exist.
func TestLoadSchemaOnlyUnknownFieldErrorsGetTheUnknownFieldHint(t *testing.T) {
	const head = "type: impl-plan\ntitle_prefix: \"[PLAN-XXXXX] <T>\"\n"
	for name, yml := range map[string]string{
		"syntax error":  head + "sections:\n  - id: a\n   heading: A\n",
		"type error":    head + "sections:\n  - id: a\n    heading: A\n    required: maybe\n",
		"unclosed flow": "type: [not, a, schema\n",
	} {
		_, err := tmpl.LoadSchema(overrideOrigin, []byte(yml))
		if err == nil {
			t.Fatalf("%s: want an error", name)
		}
		if strings.Contains(err.Error(), "does not know") || !strings.Contains(err.Error(), "parsing schema") || !strings.Contains(err.Error(), overrideOrigin) {
			t.Errorf("%s: want the YAML error alone, got %q", name, err)
		}
	}
	_, err := tmpl.LoadSchema(overrideOrigin, []byte(head+"requried: true\nmetdata: [A]\n"))
	if err == nil || strings.Count(err.Error(), "does not know") != 1 || !strings.Contains(err.Error(), "requried") || !strings.Contains(err.Error(), "metdata") {
		t.Errorf("several unknown fields: want both named and the hint once, got %v", err)
	}
}

// TestLoadSchemaRejectsWhatItsOwnRenderWouldFail covers overrides that load
// but then render something the markdown check refuses: headings equal once
// trimmed, an empty metadata key, and a second YAML document that would be
// ignored silently.
func TestLoadSchemaRejectsWhatItsOwnRenderWouldFail(t *testing.T) {
	const head = "type: impl-plan\ntitle_prefix: \"[PLAN-XXXXX] <T>\"\n"
	sec := func(id, heading string) string {
		return fmt.Sprintf("  - id: %s\n    heading: %s\n    required: true\n", id, heading)
	}
	cases := []struct {
		name string
		yaml string
		want []string
	}{
		{"headings equal once trimmed", head + "sections:\n" + sec("a", `" A"`) + sec("b", "A"),
			[]string{overrideOrigin, `"heading"`, "twice", "Rename"}},
		{"empty metadata key", head + "metadata: [Author, \"\"]\n",
			[]string{overrideOrigin, "metadata", "empty", "Remove"}},
		{"second document", head + "sections:\n" + sec("a", "A") + "---\ntype: other\n",
			[]string{overrideOrigin, "second YAML document", "---"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := tmpl.LoadSchema(overrideOrigin, []byte(c.yaml))
			if err == nil {
				t.Fatal("want a schema error, got nil")
			}
			for _, w := range c.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q missing %q", err, w)
				}
			}
		})
	}
}

// TestLoadSchemaUnknownFieldErrorsSayWhereNotWhichGoType verifies an
// unknown-field error says where the field is, not which Go type failed to
// decode it.
func TestLoadSchemaUnknownFieldErrorsSayWhereNotWhichGoType(t *testing.T) {
	const head = "type: impl-plan\ntitle_prefix: \"[PLAN-XXXXX] <T>\"\n"
	for name, c := range map[string]struct{ yaml, want string }{
		"top level":  {head + "requried: true\n", "requried not found at the top level"},
		"in section": {head + "sections:\n  - id: a\n    heading: A\n    requried: true\n", "requried not found in a section"},
		"validation": {head + "validation:\n  rulez: []\n", "rulez not found under validation"},
	} {
		_, err := tmpl.LoadSchema(overrideOrigin, []byte(c.yaml))
		if err == nil || !strings.Contains(err.Error(), c.want) || strings.Contains(err.Error(), "template.") {
			t.Errorf("%s: want %q and no Go type name, got %v", name, c.want, err)
		}
	}
}
