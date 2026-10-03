package template

import "fmt"

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

// Check applies the conformance rules to m.
func Check(sc *Schema, origin string, m *SectionMap) Result {
	res := Result{Type: sc.Type, Origin: origin}
	for _, sec := range sc.Sections {
		if sec.Required && len(m.Sections[sec.ID]) == 0 {
			res.Findings = append(res.Findings, Finding{
				Severity: SeverityError,
				Location: "section:" + sec.ID,
				Message: fmt.Sprintf("required section %q is missing. Add a %q block under \"sections:\" in the draft.",
					sec.Heading, sec.ID+": |"),
			})
		}
	}
	return res
}

// Format renders the result as plain text.
func (r Result) Format() string {
	return ""
}
