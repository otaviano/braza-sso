---
name: golang-project-structure
description: Go project layout, package organization, dependency injection, and module conventions based on official Go guidelines.
applyTo: "**/*.{go,mod,sum}"
---

# Go Project Structure & Package Organization

## Module and Package Layout

```
myapp/
├── go.mod
├── go.sum
├── main.go              # or cmd/myapp/main.go for multi-binary repos
├── cmd/
│   └── myapp/
│       └── main.go      # entry point; wires dependencies (composition root)
├── internal/            # packages not importable by external modules
│   ├── domain/          # pure business logic; no external dependencies
│   ├── service/         # orchestration; depends on domain + ports
│   └── repository/      # data access implementations
├── pkg/                 # packages safe for external import (use sparingly)
├── handler/             # HTTP handlers / gRPC servers
└── config/              # configuration loading
```

## Package Rules
- One coherent responsibility per package — if you can't name it without "and", split it
- `internal/` for packages that must not be imported by other modules
- Keep `main.go` (or `cmd/`) thin: only wiring (dependency injection), no business logic
- Avoid circular imports — they are a compile error and a design smell
- Group imports: standard library → third-party → internal (blank line between each group)

## Dependency Injection
- Wire dependencies at the composition root (`main.go` or `cmd/`)
- Business logic receives dependencies via constructor parameters — never calls `new` on concrete infra types
- Define interfaces (ports) inside the package that uses them, not the package that implements them
- Use constructor functions (`NewService(...)`) to make dependencies explicit

## Configuration
- Load configuration once at startup; pass it down as a struct
- Never read `os.Getenv` inside business logic — inject config as a dependency
- Validate configuration at startup and fail fast with a clear error message

## HTTP / gRPC Layer
- Handlers are thin: decode request → call service → encode response
- No business logic in handlers
- Use `context.Context` propagation from request to service to repository

## Error Flow
- Return errors up the call stack; add context with `fmt.Errorf("doing X: %w", err)`
- Only handle (log + stop propagating) errors at the top of the call stack (handler layer)
- Never log and return the same error (double logging)

## Testing Layout
```
internal/
├── service/
│   ├── order.go
│   └── order_test.go    # unit tests alongside implementation
└── repository/
    ├── order.go
    └── order_test.go
```
- Integration/e2e tests in a separate top-level `test/` directory if needed
- Use `testify` or standard `testing` package; table-driven tests are idiomatic
- Mock interfaces (defined in consumer packages) with `mockery` or hand-written fakes

## go.mod Conventions
- Module path should match the repository URL (e.g. `github.com/org/myapp`)
- Pin Go version explicitly: `go 1.22`
- Keep dependencies tidy with `go mod tidy` before committing
