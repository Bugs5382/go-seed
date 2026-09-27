# AGENTS.md - go-seed

Guide for AI agents working in this repository. Pair with `CLAUDE.md` (the working agreement and
hook-enforced rules). Keep this file current when the build, layout, or public API changes.

## What this is

A generic, idempotent, row-asserting seed runner for Go. It replaces hand-rolled seed scripts:
declare ordered seed steps, run them, and have each step assert the expected rows/state exist. A
clean apply is never trusted on its own — exit 0 is not proof of seeding. Shipped as a library.

## Using go-seed

The contract a consumer respects:

- Build `Step[T]` values with a `Name`, an idempotent `Apply`, and (strongly recommended) an
  `Assert` that verifies the seeded state. `T` is the app's target type (a `*sql.DB`, a repository,
  an in-memory store) — the core binds to no storage.
- Add steps to a `Runner[T]` (`New` then `Add`) and call `Run`. Steps run in order; the run is
  fail-fast and returns a `*StepError` naming the failed step and phase.
- The point of the library is the assertion. Do not add a step with a nil `Assert` unless the step
  genuinely has nothing to verify; the runner logs such a step as unverified.
- SQL apps can use `RowSpec` + `Count` over the `Executor` interface instead of writing hooks.

## Layout

- `seed.go` - core types: `Step`, `Phase`, `StepError`, sentinel config errors.
- `runner.go` - `Runner`, `New`, `Add`, `Run`, `Option`, `WithLogger`.
- `sql.go` - the `database/sql` helper path: `Executor`, `RowSpec`, `Count`.
- `doc.go` - package documentation.
- `*_test.go` - unit tests and runnable examples (`example_test.go`). The SQL tests use an
  in-process fake `database/sql` driver, so no external database is required.

## Build, test, lint

- Build: `task build` (`go build ./...`)
- Test: `task test` (`go test ./...`); no external services required.
- Lint: `task lint` (gofmt check + `golangci-lint` + `yamllint`); `task ci` for the full gate.
- License headers: `task license` (golic dry-run; CI-safe, never writes).

## Conventions and gotchas

- See `CLAUDE.md` for the branch/commit/PR rules; they are enforced by the git hooks in
  `.claude/hooks` (run `bash .claude/hooks/install.sh` once per clone).
- Open every PR as a draft. CI skips drafts, so run the full checks locally, push once they pass,
  and mark the PR ready when the work is finished; see CLAUDE.md "CI and Actions minutes".
- The public surface is stable under semver as of v1.0.0. New behavior arrives as an additive
  `Option` or a new struct field, never a breaking change without a v2.
- No third-party runtime dependencies: logging is `log/slog`, the SQL path targets only
  `database/sql`. Keep it that way.
