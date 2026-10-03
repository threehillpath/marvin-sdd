// Package parse extracts plan identifiers and structured data from issue titles
// and comment bodies. All functions are tolerant of surrounding text; a missing
// match is returned as (zero-value, false), not an error.
package parse

import (
	"regexp"
	"strconv"
	"strings"
	"threehillpath.com/marvin-sdd/tool/internal/names"
)

// Kind distinguishes an arch plan from an impl plan or phase.
type Kind int

const (
	// KindImpl covers both impl plans and phases (phases are further
	// distinguished by Phase != 0).
	KindImpl Kind = iota
	// KindArch marks an arch plan (a "-ARCH" suffixed ident).
	KindArch
)

// Ident holds the parsed components of a plan bracket token like [PLAN-00042-A-3].
// Phase == 0 means no phase (arch plan or impl plan).
// Suffix == "" means no multi-impl suffix.
type Ident struct {
	Plan   int    // zero-padded issue number
	Suffix string // e.g. "A" or "B"; empty if none
	Phase  int    // phase ordinal; 0 if none
	Kind   Kind   // KindArch for "-ARCH" idents, KindImpl otherwise
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
//	[PLAN-00042]        → {42, "", 0, KindImpl}
//	[PLAN-00042-ARCH]   → {42, "", 0, KindArch}  (ARCH sets Kind, not phase/suffix)
//	[PLAN-00042-3]      → {42, "", 3, KindImpl}
//	[PLAN-00042-A]      → {42, "A", 0, KindImpl}
//	[PLAN-00042-A-2]    → {42, "A", 2, KindImpl}
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
		return Ident{Plan: planNum, Kind: KindArch}, true
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

// bracketPrefixRe matches a leading "[...]" bracket token (and any following
// whitespace) at the start of a title, e.g. the "[PLAN-00042-1] " in
// "[PLAN-00042-1] Person & Role Rendering Logic".
var bracketPrefixRe = regexp.MustCompile(`^\s*\[[^\]]*\]\s*`)

// nonAlnumRunRe matches one or more consecutive non-alphanumeric characters,
// collapsed to a single hyphen by Slugify.
var nonAlnumRunRe = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify converts arbitrary text into a lowercase, hyphen-separated,
// filesystem-safe slug: every run of non-alphanumeric characters becomes a
// single hyphen, and leading/trailing hyphens are trimmed.
// Example: "Person & Role Rendering Logic" → "person-role-rendering-logic"
func Slugify(s string) string {
	s = strings.ToLower(s)
	s = nonAlnumRunRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// TitleSlug strips a leading "[...]" bracket ident (if present) from title —
// e.g. "[PLAN-00042-1] " — and slugifies whatever remains. Used to name a
// durable per-phase doc file (docs/stories/<plan>/phase-NN-<slug>.md) so the
// same title always produces the same filename regardless of which skill
// invocation computes it.
func TitleSlug(title string) string {
	rest := bracketPrefixRe.ReplaceAllString(title, "")
	return Slugify(rest)
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

// leadingBracketRe captures the bracket token at the start of a title
// (optional whitespace, then "["), the same anchoring TitleSlug uses.
var leadingBracketRe = regexp.MustCompile(`^\s*(\[[^\]]*\])`)

// taskIdentRe matches a whole [TASK-XXXXX] bracket token.
var taskIdentRe = regexp.MustCompile(`^\[TASK-(\d{5})\]$`)

// planTokenRe matches exactly the accepted whole-token plan forms:
// [PLAN-XXXXX], [PLAN-XXXXX-ARCH], [PLAN-XXXXX-<letters>], [PLAN-XXXXX-N], and
// [PLAN-XXXXX-<letters>-N] (N a positive integer without a leading zero).
var planTokenRe = regexp.MustCompile(`^\[PLAN-\d{5}(-[A-Za-z]+)?(-[1-9]\d*)?\]$`)

// Classify classifies a title by its leading bracket token only; a later
// bracket never affects the result. Plan tokens are judged by PlanIdent so the
// two never disagree. The bool is the only "not found" signal: names.Kind's
// zero value is Arch.
func Classify(title string) (names.Kind, bool) {
	m := leadingBracketRe.FindStringSubmatch(title)
	if m == nil {
		return names.Arch, false
	}
	tok := m[1]
	if taskIdentRe.MatchString(tok) {
		return names.Task, true
	}
	if !strings.HasPrefix(tok, "[PLAN-") {
		return names.Arch, false
	}
	if !planTokenRe.MatchString(tok) {
		return names.Arch, false
	}
	ident, ok := PlanIdent(tok)
	if !ok {
		return names.Arch, false
	}
	switch {
	case ident.Phase != 0:
		return names.Phase, true
	case ident.Kind == KindArch:
		return names.Arch, true
	}
	return names.Impl, true
}

// TaskIdent returns the task number from a leading [TASK-XXXXX] bracket.
func TaskIdent(title string) (int, bool) {
	m := leadingBracketRe.FindStringSubmatch(title)
	if m == nil {
		return 0, false
	}
	tm := taskIdentRe.FindStringSubmatch(m[1])
	if tm == nil {
		return 0, false
	}
	n, err := strconv.Atoi(tm[1])
	if err != nil {
		return 0, false
	}
	return n, true
}
