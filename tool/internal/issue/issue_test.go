package issue_test

import (
	"context"
	"testing"

	"threehillpath.com/marvin-sdd/tool/internal/config"
	"threehillpath.com/marvin-sdd/tool/internal/exectest"
	"threehillpath.com/marvin-sdd/tool/internal/issue"
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

// TestCreatePassesRepoThrough verifies that Create resolves cfg.Repo and
// passes it to the underlying gh call, returning the (number, url) pair
// unchanged.
func TestCreatePassesRepoThrough(t *testing.T) {
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("https://github.com/owner/repo/issues/77\n")})

	cfg := buildConfig()
	number, url, err := issue.Create(context.Background(), fake, cfg, "title", "body", []string{"bug"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if number != 77 {
		t.Errorf("number = %d, want 77", number)
	}
	if url != "https://github.com/owner/repo/issues/77" {
		t.Errorf("url = %q, want %q", url, "https://github.com/owner/repo/issues/77")
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(fake.Calls))
	}
	args := fake.Calls[0].Args
	repoIdx := -1
	for i, a := range args {
		if a == "--repo" {
			repoIdx = i
			break
		}
	}
	if repoIdx == -1 || repoIdx+1 >= len(args) || args[repoIdx+1] != cfg.Repo {
		t.Errorf("args = %v, want --repo %q", args, cfg.Repo)
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
