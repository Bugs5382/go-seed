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
	"database/sql/driver"
	"errors"
	"io"
	"sync"
	"testing"
)

// The tests below use a minimal in-process database/sql driver so RowSpec can be
// exercised against a real *sql.DB / *sql.Row without any external dependency.

// fakeDB is the shared state a fake connection reads: the count its query
// returns and a log of the statements Exec received.
type fakeDB struct {
	count int64
	execs []string
}

var (
	fakeMu   sync.Mutex
	fakeReg  = map[string]*fakeDB{}
	fakeOnce sync.Once
)

func registerFake(t *testing.T, count int64) (*sql.DB, *fakeDB) {
	t.Helper()
	fakeOnce.Do(func() { sql.Register("seedfake", fakeDriver{}) })

	fdb := &fakeDB{count: count}
	dsn := t.Name()
	fakeMu.Lock()
	fakeReg[dsn] = fdb
	fakeMu.Unlock()

	db, err := sql.Open("seedfake", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, fdb
}

type fakeDriver struct{}

func (fakeDriver) Open(dsn string) (driver.Conn, error) {
	fakeMu.Lock()
	defer fakeMu.Unlock()
	return &fakeConn{db: fakeReg[dsn]}, nil
}

type fakeConn struct{ db *fakeDB }

func (c *fakeConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("prepare unused") }
func (c *fakeConn) Close() error                        { return nil }
func (c *fakeConn) Begin() (driver.Tx, error)           { return nil, errors.New("tx unused") }

func (c *fakeConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	c.db.execs = append(c.db.execs, query)
	return driver.RowsAffected(1), nil
}

func (c *fakeConn) QueryContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
	return &fakeRows{val: c.db.count}, nil
}

type fakeRows struct {
	val  int64
	done bool
}

func (r *fakeRows) Columns() []string { return []string{"count"} }
func (r *fakeRows) Close() error      { return nil }
func (r *fakeRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	dest[0] = r.val
	return nil
}

func TestRowSpecStepAppliesThenAsserts(t *testing.T) {
	t.Parallel()

	db, fdb := registerFake(t, 1)
	step := RowSpec{
		Name:      "seed-admin",
		Apply:     "INSERT INTO roles(name) VALUES ($1) ON CONFLICT DO NOTHING",
		ApplyArgs: []any{"admin"},
		Count:     "SELECT count(*) FROM roles WHERE name = $1",
		CountArgs: []any{"admin"},
	}.Step()

	if err := quietRunner[Executor](db).Add(step).Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(fdb.execs) != 1 {
		t.Fatalf("apply ran %d times, want 1", len(fdb.execs))
	}
}

func TestRowSpecAssertionFailsBelowWanted(t *testing.T) {
	t.Parallel()

	db, _ := registerFake(t, 0) // count query returns 0 -> unseeded
	step := RowSpec{
		Name:  "missing",
		Apply: "INSERT ...",
		Count: "SELECT count(*) FROM roles",
	}.Step()

	err := quietRunner[Executor](db).Add(step).Run(context.Background())
	var se *StepError
	if !errors.As(err, &se) || se.Phase != PhaseAssert {
		t.Fatalf("expected an assert-phase StepError, got %v", err)
	}
}

func TestRowSpecWantAtLeastZeroDefaultsToOne(t *testing.T) {
	t.Parallel()

	// WantAtLeast left at zero; a single row must satisfy it.
	db, _ := registerFake(t, 1)
	step := RowSpec{Name: "s", Apply: "INSERT ...", Count: "SELECT count(*)"}.Step()

	if err := quietRunner[Executor](db).Add(step).Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRowSpecHonorsHigherWantAtLeast(t *testing.T) {
	t.Parallel()

	db, _ := registerFake(t, 2) // only 2 rows present
	step := RowSpec{
		Name:        "needs-three",
		Apply:       "INSERT ...",
		Count:       "SELECT count(*)",
		WantAtLeast: 3,
	}.Step()

	err := quietRunner[Executor](db).Add(step).Run(context.Background())
	var se *StepError
	if !errors.As(err, &se) || se.Phase != PhaseAssert {
		t.Fatalf("expected assert failure for 2 < 3, got %v", err)
	}
}

func TestCountReturnsScalar(t *testing.T) {
	t.Parallel()

	db, _ := registerFake(t, 7)
	got, err := Count(context.Background(), db, "SELECT count(*)")
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if got != 7 {
		t.Fatalf("Count = %d, want 7", got)
	}
}
