package runner

import (
	"context"
	"testing"
)

func TestCmdString(t *testing.T) {
	tests := []struct {
		cmd  Cmd
		want string
	}{
		{Cmd{Name: "dnf", Args: []string{"install", "-y", "git"}, Sudo: true}, "sudo dnf install -y git"},
		{Cmd{Name: "gsettings", Args: []string{"set", "a.b", "k", "it's"}}, `gsettings set a.b k 'it'\''s'`},
		{Cmd{Name: "tee", Args: []string{"-a", "/etc/x"}, Stdin: "a=1\n"}, "tee -a /etc/x  <<< a=1"},
	}
	for _, tt := range tests {
		if got := tt.cmd.String(); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}

func TestExecDryRunDoesNothing(t *testing.T) {
	var logged []string
	e := &Exec{Dry: true, OnRun: func(c Cmd) { logged = append(logged, c.String()) }}
	// "false" would fail if it actually ran.
	if err := e.Run(context.Background(), Cmd{Name: "false"}); err != nil {
		t.Fatalf("dry run executed the command: %v", err)
	}
	if len(logged) != 1 || logged[0] != "false" {
		t.Errorf("OnRun not called correctly: %v", logged)
	}
}

func TestExecQuery(t *testing.T) {
	if !Has("echo") {
		t.Skip("echo not available")
	}
	out, err := NewExec(true).Query(context.Background(), Cmd{Name: "echo", Args: []string{"hi"}})
	if err != nil || out != "hi\n" {
		t.Fatalf("Query = %q, %v", out, err)
	}
}
