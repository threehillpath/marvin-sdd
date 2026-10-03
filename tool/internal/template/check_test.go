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
			[]string{overrideOrigin, `"type"`, "Add", "arch-plan", "impl-plan", "impl-phase", "quick-task"}},
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
		})
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
	wantOne(t, res, tmpl.SeverityWarning, "draft", `"## Extras"`, "Rename")
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
	wantOne(t, check(t, "impl-phase", m), tmpl.SeverityWarning, "section:objective", `"Objective"`, "Move")
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
