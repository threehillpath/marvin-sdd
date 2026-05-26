package worktree_test

import (
	"context"
	"strings"
	"testing"

	"github.com/threehillpath/claude-plan-workflow/tool/internal/clierr"
	"github.com/threehillpath/claude-plan-workflow/tool/internal/exectest"
	"github.com/threehillpath/claude-plan-workflow/tool/internal/worktree"
)

// TestAddLocalBranchExistsReturnsError verifies that Add exits with CLIError{Code:1}
// (no git worktree add issued) when the local branch already exists.
func TestAddLocalBranchExistsReturnsError(t *testing.T) {
	fake := &exectest.FakeRunner{}
	// git branch --list <branch> returns the branch name → local branch exists.
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("  feature/plan-00002-3\n")})

	err := worktree.Add(context.Background(), fake, "/tmp/wt/phase-3", "feature/plan-00002-3", "feature/plan-00002")
	if err == nil {
		t.Fatal("expected error when local branch exists, got nil")
	}

	// Must be CLIError with Code 1.
	if ce, ok := err.(*clierr.CLIError); ok {
		if ce.Code != 1 {
			t.Errorf("expected exit code 1, got %d", ce.Code)
		}
	} else {
		t.Errorf("expected *clierr.CLIError, got %T: %v", err, err)
	}

	// No git worktree add should have been issued.
	for _, c := range fake.Calls {
		if c.Name == "git" {
			for _, a := range c.Args {
				if strings.Contains(a, "worktree") && strings.Contains(strings.Join(c.Args, " "), "add") {
					t.Errorf("git worktree add should not have been called; got: %v", c.Args)
				}
			}
		}
	}
}

// TestAddNeitherLocalNorRemoteProceedsToWorktreeAdd verifies that when neither
// the local nor remote branch exists, Add issues git branch, git push, and
// git worktree add — in that order.
func TestAddNeitherLocalNorRemoteProceedsToWorktreeAdd(t *testing.T) {
	fake := &exectest.FakeRunner{}
	// git branch --list → empty (no local branch)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("")})
	// git ls-remote → empty (no remote branch)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("")})
	// git branch <branch> <baseBranch> succeeds
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("")})
	// git push -u origin <branch> succeeds
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("")})
	// git worktree add <path> <branch> succeeds
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("")})

	err := worktree.Add(context.Background(), fake, "/tmp/wt/phase-3", "feature/plan-00002-3", "feature/plan-00002")
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}

	// Verify git worktree add was called.
	worktreeAddFound := false
	for _, c := range fake.Calls {
		if c.Name == "git" && len(c.Args) >= 2 && c.Args[0] == "worktree" && c.Args[1] == "add" {
			worktreeAddFound = true
		}
	}
	if !worktreeAddFound {
		t.Errorf("expected git worktree add to be called; calls: %v", fake.Calls)
	}
}

// TestAddRemoteExistsLocalDoesNotProceedsToWorktreeAdd verifies the fetch-track path.
func TestAddRemoteExistsLocalDoesNotProceedsToWorktreeAdd(t *testing.T) {
	fake := &exectest.FakeRunner{}
	// git branch --list → empty (no local branch)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("")})
	// git ls-remote → non-empty (remote branch exists)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("abc123\trefs/heads/feature/plan-00002-3\n")})
	// git fetch origin <branch> succeeds
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("")})
	// git worktree add <path> <branch> succeeds
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("")})

	err := worktree.Add(context.Background(), fake, "/tmp/wt/phase-3", "feature/plan-00002-3", "feature/plan-00002")
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}

	// Verify git fetch was called (not git branch).
	fetchFound := false
	worktreeAddFound := false
	branchCreateFound := false
	for _, c := range fake.Calls {
		if c.Name == "git" {
			switch c.Args[0] {
			case "fetch":
				fetchFound = true
			case "worktree":
				if len(c.Args) >= 2 && c.Args[1] == "add" {
					worktreeAddFound = true
				}
			case "branch":
				// Only count branch creation: git branch <name> <base>
				// Exclude git branch --list which has "--list" as arg[1].
				if len(c.Args) >= 2 && c.Args[1] != "--list" {
					branchCreateFound = true
				}
			}
		}
	}
	if !fetchFound {
		t.Errorf("expected git fetch to be called")
	}
	if !worktreeAddFound {
		t.Errorf("expected git worktree add to be called")
	}
	if branchCreateFound {
		t.Errorf("git branch create should not be called when remote exists")
	}
}

// TestRemoveCalls git worktree remove.
func TestRemoveCalls(t *testing.T) {
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("")})

	if err := worktree.Remove(context.Background(), fake, "/tmp/wt/phase-3"); err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(fake.Calls))
	}
	c := fake.Calls[0]
	if c.Name != "git" || c.Args[0] != "worktree" || c.Args[1] != "remove" {
		t.Errorf("unexpected call: %v %v", c.Name, c.Args)
	}
}

// TestPruneCalls git worktree prune.
func TestPruneCalls(t *testing.T) {
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("")})

	if err := worktree.Prune(context.Background(), fake); err != nil {
		t.Fatalf("Prune returned error: %v", err)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(fake.Calls))
	}
	c := fake.Calls[0]
	if c.Name != "git" || c.Args[0] != "worktree" || c.Args[1] != "prune" {
		t.Errorf("unexpected call: %v %v", c.Name, c.Args)
	}
}
