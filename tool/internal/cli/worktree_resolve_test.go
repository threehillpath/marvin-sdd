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

// TestWorktreeResolvePrintsAbsolutePath verifies that
// `marvin worktree resolve <path>` prints the absolute path resolved via
// worktree.Resolve: the repo-relative input joined against the (faked)
// repo root, with the process's real cwd set to that same root so the
// execution-scope check passes.
func TestWorktreeResolvePrintsAbsolutePath(t *testing.T) {
	root := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
	root, err = os.Getwd() // canonicalize, consistent with worktree.Resolve's own os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(filepath.Join(root, ".git") + "\n")})

	cmd := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	cmd.SetArgs([]string{"worktree", "resolve", filepath.Join(".worktrees", "phase-3")})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("worktree resolve returned error: %v\nstderr: %s", err, stderr.String())
	}

	got := strings.TrimSpace(stdout.String())
	want := filepath.Join(root, ".worktrees", "phase-3")
	if got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

// TestWorktreeResolveAbsolutePathRejected verifies that an already-absolute
// path is rejected rather than trusted verbatim: Resolve never short-
// circuits on caller-supplied absolute input, since that is exactly the
// kind of unverified value that could point outside the repository.
func TestWorktreeResolveAbsolutePathRejected(t *testing.T) {
	var stdout, stderr bytes.Buffer
	fake := &exectest.FakeRunner{}

	cmd := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	cmd.SetArgs([]string{"worktree", "resolve", "/tmp/wt/phase-3"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for an absolute path argument, got nil")
	}
	if len(fake.Calls) != 0 {
		t.Errorf("expected 0 runner calls for a rejected absolute path, got %d: %v", len(fake.Calls), fake.Calls)
	}
}

// TestWorktreeResolveRejectsSiblingWorktree verifies the CLI surfaces the
// execution-scope rejection (not just the bare package function) when
// invoked from inside a worktree, targeting a sibling worktree.
func TestWorktreeResolveRejectsSiblingWorktree(t *testing.T) {
	root := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	root, err = os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	phase1 := filepath.Join(root, ".worktrees", "phase-1")
	if err := os.MkdirAll(phase1, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(phase1); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(filepath.Join(root, ".git") + "\n")})

	cmd := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	cmd.SetArgs([]string{"worktree", "resolve", filepath.Join(".worktrees", "phase-2")})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error resolving a sibling worktree, got nil")
	}
}
