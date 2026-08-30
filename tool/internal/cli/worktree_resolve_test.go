package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"threehillpath.com/marvin-sdd/tool/internal/cli"
	"threehillpath.com/marvin-sdd/tool/internal/exectest"
)

// TestWorktreeResolvePrintsAbsolutePath verifies that
// `marvin worktree resolve <path>` prints the absolute path resolved via
// worktree.Resolve: unchanged for an already-absolute path, joined against
// the fake repoRoot response for a relative one.
func TestWorktreeResolvePrintsAbsolutePath(t *testing.T) {
	var stdout, stderr bytes.Buffer
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("/repo/.git\n")})

	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	root.SetArgs([]string{"worktree", "resolve", ".worktrees/phase-3"})

	if err := root.Execute(); err != nil {
		t.Fatalf("worktree resolve returned error: %v\nstderr: %s", err, stderr.String())
	}

	got := strings.TrimSpace(stdout.String())
	want := "/repo/.worktrees/phase-3"
	if got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

// TestWorktreeResolveAbsolutePathUnchanged verifies that an already-absolute
// path is printed unchanged, with zero runner calls (Resolve short-circuits).
func TestWorktreeResolveAbsolutePathUnchanged(t *testing.T) {
	var stdout, stderr bytes.Buffer
	fake := &exectest.FakeRunner{}

	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	root.SetArgs([]string{"worktree", "resolve", "/tmp/wt/phase-3"})

	if err := root.Execute(); err != nil {
		t.Fatalf("worktree resolve returned error: %v\nstderr: %s", err, stderr.String())
	}

	got := strings.TrimSpace(stdout.String())
	if got != "/tmp/wt/phase-3" {
		t.Errorf("stdout = %q, want %q", got, "/tmp/wt/phase-3")
	}
	if len(fake.Calls) != 0 {
		t.Errorf("expected 0 runner calls for an absolute path, got %d: %v", len(fake.Calls), fake.Calls)
	}
}
