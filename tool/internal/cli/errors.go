package cli

import (
	"fmt"
	"io"

	"threehillpath.com/marvin-sdd/tool/internal/clierr"
)

// CLIError is re-exported from clierr so callers can import just this package.
type CLIError = clierr.CLIError

// configHint is written to stderr for any Code-2 error so users know how to fix
// a missing or malformed config.
const configHint = "Run /configure-plan-plugin to create or repair the config file."

// RunWithStreams executes fn and returns an exit code derived from the error:
//   - nil → 0
//   - *CLIError{Code:N} → N (message to stderr; if Code==2 the config hint is appended)
//   - any other error → 1
//
// stdout is left empty on error; all diagnostics go to stderr.
func RunWithStreams(stdout, stderr io.Writer, fn func() error) int {
	err := fn()
	if err == nil {
		return 0
	}
	var cliErr *CLIError
	switch e := err.(type) {
	case *CLIError:
		cliErr = e
	default:
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	fmt.Fprintf(stderr, "error: %s\n", cliErr.Msg)
	if cliErr.Code == 2 {
		fmt.Fprintf(stderr, "%s\n", configHint)
	}
	return cliErr.Code
}
