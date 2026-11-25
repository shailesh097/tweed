// Package steps turns config modules into idempotent setup steps.
//
// Each Step can Check whether it is already done and Apply itself. Running
// tweed twice therefore does nothing the second time.
package steps

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/shailesh097/tweed/internal/config"
	"github.com/shailesh097/tweed/internal/runner"
)

// Step is one idempotent unit of work.
type Step interface {
	// Describe returns a short human-readable description.
	Describe() string
	// Check reports whether the step is already satisfied. It must not change
	// the system. An error means "could not tell"; the step will be applied.
	Check(ctx context.Context, r runner.Runner) (done bool, err error)
	// Apply makes the change. It is only called when Check returned false.
	Apply(ctx context.Context, r runner.Runner) error
}

// Paths that steps read. Variables so tests can point them at temp dirs.
var (
	DNFConfPath = "/etc/dnf/dnf.conf"
	ReposDir    = "/etc/yum.repos.d"
)

// FromModule builds the ordered steps for a module.
func FromModule(m config.Module) []Step {
	var out []Step
	for _, k := range sortedKeys(m.DNFConfig) {
		out = append(out, &dnfConfigStep{key: k, value: m.DNFConfig[k]})
	}
	for _, r := range m.Repos {
		out = append(out, repoStep(r))
	}
	if len(m.DNF) > 0 {
		out = append(out, &dnfPackagesStep{pkgs: m.DNF})
	}
	if len(m.Flatpak) > 0 {
		out = append(out, &flatpakStep{apps: m.Flatpak})
	}
	for _, g := range m.GSettings {
		out = append(out, &gsettingStep{g: g})
	}
	for _, c := range m.Commands {
		out = append(out, &commandStep{c: c})
	}
	return out
}

// ---- dnf.conf ---------------------------------------------------------------

type dnfConfigStep struct{ key, value string }

var validDNFKey = regexp.MustCompile(`^[a-z_]+$`)

func (s *dnfConfigStep) Describe() string {
	return fmt.Sprintf("dnf.conf: %s=%s", s.key, s.value)
}

func (s *dnfConfigStep) current() (val string, present bool, err error) {
	data, err := os.ReadFile(DNFConfPath)
	if err != nil {
		return "", false, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
		if ok && strings.TrimSpace(k) == s.key {
			val, present = strings.TrimSpace(v), true
		}
	}
	return val, present, nil
}

func (s *dnfConfigStep) Check(context.Context, runner.Runner) (bool, error) {
	val, present, err := s.current()
	if err != nil {
		return false, err
	}
	return present && strings.EqualFold(val, s.value), nil
}

func (s *dnfConfigStep) Apply(ctx context.Context, r runner.Runner) error {
	if !validDNFKey.MatchString(s.key) {
		return fmt.Errorf("invalid dnf.conf key %q", s.key)
	}
	line := s.key + "=" + s.value
	if _, present, _ := s.current(); present {
		return r.Run(ctx, runner.Cmd{Name: "sed", Args: []string{"-i", "s|^" + s.key + "=.*|" + line + "|", DNFConfPath}, Sudo: true})
	}
	return r.Run(ctx, runner.Cmd{Name: "tee", Args: []string{"-a", DNFConfPath}, Sudo: true, Stdin: line + "\n"})
}

// ---- repositories -----------------------------------------------------------

func repoStep(r config.Repo) Step {
	switch r.Kind() {
	case "builtin":
		if r.Builtin == "flathub" {
			return &flathubStep{}
		}
		return &rpmFusionStep{variant: strings.TrimPrefix(r.Builtin, "rpmfusion-")}
	case "copr":
		return &coprStep{project: r.Copr}
	case "repofile":
		return &repoFileStep{url: r.RepoFile}
	default:
		return &inlineRepoStep{r: r}
	}
}

type rpmFusionStep struct{ variant string } // "free" or "nonfree"

func (s *rpmFusionStep) pkg() string      { return "rpmfusion-" + s.variant + "-release" }
func (s *rpmFusionStep) Describe() string { return "repo: RPM Fusion " + s.variant }

func (s *rpmFusionStep) Check(ctx context.Context, r runner.Runner) (bool, error) {
	_, err := r.Query(ctx, runner.Cmd{Name: "rpm", Args: []string{"-q", s.pkg()}})
	return err == nil, nil
}

func (s *rpmFusionStep) Apply(ctx context.Context, r runner.Runner) error {
	ver, err := FedoraVersion(ctx, r)
	if err != nil {
		if !r.DryRun() {
			return err
		}
		ver = "$(rpm -E %fedora)"
	}
	url := fmt.Sprintf("https://download1.rpmfusion.org/%s/fedora/%s-%s.noarch.rpm", s.variant, s.pkg(), ver)
	return r.Run(ctx, runner.Cmd{Name: "dnf", Args: []string{"install", "-y", url}, Sudo: true})
}

// FedoraVersion returns the running Fedora release number, e.g. "44".
func FedoraVersion(ctx context.Context, r runner.Runner) (string, error) {
	out, err := r.Query(ctx, runner.Cmd{Name: "rpm", Args: []string{"-E", "%fedora"}})
	if err != nil {
		return "", fmt.Errorf("detecting Fedora version: %w", err)
	}
	return strings.TrimSpace(out), nil
}

type flathubStep struct{}

const flathubURL = "https://dl.flathub.org/repo/flathub.flatpakrepo"

func (s *flathubStep) Describe() string { return "repo: Flathub" }

func (s *flathubStep) Check(ctx context.Context, r runner.Runner) (bool, error) {
	out, err := r.Query(ctx, runner.Cmd{Name: "flatpak", Args: []string{"remotes", "--columns=name"}})
	if err != nil {
		return false, err
	}
	return hasLine(out, "flathub"), nil
}

func (s *flathubStep) Apply(ctx context.Context, r runner.Runner) error {
	return r.Run(ctx, runner.Cmd{Name: "flatpak", Args: []string{"remote-add", "--if-not-exists", "flathub", flathubURL}})
}

type coprStep struct{ project string } // "owner/project"

func (s *coprStep) Describe() string { return "repo: COPR " + s.project }

func (s *coprStep) Check(context.Context, runner.Runner) (bool, error) {
	owner, proj, _ := strings.Cut(s.project, "/")
	name := fmt.Sprintf("_copr:copr.fedorainfracloud.org:%s:%s.repo", owner, proj)
	return fileExists(filepath.Join(ReposDir, name)), nil
}

func (s *coprStep) Apply(ctx context.Context, r runner.Runner) error {
	return r.Run(ctx, runner.Cmd{Name: "dnf", Args: []string{"copr", "enable", "-y", s.project}, Sudo: true})
}

type repoFileStep struct{ url string }

func (s *repoFileStep) Describe() string { return "repo: " + path.Base(s.url) }

func (s *repoFileStep) Check(context.Context, runner.Runner) (bool, error) {
	return fileExists(filepath.Join(ReposDir, path.Base(s.url))), nil
}

func (s *repoFileStep) Apply(ctx context.Context, r runner.Runner) error {
	return r.Run(ctx, runner.Cmd{Name: "dnf", Args: []string{"config-manager", "addrepo", "--from-repofile=" + s.url}, Sudo: true})
}

type inlineRepoStep struct{ r config.Repo }

func (s *inlineRepoStep) file() string     { return filepath.Join(ReposDir, s.r.ID+".repo") }
func (s *inlineRepoStep) Describe() string { return "repo: " + s.r.ID }

func (s *inlineRepoStep) Check(context.Context, runner.Runner) (bool, error) {
	return fileExists(s.file()), nil
}

func (s *inlineRepoStep) Apply(ctx context.Context, r runner.Runner) error {
	name := s.r.Name
	if name == "" {
		name = s.r.ID
	}
	var b strings.Builder
	fmt.Fprintf(&b, "[%s]\nname=%s\nbaseurl=%s\nenabled=1\n", s.r.ID, name, s.r.BaseURL)
	if s.r.GPGKey != "" {
		fmt.Fprintf(&b, "gpgcheck=1\ngpgkey=%s\n", s.r.GPGKey)
		if err := r.Run(ctx, runner.Cmd{Name: "rpm", Args: []string{"--import", s.r.GPGKey}, Sudo: true}); err != nil {
			return err
		}
	}
	return r.Run(ctx, runner.Cmd{Name: "tee", Args: []string{s.file()}, Sudo: true, Stdin: b.String()})
}

// ---- packages ---------------------------------------------------------------

type dnfPackagesStep struct {
	pkgs    []string
	missing []string
}

func (s *dnfPackagesStep) Describe() string {
	return fmt.Sprintf("dnf: %d package(s)", len(s.pkgs))
}

func (s *dnfPackagesStep) Check(ctx context.Context, r runner.Runner) (bool, error) {
	s.missing = s.pkgs
	// --whatprovides also resolves names that are only provided by another
	// package (on Fedora 44 "wget" comes from wget2-wget and "npm" from
	// nodejs22-npm); plain "rpm -q" would report those as missing on every run.
	// rpm exits non-zero if any name is missing, so look at the output.
	out, err := r.Query(ctx, runner.Cmd{Name: "rpm", Args: append([]string{"-q", "--whatprovides"}, s.pkgs...)})
	if err != nil && out == "" {
		return false, err
	}
	s.missing = nil
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if name, ok := strings.CutPrefix(line, "no package provides "); ok {
			s.missing = append(s.missing, name)
		} else if p, ok := strings.CutPrefix(line, "package "); ok {
			if name, ok := strings.CutSuffix(p, " is not installed"); ok {
				s.missing = append(s.missing, name)
			}
		}
	}
	return len(s.missing) == 0, nil
}

func (s *dnfPackagesStep) Apply(ctx context.Context, r runner.Runner) error {
	args := append([]string{"install", "-y", "--skip-unavailable"}, s.missing...)
	return r.Run(ctx, runner.Cmd{Name: "dnf", Args: args, Sudo: true})
}

type flatpakStep struct {
	apps    []string
	missing []string
}

func (s *flatpakStep) Describe() string {
	return fmt.Sprintf("flatpak: %d app(s)", len(s.apps))
}

func (s *flatpakStep) Check(ctx context.Context, r runner.Runner) (bool, error) {
	s.missing = s.apps
	out, err := r.Query(ctx, runner.Cmd{Name: "flatpak", Args: []string{"list", "--app", "--columns=application"}})
	if err != nil {
		return false, err
	}
	s.missing = nil
	for _, a := range s.apps {
		if !hasLine(out, a) {
			s.missing = append(s.missing, a)
		}
	}
	return len(s.missing) == 0, nil
}

func (s *flatpakStep) Apply(ctx context.Context, r runner.Runner) error {
	args := append([]string{"install", "-y", "--noninteractive", "flathub"}, s.missing...)
	return r.Run(ctx, runner.Cmd{Name: "flatpak", Args: args})
}

// ---- gsettings --------------------------------------------------------------

type gsettingStep struct{ g config.GSetting }

func (s *gsettingStep) Describe() string {
	return fmt.Sprintf("gsettings: %s %s = %s", s.g.Schema, s.g.Key, s.g.Value)
}

func (s *gsettingStep) Check(ctx context.Context, r runner.Runner) (bool, error) {
	out, err := r.Query(ctx, runner.Cmd{Name: "gsettings", Args: []string{"get", s.g.Schema, s.g.Key}})
	if err != nil {
		return false, err
	}
	return NormalizeGVariant(out) == NormalizeGVariant(s.g.Value), nil
}

func (s *gsettingStep) Apply(ctx context.Context, r runner.Runner) error {
	return r.Run(ctx, runner.Cmd{Name: "gsettings", Args: []string{"set", s.g.Schema, s.g.Key, s.g.Value}})
}

var gvariantTypePrefix = regexp.MustCompile(`^(u?int(16|32|64)|byte|double|@[a-z]+)\s+`)

// NormalizeGVariant makes `gsettings get` output comparable with config values:
// "uint32 960" → "960", "'nothing'" → "nothing".
func NormalizeGVariant(s string) string {
	s = strings.TrimSpace(s)
	s = gvariantTypePrefix.ReplaceAllString(s, "")
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		s = s[1 : len(s)-1]
	}
	return s
}

// ---- arbitrary commands -----------------------------------------------------

type commandStep struct{ c config.Command }

func (s *commandStep) Describe() string { return "command: " + s.c.Name }

func (s *commandStep) Check(ctx context.Context, r runner.Runner) (bool, error) {
	if s.c.Unless == "" {
		return false, nil
	}
	_, err := r.Query(ctx, runner.Cmd{Name: "sh", Args: []string{"-c", s.c.Unless}})
	return err == nil, nil
}

func (s *commandStep) Apply(ctx context.Context, r runner.Runner) error {
	return r.Run(ctx, runner.Cmd{Name: "sh", Args: []string{"-c", s.c.Run}, Sudo: s.c.Sudo})
}

// ---- helpers ----------------------------------------------------------------

func hasLine(out, want string) bool {
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == want {
			return true
		}
	}
	return false
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
