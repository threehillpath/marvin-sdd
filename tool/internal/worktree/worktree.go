// Package worktree provides git worktree lifecycle operations.
// All operations are mediated through an exec.Runner so they can be tested
// without real git state.
package worktree

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"threehillpath.com/marvin-sdd/tool/internal/clierr"
	"threehillpath.com/marvin-sdd/tool/internal/exec"
)

// repoRoot returns the main repository's root, correct whether invoked from
// the main checkout or a linked worktree of it. It relies on
// `--git-common-dir`, which always points at the main repo's .git directory
// (never a linked worktree's private administrative area), so its parent is
// the main root regardless of which worktree the process is running from.
func repoRoot(ctx context.Context, runner exec.Runner) (string, error) {
	stdout, stderr, code, err := runner.Run(ctx, "git", "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", fmt.Errorf("worktree: git rev-parse --git-common-dir: %w", err)
	}
	if code != 0 {
		return "", fmt.Errorf("worktree: git rev-parse --git-common-dir exited %d: %s", code, stderr)
	}
	commonDir := strings.TrimSpace(string(stdout))
	return filepath.Dir(commonDir), nil
}

// Resolve returns path as an absolute path: unchanged if already absolute
// (zero runner calls), otherwise joined against repoRoot — never against the
// calling process's CWD, which may not be the main repo root when invoked
// from inside a linked worktree.
func Resolve(ctx context.Context, runner exec.Runner, path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}
	root, err := repoRoot(ctx, runner)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, path), nil
}

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
//   - Neither local nor remote: fetches baseBranch from origin, creates the new
//     branch from origin/baseBranch (so a just-merged prior phase's PR is
//     included even if the local baseBranch ref is stale), pushes to origin,
//     then adds the worktree.
func Add(ctx context.Context, runner exec.Runner, path, branch, baseBranch string) error {
	// Resolve path once, up front. Both the idempotency check below and the
	// actual `git worktree add` call at the end must use this same resolved
	// value — a prior bug had them resolving against different roots.
	resolved, err := Resolve(ctx, runner, path)
	if err != nil {
		return err
	}

	// Check local branch.
	local, err := localBranchExists(ctx, runner, branch)
	if err != nil {
		return err
	}
	if local {
		// Idempotent: same path already checked out to same branch → no-op.
		already, err := worktreeAlreadyAdded(ctx, runner, resolved, branch)
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
		// Fetch baseBranch from origin first: baseBranch is typically the
		// implementation branch, and a prior phase's PR may have just been
		// merged into it on GitHub. Without this, a stale local ref would
		// silently drop the previous phase's merged work from the new phase
		// branch. Branching from origin/baseBranch (not the local ref)
		// guarantees the new branch reflects the latest merged state.
		_, stderr, code, err := runner.Run(ctx, "git", "fetch", "origin", baseBranch)
		if err != nil {
			return fmt.Errorf("worktree add: git fetch base: %w", err)
		}
		if code != 0 {
			return fmt.Errorf("worktree add: git fetch base exited %d: %s", code, stderr)
		}

		// Create local branch from the freshly-fetched base.
		_, stderr, code, err = runner.Run(ctx, "git", "branch", branch, "origin/"+baseBranch)
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

	// Add the worktree, using the same resolved path as the idempotency check above.
	_, stderr, code, err := runner.Run(ctx, "git", "worktree", "add", resolved, branch)
	if err != nil {
		return fmt.Errorf("worktree add: git worktree add: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("worktree add: git worktree add exited %d: %s", code, stderr)
	}
	return nil
}

// isRegisteredWorktree returns true when resolved (an absolute path) appears
// as a `worktree <path>` line in `git worktree list --porcelain` output —
// i.e. whether git still knows about this worktree at all, regardless of
// whether the directory itself still exists on disk (a manually-deleted but
// still-registered "prunable" worktree is registered).
func isRegisteredWorktree(ctx context.Context, runner exec.Runner, resolved string) (bool, error) {
	stdout, stderr, code, err := runner.Run(ctx, "git", "worktree", "list", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("worktree: list: %w", err)
	}
	if code != 0 {
		return false, fmt.Errorf("worktree: git worktree list exited %d: %s", code, stderr)
	}
	for _, line := range strings.Split(string(stdout), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "worktree ") && strings.TrimPrefix(line, "worktree ") == resolved {
			return true, nil
		}
	}
	return false, nil
}

// Remove removes the worktree at path via git worktree remove --force, using
// git's own bookkeeping (git worktree list --porcelain) as the sole source of
// truth for whether there is anything to remove: not registered → no-op;
// registered → git worktree remove --force, which already correctly handles
// a manually-deleted-but-still-registered ("prunable") worktree. path is
// resolved via Resolve before either check, so a CWD-relative path is
// anchored against the main repo root rather than the calling process's CWD.
func Remove(ctx context.Context, runner exec.Runner, path string) error {
	resolved, err := Resolve(ctx, runner, path)
	if err != nil {
		return err
	}
	registered, err := isRegisteredWorktree(ctx, runner, resolved)
	if err != nil {
		return err
	}
	if !registered {
		return nil
	}
	_, stderr, code, err := runner.Run(ctx, "git", "worktree", "remove", "--force", resolved)
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
