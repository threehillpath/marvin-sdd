package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"threehillpath.com/claude-plan-workflow/tool/internal/cli"
	"threehillpath.com/claude-plan-workflow/tool/internal/exectest"
)

// TestParsePhaseListReadsInjectedStdin verifies that `parse phase-list` reads
// from the stdin injected into NewRootCmd, not the process's real os.Stdin.
// Real os.Stdin is temporarily redirected to a pipe carrying a DIFFERENT
// "Phases created:" comment (with issue #99) so the test can prove which
// source the command actually read from: if it read os.Stdin instead of the
// injected reader, the assertion on issue #12 below would fail.
func TestParsePhaseListReadsInjectedStdin(t *testing.T) {
	origStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = r
	defer func() {
		os.Stdin = origStdin
		r.Close()
	}()
	go func() {
		defer w.Close()
		w.WriteString("Phases created:\n- #99 [PLAN-00099-1] Decoy\n")
	}()

	injected := strings.NewReader("Phases created:\n- #12 [PLAN-00042-1] Real\n")

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(injected, &stdout, &stderr, &exectest.FakeRunner{})
	root.SetArgs([]string{"parse", "phase-list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("parse phase-list returned error: %v\nstderr: %s", err, stderr.String())
	}

	var out struct {
		Found  bool  `json:"found"`
		Issues []int `json:"issues"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal stdout: %v\nstdout: %s", err, stdout.String())
	}

	if !out.Found || len(out.Issues) != 1 || out.Issues[0] != 12 {
		t.Errorf("issues = %v (found=%v), want [12] (found=true) from the injected reader, not os.Stdin", out.Issues, out.Found)
	}
}
