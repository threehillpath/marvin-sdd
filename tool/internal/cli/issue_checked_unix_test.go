//go:build unix

package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"threehillpath.com/marvin-sdd/tool/internal/exectest"
)

// TestIssueCreateBodyFileIsReadOnce verifies create --template --body-file
// sends the bytes it checked rather than reading the file again. The body
// file is a FIFO that yields its content once; a second read blocks, so a
// re-read shows up as a timeout.
func TestIssueCreateBodyFileIsReadOnce(t *testing.T) {
	withConfigFixture(t)
	fifo := filepath.Join(t.TempDir(), "body.fifo")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Skipf("cannot create a FIFO: %v", err)
	}
	go func() {
		f, err := os.OpenFile(fifo, os.O_WRONLY, 0)
		if err != nil {
			return
		}
		f.WriteString(conformingPhaseBody)
		f.Close()
	}()

	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("https://github.com/threehillpath/marvin-sdd/issues/56\n")})
	done := make(chan error, 1)
	go func() {
		_, _, err := runIssue(fake, "issue", "create", "--template", "impl-phase", "--body-file", fifo, "--title", "[PLAN-00112-5] Phase title")
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("create returned %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("create read the body file a second time (blocked on the FIFO)")
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("want 1 gh call, got %v", fake.Calls)
	}
	want := []string{"issue", "create", "--repo", "threehillpath/marvin-sdd", "--title", "[PLAN-00112-5] Phase title", "--body", conformingPhaseBody}
	if got := fake.Calls[0].Args; strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Errorf("args = %q\nwant   %q", got, want)
	}
}
