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

// TestFindingsCacheReadsInjectedStdin verifies that `findings cache` reads
// from the stdin injected into NewRootCmd (via cmd.InOrStdin()), not the
// process's real os.Stdin. Real os.Stdin is temporarily redirected to a pipe
// carrying a DIFFERENT payload so the test can prove which source the
// command actually read from: if it read os.Stdin instead of the injected
// reader, the cached file would contain the decoy payload instead.
func TestFindingsCacheReadsInjectedStdin(t *testing.T) {
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })

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
		w.WriteString(`{"decoy":true}`)
	}()

	injected := strings.NewReader(`{"verdict":"comment"}`)

	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte(filepath.Join(dir, ".git") + "\n")})

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(injected, &stdout, &stderr, fake)
	root.SetArgs([]string{"findings", "cache", "plan-00042", "review", "phase-3"})

	if err := root.Execute(); err != nil {
		t.Fatalf("findings cache returned error: %v\nstderr: %s", err, stderr.String())
	}

	got, err := os.ReadFile(filepath.Join(dir, ".claude", "cache", "plan-00042", "review", "phase-3.json"))
	if err != nil {
		t.Fatalf("reading cached file: %v", err)
	}
	if string(got) != `{"verdict":"comment"}` {
		t.Errorf("cached payload = %q, want %q (from the injected reader, not os.Stdin)", got, `{"verdict":"comment"}`)
	}
}
