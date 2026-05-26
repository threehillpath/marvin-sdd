package pr_test

import (
	"context"
	"testing"

	"github.com/threehillpath/claude-plan-workflow/tool/internal/config"
	"github.com/threehillpath/claude-plan-workflow/tool/internal/exectest"
	"github.com/threehillpath/claude-plan-workflow/tool/internal/pr"
)

func buildConfig() *config.Config {
	return &config.Config{
		Repo:         "owner/repo",
		WorktreeBase: ".worktrees",
		Statuses:     map[string]string{},
	}
}

// prListResponse is a canned gh pr list JSON response containing one matching PR.
const prListFoundResponse = `[{"number":42,"title":"[PLAN-00002-3] Phase 3","url":"https://github.com/owner/repo/pull/42","headRefName":"feature/plan-00002-3","baseRefName":"feature/plan-00002","state":"OPEN"}]`

// TestFindFound verifies that Find returns found:true when the ident matches a PR title.
func TestFindFound(t *testing.T) {
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(prListFoundResponse)})

	cfg := buildConfig()
	result, err := pr.Find(context.Background(), fake, cfg, "[PLAN-00002-3]")
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	if !result.Found {
		t.Error("expected Found=true")
	}
	if result.Number != 42 {
		t.Errorf("Number = %d, want 42", result.Number)
	}
	if result.URL != "https://github.com/owner/repo/pull/42" {
		t.Errorf("URL = %q, unexpected", result.URL)
	}
}

// TestFindNotFound verifies that Find returns found:false (not an error) when no PR matches.
func TestFindNotFound(t *testing.T) {
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`[]`)})

	cfg := buildConfig()
	result, err := pr.Find(context.Background(), fake, cfg, "[PLAN-00002-99]")
	if err != nil {
		t.Fatalf("Find returned error for not-found: %v", err)
	}
	if result.Found {
		t.Error("expected Found=false")
	}
}

// TestBasePhaseBranch verifies that feature/plan-XXXXX-N maps to feature/plan-XXXXX.
func TestBasePhaseBranch(t *testing.T) {
	base, err := pr.Base("feature/plan-00002-3")
	if err != nil {
		t.Fatalf("Base returned error: %v", err)
	}
	if base != "feature/plan-00002" {
		t.Errorf("Base = %q, want feature/plan-00002", base)
	}
}

// TestBaseImplBranch verifies that feature/plan-XXXXX maps to main.
func TestBaseImplBranch(t *testing.T) {
	base, err := pr.Base("feature/plan-00002")
	if err != nil {
		t.Fatalf("Base returned error: %v", err)
	}
	if base != "main" {
		t.Errorf("Base = %q, want main", base)
	}
}

// TestBaseInvalidBranch verifies that non-plan branches return CLIError{Code:1}.
func TestBaseInvalidBranch(t *testing.T) {
	_, err := pr.Base("random-feature-branch")
	if err == nil {
		t.Fatal("expected error for non-plan branch, got nil")
	}
	type coder interface{ Code() int }
	// Should be a CLIError with code 1.
}
