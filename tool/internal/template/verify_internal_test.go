package template

import (
	"strings"
	"testing"
)

// TestVerifyBodyBansInlineHTMLOpeners calls the goldmark backstop directly on
// a rendered body whose metadata lines are joined into one processing
// instruction, CDATA section or declaration. Check normally stops these
// first; verifyBody must still refuse them, because it is the guarantee that
// the parser and GitHub see the same structure.
func TestVerifyBodyBansInlineHTMLOpeners(t *testing.T) {
	for _, c := range []struct{ opener, closer string }{{"<?x", "?>"}, {"<![CDATA[x", "]]>"}, {"<!DOCTYPE x", ">"}} {
		opener := c.opener
		body := "**Status:** " + opener + "\n**Plan Number:** end " + c.closer + "\n"
		origins := []lineOrigin{
			{loc: "metadata:Status", what: `metadata value "Status"`, line: 1, where: `in the "**Status:**" line`},
			{loc: "metadata:Plan Number", what: `metadata value "Plan Number"`, line: 2},
			{loc: "draft", what: "the rendered body"},
		}
		fs := verifyBody(body, origins, nil)
		if len(fs) != 1 || fs[0].Severity != SeverityError || fs[0].Location != "metadata:Status" || !strings.Contains(fs[0].Message, "raw HTML") {
			t.Errorf("opener %q: want one raw HTML error at metadata:Status, got %+v", opener, fs)
		}
	}
}

// TestVerifyBodyBansHTMLBlocksStartingWithOpeners keeps the goldmark
// HTML-block backstop guarded for the line-start forms of the same openers
// (Check now reports them first as raw HTML).
func TestVerifyBodyBansHTMLBlocksStartingWithOpeners(t *testing.T) {
	for _, opener := range []string{"<?php x", "<![CDATA[ x", "<!DOCTYPE html"} {
		body := "## Scope\n\n" + opener + "\nmore\n"
		origins := make([]lineOrigin, 5)
		for i := range origins {
			origins[i] = lineOrigin{loc: "section:scope", what: `section "Scope"`, line: 9, content: max(i-1, 0)}
		}
		fs := verifyBody(body, origins, []emittedHeading{{line: 1, text: "Scope"}})
		found := false
		for _, f := range fs {
			if f.Severity == SeverityError && strings.Contains(f.Message, "HTML block") && strings.Contains(f.Message, opener) {
				found = true
			}
		}
		if !found {
			t.Errorf("opener %q: want an HTML block error, got %+v", opener, fs)
		}
	}
}
