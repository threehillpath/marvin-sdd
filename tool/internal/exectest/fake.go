// Package exectest provides a FakeRunner for use in tests across all packages
// that depend on exec.Runner.
package exectest

import (
	"context"
	"fmt"
)

// Call records a single invocation of FakeRunner.Run.
type Call struct {
	Name string
	Args []string
}

// FakeResponse is the canned response for a single matched command.
type FakeResponse struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// FakeRunner records every Run call and returns canned responses in FIFO order.
// Each registered response is consumed once. Unmatched or exhausted calls return an error.
type FakeRunner struct {
	// Calls is the record of all Run invocations, in order.
	Calls []Call
	// responses is the queue of canned responses (consumed FIFO).
	responses []FakeResponse
}

// Enqueue appends a canned response to the queue. Responses are returned in the
// order they are enqueued, one per Run call.
func (f *FakeRunner) Enqueue(resp FakeResponse) {
	f.responses = append(f.responses, resp)
}

// Run records the call and returns the next queued response. It returns an error
// if no response is queued.
func (f *FakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, []byte, int, error) {
	f.Calls = append(f.Calls, Call{Name: name, Args: args})
	if len(f.responses) == 0 {
		return nil, nil, 0, fmt.Errorf("FakeRunner: no response queued for %q (args: %v)", name, args)
	}
	resp := f.responses[0]
	f.responses = f.responses[1:]
	return resp.Stdout, resp.Stderr, resp.ExitCode, nil
}
