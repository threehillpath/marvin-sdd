package worktree_test

import (
	"context"
	"strings"
	"testing"

	"threehillpath.com/marvin-sdd/tool/internal/clierr"
	"threehillpath.com/marvin-sdd/tool/internal/exectest"
	"threehillpath.com/marvin-sdd/tool/internal/worktree"
)

// TestAddLocalBranchExistsReturnsError verifies that Add exits with CLIError{Code:1}
// (no git worktree add issued) when the local branch exists for a different path.
func TestAddLocalBranchExistsReturnsError(t *testing.T) {
	fake := &exectest.FakeRunner{}
	// git branch --list <branch> returns the branch name → local branch exists.
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("  feature/plan-00002-3\n")})
	// git worktree list --porcelain → branch checked out at a DIFFERENT path.
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(
		"worktree /tmp/wt/other-path\nHEAD abc123\nbranch refs/heads/feature/plan-00002-3\n\n",
	)})

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

// TestAddIdempotentNoop verifies that Add is a no-op when path is already a
// worktree for branch (same path + same branch → idempotent success).
func TestAddIdempotentNoop(t *testing.T) {
	fake := &exectest.FakeRunner{}
	// git branch --list → branch exists locally.
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("  feature/plan-00002-3\n")})
	// git worktree list --porcelain → same path already checked out to same branch.
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(
		"worktree /tmp/wt/phase-3\nHEAD abc123\nbranch refs/heads/feature/plan-00002-3\n\n",
	)})

	err := worktree.Add(context.Background(), fake, "/tmp/wt/phase-3", "feature/plan-00002-3", "feature/plan-00002")
	if err != nil {
		t.Fatalf("Add should be a no-op for idempotent re-add, got: %v", err)
	}
	// No git worktree add should have been issued.
	for _, c := range fake.Calls {
		if c.Name == "git" && len(c.Args) >= 2 && c.Args[0] == "worktree" && c.Args[1] == "add" {
			t.Errorf("git worktree add should not be called for idempotent re-add; got: %v", c.Args)
		}
	}
}

// TestAddNeitherLocalNorRemoteProceedsToWorktreeAdd verifies that when neither
// the local nor remote branch exists, Add issues git fetch (of baseBranch),
// git branch, git push, and git worktree add — in that order.
func TestAddNeitherLocalNorRemoteProceedsToWorktreeAdd(t *testing.T) {
	fake := &exectest.FakeRunner{}
	// git branch --list → empty (no local branch)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("")})
	// git ls-remote → empty (no remote branch)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("")})
	// git fetch origin <baseBranch> succeeds
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("")})
	// git branch <branch> origin/<baseBranch> succeeds
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

// TestAddNeitherLocalNorRemoteFetchesBaseBeforeBranching verifies the fix for
// the stale-base-branch bug: Add must fetch baseBranch from origin, and create
// the new branch from origin/<baseBranch> (not the possibly-stale local ref),
// so a just-merged prior phase's PR is always included in the next phase.
func TestAddNeitherLocalNorRemoteFetchesBaseBeforeBranching(t *testing.T) {
	fake := &exectest.FakeRunner{}
	// git branch --list → empty (no local branch)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("")})
	// git ls-remote → empty (no remote branch)
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("")})
	// git fetch origin <baseBranch> succeeds
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("")})
	// git branch <branch> origin/<baseBranch> succeeds
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("")})
	// git push -u origin <branch> succeeds
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("")})
	// git worktree add <path> <branch> succeeds
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("")})

	err := worktree.Add(context.Background(), fake, "/tmp/wt/phase-3", "feature/plan-00002-3", "feature/plan-00002")
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}

	var fetchIdx, branchIdx = -1, -1
	for i, c := range fake.Calls {
		if c.Name != "git" || len(c.Args) < 2 {
			continue
		}
		switch c.Args[0] {
		case "fetch":
			if c.Args[1] == "origin" && len(c.Args) >= 3 && c.Args[2] == "feature/plan-00002" {
				fetchIdx = i
			}
		case "branch":
			if c.Args[1] != "--list" {
				branchIdx = i
				if len(c.Args) < 3 || c.Args[2] != "origin/feature/plan-00002" {
					t.Errorf("expected git branch to use origin/feature/plan-00002 as source, got: %v", c.Args)
				}
			}
		}
	}
	if fetchIdx == -1 {
		t.Fatalf("expected git fetch origin feature/plan-00002 to be called; calls: %v", fake.Calls)
	}
	if branchIdx == -1 {
		t.Fatalf("expected git branch create to be called; calls: %v", fake.Calls)
	}
	if fetchIdx > branchIdx {
		t.Errorf("expected git fetch to happen before git branch create; fetchIdx=%d branchIdx=%d", fetchIdx, branchIdx)
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

// TestRemoveCalls git worktree remove --force on an existing path.
func TestRemoveCalls(t *testing.T) {
	// Create a real temp dir so the os.Stat existence check passes.
	dir := t.TempDir()

	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("")})

	if err := worktree.Remove(context.Background(), fake, dir); err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(fake.Calls))
	}
	c := fake.Calls[0]
	if c.Name != "git" || c.Args[0] != "worktree" || c.Args[1] != "remove" || c.Args[2] != "--force" {
		t.Errorf("unexpected call: %v %v", c.Name, c.Args)
	}
}

// TestRemoveSkipsIfPathMissing verifies that Remove returns nil without calling
// git when the worktree path has already been removed.
func TestRemoveSkipsIfPathMissing(t *testing.T) {
	fake := &exectest.FakeRunner{}

	if err := worktree.Remove(context.Background(), fake, "/nonexistent/path/phase-00042-3"); err != nil {
		t.Fatalf("Remove returned error for missing path: %v", err)
	}

	if len(fake.Calls) != 0 {
		t.Errorf("expected no git calls for missing path, got %d", len(fake.Calls))
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
