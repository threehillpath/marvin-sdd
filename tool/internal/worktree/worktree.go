// Package worktree provides git worktree lifecycle operations.
// All operations are mediated through an exec.Runner so they can be tested
// without real git state.
package worktree

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"threehillpath.com/marvin-sdd/tool/internal/clierr"
	"threehillpath.com/marvin-sdd/tool/internal/exec"
)

// RepoRoot returns the main repository's root, correct whether invoked from
// the main checkout or a linked worktree of it. It relies on
// `--git-common-dir`, which always points at the main repo's .git directory
// (never a linked worktree's private administrative area), so its parent is
// the main root regardless of which worktree the process is running from.
// Exported so other packages (e.g. findings) can anchor their own
// CWD-relative paths against the same root, rather than the calling
// process's CWD.
func RepoRoot(ctx context.Context, runner exec.Runner) (string, error) {
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

// Resolve turns a repository-relative path into an absolute one, deterministically
// and entirely from calls Resolve makes itself (git, then the OS) — never by
// trusting a path an external caller has already resolved. path must be
// relative to the repository root; an absolute path is rejected outright
// rather than trusted verbatim, since a caller-supplied absolute path is
// exactly the kind of unverified input that could otherwise point anywhere
// on disk. path must also not escape the repository root via `..`
// traversal.
//
// The resulting absolute path is further constrained to this process's
// execution scope: it must be the calling process's own working directory,
// or a descendant of it. This lets a process running at an ancestor
// position (typically the main repo root) operate on any worktree beneath
// it — the ordinary "orchestrator cleans up a phase worktree" case — while
// refusing to resolve into a sibling worktree (one linked worktree has no
// business touching another's files) or "up" into an ancestor directory
// (which could be the main checkout itself, or a directory further up the
// tree). Determinism follows from this: given the same git and filesystem
// state, Resolve always computes the same result, and it never depends on
// a value handed to it that it has not itself verified.
func Resolve(ctx context.Context, runner exec.Runner, path string) (string, error) {
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("worktree: resolve: path must be relative to the repository root, got absolute path %q", path)
	}
	cleaned := filepath.Clean(path)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("worktree: resolve: path %q escapes the repository root", path)
	}

	root, err := RepoRoot(ctx, runner)
	if err != nil {
		return "", err
	}
	resolved := filepath.Join(root, cleaned)

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("worktree: resolve: determine working directory: %w", err)
	}
	if !withinExecutionScope(cwd, resolved) {
		return "", fmt.Errorf("worktree: resolve: %q is outside this process's execution scope (working directory %q is neither %q itself nor an ancestor of it)", resolved, cwd, resolved)
	}
	return resolved, nil
}

// withinExecutionScope reports whether target is cwd itself or a descendant
// of cwd. This is deliberately one-directional: a process running from an
// ancestor position (cwd above target) may reach down into target, but a
// process running from inside target may never resolve back "up" past its
// own cwd, and two paths that share neither relationship (siblings) are
// always rejected.
func withinExecutionScope(cwd, target string) bool {
	cwd = filepath.Clean(cwd)
	target = filepath.Clean(target)
	if cwd == target {
		return true
	}
	rel, err := filepath.Rel(cwd, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
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
		if wtPath, ok := strings.CutPrefix(line, "worktree "); ok {
			inTarget = wtPath == path
		} else if branchRef, ok := strings.CutPrefix(line, "branch "); inTarget && ok {
			if branchRef == targetRef {
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
		if wtPath, ok := strings.CutPrefix(line, "worktree "); ok && wtPath == resolved {
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
