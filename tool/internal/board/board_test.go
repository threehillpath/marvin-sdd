package board_test

import (
	"context"
	"strings"
	"testing"

	"threehillpath.com/claude-plan-workflow/tool/internal/board"
	"threehillpath.com/claude-plan-workflow/tool/internal/config"
	"threehillpath.com/claude-plan-workflow/tool/internal/exectest"
)

// buildConfig returns a minimal Config with a done option ID and in_progress option ID.
func buildConfig() *config.Config {
	return &config.Config{
		Repo:          "owner/repo",
		ProjectNumber: 4,
		ProjectID:     "PVT_test",
		StatusFieldID: "PVTSSF_test",
		Statuses: map[string]string{
			"done":        "done-option-id",
			"in_progress": "ip-option-id",
			"backlog":     "bl-option-id",
		},
		WorktreeBase: ".worktrees",
	}
}

// itemAddResponse is a canned gh project item-add JSON response.
const itemAddResponse = `{"id":"ITEM_001","title":"test","type":"ISSUE"}`

// TestMoveDone verifies that Move("done") issues gh project item-add,
// gh project item-edit (set-status), and gh issue close — in that order —
// and records the correct command sequence in the fake runner.
func TestMoveDone(t *testing.T) {
	fake := &exectest.FakeRunner{}
	// 1) item-add returns an item ID
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(itemAddResponse)})
	// 2) item-edit (set status) succeeds
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{}`)})
	// 3) issue close succeeds
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(``)})

	cfg := buildConfig()

	if err := board.Move(context.Background(), fake, cfg, 42, "done"); err != nil {
		t.Fatalf("Move returned error: %v", err)
	}

	if len(fake.Calls) != 3 {
		t.Fatalf("expected 3 calls, got %d: %v", len(fake.Calls), fake.Calls)
	}

	// Call 0: gh project item-add
	c0 := fake.Calls[0]
	if c0.Name != "gh" {
		t.Errorf("call[0].Name = %q, want gh", c0.Name)
	}
	if !contains(c0.Args, "item-add") {
		t.Errorf("call[0].Args missing item-add: %v", c0.Args)
	}

	// Call 1: gh project item-edit (set status)
	c1 := fake.Calls[1]
	if c1.Name != "gh" {
		t.Errorf("call[1].Name = %q, want gh", c1.Name)
	}
	if !contains(c1.Args, "item-edit") {
		t.Errorf("call[1].Args missing item-edit: %v", c1.Args)
	}
	// Must use the done option ID
	if !contains(c1.Args, "done-option-id") {
		t.Errorf("call[1].Args missing done-option-id: %v", c1.Args)
	}

	// Call 2: gh issue close
	c2 := fake.Calls[2]
	if c2.Name != "gh" {
		t.Errorf("call[2].Name = %q, want gh", c2.Name)
	}
	if !contains(c2.Args, "issue") || !contains(c2.Args, "close") {
		t.Errorf("call[2].Args missing 'issue close': %v", c2.Args)
	}
}

// TestMoveInProgress verifies that Move("in_progress") issues gh issue reopen
// instead of close.
func TestMoveInProgress(t *testing.T) {
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(itemAddResponse)})
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{}`)})
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(``)})

	cfg := buildConfig()

	if err := board.Move(context.Background(), fake, cfg, 42, "in_progress"); err != nil {
		t.Fatalf("Move returned error: %v", err)
	}

	if len(fake.Calls) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(fake.Calls))
	}

	c2 := fake.Calls[2]
	if !contains(c2.Args, "reopen") {
		t.Errorf("expected 'reopen' for in_progress, got args: %v", c2.Args)
	}
}

// TestMoveNA verifies that Move with a status mapped to "n/a" is a no-op.
func TestMoveNA(t *testing.T) {
	fake := &exectest.FakeRunner{}
	cfg := &config.Config{
		Repo:          "owner/repo",
		ProjectNumber: 4,
		ProjectID:     "PVT_test",
		StatusFieldID: "PVTSSF_test",
		Statuses: map[string]string{
			"in_review": "n/a",
		},
		WorktreeBase: ".worktrees",
	}

	if err := board.Move(context.Background(), fake, cfg, 42, "in_review"); err != nil {
		t.Fatalf("Move returned error for n/a: %v", err)
	}
	if len(fake.Calls) != 0 {
		t.Errorf("expected 0 calls for n/a status, got %d: %v", len(fake.Calls), fake.Calls)
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if strings.Contains(s, needle) || s == needle {
			return true
		}
	}
	return false
}
