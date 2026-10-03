package template_test

import (
	"fmt"
	"strings"
	"testing"

	tmpl "threehillpath.com/marvin-sdd/tool/internal/template"
)

// schema loads a built-in schema's YAML bytes by name, failing the test if
// it isn't embedded.
func schema(t *testing.T, name string) []byte {
	t.Helper()
	data, ok := tmpl.DefaultSchema(name)
	if !ok {
		t.Fatalf("no built-in schema embedded for %q", name)
	}
	return data
}

// entries wraps content blocks as unnamed section entries.
func entries(blocks ...string) []tmpl.Entry {
	var out []tmpl.Entry
	for _, b := range blocks {
		out = append(out, tmpl.Entry{Content: b})
	}
	return out
}

// meta builds the metadata half of a section map from key/value pairs.
func meta(kv ...string) map[string]tmpl.Field {
	out := map[string]tmpl.Field{}
	for i := 0; i+1 < len(kv); i += 2 {
		out[kv[i]] = tmpl.Field{Value: kv[i+1]}
	}
	return out
}

// render renders m with the named built-in schema, failing the test on any
// finding.
func render(t *testing.T, name string, m *tmpl.SectionMap) string {
	t.Helper()
	sc := loadBuiltIn(t, name)
	out, res := tmpl.Render(sc, builtIn, m)
	if len(res.Findings) != 0 {
		t.Fatalf("Render returned findings:\n%s", res.Format())
	}
	return out
}

// TestImplPlanNumberedSections asserts that two component sections and one
// verification_steps section produce ## 1. / ## 2. / ## 3. headings
// with the metadata block above them.
func TestImplPlanNumberedSections(t *testing.T) {
	m := &tmpl.SectionMap{
		Source: tmpl.SourceYAML,
		Title:  "[PLAN-00014] Build something",
		Metadata: meta(
			"Objective", "Build something",
			"Architecture Plan", "#14",
			"Source Issue", "#14",
			"Author", "Test",
			"Status", "Upcoming",
			"Last Updated", "2026-01-01"),
		Sections: map[string][]tmpl.Entry{
			"scope": entries("**Includes:** stuff\n\n**Does NOT include:** nothing"),
			"component": {
				{Name: "First Component", Content: "First component content"},
				{Name: "Second Component", Content: "Second component content"},
			},
			"verification_steps": entries("Verify step body"),
			"design_notes":       entries("Some design notes"),
			"success_criteria":   entries("- [ ] Passes"),
		},
	}
	out := render(t, "impl-plan", m)

	// Metadata block
	if !strings.Contains(out, "**Objective:** Build something") {
		t.Error("missing Objective metadata line")
	}

	// Numbered component headings — assert full heading text, not just prefix.
	if !strings.Contains(out, "## 1. First Component") {
		t.Error("missing ## 1. First Component heading")
	}
	if !strings.Contains(out, "## 2. Second Component") {
		t.Error("missing ## 2. Second Component heading")
	}
	// Verification steps continues the ordinal from components (2 components → ## 3.)
	// and renders its literal schema heading, not its first content line.
	if !strings.Contains(out, "## 3. Verification Steps") {
		t.Errorf("missing ## 3. Verification Steps heading for verification_steps, got:\n%s", out)
	}
	if strings.Contains(out, "<Component or Layer Name>") {
		t.Error("the component placeholder heading must never be rendered")
	}

	// Non-numbered sections use plain ## headings
	for _, h := range []string{"## Scope", "## Design Notes", "## Success Criteria"} {
		if !strings.Contains(out, h) {
			t.Errorf("missing %s heading", h)
		}
	}
}

// TestImplPhaseOptionalTDDEntryPoint verifies that omitting the optional
// tdd_entry_point section in impl-phase produces no heading for it and exits 0.
func TestImplPhaseOptionalTDDEntryPoint(t *testing.T) {
	m := phaseMap()
	// tdd_entry_point is not in phaseMap, so it is omitted.
	out := render(t, "impl-phase", m)

	if strings.Contains(out, "TDD Entry Point") {
		t.Error("output should not contain TDD Entry Point heading when section is omitted")
	}
	if !strings.Contains(out, "## Objective") {
		t.Error("missing ## Objective heading")
	}
}

// TestArchPlanMetadataKey verifies that rendering an arch-plan with a "Date" metadata
// entry emits "**Date:**" in the output.
func TestArchPlanMetadataKey(t *testing.T) {
	m := archMap(t)
	m.Metadata["Date"] = tmpl.Field{Value: "2026-01-01"}
	out := render(t, "arch-plan", m)

	if !strings.Contains(out, "**Date:** 2026-01-01") {
		t.Errorf("expected the Date metadata line in output, got:\n%s", out)
	}
}

// skeleton returns the named built-in schema's skeleton.
func skeleton(t *testing.T, name string) string {
	t.Helper()
	return tmpl.Skeleton(loadBuiltIn(t, name))
}

// TestSkeletonIsAYAMLDraft verifies that Skeleton emits an empty YAML draft:
// a double-quoted title placeholder from title_prefix, every metadata key as
// "", and every section key, with no comments.
func TestSkeletonIsAYAMLDraft(t *testing.T) {
	out := skeleton(t, "impl-plan")

	if !strings.HasPrefix(out, "title: \"[PLAN-XXXXX] <Title>\"\n") {
		t.Errorf("want a double-quoted title placeholder, got:\n%s", out)
	}
	for _, key := range []string{"Objective", "Architecture Plan", "Source Issue", "Author", "Status", "Last Updated"} {
		if want := "\n  " + key + ": \"\"\n"; !strings.Contains(out, want) {
			t.Errorf("want metadata line %q in skeleton, got:\n%s", want, out)
		}
	}
	for _, want := range []string{
		"\n  scope: |\n",
		"\n  component:\n    - name: \"\"\n      content: |\n",
		"\n  verification_steps:\n    - |\n",
		"\n  design_notes: |\n",
		"\n  success_criteria: |\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("want %q in skeleton, got:\n%s", want, out)
		}
	}
	// A skeleton is comment-free: drafts take no YAML comments, and guidance
	// is printed separately by Guidance.
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "#") {
			t.Errorf("skeleton must not contain comments, got line %q", line)
		}
	}
	if strings.Contains(out, "Two sub-sections") {
		t.Errorf("guidance text belongs in Guidance, not the skeleton:\n%s", out)
	}
	// The old markdown skeleton is gone.
	if strings.Contains(out, "\n## ") || strings.Contains(out, "**Objective:**") {
		t.Errorf("skeleton must be YAML, not markdown:\n%s", out)
	}
}

// TestSkeletonLoadsAsADraft verifies that a skeleton is itself a loadable
// draft: no loader findings, and the check then reports what is still empty
// instead of a structural problem.
func TestSkeletonLoadsAsADraft(t *testing.T) {
	for _, name := range []string{"arch-plan", "impl-plan", "impl-phase", "quick-task"} {
		t.Run(name, func(t *testing.T) {
			sc := loadBuiltIn(t, name)
			m, fs := tmpl.LoadDraft(sc, []byte(tmpl.Skeleton(sc)))
			if len(fs) != 0 || m == nil {
				t.Fatalf("skeleton must load cleanly, got %+v", fs)
			}
			res := tmpl.Check(sc, builtIn, m)
			if !res.HasErrors() {
				t.Fatalf("an empty skeleton cannot conform:\n%s", res.Format())
			}
		})
	}
}

// TestQuickTaskSkeletonKeysInOrder verifies that quick-task's skeleton
// includes all six section keys in the schema's order.
func TestQuickTaskSkeletonKeysInOrder(t *testing.T) {
	out := skeleton(t, "quick-task")

	keys := []string{
		"\n  problem_statement: |",
		"\n  scope: |",
		"\n  technical_analysis: |",
		"\n  tdd_entry_point: |",
		"\n  implementation_notes: |",
		"\n  success_criteria: |",
	}

	lastIdx := -1
	for _, k := range keys {
		idx := strings.Index(out, k)
		if idx == -1 {
			t.Fatalf("missing key %q in skeleton output:\n%s", k, out)
		}
		if idx <= lastIdx {
			t.Fatalf("key %q out of order in skeleton output:\n%s", k, out)
		}
		lastIdx = idx
	}
}

// quickTaskMap returns a conformant quick-task section map.
func quickTaskMap() *tmpl.SectionMap {
	return &tmpl.SectionMap{
		Source: tmpl.SourceYAML,
		Title:  "[TASK-00091] Fix the thing",
		Metadata: meta(
			"Source Issue", "#91",
			"Task Number", "TASK-00091",
			"Author", "Test",
			"Status", "Upcoming",
			"Date", "2026-01-01"),
		Sections: map[string][]tmpl.Entry{
			"problem_statement":    entries("The problem"),
			"scope":                entries("**Includes:** x"),
			"technical_analysis":   entries("Analysis"),
			"tdd_entry_point":      entries("What: ...\nWhere: ...\nPasses when: ..."),
			"implementation_notes": entries("Notes"),
			"success_criteria":     entries("- [ ] Done"),
		},
	}
}

// wantRefused asserts that Render rendered nothing and reported exactly one
// error at loc whose message contains every substring in want.
func wantRefused(t *testing.T, name string, m *tmpl.SectionMap, loc string, want ...string) {
	t.Helper()
	out, res := tmpl.Render(loadBuiltIn(t, name), builtIn, m)
	if out != "" {
		t.Errorf("Render must produce no output when the check fails, got:\n%s", out)
	}
	wantOne(t, res, tmpl.SeverityError, loc, want...)
}

// TestQuickTaskRenderMissingTDDEntryPoint verifies that Render refuses when
// the required tdd_entry_point section is omitted.
func TestQuickTaskRenderMissingTDDEntryPoint(t *testing.T) {
	m := quickTaskMap()
	delete(m.Sections, "tdd_entry_point")
	wantRefused(t, "quick-task", m, "section:tdd_entry_point", `"TDD Entry Point"`, "Add")
}

// TestQuickTaskRenderMissingTechnicalAnalysis verifies that Render refuses
// when the required technical_analysis section is omitted.
func TestQuickTaskRenderMissingTechnicalAnalysis(t *testing.T) {
	m := quickTaskMap()
	delete(m.Sections, "technical_analysis")
	wantRefused(t, "quick-task", m, "section:technical_analysis", `"Technical Analysis"`, "Add")
}

// TestRenderRequiredSectionMissing verifies that omitting a required section
// returns a finding.
func TestRenderRequiredSectionMissing(t *testing.T) {
	m := phaseMap()
	delete(m.Sections, "objective")
	wantRefused(t, "impl-phase", m, "section:objective", `"Objective"`, "Add")
}

// TestRenderRefusesOnAnyCheckError verifies that Render runs the full check,
// not just the required-section rule it used to own: a wrong title kind
// refuses even though every section is present.
func TestRenderRefusesOnAnyCheckError(t *testing.T) {
	m := phaseMap()
	m.Title = "[PLAN-00112] Not a phase title"
	wantRefused(t, "impl-phase", m, "title", "impl", "phase")
}

// TestRenderNonRepeatableWithTwoEntriesRefused verifies the old
// "not repeatable" rule now lives only in the check.
func TestRenderNonRepeatableWithTwoEntriesRefused(t *testing.T) {
	m := phaseMap()
	m.Sections["scope"] = entries("a", "b")
	wantRefused(t, "impl-phase", m, "section:scope", `"Scope"`, "not repeatable")
}

// TestRenderMetadataAndHeadingsInSchemaOrder is the TDD entry point: a draft
// written with its metadata and sections out of order renders in schema
// order.
func TestRenderMetadataAndHeadingsInSchemaOrder(t *testing.T) {
	draft := `title: "[PLAN-00112-1] Add the thing"
metadata:
  Status: "Upcoming"
  Plan Number: "PLAN-00112"
  Implementation Plan: "#132 ([PLAN-00112])"
sections:
  success_criteria: |
    - [ ] Done
  verification: |
    go test ./...
  components: |
    Stuff.
  scope: |
    In scope.
  objective: |
    Do the thing.
`
	sc := loadBuiltIn(t, "impl-phase")
	m, fs := tmpl.LoadDraft(sc, []byte(draft))
	if len(fs) != 0 {
		t.Fatalf("LoadDraft findings: %+v", fs)
	}
	out := render(t, "impl-phase", m)
	want := []string{
		"**Implementation Plan:** #132 ([PLAN-00112])\n**Plan Number:** PLAN-00112\n**Status:** Upcoming\n",
		"\n## Objective\n\nDo the thing.\n",
		"\n## Scope\n\nIn scope.\n",
		"\n## Components\n\nStuff.\n",
		"\n## Verification\n\ngo test ./...\n",
		"\n## Success Criteria\n\n- [ ] Done\n",
	}
	last := -1
	for _, w := range want {
		idx := strings.Index(out, w)
		if idx < 0 {
			t.Fatalf("missing %q in:\n%s", w, out)
		}
		if idx <= last {
			t.Fatalf("%q out of schema order in:\n%s", w, out)
		}
		last = idx
	}
	if strings.Contains(out, "\n\n\n") {
		t.Errorf("content's trailing newline must not add a blank line:\n%s", out)
	}
}

// TestRenderDraftWithH2InContentIsRefused covers the last TDD entry point
// case: a "## " line in section content outside a fence is one check error
// on the YAML path, and Render refuses.
func TestRenderDraftWithH2InContentIsRefused(t *testing.T) {
	sc := loadBuiltIn(t, "impl-phase")
	d := strings.Replace(phaseDraft, "    - one\n", "    - one\n    ## Sub\n", 1)
	m, fs := tmpl.LoadDraft(sc, []byte(d))
	if len(fs) != 0 {
		t.Fatalf("LoadDraft findings: %+v", fs)
	}
	wantRefused(t, "impl-phase", m, "section:scope", `"## Sub"`, "line 3 of the section", "###")
}

// TestRenderNamedEntryNameCannotChangeStructure verifies that an entry name
// with a line break is refused, at its own line, by the check that Render runs.
func TestRenderNamedEntryNameCannotChangeStructure(t *testing.T) {
	m := implPlanMap()
	m.Sections["component"] = []tmpl.Entry{{Name: "Foo\n## Injected", Content: "body", Line: 12}}
	wantRefused(t, "impl-plan", m, "section:component", `"Foo\n## Injected"`, "line break", "single line", `"component"`)
	_, res := tmpl.Render(loadBuiltIn(t, "impl-plan"), builtIn, m)
	if res.Findings[0].Line != 12 {
		t.Errorf("line = %d, want 12", res.Findings[0].Line)
	}
}

// TestRenderKeepsFencedHeadingsInContent verifies that sub-structure inside
// fences renders untouched.
func TestRenderKeepsFencedHeadingsInContent(t *testing.T) {
	m := phaseMap()
	m.Sections["scope"] = entries("```md\n## example\n```\n\n### Sub")
	out := render(t, "impl-phase", m)
	if !strings.Contains(out, "```md\n## example\n```\n\n### Sub\n") {
		t.Errorf("fenced content was altered:\n%s", out)
	}
}

// TestGuidancePrintsSectionsAndRules verifies the plain-text guidance output:
// per section in schema order its heading, whether it is required, whether
// it repeats or is numbered, and its guidance, then the draft-writing rules.
func TestGuidancePrintsSectionsAndRules(t *testing.T) {
	out := tmpl.Guidance(loadBuiltIn(t, "impl-plan"))

	last := -1
	for _, w := range []string{
		"schema: impl-plan",
		"Scope (required)",
		"scope: a single | block",
		"Two sub-sections: **Includes**",
		"Component or Layer Name (required, repeatable, numbered, named)",
		"component: a list of entries with name: and content: |",
		"Verification Steps (required, repeatable, numbered)",
		"verification_steps: a list of | blocks",
		"Design Notes",
		"Success Criteria",
	} {
		idx := strings.Index(out, w)
		if idx < 0 {
			t.Fatalf("guidance missing %q:\n%s", w, out)
		}
		if idx <= last {
			t.Fatalf("%q out of schema order:\n%s", w, out)
		}
		last = idx
	}
	for _, rule := range []string{
		"double quotes",
		"| block",
		"###",
		"never ##",
		"no # comments",
		"--- or ... at column 0",
		"blank line before a --- rule",
		"never --- or === directly under text",
		"close every code fence inside the same section",
		"never # or ##, including inside list items and quotes",
		"inside a list item, indent the fence, every code line and the closing fence at least as far as the opening fence",
		"no line may start with an HTML tag, <? or <!",
		"no raw HTML",
		"`<!--`, `<details>`, `<pre>`, `<script>`, `<style>` and `<textarea>`",
	} {
		if !strings.Contains(out, rule) {
			t.Errorf("guidance missing the rule %q:\n%s", rule, out)
		}
	}
	if strings.Contains(out, "\n  #") {
		t.Errorf("guidance is plain text, not YAML comments:\n%s", out)
	}
}

// wantRefusedAt asserts Render refused (no body) and that some finding is
// located at loc, names "line N of the section" (or, for loc starting with
// "metadata:", the metadata key) and contains every substring in want.
func wantRefusedAt(t *testing.T, name string, m *tmpl.SectionMap, loc string, line int, want ...string) {
	t.Helper()
	out, res := tmpl.Render(loadBuiltIn(t, name), builtIn, m)
	if out != "" {
		t.Errorf("Render must produce no body when verification fails, got:\n%s", out)
	}
	if !res.HasErrors() {
		t.Fatalf("Render must report an error")
	}
	if line > 0 {
		want = append(want, fmt.Sprintf("line %d of the section", line))
	}
	for _, f := range res.Findings {
		if f.Location != loc {
			continue
		}
		ok := true
		for _, w := range want {
			if !strings.Contains(f.Message, w) {
				ok = false
			}
		}
		if ok && f.Severity == tmpl.SeverityError {
			return
		}
	}
	t.Fatalf("no error at %s containing %q:\n%s", loc, want, res.Format())
}

func scopeMap(content string) *tmpl.SectionMap {
	m := phaseMap()
	m.Sections["scope"] = []tmpl.Entry{{Content: content, Line: 9}}
	return m
}

// TestRenderRefusesHTMLBlocksOfEveryType covers blocks the line scanner never
// sees: processing instructions, CDATA, declarations and a lone tag, all of
// which swallow what follows or render nothing.
func TestRenderRefusesHTMLBlocksOfEveryType(t *testing.T) {
	cases := []struct {
		name, content string
		line          int
		quote         string
	}{
		{"unclosed processing instruction", "Every file must start with\n<?php declare(strict_types=1);", 2, "<?php declare(strict_types=1);"},
		{"cdata", "intro\n\n<![CDATA[ stuff", 3, "<![CDATA["},
		{"doctype without >", "<!DOCTYPE html\nbody", 1, "<!DOCTYPE html"},
		{"div", "text\n\n<div>\nx\n</div>", 3, "<div>"},
		{"a lone inline tag becomes a block", "text\n\n<kbd>\n\nmore", 3, "<kbd>"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wantRefusedAt(t, "impl-phase", scopeMap(c.content), "section:scope", c.line,
				"HTML block", c.quote, "swallow", "backticks", "remove")
		})
	}
}

func TestRenderRefusesInlineHTMLThatSlipsPastTheScanner(t *testing.T) {
	// A list-item fence ended early by a dedented line: the fence that GitHub
	// then opens at line 4 swallows every later section.
	for name, content := range map[string]string{
		"go build":    "1. Build:\n   ```bash\ngo build ./...\n   ```\n2. Test.",
		"run details": "1. Build:\n   ```bash\nrun <details>\n   ```\n2. Test.",
	} {
		t.Run(name, func(t *testing.T) {
			wantRefusedAt(t, "impl-phase", scopeMap(content), "section:scope", 4,
				"fenced code block", "swallow", `"## Components"`, "indent every line")
		})
	}
	t.Run("run details also reports the inline tag", func(t *testing.T) {
		wantRefusedAt(t, "impl-phase", scopeMap("1. Build:\n   ```bash\nrun <details>\n   ```\n2. Test."), "section:scope", 3,
			"raw HTML", `"<details>"`, "backticks")
	})
}

func TestRenderRefusesExtraHeadingsInTheParsedBody(t *testing.T) {
	cases := []struct{ name, content string }{
		{"level 1 heading", "intro\n\n# Big"},
		{"heading in a list", "- item\n- # nested"},
		{"heading in a quote", "> ## quoted"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wantRefusedAt(t, "impl-phase", scopeMap(c.content), "section:scope", strings.Count(c.content, "\n")+1,
				"heading", "rendered body", "###")
		})
	}
}

func TestRenderRefusesRawHTMLInAMetadataValue(t *testing.T) {
	m := phaseMap()
	m.Metadata["Status"] = tmpl.Field{Value: "Use ``` for fences and <details> blocks, see `x`.", Line: 4}
	wantRefusedAt(t, "impl-phase", m, "metadata:Status", 0, `"<details>"`, "backticks")
}

func TestRenderAcceptsBenignMarkdown(t *testing.T) {
	for name, content := range map[string]string{
		"fenced block":           "```go\nfunc main() {}\n```",
		"list with proper fence": "1. Build:\n   ```bash\n   go build ./...\n   ```\n2. Test.",
		"kbd":                    "Press <kbd>Ctrl</kbd>+C and <br> see <https://example.com>.",
		"table":                  "| a | b |\n|---|---|\n| 1 | 2 |",
		"task list":              "- [ ] one\n- [x] two",
		"double backtick span":   "Use ``<details>`` here.",
		"single backtick span":   "Use `<details>` here.",
		"h3 and bold":            "### Sub\n\n**bold** text",
	} {
		t.Run(name, func(t *testing.T) {
			out, res := tmpl.Render(loadBuiltIn(t, "impl-phase"), builtIn, scopeMap(content))
			if res.HasErrors() || out == "" {
				t.Fatalf("want a clean render, got:\n%s", res.Format())
			}
		})
	}
}

func TestRenderRefusesEmptyHeadingsAtTheirLine(t *testing.T) {
	wantRefusedAt(t, "impl-phase", scopeMap("intro\n\n#"), "section:scope", 3, "heading", `"#"`, `\#`, "###")
	wantRefusedAt(t, "impl-phase", scopeMap("- #"), "section:scope", 1, "heading", `"- #"`, `\#`, "###")
}

func TestRenderRefusesLoneCarriageReturnInContent(t *testing.T) {
	m := scopeMap("a\r## X")
	wantRefusedAt(t, "impl-phase", m, "section:scope", 1, `"Scope"`, "carriage return", "replace the carriage return with a line break")
}
