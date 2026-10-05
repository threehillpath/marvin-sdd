package parse_test

import (
	"testing"

	"threehillpath.com/marvin-sdd/tool/internal/names"

	"threehillpath.com/marvin-sdd/tool/internal/parse"
)

func TestPlanIdentNoSuffix(t *testing.T) {
	ident, ok := parse.PlanIdent("[PLAN-00042-3] Backend — Domain")
	if !ok {
		t.Fatal("expected match, got false")
	}
	if ident.Plan != 42 {
		t.Errorf("Plan = %d, want 42", ident.Plan)
	}
	if ident.Suffix != "" {
		t.Errorf("Suffix = %q, want empty", ident.Suffix)
	}
	if ident.Phase != 3 {
		t.Errorf("Phase = %d, want 3", ident.Phase)
	}
}

func TestPlanIdentWithSuffix(t *testing.T) {
	// From the spec: "[PLAN-00042-A-2] Backend — Domain"
	ident, ok := parse.PlanIdent("[PLAN-00042-A-2] Backend — Domain")
	if !ok {
		t.Fatal("expected match, got false")
	}
	if ident.Plan != 42 {
		t.Errorf("Plan = %d, want 42", ident.Plan)
	}
	if ident.Suffix != "A" {
		t.Errorf("Suffix = %q, want A", ident.Suffix)
	}
	if ident.Phase != 2 {
		t.Errorf("Phase = %d, want 2", ident.Phase)
	}
}

func TestPlanIdentImplPlan(t *testing.T) {
	// Impl plan title: [PLAN-00042] Title
	ident, ok := parse.PlanIdent("[PLAN-00042] Member Invitations")
	if !ok {
		t.Fatal("expected match, got false")
	}
	if ident.Plan != 42 {
		t.Errorf("Plan = %d, want 42", ident.Plan)
	}
	if ident.Suffix != "" {
		t.Errorf("Suffix = %q, want empty", ident.Suffix)
	}
	if ident.Phase != 0 {
		t.Errorf("Phase = %d, want 0", ident.Phase)
	}
	if ident.Kind != parse.KindImpl {
		t.Errorf("Kind = %v, want KindImpl", ident.Kind)
	}
}

func TestPlanIdentArch(t *testing.T) {
	// Arch plan: [PLAN-00042-ARCH] Title
	ident, ok := parse.PlanIdent("[PLAN-00042-ARCH] Some Feature")
	if !ok {
		t.Fatal("expected match, got false")
	}
	if ident.Plan != 42 {
		t.Errorf("Plan = %d, want 42", ident.Plan)
	}
	if ident.Suffix != "" {
		t.Errorf("Suffix = %q, want empty for ARCH", ident.Suffix)
	}
	if ident.Phase != 0 {
		t.Errorf("Phase = %d, want 0", ident.Phase)
	}
	if ident.Kind != parse.KindArch {
		t.Errorf("Kind = %v, want KindArch for ARCH", ident.Kind)
	}
}

func TestPlanIdentNoMatch(t *testing.T) {
	_, ok := parse.PlanIdent("some random title without bracket token")
	if ok {
		t.Error("expected no match, got true")
	}
}

func TestPhaseListFromComment(t *testing.T) {
	comment := "- #12 [PLAN-00042-1]\n- #13 [PLAN-00042-2]"
	nums, ok := parse.PhaseListFromComment(comment)
	if !ok {
		t.Fatal("expected match, got false")
	}
	if len(nums) != 2 {
		t.Fatalf("expected 2 numbers, got %d: %v", len(nums), nums)
	}
	if nums[0] != 12 {
		t.Errorf("nums[0] = %d, want 12", nums[0])
	}
	if nums[1] != 13 {
		t.Errorf("nums[1] = %d, want 13", nums[1])
	}
}

func TestPhaseListFromCommentNoMatch(t *testing.T) {
	_, ok := parse.PhaseListFromComment("nothing here")
	if ok {
		t.Error("expected no match, got true")
	}
}

// TestTitleSlugStripsBracketAndPunctuation is the TDD entry point for
// per-phase doc filenames (docs/stories/<plan>/phase-NN-<slug>.md): the
// ampersand and repeated whitespace must collapse to single hyphens, not
// leak through or double up.
func TestTitleSlugStripsBracketAndPunctuation(t *testing.T) {
	got := parse.TitleSlug("[PLAN-00042-1] Person & Role Rendering Logic")
	want := "person-role-rendering-logic"
	if got != want {
		t.Errorf("TitleSlug = %q, want %q", got, want)
	}
}

func TestTitleSlugNoBracketPrefix(t *testing.T) {
	got := parse.TitleSlug("no bracket token here")
	want := "no-bracket-token-here"
	if got != want {
		t.Errorf("TitleSlug = %q, want %q", got, want)
	}
}

func TestTitleSlugEmDash(t *testing.T) {
	// Em dash is not ASCII alphanumeric, so it collapses like any other
	// separator — this exercises a multi-byte rune passing through Slugify.
	got := parse.TitleSlug("[PLAN-00042-3] Backend — Domain")
	want := "backend-domain"
	if got != want {
		t.Errorf("TitleSlug = %q, want %q", got, want)
	}
}

func TestClassify(t *testing.T) {
	tests := []struct {
		title string
		kind  names.Kind
		found bool
		task  int // expected task number when kind == Task
	}{
		{"[TASK-00091] Fix X", names.Task, true, 91},
		{"  [TASK-00091] leading space", names.Task, true, 91},
		{"[PLAN-00112-ARCH] X", names.Arch, true, 0},
		{"[PLAN-00112] X", names.Impl, true, 0},
		{"[PLAN-00112-A] X", names.Impl, true, 0},
		{"[PLAN-00042-a] X", names.Impl, true, 0},
		{"[PLAN-00112-3] X", names.Phase, true, 0},
		{"[PLAN-00112-A-2] X", names.Phase, true, 0},
		{"[TASK-00140] Fix regression from [PLAN-00112-3]", names.Task, true, 140},
		{"Notes on [PLAN-00042-1]", names.Arch, false, 0},
		{"Fix X", names.Arch, false, 0},
		{"", names.Arch, false, 0},
		{"[PLAN-XXXXX-ARCH] X", names.Arch, false, 0},
		{"[TASK-XXXXX] X", names.Arch, false, 0},
		{"[OTHER-00001] X", names.Arch, false, 0},
		{"[PLAN-000421] X", names.Arch, false, 0},
		{"[PLAN-00042-] X", names.Arch, false, 0},
		{"[PLAN-00042-0] X", names.Arch, false, 0},
		{"[PLAN-00042-A1] X", names.Arch, false, 0},
		{"[PLAN-00042-1-2] X", names.Arch, false, 0},
	}
	for _, tc := range tests {
		kind, ok := parse.Classify(tc.title)
		if ok != tc.found || (ok && kind != tc.kind) {
			t.Errorf("Classify(%q) = (%v, %v), want (%v, %v)", tc.title, kind, ok, tc.kind, tc.found)
		}
		n, tok := parse.TaskIdent(tc.title)
		wantTask := tc.found && tc.kind == names.Task
		if tok != wantTask || n != tc.task {
			t.Errorf("TaskIdent(%q) = (%d, %v), want (%d, %v)", tc.title, n, tok, tc.task, wantTask)
		}
	}
}
