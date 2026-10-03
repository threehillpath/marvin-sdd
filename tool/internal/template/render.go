// Package template renders plan issue bodies from YAML schema definitions.
// It assembles structure deterministically — ordered metadata, correct headings,
// repeatable/numbered section handling — but never authors section content.
package template

import (
	"embed"
	"fmt"
	"strings"

	"threehillpath.com/marvin-sdd/tool/internal/names"
)

// defaultSchemas embeds the plugin's built-in YAML schemas at compile time,
// so rendering never depends on skills/SHARED/templates/ (or any other path)
// being reachable on disk from the caller's working directory.
//
//go:embed schemas/*.yml
var defaultSchemas embed.FS

// DefaultSchema returns the plugin's built-in schema bytes for name (e.g.
// "impl-plan"), and false if no built-in schema exists under that name.
func DefaultSchema(name string) ([]byte, bool) {
	data, err := defaultSchemas.ReadFile("schemas/" + name + ".yml")
	if err != nil {
		return nil, false
	}
	return data, true
}

// SchemaSection mirrors the YAML section definition.
type SchemaSection struct {
	ID         string `yaml:"id"`
	Heading    string `yaml:"heading"`
	Required   bool   `yaml:"required"`
	Repeatable bool   `yaml:"repeatable"`
	Numbered   bool   `yaml:"numbered"`
	// Named is meaningful only on numbered sections: true means each
	// instance's heading text comes from content. It is a pointer so a
	// missing field is distinguishable from an explicit false.
	Named *bool `yaml:"named"`
	// Guidance is how-to-fill prose; Skeleton emits it as YAML comments.
	Guidance string `yaml:"guidance"`
}

// Schema is the top-level YAML structure.
type Schema struct {
	Type        string          `yaml:"type"`
	TitlePrefix string          `yaml:"title_prefix"`
	Metadata    []string        `yaml:"metadata"`
	Sections    []SchemaSection `yaml:"sections"`

	// ExpectedKind is the title kind derived from TitlePrefix by LoadSchema.
	ExpectedKind names.Kind `yaml:"-"`

	// loaded is set only by LoadSchema; Check rejects a Schema without it.
	loaded bool
}

// HasErrors reports whether any finding is an error.
func (r Result) HasErrors() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Render assembles a plan issue body from a section map. It runs Check first
// and refuses, returning no body, when the check reports any error; the
// returned Result carries every finding, warnings included. Rendering applies
// the schema's order and numbering whatever order the map was written in.
//
// Because a draft's content is data, Render also verifies the rendered body:
// its "## " headings must be exactly the ones Render emitted, so no content
// can add, hide or reshape a section.
func Render(sc *Schema, origin string, m *SectionMap) (string, Result) {
	res := Check(sc, origin, m)
	if res.HasErrors() {
		return "", res
	}

	var sb strings.Builder
	for _, key := range sc.Metadata {
		fmt.Fprintf(&sb, "**%s:** %s\n", key, strings.TrimSpace(m.Metadata[key].Value))
	}

	type emitted struct {
		id, heading string
		line        int
	}
	var want []emitted
	emit := func(sec SchemaSection, heading, content string, line int) {
		want = append(want, emitted{sec.ID, heading, line})
		content = strings.Trim(strings.TrimRight(content, " \t\r\n"), "\n")
		if content == "" {
			fmt.Fprintf(&sb, "\n## %s\n", heading)
			return
		}
		fmt.Fprintf(&sb, "\n## %s\n\n%s\n", heading, content)
	}

	// One running ordinal across all numbered sections.
	ordinal := 0
	for _, sec := range sc.Sections {
		for _, e := range m.Sections[sec.ID] {
			heading := sec.Heading
			if sec.Numbered {
				ordinal++
				if isNamed(sec) {
					heading = strings.TrimSpace(e.Name)
				}
				heading = fmt.Sprintf("%d. %s", ordinal, heading)
			}
			emit(sec, heading, e.Content, e.Line)
		}
	}

	// A heading that does not parse back as exactly itself (a line break in
	// an entry name) would add headings of its own.
	for _, w := range want {
		hs := FindH2Lines("## " + w.heading + "\n")
		if len(hs) == 1 && hs[0].Text == w.heading {
			continue
		}
		var texts []string
		for _, h := range hs {
			texts = append(texts, fmt.Sprintf("%q", h.Text))
		}
		res.Findings = append(res.Findings, Finding{Severity: SeverityError, Location: "section:" + w.id, Line: w.line, Message: fmt.Sprintf("the heading Render builds for section %q, %q, would be rendered as %d headings (%s). Section content must never change the document's structure: keep every entry name on a single line, and use \"###\" for sub-headings.", w.id, w.heading, len(hs), strings.Join(texts, ", "))})
		return "", res
	}

	body := sb.String()
	got := FindH2Lines(body)
	for i := 0; i < len(want) || i < len(got); i++ {
		switch {
		case i >= len(want):
			res.Findings = append(res.Findings, Finding{Severity: SeverityError, Location: "draft", Message: fmt.Sprintf("the rendered body has an extra heading %q (line %d of the rendered body) that Render did not emit. Section content must never change the document's structure: use \"###\" for sub-headings and keep every entry name on a single line.", got[i].Text, got[i].Line)})
			return "", res
		case i >= len(got) || got[i].Text != want[i].heading:
			found := "no heading"
			if i < len(got) {
				found = fmt.Sprintf("the heading %q (line %d of the rendered body)", got[i].Text, got[i].Line)
			}
			res.Findings = append(res.Findings, Finding{Severity: SeverityError, Location: "section:" + want[i].id, Line: want[i].line, Message: fmt.Sprintf("rendering section %q did not produce the heading Render emitted: expected the rendered body's heading #%d to be %q but found %s. Section content must never change the document's structure: use \"###\" for sub-headings and keep every entry name on a single line.", want[i].id, i+1, want[i].heading, found)})
			return "", res
		}
	}
	return body, res
}

// Skeleton returns an empty YAML draft for sc: a double-quoted title
// placeholder from title_prefix, every metadata key as "", and every section
// key with its empty shape (an empty block, a list of blocks, or a list of
// {name, content} entries). It contains no comments, because drafts take
// none; Guidance prints the how-to-fill text separately. It is itself a
// loadable draft; Check then reports what is still empty.
func Skeleton(sc *Schema) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "title: %s\n", yamlQuote(sc.TitlePrefix))
	sb.WriteString("metadata:\n")
	for _, key := range sc.Metadata {
		fmt.Fprintf(&sb, "  %s: \"\"\n", key)
	}
	sb.WriteString("sections:\n")
	for _, sec := range sc.Sections {
		switch {
		case isNamed(sec):
			fmt.Fprintf(&sb, "  %s:\n    - name: \"\"\n      content: |\n", sec.ID)
		case sec.Repeatable:
			fmt.Fprintf(&sb, "  %s:\n    - |\n", sec.ID)
		default:
			fmt.Fprintf(&sb, "  %s: |\n", sec.ID)
		}
	}
	return sb.String()
}

// Guidance returns plain-text help for filling in a draft of sc: the rules
// for writing a draft, then, per section in schema order, its heading,
// whether it is required, repeatable, numbered or named, the draft key shape
// and the schema's guidance text.
func Guidance(sc *Schema) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "schema: %s\n\n", sc.Type)
	sb.WriteString("Rules for writing a draft:\n")
	sb.WriteString("- Keep the title and every metadata value in double quotes; write \\\" for a quote and \\\\ for a backslash inside them.\n")
	sb.WriteString("- Write section content as | blocks, indenting every line at least as far as the block's first line.\n")
	sb.WriteString("- Use ### for sub-headings inside content, never ##: a ## line would start a new section.\n")
	sb.WriteString("- Write no # comments anywhere in the draft: YAML drops them, and a # line meant as content would be lost.\n")
	sb.WriteString("- Use no --- or ... at column 0: they are YAML document markers. Indent a horizontal rule to the block's level.\n")
	sb.WriteString("- Leave a blank line before a --- rule, and never --- or === directly under text: markdown reads that as a heading underline.\n")
	sb.WriteString("- Always close every code fence inside the same section that opens it: an open fence swallows the sections after it.\n")
	sb.WriteString("- Write no raw HTML: `<!--`, `<details>`, `<pre>`, `<script>`, `<style>` and `<textarea>` (opening or closing) must be in backticks as inline code, in content, metadata values, names and the title.\n")
	fmt.Fprintf(&sb, "\nMetadata keys (each required, one line): %s\n", strings.Join(sc.Metadata, ", "))
	sb.WriteString("\nSections, in render order:\n")
	for _, sec := range sc.Sections {
		attrs := []string{"optional"}
		if sec.Required {
			attrs = []string{"required"}
		}
		if sec.Repeatable {
			attrs = append(attrs, "repeatable")
		}
		if sec.Numbered {
			attrs = append(attrs, "numbered")
		}
		if isNamed(sec) {
			attrs = append(attrs, "named")
		}
		heading := strings.Trim(sec.Heading, "<>")
		fmt.Fprintf(&sb, "\n%s (%s)\n", heading, strings.Join(attrs, ", "))
		switch {
		case isNamed(sec):
			fmt.Fprintf(&sb, "  %s: a list of entries with name: and content: |; each name becomes the heading text\n", sec.ID)
		case sec.Repeatable:
			fmt.Fprintf(&sb, "  %s: a list of | blocks, each starting with - |\n", sec.ID)
		default:
			fmt.Fprintf(&sb, "  %s: a single | block\n", sec.ID)
		}
		for _, line := range wrapComment(sec.Guidance, 76) {
			fmt.Fprintf(&sb, "  %s\n", line)
		}
	}
	return sb.String()
}

// wrapComment folds text to lines of at most width characters.
func wrapComment(text string, width int) []string {
	var lines []string
	line := ""
	for _, w := range strings.Fields(text) {
		if line != "" && len(line)+1+len(w) > width {
			lines = append(lines, line)
			line = ""
		}
		if line != "" {
			line += " "
		}
		line += w
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}
