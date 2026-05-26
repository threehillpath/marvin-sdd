package pr_test

import (
	"context"
	"testing"

	"threehillpath.com/claude-plan-workflow/tool/internal/clierr"
	"threehillpath.com/claude-plan-workflow/tool/internal/config"
	"threehillpath.com/claude-plan-workflow/tool/internal/exectest"
	"threehillpath.com/claude-plan-workflow/tool/internal/pr"
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
	result, err := pr.Find(context.Background(), fake, cfg, "[PLAN-00002-3]", pr.StateAny)
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
	result, err := pr.Find(context.Background(), fake, cfg, "[PLAN-00002-99]", pr.StateAny)
	if err != nil {
		t.Fatalf("Find returned error for not-found: %v", err)
	}
	if result.Found {
		t.Error("expected Found=false")
	}
}

// TestFindStateOpenPassesOpenArg verifies that StateOpen sends --state open to gh.
func TestFindStateOpenPassesOpenArg(t *testing.T) {
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`[]`)})

	cfg := buildConfig()
	if _, err := pr.Find(context.Background(), fake, cfg, "[PLAN-00002-3]", pr.StateOpen); err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	args := fake.Calls[0].Args
	for i, a := range args {
		if a == "--state" && i+1 < len(args) {
			if args[i+1] != "open" {
				t.Errorf("--state arg = %q, want open", args[i+1])
			}
			return
		}
	}
	t.Error("--state flag not found in gh args")
}

// TestFindStateMergedPassesMergedArg verifies that StateMerged sends --state merged to gh.
func TestFindStateMergedPassesMergedArg(t *testing.T) {
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`[]`)})

	cfg := buildConfig()
	if _, err := pr.Find(context.Background(), fake, cfg, "[PLAN-00002-3]", pr.StateMerged); err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	args := fake.Calls[0].Args
	for i, a := range args {
		if a == "--state" && i+1 < len(args) {
			if args[i+1] != "merged" {
				t.Errorf("--state arg = %q, want merged", args[i+1])
			}
			return
		}
	}
	t.Error("--state flag not found in gh args")
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
	ce, ok := err.(*clierr.CLIError)
	if !ok {
		t.Fatalf("expected *clierr.CLIError, got %T: %v", err, err)
	}
	if ce.Code != 1 {
		t.Errorf("expected exit code 1, got %d", ce.Code)
	}
}
