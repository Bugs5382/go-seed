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
	"fmt"
)

// Step is one named unit of seeding. Both hooks receive the app-supplied target
// T, so the core binds to no particular storage.
//
// Apply performs the write and must be idempotent: running it a second time
// must not duplicate rows or fail. Assert verifies the expected state exists
// after Apply and returns an error if it does not; a zero-error Apply is never
// trusted on its own. Assert is optional but strongly recommended, because a
// step with no assertion cannot prove it seeded anything.
type Step[T any] struct {
	// Name identifies the step in logs and errors. It must be non-empty and
	// unique within a Runner.
	Name string
	// Apply performs the idempotent write. It must be non-nil.
	Apply func(ctx context.Context, target T) error
	// Assert verifies the expected rows/state exist after Apply. It may be nil,
	// in which case the Runner logs a warning that the step is unverified.
	Assert func(ctx context.Context, target T) error
}

// Phase identifies which hook of a step failed.
type Phase string

const (
	// PhaseApply marks a failure raised by a step's Apply hook.
	PhaseApply Phase = "apply"
	// PhaseAssert marks a failure raised by a step's Assert hook.
	PhaseAssert Phase = "assert"
)

// StepError reports which step failed and in which phase. It wraps the
// underlying cause, which is available via errors.Unwrap / errors.Is / errors.As.
type StepError struct {
	Step  string
	Phase Phase
	Err   error
}

// Error implements error, naming the step and the phase that failed.
func (e *StepError) Error() string {
	return fmt.Sprintf("seed step %q failed during %s: %v", e.Step, e.Phase, e.Err)
}

// Unwrap returns the underlying cause.
func (e *StepError) Unwrap() error { return e.Err }

// Configuration errors returned by Run before any step executes.
var (
	// ErrEmptyStepName is returned when a step has no name.
	ErrEmptyStepName = errors.New("seed: step name must not be empty")
	// ErrDuplicateStepName is returned when two steps share a name.
	ErrDuplicateStepName = errors.New("seed: duplicate step name")
	// ErrNilApply is returned when a step has a nil Apply hook.
	ErrNilApply = errors.New("seed: step Apply must not be nil")
)
