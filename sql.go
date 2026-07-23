package seed

/*
MIT License

Copyright (c) 2026 Shane

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
*/

import (
	"context"
	"database/sql"
	"fmt"
)

// Executor is the minimal database surface a SQL seed step needs. It binds only
// to database/sql, never to a driver; both *sql.DB and *sql.Tx satisfy it, so a
// step may run against a pool or inside a transaction the app controls.
type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// RowSpec declares a SQL seed step: an idempotent write plus the count query
// that proves it took effect. RowSpec.Step turns it into a Step[Executor].
type RowSpec struct {
	// Name identifies the step. Required.
	Name string
	// Apply is the idempotent write statement, e.g. an
	// INSERT ... ON CONFLICT DO NOTHING or an upsert. Required.
	Apply string
	// ApplyArgs are the bound arguments for Apply.
	ApplyArgs []any
	// Count is a query returning a single integer row count that proves the
	// seed exists, e.g. SELECT count(*) FROM roles WHERE name = $1. Required.
	Count string
	// CountArgs are the bound arguments for Count.
	CountArgs []any
	// WantAtLeast is the minimum row count the assertion requires. A value of
	// zero is treated as 1, so the default assertion is "at least one row".
	WantAtLeast int
}

// Step builds a Step[Executor] from the spec. Its Apply runs the write; its
// Assert runs Count and fails unless the result is at least WantAtLeast. "At
// least" (not "exactly") is intentional: a re-run over already-seeded data must
// still pass.
func (s RowSpec) Step() Step[Executor] {
	want := s.WantAtLeast
	if want < 1 {
		want = 1
	}

	return Step[Executor]{
		Name: s.Name,
		Apply: func(ctx context.Context, db Executor) error {
			if _, err := db.ExecContext(ctx, s.Apply, s.ApplyArgs...); err != nil {
				return fmt.Errorf("apply statement: %w", err)
			}
			return nil
		},
		Assert: func(ctx context.Context, db Executor) error {
			got, err := Count(ctx, db, s.Count, s.CountArgs...)
			if err != nil {
				return err
			}
			if got < want {
				return fmt.Errorf("expected at least %d row(s), found %d", want, got)
			}
			return nil
		},
	}
}

// Count runs a query that must return a single integer (typically a
// SELECT count(*)) and returns it. It is exported so apps can build custom
// assertions without depending on RowSpec.
func Count(ctx context.Context, db Executor, query string, args ...any) (int, error) {
	var n int
	if err := db.QueryRowContext(ctx, query, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("count query: %w", err)
	}
	return n, nil
}
