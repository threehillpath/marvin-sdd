package names_test

import (
	"strings"
	"testing"

	"threehillpath.com/marvin-sdd/tool/internal/names"
)

func TestPlanNumber(t *testing.T) {
	tests := []struct {
		issue int
		want  string
	}{
		{42, "PLAN-00042"},
		{111, "PLAN-00111"},
		{1, "PLAN-00001"},
		{99999, "PLAN-99999"},
	}
	for _, tc := range tests {
		if got := names.PlanNumber(tc.issue); got != tc.want {
			t.Errorf("PlanNumber(%d) = %q, want %q", tc.issue, got, tc.want)
		}
	}
}

func TestPlanID(t *testing.T) {
	tests := []struct {
		issue int
		want  string
	}{
		{42, "plan-00042"},
		{111, "plan-00111"},
		{1, "plan-00001"},
		{99999, "plan-99999"},
	}
	for _, tc := range tests {
		if got := names.PlanID(tc.issue); got != tc.want {
			t.Errorf("PlanID(%d) = %q, want %q", tc.issue, got, tc.want)
		}
	}
}

// TestTrunkBranch is the phase's TDD entry point: the trunk constructor,
// given an explicit type, issue, and optional suffix, produces the new
// nested "<type>/PLAN-XXXXX/main[-suffix]" shape.
func TestTrunkBranch(t *testing.T) {
	tests := []struct {
		typ    string
		issue  int
		suffix string
		want   string
	}{
		{"feature", 42, "", "feature/PLAN-00042/main"},
		{"bug", 42, "", "bug/PLAN-00042/main"},
		{"", 42, "", "feature/PLAN-00042/main"},         // empty type defaults to feature
		{"feature", 42, "a", "feature/PLAN-00042/main-a"}, // suffix lowercased
		{"feature", 42, "A", "feature/PLAN-00042/main-a"}, // suffix lowercased regardless of input case
	}
	for _, tc := range tests {
		if got := names.TrunkBranch(tc.typ, tc.issue, tc.suffix); got != tc.want {
			t.Errorf("TrunkBranch(%q, %d, %q) = %q, want %q", tc.typ, tc.issue, tc.suffix, got, tc.want)
		}
	}
}

// TestTrunkBranchNoTrailingArtifact is the git-ref-validity edge case: an empty
// suffix must be treated as "no suffix", never producing a trailing "-" or "-/".
func TestTrunkBranchNoTrailingArtifact(t *testing.T) {
	got := names.TrunkBranch("feature", 42, "")
	want := "feature/PLAN-00042/main"
	if got != want {
		t.Errorf("TrunkBranch(feature, 42, \"\") = %q, want %q (no trailing artifact)", got, want)
	}
	if strings.HasSuffix(got, "-") || strings.Contains(got, "-/") || strings.HasSuffix(got, "/") {
		t.Errorf("TrunkBranch(feature, 42, \"\") = %q contains a trailing/consecutive artifact", got)
	}
}

func TestPhaseBranch(t *testing.T) {
	tests := []struct {
		typ    string
		issue  int
		suffix string
		phase  int
		want   string
	}{
		{"feature", 42, "", 3, "feature/PLAN-00042/phase-3"},
		{"bug", 143, "a", 2, "bug/PLAN-00143/phase-a-2"},
		{"feature", 42, "A", 2, "feature/PLAN-00042/phase-a-2"}, // suffix lowercased
		{"", 42, "", 3, "feature/PLAN-00042/phase-3"},           // empty type defaults to feature
	}
	for _, tc := range tests {
		if got := names.PhaseBranch(tc.typ, tc.issue, tc.suffix, tc.phase); got != tc.want {
			t.Errorf("PhaseBranch(%q, %d, %q, %d) = %q, want %q", tc.typ, tc.issue, tc.suffix, tc.phase, got, tc.want)
		}
	}
}

func TestWorktreePath(t *testing.T) {
	tests := []struct {
		base   string
		issue  int
		suffix string
		phase  int
		want   string
	}{
		{".worktrees", 42, "", 3, ".worktrees/phase-00042-3"},
		{".worktrees", 42, "a", 2, ".worktrees/phase-00042-a-2"},
		{"custom/path", 42, "", 3, "custom/path/phase-00042-3"},
	}
	for _, tc := range tests {
		if got := names.WorktreePath(tc.base, tc.issue, tc.suffix, tc.phase); got != tc.want {
			t.Errorf("WorktreePath(%q, %d, %q, %d) = %q, want %q", tc.base, tc.issue, tc.suffix, tc.phase, got, tc.want)
		}
	}
}

func TestTitlePrefix(t *testing.T) {
	tests := []struct {
		kind   names.Kind
		issue  int
		suffix string
		phase  int
		want   string
	}{
		{names.Arch, 42, "", 0, "[PLAN-00042-ARCH]"},
		{names.Impl, 42, "", 0, "[PLAN-00042]"},
		{names.Phase, 42, "", 3, "[PLAN-00042-3]"},
		// Suffix uppercased in title prefixes
		{names.Impl, 42, "a", 0, "[PLAN-00042-A]"},
		{names.Phase, 42, "a", 2, "[PLAN-00042-A-2]"},
		{names.Phase, 42, "A", 2, "[PLAN-00042-A-2]"},
	}
	for _, tc := range tests {
		if got := names.TitlePrefix(tc.kind, tc.issue, tc.suffix, tc.phase); got != tc.want {
			t.Errorf("TitlePrefix(%v, %d, %q, %d) = %q, want %q", tc.kind, tc.issue, tc.suffix, tc.phase, got, tc.want)
		}
	}
}

// TestDeriveNoSuffix verifies the GLOSSARY.md example: issue=42, phase=3.
func TestDeriveNoSuffix(t *testing.T) {
	issue, phase := 42, 3

	if got, want := names.PlanNumber(issue), "PLAN-00042"; got != want {
		t.Errorf("PlanNumber: got %q want %q", got, want)
	}
	if got, want := names.PhaseBranch("feature", issue, "", phase), "feature/PLAN-00042/phase-3"; got != want {
		t.Errorf("PhaseBranch: got %q want %q", got, want)
	}
	if got, want := names.WorktreePath(".worktrees", issue, "", phase), ".worktrees/phase-00042-3"; got != want {
		t.Errorf("WorktreePath: got %q want %q", got, want)
	}
	if got, want := names.TitlePrefix(names.Phase, issue, "", phase), "[PLAN-00042-3]"; got != want {
		t.Errorf("TitlePrefix(Phase): got %q want %q", got, want)
	}
}

// TestDeriveSuffix verifies the -A/-B case-flip: branch lowercased, title prefix uppercased.
func TestDeriveSuffix(t *testing.T) {
	issue, suffix, phase := 42, "a", 2

	if got, want := names.TrunkBranch("feature", issue, suffix), "feature/PLAN-00042/main-a"; got != want {
		t.Errorf("TrunkBranch: got %q want %q", got, want)
	}
	if got, want := names.PhaseBranch("feature", issue, suffix, phase), "feature/PLAN-00042/phase-a-2"; got != want {
		t.Errorf("PhaseBranch: got %q want %q", got, want)
	}
	if got, want := names.TitlePrefix(names.Phase, issue, suffix, phase), "[PLAN-00042-A-2]"; got != want {
		t.Errorf("TitlePrefix(Phase): got %q want %q", got, want)
	}
}
