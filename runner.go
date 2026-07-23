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
	"fmt"
	"log/slog"
)

// options holds Runner configuration set through the functional Option type.
type options struct {
	logger *slog.Logger
}

// Option configures a Runner at construction time.
type Option func(*options)

// WithLogger sets the slog.Logger the Runner uses. When unset, the Runner uses
// slog.Default().
func WithLogger(l *slog.Logger) Option {
	return func(o *options) {
		if l != nil {
			o.logger = l
		}
	}
}

// Runner applies an ordered set of Steps against a single target and asserts
// each one. It is not safe for concurrent use.
type Runner[T any] struct {
	target T
	steps  []Step[T]
	logger *slog.Logger
}

// New returns a Runner bound to target. Steps are added with Add and applied in
// the order they were added.
func New[T any](target T, opts ...Option) *Runner[T] {
	o := options{logger: slog.Default()}
	for _, opt := range opts {
		opt(&o)
	}
	return &Runner[T]{target: target, logger: o.logger}
}

// Add appends steps to the runner in order and returns the runner for chaining.
func (r *Runner[T]) Add(steps ...Step[T]) *Runner[T] {
	r.steps = append(r.steps, steps...)
	return r
}

// Run validates the step set, then applies and asserts each step in order.
//
// Validation runs first and rejects an empty or duplicate step name or a nil
// Apply hook, returning a configuration error before any step executes. Each
// step then runs Apply and, if set, Assert. A step with no Assert is applied
// but logged as unverified, because a clean Apply is not proof of seeding.
//
// Run is fail-fast: the first failing step stops the run and returns a
// *StepError naming the step and the phase (PhaseApply or PhaseAssert) that
// failed. Because every Apply must be idempotent, Run is safe to call
// repeatedly.
func (r *Runner[T]) Run(ctx context.Context) error {
	if err := r.validate(); err != nil {
		return err
	}

	for _, step := range r.steps {
		r.logger.InfoContext(ctx, "applying seed step", slog.String("step", step.Name))

		if err := step.Apply(ctx, r.target); err != nil {
			return &StepError{Step: step.Name, Phase: PhaseApply, Err: err}
		}

		if step.Assert == nil {
			r.logger.WarnContext(ctx, "seed step has no assertion; a clean apply is not proof it seeded",
				slog.String("step", step.Name))
			continue
		}

		if err := step.Assert(ctx, r.target); err != nil {
			return &StepError{Step: step.Name, Phase: PhaseAssert, Err: err}
		}

		r.logger.InfoContext(ctx, "seed step verified", slog.String("step", step.Name))
	}

	return nil
}

// validate checks the step set for configuration errors before any step runs.
func (r *Runner[T]) validate() error {
	seen := make(map[string]struct{}, len(r.steps))
	for i, step := range r.steps {
		if step.Name == "" {
			return fmt.Errorf("%w: step at index %d", ErrEmptyStepName, i)
		}
		if _, dup := seen[step.Name]; dup {
			return fmt.Errorf("%w: %q", ErrDuplicateStepName, step.Name)
		}
		seen[step.Name] = struct{}{}
		if step.Apply == nil {
			return fmt.Errorf("%w: step %q", ErrNilApply, step.Name)
		}
	}
	return nil
}
