package template

import (
	"fmt"
	"regexp"
	"strings"

	"threehillpath.com/marvin-sdd/tool/internal/parse"
)

// Source says which input produced a SectionMap.
type Source int

const (
	SourceYAML Source = iota
	SourceMarkdown
)

// Field is a metadata value with its source line (0 when unknown).
type Field struct {
	Value string
	Line  int
}

// Entry is one instance of a section: an optional name (named numbered
// sections only), its content, the source line, and (markdown only) the
// number in the heading.
type Entry struct {
	Name    string
	Content string
	Line    int
	Number  int
}

// Heading is a "## " heading found in a markdown body.
type Heading struct {
	Text string
	Line int
}

// SectionMap is the input-independent shape the conformance check runs on.
type SectionMap struct {
	Source          Source
	Title           string
	TitleLine       int
	Metadata        map[string]Field
	Sections        map[string][]Entry
	UnknownHeadings []Heading // markdown only
}

// Severity of a finding.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Finding is one conformance problem.
type Finding struct {
	Severity Severity
	Location string // section:<id>, metadata:<Key>, title, or draft
	Line     int    // 0 when unknown
	Message  string
}

// Result is the outcome of a conformance check.
type Result struct {
	Type     string
	Origin   string
	Findings []Finding
}

// checker accumulates findings for one Check call.
type checker struct {
	sc  *Schema
	m   *SectionMap
	res Result
}

func (c *checker) add(sev Severity, loc string, line int, format string, args ...any) {
	c.res.Findings = append(c.res.Findings, Finding{Severity: sev, Location: loc, Line: line, Message: fmt.Sprintf(format, args...)})
}

// Check applies the conformance rules to m.
func Check(sc *Schema, origin string, m *SectionMap) Result {
	c := &checker{sc: sc, m: m, res: Result{Type: sc.Type, Origin: origin}}
	c.checkTitle()
	c.checkSections()
	return c.res
}

func (c *checker) checkTitle() {
	title := strings.TrimSpace(c.m.Title)
	line := c.m.TitleLine
	if title == "" {
		c.add(SeverityError, "title", line, "title is missing. The schema expects a title like %q. Set \"title:\" in the draft.", c.sc.TitlePrefix)
		return
	}
	if strings.Contains(title, "\n") {
		c.add(SeverityError, "title", line, "title %q spans more than one line. Use a single line.", title)
	}
	kind, ok := parse.Classify(title)
	if !ok {
		c.add(SeverityError, "title", line, "title %q has no recognizable leading identifier. The schema expects a title like %q with the real numbers filled in (no XXXXX). Start the title with the identifier%s.",
			title, c.sc.TitlePrefix, c.example())
		return
	}
	if kind != c.sc.ExpectedKind {
		c.add(SeverityError, "title", line, "%q is %s title, but schema %s expects %s title like %q. Change the title's identifier to match%s.",
			title, article(kind.String()), c.sc.Type, article(c.sc.ExpectedKind.String()), c.sc.TitlePrefix, c.example())
	}
}

var fiveDigits = regexp.MustCompile(`\d{5}`)

// example returns `, e.g. "<concrete prefix>"` when the metadata carries a
// plan or task number to fill XXXXX with, else "".
func (c *checker) example() string {
	for _, key := range []string{"Plan Number", "Task Number"} {
		if n := fiveDigits.FindString(c.m.Metadata[key].Value); n != "" {
			ex := strings.ReplaceAll(c.sc.TitlePrefix, "XXXXX", n)
			ex = strings.ReplaceAll(ex, "-N]", "-1]")
			return fmt.Sprintf(", e.g. %q", ex)
		}
	}
	return ""
}

func article(kind string) string {
	if kind == "arch" || kind == "impl" {
		return "an " + kind
	}
	return "a " + kind
}

func (c *checker) checkSections() {
	for _, sec := range c.sc.Sections {
		if sec.Required && len(c.m.Sections[sec.ID]) == 0 {
			c.add(SeverityError, "section:"+sec.ID, 0, "required section %q is missing. Add a %q block under \"sections:\" in the draft.",
				sec.Heading, sec.ID+": |")
		}
	}
}

// Format renders the result as plain text.
func (r Result) Format() string {
	return ""
}
