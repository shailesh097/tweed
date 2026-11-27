package runner

import (
	"context"
	"errors"
	"fmt"
)

// Fake is an in-memory Runner for tests. Query answers come from Responses,
// keyed by Cmd.String(); unknown queries fail. Run calls are recorded.
type Fake struct {
	Dry       bool
	Responses map[string]FakeResponse
	Ran       []string
}

// FakeResponse is the canned result of a Query.
type FakeResponse struct {
	Out string
	Err error
}

// ExitError is a fake process exit for FakeResponse.Err, so code under test
// can inspect the status with ExitCode.
type ExitError int

// Error implements error.
func (e ExitError) Error() string { return fmt.Sprintf("exit status %d", int(e)) }

// ExitCode returns the fake exit status.
func (e ExitError) ExitCode() int { return int(e) }

// ErrNoResponse is returned for queries without a canned response.
var ErrNoResponse = errors.New("fake: no response configured")

// DryRun reports the configured Dry value.
func (f *Fake) DryRun() bool { return f.Dry }

// Query returns the canned response for c, or ErrNoResponse.
func (f *Fake) Query(_ context.Context, c Cmd) (string, error) {
	if r, ok := f.Responses[c.String()]; ok {
		return r.Out, r.Err
	}
	return "", ErrNoResponse
}

// Run records c in Ran and always succeeds.
func (f *Fake) Run(_ context.Context, c Cmd) error {
	f.Ran = append(f.Ran, c.String())
	return nil
}
