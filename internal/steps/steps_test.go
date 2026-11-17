package steps

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/shailesh097/tweed/internal/config"
	"github.com/shailesh097/tweed/internal/runner"
	"github.com/shailesh097/tweed/internal/ui"
)

var ctx = context.Background()

func TestDNFPackagesOnlyInstallsMissing(t *testing.T) {
	f := &runner.Fake{Responses: map[string]runner.FakeResponse{
		"rpm -q git fish btop": {
			Out: "git-2.47.0-1.fc44.x86_64\npackage fish is not installed\nbtop-1.4.0-1.fc44.x86_64\n",
			Err: errors.New("exit status 1"),
		},
	}}
	s := &dnfPackagesStep{pkgs: []string{"git", "fish", "btop"}}
	done, err := s.Check(ctx, f)
	if err != nil || done {
		t.Fatalf("Check = %v, %v; want false, nil", done, err)
	}
	if err := s.Apply(ctx, f); err != nil {
		t.Fatal(err)
	}
	want := []string{"sudo dnf install -y --skip-unavailable fish"}
	if !reflect.DeepEqual(f.Ran, want) {
		t.Errorf("Ran = %v, want %v", f.Ran, want)
	}
}

func TestFlatpakAlreadyInstalled(t *testing.T) {
	f := &runner.Fake{Responses: map[string]runner.FakeResponse{
		"flatpak list --app --columns=application": {Out: "com.spotify.Client\nmd.obsidian.Obsidian\n"},
	}}
	done, err := (&flatpakStep{apps: []string{"md.obsidian.Obsidian"}}).Check(ctx, f)
	if err != nil || !done {
		t.Fatalf("Check = %v, %v; want true, nil", done, err)
	}
}

func TestNormalizeGVariant(t *testing.T) {
	for in, want := range map[string]string{
		"uint32 960\n":    "960",
		"'nothing'\n":     "nothing",
		"nothing":         "nothing",
		"@as []":          "[]",
		"'appmenu:close'": "appmenu:close",
	} {
		if got := NormalizeGVariant(in); got != want {
			t.Errorf("NormalizeGVariant(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGSettingIdempotent(t *testing.T) {
	g := config.GSetting{Schema: "org.gnome.desktop.session", Key: "idle-delay", Value: "960"}
	f := &runner.Fake{Responses: map[string]runner.FakeResponse{
		"gsettings get org.gnome.desktop.session idle-delay": {Out: "uint32 960\n"},
	}}
	done, _ := (&gsettingStep{g: g}).Check(ctx, f)
	if !done {
		t.Error("expected idle-delay 960 to be detected as already set")
	}
}

func TestDNFConfig(t *testing.T) {
	dir := t.TempDir()
	DNFConfPath = filepath.Join(dir, "dnf.conf")
	t.Cleanup(func() { DNFConfPath = "/etc/dnf/dnf.conf" })
	if err := os.WriteFile(DNFConfPath, []byte("[main]\ngpgcheck=True\nmax_parallel_downloads=3\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Existing key with a different value → sed replace.
	f := &runner.Fake{}
	s := &dnfConfigStep{key: "max_parallel_downloads", value: "10"}
	if done, _ := s.Check(ctx, f); done {
		t.Fatal("should not be done")
	}
	_ = s.Apply(ctx, f)
	// Missing key → append.
	s2 := &dnfConfigStep{key: "defaultyes", value: "True"}
	_ = s2.Apply(ctx, f)
	want := []string{
		"sudo sed -i 's|^max_parallel_downloads=.*|max_parallel_downloads=10|' " + DNFConfPath,
		"sudo tee -a " + DNFConfPath + "  <<< defaultyes=True",
	}
	if !reflect.DeepEqual(f.Ran, want) {
		t.Errorf("Ran =\n%v\nwant\n%v", f.Ran, want)
	}
	// Matching value (case-insensitive) → done.
	if done, _ := (&dnfConfigStep{key: "gpgcheck", value: "true"}).Check(ctx, f); !done {
		t.Error("gpgcheck=True should satisfy gpgcheck=true")
	}
}

func TestCoprCheckUsesRepoFile(t *testing.T) {
	dir := t.TempDir()
	ReposDir = dir
	t.Cleanup(func() { ReposDir = "/etc/yum.repos.d" })
	s := &coprStep{project: "atim/starship"}
	if done, _ := s.Check(ctx, &runner.Fake{}); done {
		t.Fatal("should not be done before repo file exists")
	}
	_ = os.WriteFile(filepath.Join(dir, "_copr:copr.fedorainfracloud.org:atim:starship.repo"), nil, 0o644)
	if done, _ := s.Check(ctx, &runner.Fake{}); !done {
		t.Fatal("should be done once repo file exists")
	}
}

func TestExecuteStopsOnFailureUnlessKeepGoing(t *testing.T) {
	cfg := &config.Config{Modules: map[string]config.Module{
		"m": {Commands: []config.Command{
			{Name: "one", Run: "true"},
			{Name: "two", Run: "true"},
		}},
	}}
	var buf bytes.Buffer
	log := &ui.Logger{Out: &buf, Err: &buf}

	failing := &failRunner{}
	sum, err := Execute(ctx, cfg, []string{"m"}, failing, log, Options{})
	if err == nil || sum.Failed != 1 || failing.runs != 1 {
		t.Fatalf("stop mode: sum=%+v err=%v runs=%d", sum, err, failing.runs)
	}
	failing = &failRunner{}
	sum, err = Execute(ctx, cfg, []string{"m"}, failing, log, Options{KeepGoing: true})
	if err == nil || sum.Failed != 2 || failing.runs != 2 {
		t.Fatalf("keep-going mode: sum=%+v err=%v runs=%d", sum, err, failing.runs)
	}
}

type failRunner struct{ runs int }

func (f *failRunner) DryRun() bool { return false }
func (f *failRunner) Query(context.Context, runner.Cmd) (string, error) {
	return "", errors.New("nope")
}
func (f *failRunner) Run(context.Context, runner.Cmd) error {
	f.runs++
	return errors.New("boom")
}
