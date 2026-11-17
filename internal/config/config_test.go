package config

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

const validYAML = `
profiles:
  workstation: [base, desktop]
  server: [base]
modules:
  base:
    dnf_config:
      defaultyes: "True"
    repos:
      - builtin: flathub
      - copr: atim/starship
    dnf: [git, fish]
  desktop:
    flatpak: [com.spotify.Client]
    gsettings:
      - schema: org.gnome.desktop.wm.preferences
        key: button-layout
        value: "appmenu:minimize,maximize,close"
`

func TestParseValid(t *testing.T) {
	c, err := Parse([]byte(validYAML))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := c.ProfileNames(); !reflect.DeepEqual(got, []string{"server", "workstation"}) {
		t.Errorf("ProfileNames = %v", got)
	}
	if got := c.Modules["base"].Repos[1].Kind(); got != "copr" {
		t.Errorf("repo kind = %q, want copr", got)
	}
}

func TestParseRejectsUnknownKeys(t *testing.T) {
	_, err := Parse([]byte("modules:\n  base:\n    dnff: [git]\n"))
	if err == nil || !strings.Contains(err.Error(), "dnff") {
		t.Fatalf("expected unknown-field error mentioning dnff, got %v", err)
	}
}

func TestValidateCollectsAllErrors(t *testing.T) {
	bad := `
profiles:
  ws: [base, missing]
  empty: []
modules:
  base:
    repos:
      - builtin: nope
      - copr: a/b
        repofile: https://x/y.repo
      - id: code
    gsettings:
      - key: only-key
    commands:
      - name: no-run
`
	_, err := Parse([]byte(bad))
	if err == nil {
		t.Fatal("expected errors")
	}
	for _, want := range []string{
		`unknown module "missing"`,
		`profile "empty" is empty`,
		`unknown builtin "nope"`,
		`set exactly one of`,
		`inline repos need both id and baseurl`,
		`schema and key are required`,
		`name and run are required`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error missing %q\nfull error:\n%v", want, err)
		}
	}
}

func TestResolve(t *testing.T) {
	c, err := Parse([]byte(validYAML))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		profile string
		modules []string
		want    []string
		wantErr string
	}{
		{"profile only", "workstation", nil, []string{"base", "desktop"}, ""},
		{"modules only", "", []string{"desktop"}, []string{"desktop"}, ""},
		{"dedupe", "server", []string{"base", "desktop"}, []string{"base", "desktop"}, ""},
		{"nothing", "", nil, nil, "nothing selected"},
		{"bad profile", "laptop", nil, nil, `unknown profile "laptop"`},
		{"bad module", "", []string{"x"}, nil, `unknown module "x"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.Resolve(tt.profile, tt.modules)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestShippedConfig makes sure the example config in the repo always parses.
func TestShippedConfig(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(thisFile), "..", "..", "configs", "tweed.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Skip("configs/tweed.yaml not found")
	}
	if _, err := Load(path); err != nil {
		t.Fatalf("configs/tweed.yaml is invalid: %v", err)
	}
}
