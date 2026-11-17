// Package runner executes external commands, with support for sudo and dry-run.
//
// Every step goes through a Runner, which keeps two promises:
//   - Query never changes the system, so it also runs during --dry-run.
//   - Run changes the system, so during --dry-run it only prints the command.
package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Cmd is a command to execute.
type Cmd struct {
	Name  string
	Args  []string
	Sudo  bool
	Stdin string // optional data written to the command's stdin
}

// String renders the command roughly as a user would type it.
func (c Cmd) String() string {
	parts := []string{}
	if c.Sudo {
		parts = append(parts, "sudo")
	}
	parts = append(parts, c.Name)
	for _, a := range c.Args {
		if a == "" || strings.ContainsAny(a, " \t'\"$`\\|&;<>(){}*?!") {
			a = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
		}
		parts = append(parts, a)
	}
	s := strings.Join(parts, " ")
	if c.Stdin != "" {
		s += "  <<< " + strings.ReplaceAll(strings.TrimSpace(c.Stdin), "\n", `\n`)
	}
	return s
}

// Runner is the interface steps use. Tests replace it with a fake.
type Runner interface {
	// Query runs a read-only command and returns its stdout. It runs even in dry-run.
	Query(ctx context.Context, c Cmd) (string, error)
	// Run runs a command that changes the system, streaming its output.
	Run(ctx context.Context, c Cmd) error
	// DryRun reports whether Run is a no-op.
	DryRun() bool
}

// Exec is the real Runner backed by os/exec.
type Exec struct {
	Dry    bool
	Stdout io.Writer
	Stderr io.Writer
	// OnRun is called before every Run (for logging), if set.
	OnRun func(c Cmd)
}

// NewExec returns an Exec runner writing to the process's stdout/stderr.
func NewExec(dry bool) *Exec {
	return &Exec{Dry: dry, Stdout: os.Stdout, Stderr: os.Stderr}
}

func (e *Exec) DryRun() bool { return e.Dry }

func (e *Exec) Query(ctx context.Context, c Cmd) (string, error) {
	cmd := build(ctx, c)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	if err != nil {
		return out.String(), fmt.Errorf("%s: %w: %s", c, err, strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}

func (e *Exec) Run(ctx context.Context, c Cmd) error {
	if e.OnRun != nil {
		e.OnRun(c)
	}
	if e.Dry {
		return nil
	}
	cmd := build(ctx, c)
	cmd.Stdout, cmd.Stderr = e.Stdout, e.Stderr
	if c.Stdin == "" {
		cmd.Stdin = os.Stdin // allow sudo password prompts
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", c, err)
	}
	return nil
}

func build(ctx context.Context, c Cmd) *exec.Cmd {
	name, args := c.Name, c.Args
	if c.Sudo && os.Geteuid() != 0 {
		name, args = "sudo", append([]string{c.Name}, c.Args...)
	}
	cmd := exec.CommandContext(ctx, name, args...)
	if c.Stdin != "" {
		cmd.Stdin = strings.NewReader(c.Stdin)
	}
	return cmd
}

// Has reports whether a program is available in PATH.
func Has(program string) bool {
	_, err := exec.LookPath(program)
	return err == nil
}
