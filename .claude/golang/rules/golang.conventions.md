---
name: golang-conventions
description: General Go coding conventions based on Effective Go, CodeReviewComments, and Google Go Style Guide. Naming, formatting, error handling, concurrency, and testing patterns.
applyTo: "**/*.go"
---

# General Go Coding Conventions

## Formatting
- Run `gofmt` (or `goimports`) on every file — non-negotiable; all Go code must be gofmt-clean
- Use `goimports` to organize and auto-clean imports (replaces `gofmt`)
- Tabs for indentation (enforced by gofmt)
- No parentheses around `if`/`for`/`switch` conditions
- Opening brace on the same line as the statement

## Naming
- Packages: lowercase, single word, no underscores, no mixedCaps (e.g. `bufio`, `httputil`)
- Exported names: `MixedCaps`; unexported: `mixedCaps`; never use `snake_case`
- Initialisms stay uniformly cased: `URL`, `HTTP`, `ID` — never `Url`, `Http`, `Id`
- Getters omit "Get": `Owner()` not `GetOwner()`; setters use `SetOwner()`
- One-method interfaces: method name + "-er" suffix (`Reader`, `Writer`, `Stringer`)
- Method receivers: 1–2 letter abbreviation of the type (`f *File`), never `self` or `this`
- Short names for local scope; more descriptive for package-level exports
- Avoid stutter: don't repeat the package name in exported identifiers (`bufio.Reader` not `bufio.BufReader`)

## Error Handling
- Always handle errors explicitly — never discard with `_` unless intentional and commented
- Guard early: check error immediately after the call, return or handle before continuing
- Error strings: lowercase, no ending punctuation (unless proper noun) — they are often wrapped
- Sentinel errors: `var ErrNotFound = errors.New("not found")` for package-level errors
- Custom error types: implement `error` interface; use `errors.Is` / `errors.As` for unwrapping
- Reserve `panic` for unrecoverable initialization failures — not for normal control flow
- Use `defer` for resource cleanup (`f.Close()`, `mu.Unlock()`, `cancel()`)

## Interfaces
- Define interfaces in the consumer package, not the provider package
- Keep interfaces small: 1–2 methods is ideal; compose larger interfaces from small ones
- Don't define interfaces prematurely "just in case" — add them when you have ≥2 implementations or a clear testability need
- Use the "comma ok" idiom for type assertions: `v, ok := x.(T)`

## Concurrency
- "Don't communicate by sharing memory; share memory by communicating"
- Pass `context.Context` as the first parameter of any function that may block or be cancelled
- Never store `context.Context` in a struct field
- Make goroutine exit conditions explicit — prevent goroutine leaks
- Use `go test -race` to detect data races; fix all races before merging
- Prefer channels for signalling; use `sync` primitives (Mutex, RWMutex, WaitGroup) for shared state

## Zero Values and Allocation
- Design types so their zero value is valid and usable (e.g. `var mu sync.Mutex`, `var buf bytes.Buffer`)
- Use `new(T)` for zero-initialized pointer allocation; `make` for slices, maps, channels
- Pre-allocate slices when capacity is known: `make([]T, 0, n)`
- Use `var s []string` (nil slice) over `s := []string{}` when starting empty
- "Comma ok" for map lookups: `v, ok := m[k]`

## Documentation
- All exported symbols must have a doc comment starting with the symbol name
- Package comment goes above the `package` declaration (or in `doc.go` for large packages)
- Comments are complete sentences; they appear in `go doc` output

## Testing
- Table-driven tests are the standard pattern
- Test files: `*_test.go`, same package or `package foo_test` for black-box tests
- Test function names: `TestFunctionName`, `TestFunctionName_scenario`
- Benchmark functions: `BenchmarkFunctionName`
- Failure messages must include input, got, and want: `t.Errorf("Foo(%q) = %v, want %v", input, got, want)`
- Run `go test -race ./...` in CI

## Common Pitfalls — Must Avoid
- Ignoring returned errors
- Using `panic` for normal error paths
- Storing `context.Context` in structs
- Goroutine leaks (no cancellation path)
- Using `math/rand` for cryptographic purposes (use `crypto/rand`)
- Defining interfaces in the provider package
- Inconsistent receiver types (mix of pointer and value receivers on the same type)
- Named return values used as implicit documentation (use only when they genuinely clarify)
