// Package parse extracts plan identifiers and structured data from issue titles
// and comment bodies. All functions are tolerant of surrounding text; a missing
// match is returned as (zero-value, false), not an error.
package parse

import (
	"regexp"
	"strconv"
	"strings"
)

// Ident holds the parsed components of a plan bracket token like [PLAN-00042-A-3].
// Phase == 0 means no phase (arch plan or impl plan).
// Suffix == "" means no multi-impl suffix.
type Ident struct {
	Plan   int    // zero-padded issue number
	Suffix string // e.g. "A" or "B"; empty if none
	Phase  int    // phase ordinal; 0 if none
}

// planIdentRe matches the canonical bracket token at the start of an issue title.
// Groups: (issue-number) (optional-rest: -ARCH | -SUFFIX-PHASE | -PHASE)
// We parse the rest manually for clarity.
var planIdentRe = regexp.MustCompile(`\[PLAN-(\d{5})([^\]]*)\]`)

// PlanIdent parses a [PLAN-XXXXX...] bracket token from title.
// Returns the Ident and true on success; zero-value Ident and false on no match.
//
// Supported forms:
//
//	[PLAN-00042]        → {42, "", 0}
//	[PLAN-00042-ARCH]   → {42, "", 0}  (ARCH treated as no phase/suffix)
//	[PLAN-00042-3]      → {42, "", 3}
//	[PLAN-00042-A]      → {42, "A", 0}
//	[PLAN-00042-A-2]    → {42, "A", 2}
func PlanIdent(title string) (Ident, bool) {
	m := planIdentRe.FindStringSubmatch(title)
	if m == nil {
		return Ident{}, false
	}

	planNum, err := strconv.Atoi(m[1])
	if err != nil {
		return Ident{}, false
	}

	rest := m[2] // everything after the 5-digit number, inside the brackets
	if rest == "" {
		return Ident{Plan: planNum}, true
	}
	// rest starts with "-"
	rest = strings.TrimPrefix(rest, "-")

	// Special keyword ARCH → no suffix, no phase
	if rest == "ARCH" {
		return Ident{Plan: planNum}, true
	}

	parts := strings.Split(rest, "-")
	switch len(parts) {
	case 1:
		// Either a phase number ("-3") or a suffix letter ("-A")
		if n, err := strconv.Atoi(parts[0]); err == nil {
			return Ident{Plan: planNum, Phase: n}, true
		}
		// Single letter suffix, no phase
		return Ident{Plan: planNum, Suffix: strings.ToUpper(parts[0])}, true
	case 2:
		// Suffix + phase: "-A-2"
		suffix := strings.ToUpper(parts[0])
		phase, err := strconv.Atoi(parts[1])
		if err != nil {
			return Ident{}, false
		}
		return Ident{Plan: planNum, Suffix: suffix, Phase: phase}, true
	}
	return Ident{}, false
}

// phaseListLineRe matches a line like "- #12 [PLAN-00042-1]" and captures the issue number.
var phaseListLineRe = regexp.MustCompile(`-\s+#(\d+)\s+\[PLAN-`)

// PhaseListFromComment extracts GitHub issue numbers from the "Phases created:" comment
// format used by phase-split. Returns ([]int, true) when at least one number is found.
// The returned slice contains GitHub issue numbers, not phase ordinals.
func PhaseListFromComment(comment string) ([]int, bool) {
	matches := phaseListLineRe.FindAllStringSubmatch(comment, -1)
	if len(matches) == 0 {
		return nil, false
	}
	nums := make([]int, 0, len(matches))
	for _, m := range matches {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		nums = append(nums, n)
	}
	if len(nums) == 0 {
		return nil, false
	}
	return nums, true
}
