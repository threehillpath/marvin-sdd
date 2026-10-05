// Package clierr defines the typed CLIError used throughout marvin.
// It lives in its own package so both cli and config can import it without cycles.
package clierr

import "fmt"

// CLIError carries a typed exit code for use by the CLI layer.
// Code 1 = operational error, Code 2 = config missing or malformed,
// Code 3 = a draft or body that does not conform to its schema.
type CLIError struct {
	Code int
	Msg  string
}

func (e *CLIError) Error() string {
	return e.Msg
}

// Config returns a CLIError{Code:2} with a message pointing at /configure-plan-plugin.
func Config(msg string) *CLIError {
	return &CLIError{Code: 2, Msg: msg}
}

// ConfigMissing returns a standard "config not found" CLIError{Code:2}.
func ConfigMissing(startDir string) *CLIError {
	return &CLIError{
		Code: 2,
		Msg: fmt.Sprintf(
			"no plan-workflow config found in %q or any parent directory. "+
				"Run /configure-plan-plugin to create one.",
			startDir,
		),
	}
}

// ConfigBad returns a standard "malformed config" CLIError{Code:2}.
func ConfigBad(path string, cause error) *CLIError {
	return &CLIError{
		Code: 2,
		Msg:  fmt.Sprintf("malformed config at %q: %v", path, cause),
	}
}

// Operational returns a CLIError{Code:1}.
func Operational(msg string) *CLIError {
	return &CLIError{Code: 1, Msg: msg}
}

// NonConforming returns a CLIError{Code:3}: the input was read and checked,
// and it does not conform to its schema. The findings themselves are written
// by the caller; Msg is only the one-line summary.
func NonConforming(msg string) *CLIError {
	return &CLIError{Code: 3, Msg: msg}
}
