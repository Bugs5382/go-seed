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
	"errors"
	"io"
	"log/slog"
	"testing"
)

// quietRunner builds a Runner whose logs are discarded, keeping test output clean.
func quietRunner[T any](target T) *Runner[T] {
	return New(target, WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
}

// store is an in-memory target used to exercise the storage-agnostic core.
type store struct {
	rows  map[string]int
	order []string
}

func newStore() *store { return &store{rows: map[string]int{}} }

func TestRunAppliesStepsInOrder(t *testing.T) {
	t.Parallel()

	s := newStore()
	step := func(name string) Step[*store] {
		return Step[*store]{
			Name: name,
			Apply: func(_ context.Context, st *store) error {
				st.order = append(st.order, name)
				st.rows[name]++
				return nil
			},
			Assert: func(_ context.Context, st *store) error {
				if st.rows[name] < 1 {
					return errors.New("row missing")
				}
				return nil
			},
		}
	}

	err := quietRunner(s).Add(step("a"), step("b"), step("c")).Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	got := s.order
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("steps ran out of order: %v", got)
	}
}

func TestRunIsIdempotentAcrossReRuns(t *testing.T) {
	t.Parallel()

	s := newStore()
	// Apply is written idempotently: it sets the row rather than incrementing.
	r := quietRunner(s).Add(Step[*store]{
		Name: "role",
		Apply: func(_ context.Context, st *store) error {
			st.rows["role"] = 1
			return nil
		},
		Assert: func(_ context.Context, st *store) error {
			if st.rows["role"] != 1 {
				return errors.New("expected exactly one role")
			}
			return nil
		},
	})

	for i := 0; i < 3; i++ {
		if err := r.Run(context.Background()); err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
	}
	if s.rows["role"] != 1 {
		t.Fatalf("re-runs mutated state: got %d", s.rows["role"])
	}
}

func TestRunReturnsStepErrorOnFailedAssertion(t *testing.T) {
	t.Parallel()

	s := newStore()
	// Apply is a no-op: it exits clean but writes nothing. The assertion must
	// catch that the row is absent -- the whole point of the runner.
	err := quietRunner(s).Add(Step[*store]{
		Name:  "unseeded",
		Apply: func(_ context.Context, _ *store) error { return nil },
		Assert: func(_ context.Context, st *store) error {
			if st.rows["unseeded"] < 1 {
				return errors.New("no rows written")
			}
			return nil
		},
	}).Run(context.Background())

	var se *StepError
	if !errors.As(err, &se) {
		t.Fatalf("expected *StepError, got %T: %v", err, err)
	}
	if se.Step != "unseeded" || se.Phase != PhaseAssert {
		t.Fatalf("wrong StepError: step=%q phase=%q", se.Step, se.Phase)
	}
}

func TestRunWrapsApplyFailureWithApplyPhase(t *testing.T) {
	t.Parallel()

	boom := errors.New("connection refused")
	err := quietRunner(newStore()).Add(Step[*store]{
		Name:   "s",
		Apply:  func(_ context.Context, _ *store) error { return boom },
		Assert: func(_ context.Context, _ *store) error { return nil },
	}).Run(context.Background())

	var se *StepError
	if !errors.As(err, &se) {
		t.Fatalf("expected *StepError, got %T", err)
	}
	if se.Phase != PhaseApply {
		t.Fatalf("phase = %q, want apply", se.Phase)
	}
	if !errors.Is(err, boom) {
		t.Fatalf("StepError did not wrap the apply cause")
	}
}

func TestRunFailsFastAndStopsLaterSteps(t *testing.T) {
	t.Parallel()

	s := newStore()
	ran := false
	err := quietRunner(s).Add(
		Step[*store]{
			Name:   "first",
			Apply:  func(_ context.Context, _ *store) error { return errors.New("fail") },
			Assert: func(_ context.Context, _ *store) error { return nil },
		},
		Step[*store]{
			Name: "second",
			Apply: func(_ context.Context, _ *store) error {
				ran = true
				return nil
			},
		},
	).Run(context.Background())

	if err == nil {
		t.Fatal("expected an error")
	}
	if ran {
		t.Fatal("second step ran after the first failed; not fail-fast")
	}
}

func TestRunAllowsNilAssertButRunsApply(t *testing.T) {
	t.Parallel()

	s := newStore()
	err := quietRunner(s).Add(Step[*store]{
		Name:  "unverified",
		Apply: func(_ context.Context, st *store) error { st.rows["x"] = 1; return nil },
	}).Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if s.rows["x"] != 1 {
		t.Fatal("apply did not run for a nil-assert step")
	}
}

func TestRunRejectsInvalidStepSets(t *testing.T) {
	t.Parallel()

	ok := func(_ context.Context, _ *store) error { return nil }

	tests := map[string]struct {
		steps  []Step[*store]
		target error
	}{
		"empty name": {
			steps:  []Step[*store]{{Name: "", Apply: ok}},
			target: ErrEmptyStepName,
		},
		"duplicate name": {
			steps:  []Step[*store]{{Name: "dup", Apply: ok}, {Name: "dup", Apply: ok}},
			target: ErrDuplicateStepName,
		},
		"nil apply": {
			steps:  []Step[*store]{{Name: "s", Apply: nil}},
			target: ErrNilApply,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := quietRunner(newStore()).Add(tc.steps...).Run(context.Background())
			if !errors.Is(err, tc.target) {
				t.Fatalf("got %v, want %v", err, tc.target)
			}
		})
	}
}

func TestNewFallsBackToDefaultLogger(t *testing.T) {
	t.Parallel()

	// A nil logger option must not panic; the runner falls back to the default.
	r := New(newStore(), WithLogger(nil))
	if r.logger == nil {
		t.Fatal("runner logger is nil")
	}
}
