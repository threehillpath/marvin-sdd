package template_test

import (
	"strings"
	"testing"

	tmpl "threehillpath.com/marvin-sdd/tool/internal/template"
)

const phaseBody = `**Implementation Plan:** #132 ([PLAN-00112])
**Plan Number:** PLAN-00112
**Status:** Upcoming

## Objective

Do it.

## Scope

Includes things.

## Components

Stuff.

## Verification

go test ./...

## Success Criteria

- [ ] Done
`

// phaseBodyWithout returns phaseBody with the "## <heading>" section removed.
func phaseBodyWithout(heading string) string {
	parts := strings.Split(phaseBody, "\n## ")
	var keep []string
	for _, p := range parts {
		if !strings.HasPrefix(p, heading+"\n") {
			keep = append(keep, p)
		}
	}
	return strings.Join(keep, "\n## ")
}

func parseCheck(t *testing.T, name, title, body string) tmpl.Result {
	t.Helper()
	sc := loadBuiltIn(t, name)
	return tmpl.Check(sc, builtIn, tmpl.ParseMarkdown(sc, title, body))
}

// The markdown path gives the same verdict as Phase 2's YAML entry test: one
// error at section:verification.
func TestParseMarkdownMissingVerification(t *testing.T) {
	res := parseCheck(t, "impl-phase", "[PLAN-00112-1] X", phaseBodyWithout("Verification"))
	wantOne(t, res, tmpl.SeverityError, "section:verification", `"Verification"`, `Add a "## Verification" heading`)
	if got := parseCheck(t, "impl-phase", "[PLAN-00112-1] X", phaseBody); len(got.Findings) != 0 {
		t.Fatalf("complete body should have no findings:\n%s", got.Format())
	}
}
