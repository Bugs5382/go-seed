package seed_test

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
	"io"
	"log/slog"

	seed "github.com/Bugs5382/go-seed"
)

// roleStore is a tiny in-memory target, standing in for whatever storage an app
// seeds. Because the runner is generic over the target type, no database or
// driver is needed to demonstrate it.
type roleStore struct{ roles map[string]bool }

// quiet discards the runner's own log output so example output is deterministic.
func quiet() seed.Option {
	return seed.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

// Example declares two ordered seed steps and runs them. Each step applies an
// idempotent write and then asserts the expected state exists.
func Example() {
	db := &roleStore{roles: map[string]bool{}}

	runner := seed.New(db, quiet()).Add(
		seed.Step[*roleStore]{
			Name:   "admin-role",
			Apply:  func(_ context.Context, s *roleStore) error { s.roles["admin"] = true; return nil },
			Assert: assertRole("admin"),
		},
		seed.Step[*roleStore]{
			Name:   "auditor-role",
			Apply:  func(_ context.Context, s *roleStore) error { s.roles["auditor"] = true; return nil },
			Assert: assertRole("auditor"),
		},
	)

	if err := runner.Run(context.Background()); err != nil {
		fmt.Println("seed failed:", err)
		return
	}
	fmt.Println("seeded and verified")
	// Output: seeded and verified
}

// Example_assertionCatchesUnseeded shows the core guarantee: a step whose Apply
// exits cleanly but writes nothing is caught by its assertion, instead of being
// mistaken for a successful seed.
func Example_assertionCatchesUnseeded() {
	db := &roleStore{roles: map[string]bool{}}

	runner := seed.New(db, quiet()).Add(seed.Step[*roleStore]{
		Name:   "admin-role",
		Apply:  func(_ context.Context, _ *roleStore) error { return nil }, // exits 0, seeds nothing
		Assert: assertRole("admin"),
	})

	err := runner.Run(context.Background())

	var stepErr *seed.StepError
	if errors.As(err, &stepErr) {
		fmt.Printf("caught: step %q failed during %s\n", stepErr.Step, stepErr.Phase)
	}
	// Output: caught: step "admin-role" failed during assert
}

func assertRole(name string) func(context.Context, *roleStore) error {
	return func(_ context.Context, s *roleStore) error {
		if !s.roles[name] {
			return fmt.Errorf("role %q not present", name)
		}
		return nil
	}
}
