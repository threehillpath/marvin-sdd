package findings_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"threehillpath.com/marvin-sdd/tool/internal/exectest"
	"threehillpath.com/marvin-sdd/tool/internal/findings"
)

// fakeRunnerAt returns a FakeRunner whose git rev-parse --git-common-dir
// response resolves worktree.RepoRoot to root/.git — i.e. Cache/Clear will
// anchor at root regardless of the test process's actual CWD.
func fakeRunnerAt(root string) *exectest.FakeRunner {
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(filepath.Join(root, ".git") + "\n")})
	return fake
}

// TestCachePath verifies the path formula.
func TestCachePath(t *testing.T) {
	got := findings.CachePath("plan-00002", "review", "phase-1")
	want := ".claude/cache/plan-00002/review/phase-1.json"
	if got != want {
		t.Errorf("CachePath = %q, want %q", got, want)
	}
}

// TestCacheWritesValidJSON verifies that Cache creates the directory tree and
// writes the payload when the input is valid JSON, anchored at the
// RepoRoot-resolved directory rather than the process CWD.
func TestCacheWritesValidJSON(t *testing.T) {
	dir := t.TempDir()
	fake := fakeRunnerAt(dir)

	payload := []byte(`{"findings":[]}`)
	if err := findings.Cache(context.Background(), fake, "plan-00002", "review", "phase-1", payload); err != nil {
		t.Fatalf("Cache returned error: %v", err)
	}

	path := filepath.Join(dir, ".claude", "cache", "plan-00002", "review", "phase-1.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file not created at %q: %v", path, err)
	}
	if string(data) != string(payload) {
		t.Errorf("file content = %q, want %q", data, payload)
	}
}

// TestCacheAnchorsAtRepoRootNotCWD verifies that Cache writes under the
// RepoRoot-resolved directory even when the process's actual CWD is a
// different directory entirely (e.g. a linked worktree) — the prior
// implementation wrote relative to the CWD directly, silently fragmenting
// the cache when invoked from inside a worktree.
func TestCacheAnchorsAtRepoRootNotCWD(t *testing.T) {
	root := t.TempDir()
	cwd := t.TempDir() // a different directory than root, simulating a worktree
	fake := fakeRunnerAt(root)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	payload := []byte(`{"findings":[]}`)
	if err := findings.Cache(context.Background(), fake, "plan-00002", "review", "phase-1", payload); err != nil {
		t.Fatalf("Cache returned error: %v", err)
	}

	wantPath := filepath.Join(root, ".claude", "cache", "plan-00002", "review", "phase-1.json")
	if _, err := os.Stat(wantPath); err != nil {
		t.Errorf("expected cache file at RepoRoot-anchored path %q, got: %v", wantPath, err)
	}

	unwantedPath := filepath.Join(cwd, ".claude", "cache", "plan-00002", "review", "phase-1.json")
	if _, err := os.Stat(unwantedPath); !os.IsNotExist(err) {
		t.Errorf("cache file should not be written relative to CWD %q, but it was", cwd)
	}
}

// TestCacheRejectsInvalidJSON verifies that Cache returns an error for invalid JSON.
func TestCacheRejectsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	fake := fakeRunnerAt(dir)

	if err := findings.Cache(context.Background(), fake, "plan-00002", "review", "bad", []byte(`not json`)); err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// TestClearRemovesPlanDirectory verifies that Clear deletes the plan-scoped directory.
func TestClearRemovesPlanDirectory(t *testing.T) {
	dir := t.TempDir()
	fake := fakeRunnerAt(dir)

	// Pre-create a cache file.
	cacheDir := filepath.Join(dir, ".claude", "cache", "plan-00002", "review")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "phase-1.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := findings.Clear(context.Background(), fake, "plan-00002"); err != nil {
		t.Fatalf("Clear returned error: %v", err)
	}

	planDir := filepath.Join(dir, ".claude", "cache", "plan-00002")
	if _, err := os.Stat(planDir); !os.IsNotExist(err) {
		t.Error("expected plan directory to be removed after Clear")
	}
}

// TestClearIsNoOpWhenAbsent verifies that Clear is a no-op if the directory does not exist.
func TestClearIsNoOpWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	fake := fakeRunnerAt(dir)

	if err := findings.Clear(context.Background(), fake, "PLAN-00099"); err != nil {
		t.Fatalf("Clear returned error for absent dir: %v", err)
	}
}

// TestClearDoesNotAffectSiblings verifies that Clear only removes the target plan's dir.
func TestClearDoesNotAffectSiblings(t *testing.T) {
	dir := t.TempDir()
	fake := fakeRunnerAt(dir)

	// Create dirs for two plans.
	for _, plan := range []string{"plan-00001", "plan-00002"} {
		d := filepath.Join(dir, ".claude", "cache", plan)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(d, "data.json"), []byte(`{}`), 0o644)
	}

	if err := findings.Clear(context.Background(), fake, "plan-00001"); err != nil {
		t.Fatalf("Clear returned error: %v", err)
	}

	// plan-00002 should still exist.
	sibling := filepath.Join(dir, ".claude", "cache", "plan-00002")
	if _, err := os.Stat(sibling); err != nil {
		t.Errorf("sibling plan dir unexpectedly removed: %v", err)
	}

	// plan-00001 should be gone.
	removed := filepath.Join(dir, ".claude", "cache", "plan-00001")
	if _, err := os.Stat(removed); !os.IsNotExist(err) {
		t.Error("plan-00001 dir should be removed")
	}
}
