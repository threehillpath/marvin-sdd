package template_test

import (
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
