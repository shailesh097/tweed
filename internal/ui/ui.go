// Package ui provides tweed's terminal output: colored log lines and prompts.
package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
)

// Logger writes human-readable progress output.
type Logger struct {
	Out   io.Writer
	Err   io.Writer
	Color bool
}

// New returns a Logger for stdout/stderr. Color is disabled when NO_COLOR is
// set (https://no-color.org) or stdout is not a terminal.
func New() *Logger {
	return &Logger{Out: os.Stdout, Err: os.Stderr, Color: colorEnabled(os.Stdout)}
}

func (l *Logger) paint(color, s string) string {
	if !l.Color {
		return s
	}
	return color + s + reset
}

// Header prints a section title, e.g. a module name.
func (l *Logger) Header(format string, a ...any) {
	fmt.Fprintln(l.Out, l.paint(bold+cyan, "==> "+fmt.Sprintf(format, a...)))
}

// Info prints a normal progress line.
func (l *Logger) Info(format string, a ...any) {
	fmt.Fprintln(l.Out, "  "+fmt.Sprintf(format, a...))
}

// Success prints a completed action.
func (l *Logger) Success(format string, a ...any) {
	fmt.Fprintln(l.Out, l.paint(green, "  ✓ "+fmt.Sprintf(format, a...)))
}

// Skip prints something that was already done.
func (l *Logger) Skip(format string, a ...any) {
	fmt.Fprintln(l.Out, l.paint(dim, "  • "+fmt.Sprintf(format, a...)))
}

// Warn prints a non-fatal problem.
func (l *Logger) Warn(format string, a ...any) {
	fmt.Fprintln(l.Out, l.paint(yellow, "  ! "+fmt.Sprintf(format, a...)))
}

// Error prints a failure to stderr.
func (l *Logger) Error(format string, a ...any) {
	fmt.Fprintln(l.Err, l.paint(red, "  ✗ "+fmt.Sprintf(format, a...)))
}

// Command prints a command that is about to run (or would run in dry-run).
func (l *Logger) Command(dry bool, cmd string) {
	prefix := "  $ "
	if dry {
		prefix = "  [dry-run] $ "
	}
	fmt.Fprintln(l.Out, l.paint(dim, prefix+cmd))
}

// Confirm asks a yes/no question on in; the default answer is no.
func Confirm(in io.Reader, out io.Writer, question string) bool {
	fmt.Fprintf(out, "%s [y/N]: ", question)
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && line == "" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	}
	return false
}

func colorEnabled(f *os.File) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
