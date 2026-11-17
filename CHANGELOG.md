# Changelog

All notable changes are documented here. Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
versioning: [SemVer](https://semver.org/).

## [Unreleased]

### Added
- Go rewrite of [fedora_tweed](https://github.com/shailesh097/fedora_tweed): declarative `tweed.yaml` with profiles and modules
- Step types: `dnf_config`, `repos` (RPM Fusion, Flathub, COPR, repo file, inline), `dnf`, `flatpak`, `gsettings`, `commands`
- Commands: `apply` (with `--dry-run`, `--keep-going`, `--yes`), `list`, `doctor`, `init`, `version`
- Default config built into the binary
- CI (tests, lint, Fedora container dry-run), GoReleaser release pipeline, install script
