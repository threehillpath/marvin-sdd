// Package worktree provides git worktree lifecycle operations.
// All operations are mediated through an exec.Runner so they can be tested
// without real git state.
package worktree

import (
	"context"
	"fmt"
	"strings"

	"threehillpath.com/claude-plan-workflow/tool/internal/clierr"
	"threehillpath.com/claude-plan-workflow/tool/internal/exec"
)

// localBranchExists returns true if <branch> is already checked out as a local branch.
func localBranchExists(ctx context.Context, runner exec.Runner, branch string) (bool, error) {
	stdout, stderr, code, err := runner.Run(ctx, "git", "branch", "--list", branch)
	if err != nil {
		return false, fmt.Errorf("worktree: check local branch: %w", err)
	}
	if code != 0 {
		return false, fmt.Errorf("worktree: git branch --list exited %d: %s", code, stderr)
	}
	return strings.TrimSpace(string(stdout)) != "", nil
}

// remoteBranchExists returns true if origin/<branch> exists on the remote.
func remoteBranchExists(ctx context.Context, runner exec.Runner, branch string) (bool, error) {
	stdout, stderr, code, err := runner.Run(ctx, "git", "ls-remote", "--heads", "origin", branch)
	if err != nil {
		return false, fmt.Errorf("worktree: check remote branch: %w", err)
	}
	if code != 0 {
		return false, fmt.Errorf("worktree: git ls-remote exited %d: %s", code, stderr)
	}
	return strings.TrimSpace(string(stdout)) != "", nil
}

// worktreeAlreadyAdded returns true when path is already registered as a worktree
// for branch by parsing git worktree list --porcelain output.
func worktreeAlreadyAdded(ctx context.Context, runner exec.Runner, path, branch string) (bool, error) {
	stdout, stderr, code, err := runner.Run(ctx, "git", "worktree", "list", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("worktree: list: %w", err)
	}
	if code != 0 {
		return false, fmt.Errorf("worktree: git worktree list exited %d: %s", code, stderr)
	}
	targetRef := "refs/heads/" + branch
	var inTarget bool
	for _, line := range strings.Split(string(stdout), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "worktree ") {
			inTarget = strings.TrimPrefix(line, "worktree ") == path
		} else if inTarget && strings.HasPrefix(line, "branch ") {
			if strings.TrimPrefix(line, "branch ") == targetRef {
				return true, nil
			}
		}
	}
	return false, nil
}

// Add creates a git worktree at path for branch, handling three cases:
//
//   - path is already a worktree for branch: no-op success (idempotent).
//   - Local branch already exists for a different worktree: returns CLIError{Code:1}.
//   - Remote exists, local does not: fetches the remote branch, then adds the worktree.
//   - Neither local nor remote: creates the branch from baseBranch, pushes to origin,
//     then adds the worktree.
func Add(ctx context.Context, runner exec.Runner, path, branch, baseBranch string) error {
	// Check local branch.
	local, err := localBranchExists(ctx, runner, branch)
	if err != nil {
		return err
	}
	if local {
		// Idempotent: same path already checked out to same branch → no-op.
		already, err := worktreeAlreadyAdded(ctx, runner, path, branch)
		if err != nil {
			return err
		}
		if already {
			return nil
		}
		return clierr.Operational(fmt.Sprintf("worktree add: local branch %q already exists; will not overwrite", branch))
	}

	// Check remote branch.
	remote, err := remoteBranchExists(ctx, runner, branch)
	if err != nil {
		return err
	}

	if remote {
		// Fetch and track the remote branch.
		_, stderr, code, err := runner.Run(ctx, "git", "fetch", "origin", branch)
		if err != nil {
			return fmt.Errorf("worktree add: git fetch: %w", err)
		}
		if code != 0 {
			return fmt.Errorf("worktree add: git fetch exited %d: %s", code, stderr)
		}
	} else {
		// Create local branch from baseBranch.
		_, stderr, code, err := runner.Run(ctx, "git", "branch", branch, baseBranch)
		if err != nil {
			return fmt.Errorf("worktree add: git branch: %w", err)
		}
		if code != 0 {
			return fmt.Errorf("worktree add: git branch exited %d: %s", code, stderr)
		}

		// Push to origin and set upstream.
		_, stderr, code, err = runner.Run(ctx, "git", "push", "-u", "origin", branch)
		if err != nil {
			return fmt.Errorf("worktree add: git push: %w", err)
		}
		if code != 0 {
			return fmt.Errorf("worktree add: git push exited %d: %s", code, stderr)
		}
	}

	// Add the worktree.
	_, stderr, code, err := runner.Run(ctx, "git", "worktree", "add", path, branch)
	if err != nil {
		return fmt.Errorf("worktree add: git worktree add: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("worktree add: git worktree add exited %d: %s", code, stderr)
	}
	return nil
}

// Remove removes the worktree at path via git worktree remove.
func Remove(ctx context.Context, runner exec.Runner, path string) error {
	_, stderr, code, err := runner.Run(ctx, "git", "worktree", "remove", path)
	if err != nil {
		return fmt.Errorf("worktree remove: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("worktree remove: exited %d: %s", code, stderr)
	}
	return nil
}

// Prune prunes stale worktree administrative files via git worktree prune.
func Prune(ctx context.Context, runner exec.Runner) error {
	_, stderr, code, err := runner.Run(ctx, "git", "worktree", "prune")
	if err != nil {
		return fmt.Errorf("worktree prune: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("worktree prune: exited %d: %s", code, stderr)
	}
	return nil
}
