package findings_test

import (
	"os"
	"path/filepath"
	"testing"

	"threehillpath.com/claude-plan-workflow/tool/internal/findings"
)

// TestCachePath verifies the path formula.
func TestCachePath(t *testing.T) {
	got := findings.CachePath("plan-00002", "review", "phase-1")
	want := ".claude/cache/plan-00002/review/phase-1.json"
	if got != want {
		t.Errorf("CachePath = %q, want %q", got, want)
	}
}

// TestCacheWritesValidJSON verifies that Cache creates the directory tree and
// writes the payload when the input is valid JSON.
func TestCacheWritesValidJSON(t *testing.T) {
	dir := t.TempDir()
	// Override the cache base to the temp dir.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	payload := []byte(`{"findings":[]}`)
	if err := findings.Cache("plan-00002", "review", "phase-1", payload); err != nil {
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

// TestCacheRejectsInvalidJSON verifies that Cache returns an error for invalid JSON.
func TestCacheRejectsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	if err := findings.Cache("plan-00002", "review", "bad", []byte(`not json`)); err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// TestClearRemovesPlanDirectory verifies that Clear deletes the plan-scoped directory.
func TestClearRemovesPlanDirectory(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Pre-create a cache file.
	cacheDir := filepath.Join(dir, ".claude", "cache", "plan-00002", "review")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "phase-1.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := findings.Clear("plan-00002"); err != nil {
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
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	if err := findings.Clear("PLAN-00099"); err != nil {
		t.Fatalf("Clear returned error for absent dir: %v", err)
	}
}

// TestClearDoesNotAffectSiblings verifies that Clear only removes the target plan's dir.
func TestClearDoesNotAffectSiblings(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Create dirs for two plans.
	for _, plan := range []string{"plan-00001", "plan-00002"} {
		d := filepath.Join(dir, ".claude", "cache", plan)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(d, "data.json"), []byte(`{}`), 0o644)
	}

	if err := findings.Clear("plan-00001"); err != nil {
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
