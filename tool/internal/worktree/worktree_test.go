package worktree_test

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"threehillpath.com/marvin-sdd/tool/internal/clierr"
	"threehillpath.com/marvin-sdd/tool/internal/exectest"
	"threehillpath.com/marvin-sdd/tool/internal/worktree"
)

// TestRepoRootFromGitCommonDir verifies that Resolve (via the unexported
// repoRoot helper) issues exactly `git rev-parse --path-format=absolute
// --git-common-dir` and joins a relative path against filepath.Dir of the
// reply, never against the calling process's CWD.
func TestRepoRootFromGitCommonDir(t *testing.T) {
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("/repo/.git\n")})

	got, err := worktree.Resolve(context.Background(), fake, "phase-3")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d: %v", len(fake.Calls), fake.Calls)
	}
	c := fake.Calls[0]
	wantArgs := []string{"rev-parse", "--path-format=absolute", "--git-common-dir"}
	if c.Name != "git" || !reflect.DeepEqual(c.Args, wantArgs) {
		t.Fatalf("unexpected call: %v %v", c.Name, c.Args)
	}

	want := filepath.Join("/repo", "phase-3")
	if got != want {
		t.Errorf("Resolve(%q) = %q, want %q", "phase-3", got, want)
	}
}

// TestResolveAbsoluteShortCircuit verifies that Resolve returns an already-
// absolute path unchanged, issuing zero runner calls.
func TestResolveAbsoluteShortCircuit(t *testing.T) {
	fake := &exectest.FakeRunner{}

	got, err := worktree.Resolve(context.Background(), fake, "/tmp/wt/phase-3")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if got != "/tmp/wt/phase-3" {
		t.Errorf("Resolve = %q, want unchanged absolute path", got)
	}
	if len(fake.Calls) != 0 {
		t.Errorf("expected 0 calls for an absolute path, got %d: %v", len(fake.Calls), fake.Calls)
	}
}

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

// TestAddIdempotencyCheckUsesResolvedPath verifies that Add resolves a
// relative path before comparing it in worktreeAlreadyAdded — the check must
// compare against the same resolved (absolute) value that the actual
// git worktree add call would use, not the raw CWD-relative input.
func TestAddIdempotencyCheckUsesResolvedPath(t *testing.T) {
	fake := &exectest.FakeRunner{}
	// git rev-parse --path-format=absolute --git-common-dir
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("/repo/.git\n")})
	// git branch --list → branch exists locally.
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("  feature/plan-00002-3\n")})
	// git worktree list --porcelain → registered at the RESOLVED absolute path.
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(
		"worktree /repo/.worktrees/phase-3\nHEAD abc123\nbranch refs/heads/feature/plan-00002-3\n\n",
	)})

	err := worktree.Add(context.Background(), fake, ".worktrees/phase-3", "feature/plan-00002-3", "feature/plan-00002")
	if err != nil {
		t.Fatalf("Add should be a no-op when the resolved path is already registered, got: %v", err)
	}
	if len(fake.Calls) != 3 {
		t.Fatalf("expected 3 calls, got %d: %v", len(fake.Calls), fake.Calls)
	}
}

// TestAddCreationCallUsesResolvedPath verifies that Add's actual
// `git worktree add` call is passed the same resolved (absolute) path used
// by the idempotency check, not the raw CWD-relative input — the prior bug
// this phase fixes had the two halves resolving against different roots.
func TestAddCreationCallUsesResolvedPath(t *testing.T) {
	fake := &exectest.FakeRunner{}
	// git rev-parse --path-format=absolute --git-common-dir
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("/repo/.git\n")})
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
	// git worktree add <resolved-path> <branch> succeeds
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("")})

	err := worktree.Add(context.Background(), fake, ".worktrees/phase-3", "feature/plan-00002-3", "feature/plan-00002")
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}

	var addCall *exectest.Call
	for i, c := range fake.Calls {
		if c.Name == "git" && len(c.Args) >= 2 && c.Args[0] == "worktree" && c.Args[1] == "add" {
			addCall = &fake.Calls[i]
		}
	}
	if addCall == nil {
		t.Fatalf("expected git worktree add to be called; calls: %v", fake.Calls)
	}
	want := filepath.Join("/repo", ".worktrees/phase-3")
	if addCall.Args[2] != want {
		t.Errorf("git worktree add called with path %q, want resolved path %q", addCall.Args[2], want)
	}
}

// TestRemoveCalls verifies that Remove, for a path registered as a worktree,
// issues `git worktree list --porcelain` (the source-of-truth check) followed
// by `git worktree remove --force <resolved-path>`.
func TestRemoveCalls(t *testing.T) {
	fake := &exectest.FakeRunner{}
	// git worktree list --porcelain → path is registered.
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(
		"worktree /tmp/wt/phase-3\nHEAD abc123\nbranch refs/heads/feature/plan-00002-3\n\n",
	)})
	// git worktree remove --force succeeds.
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("")})

	if err := worktree.Remove(context.Background(), fake, "/tmp/wt/phase-3"); err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}

	if len(fake.Calls) != 2 {
		t.Fatalf("expected 2 calls, got %d: %v", len(fake.Calls), fake.Calls)
	}
	listCall := fake.Calls[0]
	if listCall.Name != "git" || !reflect.DeepEqual(listCall.Args, []string{"worktree", "list", "--porcelain"}) {
		t.Errorf("unexpected first call: %v %v", listCall.Name, listCall.Args)
	}
	removeCall := fake.Calls[1]
	if removeCall.Name != "git" || removeCall.Args[0] != "worktree" || removeCall.Args[1] != "remove" || removeCall.Args[2] != "--force" || removeCall.Args[3] != "/tmp/wt/phase-3" {
		t.Errorf("unexpected second call: %v %v", removeCall.Name, removeCall.Args)
	}
}

// TestRemoveSkipsIfPathMissing verifies that Remove returns nil, issuing only
// the `git worktree list --porcelain` check (no `git worktree remove` call),
// when the path is not registered as a worktree with git — the new
// registration-based no-op semantics (replacing the old os.Stat check).
func TestRemoveSkipsIfPathMissing(t *testing.T) {
	fake := &exectest.FakeRunner{}
	// git worktree list --porcelain → no worktree registered at this path.
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(
		"worktree /tmp/wt/other-path\nHEAD abc123\nbranch refs/heads/feature/plan-00002-3\n\n",
	)})

	if err := worktree.Remove(context.Background(), fake, "/nonexistent/path/phase-00042-3"); err != nil {
		t.Fatalf("Remove returned error for unregistered path: %v", err)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("expected exactly 1 call (list only), got %d: %v", len(fake.Calls), fake.Calls)
	}
	c := fake.Calls[0]
	if c.Name != "git" || !reflect.DeepEqual(c.Args, []string{"worktree", "list", "--porcelain"}) {
		t.Errorf("unexpected call: %v %v", c.Name, c.Args)
	}
}

// TestRemoveResolvesRelativePath verifies that Remove resolves a relative
// path against repoRoot before checking registration and before calling
// git worktree remove — the resolved (absolute) path must be what's compared
// against git's own porcelain output and what's passed to the remove call.
func TestRemoveResolvesRelativePath(t *testing.T) {
	fake := &exectest.FakeRunner{}
	// git rev-parse --path-format=absolute --git-common-dir
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("/repo/.git\n")})
	// git worktree list --porcelain → registered at the resolved absolute path.
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(
		"worktree /repo/.worktrees/phase-3\nHEAD abc123\nbranch refs/heads/feature/plan-00002-3\n\n",
	)})
	// git worktree remove --force succeeds.
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("")})

	if err := worktree.Remove(context.Background(), fake, ".worktrees/phase-3"); err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}

	if len(fake.Calls) != 3 {
		t.Fatalf("expected 3 calls, got %d: %v", len(fake.Calls), fake.Calls)
	}
	removeCall := fake.Calls[2]
	wantPath := filepath.Join("/repo", ".worktrees/phase-3")
	if removeCall.Args[3] != wantPath {
		t.Errorf("git worktree remove called with %q, want %q", removeCall.Args[3], wantPath)
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
