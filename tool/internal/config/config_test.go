package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"threehillpath.com/claude-plan-workflow/tool/internal/config"
)

const yamlFixture = `
repo: threehillpath/claude-plan-workflow
project_number: 4
project_id: PVT_kwDOAY3Pus4BYn3N
status_field_id: PVTSSF_lADOAY3Pus4BYn3NzhTsYTU
worktree_base: .worktrees
statuses:
  backlog: f75ad846
  ready: "n/a"
  in_progress: 47fc9ee4
  in_review: "n/a"
  done: "98236657"
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
| Project ID | ` + "`PVT_kwDOAY3Pus4BYn3N`" + ` |
| Status field ID | ` + "`PVTSSF_lADOAY3Pus4BYn3NzhTsYTU`" + ` |
| Worktree base | ` + "`.worktrees`" + ` |
| "Backlog" option ID | ` + "`f75ad846`" + ` |
| "Ready" option ID | ` + "`n/a`" + ` |
| "In Progress" option ID | ` + "`47fc9ee4`" + ` |
| "In Review" option ID | ` + "`n/a`" + ` |
| "Done" option ID | ` + "`98236657`" + ` |

## Test Commands

| Scope | Command (run from repo root) |
|---|---|
| Backend | ` + "`go test ./...`" + ` |
| Frontend | ` + "`none`" + ` |
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
	if id != "47fc9ee4" {
		t.Errorf("StatusOptionID(in_progress) = %q, want 47fc9ee4", id)
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
	if id != "47fc9ee4" {
		t.Errorf("StatusOptionID(in_progress) = %q, want 47fc9ee4", id)
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

func TestWorktreeBaseMissingErrors(t *testing.T) {
	// YAML without worktree_base → error, not a silent default.
	noWorktreeBase := `
repo: threehillpath/claude-plan-workflow
project_number: 4
project_id: PVT_test
status_field_id: PVTSSF_test
statuses:
  in_progress: abc123
`
	dir := t.TempDir()
	writeFixture(t, filepath.Join(dir, ".claude"), "plan-workflow-config.yml", noWorktreeBase)

	_, err := config.Load(dir)
	if err == nil {
		t.Fatal("expected error for missing worktree_base, got nil")
	}
}

func TestWorktreeBaseCustom(t *testing.T) {
	// YAML with worktree_base set → uses that value.
	custom := `
repo: threehillpath/claude-plan-workflow
project_number: 4
project_id: PVT_test
status_field_id: PVTSSF_test
worktree_base: custom/worktrees
statuses:
  in_progress: abc123
`
	dir := t.TempDir()
	writeFixture(t, filepath.Join(dir, ".claude"), "plan-workflow-config.yml", custom)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.WorktreeBase != "custom/worktrees" {
		t.Errorf("WorktreeBase = %q, want custom/worktrees", cfg.WorktreeBase)
	}
}

func TestLoadLegacyMarkdownTestCommands(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, filepath.Join(dir, ".claude"), "plan-workflow-config.md", legacyMarkdownFixture)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got := cfg.TestCommands["backend"]; got != "go test ./..." {
		t.Errorf("TestCommands[backend] = %q, want %q", got, "go test ./...")
	}
	if got := cfg.TestCommands["frontend"]; got != "none" {
		t.Errorf("TestCommands[frontend] = %q, want %q", got, "none")
	}
}

func TestWorktreeBaseLegacyMissingErrors(t *testing.T) {
	// Legacy markdown without Worktree base row → error, not a silent default.
	noWorktreeBase := `# Skill Set Configuration

| Setting | Value |
|---|---|
| GitHub repo | ` + "`threehillpath/claude-plan-workflow`" + ` |
| Project number | ` + "`4`" + ` |
| Project ID | ` + "`PVT_test`" + ` |
| Status field ID | ` + "`PVTSSF_test`" + ` |
`
	dir := t.TempDir()
	writeFixture(t, filepath.Join(dir, ".claude"), "plan-workflow-config.md", noWorktreeBase)

	_, err := config.Load(dir)
	if err == nil {
		t.Fatal("expected error for missing Worktree base, got nil")
	}
}

func TestStatusOptionIDHyphenNormalization(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, filepath.Join(dir, ".claude"), "plan-workflow-config.yml", yamlFixture)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"in-progress", "in progress", "in_progress"} {
		id, present, err := cfg.StatusOptionID(name)
		if err != nil {
			t.Fatalf("StatusOptionID(%q) error: %v", name, err)
		}
		if !present {
			t.Errorf("StatusOptionID(%q): expected present=true", name)
		}
		if id != "47fc9ee4" {
			t.Errorf("StatusOptionID(%q) = %q, want 47fc9ee4", name, id)
		}
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
