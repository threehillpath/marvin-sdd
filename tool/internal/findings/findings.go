// Package findings provides plan-scoped JSON cache storage under .claude/cache/.
// Cache writes are validated as JSON before persisting. Clear removes only the
// target plan's directory, leaving sibling plans unaffected.
package findings

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"threehillpath.com/marvin-sdd/tool/internal/exec"
	"threehillpath.com/marvin-sdd/tool/internal/worktree"
)

// CachePath returns the canonical path for a cached findings file, relative
// to the repository root (no leading slash).
// Formula: .claude/cache/<planNumber>/<kind>/<name>.json
func CachePath(planNumber, kind, name string) string {
	return filepath.Join(".claude", "cache", planNumber, kind, name+".json")
}

// Cache validates payload as JSON, creates the directory tree, and writes the
// payload to CachePath(planNumber, kind, name), anchored against the main
// repository root (worktree.RepoRoot) rather than the calling process's CWD
// — so the cache lands in one consistent place regardless of whether this is
// invoked from the main checkout or a linked worktree. Returns an error if
// the payload is not valid JSON or if the file cannot be written.
func Cache(ctx context.Context, runner exec.Runner, planNumber, kind, name string, payload []byte) error {
	if !json.Valid(payload) {
		return fmt.Errorf("findings cache: payload is not valid JSON")
	}

	root, err := worktree.RepoRoot(ctx, runner)
	if err != nil {
		return fmt.Errorf("findings cache: %w", err)
	}

	path := filepath.Join(root, CachePath(planNumber, kind, name))
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("findings cache: create directories: %w", err)
	}

	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("findings cache: write file: %w", err)
	}
	return nil
}

// Clear removes the .claude/cache/<planNumber>/ directory and all its
// contents, anchored against the main repository root (worktree.RepoRoot)
// rather than the calling process's CWD. It is a no-op if the directory does
// not exist. Sibling plans are not affected.
func Clear(ctx context.Context, runner exec.Runner, planNumber string) error {
	root, err := worktree.RepoRoot(ctx, runner)
	if err != nil {
		return fmt.Errorf("findings clear: %w", err)
	}

	dir := filepath.Join(root, ".claude", "cache", planNumber)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("findings clear: %w", err)
	}
	return nil
}
