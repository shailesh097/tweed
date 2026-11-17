# Contributing

Ideas and bug reports are welcome. Please open an issue first for anything big.

## Setup

```bash
git clone https://github.com/shailesh097/tweed && cd tweed
go mod tidy
make check      # tidy + fmt + lint + test, same as CI
make dry-run    # build and preview the workstation profile
```

Tools: Go 1.24+, [golangci-lint](https://golangci-lint.run/) v2, optionally
[goreleaser](https://goreleaser.com/) and podman (`make fedora-test`).

## Workflow

1. Pick or open an issue.
2. Branch from `main`: `feat/<short-name>`, `fix/<short-name>`, `docs/...`, `chore/...`.
3. Small commits using [Conventional Commits](https://www.conventionalcommits.org/):

   | Prefix | Use for |
   |---|---|
   | `feat:` | new step type, command, module |
   | `fix:` | bug fix |
   | `refactor:` | code change, same behavior |
   | `test:` | tests only |
   | `docs:` | documentation only |
   | `chore:` | tooling, deps, cleanup, removals |
   | `ci:` | GitHub Actions |

   Add `!` for breaking config changes: `feat!: rename dnf to packages`.
4. Open a PR into `main` and reference the issue (`Closes #12`). CI must be green.
5. Squash-merge, so `main` keeps one clean commit per PR.

## Releases

`main` is always releasable. To release:

```bash
# update CHANGELOG.md (move "Unreleased" items under the new version), commit
git tag -a v0.2.0 -m "v0.2.0"
git push origin v0.2.0      # GitHub Actions builds binaries and the release
```

Versioning follows [SemVer](https://semver.org/). Before 1.0, minor versions may
change the config format; note it in the changelog.

## Testing changes that touch the system

Unit tests use `runner.Fake` and never touch your machine. For real runs, use a
Fedora VM (GNOME Boxes) or `make fedora-test`, never only your daily driver.
