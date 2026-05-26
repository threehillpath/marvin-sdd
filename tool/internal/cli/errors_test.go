package cli_test

import (
	"bytes"
	"testing"

	"threehillpath.com/claude-plan-workflow/tool/internal/cli"
)

// TestCLIErrorExitCode verifies that when the run helper encounters a CLIError{Code:2},
// it returns exit code 2, writes the /configure-plan-plugin hint to stderr, and leaves stdout empty.
func TestCLIErrorExitCode(t *testing.T) {
	var stdout, stderr bytes.Buffer

	exitCode := cli.RunWithStreams(
		&stdout,
		&stderr,
		func() error {
			return &cli.CLIError{Code: 2, Msg: "config not found"}
		},
	)

	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Errorf("expected empty stdout, got %q", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Error("expected non-empty stderr")
	}
	const hint = "/configure-plan-plugin"
	if !bytes.Contains(stderr.Bytes(), []byte(hint)) {
		t.Errorf("expected stderr to contain %q, got %q", hint, stderr.String())
	}
}

// TestCLIErrorCode1 verifies operational errors use exit code 1.
func TestCLIErrorCode1(t *testing.T) {
	var stdout, stderr bytes.Buffer

	exitCode := cli.RunWithStreams(
		&stdout,
		&stderr,
		func() error {
			return &cli.CLIError{Code: 1, Msg: "something went wrong"}
		},
	)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Errorf("expected empty stdout, got %q", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Error("expected non-empty stderr")
	}
}

// TestNilErrorExitCode0 verifies a nil error → exit code 0.
func TestNilErrorExitCode0(t *testing.T) {
	var stdout, stderr bytes.Buffer

	exitCode := cli.RunWithStreams(
		&stdout,
		&stderr,
		func() error { return nil },
	)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}
