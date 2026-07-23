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
	"errors"
	"strings"
	"testing"
)

func TestStepErrorMessageNamesStepAndPhase(t *testing.T) {
	t.Parallel()

	err := &StepError{Step: "admin-role", Phase: PhaseAssert, Err: errors.New("found 0 rows")}
	msg := err.Error()

	for _, want := range []string{"admin-role", "assert", "found 0 rows"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message %q missing %q", msg, want)
		}
	}
}

func TestStepErrorUnwrapReturnsCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("boom")
	err := &StepError{Step: "s", Phase: PhaseApply, Err: cause}

	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is could not find the wrapped cause")
	}
}
