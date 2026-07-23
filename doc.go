// Package seed is a generic, idempotent, row-asserting seed runner. It replaces
// hand-rolled seed scripts with declarative, ordered seed steps that verify
// their own effect: applying a step is never trusted on its own, because a
// clean exit is not proof that any row was written.
//
// An app declares Steps, each with an idempotent Apply hook and an Assert hook
// that checks the expected rows/state exist, then runs them in order with a
// Runner. The core is storage-agnostic: the target is a type parameter, so a
// step can run against a *sql.DB, a repository, or an in-memory store. For the
// common relational case, RowSpec builds a Step from a declarative write plus a
// count assertion against the database/sql Executor surface.
//
// Logging uses the standard library log/slog; there is no third-party logging
// dependency.
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
