package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfirm(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"y\n", true}, {"YES\n", true}, {" yes \n", true}, // surrounding spaces are trimmed
		{"n\n", false}, {"\n", false}, {"", false}, {"maybe\n", false},
	}
	for _, tt := range tests {
		var out bytes.Buffer
		if got := Confirm(strings.NewReader(tt.in), &out, "Continue?"); got != tt.want {
			t.Errorf("Confirm(%q) = %v, want %v", tt.in, got, tt.want)
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
