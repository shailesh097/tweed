# Configuration reference

A config has `profiles` and `modules`. Unknown keys are an error, so typos fail loudly.

```yaml
profiles:
  <name>: [<module>, <module>, ...]   # applied in this order

modules:
  <name>:
    description: <text shown by `tweed list`>
    dnf_config:  {<key>: "<value>"}
    repos:       [<repo>, ...]
    dnf:         [<package>, ...]
    flatpak:     [<app id>, ...]
    gsettings:   [{schema, key, value}, ...]
    commands:    [{name, run, unless, sudo}, ...]
```

Within a module, steps always run in the order listed above.

## `dnf_config`

Lines in `/etc/dnf/dnf.conf`. Existing keys are replaced, missing keys appended.
Values are strings (`"True"`, `"10"`).

## `repos`

Each entry uses **exactly one** form:

| Form | Example | Done when |
|---|---|---|
| `builtin` | `rpmfusion-free`, `rpmfusion-nonfree`, `flathub` | release package / remote exists |
| `copr` | `atim/starship` | COPR repo file exists |
| `repofile` | URL to a `.repo` file | `/etc/yum.repos.d/<filename>` exists |
| inline | `id`, `baseurl`, optional `name`, `gpgkey` | `/etc/yum.repos.d/<id>.repo` exists |

## `dnf` / `flatpak`

Package names / Flatpak app IDs. Only missing ones are installed. dnf uses
`--skip-unavailable`, so a package that doesn't exist on your release is skipped
instead of failing the whole step. Flatpaks install from Flathub.

## `gsettings`

```yaml
- schema: org.gnome.desktop.session
  key: idle-delay
  value: "900"
```

Compared against `gsettings get` (type prefixes like `uint32` and quotes are ignored).

## `commands`

The escape hatch for anything without a dedicated step yet.

```yaml
- name: NvChad starter config         # shown in output
  run: git clone https://github.com/NvChad/starter "$HOME/.config/nvim"
  unless: test -d "$HOME/.config/nvim" # optional: exit 0 = already done
  sudo: false                          # optional: run `run` via sudo
```

Without `unless`, the command runs every time, so add one whenever you can.
If a command keeps showing up across modules, it's a sign it deserves a real step type.
