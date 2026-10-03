package template

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

// lineOrigin says where one line of the rendered body came from, so a problem
// the parser finds in the body can be reported against the draft.
type lineOrigin struct {
	loc     string // "metadata:<Key>" or "section:<id>"
	what    string // `metadata value "Key"` or `section "Scope"`
	content int    // line within the entry's content, 0 for a metadata or heading line
	line    int    // line of the entry in the input, 0 when unknown
	where   string // path-specific place to edit, e.g. `inside the "scope" block`
}

// emittedHeading is a "## " heading Render wrote, at a rendered line.
type emittedHeading struct {
	line int // one-based line of the rendered body
	text string
}

var inlineBannedRe = regexp.MustCompile(`(?i)^(?:<!--|</?(?:details|pre|script|style|textarea)(?:[\s/>]|$))`)

// verifyBody parses the rendered body the way GitHub does (CommonMark with
// the GFM extensions) and reports anything that changes the document's
// structure: a level 1 or 2 heading that is not one Render emitted, a heading
// swallowed by an earlier block, any HTML block, and inline HTML that opens a
// comment or a collapsible or raw block. It is the guarantee behind the
// scanner checks in Check, which give the specific messages.
func verifyBody(body string, origins []lineOrigin, heads []emittedHeading) []Finding {
	src := []byte(body)
	doc := goldmark.New(goldmark.WithExtensions(extension.GFM)).Parser().Parse(text.NewReader(src))

	starts := []int{0}
	for i, b := range src {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}
	lineOf := func(off int) int { return sort.Search(len(starts), func(i int) bool { return starts[i] > off }) }
	lineText := func(line int) string {
		if line < 1 || line > len(starts) {
			return ""
		}
		end := len(src)
		if line < len(starts) {
			end = starts[line] - 1
		}
		return strings.TrimSpace(string(src[starts[line-1]:end]))
	}
	originOf := func(line int) lineOrigin {
		if line < 1 || line > len(origins) {
			return lineOrigin{loc: "draft", what: "the rendered body"}
		}
		return origins[line-1]
	}
	var out []Finding
	// report adds an error located at the origin of rendered line; the
	// message is the origin ("section "Scope" on line N of the section"), then
	// the formatted tail.
	report := func(line int, format string, args ...any) {
		o := originOf(line)
		pos := ""
		if o.content > 0 {
			pos = fmt.Sprintf(" on line %d of the section", o.content)
		}
		out = append(out, Finding{Severity: SeverityError, Location: o.loc, Line: o.line, Message: o.what + pos + fmt.Sprintf(format, args...)})
	}
	quote := func(line int) string {
		t := lineText(line)
		if len(t) > 60 {
			t = t[:60] + "..."
		}
		return fmt.Sprintf("%q", t)
	}
	where := func(line int) string { return originOf(line).where }

	var blockStart func(n ast.Node) int
	blockStart = func(n ast.Node) int {
		if f, ok := n.(*ast.FencedCodeBlock); ok {
			if f.Info != nil {
				return lineOf(f.Info.Segment.Start)
			}
			if f.Lines().Len() > 0 {
				return lineOf(f.Lines().At(0).Start) - 1
			}
			return 0
		}
		if n.Type() == ast.TypeBlock && n.Lines().Len() > 0 {
			return lineOf(n.Lines().At(0).Start)
		}
		best := 0
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if l := blockStart(c); l > 0 && (best == 0 || l < best) {
				best = l
			}
		}
		return best
	}

	seen := map[int]bool{}
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.Heading:
			if v.Level > 2 {
				break
			}
			line := 0
			if v.Lines().Len() > 0 {
				line = lineOf(v.Lines().At(0).Start)
			} else if p := v.Pos(); p >= 0 {
				// An empty heading ("#", "- #") has no text line.
				line = lineOf(p)
			}
			topLevel := v.Parent() != nil && v.Parent().Kind() == ast.KindDocument
			atx := strings.HasPrefix(lineText(line), "#")
			isEmitted := false
			for _, h := range heads {
				if h.line == line {
					isEmitted = true
				}
			}
			if isEmitted && topLevel && atx && v.Level == 2 {
				seen[line] = true
				break
			}
			if isEmitted {
				seen[line] = true
			}
			report(line, " becomes a level-%d heading %s in the rendered body, which would add or reshape a section. Section content must never change the document's structure: write sub-headings as \"### ...\", start them at the beginning of a line with a blank line before them, and use no \"#\" or \"##\" headings. If the # is literal text, escape it as \\#. Edit it %s.", v.Level, quote(line), where(line))
		case *ast.HTMLBlock:
			line := lineOf(v.Lines().At(0).Start)
			report(line, " contains an HTML block starting with %s. Raw HTML blocks (a tag alone on a line, <div>, <?php, <![CDATA[, <!DOCTYPE ...) can hide or swallow the sections after them when rendered, so drafts don't allow them. Wrap the HTML in backticks as inline code or remove it, %s.", quote(line), where(line))
		case *ast.RawHTML:
			var sb strings.Builder
			for i := 0; i < v.Segments.Len(); i++ {
				seg := v.Segments.At(i)
				sb.Write(seg.Value(src))
			}
			raw := sb.String()
			if !inlineBannedRe.MatchString(raw) || v.Segments.Len() == 0 {
				break
			}
			line := lineOf(v.Segments.At(0).Start)
			report(line, " contains raw HTML %q. Raw HTML could hide or swallow the sections after it when rendered, so drafts don't allow it. Wrap it in backticks as inline code (for example `<details>`) or remove it, %s.", strings.TrimRight(raw, " \t/"), where(line))
		}
		return ast.WalkContinue, nil
	})

	// A heading Render emitted that the parser does not see was swallowed by
	// an earlier block. Report the top-level block that contains it once.
	type top struct {
		node  ast.Node
		start int
	}
	var tops []top
	for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
		if l := blockStart(c); l > 0 {
			tops = append(tops, top{c, l})
		}
	}
	reported := map[ast.Node]bool{}
	for _, h := range heads {
		if seen[h.line] {
			continue
		}
		var owner *top
		for i := range tops {
			if tops[i].start <= h.line {
				owner = &tops[i]
			}
		}
		if owner == nil || reported[owner.node] {
			continue
		}
		reported[owner.node] = true
		if owner.node.Kind() == ast.KindHTMLBlock {
			continue // reported as an HTML block above
		}
		kind := strings.ToLower(owner.node.Kind().String())
		switch owner.node.(type) {
		case *ast.FencedCodeBlock:
			kind = "fenced code block"
		case *ast.CodeBlock:
			kind = "indented code block"
		}
		report(owner.start, " starts a %s that swallows the heading %q and everything after it in the rendered body, so those sections would be lost. Close the code fence with a matching line, and if it is inside a list item indent every line of it, the code and the closing fence, at least as far as the opening fence; otherwise end or remove the %s, %s.", kind, "## "+h.text, kind, where(owner.start))
	}
	return out
}
