package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfirm(t *testing.T) {
	for in, want := range map[string]bool{
		"y\n": true, "YES\n": true, " yes \n": true,
		"n\n": false, "\n": false, "": false, "maybe\n": false,
	} {
		var out bytes.Buffer
		if got := Confirm(strings.NewReader(in), &out, "Continue?"); got != want {
			t.Errorf("Confirm(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestLoggerNoColor(t *testing.T) {
	var out bytes.Buffer
	l := &Logger{Out: &out, Err: &out, Color: false}
	l.Success("done %d", 3)
	if got := out.String(); got != "  ✓ done 3\n" {
		t.Errorf("got %q", got)
	}
}
