// Package exec provides the Runner interface and its OS-backed implementation.
// All packages that invoke gh or git depend on Runner, not os/exec directly,
// so they can be tested with a fake implementation injected by the test.
package exec

import (
	"bytes"
	"context"
	"os/exec"
)

// Runner abstracts command execution so callers can be tested without real processes.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (stdout, stderr []byte, exitCode int, err error)
}

// OSRunner wraps os/exec and satisfies Runner with real process execution.
type OSRunner struct{}

// Run executes name with args, capturing stdout and stderr. It does not return
// an error for non-zero exit codes; the exit code is returned separately so
// callers can distinguish failure types.
func (OSRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
	code := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
			runErr = nil // non-zero exit is not a Go error here
		} else {
			return out.Bytes(), errBuf.Bytes(), -1, runErr
		}
	}
	return out.Bytes(), errBuf.Bytes(), code, nil
}
