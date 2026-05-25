package parse_test

import (
	"testing"

	"github.com/threehillpath/claude-plan-workflow/tool/internal/parse"
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
}

func TestPlanIdentNoMatch(t *testing.T) {
	_, ok := parse.PlanIdent("some random title without bracket token")
	if ok {
		t.Error("expected no match, got true")
	}
}

func TestImplPlanNumberFromPhaseBody(t *testing.T) {
	body := `**Implementation Plan:** #15 ([PLAN-00002])
**Plan Number:** PLAN-00002
**Status:** Upcoming`

	n, ok := parse.ImplPlanNumberFromPhaseBody(body)
	if !ok {
		t.Fatal("expected match, got false")
	}
	if n != 15 {
		t.Errorf("got %d, want 15", n)
	}
}

func TestImplPlanNumberNotFound(t *testing.T) {
	_, ok := parse.ImplPlanNumberFromPhaseBody("no match here")
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
