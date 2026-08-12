package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"threehillpath.com/marvin-sdd/tool/internal/cli"
	"threehillpath.com/marvin-sdd/tool/internal/exectest"
)

const configGetFixture = `
repo: threehillpath/claude-plan-workflow
project_number: 4
project_id: PVT_test
status_field_id: PVTSSF_test
worktree_base: .worktrees
statuses:
  in_progress: abc123
`

// TestConfigGetWorktreeBase verifies that 'marvin config get worktree_base'
// returns the configured value rather than falling through to a status lookup.
func TestConfigGetWorktreeBase(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "plan-workflow-config.yml"), []byte(configGetFixture), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"config", "get", "worktree_base"})

	if err := root.Execute(); err != nil {
		t.Fatalf("config get worktree_base returned error: %v\nstderr: %s", err, stderr.String())
	}

	got := strings.TrimSpace(stdout.String())
	if got != ".worktrees" {
		t.Errorf("config get worktree_base = %q, want .worktrees", got)
	}
}
