package names_test

import (
	"testing"

	"github.com/threehillpath/claude-plan-workflow/tool/internal/names"
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

func TestImplBranch(t *testing.T) {
	tests := []struct {
		issue  int
		suffix string
		want   string
	}{
		{42, "", "feature/plan-00042"},
		{42, "a", "feature/plan-00042-a"},  // suffix passed as lowercase
		{42, "A", "feature/plan-00042-a"},  // suffix should be lowercased in branch
		{42, "B", "feature/plan-00042-b"},
	}
	for _, tc := range tests {
		if got := names.ImplBranch(tc.issue, tc.suffix); got != tc.want {
			t.Errorf("ImplBranch(%d, %q) = %q, want %q", tc.issue, tc.suffix, got, tc.want)
		}
	}
}

func TestPhaseBranch(t *testing.T) {
	tests := []struct {
		issue  int
		suffix string
		phase  int
		want   string
	}{
		{42, "", 3, "feature/plan-00042-3"},
		{42, "a", 2, "feature/plan-00042-a-2"},
		{42, "A", 2, "feature/plan-00042-a-2"}, // suffix lowercased
	}
	for _, tc := range tests {
		if got := names.PhaseBranch(tc.issue, tc.suffix, tc.phase); got != tc.want {
			t.Errorf("PhaseBranch(%d, %q, %d) = %q, want %q", tc.issue, tc.suffix, tc.phase, got, tc.want)
		}
	}
}

func TestWorktreePath(t *testing.T) {
	tests := []struct {
		issue  int
		suffix string
		phase  int
		want   string
	}{
		{42, "", 3, ".claude/worktrees/phase-00042-3"},
		{42, "a", 2, ".claude/worktrees/phase-00042-a-2"},
	}
	for _, tc := range tests {
		if got := names.WorktreePath(tc.issue, tc.suffix, tc.phase); got != tc.want {
			t.Errorf("WorktreePath(%d, %q, %d) = %q, want %q", tc.issue, tc.suffix, tc.phase, got, tc.want)
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
	if got, want := names.PhaseBranch(issue, "", phase), "feature/plan-00042-3"; got != want {
		t.Errorf("PhaseBranch: got %q want %q", got, want)
	}
	if got, want := names.WorktreePath(issue, "", phase), ".claude/worktrees/phase-00042-3"; got != want {
		t.Errorf("WorktreePath: got %q want %q", got, want)
	}
	if got, want := names.TitlePrefix(names.Phase, issue, "", phase), "[PLAN-00042-3]"; got != want {
		t.Errorf("TitlePrefix(Phase): got %q want %q", got, want)
	}
}

// TestDeriveSuffix verifies the -A/-B case-flip: branch lowercased, title prefix uppercased.
func TestDeriveSuffix(t *testing.T) {
	issue, suffix, phase := 42, "a", 2

	if got, want := names.ImplBranch(issue, suffix), "feature/plan-00042-a"; got != want {
		t.Errorf("ImplBranch: got %q want %q", got, want)
	}
	if got, want := names.PhaseBranch(issue, suffix, phase), "feature/plan-00042-a-2"; got != want {
		t.Errorf("PhaseBranch: got %q want %q", got, want)
	}
	if got, want := names.TitlePrefix(names.Phase, issue, suffix, phase), "[PLAN-00042-A-2]"; got != want {
		t.Errorf("TitlePrefix(Phase): got %q want %q", got, want)
	}
}
