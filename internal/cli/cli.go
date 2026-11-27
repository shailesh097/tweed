// Package cli implements tweed's command-line interface.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/shailesh097/tweed/configs"
	"github.com/shailesh097/tweed/internal/config"
	"github.com/shailesh097/tweed/internal/runner"
	"github.com/shailesh097/tweed/internal/steps"
	"github.com/shailesh097/tweed/internal/system"
	"github.com/shailesh097/tweed/internal/ui"
)

const usage = `tweed - set up Fedora from a declarative config

Usage:
  tweed <command> [flags]

Commands:
  apply     Apply a profile and/or modules
  list      Show profiles and modules in the config
  doctor    Check that this system can run tweed
  init      Write the default config to ~/.config/tweed/tweed.yaml
  version   Print the version

Run 'tweed <command> -h' for command flags.

Examples:
  tweed apply --profile workstation --dry-run
  tweed apply -m base -m dev
  tweed apply -p workstation --no-upgrade
  tweed list
`

// Run executes the CLI and returns the process exit code.
func Run(args []string, version string) int {
	log := ui.New()
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var err error
	switch cmd, rest := args[0], args[1:]; cmd {
	case "apply":
		err = runApply(ctx, rest, log)
	case "list":
		err = runList(rest, os.Stdout)
	case "doctor":
		err = runDoctor(rest, log)
	case "init":
		err = runInit(rest, log)
	case "version", "--version", "-v":
		fmt.Println("tweed", version)
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", cmd, usage)
		return 2
	}
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if err != nil {
		log.Error("%v", err)
		return 1
	}
	return 0
}

// ---- shared flags -----------------------------------------------------------

// multiFlag collects repeated and/or comma-separated values: -m a -m b,c
type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error {
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			*m = append(*m, p)
		}
	}
	return nil
}

func addConfigFlag(fs *flag.FlagSet) *string {
	p := new(string)
	fs.StringVar(p, "config", "", "path to config file (default: ~/.config/tweed/tweed.yaml, else built-in)")
	fs.StringVar(p, "c", "", "shorthand for --config")
	return p
}

// UserConfigPath is where `tweed init` writes and where tweed looks first.
func UserConfigPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "tweed", "tweed.yaml")
}

// loadConfig resolves the config: explicit path > user config > built-in default.
func loadConfig(path string) (*config.Config, string, error) {
	if path != "" {
		c, err := config.Load(path)
		return c, path, err
	}
	if p := UserConfigPath(); fileExists(p) {
		c, err := config.Load(p)
		return c, p, err
	}
	c, err := config.Parse(configs.Default)
	return c, "built-in default", err
}

// ---- apply ------------------------------------------------------------------

func runApply(ctx context.Context, args []string, log *ui.Logger) error {
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	cfgPath := addConfigFlag(fs)
	var profile string
	var modules multiFlag
	var dry, yes, keepGoing, noUpgrade bool
	fs.StringVar(&profile, "profile", "", "profile to apply")
	fs.StringVar(&profile, "p", "", "shorthand for --profile")
	fs.Var(&modules, "module", "module to apply (repeatable, comma-separated)")
	fs.Var(&modules, "m", "shorthand for --module")
	fs.BoolVar(&dry, "dry-run", false, "show what would change without changing anything")
	fs.BoolVar(&dry, "n", false, "shorthand for --dry-run")
	fs.BoolVar(&yes, "yes", false, "do not ask for confirmation")
	fs.BoolVar(&yes, "y", false, "shorthand for --yes")
	fs.BoolVar(&keepGoing, "keep-going", false, "continue after a failed step")
	fs.BoolVar(&noUpgrade, "no-upgrade", false, "skip the 'dnf upgrade' of modules that set upgrade: true")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected arguments: %v", fs.Args())
	}

	cfg, src, err := loadConfig(*cfgPath)
	if err != nil {
		return err
	}
	selected, err := cfg.Resolve(profile, modules)
	if err != nil {
		return err
	}

	if !dry {
		if os.Geteuid() == 0 {
			return errors.New("run tweed as your normal user; it calls sudo when needed (GNOME settings would otherwise go to root)")
		}
		osr, err := system.ReadOSRelease()
		if err != nil || !osr.IsFedora() {
			return fmt.Errorf("this does not look like Fedora (%s); use --dry-run to preview", osr.Pretty)
		}
	}

	log.Info("config: %s", src)
	log.Info("modules: %s", strings.Join(selected, ", "))
	if !dry && !yes && !ui.Confirm(os.Stdin, os.Stdout, "Apply these changes?") {
		return errors.New("aborted")
	}

	r := runner.NewExec(dry)
	r.OnRun = func(c runner.Cmd) { log.Command(dry, c.String()) }
	sum, err := steps.Execute(ctx, cfg, selected, r, log, steps.Options{KeepGoing: keepGoing, NoUpgrade: noUpgrade})
	if dry {
		log.Header("summary: %d would change, %d already done", sum.Applied, sum.Skipped)
		log.Info("dry run: nothing was changed")
	} else {
		log.Header("summary: %s", sum)
	}
	return err
}

// ---- list -------------------------------------------------------------------

func runList(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	cfgPath := addConfigFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, src, err := loadConfig(*cfgPath)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Config: %s\n\nProfiles:\n", src)
	for _, p := range cfg.ProfileNames() {
		fmt.Fprintf(out, "  %-20s %s\n", p, strings.Join(cfg.Profiles[p], ", "))
	}
	fmt.Fprintln(out, "\nModules:")
	for _, name := range cfg.ModuleNames() {
		m := cfg.Modules[name]
		fmt.Fprintf(out, "  %-20s %s\n", name, m.Description)
		upgrade := ""
		if m.Upgrade {
			upgrade = "upgrade, "
		}
		fmt.Fprintf(out, "  %-20s %d steps (%sdnf: %d, flatpak: %d, repos: %d, gsettings: %d, commands: %d)\n",
			"", len(steps.FromModule(m)), upgrade, len(m.DNF), len(m.Flatpak), len(m.Repos), len(m.GSettings), len(m.Commands))
	}
	return nil
}

// ---- doctor -----------------------------------------------------------------

func runDoctor(args []string, log *ui.Logger) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	cfgPath := addConfigFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	problems := 0
	log.Header("system")
	if osr, err := system.ReadOSRelease(); err != nil {
		log.Error("cannot read /etc/os-release: %v", err)
		problems++
	} else if !osr.IsFedora() {
		log.Error("not Fedora: %s", osr.Pretty)
		problems++
	} else {
		log.Success("%s", osr.Pretty)
	}
	if os.Geteuid() == 0 {
		log.Warn("running as root; run tweed as your normal user")
	}

	log.Header("tools")
	for _, tool := range []string{"sudo", "dnf", "rpm", "flatpak", "gsettings", "sh"} {
		if runner.Has(tool) {
			log.Success("%s", tool)
		} else {
			log.Error("%s not found in PATH", tool)
			problems++
		}
	}

	log.Header("config")
	if _, src, err := loadConfig(*cfgPath); err != nil {
		log.Error("%v", err)
		problems++
	} else {
		log.Success("valid (%s)", src)
	}

	if problems > 0 {
		return fmt.Errorf("%d problem(s) found", problems)
	}
	return nil
}

// ---- init -------------------------------------------------------------------

func runInit(args []string, log *ui.Logger) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	force := fs.Bool("force", false, "overwrite an existing config")
	if err := fs.Parse(args); err != nil {
		return err
	}
	path := UserConfigPath()
	if fileExists(path) && !*force {
		return fmt.Errorf("%s already exists (use --force to overwrite)", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, configs.Default, 0o644); err != nil {
		return err
	}
	log.Success("wrote %s; edit it, then run 'tweed apply --profile workstation --dry-run'", path)
	return nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
