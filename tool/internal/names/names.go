// Package names derives canonical branch names, worktree paths, and title prefixes
// from a GitHub issue number per the rules in skills/SHARED/GLOSSARY.md.
// All functions are pure — no I/O, no external dependencies.
package names

import (
	"fmt"
	"strings"
)

// Kind distinguishes the three plan issue types.
type Kind int

const (
	Arch  Kind = iota // architecture plan
	Impl              // implementation plan
	Phase             // phase issue
)

// PlanNumber formats a GitHub issue number as a zero-padded 5-digit plan number.
// Example: 42 → "PLAN-00042"
func PlanNumber(issue int) string {
	return fmt.Sprintf("PLAN-%05d", issue)
}

// PlanID returns the lowercase, path-safe plan identifier used in branch names,
// worktree paths, and the findings cache directory.
// Example: 42 → "plan-00042"
func PlanID(issue int) string {
	return fmt.Sprintf("plan-%05d", issue)
}

// ImplBranch returns the implementation branch name for an issue.
// Suffix (if non-empty) is lowercased, matching the GLOSSARY.md convention.
// Example: 42, "" → "feature/plan-00042"
// Example: 42, "A" → "feature/plan-00042-a"
func ImplBranch(issue int, suffix string) string {
	base := fmt.Sprintf("feature/plan-%05d", issue)
	if suffix == "" {
		return base
	}
	return base + "-" + strings.ToLower(suffix)
}

// PhaseBranch returns the phase branch name.
// Suffix is lowercased; phase is a positive integer.
// Example: 42, "", 3  → "feature/plan-00042-3"
// Example: 42, "A", 2 → "feature/plan-00042-a-2"
func PhaseBranch(issue int, suffix string, phase int) string {
	if suffix == "" {
		return fmt.Sprintf("feature/plan-%05d-%d", issue, phase)
	}
	return fmt.Sprintf("feature/plan-%05d-%s-%d", issue, strings.ToLower(suffix), phase)
}

// WorktreePath returns the relative worktree path from the repo root.
// base is the configured worktree directory (e.g. from Config.WorktreeBase).
// Suffix is lowercased.
// Example: ".worktrees", 42, "", 3  → ".worktrees/phase-00042-3"
// Example: ".worktrees", 42, "a", 2 → ".worktrees/phase-00042-a-2"
func WorktreePath(base string, issue int, suffix string, phase int) string {
	if suffix == "" {
		return fmt.Sprintf("%s/phase-%05d-%d", base, issue, phase)
	}
	return fmt.Sprintf("%s/phase-%05d-%s-%d", base, issue, strings.ToLower(suffix), phase)
}

// TitlePrefix returns the bracket prefix for a plan issue title.
// Suffix (if non-empty) is uppercased in title prefixes per GLOSSARY.md case-flip rule.
// Examples:
//
//	Arch, 42, "", 0  → "[PLAN-00042-ARCH]"
//	Impl, 42, "", 0  → "[PLAN-00042]"
//	Impl, 42, "a", 0 → "[PLAN-00042-A]"
//	Phase, 42, "", 3 → "[PLAN-00042-3]"
//	Phase, 42, "a", 2 → "[PLAN-00042-A-2]"
func TitlePrefix(kind Kind, issue int, suffix string, phase int) string {
	num := fmt.Sprintf("%05d", issue)
	switch kind {
	case Arch:
		return fmt.Sprintf("[PLAN-%s-ARCH]", num)
	case Impl:
		if suffix == "" {
			return fmt.Sprintf("[PLAN-%s]", num)
		}
		return fmt.Sprintf("[PLAN-%s-%s]", num, strings.ToUpper(suffix))
	case Phase:
		if suffix == "" {
			return fmt.Sprintf("[PLAN-%s-%d]", num, phase)
		}
		return fmt.Sprintf("[PLAN-%s-%s-%d]", num, strings.ToUpper(suffix), phase)
	default:
		return fmt.Sprintf("[PLAN-%s]", num)
	}
}
