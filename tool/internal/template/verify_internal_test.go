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
	for _, opener := range []string{"<?x", "<![CDATA[x", "<!DOCTYPE x"} {
		body := "**Status:** " + opener + "\n**Plan Number:** end ?>\n"
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
