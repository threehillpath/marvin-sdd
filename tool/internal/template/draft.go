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
			l.m.Title = l.text(kv, `"title"`)
			l.m.TitleLine = kv.val.Line
		case "metadata":
			l.walkMetadata(kv.val)
		case "sections":
			l.walkSections(kv.val)
		}
	}
}

type pair struct{ key, val *yaml.Node }

// pairs returns the key/value pairs of a mapping node, reporting (and
// dropping) a key that repeats an earlier one. yaml.v3 raises that error
// itself when decoding into structs, but not when decoding into nodes.
func (l *draftLoader) pairs(n *yaml.Node, what string) []pair {
	var out []pair
	first := map[string]int{}
	for i := 0; i+1 < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		if at, dup := first[k.Value]; dup {
			l.repeatedKey(what, k, at)
			continue
		}
		first[k.Value] = k.Line
		out = append(out, pair{k, v})
	}
	return out
}

func (l *draftLoader) repeatedKey(what string, k *yaml.Node, firstLine int) {
	for _, sec := range l.sc.Sections {
		if what == "sections" && sec.ID == k.Value && sec.Repeatable {
			l.fail(k.Line, "key %q appears twice (line %d and line %d). A repeatable section is one list: put every entry under a single \"%s:\" key, each starting with \"- \"", k.Value, firstLine, k.Line, k.Value)
			return
		}
	}
	if what == "sections" {
		l.fail(k.Line, "key %q appears twice (line %d and line %d). Merge them into one \"%s: |\" block", k.Value, firstLine, k.Line, k.Value)
		return
	}
	l.fail(k.Line, "key %q appears twice in %s (line %d and line %d). Remove the duplicate or merge its value into the first", k.Value, what, firstLine, k.Line)
}

func (l *draftLoader) walkMetadata(n *yaml.Node) {
	for _, kv := range l.pairs(n, "metadata") {
		l.m.Metadata[kv.key.Value] = Field{Value: l.text(kv, fmt.Sprintf("%q", kv.key.Value)), Line: kv.val.Line}
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
	id := fmt.Sprintf("%q", sec.ID)
	switch {
	case !sec.Repeatable:
		return []Entry{{Content: l.content(pair{key: &yaml.Node{Value: sec.ID}, val: v}, "section "+id, sec.ID+": |"), Line: v.Line}}
	case isNamed(sec):
		var out []Entry
		for _, item := range v.Content {
			e := Entry{Line: item.Line}
			kvs := l.pairs(item, "an entry of section "+id)
			for _, kv := range kvs {
				if kv.key.Value == "name" {
					e.Name = l.text(kv, `"name"`)
				}
			}
			for _, kv := range kvs {
				if kv.key.Value == "content" {
					subject := `the "content" of an entry of section ` + id
					if e.Name != "" {
						subject = fmt.Sprintf("the \"content\" of the entry %q of section %s", e.Name, id)
					}
					e.Content = l.content(kv, subject, "content: |")
				}
			}
			out = append(out, e)
		}
		return out
	default:
		var out []Entry
		for _, item := range v.Content {
			out = append(out, Entry{Content: l.content(pair{key: &yaml.Node{Value: sec.ID}, val: item}, "section "+id, "- |"), Line: item.Line})
		}
		return out
	}
}

// cutComment returns the comment yaml.v3 attached to a scalar pair, and
// whether the value was plain (so the comment cut it) rather than quoted. A
// comment after a literal "|" header is harmless and not reported.
func cutComment(kv pair) (comment string, plain bool) {
	v := kv.val
	if v.Kind != yaml.ScalarNode || v.Style&(yaml.LiteralStyle|yaml.FoldedStyle) != 0 {
		return "", false
	}
	plain = v.Style&(yaml.DoubleQuotedStyle|yaml.SingleQuotedStyle) == 0
	if c := v.LineComment; c != "" {
		return c, plain
	}
	return kv.key.LineComment, plain
}

// text reads a single-line scalar (title, metadata value, entry name),
// reporting a comment that cut or trails it.
func (l *draftLoader) text(kv pair, subject string) string {
	if c, plain := cutComment(kv); c != "" {
		if plain {
			l.fail(kv.val.Line, "the value of %s was cut at %q: an unquoted \"#\" starts a YAML comment, so the rest of the value was dropped. Wrap the whole value in double quotes", subject, c)
		} else {
			l.fail(kv.val.Line, "the value of %s is followed by the comment %q, which YAML ignores. If it is part of the value, move it inside the quotes; otherwise remove the comment", subject, c)
		}
	}
	return strings.TrimSpace(kv.val.Value)
}

// content reads section content, which must be a literal "|" block. subject
// names the content in messages and how is the YAML to write instead.
func (l *draftLoader) content(kv pair, subject, how string) string {
	v := kv.val
	if c, plain := cutComment(kv); c != "" {
		if plain {
			l.fail(v.Line, "the content of %s was cut at %q: an unquoted \"#\" starts a YAML comment, so the rest of the content was dropped. Replace it with a | block: write \"%s\" and indent the whole text below it", subject, c, how)
		} else {
			l.fail(v.Line, "the content of %s is followed by the comment %q, which YAML ignores. Write the content as a | block: \"%s\" with the text indented below it", subject, c, how)
		}
		return v.Value
	}
	if v.Kind == yaml.ScalarNode && v.Style&yaml.LiteralStyle == 0 {
		var what string
		switch {
		case v.Style&yaml.FoldedStyle != 0:
			what = "a folded (>) block, which reflows lines and silently breaks markdown lists"
		case v.Style&yaml.DoubleQuotedStyle != 0:
			what = "a double-quoted scalar"
		case v.Style&yaml.SingleQuotedStyle != 0:
			what = "a single-quoted scalar"
		default:
			what = "a plain scalar, which YAML folds and truncates at \" #\""
		}
		l.fail(v.Line, "the content of %s is %s, not a literal | block. Write it as \"%s\" with the text indented below it", subject, what, how)
	}
	return v.Value
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

// parseError turns a yaml.v3 syntax error into findings.
func (l *draftLoader) parseError(data []byte, err error) []Finding {
	l.fail(0, "the draft is not valid YAML (%v)", err)
	return l.findings
}
