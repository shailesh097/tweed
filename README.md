# tweed

[![CI](https://github.com/shailesh097/tweed/actions/workflows/ci.yml/badge.svg)](https://github.com/shailesh097/tweed/actions/workflows/ci.yml)

**Set up a fresh Fedora install from one config file.**

tweed is the Go successor to [fedora_tweed](https://github.com/shailesh097/fedora_tweed).
Instead of one long script, you describe *what* you want in `tweed.yaml` (packages,
repos, Flatpaks, GNOME settings...), grouped into **modules** and **profiles**,
and tweed works out what's missing and applies only that.

```console
$ tweed apply --profile workstation --dry-run
==> module base
  • dnf.conf: defaultyes=True (already done)
  dnf: 15 package(s)
  [dry-run] $ sudo dnf install -y --skip-unavailable fzf zoxide tldr
  ...
==> summary: 12 applied, 9 already done, 0 failed
```

## Features

- **Declarative**: your setup lives in YAML, not in shell logic
- **Profiles**: `workstation`, `server`, or your own mix of modules
- **Idempotent**: run it as often as you like; finished steps are skipped
- **Dry run**: see every command before anything changes
- **Single binary**: no need to clone the repo on a fresh install; the default config is built in

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/shailesh097/tweed/main/scripts/install.sh | bash
```

Or from source (Go 1.24+):

```bash
git clone https://github.com/shailesh097/tweed && cd tweed
make install        # → ~/.local/bin/tweed
```

## Usage

```bash
tweed doctor                                  # is this system ready?
tweed list                                    # show profiles and modules
tweed apply --profile workstation --dry-run   # preview
tweed apply --profile workstation             # do it
tweed apply -m dev,apps                       # just some modules
tweed init                                    # copy the default config to ~/.config/tweed/tweed.yaml to customise
```

| Flag | Meaning |
|---|---|
| `-p, --profile` | Profile to apply |
| `-m, --module` | Module(s) to apply; repeatable or comma-separated |
| `-n, --dry-run` | Show commands, change nothing |
| `-y, --yes` | Skip the confirmation prompt |
| `--keep-going` | Don't stop at the first failed step |
| `--no-upgrade` | Skip `dnf upgrade` in modules with `upgrade: true` |
| `-c, --config` | Use a specific config file |

Config lookup order: `--config` → `~/.config/tweed/tweed.yaml` → built-in default.

## Configuration

See [`configs/tweed.yaml`](configs/tweed.yaml) for the full default setup and
[`docs/config.md`](docs/config.md) for the reference.

```yaml
profiles:
  server: [base, shell]

modules:
  shell:
    repos:
      - copr: atim/starship
    dnf: [fish, starship]
    commands:
      - name: set fish as login shell
        run: sudo chsh -s /usr/bin/fish "$USER"
        unless: getent passwd "$USER" | grep -q fish
```

## Requirements

Fedora 41 or newer (dnf5), running as your normal user (tweed calls `sudo` when needed).

## Project

- [Roadmap](docs/ROADMAP.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Contributing](CONTRIBUTING.md)
- [Changelog](CHANGELOG.md)

## License

[MIT](LICENSE)
