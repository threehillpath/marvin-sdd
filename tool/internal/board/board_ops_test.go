package board_test

import (
	"context"
	"testing"

	"threehillpath.com/marvin-sdd/tool/internal/board"
	"threehillpath.com/marvin-sdd/tool/internal/exectest"
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

	if err := board.SetStatus(context.Background(), fake, cfg, 42, "in_review"); err != nil {
		t.Fatalf("SetStatus n/a returned error: %v", err)
	}
	if len(fake.Calls) != 0 {
		t.Errorf("expected 0 calls for n/a, got %d", len(fake.Calls))
	}
}

// TestListReturnsAllItems verifies that List with no status filter returns all
// items with a linked issue (content.number > 0).
func TestListReturnsAllItems(t *testing.T) {
	fake := &exectest.FakeRunner{}
	payload := []byte(`{"items":[` +
		`{"id":"I1","title":"Phase 1","status":"In Progress","content":{"number":10,"title":"Phase 1","url":"https://github.com/o/r/issues/10"}},` +
		`{"id":"I2","title":"Draft","status":"Backlog","content":{"number":0,"title":"","url":""}}` +
		`]}`)
	fake.Enqueue(exectest.FakeResponse{Stdout: payload})

	cfg := buildConfig()
	items, err := board.List(context.Background(), fake, cfg, "", 0)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item (draft skipped), got %d", len(items))
	}
	if items[0].Number != 10 {
		t.Errorf("Number = %d, want 10", items[0].Number)
	}
	if items[0].Status != "In Progress" {
		t.Errorf("Status = %q, want In Progress", items[0].Status)
	}
}

// TestListFiltersByStatus verifies that --status filters items case-insensitively
// and accepts both display names and config-key forms.
func TestListFiltersByStatus(t *testing.T) {
	fake := &exectest.FakeRunner{}
	payload := []byte(`{"items":[` +
		`{"id":"I1","title":"Phase 1","status":"In Progress","content":{"number":10,"title":"Phase 1","url":""}},` +
		`{"id":"I2","title":"Phase 2","status":"In Review","content":{"number":11,"title":"Phase 2","url":""}}` +
		`]}`)
	fake.Enqueue(exectest.FakeResponse{Stdout: payload})

	cfg := buildConfig()
	items, err := board.List(context.Background(), fake, cfg, "in_progress", 0)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item after status filter, got %d", len(items))
	}
	if items[0].Number != 10 {
		t.Errorf("Number = %d, want 10", items[0].Number)
	}
}

// TestStatusFound verifies that Status returns the status string for a known issue.
func TestStatusFound(t *testing.T) {
	fake := &exectest.FakeRunner{}
	payload := []byte(`{"items":[{"id":"I1","title":"Phase 1","status":"In Review","content":{"number":42,"title":"Phase 1","url":""}}]}`)
	fake.Enqueue(exectest.FakeResponse{Stdout: payload})

	cfg := buildConfig()
	status, err := board.Status(context.Background(), fake, cfg, 42)
	if err != nil {
		t.Fatalf("Status error: %v", err)
	}
	if status != "In Review" {
		t.Errorf("Status = %q, want In Review", status)
	}
}

// TestStatusNotOnBoard verifies that Status returns "not-on-board" for a missing issue.
func TestStatusNotOnBoard(t *testing.T) {
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"items":[]}`)})

	cfg := buildConfig()
	status, err := board.Status(context.Background(), fake, cfg, 99)
	if err != nil {
		t.Fatalf("Status error: %v", err)
	}
	if status != "not-on-board" {
		t.Errorf("Status = %q, want not-on-board", status)
	}
}

// TestSetStatusResolvesOptionID verifies that SetStatus adds the issue to the
// board and then passes the correct option ID to item-edit.
func TestSetStatusResolvesOptionID(t *testing.T) {
	fake := &exectest.FakeRunner{}
	// 1) item-add returns an item ID
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{"id":"ITEM_XYZ"}`)})
	// 2) item-edit succeeds
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(`{}`)})

	cfg := buildConfig()
	if err := board.SetStatus(context.Background(), fake, cfg, 42, "done"); err != nil {
		t.Fatalf("SetStatus error: %v", err)
	}
	if len(fake.Calls) != 2 {
		t.Fatalf("expected 2 calls (item-add + item-edit), got %d", len(fake.Calls))
	}
	if !contains(fake.Calls[0].Args, "item-add") {
		t.Errorf("call[0] should be item-add, got %v", fake.Calls[0].Args)
	}
	if !contains(fake.Calls[1].Args, "done-option-id") {
		t.Errorf("SetStatus did not pass done-option-id: %v", fake.Calls[1].Args)
	}
}
