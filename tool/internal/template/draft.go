package template

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
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
	lines    []string            // raw draft lines, to locate head and foot comments
	code     []string            // the same lines with block-scalar content blanked
	used     map[int]bool        // comment lines already reported
	handled  map[*yaml.Node]bool // nodes whose line comment already has a finding
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
	}, lines: strings.Split(string(data), "\n"), handled: map[*yaml.Node]bool{}, code: rawLines(data), used: map[int]bool{}}
	// Drafts never use document markers: a column-0 "---" or "..." ends the
	// open block and starts a document the loader would drop.
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		if markerRe.MatchString(line) {
			l.fail(i+1, "%q at column 0 is a YAML document marker, which drafts never use: it ends the draft's document, so everything after it would be dropped. If it is a horizontal rule in section content, indent it to the same level as the text of its | block and leave a blank line above it (a rule directly under text is read as a heading underline); otherwise delete the line", strings.TrimRight(line, " \t"))
		}
	}
	if len(l.findings) > 0 {
		return nil, l.findings
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var doc yaml.Node
	if err := dec.Decode(&doc); err != nil && err != io.EOF {
		return nil, l.parseError(data, err)
	}
	var second yaml.Node
	if err := dec.Decode(&second); err != io.EOF {
		line := second.Line
		l.fail(line, "the draft contains a second YAML document, which would be dropped. A draft is exactly one document: remove the \"---\" or \"...\" that starts it and put everything under the single title:, metadata: and sections: keys")
		return nil, l.findings
	}
	root := &doc
	if root.Kind == yaml.DocumentNode && len(root.Content) == 1 {
		root = root.Content[0]
	}
	if root.Kind == 0 || (root.Kind == yaml.ScalarNode && root.Tag == "!!null") {
		l.fail(0, "the draft is empty. A draft needs the keys title:, metadata: and sections:. Start from an empty draft with: marvin template render %s --skeleton, then fill it in", sc.Type)
		return nil, l.findings
	}
	if root.Kind != yaml.MappingNode {
		l.fail(root.Line, "the draft must be a mapping with the keys title:, metadata: and sections:, but it is %s. Start from: marvin template render %s --skeleton", kindName(root), sc.Type)
		return nil, l.findings
	}
	l.walkRoot(root)
	l.comments(&doc)
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
			if l.expectMapping(kv.val, `"metadata"`, "metadata keys to values. Write one `Key: \"value\"` line per metadata key under \"metadata:\"") {
				l.walkMetadata(kv.val)
			}
		case "sections":
			if l.expectMapping(kv.val, `"sections"`, "section ids to their content. Write one entry per section under \"sections:\"") {
				l.walkSections(kv.val)
			}
		default:
			l.unknownKey(kv.key, "at the top level of the draft", []string{"title", "metadata", "sections"}, true)
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
		if !contains(l.sc.Metadata, kv.key.Value) {
			l.unknownKey(kv.key, "under \"metadata:\"", l.sc.Metadata, false)
			continue
		}
		l.m.Metadata[kv.key.Value] = Field{Value: l.text(kv, fmt.Sprintf("%q", kv.key.Value)), Line: kv.val.Line}
	}
}

func (l *draftLoader) walkSections(n *yaml.Node) {
	var ids []string
	for _, sec := range l.sc.Sections {
		ids = append(ids, sec.ID)
	}
	for _, kv := range l.pairs(n, "sections") {
		if !contains(ids, kv.key.Value) {
			l.unknownKey(kv.key, "under \"sections:\"", ids, true)
			continue
		}
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
	case !sec.Repeatable && v.Kind != yaml.ScalarNode:
		l.fail(v.Line, "section %s is not repeatable, so it takes one | block, but the draft gives %s. Write \"%s: |\" and indent the text below it", id, kindName(v), sec.ID)
		return nil
	case sec.Repeatable && v.Kind != yaml.SequenceNode:
		if isNamed(sec) {
			l.fail(v.Line, "section %s is repeatable and each entry is named: write it as a list of entries, each with \"name:\" and \"content: |\", but the draft gives %s", id, kindName(v))
		} else {
			l.fail(v.Line, "section %s is repeatable: write it as a list of | blocks, each starting with \"- |\", but the draft gives %s", id, kindName(v))
		}
		return nil
	case !sec.Repeatable:
		return []Entry{{Content: l.content(pair{key: &yaml.Node{Value: sec.ID}, val: v}, "section "+id, sec.ID+": |"), Line: v.Line}}
	case isNamed(sec):
		var out []Entry
		for _, item := range v.Content {
			if !l.expectMapping(item, "an entry of section "+id, "name: and content: |, like \"- name: ...\" followed by \"content: |\"") {
				continue
			}
			e := Entry{Line: item.Line}
			kvs := l.pairs(item, "an entry of section "+id)
			for _, kv := range kvs {
				if kv.key.Value != "name" && kv.key.Value != "content" {
					l.unknownKey(kv.key, "in an entry of section "+id, []string{"name", "content"}, true)
				}
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
// whether the value was plain (so the comment cut it) rather than quoted. It
// returns no comment for a literal or folded block: a comment after a "|"
// header cannot cut anything, and the general comment walk reports it.
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
	if kv.val.Kind != yaml.ScalarNode {
		l.fail(kv.val.Line, "%s must be a single line of text, but the draft gives %s. Write it as a double-quoted string", subject, kindName(kv.val))
		return ""
	}
	if c, plain := cutComment(kv); c != "" {
		l.handled[kv.key], l.handled[kv.val] = true, true
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
	if v.Kind != yaml.ScalarNode {
		l.fail(v.Line, "the content of %s must be a literal | block, but the draft gives %s. Write it as \"%s\" with the text indented below it", subject, kindName(v), how)
		return ""
	}
	if c, plain := cutComment(kv); c != "" {
		l.handled[kv.key], l.handled[kv.val] = true, true
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

var (
	yamlLineRe  = regexp.MustCompile(`^yaml: (?:line (\d+): )?(.*)$`)
	markerRe    = regexp.MustCompile(`^(?:---|\.\.\.)(?:[ \t]|$)`)
	titleRawRe  = regexp.MustCompile(`^title:\s*\[`)
	blockHeadRe = regexp.MustCompile(`^(\s*)(?:-\s+)?(?:[^#\s][^:]*:\s+)?[|>][+-]?\d?\s*(?:#.*)?$`)
	// quotedValueRe splits a line whose value starts with a double quote
	// into everything before that quote (never a quoted key) and the rest.
	quotedValueRe = regexp.MustCompile(`^(\s*(?:-\s+)?(?:[^\s:"'#][^:]*:\s+)?)"(.*)$`)
	badEscapeRe   = regexp.MustCompile(`\\[^"0abtnvfre NLP_xuU/\t]`)
)

// rawLines returns the draft's lines with those inside a block scalar blanked,
// so a quote or a bracket in content is never mistaken for a YAML value.
func rawLines(data []byte) []string {
	lines := strings.Split(string(data), "\n")
	inBlock, blockIndent := false, 0
	for i, line := range lines {
		line = strings.TrimRight(line, "\r")
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if inBlock {
			if strings.TrimSpace(line) == "" || indent > blockIndent {
				lines[i] = ""
				continue
			}
			inBlock = false
		}
		if m := blockHeadRe.FindStringSubmatch(line); m != nil {
			inBlock, blockIndent = true, len(m[1])
		}
		lines[i] = line
	}
	return lines
}

// parseError turns a yaml.v3 syntax error into a finding with a fix. yaml.v3
// gives the same message ("did not find expected key", often with no line or
// the wrong one) for several different causes, so the raw lines are inspected
// first to name the real one.
func (l *draftLoader) parseError(data []byte, err error) []Finding {
	lines := rawLines(data)
	parserLine, text := 0, err.Error()
	if m := yamlLineRe.FindStringSubmatch(text); m != nil {
		parserLine, _ = strconv.Atoi(m[1])
		text = m[2]
	}
	for i, line := range lines {
		if titleRawRe.MatchString(line) {
			value := strings.TrimSpace(strings.TrimPrefix(line, "title:"))
			l.fail(i+1, "the \"title\" value is not quoted, so YAML reads its leading \"[...]\" as a list and cannot parse the rest of the line. Wrap the whole title in double quotes: title: %s", yamlQuote(value))
			return l.findings
		}
	}
	for i, line := range lines {
		// yaml.v3 names the line before the offending one, and for "did not
		// find expected key" often a wrong line; that message gets no limit.
		// Any other message names a line the cause cannot lie beyond.
		if parserLine > 0 && !strings.Contains(text, "did not find expected key") && i+1 > parserLine+1 {
			break
		}
		m := quotedValueRe.FindStringSubmatch(line)
		if m == nil || !failsAlone(line) {
			continue
		}
		prefix, rest := m[1], m[2]
		closeAt := closingQuote(rest)
		if closeAt < 0 || strings.TrimSpace(rest[closeAt+1:]) == "" {
			continue
		}
		after := rest[closeAt+1:]
		var example string
		if trimmed := strings.TrimRight(rest, " \t"); strings.HasSuffix(trimmed, `"`) && len(trimmed)-1 > closeAt {
			example = prefix + `"` + escapeQuotes(trimmed[:len(trimmed)-1]) + `"`
		} else {
			example = prefix + `"` + escapeQuotes(rest[:closeAt]+after) + `"`
		}
		l.fail(i+1, "the closing quote of this double-quoted value comes before the end of the value: the string ends at %q and %q follows, which YAML cannot parse. Put the whole value in one pair of double quotes, writing \\\" for any quote character that is part of the text, or wrap the whole value in single quotes. For example: %s",
			`"`+rest[:closeAt]+`"`, strings.TrimSpace(after), strings.TrimSpace(example))
		return l.findings
	}
	if strings.Contains(text, "unknown anchor") {
		l.fail(parserLine, "the draft uses a YAML alias (%s). Drafts don't use tags, anchors or aliases: if the * starts text, put the whole value in double quotes (use a | block for content); otherwise delete it", text)
		return l.findings
	}
	if strings.Contains(text, "unknown escape character") || strings.Contains(text, "hexdecimal number") {
		line := parserLine
		if line == 0 {
			for i, raw := range lines {
				if badEscapeRe.MatchString(strings.ReplaceAll(raw, `\\`, "")) {
					line = i + 1
					break
				}
			}
		}
		offending := ""
		if line >= 1 && line <= len(lines) {
			offending = strings.TrimSpace(lines[line-1])
		}
		l.fail(line, "the double-quoted value on the line \"%s\" has a backslash that YAML reads as an escape sequence it cannot read (%s). Inside double quotes write \\\\ for each literal backslash, for example: %s. Or wrap the value in single quotes, where backslashes are literal.", offending, text, fixEscapes(offending))
		return l.findings
	}
	l.fail(parserLine, "the draft is not valid YAML (%s). Check: section content uses \"|\" block scalars indented consistently; the title and metadata values are double-quoted with inner \" and \\ escaped; no tabs.", text)
	return l.findings
}

// yamlQuote returns s as a YAML double-quoted scalar, escaping backslashes and
// quotes.
func yamlQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// unknownKey reports a key the draft format does not define. On the YAML path
// this is an error, unlike an unknown heading in a markdown body, because an
// unknown key almost always means lost content: a content line indented less
// than the rest of its block is read as a key. underIndent adds that hint.
func (l *draftLoader) unknownKey(k *yaml.Node, where string, valid []string, underIndent bool) {
	hint := "Remove it"
	if underIndent {
		hint = "If it is meant to be part of the text of the block above it, it is under-indented: indent every line of a | block at least as far as the block's first line. Otherwise remove it"
	}
	l.fail(k.Line, "unknown key %q %s. The valid keys here are: %s. %s", k.Value, where, strings.Join(valid, ", "), hint)
}

// kindName describes a node for a message.
func kindName(n *yaml.Node) string {
	switch {
	case n.Kind == yaml.MappingNode:
		return "a mapping"
	case n.Kind == yaml.SequenceNode:
		return "a list"
	case n.Kind == yaml.AliasNode:
		return "an alias"
	case n.Kind == yaml.ScalarNode && n.Tag == "!!null":
		return "nothing"
	default:
		return "text"
	}
}

// expectMapping reports a wrong node type unless n is a mapping. what names
// the node and keys describes what the mapping holds.
func (l *draftLoader) expectMapping(n *yaml.Node, what, keys string) bool {
	if n.Kind == yaml.MappingNode {
		return true
	}
	l.fail(n.Line, "%s must be a mapping of %s, but the draft gives %s", what, keys, kindName(n))
	return false
}

// comments reports every YAML comment the parser attached to any node, one
// finding per comment line. Drafts take no comments: YAML would drop the text,
// and a "#" line that was meant as content (a markdown heading, an issue
// reference) would be lost silently.
func (l *draftLoader) comments(n *yaml.Node) {
	l.markup(n)
	for _, c := range []struct{ text, kind string }{{n.HeadComment, "head"}, {n.LineComment, "line"}, {n.FootComment, "foot"}} {
		if c.text == "" || (c.kind == "line" && l.handled[n]) {
			continue
		}
		for _, text := range strings.Split(c.text, "\n") {
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			line := l.commentLine(text, n.Line, c.kind == "line")
			if c.kind == "line" {
				l.fail(line, "%q is a YAML comment: YAML treats a # at this position as a comment and would drop it. Drafts do not take comments, so delete it", text)
				continue
			}
			l.fail(line, "%q is a YAML comment: YAML treats a # at this position as a comment and would drop it. If the line is part of section content, indent it to the level of the | block's text. If it continues a title, metadata value or name, put the whole value in double quotes. Otherwise delete it: drafts don't take comments", text)
		}
	}
	for _, c := range n.Content {
		l.comments(c)
	}
}

// commentLine finds the raw line, nearest to hint, that holds a comment line
// and has not been reported yet; 0 when it cannot be located. An own-line
// comment must be the whole trimmed line and a line comment must end the
// line; lines inside a block scalar are content and never match.
func (l *draftLoader) commentLine(text string, hint int, trailing bool) int {
	if hint < 1 {
		hint = 1
	}
	match := func(i int) bool {
		raw := strings.TrimSpace(l.code[i-1])
		if l.used[i] || raw == "" {
			return false
		}
		if trailing {
			return strings.HasSuffix(raw, text)
		}
		return raw == text
	}
	for d := 0; d < len(l.lines); d++ {
		for _, i := range []int{hint + d, hint - d} {
			if i >= 1 && i <= len(l.lines) && match(i) {
				l.used[i] = true
				return i
			}
		}
	}
	return 0
}

// failsAlone reports whether a single raw line is not valid YAML on its own.
func failsAlone(line string) bool {
	var n yaml.Node
	return yaml.Unmarshal([]byte(strings.TrimSpace(line)), &n) != nil
}

// closingQuote returns the index in s of the first unescaped double quote,
// or -1.
func closingQuote(s string) int {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '"':
			return i
		}
	}
	return -1
}

// escapeQuotes escapes each unescaped double quote in s.
func escapeQuotes(s string) string {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '\\' && i+1 < len(s):
			sb.WriteByte(s[i])
			i++
			sb.WriteByte(s[i])
		case s[i] == '"':
			sb.WriteString(`\"`)
		default:
			sb.WriteByte(s[i])
		}
	}
	return sb.String()
}

// fixEscapes doubles every backslash in line that does not begin a valid
// YAML double-quoted escape.
func fixEscapes(line string) string {
	var sb strings.Builder
	for i := 0; i < len(line); i++ {
		if line[i] != '\\' {
			sb.WriteByte(line[i])
			continue
		}
		if i+1 >= len(line) {
			sb.WriteString(`\\`)
			continue
		}
		next := line[i+1]
		valid := strings.IndexByte("\\\"0abtnvfre NLP_/\t", next) >= 0
		if hex := map[byte]int{'x': 2, 'u': 4, 'U': 8}[next]; hex > 0 {
			valid = i+2+hex <= len(line) && isHex(line[i+2:i+2+hex])
		}
		if valid {
			sb.WriteByte('\\')
			sb.WriteByte(next)
			i++
		} else {
			sb.WriteString(`\\`)
		}
	}
	return sb.String()
}

func isHex(s string) bool {
	for i := 0; i < len(s); i++ {
		if !strings.ContainsRune("0123456789abcdefABCDEF", rune(s[i])) {
			return false
		}
	}
	return true
}

// markup reports a YAML anchor, alias or explicit tag on n: YAML applies them
// silently, and a tag or anchor ahead of a value (!Important, &ref) is removed
// from the text.
func (l *draftLoader) markup(n *yaml.Node) {
	var what, token string
	switch {
	case n.Kind == yaml.AliasNode:
		what, token = "alias", "*"+n.Value
	case n.Anchor != "":
		what, token = "anchor", "&"+n.Anchor
	case n.Style&yaml.TaggedStyle != 0:
		what, token = "tag", n.Tag
	default:
		return
	}
	l.fail(n.Line, "%q is a YAML %s. Drafts don't use tags, anchors or aliases: YAML would drop or change the text. If it is part of a value, a name or section content, put the whole value in double quotes (use a | block for content); otherwise delete it", token, what)
}
