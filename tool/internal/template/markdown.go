package template

import (
	"fmt"
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
// heading that GitHub shows as metadata (outside a fence, and the first
// non-blank line, after a blank line, or directly below another metadata
// line); other text there (a revision blockquote, say) is kept as the
// preamble. Sections are split with FindH2Lines' scanner, the one Check uses,
// so the parser and the checker agree on where a section starts and ends.
// Each section runs from its heading to the next "## " heading.
//
// Pairing ParseMarkdown with Check skips the goldmark verification that
// CheckMarkdown adds, so callers that validate a body should use
// CheckMarkdown.
func ParseMarkdown(sc *Schema, title, body string) *SectionMap {
	m, _, _, _ := parseMarkdown(sc, title, body)
	return m
}

// parseMarkdown is ParseMarkdown plus what verifyBody needs: the normalized
// body, where each of its lines came from, and the "## " headings found.
func parseMarkdown(sc *Schema, title, body string) (*SectionMap, string, []lineOrigin, []emittedHeading) {
	m := &SectionMap{
		Source:   SourceMarkdown,
		Title:    title,
		Metadata: map[string]Field{},
		Sections: map[string][]Entry{},
	}
	body = strings.ReplaceAll(body, "\r\n", "\n")
	lines := strings.Split(body, "\n")
	scan := scanContent(body)
	heads := scan.Headings
	origins := make([]lineOrigin, len(lines))
	var emitted []emittedHeading

	end := len(lines)
	if len(heads) > 0 {
		end = heads[0].Line - 1
	}
	m.Preamble = strings.Join(lines[:end], "\n")
	prevBlank, prevAccepted := true, false
	for i, line := range lines[:end] {
		origins[i] = lineOrigin{loc: "draft", what: "the text above the first \"## \" heading", line: i + 1, where: `above the first "## " heading`}
		blank := strings.TrimSpace(line) == ""
		accepted := false
		if mm := metadataLineRe.FindStringSubmatch(line); mm != nil {
			key := mm[1]
			switch {
			case scan.InFence[i]:
				m.MisplacedMetadata = append(m.MisplacedMetadata, MisplacedField{Key: key, Line: i + 1, InFence: true})
			case !prevBlank && !prevAccepted:
				m.MisplacedMetadata = append(m.MisplacedMetadata, MisplacedField{Key: key, Line: i + 1, Above: strings.TrimSpace(lines[i-1])})
			default:
				accepted = true
				if first, dup := m.Metadata[key]; dup {
					m.RepeatedMetadata = append(m.RepeatedMetadata, RepeatedField{Key: key, FirstLine: first.Line, Line: i + 1})
				} else {
					m.Metadata[key] = Field{Value: strings.TrimSpace(mm[2]), Line: i + 1}
				}
				origins[i] = lineOrigin{loc: "metadata:" + key, what: fmt.Sprintf("metadata value %q", key), line: i + 1, where: fmt.Sprintf("in the \"**%s:**\" line", key)}
			}
		}
		prevBlank, prevAccepted = blank, accepted
	}

	for i, h := range heads {
		emitted = append(emitted, emittedHeading{line: h.Line, text: h.Text})
		stop := len(lines)
		if i+1 < len(heads) {
			stop = heads[i+1].Line - 1
		}
		raw := strings.Join(lines[h.Line:stop], "\n")
		content := strings.TrimLeft(raw, "\n")
		lead := strings.Count(raw, "\n") - strings.Count(content, "\n")
		content = strings.TrimRight(content, " \t\n")

		o := lineOrigin{loc: "draft", what: fmt.Sprintf("the unknown heading \"## %s\"", h.Text), line: h.Line, where: "under that heading"}
		if id, e, ok := classifyHeading(sc, h.Text); ok {
			e.Content, e.Line = content, h.Line
			m.Sections[id] = append(m.Sections[id], e)
			for _, sec := range sc.Sections {
				if sec.ID == id {
					o.loc, o.what = "section:"+id, "section "+label(sec)
				}
			}
		} else {
			h.Content = content
			m.UnknownHeadings = append(m.UnknownHeadings, h)
		}
		for j := h.Line - 1; j < stop; j++ {
			origins[j] = o
			if c := j - h.Line + 1 - lead; c > 0 {
				origins[j].content = c
			}
		}
	}
	return m, body, origins, emitted
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
// body is also verified the way Render verifies what it emits, so a body that
// GitHub would structure differently from the parser is refused.
func CheckMarkdown(sc *Schema, origin, title, body string) Result {
	m, norm, origins, heads := parseMarkdown(sc, title, body)
	res := Check(sc, origin, m)
	if !res.HasErrors() {
		res.Findings = append(res.Findings, verifyBody(norm, origins, heads)...)
	}
	return res
}
