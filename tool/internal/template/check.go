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
	case isNamed(sec) && name != "":
		y = fmt.Sprintf("Fill \"content: |\" of the entry named %q in %q with content", name, sec.ID)
		num := "<n>"
		if e.Number > 0 {
			num = strconv.Itoa(e.Number)
		}
		md = fmt.Sprintf("Fill the \"## %s. %s\" section with content", num, name)
	case sec.Repeatable:
		y = fmt.Sprintf("Fill the empty \"- |\" block in %q with content", sec.ID)
		heading := "## " + sec.Heading
		if sec.Numbered {
			heading = "## <n>. " + sec.Heading
			if e.Number > 0 {
				heading = fmt.Sprintf("## %d. %s", e.Number, sec.Heading)
			}
		}
		md = fmt.Sprintf("Fill the %q section with content", heading)
	default:
		y = fmt.Sprintf("Fill the %q block in the draft with content", sec.ID)
		md = fmt.Sprintf("Fill the \"## %s\" section with content", sec.Heading)
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
			if h, ok := fencedH2(e.Content); ok {
				c.add(SeverityError, loc, e.Line, "content of section %s contains the line %q, which would become a new top-level section when rendered. The schema expects sub-headings below \"## \". Use \"###\" instead.", label(sec), h)
			}
		}
	}
}

// fencedH2 returns the first "## " line outside a fenced code block.
func fencedH2(content string) (string, bool) {
	fence := ""
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimLeft(line, " ")
		if len(line)-len(t) <= 3 {
			for _, f := range []string{"```", "~~~"} {
				if strings.HasPrefix(t, f) {
					switch {
					case fence == "":
						fence = f
					case fence == f:
						fence = ""
					}
					break
				}
			}
		}
		if fence == "" && strings.HasPrefix(line, "## ") {
			return strings.TrimRight(line, " \t\r"), true
		}
	}
	return "", false
}

// checkMarkdownOnly applies the rules that only make sense for a markdown
// body: on the YAML path unknown keys are loader errors and rendering imposes
// schema order and numbering.
func (c *checker) checkMarkdownOnly() {
	if c.m.Source != SourceMarkdown {
		return
	}
	for _, h := range c.m.UnknownHeadings {
		c.add(SeverityWarning, "draft", h.Line, "heading \"## %s\" is not a section of schema %s. Rename it to one of the schema's headings, or remove it.", h.Text, c.sc.Type)
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

// checkOrder warns about sections outside the schema's order. Sections that
// are not part of the longest in-order run (by first-entry line) are the ones
// reported, so one displaced section yields one warning.
func (c *checker) checkOrder() {
	type placed struct {
		sec  SchemaSection
		line int
	}
	var ps []placed
	for _, sec := range c.sc.Sections {
		if es := c.m.Sections[sec.ID]; len(es) > 0 && es[0].Line > 0 {
			ps = append(ps, placed{sec, es[0].Line})
		}
	}
	n := len(ps)
	length := make([]int, n)
	prev := make([]int, n)
	best := -1
	for i := range ps {
		length[i], prev[i] = 1, -1
		for j := 0; j < i; j++ {
			if ps[j].line < ps[i].line && length[j]+1 > length[i] {
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
	for i, p := range ps {
		if !inRun[i] {
			c.add(SeverityWarning, "section:"+p.sec.ID, p.line, "section %q is out of schema order. Move \"## %s\" to its place in the schema's section order.", p.sec.Heading, p.sec.Heading)
		}
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
		heading := it.sec.Heading
		if it.e.Name != "" {
			heading = it.e.Name
		}
		c.add(SeverityWarning, "section:"+it.sec.ID, it.e.Line, "numbered heading \"## %d. %s\" breaks the sequence. The schema expects consecutive numbers from 1. Renumber it to \"## %d.\".", it.e.Number, heading, i+1)
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
