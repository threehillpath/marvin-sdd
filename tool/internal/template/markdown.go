package template

import (
	"regexp"
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
		if id, ok := literalSection(sc, h.Text); ok {
			m.Sections[id] = append(m.Sections[id], Entry{Content: content, Line: h.Line})
			continue
		}
		m.UnknownHeadings = append(m.UnknownHeadings, h)
	}
	return m
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
