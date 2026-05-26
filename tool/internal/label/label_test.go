package label_test

import (
	"context"
	"strings"
	"testing"

	"github.com/threehillpath/claude-plan-workflow/tool/internal/config"
	"github.com/threehillpath/claude-plan-workflow/tool/internal/exectest"
	"github.com/threehillpath/claude-plan-workflow/tool/internal/label"
)

func buildConfig() *config.Config {
	return &config.Config{
		Repo:         "owner/repo",
		WorktreeBase: ".worktrees",
		Statuses:     map[string]string{},
	}
}

// TestEnsureCreatesLabel verifies that Ensure calls gh label create when the label
// does not exist (gh label list returns empty JSON array).
func TestEnsureCreatesLabel(t *testing.T) {
	fake := &exectest.FakeRunner{}
	// gh label list returns empty list → label absent
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`[]`)})
	// gh label create succeeds
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(``)})

	cfg := buildConfig()
	if err := label.Ensure(context.Background(), fake, cfg, "plan:arch", "Architecture plans", "0075ca"); err != nil {
		t.Fatalf("Ensure returned error: %v", err)
	}

	if len(fake.Calls) != 2 {
		t.Fatalf("expected 2 calls, got %d: %v", len(fake.Calls), fake.Calls)
	}

	c0 := fake.Calls[0]
	if !containsArg(c0.Args, "list") {
		t.Errorf("call[0] missing 'list': %v", c0.Args)
	}

	c1 := fake.Calls[1]
	if !containsArg(c1.Args, "create") {
		t.Errorf("call[1] missing 'create': %v", c1.Args)
	}
	if !containsArg(c1.Args, "plan:arch") {
		t.Errorf("call[1] missing label name: %v", c1.Args)
	}
}

// TestEnsureIdempotent verifies that Ensure is a no-op when the label already exists.
func TestEnsureIdempotent(t *testing.T) {
	fake := &exectest.FakeRunner{}
	// gh label list returns a match
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`[{"name":"plan:arch","color":"0075ca","description":"Architecture plans"}]`)})

	cfg := buildConfig()
	if err := label.Ensure(context.Background(), fake, cfg, "plan:arch", "Architecture plans", "0075ca"); err != nil {
		t.Fatalf("Ensure returned error for existing label: %v", err)
	}

	// Only 1 call (label list) — no create
	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d: %v", len(fake.Calls), fake.Calls)
	}
	if containsArg(fake.Calls[0].Args, "create") {
		t.Error("Ensure called gh label create for existing label")
	}
}

func containsArg(args []string, needle string) bool {
	for _, a := range args {
		if strings.Contains(a, needle) || a == needle {
			return true
		}
	}
	return false
}
