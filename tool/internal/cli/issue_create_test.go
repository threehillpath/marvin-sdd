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

// TestIssueCreatePlainText verifies that marvin issue create splits
// --label on commas into multiple gh --label flags and, in plain-text mode,
// prints the issue number on one line then the URL on the next.
func TestIssueCreatePlainText(t *testing.T) {
	withConfigFixture(t)

	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("https://github.com/owner/repo/issues/42\n")})

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	root.SetArgs([]string{"issue", "create", "--title", "New issue", "--body", "the body", "--label", "bug,urgent"})

	if err := root.Execute(); err != nil {
		t.Fatalf("issue create returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := "42\nhttps://github.com/owner/repo/issues/42\n"
	if stdout.String() != want {
		t.Errorf("plain-text output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}

	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(fake.Calls))
	}
	args := fake.Calls[0].Args
	labelCount := 0
	for i, a := range args {
		if a == "--label" && i+1 < len(args) {
			labelCount++
		}
	}
	if labelCount != 2 {
		t.Errorf("expected 2 --label flags (from comma split), got %d in args %v", labelCount, args)
	}
}

// TestIssueCreateJSON verifies that --json prints {"number":<n>,"url":"<url>"}
// pretty-printed via the existing enc.SetIndent("", "  ") convention.
func TestIssueCreateJSON(t *testing.T) {
	withConfigFixture(t)

	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("https://github.com/owner/repo/issues/9\n")})

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	root.SetArgs([]string{"issue", "create", "--title", "T", "--body", "B", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("issue create --json returned error: %v\nstderr: %s", err, stderr.String())
	}

	want := "{\n  \"number\": 9,\n  \"url\": \"https://github.com/owner/repo/issues/9\"\n}\n"
	if stdout.String() != want {
		t.Errorf("--json output mismatch\ngot:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

// TestIssueCreateBodyFile verifies that --body-file reads the file's
// contents and uses them as the issue body.
func TestIssueCreateBodyFile(t *testing.T) {
	withConfigFixture(t)

	dir := t.TempDir()
	bodyPath := filepath.Join(dir, "body.md")
	if err := os.WriteFile(bodyPath, []byte("multi\nparagraph\nbody"), 0o644); err != nil {
		t.Fatal(err)
	}

	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stdout: []byte("https://github.com/owner/repo/issues/5\n")})

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	root.SetArgs([]string{"issue", "create", "--title", "T", "--body-file", bodyPath})

	if err := root.Execute(); err != nil {
		t.Fatalf("issue create --body-file returned error: %v\nstderr: %s", err, stderr.String())
	}

	args := fake.Calls[0].Args
	found := false
	for i, a := range args {
		if a == "--body" && i+1 < len(args) {
			if args[i+1] != "multi\nparagraph\nbody" {
				t.Errorf("--body value = %q, want file contents", args[i+1])
			}
			found = true
		}
	}
	if !found {
		t.Errorf("expected --body flag built from --body-file contents, args = %v", args)
	}
}

// TestIssueCreateBodyAndBodyFileMutuallyExclusive verifies Code 1 when both
// --body and --body-file are set.
func TestIssueCreateBodyAndBodyFileMutuallyExclusive(t *testing.T) {
	withConfigFixture(t)

	dir := t.TempDir()
	bodyPath := filepath.Join(dir, "body.md")
	if err := os.WriteFile(bodyPath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	fake := &exectest.FakeRunner{}
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	root.SetArgs([]string{"issue", "create", "--title", "T", "--body", "B", "--body-file", bodyPath})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when both --body and --body-file are set")
	}
	ce, ok := err.(*cli.CLIError)
	if !ok {
		t.Fatalf("expected *cli.CLIError, got %T: %v", err, err)
	}
	if ce.Code != 1 {
		t.Errorf("Code = %d, want 1", ce.Code)
	}
	if len(fake.Calls) != 0 {
		t.Errorf("expected no gh calls, got %v", fake.Calls)
	}
}

// TestIssueCreateGhFailureSurfacesNonZeroExit verifies that a non-zero exit
// from gh surfaces as a non-zero marvin exit (Code 1) with gh's stderr
// preserved in the error message — never a silent success.
func TestIssueCreateGhFailureSurfacesNonZeroExit(t *testing.T) {
	withConfigFixture(t)

	fake := &exectest.FakeRunner{}
	fake.Enqueue(exectest.FakeResponse{Stderr: []byte("gh: validation failed (HTTP 422)"), ExitCode: 1})

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	root.SetArgs([]string{"issue", "create", "--title", "T", "--body", "B"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error on gh failure, got nil")
	}
	ce, ok := err.(*cli.CLIError)
	if !ok {
		t.Fatalf("expected *cli.CLIError, got %T: %v", err, err)
	}
	if ce.Code != 1 {
		t.Errorf("Code = %d, want 1", ce.Code)
	}
	if !strings.Contains(ce.Msg, "validation failed") {
		t.Errorf("Msg = %q, want it to contain gh's stderr (%q)", ce.Msg, "validation failed")
	}
	if stdout.Len() != 0 {
		t.Errorf("expected empty stdout on failure, got %q", stdout.String())
	}
}

// TestIssueCreateNeitherBodyNorBodyFile verifies Code 1 when neither --body
// nor --body-file is set.
func TestIssueCreateNeitherBodyNorBodyFile(t *testing.T) {
	withConfigFixture(t)

	fake := &exectest.FakeRunner{}
	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd(strings.NewReader(""), &stdout, &stderr, fake)
	root.SetArgs([]string{"issue", "create", "--title", "T"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when neither --body nor --body-file is set")
	}
	ce, ok := err.(*cli.CLIError)
	if !ok {
		t.Fatalf("expected *cli.CLIError, got %T: %v", err, err)
	}
	if ce.Code != 1 {
		t.Errorf("Code = %d, want 1", ce.Code)
	}
	if len(fake.Calls) != 0 {
		t.Errorf("expected no gh calls, got %v", fake.Calls)
	}
}
