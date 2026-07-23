# go-seed 🌱

> A generic, idempotent, **row-asserting** seed runner for Go. Declare ordered seed steps, run them, and have each step prove it actually seeded — because exit 0 is not proof.

## 📦 Install

```bash
go get github.com/Bugs5382/go-seed
```

## Why

Hand-rolled seed scripts share three defects: they are not ordered in a
reviewable way, they are not idempotent, and they treat a zero exit code as
success. A script can exit clean having seeded nothing. `go-seed` fixes the
third defect structurally: every step is followed by an assertion that the
expected rows/state exist, and the run fails loudly (naming the step) if not.

## 🚀 Usage

Declare steps against any target — the runner is generic, so the target can be a
`*sql.DB`, a repository, or an in-memory store. Each step has an idempotent
`Apply` and an `Assert` that verifies the result.

```go
runner := seed.New(db).Add(
    seed.Step[*sql.DB]{
        Name:   "admin-role",
        Apply:  func(ctx context.Context, db *sql.DB) error { /* idempotent write */ return nil },
        Assert: func(ctx context.Context, db *sql.DB) error { /* verify row exists */ return nil },
    },
)

if err := runner.Run(ctx); err != nil {
    // err is a *seed.StepError naming the step and phase (apply/assert) that failed.
    log.Fatal(err)
}
```

### SQL helper

For the common relational case, declare a `RowSpec` instead of writing the hooks
by hand. `Apply` must be an idempotent statement; `Count` proves it took effect.

```go
step := seed.RowSpec{
    Name:      "admin-role",
    Apply:     "INSERT INTO roles(name) VALUES ($1) ON CONFLICT DO NOTHING",
    ApplyArgs: []any{"admin"},
    Count:     "SELECT count(*) FROM roles WHERE name = $1",
    CountArgs: []any{"admin"},
    // WantAtLeast defaults to 1.
}.Step()

// *sql.DB and *sql.Tx both satisfy seed.Executor.
err := seed.New[seed.Executor](db).Add(step).Run(ctx)
```

## Behavior

- **Ordered**: steps run in the order they are added.
- **Idempotent**: your `Apply` must be idempotent; the assertion is what makes a
  re-run safe to trust. Running twice does not duplicate or error.
- **Row-asserting**: a clean `Apply` is never trusted. A step with no `Assert` is
  applied but logged as unverified.
- **Fail-fast**: the first failing step stops the run and returns a
  `*seed.StepError` naming the step and the phase.
- **Logging**: standard-library `log/slog` (`seed.WithLogger` to override). No
  third-party logging dependency.

## 🛠 Develop

```bash
task build    # go build ./...
task test     # go test ./...
task lint     # gofmt check + golangci-lint + yamllint
task ci       # full local gate
task license  # verify MIT headers (golic)
```

## ⚖️ License

MIT © 2026 Shane
