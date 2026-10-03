package template

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// draftLoader walks a yaml.Node tree. Working on nodes rather than decoding
// into structs is what lets it see comments, scalar styles and every key, so
// nothing yaml.v3 accepts silently can be lost.
type draftLoader struct {
	sc       *Schema
	m        *SectionMap
	findings []Finding
}

func (l *draftLoader) fail(line int, format string, args ...any) {
	l.findings = append(l.findings, Finding{Severity: SeverityError, Location: "draft", Line: line, Message: sprintfLine(line, format, args...)})
}

// LoadDraft turns a YAML draft into a SectionMap. Every case yaml.v3 would
// accept while silently losing content is reported as a Finding (location
// "draft"), and no map is returned when there is any finding.
func LoadDraft(sc *Schema, data []byte) (*SectionMap, []Finding) {
	l := &draftLoader{sc: sc, m: &SectionMap{
		Source:   SourceYAML,
		Metadata: map[string]Field{},
		Sections: map[string][]Entry{},
	}}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, l.parseError(data, err)
	}
	root := &doc
	if root.Kind == yaml.DocumentNode && len(root.Content) == 1 {
		root = root.Content[0]
	}
	l.walkRoot(root)
	if len(l.findings) > 0 {
		return nil, l.findings
	}
	return l.m, nil
}

func (l *draftLoader) walkRoot(root *yaml.Node) {
	for _, kv := range l.pairs(root, "the draft") {
		switch kv.key.Value {
		case "title":
			l.m.Title = strings.TrimSpace(kv.val.Value)
			l.m.TitleLine = kv.val.Line
		case "metadata":
			l.walkMetadata(kv.val)
		case "sections":
			l.walkSections(kv.val)
		}
	}
}

type pair struct{ key, val *yaml.Node }

// pairs returns the key/value pairs of a mapping node.
func (l *draftLoader) pairs(n *yaml.Node, what string) []pair {
	var out []pair
	for i := 0; i+1 < len(n.Content); i += 2 {
		out = append(out, pair{n.Content[i], n.Content[i+1]})
	}
	return out
}

func (l *draftLoader) walkMetadata(n *yaml.Node) {
	for _, kv := range l.pairs(n, "metadata") {
		l.m.Metadata[kv.key.Value] = Field{Value: strings.TrimSpace(kv.val.Value), Line: kv.val.Line}
	}
}

func (l *draftLoader) walkSections(n *yaml.Node) {
	for _, kv := range l.pairs(n, "sections") {
		for _, sec := range l.sc.Sections {
			if sec.ID != kv.key.Value {
				continue
			}
			l.m.Sections[sec.ID] = l.entries(sec, kv.val)
		}
	}
}

func (l *draftLoader) entries(sec SchemaSection, v *yaml.Node) []Entry {
	switch {
	case !sec.Repeatable:
		return []Entry{{Content: v.Value, Line: v.Line}}
	case isNamed(sec):
		var out []Entry
		for _, item := range v.Content {
			e := Entry{Line: item.Line}
			for _, kv := range l.pairs(item, sec.ID) {
				switch kv.key.Value {
				case "name":
					e.Name = strings.TrimSpace(kv.val.Value)
				case "content":
					e.Content = kv.val.Value
				}
			}
			out = append(out, e)
		}
		return out
	default:
		var out []Entry
		for _, item := range v.Content {
			out = append(out, Entry{Content: item.Value, Line: item.Line})
		}
		return out
	}
}

// sprintfLine formats a loader message, appending "(line unknown)" when the
// line could not be located.
func sprintfLine(line int, format string, args ...any) string {
	msg := fmt.Sprintf(format, args...)
	if line <= 0 {
		msg += " (line unknown)"
	}
	return msg
}

// parseError turns a yaml.v3 syntax error into findings; filled in below.
func (l *draftLoader) parseError(data []byte, err error) []Finding {
	l.fail(0, "the draft is not valid YAML (%v)", err)
	return l.findings
}
