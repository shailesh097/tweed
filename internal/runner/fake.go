package runner

import (
	"context"
	"errors"
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

// ErrNoResponse is returned for queries without a canned response.
var ErrNoResponse = errors.New("fake: no response configured")

func (f *Fake) DryRun() bool { return f.Dry }

func (f *Fake) Query(_ context.Context, c Cmd) (string, error) {
	if r, ok := f.Responses[c.String()]; ok {
		return r.Out, r.Err
	}
	return "", ErrNoResponse
}

func (f *Fake) Run(_ context.Context, c Cmd) error {
	f.Ran = append(f.Ran, c.String())
	return nil
}
