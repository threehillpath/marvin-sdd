package cli_test

import (
	"os"
	"path/filepath"
	"testing"
)

// withConfigFixture chdirs the test into a temp directory containing a
// minimal .claude/plan-workflow-config.yml (reusing boardJSONConfigFixture
// from board_json_test.go) and restores the original cwd on cleanup.
func withConfigFixture(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "plan-workflow-config.yml"), []byte(boardJSONConfigFixture), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
}
