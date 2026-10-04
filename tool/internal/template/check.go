package template

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"threehillpath.com/marvin-sdd/tool/internal/names"
	"threehillpath.com/marvin-sdd/tool/internal/parse"
)

// Source says which input produced a SectionMap.
type Source int

const (
	// SourceUnknown is the zero value: the caller did not say which input
	// path produced the map. Check reports it as an error.
	SourceUnknown Source = iota
	SourceYAML
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

// Heading is a "## " heading found in a markdown body. Content is the text
// below it, set only for unknown headings so Check can apply the content
// guards to it.
type Heading struct {
	Text    string
	Line    int
	Content string
}

// SectionMap is the input-independent shape the conformance check runs on.
type SectionMap struct {
	Source          Source
	Title           string
	TitleLine       int
	Metadata        map[string]Field
	Sections        map[string][]Entry
	UnknownHeadings []Heading // markdown only

	// RepeatedMetadata lists metadata lines that repeat an earlier key; the
	// map keeps the first value. Markdown only.
	RepeatedMetadata []RepeatedField

	// Preamble is the text above the first "## " heading (the whole body when
	// there is none), markdown only. Check applies the content guards to it.
	Preamble string

	// MisplacedMetadata lists metadata-shaped lines that GitHub does not show
	// as metadata (markdown only).
	MisplacedMetadata []MisplacedField
}

// MisplacedField is a "**Key:**" line above the first heading that sits in a
// code fence (InFence) or directly below the non-blank line Above.
type MisplacedField struct {
	Key     string
	Line    int
	InFence bool
	// Indented is set for a line indented 4 or more columns after a blank
	// line, which GitHub shows as code.
	Indented  bool
	Above     string // first line of the run that is not metadata
	AboveLine int
}

// RepeatedField is a metadata key written a second time, at Line, after
// FirstLine.
type RepeatedField struct {
	Key       string
	FirstLine int
	Line      int
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
	if !sc.loaded {
		c.add(SeverityError, "draft", 0, "Check was given a Schema that did not come from LoadSchema, so its title kind and validation are unset. This is a bug in the calling code: build the Schema with LoadSchema(origin, data) and pass that.")
		return c.res
	}
	if m.Source == SourceUnknown {
		c.add(SeverityError, "draft", 0, "SectionMap.Source is not set, so the fix text cannot match the input path. This is a bug in the calling code: set Source to SourceYAML for a YAML draft or SourceMarkdown for a markdown body.")
		return c.res
	}
	c.checkTitle()
	c.checkMetadata()
	c.checkCrossRefs()
	c.checkSections()
	c.checkMarkdownOnly()
	return c.res
}

func (c *checker) checkTitle() {
	title := strings.TrimSpace(c.m.Title)
	line := c.m.TitleLine
	if title == "" {
		c.add(SeverityError, "title", line, "title is missing. The schema expects a title like %q. %s", c.sc.TitlePrefix,
			c.fix("Set \"title:\" in the draft.", "Supply the issue title with --title."))
		return
	}
	if strings.ContainsAny(title, "\r\n") {
		c.add(SeverityError, "title", line, "title %q spans more than one line. Use a single line.", title)
	}
	c.checkRawHTML("title", line, fmt.Sprintf("the title %q", title), title,
		c.fix("edit \"title:\" in the draft", "edit the issue title"))
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

var (
	issueRefRe = regexp.MustCompile(`^#\d+`)
	planIdRe   = regexp.MustCompile(`PLAN-(\d{5})`)
)

// titleNumber returns the plan or task number of the title's leading
// identifier, and whether the title is a task title. ok is false when the
// title does not classify.
func (c *checker) titleNumber() (n int, task, ok bool) {
	title := strings.TrimSpace(c.m.Title)
	kind, found := parse.Classify(title)
	if !found {
		return 0, false, false
	}
	tok := leadingBracket.FindString(title)
	if kind == names.Task {
		n, ok = parse.TaskIdent(tok)
		return n, true, ok
	}
	id, ok := parse.PlanIdent(tok)
	return id.Plan, false, ok
}

// checkCrossRefs verifies metadata keys that must agree with the title. The
// checks are keyed by key name so project overrides that keep the names
// inherit them.
func (c *checker) checkCrossRefs() {
	n, task, ok := c.titleNumber()
	if !ok {
		return
	}
	title := strings.TrimSpace(c.m.Title)
	for _, key := range c.sc.Metadata {
		f, present := c.m.Metadata[key]
		v := strings.TrimSpace(f.Value)
		if !present || v == "" {
			continue
		}
		loc := "metadata:" + key
		setFix := fmt.Sprintf("Set %q to %%s, or fix the title.", key)
		switch key {
		case "Plan Number", "Task Number":
			if (key == "Task Number") != task {
				continue // title family differs; the title kind check reports it
			}
			want := names.PlanNumber(n)
			if task {
				want = names.TaskNumber(n)
			}
			if v != want {
				c.add(SeverityError, loc, f.Line, "value %q does not match the title's %s %s (title %q). "+setFix, v, strings.ToLower(key), want, title, fmt.Sprintf("%q", want))
			}
		case "Source Issue", "Architecture Plan", "Implementation Plan":
			if !issueRefRe.MatchString(v) {
				c.add(SeverityError, loc, f.Line, "value %q does not begin with an issue reference. The schema expects \"#<n>\", optionally followed by text like \"#56 ([PLAN-00041-ARCH])\". Set %q to start with the issue number.", v, key)
				continue
			}
			if task {
				continue
			}
			if m := planIdRe.FindStringSubmatch(v[len(issueRefRe.FindString(v)):]); m != nil {
				if got, _ := strconv.Atoi(m[1]); got != n {
					c.add(SeverityError, loc, f.Line, "value %q names plan %s, but the title's plan number is %s. "+setFix, v, m[0], names.PlanNumber(n), fmt.Sprintf("a reference to %s", names.PlanNumber(n)))
				}
			}
		}
	}
}

// fix returns the fix text for the input path m came from.
func (c *checker) fix(yamlFix, mdFix string) string {
	if c.m.Source == SourceMarkdown {
		return mdFix
	}
	return yamlFix
}

// startsContainerRe matches a line that opens a block quote or list item.
var startsContainerRe = regexp.MustCompile(`^(?:>|[-*+]\s|\d+[.)]\s)`)

// misplaced reports whether key has a metadata-shaped line that was rejected.
func (c *checker) misplaced(key string) bool {
	for _, f := range c.m.MisplacedMetadata {
		if f.Key == key {
			return true
		}
	}
	return false
}

func (c *checker) checkMetadata() {
	for _, key := range c.sc.Metadata {
		loc := "metadata:" + key
		f, ok := c.m.Metadata[key]
		if !ok {
			if c.misplaced(key) {
				continue // reported as a misplaced line, which says how to fix it
			}
			c.add(SeverityError, loc, 0, "metadata key %q is missing. The schema requires every metadata key. %s",
				key, c.fix(fmt.Sprintf("Add a %q key under \"metadata:\" in the draft.", key),
					fmt.Sprintf("Add a line \"**%s:** <value>\" above the first \"## \" heading.", key)))
			continue
		}
		v := strings.TrimSpace(f.Value)
		if v == "" {
			c.add(SeverityError, loc, f.Line, "metadata key %q has an empty value. The schema requires a value. %s",
				key, c.fix(fmt.Sprintf("Set %q under \"metadata:\" in the draft to a value.", key),
					fmt.Sprintf("Set \"**%s:**\" to a value.", key)))
			continue
		}
		c.checkRawHTML(loc, f.Line, fmt.Sprintf("metadata value %q for %q", v, key), v,
			c.fix(fmt.Sprintf("edit %q under \"metadata:\" in the draft", key), fmt.Sprintf("edit the \"**%s:**\" line", key)))
		if strings.ContainsAny(v, "\r\n") {
			c.add(SeverityError, loc, f.Line, "metadata value %q for %q spans more than one line. Use a single line.", v, key)
		}
	}
}

// isNamed reports whether sec is a numbered section whose entries carry
// their own heading text.
func isNamed(sec SchemaSection) bool {
	return sec.Numbered && sec.Named != nil && *sec.Named
}

// label describes a section in a message: its quoted heading, or, for a named
// section whose heading is only a placeholder, its quoted id.
func label(sec SchemaSection) string {
	if isNamed(sec) {
		return fmt.Sprintf("%q", sec.ID)
	}
	return fmt.Sprintf("%q", sec.Heading)
}

// missingFix returns the fix for a missing section, shaped by the section's
// kind and the input path.
func (c *checker) missingFix(sec SchemaSection) string {
	var y, md string
	switch {
	case isNamed(sec):
		y = fmt.Sprintf("Add %q under \"sections:\" in the draft as a list of entries, each with \"name:\" and \"content: |\".", sec.ID+":")
		md = "Add one or more headings like \"## <n>. <Name>\", where <Name> is the entry's own name and <n> continues the consecutive numbering."
	case sec.Repeatable && sec.Numbered:
		y = fmt.Sprintf("Add %q under \"sections:\" in the draft as a list of blocks, each starting with \"- |\".", sec.ID+":")
		md = fmt.Sprintf("Add one or more headings like \"## <n>. %s\" with content, where <n> continues the consecutive numbering.", sec.Heading)
	case sec.Repeatable:
		y = fmt.Sprintf("Add %q under \"sections:\" in the draft as a list of blocks, each starting with \"- |\".", sec.ID+":")
		md = fmt.Sprintf("Add one or more \"## %s\" headings with content.", sec.Heading)
	case sec.Numbered:
		y = fmt.Sprintf("Add a %q block under \"sections:\" in the draft.", sec.ID+": |")
		md = fmt.Sprintf("Add a \"## <n>. %s\" heading with content.", sec.Heading)
	default:
		y = fmt.Sprintf("Add a %q block under \"sections:\" in the draft.", sec.ID+": |")
		md = fmt.Sprintf("Add a \"## %s\" heading with content.", sec.Heading)
	}
	return c.fix(y, md)
}

// emptyFix returns the fix for an empty entry; optional sections may also be
// removed.
func (c *checker) emptyFix(sec SchemaSection, e Entry) string {
	var y, md string
	name := strings.TrimSpace(e.Name)
	switch {
	case isNamed(sec):
		if name != "" {
			y = fmt.Sprintf("Fill \"content: |\" of the entry named %q in %q with content", name, sec.ID)
		} else {
			y = fmt.Sprintf("Fill \"content: |\" of the unnamed entry in %q with content and give it a \"name:\"", sec.ID)
		}
		md = fmt.Sprintf("Fill the %q section with content", templateHeading(sec, e))
	case sec.Repeatable:
		y = fmt.Sprintf("Fill the empty \"- |\" block in %q with content", sec.ID)
		md = fmt.Sprintf("Fill the %q section with content", templateHeading(sec, e))
	default:
		y = fmt.Sprintf("Fill the %q block in the draft with content", sec.ID)
		md = fmt.Sprintf("Fill the %q section with content", templateHeading(sec, e))
	}
	fix := c.fix(y, md)
	if !sec.Required {
		fix += ", or remove it"
	}
	return fix + "."
}

func (c *checker) checkSections() {
	for _, sec := range c.sc.Sections {
		loc := "section:" + sec.ID
		entries := c.m.Sections[sec.ID]
		if len(entries) == 0 {
			if sec.Required {
				c.add(SeverityError, loc, 0, "required section %s is missing. %s", label(sec), c.missingFix(sec))
			}
			continue
		}
		if !sec.Repeatable && len(entries) > 1 {
			c.add(SeverityError, loc, entries[1].Line, "section %s is not repeatable but has %d entries. Merge them into one.", label(sec), len(entries))
		}
		for _, e := range entries {
			if strings.TrimSpace(e.Content) == "" {
				sev, word := SeverityWarning, "optional"
				if sec.Required {
					sev, word = SeverityError, "required"
				}
				what := "section " + label(sec)
				if n := strings.TrimSpace(e.Name); isNamed(sec) && n != "" {
					what = fmt.Sprintf("entry %q of section %s", n, label(sec))
				}
				c.add(sev, loc, e.Line, "%s %s is empty. %s", word, what, c.emptyFix(sec, e))
			}
			if isNamed(sec) {
				c.checkRawHTML(loc, e.Line, fmt.Sprintf("the name %q of an entry of numbered section %s", e.Name, label(sec)), e.Name,
					c.fix(fmt.Sprintf("edit \"name:\" of that %q entry in the draft", sec.ID), "edit the heading text after the number"))
			}
			if isNamed(sec) && strings.ContainsAny(e.Name, "\r\n") {
				c.add(SeverityError, loc, e.Line, "the name %q of an entry of numbered section %s contains a line break, so it would render as more than one heading. %s", e.Name, label(sec),
					c.fix(fmt.Sprintf("Set \"name:\" of that %q entry to a single line of text.", sec.ID),
						"Put the heading text on a single line."))
			}
			if isNamed(sec) && strings.TrimSpace(e.Name) == "" {
				c.add(SeverityError, loc, e.Line, "an entry of numbered section %s has an empty name. The schema expects each entry to be named. %s", label(sec),
					c.fix(fmt.Sprintf("Give every %q entry a non-empty \"name:\" in the draft.", sec.ID),
						"Give the heading text after the number, like \"## 1. <Name>\"."))
			}
			c.checkContentStructure("section:"+sec.ID, "section "+label(sec), e.Content, e.Line)
		}
	}
}

var (
	fenceRe = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})(.*)$")
	h2Re    = regexp.MustCompile(`^ {0,3}##(\s|$)`)
)

// openFence is a code fence that the end of the scanned text left open.
type openFence struct {
	Run  string // the opening run of backticks or tildes
	Line int    // one-based line the fence opens on
}

// setextHit is a paragraph line directly followed by a "===" or "---" line,
// which Markdown renders as a heading.
type setextHit struct {
	Text      string // the line that becomes the heading
	Line      int    // its one-based line
	Underline string // the "===" or "---" line
}

// contentScan is what scanContent learns about a piece of section content.
type contentScan struct {
	Headings []Heading  // "## " headings outside fences
	Fence    *openFence // fence left open at the end, if any
	Setext   *setextHit // first line turned into a heading by an underline
	HTML     *htmlHit   // first raw HTML construct outside code
	// InFence[i] is true when line i+1 is a fence line or inside a fence.
	InFence []bool
}

// htmlHit is a banned raw HTML construct.
type htmlHit struct {
	Tag  string // as written, e.g. "<details>" or "<!--"
	Line int    // one-based line within the scanned text
}

var (
	setextRe = regexp.MustCompile(`^ {0,3}(?:=+|-+)[ \t]*$`)
	// rawHTMLRe matches the constructs that can hide or swallow later
	// sections: comments and the details, pre, script, style and textarea
	// tags, opening or closing, in any case.
	rawHTMLRe = regexp.MustCompile(`(?i)<!--|</?(?:details|pre|script|style|textarea)(?:>|[\s/]|$)`)
)

// maskCodeSpans returns text with every code span, delimiters included,
// replaced by spaces (newlines kept), following CommonMark: a run of n
// backticks closes at the next run of exactly n backticks, an unmatched run
// is literal text, a backslash makes the next character literal outside code
// spans (so "\\`" is not a delimiter), and spans may cross lines.
func maskCodeSpans(text string) string {
	b := []byte(text)
	for i := 0; i < len(b); {
		switch b[i] {
		case '\\':
			i += 2
		case '`':
			n := 0
			for i+n < len(b) && b[i+n] == '`' {
				n++
			}
			closeAt := -1
			for j := i + n; j < len(b); {
				if b[j] != '`' {
					j++
					continue
				}
				m := 0
				for j+m < len(b) && b[j+m] == '`' {
					m++
				}
				if m == n {
					closeAt = j
					break
				}
				j += m
			}
			if closeAt < 0 {
				i += n
				continue
			}
			for k := i; k < closeAt+n; k++ {
				if b[k] != '\n' {
					b[k] = ' '
				}
			}
			i = closeAt + n
		default:
			i++
		}
	}
	return string(b)
}

// rawHTMLAt returns the first banned raw HTML construct in text (which may
// span lines of one paragraph) outside code spans, with its closing ">" when
// it follows directly, and its byte offset.
func rawHTMLAt(text string) (tag string, off int, ok bool) {
	masked := maskCodeSpans(text)
	loc := rawHTMLRe.FindStringIndex(masked)
	if loc == nil {
		return "", 0, false
	}
	return strings.TrimRight(masked[loc[0]:loc[1]], " \t/"), loc[0], true
}

// rawHTML is rawHTMLAt for a single-line field.
func rawHTML(text string) (string, bool) {
	tag, _, ok := rawHTMLAt(text)
	return tag, ok
}

// scanContent walks body once, tracking CommonMark fences: an opening run of
// three or more backticks or tildes (up to three spaces of indent) closes only
// on a run of the same character that is at least as long and carries nothing
// but whitespace after it. Outside fences it records the "## " headings, the
// first underline directly below a non-blank line, and the first raw HTML
// construct outside inline code.
func scanContent(body string) contentScan {
	var out contentScan
	var fenceCh byte
	fenceLen := 0
	prevText, prevNo := "", 0 // non-blank line directly above
	var para []string         // consecutive non-blank lines outside fences
	paraStart := 0            // one-based line of para[0]
	flush := func() {
		if len(para) > 0 && out.HTML == nil {
			if tag, off, ok := rawHTMLAt(strings.Join(para, "\n")); ok {
				out.HTML = &htmlHit{Tag: tag, Line: paraStart + strings.Count(strings.Join(para, "\n")[:off], "\n")}
			}
		}
		para = nil
	}
	allLines := strings.Split(body, "\n")
	out.InFence = make([]bool, len(allLines))
	for i, line := range allLines {
		line = strings.TrimRight(line, "\r")
		prev, prevLineNo := prevText, prevNo
		out.InFence[i] = fenceCh != 0
		prevText = ""
		if m := fenceRe.FindStringSubmatch(line); m != nil {
			run, rest := m[1], m[2]
			switch {
			case fenceCh == 0:
				if run[0] != '`' || !strings.Contains(rest, "`") {
					flush()
					fenceCh, fenceLen = run[0], len(run)
					out.InFence[i] = true
					out.Fence = &openFence{Run: run, Line: i + 1}
				}
				continue
			case run[0] == fenceCh && len(run) >= fenceLen && strings.TrimSpace(rest) == "":
				fenceCh, fenceLen = 0, 0
				out.Fence = nil
				continue
			}
		}
		if fenceCh != 0 {
			continue
		}
		if h2Re.MatchString(line) {
			text := strings.TrimSpace(strings.TrimLeft(line, " ")[2:])
			out.Headings = append(out.Headings, Heading{Text: text, Line: i + 1})
		}
		if strings.TrimSpace(line) == "" {
			flush()
		} else {
			if len(para) == 0 {
				paraStart = i + 1
			}
			para = append(para, line)
		}
		// Conservative setext rule: an underline-shaped line directly after
		// any non-blank line is reported, whatever that line contains.
		if prev != "" && setextRe.MatchString(line) {
			if out.Setext == nil {
				out.Setext = &setextHit{Text: strings.TrimSpace(prev), Line: prevLineNo, Underline: strings.TrimSpace(line)}
			}
		} else if strings.TrimSpace(line) != "" {
			prevText, prevNo = line, i+1
		}
	}
	flush()
	return out
}

// scanFences returns the "## " headings outside fences and, when the text
// ends inside a fence, that fence.
func scanFences(body string) ([]Heading, *openFence) {
	s := scanContent(body)
	return s.Headings, s.Fence
}

// FindH2Lines returns, for each "## " heading line of body that is outside a
// fenced code block, its one-based line number and its text without the "##"
// marker, trimmed (the same convention as SectionMap.UnknownHeadings). Fences
// follow CommonMark, see scanFences. Markdown parsers should use this to
// split a body into sections.
func FindH2Lines(body string) []Heading {
	hs, _ := scanFences(body)
	return hs
}

// contentLine returns the trimmed text of the one-based line n of content.
func contentLine(content string, n int) string {
	lines := strings.Split(content, "\n")
	if n < 1 || n > len(lines) {
		return ""
	}
	return strings.TrimSpace(strings.TrimRight(lines[n-1], "\r"))
}

// checkContentStructure reports content that would break the document's
// structure: a "## " heading outside a fence, which would become a new
// section, and a fence that is never closed, which would swallow every later
// section.
func (c *checker) checkContentStructure(loc, what, content string, line int, whereOverride ...string) {
	where := c.fix(fmt.Sprintf("inside the %q block", strings.TrimPrefix(loc, "section:")), "under that heading")
	unit := "the section"
	if len(whereOverride) > 0 {
		where, unit = whereOverride[0], "the body" // preamble lines count from the top of the body
	}
	add := func(format string, args ...any) {
		c.add(SeverityError, loc, line, strings.ReplaceAll(format, "of the section", "of "+unit), args...)
	}
	if norm := strings.ReplaceAll(content, "\r\n", "\n"); strings.Contains(norm, "\r") {
		n := strings.Count(norm[:strings.Index(norm, "\r")], "\n") + 1
		add("content of %s has a lone carriage return on line %d of the section, which GitHub renders as a line break the structure checks cannot see (for example \"a\\r## X\" becomes a heading). Fix: replace the carriage return with a line break (or remove it), %s.",
			what, n, where)
	}
	scan := scanContent(content)
	hs, open := scan.Headings, scan.Fence
	if h := scan.HTML; h != nil {
		add("content of %s contains raw HTML %q on line %d of the section. Raw HTML could hide or swallow the sections after it when rendered, so drafts don't allow it. Wrap it in backticks as inline code (for example `<details>`) or remove it, %s.",
			what, h.Tag, h.Line, where)
	}
	if len(hs) > 0 {
		add("content of %s contains the heading %q on line %d of the section, which would become a new top-level section when rendered. The schema expects sub-headings below \"## \". Use \"###\" instead.", what, contentLine(content, hs[0].Line), hs[0].Line)
	}
	if h := scan.Setext; h != nil {
		if strings.HasPrefix(h.Underline, "=") {
			add("content of %s has the line %q on line %d of the section directly above the underline %q, which makes it a heading when rendered and would split the section. Remove the %q line, or write the heading as \"### %s\" %s.",
				what, h.Text, h.Line, h.Underline, h.Underline, h.Text, where)
		} else {
			add("content of %s has the line %q on line %d of the section directly above the underline %q, which makes it a heading when rendered and would split the section. If you meant a horizontal rule, put a blank line before %q; otherwise write the heading as \"### %s\" or remove the underline %s.",
				what, h.Text, h.Line, h.Underline, h.Underline, h.Text, where)
		}
	}
	if open != nil {
		ch := fmt.Sprintf("%q", string(open.Run[0]))
		add("content of %s opens a code fence %q on line %d of the section that is never closed, so every later section would render as code and be lost to the parser. Close it with a matching fence line (the same character, %s, at least %d long, nothing else on the line) %s.",
			what, open.Run, open.Line, ch, len(open.Run), where)
	}
}

// checkRawHTML reports raw HTML in a single-line field (title, metadata
// value, entry name). what names the field in the message and fix is the
// path-specific place to change it.
func (c *checker) checkRawHTML(loc string, line int, what, text, fix string) {
	tag, ok := rawHTML(text)
	if !ok {
		return
	}
	c.add(SeverityError, loc, line, "%s contains raw HTML %q. Raw HTML could hide or swallow the sections after it when rendered, so drafts don't allow it. Wrap it in backticks as inline code (for example `<details>`) or remove it: %s.", what, tag, fix)
}

// checkMarkdownOnly applies the rules that only make sense for a markdown
// body: on the YAML path unknown keys are loader errors and rendering imposes
// schema order and numbering.
func (c *checker) checkMarkdownOnly() {
	if c.m.Source != SourceMarkdown {
		return
	}
	known := map[string]bool{}
	for _, key := range c.sc.Metadata {
		known[key] = true
	}
	c.checkContentStructure("draft", "the text above the first \"## \" heading", c.m.Preamble, 1, "above the first \"## \" heading")
	for _, f := range c.m.MisplacedMetadata {
		if !known[f.Key] {
			continue
		}
		loc := "metadata:" + f.Key
		switch {
		case f.Indented:
			c.add(SeverityError, loc, f.Line, "the \"**%s:**\" line (line %d) is indented 4 or more spaces, so GitHub shows it as code, not as metadata. Remove the indentation.", f.Key, f.Line)
		case f.InFence:
			c.add(SeverityError, loc, f.Line, "the \"**%s:**\" line (line %d) is inside a code fence, so GitHub shows it as code, not as metadata. Move it out of the code fence, above the first \"## \" heading.", f.Key, f.Line)
		default:
			quoted := f.Above
			container := ""
			if startsContainerRe.MatchString(f.Above) {
				container = ", so GitHub shows it inside that quote or list item"
			}
			if len(quoted) > 60 {
				quoted = quoted[:60] + "..."
			}
			c.add(SeverityError, loc, f.Line, "the \"**%s:**\" line (line %d) follows the line %q (line %d) with no blank line between. Metadata must be the first line of the body, follow a blank line, or follow another metadata line%s. Add a blank line after line %d.", f.Key, f.Line, quoted, f.AboveLine, container, f.AboveLine)
		}
	}
	for _, r := range c.m.RepeatedMetadata {
		c.add(SeverityError, "metadata:"+r.Key, r.Line, "metadata key %q appears twice (line %d and line %d). Remove the duplicate line or merge its value into the first.", r.Key, r.FirstLine, r.Line)
	}
	for _, h := range c.m.UnknownHeadings {
		c.checkContentStructure("draft", fmt.Sprintf("the unknown heading \"## %s\"", h.Text), h.Content, h.Line)
		c.add(SeverityWarning, "draft", h.Line, "heading \"## %s\" is not a section of schema %s. Rename it to one of the schema's headings (%s), or remove it.", h.Text, c.sc.Type, c.expectedHeadings())
	}
	var extra []string
	for key := range c.m.Metadata {
		if !known[key] {
			extra = append(extra, key)
		}
	}
	sort.Strings(extra)
	for _, key := range extra {
		f := c.m.Metadata[key]
		c.checkRawHTML("metadata:"+key, f.Line, fmt.Sprintf("metadata value %q for %q", strings.TrimSpace(f.Value), key), f.Value,
			fmt.Sprintf("edit the \"**%s:**\" line, or remove it", key))
		c.add(SeverityWarning, "metadata:"+key, c.m.Metadata[key].Line, "metadata key %q is not in schema %s. Remove the \"**%s:**\" line.", key, c.sc.Type, key)
	}
	c.checkOrder()
	c.checkNumbering()
}

// templateHeading is the markdown heading an entry of sec is written as,
// using the entry's own name and number when known and placeholders otherwise.
func templateHeading(sec SchemaSection, e Entry) string {
	if !sec.Numbered {
		return "## " + sec.Heading
	}
	num := "<n>"
	if e.Number > 0 {
		num = strconv.Itoa(e.Number)
	}
	if isNamed(sec) {
		name := strings.TrimSpace(e.Name)
		if name == "" {
			name = "<Name>"
		}
		return fmt.Sprintf("## %s. %s", num, name)
	}
	return fmt.Sprintf("## %s. %s", num, sec.Heading)
}

// expectedHeadings lists the schema's headings, quoted and in schema order,
// with numbered ones shown as "## <n>. ...".
func (c *checker) expectedHeadings() string {
	var hs []string
	for _, sec := range c.sc.Sections {
		hs = append(hs, fmt.Sprintf("%q", templateHeading(sec, Entry{})))
	}
	return strings.Join(hs, ", ")
}

// checkOrder warns about entries outside the schema's order. Every entry is
// placed by line; the entries in the longest non-decreasing run of schema
// positions are in order and the rest are reported, so one displaced entry
// yields one warning.
func (c *checker) checkOrder() {
	type placed struct {
		sec  SchemaSection
		e    Entry
		pos  int // index of sec in the schema
		rank int // position in schema order (section index, then line)
	}
	var ps []placed
	for i, sec := range c.sc.Sections {
		for _, e := range c.m.Sections[sec.ID] {
			if e.Line > 0 {
				ps = append(ps, placed{sec: sec, e: e, pos: i})
			}
		}
	}
	bySchema := make([]int, len(ps)) // indexes into ps, in schema order
	for i := range bySchema {
		bySchema[i] = i
	}
	sort.SliceStable(bySchema, func(a, b int) bool {
		x, y := ps[bySchema[a]], ps[bySchema[b]]
		if x.pos != y.pos {
			return x.pos < y.pos
		}
		return x.e.Line < y.e.Line
	})
	for r, i := range bySchema {
		ps[i].rank = r
	}
	sort.SliceStable(ps, func(a, b int) bool { return ps[a].e.Line < ps[b].e.Line })
	n := len(ps)
	length := make([]int, n)
	prev := make([]int, n)
	best := -1
	for i := range ps {
		length[i], prev[i] = 1, -1
		for j := 0; j < i; j++ {
			if ps[j].pos <= ps[i].pos && length[j]+1 > length[i] {
				length[i], prev[i] = length[j]+1, j
			}
		}
		if best < 0 || length[i] > length[best] {
			best = i
		}
	}
	inRun := make([]bool, n)
	for i := best; i >= 0; i = prev[i] {
		inRun[i] = true
	}
	byRank := make([]placed, n)
	for _, p := range ps {
		byRank[p.rank] = p
	}
	for i, p := range ps {
		if inRun[i] {
			continue
		}
		h := templateHeading(p.sec, p.e)
		where := ""
		switch {
		case p.rank > 0:
			q := byRank[p.rank-1]
			where = fmt.Sprintf("after %q", templateHeading(q.sec, q.e))
		case n > 1:
			q := byRank[p.rank+1]
			where = fmt.Sprintf("before %q", templateHeading(q.sec, q.e))
		}
		c.add(SeverityWarning, "section:"+p.sec.ID, p.e.Line, "heading %q is out of schema order. Move it %s; the schema lists its sections in this order: %s.", h, where, c.expectedHeadings())
	}
}

// checkNumbering warns at the first numbered heading that breaks the 1, 2, 3
// sequence across all numbered sections in line order.
func (c *checker) checkNumbering() {
	type item struct {
		sec SchemaSection
		e   Entry
	}
	var items []item
	for _, sec := range c.sc.Sections {
		if !sec.Numbered {
			continue
		}
		for _, e := range c.m.Sections[sec.ID] {
			items = append(items, item{sec, e})
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].e.Line < items[j].e.Line })
	for i, it := range items {
		if it.e.Number == i+1 {
			continue
		}
		c.add(SeverityWarning, "section:"+it.sec.ID, it.e.Line, "numbered heading %q breaks the sequence. The schema expects consecutive numbers from 1. Renumber it to \"## %d.\".", templateHeading(it.sec, it.e), i+1)
		return
	}
}

// Format renders the result as plain text: a "schema: <type> (<origin>)"
// line, then one line per finding. Errors come before warnings; within each
// group, findings with a known line come first in line order, then the rest
// in the order they were found.
func (r Result) Format() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "schema: %s (%s)\n", r.Type, r.Origin)
	for _, sev := range []Severity{SeverityError, SeverityWarning} {
		var group []Finding
		for _, f := range r.Findings {
			if f.Severity == sev {
				group = append(group, f)
			}
		}
		sort.SliceStable(group, func(i, j int) bool {
			a, b := group[i].Line, group[j].Line
			if (a == 0) != (b == 0) {
				return b == 0
			}
			return a < b
		})
		for _, f := range group {
			if f.Line > 0 {
				fmt.Fprintf(&sb, "%s %s line %d: %s\n", f.Severity, f.Location, f.Line, f.Message)
			} else {
				fmt.Fprintf(&sb, "%s %s: %s\n", f.Severity, f.Location, f.Message)
			}
		}
	}
	return sb.String()
}
