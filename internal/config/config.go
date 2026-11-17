// Package config loads and validates tweed.yaml.
//
// A config has two parts:
//   - modules:  named groups of things to set up (packages, repos, settings...)
//   - profiles: named, ordered lists of modules (e.g. "workstation", "server")
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// Config is the top-level structure of tweed.yaml.
type Config struct {
	Profiles map[string][]string `yaml:"profiles"`
	Modules  map[string]Module   `yaml:"modules"`
}

// Module is one unit of setup. Steps run in the order the fields are listed:
// dnf_config → repos → dnf → flatpak → gsettings → commands.
type Module struct {
	Description string            `yaml:"description"`
	DNFConfig   map[string]string `yaml:"dnf_config"`
	Repos       []Repo            `yaml:"repos"`
	DNF         []string          `yaml:"dnf"`
	Flatpak     []string          `yaml:"flatpak"`
	GSettings   []GSetting        `yaml:"gsettings"`
	Commands    []Command         `yaml:"commands"`
}

// Repo describes a package source. Exactly one kind is used:
//
//	builtin:  rpmfusion-free | rpmfusion-nonfree | flathub
//	copr:     "owner/project"
//	repofile: URL of a .repo file (dnf config-manager addrepo)
//	id+baseurl(+gpgkey): an inline repo written to /etc/yum.repos.d/<id>.repo
type Repo struct {
	Builtin  string `yaml:"builtin"`
	Copr     string `yaml:"copr"`
	RepoFile string `yaml:"repofile"`
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	BaseURL  string `yaml:"baseurl"`
	GPGKey   string `yaml:"gpgkey"`
}

// GSetting is a single `gsettings set <schema> <key> <value>`.
type GSetting struct {
	Schema string `yaml:"schema"`
	Key    string `yaml:"key"`
	Value  string `yaml:"value"`
}

// Command is an escape hatch for things tweed has no dedicated step for yet.
// If Unless is set and exits 0, the command is considered already done.
type Command struct {
	Name   string `yaml:"name"`
	Run    string `yaml:"run"`
	Unless string `yaml:"unless"`
	Sudo   bool   `yaml:"sudo"`
}

// Kind returns which kind of repo this is, or "" if none/ambiguous.
func (r Repo) Kind() string {
	kinds := []string{}
	if r.Builtin != "" {
		kinds = append(kinds, "builtin")
	}
	if r.Copr != "" {
		kinds = append(kinds, "copr")
	}
	if r.RepoFile != "" {
		kinds = append(kinds, "repofile")
	}
	if r.ID != "" || r.BaseURL != "" {
		kinds = append(kinds, "inline")
	}
	if len(kinds) != 1 {
		return ""
	}
	return kinds[0]
}

// KnownBuiltinRepos lists the values accepted by Repo.Builtin.
var KnownBuiltinRepos = []string{"rpmfusion-free", "rpmfusion-nonfree", "flathub"}

// Load reads and validates a config file. Unknown keys are rejected so typos
// fail loudly instead of being silently ignored.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	return Parse(data)
}

// Parse decodes and validates config bytes.
func Parse(data []byte) (*Config, error) {
	var c Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// Validate checks the config for mistakes and returns all of them at once.
func (c *Config) Validate() error {
	var errs []error
	if len(c.Modules) == 0 {
		errs = append(errs, errors.New("config defines no modules"))
	}
	for _, p := range sortedKeys(c.Profiles) {
		if len(c.Profiles[p]) == 0 {
			errs = append(errs, fmt.Errorf("profile %q is empty", p))
		}
		for _, m := range c.Profiles[p] {
			if _, ok := c.Modules[m]; !ok {
				errs = append(errs, fmt.Errorf("profile %q references unknown module %q", p, m))
			}
		}
	}
	for _, name := range sortedKeys(c.Modules) {
		m := c.Modules[name]
		for i, r := range m.Repos {
			switch r.Kind() {
			case "":
				errs = append(errs, fmt.Errorf("module %q repo #%d: set exactly one of builtin, copr, repofile, or id+baseurl", name, i+1))
			case "builtin":
				if !contains(KnownBuiltinRepos, r.Builtin) {
					errs = append(errs, fmt.Errorf("module %q repo #%d: unknown builtin %q (known: %v)", name, i+1, r.Builtin, KnownBuiltinRepos))
				}
			case "inline":
				if r.ID == "" || r.BaseURL == "" {
					errs = append(errs, fmt.Errorf("module %q repo #%d: inline repos need both id and baseurl", name, i+1))
				}
			}
		}
		for i, g := range m.GSettings {
			if g.Schema == "" || g.Key == "" {
				errs = append(errs, fmt.Errorf("module %q gsettings #%d: schema and key are required", name, i+1))
			}
		}
		for i, cmd := range m.Commands {
			if cmd.Name == "" || cmd.Run == "" {
				errs = append(errs, fmt.Errorf("module %q command #%d: name and run are required", name, i+1))
			}
		}
	}
	return errors.Join(errs...)
}

// Resolve turns a profile name and/or explicit module names into an ordered,
// de-duplicated list of module names. Explicit modules come after the profile's.
func (c *Config) Resolve(profile string, modules []string) ([]string, error) {
	var names []string
	if profile != "" {
		p, ok := c.Profiles[profile]
		if !ok {
			return nil, fmt.Errorf("unknown profile %q (available: %v)", profile, sortedKeys(c.Profiles))
		}
		names = append(names, p...)
	}
	for _, m := range modules {
		if _, ok := c.Modules[m]; !ok {
			return nil, fmt.Errorf("unknown module %q (available: %v)", m, sortedKeys(c.Modules))
		}
		names = append(names, m)
	}
	if len(names) == 0 {
		return nil, errors.New("nothing selected: pass --profile and/or --module")
	}
	seen := map[string]bool{}
	out := names[:0]
	for _, n := range names {
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out, nil
}

// ProfileNames returns profile names sorted alphabetically.
func (c *Config) ProfileNames() []string { return sortedKeys(c.Profiles) }

// ModuleNames returns module names sorted alphabetically.
func (c *Config) ModuleNames() []string { return sortedKeys(c.Modules) }

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
