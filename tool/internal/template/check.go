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

func (c *checker) checkMetadata() {
	for _, key := range c.sc.Metadata {
		loc := "metadata:" + key
		f, ok := c.m.Metadata[key]
		if !ok {
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
		if strings.Contains(v, "\n") {
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
			if isNamed(sec) && strings.TrimSpace(e.Name) == "" {
				c.add(SeverityError, loc, e.Line, "an entry of numbered section %s has an empty name. The schema expects each entry to be named. %s", label(sec),
					c.fix(fmt.Sprintf("Give every %q entry a non-empty \"name:\" in the draft.", sec.ID),
						"Give the heading text after the number, like \"## 1. <Name>\"."))
			}
			c.checkContentStructure(sec, e)
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

// openBlock is a raw HTML construct that the end of the scanned text left
// open: Tag is what opened it ("<!--", "<pre>", ...), Line where.
type openBlock struct {
	Tag  string
	Line int // one-based
}

// contentScan is what scanContent learns about a piece of section content.
type contentScan struct {
	Headings []Heading  // "## " headings outside fences
	Fence    *openFence // fence left open at the end, if any
	Comment  *openBlock // block-level "<!--" never closed by "-->"
	Setext   *setextHit // first paragraph line turned into a heading by an underline
	Raw      *openBlock // "<pre>", "<script>", "<style>" or "<textarea>" never closed
	Details  *openBlock // outermost "<details>" never closed by "</details>"
}

var (
	commentOpenRe = regexp.MustCompile(`^ {0,3}<!--`)
	detailsTagRe  = regexp.MustCompile(`(?i)<(/?)details(?:\s[^>]*)?>`)
	rawOpenRe     = regexp.MustCompile(`(?i)^ {0,3}<(pre|script|style|textarea)(?:\s|>|$)`)
	rawCloseRe    = regexp.MustCompile(`(?i)</(?:pre|script|style|textarea)>`)
	setextRe      = regexp.MustCompile(`^ {0,3}(?:=+|-+)[ \t]*$`)
	inlineCodeRe  = regexp.MustCompile("`[^`]*`")
)

// scanContent walks body once, tracking CommonMark fences: an opening run of
// three or more backticks or tildes (up to three spaces of indent) closes only
// on a run of the same character that is at least as long and carries nothing
// but whitespace after it. Outside fences it records the "## " headings and
// the raw HTML blocks that are never closed.
func scanContent(body string) contentScan {
	var out contentScan
	var fenceCh byte
	fenceLen := 0
	inComment := false
	var details []int         // lines of <details> not yet closed
	prevText, prevNo := "", 0 // candidate paragraph line directly above
	for i, line := range strings.Split(body, "\n") {
		line = strings.TrimRight(line, "\r")
		prev, prevLineNo := prevText, prevNo
		prevText = ""
		if m := fenceRe.FindStringSubmatch(line); m != nil {
			run, rest := m[1], m[2]
			switch {
			case fenceCh == 0:
				if run[0] != '`' || !strings.Contains(rest, "`") {
					fenceCh, fenceLen = run[0], len(run)
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
		if inComment {
			if strings.Contains(line, "-->") {
				inComment = false
				out.Comment = nil
			}
			continue
		}
		if commentOpenRe.MatchString(line) {
			rest := line[strings.Index(line, "<!--")+len("<!--"):]
			if !strings.Contains(rest, "-->") {
				inComment = true
				out.Comment = &openBlock{Tag: "<!--", Line: i + 1}
			}
			continue
		}
		if out.Raw != nil {
			if rawCloseRe.MatchString(line) {
				out.Raw = nil
			}
			continue
		}
		if m := rawOpenRe.FindStringSubmatch(line); m != nil {
			if !rawCloseRe.MatchString(line) {
				out.Raw = &openBlock{Tag: "<" + strings.ToLower(m[1]) + ">", Line: i + 1}
			}
			continue
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
		for _, m := range detailsTagRe.FindAllStringSubmatch(inlineCodeRe.ReplaceAllString(line, ""), -1) {
			if m[1] == "" {
				details = append(details, i+1)
			} else if len(details) > 0 {
				details = details[:len(details)-1]
			}
		}
	}
	if len(details) > 0 {
		out.Details = &openBlock{Tag: "<details>", Line: details[0]}
	}
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
func (c *checker) checkContentStructure(sec SchemaSection, e Entry) {
	loc := "section:" + sec.ID
	scan := scanContent(e.Content)
	hs, open := scan.Headings, scan.Fence
	where := fmt.Sprintf("inside the %q block", sec.ID)
	if c.m.Source == SourceMarkdown {
		where = "under that heading"
	}
	if b := scan.Comment; b != nil {
		c.add(SeverityError, loc, e.Line, "content of section %s opens an HTML comment %q on line %d of the section that is never closed by \"-->\", so every later section would be hidden when rendered. Close it with \"-->\" %s, or remove the comment.",
			label(sec), b.Tag, b.Line, where)
	}
	if len(hs) > 0 {
		c.add(SeverityError, loc, e.Line, "content of section %s contains the heading %q on line %d of the section, which would become a new top-level section when rendered. The schema expects sub-headings below \"## \". Use \"###\" instead.", label(sec), contentLine(e.Content, hs[0].Line), hs[0].Line)
	}
	if h := scan.Setext; h != nil {
		if strings.HasPrefix(h.Underline, "=") {
			c.add(SeverityError, loc, e.Line, "content of section %s has the line %q on line %d of the section directly above the underline %q, which makes it a heading when rendered and would split the section. Remove the %q line, or write the heading as \"### %s\" %s.",
				label(sec), h.Text, h.Line, h.Underline, h.Underline, h.Text, where)
		} else {
			c.add(SeverityError, loc, e.Line, "content of section %s has the line %q on line %d of the section directly above the underline %q, which makes it a heading when rendered and would split the section. If you meant a horizontal rule, put a blank line before %q; otherwise write the heading as \"### %s\" or remove the underline %s.",
				label(sec), h.Text, h.Line, h.Underline, h.Underline, h.Text, where)
		}
	}
	if b := scan.Raw; b != nil {
		c.add(SeverityError, loc, e.Line, "content of section %s opens a %q block on line %d of the section that is never closed by %q, so every later section would render as part of that block. Add a %q line %s, or remove the block.",
			label(sec), b.Tag, b.Line, "</"+b.Tag[1:], "</"+b.Tag[1:], where)
	}
	if b := scan.Details; b != nil {
		c.add(SeverityError, loc, e.Line, "content of section %s opens a %q block on line %d of the section that is never closed by \"</details>\", so every later section would render inside the collapsed block. Add a \"</details>\" line %s, or remove the block.",
			label(sec), b.Tag, b.Line, where)
	}
	if open != nil {
		ch := fmt.Sprintf("%q", string(open.Run[0]))
		c.add(SeverityError, loc, e.Line, "content of section %s opens a code fence %q on line %d of the section that is never closed, so every later section would render as code and be lost to the parser. Close it with a matching fence line (the same character, %s, at least %d long, nothing else on the line) %s.",
			label(sec), open.Run, open.Line, ch, len(open.Run), where)
	}
}

// checkMarkdownOnly applies the rules that only make sense for a markdown
// body: on the YAML path unknown keys are loader errors and rendering imposes
// schema order and numbering.
func (c *checker) checkMarkdownOnly() {
	if c.m.Source != SourceMarkdown {
		return
	}
	for _, h := range c.m.UnknownHeadings {
		c.add(SeverityWarning, "draft", h.Line, "heading \"## %s\" is not a section of schema %s. Rename it to one of the schema's headings (%s), or remove it.", h.Text, c.sc.Type, c.expectedHeadings())
	}
	known := map[string]bool{}
	for _, key := range c.sc.Metadata {
		known[key] = true
	}
	var extra []string
	for key := range c.m.Metadata {
		if !known[key] {
			extra = append(extra, key)
		}
	}
	sort.Strings(extra)
	for _, key := range extra {
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
