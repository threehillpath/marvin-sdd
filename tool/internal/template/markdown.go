package template

import (
	"regexp"
	"strconv"
	"strings"
)

var metadataLineRe = regexp.MustCompile(`^\*\*(.+?):\*\*[ \t]*(.*)$`)

// ParseMarkdown turns a markdown issue body and a caller-supplied title into
// a SectionMap for sc. The title has no source line, because a body does not
// carry one.
//
// Metadata is read only from "**Key:** value" lines above the first "## "
// heading; other text there (a revision blockquote, say) is ignored. Sections
// are split with FindH2Lines, the scanner Check uses, so the parser and the
// checker agree on where a section starts and ends. Each section runs from
// its heading to the next "## " heading.
func ParseMarkdown(sc *Schema, title, body string) *SectionMap {
	m := &SectionMap{
		Source:   SourceMarkdown,
		Title:    title,
		Metadata: map[string]Field{},
		Sections: map[string][]Entry{},
	}
	body = strings.ReplaceAll(body, "\r\n", "\n")
	lines := strings.Split(body, "\n")
	heads := FindH2Lines(body)

	end := len(lines)
	if len(heads) > 0 {
		end = heads[0].Line - 1
	}
	for i, line := range lines[:end] {
		if mm := metadataLineRe.FindStringSubmatch(line); mm != nil {
			m.Metadata[mm[1]] = Field{Value: strings.TrimSpace(mm[2]), Line: i + 1}
		}
	}

	for i, h := range heads {
		stop := len(lines)
		if i+1 < len(heads) {
			stop = heads[i+1].Line - 1
		}
		content := strings.Trim(strings.Join(lines[h.Line:stop], "\n"), "\n")
		content = strings.TrimRight(content, " \t\n")
		if id, e, ok := classifyHeading(sc, h.Text); ok {
			e.Content, e.Line = content, h.Line
			m.Sections[id] = append(m.Sections[id], e)
			continue
		}
		h.Content = content
		m.UnknownHeadings = append(m.UnknownHeadings, h)
	}
	return m
}

var numberedRe = regexp.MustCompile(`^(\d+)\.(?:[ \t]+(.*))?$`)

// classifyHeading maps a heading's text to a section id and the entry's name
// and number. A numbered heading ("N. Text") goes to the non-named numbered
// section whose literal heading is Text, else to the schema's only named
// numbered section with name Text, else it is unknown. Any other heading
// must equal a non-numbered section's heading.
func classifyHeading(sc *Schema, text string) (id string, e Entry, ok bool) {
	if nm := numberedRe.FindStringSubmatch(text); nm != nil {
		n, _ := strconv.Atoi(nm[1])
		rest := strings.TrimSpace(nm[2])
		var named []SchemaSection
		for _, sec := range sc.Sections {
			switch {
			case isNamed(sec):
				named = append(named, sec)
			case sec.Numbered && sec.Heading == rest:
				return sec.ID, Entry{Number: n}, true
			}
		}
		if len(named) == 1 {
			return named[0].ID, Entry{Name: rest, Number: n}, true
		}
		return "", Entry{}, false
	}
	if id, ok := literalSection(sc, text); ok {
		return id, Entry{}, true
	}
	return "", Entry{}, false
}

// literalSection returns the id of the non-numbered section whose heading
// equals text exactly (case-sensitive).
func literalSection(sc *Schema, text string) (string, bool) {
	for _, sec := range sc.Sections {
		if !sec.Numbered && sec.Heading == text {
			return sec.ID, true
		}
	}
	return "", false
}

// CheckMarkdown parses body and checks it. It is the entry point for the
// markdown path: the Result is Check's, and when Check finds no error the
// body is also verified the way Render verifies what it emits.
func CheckMarkdown(sc *Schema, origin, title, body string) Result {
	m := ParseMarkdown(sc, title, body)
	return Check(sc, origin, m)
}
