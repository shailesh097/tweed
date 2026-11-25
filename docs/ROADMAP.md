# Roadmap

Goal: replace [fedora_tweed](https://github.com/shailesh097/fedora_tweed) (Bash) with a
declarative, idempotent, single-binary Go tool, then go beyond it.

Each milestone below maps to a **GitHub milestone**; each bullet is an **issue**.

---

## v0.2.0: Foundation ✅ (initial scaffold)

- [x] Go module, `cmd/` + `internal/` layout
- [x] Config: profiles + modules, strict YAML parsing, validation of all errors at once
- [x] Runner with dry-run and a fake for tests
- [x] Steps: `dnf_config`, `repos` (RPM Fusion, Flathub, COPR, repo file, inline), `dnf`, `flatpak`, `gsettings`, `commands`
- [x] CLI: `apply`, `list`, `doctor`, `init`, `version`
- [x] Default config ported from fedora_tweed, embedded in the binary
- [x] CI (test, lint, Fedora container dry-run), GoReleaser, install script
- [x] Create GitHub repo, labels, milestones; push; tag `v0.2.0`
- [x] First real run in a Fedora 44 VM; fix whatever breaks

## v0.3.0: GNOME desktop

- [ ] `extensions` step: install from extensions.gnome.org by UUID (`gnome-extensions install`) and enable
      (replaces the dash-to-dock zip, clipboard-indicator clone, Coverflow `make` hacks)
- [ ] `shortcuts` step: custom keybindings (`<super>Return` kitty, `<super>b` brave, `<super>f` nautilus)
      without the Python `ast` workaround
- [ ] `keybindings` step for built-in WM bindings (`<super>q` close window, `<super>s` settings, workspaces)
- [ ] Desktop module in `tweed.yaml` reaches parity with `keyboard_shortcuts/` and `gnome_shell_extensions.sh`

## v0.4.0: Dotfiles and services

- [ ] `dotfiles` step: clone a repo, link or copy folders into `~/.config`, with timestamped backups
- [ ] `tweed sync`: replaces `bins/dotfiles_sync.sh` (no more `shc`)
- [ ] `systemd_user` step: install/enable user units
- [ ] Wallpaper changer as a systemd **timer** instead of a `while true; sleep` loop
- [ ] Fonts / icons / themes from the dotfiles repo

## v0.5.0: Better UX

- [ ] Interactive module picker when no `--profile`/`--module` is given (charmbracelet/huh)
- [ ] `tweed status`: run all checks only and show what's applied vs pending
- [ ] Conditions on modules, e.g. `when: {gpu: nvidia}`, `when: {variant: workstation}` (auto-skip NVIDIA on the server)
- [ ] Reboot hint/prompt when a module declares `reboot: true` (NVIDIA)
- [ ] Log file in `~/.local/state/tweed/` for every non-dry run

## v1.0.0: Stable

- [ ] Full parity with fedora_tweed; archive the old repo with a pointer here
- [ ] Config schema versioned (`version: 1`) with clear errors for old formats
- [ ] Tested end-to-end in VMs on the two latest Fedora releases
- [ ] Docs complete: config reference, examples, FAQ

## Later / ideas

- Variables in config (`{{ .User }}`, `{{ .Home }}`) instead of hardcoded values (e.g. git identity)
- `include:` to split config across files; machine-specific overrides
- `tweed export`: generate a config from the current system
- Fedora Atomic (Silverblue) support via `rpm-ostree`
- Shell completions (fish, bash)

---

## fedora_tweed migration checklist

| fedora_tweed feature | Where in tweed | Status |
|---|---|---|
| dnf.conf tweaks | `base.dnf_config` | ✅ |
| RPM Fusion, Flathub | `base.repos` | ✅ |
| CLI tools (dnf) | `base.dnf` | ✅ |
| Workspace dir, git identity | `base.commands` | ✅ |
| fish + starship (COPR), chsh | `shell` | ✅ |
| Power settings, titlebar buttons | `desktop.gsettings` | ✅ |
| Kitty Catppuccin theme | `desktop.commands` | ✅ |
| Fedora-packaged GNOME extensions | `desktop.dnf` | ✅ |
| VS Code, Sublime, NvChad | `dev` | ✅ |
| Brave, Discord, VLC, Flatpak apps | `apps` | ✅ |
| NVIDIA driver | `nvidia` | ✅ (no reboot prompt yet) |
| X11 session | `x11` (not in profiles; dropped by recent GNOME) | ✅ |
| dash-to-dock, clipboard-indicator, Coverflow | `extensions` step | v0.3 |
| Custom keyboard shortcuts | `shortcuts` step | v0.3 |
| Dotfiles clone + copy, fonts/icons | `dotfiles` step | v0.4 |
| `dotfiles_sync.sh` | `tweed sync` | v0.4 |
| Dynamic wallpaper service | `systemd_user` step | v0.4 |

## GitHub labels

`bug` · `enhancement` · `step-type` · `config` · `cli` · `docs` · `ci` · `migration` · `good first issue`
