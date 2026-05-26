package board_test

import (
	"context"
	"testing"

	"threehillpath.com/claude-plan-workflow/tool/internal/board"
	"threehillpath.com/claude-plan-workflow/tool/internal/exectest"
)

// TestAddItem verifies that AddItem calls gh project item-add and returns the item ID.
func TestAddItem(t *testing.T) {
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"id":"ITEM_XYZ"}`)})

	cfg := buildConfig()
	id, err := board.AddItem(context.Background(), fake, cfg, 1)
	if err != nil {
		t.Fatalf("AddItem error: %v", err)
	}
	if id != "ITEM_XYZ" {
		t.Errorf("AddItem ID = %q, want ITEM_XYZ", id)
	}
	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(fake.Calls))
	}
	if !contains(fake.Calls[0].Args, "item-add") {
		t.Errorf("expected item-add arg, got %v", fake.Calls[0].Args)
	}
}

// TestSetStatusNA verifies that SetStatus with n/a status is a no-op.
func TestSetStatusNA(t *testing.T) {
	fake := &exectest.FakeRunner{}
	cfg := buildConfig()
	cfg.Statuses["in_review"] = "n/a"

	if err := board.SetStatus(context.Background(), fake, cfg, "ITEM_001", "in_review"); err != nil {
		t.Fatalf("SetStatus n/a returned error: %v", err)
	}
	if len(fake.Calls) != 0 {
		t.Errorf("expected 0 calls for n/a, got %d", len(fake.Calls))
	}
}

// TestSetStatusResolvesOptionID verifies that SetStatus passes the right option ID.
func TestSetStatusResolvesOptionID(t *testing.T) {
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{}`)})

	cfg := buildConfig()
	if err := board.SetStatus(context.Background(), fake, cfg, "ITEM_001", "done"); err != nil {
		t.Fatalf("SetStatus error: %v", err)
	}
	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(fake.Calls))
	}
	if !contains(fake.Calls[0].Args, "done-option-id") {
		t.Errorf("SetStatus did not pass done-option-id: %v", fake.Calls[0].Args)
	}
}
