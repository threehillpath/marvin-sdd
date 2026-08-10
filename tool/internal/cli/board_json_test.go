package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"threehillpath.com/claude-plan-workflow/tool/internal/cli"
	"threehillpath.com/claude-plan-workflow/tool/internal/exectest"
)

const boardJSONConfigFixture = `
repo: threehillpath/claude-plan-workflow
project_number: 4
project_id: PVT_test
status_field_id: PVTSSF_test
worktree_base: .worktrees
statuses:
  in_progress: abc123
`

// boardListPayload is a canned gh project item-list response used to assert
// board list --json fidelity against an injected FakeRunner.
const boardListPayload = `{"items":[{"id":"PVTI_1","title":"Phase 1","status":"In Progress","content":{"number":10,"title":"Phase 1","url":"https://github.com/threehillpath/claude-plan-workflow/issues/10"}}]}`

// wantBoardListJSON is the exact JSON `board list --json` produced before the
// CLI output-layer injection seam (Phase 3, PLAN-00041): a JSON array of
// board.BoardItem, 2-space indented, with a trailing newline.
const wantBoardListJSON = `[
  {
    "number": 10,
    "title": "Phase 1",
    "status": "In Progress",
    "url": "https://github.com/threehillpath/claude-plan-workflow/issues/10"
  }
]
`

// TestBoardListJSONFidelity is this phase's TDD entry point: `board list --json`,
// run through NewRootCmd against an injected FakeRunner with a canned
// ProjectItemList response, must produce output byte-identical to a fixture
// captured from the pre-refactor JSON output — proving the new stdin/runner
// injection seam does not alter observable JSON output.
func TestBoardListJSONFidelity(t *testing.T) {
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
	defer os.Chdir(orig)

	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(boardListPayload)})

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	root.SetArgs([]string{"board", "list", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("board list --json returned error: %v\nstderr: %s", err, stderr.String())
	}

	if stdout.String() != wantBoardListJSON {
		t.Errorf("board list --json output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), wantBoardListJSON)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 gh call, got %d: %v", len(fake.Calls), fake.Calls)
	}
}
