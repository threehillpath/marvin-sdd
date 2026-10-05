// Package template renders plan issue bodies from YAML schema definitions.
// It assembles structure deterministically — ordered metadata, correct headings,
// repeatable/numbered section handling — but never authors section content.
package template

import (
	"embed"
	"fmt"
	"regexp"
	"sort"
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

// DefaultSchemaNames returns the names of the built-in schemas, sorted.
func DefaultSchemaNames() []string {
	entries, err := defaultSchemas.ReadDir("schemas")
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if n, ok := strings.CutSuffix(e.Name(), ".yml"); ok {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
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
	// Guidance is how-to-fill prose; the Guidance function prints it as plain
	// text (a skeleton carries no comments, because drafts take none).
	Guidance string `yaml:"guidance"`
}

// SchemaValidation mirrors the YAML validation block.
type SchemaValidation struct {
	RequiredSections []string `yaml:"required_sections"`
	Rules            []string `yaml:"rules"`
}

// Schema is the top-level YAML structure.
type Schema struct {
	Type        string `yaml:"type"`
	TitlePrefix string `yaml:"title_prefix"`
	// DefaultLabels and Validation are documentation of the schema file: no
	// code reads them, but LoadSchema decodes strictly, so they must be known.
	DefaultLabels []string         `yaml:"default_labels"`
	Metadata      []string         `yaml:"metadata"`
	Sections      []SchemaSection  `yaml:"sections"`
	Validation    SchemaValidation `yaml:"validation"`

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
	var origins []lineOrigin
	var heads []emittedHeading
	put := func(text string, o lineOrigin) {
		sb.WriteString(text)
		sb.WriteByte('\n')
		origins = append(origins, o)
	}
	for _, key := range sc.Metadata {
		put(fmt.Sprintf("**%s:** %s", key, strings.TrimSpace(m.Metadata[key].Value)),
			lineOrigin{loc: "metadata:" + key, what: fmt.Sprintf("metadata value %q", key), line: m.Metadata[key].Line, where: fmt.Sprintf("edit %q under \"metadata:\"", key)})
	}

	emit := func(sec SchemaSection, heading, content string, line int) {
		o := lineOrigin{loc: "section:" + sec.ID, what: "section " + label(sec), line: line, where: fmt.Sprintf("inside the %q block", sec.ID)}
		if m.Source == SourceMarkdown {
			o.where = "under that heading"
		}
		put("", o)
		heads = append(heads, emittedHeading{line: len(origins) + 1, text: heading})
		put("## "+heading, o)
		content = strings.TrimRight(content, " \t\r\n")
		lead := 0
		for strings.HasPrefix(content, "\n") {
			content = content[1:]
			lead++
		}
		if content == "" {
			return
		}
		put("", o)
		for i, cl := range strings.Split(content, "\n") {
			co := o
			co.content = lead + i + 1
			put(strings.TrimRight(cl, "\r"), co)
		}
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

	body := sb.String()
	if fs := verifyBody(body, origins, heads); len(fs) > 0 {
		res.Findings = append(res.Findings, fs...)
		return "", res
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
	sb.WriteString("- Keep the title and every metadata value in double quotes. Inside them, write \\\" for a quote and \\\\ for a backslash.\n")
	sb.WriteString("- Write section content as | blocks, indenting every line at least as far as the block's first line.\n")
	sb.WriteString("- Use ### or deeper for sub-headings in content, never # or ##, including inside list items and quotes: a # or ## line would add or break a section. Escape a literal # at the start of a line as \\#.\n")
	sb.WriteString("- Write no # comments anywhere in the draft: YAML drops them, so a # line meant as content would be lost.\n")
	sb.WriteString("- Never put --- or ... at column 0: they are YAML document markers. Indent a horizontal rule to the block's level.\n")
	sb.WriteString("- Leave a blank line before a --- horizontal rule, and never put --- or === directly under a line of text: markdown reads that as a heading underline.\n")
	sb.WriteString("- Close every code fence in the section that opens it: an open fence swallows the sections after it.\n")
	sb.WriteString("- Inside a list item, indent the opening fence, every code line and the closing fence at least as far as the opening fence: a less-indented line ends the item and leaves the fence open.\n")
	sb.WriteString("- Don't start a line with an HTML tag, <? or <!, as in <div>, a tag alone on its line, <?php or <!DOCTYPE: markdown reads it as an HTML block that can swallow what follows.\n")
	sb.WriteString("- Use no raw HTML (<!--, <details>, <pre>, <script>, <style>, <textarea>, opening or closing; also <?, <![CDATA[ or <! followed by a letter) in content, metadata values, names or the title. Put it in backticks as inline code.\n")
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
		for _, line := range wrapGuidance(sec.Guidance, 76) {
			fmt.Fprintf(&sb, "  %s\n", line)
		}
	}
	return sb.String()
}

var labelLineRe = regexp.MustCompile(`^[A-Za-z][A-Za-z ]{0,30}:(?:\s|$)`)

// wrapGuidance wraps each newline-separated line of text on its own to at
// most width characters, so a schema's guidance keeps its line structure (a
// "Key:" label line, a "- " list item or a checkbox item starts its own
// output line). A line's own leading indent is kept, and an indented "- "
// item or "Key:" label line hangs its continuation lines two spaces in; prose
// that merely contains a colon does not hang. A "- [ ]" checkbox marker is
// never split and never left at the end of a line.
func wrapGuidance(text string, width int) []string {
	var out []string
	for _, raw := range strings.Split(text, "\n") {
		body := strings.TrimSpace(raw)
		if body == "" {
			continue
		}
		indent := raw[:len(raw)-len(strings.TrimLeft(raw, " \t"))]
		hang := indent
		if indent != "" && (strings.HasPrefix(body, "- ") || labelLineRe.MatchString(body)) {
			hang += "  "
		}
		words := glueMarkers(strings.Fields(body))
		line := indent
		blank := true // nothing but the indent on the line yet
		for _, w := range words {
			if !blank && len(line)+1+len(w) > width {
				out = append(out, line)
				line, blank = hang, true
			}
			if !blank {
				line += " "
			}
			line += w
			blank = false
		}
		out = append(out, line)
	}
	return out
}

// glueMarkers joins the tokens of a checkbox marker ("-", "[", "]" or "-",
// "[x]") into one word and glues it to the word after it, so a line never
// ends on the marker; a list marker at the start of the line is glued to the
// word after it too.
func glueMarkers(words []string) []string {
	var out []string
	glueNext := false
	for i := 0; i < len(words); i++ {
		w := words[i]
		marker := false
		switch {
		case w == "-" && i+2 < len(words) && words[i+1] == "[" && words[i+2] == "]":
			w, marker = "- [ ]", true
			i += 2
		case w == "-" && i+1 < len(words) && (words[i+1] == "[x]" || words[i+1] == "[X]"):
			w, marker = "- "+words[i+1], true
			i++
		case w == "-" && i == 0:
			marker = true
		}
		if glueNext {
			out[len(out)-1] += " " + w
		} else {
			out = append(out, w)
		}
		glueNext = marker
	}
	return out
}
