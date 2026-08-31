// Package names derives canonical branch names, worktree paths, and title prefixes
// from a GitHub issue number per the rules in skills/SHARED/GLOSSARY.md.
// All functions are pure — no I/O, no external dependencies.
package names

import (
	"fmt"
	"strings"
)

// Kind distinguishes the four work-item kinds.
type Kind int

const (
	Arch  Kind = iota // architecture plan
	Impl              // implementation plan
	Phase             // phase issue
	Task              // single-cycle task/bug (source-issue-keyed, no plan hierarchy)
)

// PlanNumber formats a GitHub issue number as a zero-padded 5-digit plan number.
// Example: 42 → "PLAN-00042"
func PlanNumber(issue int) string {
	return fmt.Sprintf("PLAN-%05d", issue)
}

// TaskNumber formats a GitHub issue number as a zero-padded 5-digit task
// number, keyed on the source issue number (the same convention PlanNumber
// uses for Arch/Impl/Phase — composable before any tracking issue exists).
// Example: 91 → "TASK-00091"
func TaskNumber(issue int) string {
	return fmt.Sprintf("TASK-%05d", issue)
}

// TaskBranch returns "<type>/TASK-XXXXX" — uppercase, unlike TaskWorktreePath.
// typ is "feature" or "bug"; an empty typ defaults to "feature" (via ResolveType).
// Example: "bug", 91 → "bug/TASK-00091"
func TaskBranch(typ string, issue int) string {
	return fmt.Sprintf("%s/%s", ResolveType(typ), TaskNumber(issue))
}

// TaskWorktreePath returns "<worktree_base>/task-XXXXX" — lowercase, unlike
// the branch and title-prefix forms.
// Example: ".worktrees", 91 → ".worktrees/task-00091"
func TaskWorktreePath(base string, issue int) string {
	return fmt.Sprintf("%s/task-%05d", base, issue)
}

// PlanID returns the lowercase, path-safe plan identifier used in branch names,
// worktree paths, and the findings cache directory.
// Example: 42 → "plan-00042"
func PlanID(issue int) string {
	return fmt.Sprintf("plan-%05d", issue)
}

// DefaultType is the branch-type value applied when a caller passes an empty
// type string. This is the sole place the empty-defaults-to-feature rule is
// implemented; validating that a non-empty type is exactly "feature" or "bug"
// is the CLI layer's responsibility (tool/internal/cli), not this package's.
const DefaultType = "feature"

// ResolveType returns typ, defaulting to DefaultType when typ is empty.
func ResolveType(typ string) string {
	if typ == "" {
		return DefaultType
	}
	return typ
}

// TrunkBranch returns the trunk branch name for an issue: "<type>/PLAN-XXXXX/main".
// typ is "feature" or "bug"; an empty typ defaults to "feature" (via ResolveType).
// Suffix (if non-empty) is lowercased, matching the GLOSSARY.md convention.
// Example: "feature", 42, "" → "feature/PLAN-00042/main"
// Example: "bug", 42, "" → "bug/PLAN-00042/main"
// Example: "feature", 42, "a" → "feature/PLAN-00042/main-a"
func TrunkBranch(typ string, issue int, suffix string) string {
	base := fmt.Sprintf("%s/%s/main", ResolveType(typ), PlanNumber(issue))
	if suffix == "" {
		return base
	}
	return base + "-" + strings.ToLower(suffix)
}

// PhaseBranch returns the phase branch name: "<type>/PLAN-XXXXX/phase-N".
// typ is "feature" or "bug"; an empty typ defaults to "feature" (via ResolveType).
// Suffix is lowercased; phase is a positive integer.
// Example: "feature", 42, "", 3  → "feature/PLAN-00042/phase-3"
// Example: "bug", 143, "a", 2 → "bug/PLAN-00143/phase-a-2"
func PhaseBranch(typ string, issue int, suffix string, phase int) string {
	planNum := PlanNumber(issue)
	if suffix == "" {
		return fmt.Sprintf("%s/%s/phase-%d", ResolveType(typ), planNum, phase)
	}
	return fmt.Sprintf("%s/%s/phase-%s-%d", ResolveType(typ), planNum, strings.ToLower(suffix), phase)
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
	case Task:
		return fmt.Sprintf("[TASK-%s]", num)
	default:
		return fmt.Sprintf("[PLAN-%s]", num)
	}
}
