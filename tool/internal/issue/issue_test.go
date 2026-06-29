package issue_test

import (
	"context"
	"testing"

	"threehillpath.com/claude-plan-workflow/tool/internal/config"
	"threehillpath.com/claude-plan-workflow/tool/internal/exectest"
	"threehillpath.com/claude-plan-workflow/tool/internal/issue"
)

func buildConfig() *config.Config {
	return &config.Config{
		Repo:          "owner/repo",
		ProjectNumber: 4,
		ProjectID:     "PVT_test",
		StatusFieldID: "PVTSSF_test",
		Statuses:      map[string]string{"done": "done-option-id"},
		WorktreeBase:  ".worktrees",
	}
}

// TestListReturnsItems verifies that List parses the gh response and flattens labels.
func TestListReturnsItems(t *testing.T) {
	fake := &exectest.FakeRunner{}
	payload := []byte(`[{"number":5,"title":"[PLAN-00002] Phase 1: setup","state":"OPEN","labels":[{"name":"plan:phase"}]}]`)
	fake.Enqueue(exectest.FakeResponse{Stdout: payload})

	cfg := buildConfig()
	items, err := issue.List(context.Background(), fake, cfg, "plan:phase", "", "open", 50)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Number != 5 {
		t.Errorf("Number = %d, want 5", items[0].Number)
	}
	if len(items[0].Labels) != 1 || items[0].Labels[0] != "plan:phase" {
		t.Errorf("Labels = %v, want [plan:phase]", items[0].Labels)
	}
}

// TestListTitlePrefixFilter verifies that titlePrefix excludes non-matching items.
func TestListTitlePrefixFilter(t *testing.T) {
	fake := &exectest.FakeRunner{}
	payload := []byte(`[` +
		`{"number":5,"title":"[PLAN-00002] Phase 1","state":"OPEN","labels":[]},` +
		`{"number":6,"title":"[PLAN-00003] Phase 1","state":"OPEN","labels":[]}` +
		`]`)
	fake.Enqueue(exectest.FakeResponse{Stdout: payload})

	cfg := buildConfig()
	items, err := issue.List(context.Background(), fake, cfg, "", "[PLAN-00002]", "open", 50)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item after prefix filter, got %d", len(items))
	}
	if items[0].Number != 5 {
		t.Errorf("Number = %d, want 5", items[0].Number)
	}
}

// TestListEmptyResult verifies that an empty gh response returns an empty slice.
func TestListEmptyResult(t *testing.T) {
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`[]`)})

	cfg := buildConfig()
	items, err := issue.List(context.Background(), fake, cfg, "", "", "open", 10)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}
