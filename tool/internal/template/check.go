package template

import (
	"fmt"
	"regexp"
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

func (c *checker) checkSections() {
	for _, sec := range c.sc.Sections {
		loc := "section:" + sec.ID
		entries := c.m.Sections[sec.ID]
		if len(entries) == 0 {
			if sec.Required {
				c.add(SeverityError, loc, 0, "required section %q is missing. %s", sec.Heading,
					c.fix(fmt.Sprintf("Add a %q block under \"sections:\" in the draft.", sec.ID+": |"),
						fmt.Sprintf("Add a \"## %s\" heading with content.", sec.Heading)))
			}
			continue
		}
		if !sec.Repeatable && len(entries) > 1 {
			c.add(SeverityError, loc, entries[1].Line, "section %q is not repeatable but has %d entries. Merge them into one.", sec.Heading, len(entries))
		}
		for _, e := range entries {
			if strings.TrimSpace(e.Content) == "" {
				if sec.Required {
					c.add(SeverityError, loc, e.Line, "required section %q is empty. %s", sec.Heading,
						c.fix(fmt.Sprintf("Fill the %q block in the draft with content.", sec.ID),
							fmt.Sprintf("Fill the \"## %s\" section with content.", sec.Heading)))
				}
			}
			if sec.Numbered && sec.Named != nil && *sec.Named && strings.TrimSpace(e.Name) == "" {
				c.add(SeverityError, loc, e.Line, "an entry of numbered section %q has an empty name. The schema expects each entry to be named. %s", sec.Heading,
					c.fix(fmt.Sprintf("Give every %q entry a non-empty name in the draft.", sec.ID),
						"Give the heading text after the number, like \"## 1. <Name>\"."))
			}
			if h, ok := fencedH2(e.Content); ok {
				c.add(SeverityError, loc, e.Line, "content of section %q contains the line %q, which would become a new top-level section when rendered. The schema expects sub-headings below \"## \". Use \"###\" instead.", sec.Heading, h)
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

// Format renders the result as plain text.
func (r Result) Format() string {
	return ""
}
