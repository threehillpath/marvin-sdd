package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/threehillpath/claude-plan-workflow/tool/internal/config"
)

const yamlFixture = `
repo: threehillpath/claude-plan-workflow
project_number: 4
project_id: PVT_test
status_field_id: PVTSSF_test
statuses:
  backlog: opt-backlog
  ready: "n/a"
  in_progress: opt-in-progress
  in_review: "n/a"
  done: "opt-done"
test_commands:
  backend: none
  frontend: none
`

// legacyMarkdownFixture is the markdown table format used by older configs.
const legacyMarkdownFixture = `# Skill Set Configuration

| Setting | Value |
|---|---|
| GitHub repo | ` + "`threehillpath/claude-plan-workflow`" + ` |
| Project number | ` + "`4`" + ` |
| Project ID | ` + "`PVT_test`" + ` |
| Status field ID | ` + "`PVTSSF_test`" + ` |
| "Backlog" option ID | ` + "`opt-backlog`" + ` |
| "Ready" option ID | ` + "`n/a`" + ` |
| "In Progress" option ID | ` + "`opt-in-progress`" + ` |
| "In Review" option ID | ` + "`n/a`" + ` |
| "Done" option ID | ` + "`opt-done`" + ` |
`

func writeFixture(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadYAML(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, filepath.Join(dir, ".claude"), "plan-workflow-config.yml", yamlFixture)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Repo != "threehillpath/claude-plan-workflow" {
		t.Errorf("Repo = %q, want threehillpath/claude-plan-workflow", cfg.Repo)
	}

	id, present, err := cfg.StatusOptionID("in_progress")
	if err != nil {
		t.Fatalf("StatusOptionID error: %v", err)
	}
	if !present {
		t.Error("expected in_progress to be present")
	}
	if id != "opt-in-progress" {
		t.Errorf("StatusOptionID(in_progress) = %q, want opt-in-progress", id)
	}
}

func TestLoadLegacyMarkdown(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, filepath.Join(dir, ".claude"), "plan-workflow-config.md", legacyMarkdownFixture)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Repo != "threehillpath/claude-plan-workflow" {
		t.Errorf("Repo = %q, want threehillpath/claude-plan-workflow", cfg.Repo)
	}

	id, present, err := cfg.StatusOptionID("in_progress")
	if err != nil {
		t.Fatalf("StatusOptionID error: %v", err)
	}
	if !present {
		t.Error("expected in_progress to be present")
	}
	if id != "opt-in-progress" {
		t.Errorf("StatusOptionID(in_progress) = %q, want opt-in-progress", id)
	}
}

func TestLoadMissingConfig(t *testing.T) {
	dir := t.TempDir()

	_, err := config.Load(dir)
	if err == nil {
		t.Fatal("expected error for missing config, got nil")
	}
}

func TestStatusOptionIDNA(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, filepath.Join(dir, ".claude"), "plan-workflow-config.yml", yamlFixture)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, present, err := cfg.StatusOptionID("ready")
	if err != nil {
		t.Fatalf("StatusOptionID error: %v", err)
	}
	if present {
		t.Error("expected ready (n/a) to be not present")
	}
}

func TestOwner(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, filepath.Join(dir, ".claude"), "plan-workflow-config.yml", yamlFixture)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	if got := cfg.Owner(); got != "threehillpath" {
		t.Errorf("Owner() = %q, want threehillpath", got)
	}
}

func TestCWDWalk(t *testing.T) {
	// Config is in a parent dir; Load is called from a child dir.
	parent := t.TempDir()
	writeFixture(t, filepath.Join(parent, ".claude"), "plan-workflow-config.yml", yamlFixture)

	child := filepath.Join(parent, "subdir", "nested")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(child)
	if err != nil {
		t.Fatalf("Load from child dir failed: %v", err)
	}
	if cfg.Repo != "threehillpath/claude-plan-workflow" {
		t.Errorf("Repo = %q, want threehillpath/claude-plan-workflow", cfg.Repo)
	}
}
