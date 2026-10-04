package template_test

import (
	"fmt"
	"strings"
	"testing"

	tmpl "threehillpath.com/marvin-sdd/tool/internal/template"
)

const phaseBody = `**Implementation Plan:** #132 ([PLAN-00112])
**Plan Number:** PLAN-00112
**Status:** Upcoming

## Objective

Do it.

## Scope

Includes things.

## Components

Stuff.

## Verification

go test ./...

## Success Criteria

- [ ] Done
`

// phaseBodyWithout returns phaseBody with the "## <heading>" section removed.
func phaseBodyWithout(heading string) string {
	parts := strings.Split(phaseBody, "\n## ")
	var keep []string
	for _, p := range parts {
		if !strings.HasPrefix(p, heading+"\n") {
			keep = append(keep, p)
		}
	}
	return strings.Join(keep, "\n## ")
}

func parseCheck(t *testing.T, name, title, body string) tmpl.Result {
	t.Helper()
	sc := loadBuiltIn(t, name)
	return tmpl.Check(sc, builtIn, tmpl.ParseMarkdown(sc, title, body))
}

// The markdown path gives the same verdict as Phase 2's YAML entry test: one
// error at section:verification.
func TestParseMarkdownMissingVerification(t *testing.T) {
	res := parseCheck(t, "impl-phase", "[PLAN-00112-1] X", phaseBodyWithout("Verification"))
	wantOne(t, res, tmpl.SeverityError, "section:verification", `"Verification"`, `Add a "## Verification" heading`)
	if got := parseCheck(t, "impl-phase", "[PLAN-00112-1] X", phaseBody); len(got.Findings) != 0 {
		t.Fatalf("complete body should have no findings:\n%s", got.Format())
	}
}

// A "## Foo" line inside a fenced block in Scope is content, not a section,
// and neither is one inside a longer or tilde fence.
func TestParseMarkdownFencedHeadingIsContent(t *testing.T) {
	for name, fenced := range map[string]string{
		"backticks": "```\n## Foo\n```",
		"tildes":    "~~~md\n## Foo\n~~~",
		"long":      "````\n```\n## Foo\n```\n````",
	} {
		t.Run(name, func(t *testing.T) {
			body := strings.Replace(phaseBody, "Includes things.", "Includes things.\n\n"+fenced, 1)
			sc := loadBuiltIn(t, "impl-phase")
			m := tmpl.ParseMarkdown(sc, "[PLAN-00112-1] X", body)
			if len(m.UnknownHeadings) != 0 {
				t.Fatalf("unknown headings: %+v", m.UnknownHeadings)
			}
			if got := m.Sections["scope"][0].Content; !strings.Contains(got, "## Foo") {
				t.Errorf("scope content lost the fenced heading: %q", got)
			}
			if res := tmpl.Check(sc, builtIn, m); len(res.Findings) != 0 {
				t.Fatalf("want no findings:\n%s", res.Format())
			}
		})
	}
}

const implPlanBody = `**Objective:** Build it
**Architecture Plan:** #14
**Source Issue:** #14
**Author:** Test
**Status:** Upcoming
**Last Updated:** 2026-01-01

## Scope

Stuff.

## 1. First Component

One.

## 2. Second Component

Two.

## 3. Verification Steps

Verify.

## Design Notes

Notes.

## Success Criteria

- [ ] Done
`

func TestParseMarkdownNumberedHeadings(t *testing.T) {
	sc := loadBuiltIn(t, "impl-plan")
	m := tmpl.ParseMarkdown(sc, "[PLAN-00014] Build", implPlanBody)

	comp := m.Sections["component"]
	if len(comp) != 2 || comp[0].Name != "First Component" || comp[1].Name != "Second Component" {
		t.Fatalf("component entries = %+v", comp)
	}
	if comp[0].Number != 1 || comp[1].Number != 2 || comp[0].Content != "One." {
		t.Errorf("component numbers/content = %+v", comp)
	}
	vs := m.Sections["verification_steps"]
	if len(vs) != 1 || vs[0].Name != "" || vs[0].Number != 3 || vs[0].Content != "Verify." {
		t.Fatalf("verification_steps = %+v", vs)
	}
	if len(m.UnknownHeadings) != 0 {
		t.Errorf("unknown headings: %+v", m.UnknownHeadings)
	}
	if res := tmpl.Check(sc, builtIn, m); len(res.Findings) != 0 {
		t.Fatalf("want no findings:\n%s", res.Format())
	}
}

// A numbered heading no section claims is unknown, not forced into the named
// section when the schema has none (impl-phase has no numbered sections).
func TestParseMarkdownNumberedHeadingWithoutNamedSectionIsUnknown(t *testing.T) {
	sc := loadBuiltIn(t, "impl-phase")
	m := tmpl.ParseMarkdown(sc, "[PLAN-00112-1] X", phaseBody+"\n## 1. Extra\n\nx\n")
	if len(m.UnknownHeadings) != 1 || m.UnknownHeadings[0].Text != "1. Extra" {
		t.Fatalf("unknown headings = %+v", m.UnknownHeadings)
	}
}

// Literal headings match case-sensitively.
func TestParseMarkdownLiteralHeadingIsCaseSensitive(t *testing.T) {
	sc := loadBuiltIn(t, "impl-phase")
	m := tmpl.ParseMarkdown(sc, "[PLAN-00112-1] X", strings.Replace(phaseBody, "## Scope", "## scope", 1))
	if len(m.UnknownHeadings) != 1 || m.UnknownHeadings[0].Text != "scope" || len(m.Sections["scope"]) != 0 {
		t.Fatalf("unknown = %+v, scope = %+v", m.UnknownHeadings, m.Sections["scope"])
	}
}

// fullMap returns a conformant YAML-source map for sc with every section
// filled, each with a "###" sub-heading and a fenced "## " line.
func fullMap(t *testing.T, sc *tmpl.Schema) *tmpl.SectionMap {
	t.Helper()
	titles := map[string]string{
		"arch-plan":  "[PLAN-00112-ARCH] X",
		"impl-plan":  "[PLAN-00112] X",
		"impl-phase": "[PLAN-00112-1] X",
		"quick-task": "[TASK-00091] X",
	}
	m := &tmpl.SectionMap{Source: tmpl.SourceYAML, Title: titles[sc.Type], Metadata: map[string]tmpl.Field{}, Sections: map[string][]tmpl.Entry{}}
	for _, key := range sc.Metadata {
		v := "value"
		switch key {
		case "Plan Number":
			v = "PLAN-00112"
		case "Task Number":
			v = "TASK-00091"
		case "Source Issue", "Architecture Plan", "Implementation Plan":
			v = "#112"
		}
		m.Metadata[key] = tmpl.Field{Value: v}
	}
	content := func(id string, i int) string {
		return fmt.Sprintf("Intro for %s %d.\n\n### Sub heading\n\n- item\n\n```md\n## fenced heading\n```\n\nEnd.", id, i)
	}
	for _, sec := range sc.Sections {
		n := 1
		if sec.Repeatable {
			n = 2
		}
		for i := 1; i <= n; i++ {
			e := tmpl.Entry{Content: content(sec.ID, i)}
			if sec.Numbered && sec.Named != nil && *sec.Named {
				e.Name = fmt.Sprintf("Entry %d", i)
			}
			m.Sections[sec.ID] = append(m.Sections[sec.ID], e)
		}
	}
	return m
}

// Rendering a valid map and parsing it back gives an equivalent map with no
// findings, for each built-in schema.
func TestMarkdownRoundTrip(t *testing.T) {
	for _, name := range []string{"arch-plan", "impl-plan", "impl-phase", "quick-task"} {
		t.Run(name, func(t *testing.T) {
			sc := loadBuiltIn(t, name)
			want := fullMap(t, sc)
			body, res := tmpl.Render(sc, builtIn, want)
			if len(res.Findings) != 0 {
				t.Fatalf("Render:\n%s", res.Format())
			}
			got := tmpl.ParseMarkdown(sc, want.Title, body)
			if res := tmpl.Check(sc, builtIn, got); len(res.Findings) != 0 {
				t.Fatalf("Check of parsed body:\n%s\nbody:\n%s", res.Format(), body)
			}
			if got.Title != want.Title {
				t.Errorf("title = %q", got.Title)
			}
			for key, f := range want.Metadata {
				if got.Metadata[key].Value != f.Value {
					t.Errorf("metadata %q = %q, want %q", key, got.Metadata[key].Value, f.Value)
				}
			}
			if len(got.Metadata) != len(want.Metadata) {
				t.Errorf("metadata keys = %v", got.Metadata)
			}
			for _, sec := range sc.Sections {
				w, g := want.Sections[sec.ID], got.Sections[sec.ID]
				if len(w) != len(g) {
					t.Fatalf("section %s: %d entries, want %d", sec.ID, len(g), len(w))
				}
				for i := range w {
					if g[i].Name != w[i].Name || strings.TrimRight(g[i].Content, " \t\n") != strings.TrimRight(w[i].Content, " \t\n") {
						t.Errorf("section %s entry %d = %+v, want %+v", sec.ID, i, g[i], w[i])
					}
				}
			}
			if len(got.UnknownHeadings) != 0 {
				t.Errorf("unknown headings: %+v", got.UnknownHeadings)
			}
		})
	}
}

// Text before the first "##" that is not metadata is ignored without a finding.
func TestParseMarkdownPreambleIsIgnored(t *testing.T) {
	body := "> **Revised 2026-01-01:** scope narrowed.\n\nSome intro text.\n\n" + phaseBody
	if res := parseCheck(t, "impl-phase", "[PLAN-00112-1] X", body); len(res.Findings) != 0 {
		t.Fatalf("want no findings:\n%s", res.Format())
	}
}

// A "**Key:** value" line in section content is content, not metadata, even
// when it repeats or invents a key.
func TestParseMarkdownMetadataOnlyAboveFirstHeading(t *testing.T) {
	body := strings.Replace(phaseBody, "Includes things.", "**Status:** Done\n**Extra:** x\nIncludes things.", 1)
	sc := loadBuiltIn(t, "impl-phase")
	m := tmpl.ParseMarkdown(sc, "[PLAN-00112-1] X", body)
	if got := m.Metadata["Status"].Value; got != "Upcoming" {
		t.Errorf("Status = %q, want the header value", got)
	}
	if _, ok := m.Metadata["Extra"]; ok {
		t.Errorf("a metadata-looking line in a section became metadata: %v", m.Metadata)
	}
	if got := m.Sections["scope"][0].Content; !strings.HasPrefix(got, "**Status:** Done\n**Extra:** x") {
		t.Errorf("scope content = %q", got)
	}
	if res := tmpl.Check(sc, builtIn, m); len(res.Findings) != 0 {
		t.Fatalf("want no findings:\n%s", res.Format())
	}
}

// Content under an unknown heading is kept with its heading line, so Check
// applies the same content guards to it as to a known section.
func TestParseMarkdownUnknownHeadingKeepsContentAndLine(t *testing.T) {
	body := phaseBody + "\n## Appendix\n\nSome notes.\n"
	m := tmpl.ParseMarkdown(loadBuiltIn(t, "impl-phase"), "[PLAN-00112-1] X", body)
	if len(m.UnknownHeadings) != 1 {
		t.Fatalf("unknown headings = %+v", m.UnknownHeadings)
	}
	h := m.UnknownHeadings[0]
	if h.Text != "Appendix" || h.Content != "Some notes." || h.Line != strings.Count(phaseBody, "\n")+2 {
		t.Errorf("unknown heading = %+v", h)
	}
}

func TestCheckMarkdownGuardsContentUnderUnknownHeading(t *testing.T) {
	cases := []struct {
		name, content string
		want          []string
	}{
		{"raw details", "<details>\nhidden\n</details>", []string{`"<details>"`, "backticks", `"## Appendix"`}},
		{"setext", "Title\n---", []string{"underline", `"## Appendix"`}},
		{"unclosed fence", "```\ncode", []string{"never closed", `"## Appendix"`}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := parseCheck(t, "impl-phase", "[PLAN-00112-1] X", phaseBody+"\n## Appendix\n\n"+c.content+"\n")
			var errs []tmpl.Finding
			for _, f := range res.Findings {
				if f.Severity == tmpl.SeverityError {
					errs = append(errs, f)
				}
			}
			if len(errs) != 1 {
				t.Fatalf("want exactly 1 error, got:\n%s", res.Format())
			}
			for _, w := range c.want {
				if !strings.Contains(errs[0].Message, w) {
					t.Errorf("message %q missing %q", errs[0].Message, w)
				}
			}
			if errs[0].Line == 0 {
				t.Errorf("error has no line: %+v", errs[0])
			}
		})
	}
}
