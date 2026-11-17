# Architecture

```
cmd/tweed/          main(): parses nothing, just calls cli.Run
configs/            default tweed.yaml, embedded into the binary
internal/
  cli/              subcommands and flags (apply, list, doctor, init, version)
  config/           YAML types, Load/Parse, Validate, Resolve(profile, modules)
  steps/            Step interface, one type per kind of change, Execute loop
  runner/           runs commands: Query (read-only) vs Run (changes system), dry-run, Fake for tests
  system/           facts about the machine (/etc/os-release)
  ui/               colored output, confirmation prompt
```

## Flow of `tweed apply -p workstation`

```
cli ──► config.Load ──► cfg.Resolve("workstation") ──► [base, shell, ...]
                                                           │
            for each module ─► steps.FromModule(m) ─► []Step
                                                           │
            for each step:  Check(runner) ── done? ──► skip
                                   │ no
                                   └──► Apply(runner) ──► runner.Run (or print, in dry-run)
```

## Rules that keep it maintainable

1. **Steps never call `os/exec` directly.** They go through `runner.Runner`, so
   dry-run works everywhere and every step is testable with `runner.Fake`.
2. **`Check` must not change anything.** It uses `Query`, which runs even in dry-run.
3. **Config is data, code is behavior.** Adding an app = editing YAML. Adding a
   *kind* of change (e.g. GNOME extensions) = a new `Step` type + config field + tests.
4. **Validate early.** `config.Validate` reports every problem at once, before anything runs.

## Adding a new step type (checklist)

1. Add the field to `config.Module` (+ validation in `Validate`).
2. Implement `Step` in `internal/steps/` (Describe / Check / Apply).
3. Wire it into `FromModule` in the right order.
4. Tests with `runner.Fake`: "already done" and "needs apply" cases.
5. Document it in `docs/config.md`; use it in `configs/tweed.yaml`.
