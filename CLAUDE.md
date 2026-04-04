# Braza SSO — Claude Instructions

## Project Overview

Self-hosted Identity Provider: Go 1.25 API + React 19/Vite/TypeScript frontend.
Stack: chi/v5, zerolog, gocql (Cassandra), go-redis/v9, pquerna/otp, golang-jwt/jwt/v5, golang.org/x/oauth2.
Module: `github.com/otaviano/braza-sso`.

## Non-Negotiable Rules

### Git
- Always use `main` as the default branch. Never `master`.
- Make **atomic commits per phase/feature**. Never bundle multiple unrelated changes in one commit.

### SOLID Principles (`**/*.{go,ts,tsx}`)

**SRP** — Every type/function has one reason to change. Split if it does two things.  
**OCP** — Extend via interfaces and composition, not by modifying existing types.  
**LSP** — Subtypes/implementations must be substitutable without breaking callers.  
**ISP** — Define small, role-specific interfaces per caller. No fat interfaces.  
**DIP** — Depend on abstractions (interfaces), not concrete types. Inject dependencies via constructors.

### Clean Code — Uncle Bob (`**/*.{go,ts,tsx}`)

- Functions ≤ 20 lines. If longer, extract.
- Names are self-documenting. No abbreviations (`usr`, `ctx2`, `tmp`).
- No magic numbers or magic strings — use named constants.
- One level of abstraction per function.
- No dead code, commented-out code, or TODO left in production files.
- Tests are first-class: every public behaviour has a test.

### Object Calisthenics (`**/*.{go,ts,tsx}`)

1. **One level of indentation per function** — extract inner blocks to named functions.
2. **No `else`** — use early returns, guard clauses.
3. **Wrap all primitives** that have behaviour (e.g., `Email`, `Password`, `UserID` types).
4. **First-class collections** — a type that holds a collection does nothing else.
5. **One dot/method call per line** — no chaining beyond one level.
6. **No abbreviations** — full, meaningful names everywhere.
7. **Keep entities small** — files < 50 lines, packages < 10 files where possible.
8. **No types with more than 2 instance variables** — decompose into smaller structs.
9. **No getters/setters** — expose behaviour, not data.

> These are enforced, not guidelines. If a change would violate them, refactor first.

## Go Conventions

- Error handling: check every error, wrap with `fmt.Errorf("context: %w", err)`.
- No `panic` in library code — return errors.
- Use `context.Context` as the first parameter for all IO-bound functions.
- Prefer table-driven tests with `t.Run`.
- Use interfaces for all external dependencies (DB, cache, email, clock).

## TypeScript/React Conventions

- Functional components only. No class components.
- Props interfaces defined inline in the same file, named `Props`.
- No `any`. Use `unknown` + type guards if type is truly unknown.
- `api.ts` is the single fetch abstraction — never call `fetch` directly in components.

## Commit Style

```
<type>(<scope>): <short description>

Types: feat, fix, chore, test, refactor, docs
Example: feat(auth): implement TOTP enrollment endpoint
```

## Notion Tracking

- Database ID: `337f85ec-d527-8042-a96a-e43302870699`
- Move phase cards to **In progress** when starting, **Complete** when done.
