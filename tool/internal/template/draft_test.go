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

func TestLoadDraftRepeatedKeyIsFinding(t *testing.T) {
	t.Run("repeatable section written once per entry", func(t *testing.T) {
		d := patch(t, planDraft, "  verification_steps:", `  component:
    - name: "Third one"
      content: |
        Body three.
  verification_steps:`)
		wantDraftFinding(t, "impl-plan", d, 19, `"component"`, "line 12", "one list")
	})
	t.Run("top level", func(t *testing.T) {
		d := patch(t, phaseDraft, "sections:\n", "title: \"[PLAN-00112-1] Again\"\nsections:\n")
		wantDraftFinding(t, "impl-phase", d, 6, `"title"`, "line 1", "Remove")
	})
	t.Run("metadata", func(t *testing.T) {
		d := patch(t, phaseDraft, "  Plan Number:", "  Status: \"Done\"\n  Plan Number:")
		wantDraftFinding(t, "impl-phase", d, 4, `"Status"`, "line 3", "Remove")
	})
	t.Run("section", func(t *testing.T) {
		d := patch(t, phaseDraft, "  verification: |", "  scope: |\n    again\n  verification: |")
		wantDraftFinding(t, "impl-phase", d, 16, `"scope"`, "line 10", "Merge")
	})
	t.Run("named entry", func(t *testing.T) {
		d := patch(t, planDraft, `      content: |
        Body one.`, `      name: "again"
      content: |
        Body one.`)
		wantDraftFinding(t, "impl-plan", d, 14, `"name"`, "line 13", "Remove")
	})
}

func TestLoadDraftNonLiteralContentIsFinding(t *testing.T) {
	t.Run("folded scope", func(t *testing.T) {
		d := patch(t, phaseDraft, "scope: |\n    Includes:\n    - one\n    - two", "scope: >\n    Includes:\n    - one\n    - two")
		wantDraftFinding(t, "impl-phase", d, 10, `"scope"`, "folded", "reflow", "scope: |")
	})
	t.Run("plain scalar", func(t *testing.T) {
		d := patch(t, phaseDraft, "components: |\n    Stuff.", "components: Stuff.")
		wantDraftFinding(t, "impl-phase", d, 14, `"components"`, "plain", "components: |")
	})
	t.Run("double quoted", func(t *testing.T) {
		d := patch(t, phaseDraft, "components: |\n    Stuff.", `components: "Stuff.\n- a"`)
		wantDraftFinding(t, "impl-phase", d, 14, `"components"`, "double-quoted", "components: |")
	})
	t.Run("single quoted", func(t *testing.T) {
		d := patch(t, phaseDraft, "components: |\n    Stuff.", "components: 'Stuff.'")
		wantDraftFinding(t, "impl-phase", d, 14, `"components"`, "single-quoted", "components: |")
	})
	t.Run("named entry content", func(t *testing.T) {
		d := patch(t, planDraft, "content: |\n        Body one.", "content: Body one.")
		wantDraftFinding(t, "impl-plan", d, 14, `"content"`, `"First one"`, "plain", "content: |")
	})
	t.Run("list item", func(t *testing.T) {
		d := patch(t, planDraft, "- |\n      Run the tests.", "- >\n      Run the tests.")
		wantDraftFinding(t, "impl-plan", d, 20, `"verification_steps"`, "folded", "- |")
	})
}

func TestLoadDraftUnknownKeysAreFindings(t *testing.T) {
	t.Run("dedented line inside scope becomes a sibling key", func(t *testing.T) {
		d := patch(t, phaseDraft, "    - two\n", "    - two\n  Note: x\n")
		wantDraftFinding(t, "impl-phase", d, 14, `"Note"`, "sections", "under-indented", "block")
	})
	t.Run("top level", func(t *testing.T) {
		d := patch(t, phaseDraft, "sections:\n", "extra: 1\nsections:\n")
		wantDraftFinding(t, "impl-phase", d, 6, `"extra"`, "top level", "title, metadata, sections", "under-indented")
	})
	t.Run("metadata", func(t *testing.T) {
		d := patch(t, phaseDraft, "  Plan Number:", "  Colour: \"red\"\n  Plan Number:")
		wantDraftFinding(t, "impl-phase", d, 4, `"Colour"`, "metadata", "Implementation Plan, Plan Number, Status", "Remove")
	})
	t.Run("section id", func(t *testing.T) {
		d := patch(t, phaseDraft, "  verification: |", "  extras: |\n    x\n  verification: |")
		wantDraftFinding(t, "impl-phase", d, 16, `"extras"`, "sections", "objective, scope", "under-indented")
	})
	t.Run("named entry", func(t *testing.T) {
		d := patch(t, planDraft, "      content: |\n        Body one.", "      content: |\n        Body one.\n      Note: x")
		wantDraftFinding(t, "impl-plan", d, 16, `"Note"`, `"component"`, "name, content", "under-indented")
	})
}

func TestLoadDraftWrongNodeTypeIsFinding(t *testing.T) {
	cases := []struct {
		name  string
		typ   string
		draft string
		line  int
		want  []string
	}{
		{"draft is a list", "impl-phase", "- a\n- b\n", 1, []string{"the draft", "mapping", "list", "title:", "metadata:", "sections:"}},
		{"draft is empty", "impl-phase", "# nothing\n", 0, []string{"empty", "title:", "marvin template render impl-phase --skeleton"}},
		{"title is a list", "impl-phase", patch(t, phaseDraft, `title: "[PLAN-00112-1] Add the thing"`, "title:\n  - a"), 2, []string{`"title"`, "single line of text", "list"}},
		{"metadata is text", "impl-phase", patch(t, phaseDraft, "metadata:\n  Status: \"Upcoming\"\n  Plan Number: \"PLAN-00112\"\n  Implementation Plan: \"#132 ([PLAN-00112])\"\n", "metadata: nope\n"), 2, []string{`"metadata"`, "mapping", "text"}},
		{"metadata value is a list", "impl-phase", patch(t, phaseDraft, `Status: "Upcoming"`, "Status:\n    - a"), 4, []string{`"Status"`, "single line of text", "list"}},
		{"sections is a list", "impl-phase", "title: \"[PLAN-00112-1] X\"\nsections:\n  - a\n", 3, []string{`"sections"`, "mapping", "list"}},
		{"non-repeatable section is a list", "impl-phase", patch(t, phaseDraft, "objective: |\n    Do the thing.", "objective:\n    - Do the thing."), 9, []string{`"objective"`, "not repeatable", "list", "objective: |"}},
		{"repeatable section is text", "impl-plan", patch(t, planDraft, "verification_steps:\n    - |\n      Run the tests.", "verification_steps: |\n    Run the tests."), 19, []string{`"verification_steps"`, "repeatable", "list of | blocks", "- |"}},
		{"named section is text", "impl-plan", patch(t, planDraft, planDraft[strings.Index(planDraft, "  component:"):strings.Index(planDraft, "  verification_steps:")], "  component: |\n    Body.\n"), 12, []string{`"component"`, "list of entries", "name:", "content: |"}},
		{"named entry is text", "impl-plan", patch(t, planDraft, "    - name: \"First one\"\n      content: |\n        Body one.\n", "    - just text\n"), 13, []string{`"component"`, "mapping", "name:", "content: |"}},
		{"unnamed item is a mapping", "impl-plan", patch(t, planDraft, "    - |\n      Run the tests.", "    - step: 1"), 20, []string{`"verification_steps"`, "- |", "mapping"}},
		{"name is a list", "impl-plan", patch(t, planDraft, `name: "First one"`, "name:\n        - a"), 14, []string{`"name"`, "single line of text", "list"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wantDraftFinding(t, c.typ, c.draft, c.line, c.want...)
		})
	}
}

func TestLoadDraftParserErrorsGetTargetedFixes(t *testing.T) {
	t.Run("unquoted title starting with a bracket", func(t *testing.T) {
		d := patch(t, phaseDraft, `title: "[PLAN-00112-1] Add the thing"`, `title: [PLAN-00112-1] Add the thing`)
		wantDraftFinding(t, "impl-phase", d, 1, `"title"`, "YAML reads", "double quotes", `title: "[PLAN-00112-1] Add the thing"`)
	})
	t.Run("unescaped inner quote in the title", func(t *testing.T) {
		d := patch(t, phaseDraft, `title: "[PLAN-00112-1] Add the thing"`, `title: "[PLAN-00112-1] Add "validate" command"`)
		m, fs := loadDraft(t, "impl-phase", d)
		if m != nil || len(fs) != 1 {
			t.Fatalf("want one finding, got %+v", fs)
		}
		wantDraftFinding(t, "impl-phase", d, 1, `\"`, "single quotes", `title: "[PLAN-00112-1] Add \"validate\" command"`)
		if strings.Contains(fs[0].Message, "Wrap the whole title") {
			t.Errorf("an inner-quote error must give an escaping fix, not 'quote the title': %s", fs[0].Message)
		}
	})
	t.Run("unescaped inner quote in a metadata value", func(t *testing.T) {
		d := patch(t, phaseDraft, `Status: "Upcoming"`, `Status: "He said "hi" ok"`)
		wantDraftFinding(t, "impl-phase", d, 3, `\"`, "single quotes", `Status: "He said \"hi\" ok"`)
	})
	t.Run("unknown escape", func(t *testing.T) {
		d := patch(t, phaseDraft, `Status: "Upcoming"`, `Status: "Match \d{5}"`)
		wantDraftFinding(t, "impl-phase", d, 3, "unknown escape", `\\`, "single quotes")
	})
	t.Run("block scalar lines are not mistaken for quoted values", func(t *testing.T) {
		d := patch(t, phaseDraft, "    - two\n", "    - two\n    say \"a\" b\n") + "tabbed:\n\tx: 1\n"
		m, fs := loadDraft(t, "impl-phase", d)
		if m != nil || len(fs) != 1 {
			t.Fatalf("want one finding, got %+v", fs)
		}
		if strings.Contains(fs[0].Message, "unescaped") || !strings.Contains(fs[0].Message, "not valid YAML") {
			t.Errorf("a quote inside a | block is content, not the error: %s", fs[0].Message)
		}
	})
	t.Run("unmapped parser message uses the fallback", func(t *testing.T) {
		d := patch(t, phaseDraft, "    - two\n", "    - two\n") + "tabbed:\n\tx: 1\n"
		m, fs := loadDraft(t, "impl-phase", d)
		if m != nil || len(fs) != 1 {
			t.Fatalf("want one finding, got %+v", fs)
		}
		f := fs[0]
		if f.Line == 0 {
			t.Errorf("yaml.v3 gave a line; the finding must carry it: %s", f.Message)
		}
		for _, w := range []string{"not valid YAML", "tab", `"|" block scalars`, "double-quoted", "no tabs"} {
			if !strings.Contains(f.Message, w) {
				t.Errorf("fallback message %q missing %q", f.Message, w)
			}
		}
	})
	t.Run("parser message without a line says line unknown", func(t *testing.T) {
		d := patch(t, phaseDraft, `title: "[PLAN-00112-1] Add the thing"`, "title: {x} y")
		wantDraftFinding(t, "impl-phase", d, 0, "not valid YAML", "did not find expected key")
	})
}

func TestLoadDraftDocumentMarkersAreFindings(t *testing.T) {
	const fix = "indent it"
	t.Run("horizontal rule at column 0 inside content", func(t *testing.T) {
		d := patch(t, phaseDraft, "    - two\n", "    - two\n---\n    - [ ] Second criterion\n")
		wantDraftFinding(t, "impl-phase", d, 14, `"---"`, "document marker", "dropped", fix, "| block", "delete")
	})
	t.Run("document end marker", func(t *testing.T) {
		d := patch(t, phaseDraft, "    - two\n", "    - two\n...\n")
		wantDraftFinding(t, "impl-phase", d, 14, `"..."`, "document marker", fix, "delete")
	})
	t.Run("leading marker", func(t *testing.T) {
		wantDraftFinding(t, "impl-phase", "---\n"+phaseDraft, 1, `"---"`, "document marker", "delete")
	})
	t.Run("trailing whitespace after the marker", func(t *testing.T) {
		d := patch(t, phaseDraft, "    - two\n", "    - two\n---  \n")
		wantDraftFinding(t, "impl-phase", d, 14, `"---"`, "document marker")
	})
	t.Run("an indented rule is content", func(t *testing.T) {
		d := patch(t, phaseDraft, "    - two\n", "    - two\n\n    ---\n    more\n")
		if m, fs := loadDraft(t, "impl-phase", d); m == nil || len(fs) != 0 {
			t.Fatalf("an indented --- is block content: %+v", fs)
		}
	})
	t.Run("a second document is never silently dropped", func(t *testing.T) {
		d := phaseDraft + "--- extra\n"
		m, fs := loadDraft(t, "impl-phase", d)
		if m != nil || len(fs) != 1 {
			t.Fatalf("want exactly one finding, got %+v", fs)
		}
		for _, w := range []string{"second YAML document", "one document", "dropped"} {
			if !strings.Contains(fs[0].Message, w) {
				t.Errorf("message %q missing %q", fs[0].Message, w)
			}
		}
	})
}

// The wording every comment finding must carry: YAML would drop the text.
func TestLoadDraftCommentsAreFindings(t *testing.T) {
	const ambiguous = "indent it"
	t.Run("plain note line", func(t *testing.T) {
		d := patch(t, phaseDraft, "  objective: |", "  # note\n  objective: |")
		wantDraftFinding(t, "impl-phase", d, 8, `"# note"`, "YAML comment", "drop", ambiguous, "double quotes", "delete it", "don't take comments")
	})
	t.Run("dedented hash line after a block", func(t *testing.T) {
		d := patch(t, phaseDraft, "    - two\n", "    - two\n#113 is related\n")
		wantDraftFinding(t, "impl-phase", d, 14, `"#113 is related"`, "YAML comment", ambiguous, "| block", "delete it")
	})
	t.Run("trailing foot comment", func(t *testing.T) {
		d := phaseDraft + "  ### Extra\n"
		wantDraftFinding(t, "impl-phase", d, 20, `"### Extra"`, "YAML comment", ambiguous, "delete it")
	})
	t.Run("metadata value continued by a hash line", func(t *testing.T) {
		d := patch(t, phaseDraft, `  Status: "Upcoming"`, "  Status: Up\n  #coming soon")
		wantDraftFinding(t, "impl-phase", d, 4, `"#coming soon"`, "YAML comment", "double quotes", "delete it")
	})
	t.Run("entry name continued by a hash line", func(t *testing.T) {
		d := patch(t, planDraft, `name: "First one"`, "name: First\n      #112 one")
		wantDraftFinding(t, "impl-plan", d, 14, `"#112 one"`, "YAML comment", "double quotes", "delete it")
	})
	t.Run("comment after the | header", func(t *testing.T) {
		d := patch(t, phaseDraft, "objective: |", "objective: | # the goal")
		wantDraftFinding(t, "impl-phase", d, 8, `"# the goal"`, "YAML comment", "delete it")
	})
	t.Run("comment after a list marker", func(t *testing.T) {
		d := patch(t, planDraft, "- |\n      Run the tests.", "- | # first\n      Run the tests.")
		wantDraftFinding(t, "impl-plan", d, 20, `"# first"`, "YAML comment", "delete it")
	})
}
