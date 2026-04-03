---
name: solid enforcer
description: SOLID principles enforcer for generation, review, and refactoring. Ensures code is extensible, testable, and maintainable through SRP, OCP, LSP, ISP, and DIP.
applyTo: "**/*.{cs,go,ts,tsx,js,jsx,py}"
---

# SOLID Enforcer (Generation + Review + Refactor)

## Purpose
Ensure produced or modified code adheres to SOLID, with designs that are extensible, testable, and maintainable. This rule applies to:

- Generate new code that is SOLID by construction
- Refactor/review existing code to become more SOLID without breaking behavior

## When to Activate
Activate when the task includes any of:

- refactor / redesign / architecture changes
- "make it SOLID", "clean up responsibilities", "reduce coupling"
- adding extension points / plugin systems / strategies / policies
- replacing large conditionals or type switches
- improving testability, DI, interface boundaries, modularity
- PR review focused on design/maintainability

## Operating Mode (must follow)

1. **Clarify** (max 1–3 questions) if requirements or constraints are ambiguous (behavioral invariants, extension expectations, performance constraints, public API stability).

2. **Plan first** for non-trivial changes:
   - identify files/modules to touch
   - state invariants and "must not change" behaviors
   - propose refactor steps with checkpoints (tests/typing/lint)

3. **Implement in small, verifiable steps:**
   - keep diffs tight
   - preserve stable logic
   - add/adjust tests as proof

4. **Prove correctness:**
   - add unit tests demonstrating contracts
   - run/describe the minimal commands to validate (tests, typecheck, lint)

5. **Return complete output:**
   - runnable files (or clear patches)
   - tests
   - short SOLID rationale (how SRP/OCP/LSP/ISP/DIP are satisfied)

## SOLID Baseline Constraints (always enforce)

- Prefer composition over inheritance unless full substitutability is guaranteed.
- Keep classes/modules small; if a unit grows rapidly, challenge responsibilities.
- Make dependencies explicit (constructor/params). Avoid `new` in business logic; use DI/factories.
- Return complete, runnable files and tests proving the contract.

## Principle Checklists

### SRP — Single Responsibility
**Pass if:**
- Each class/module has one reason to change.
- Logging/validation/error-handling are separated from domain logic.
- Methods do one thing; if a description needs "and/or", split it.

**Common fixes:**
- Extract validation into validators/policies
- Extract I/O (HTTP/DB/files) into adapters/clients
- Split orchestration (use-case/service) from pure domain objects

---

### OCP — Open/Closed
**Pass if:**
- New behaviors can be added by adding new implementations, not editing stable logic.
- Replace type-based switches with polymorphism (strategy/decorator/handlers).

**Common fixes:**
- Strategy pattern: Policy / Algorithm interface + implementations
- Decorator: wrap behavior without editing core
- Registration/dispatch tables (map keys → handlers) where appropriate

---

### LSP — Liskov Substitution
**Pass if:**
- Subtypes preserve invariants.
- No tighter preconditions / no looser postconditions.
- No "noop/throw overrides" that violate expectations.

**Common fixes:**
- Split base interface into smaller role interfaces (often pairs with ISP)
- Prefer composition if inheritance requires exceptions
- Add contract tests for substitutability (same test suite for all impls)

---

### ISP — Interface Segregation
**Pass if:**
- Interfaces are minimal and client-specific.
- "Fat" interfaces are split; clients depend only on what they use.

**Common fixes:**
- Break one large interface into role interfaces
- Compose roles into larger façades only where needed

---

### DIP — Dependency Inversion
**Pass if:**
- High-level modules depend on abstractions, not concretions.
- Interfaces are defined near the clients (ports), implementations elsewhere (adapters).
- Construction is pushed to the composition root; business logic receives dependencies.

**Common fixes:**
- Constructor/parameter injection
- Factories/providers passed in (not created inside domain logic)
- Avoid importing concrete implementations into core domain/use-cases
- Avoid instantiating clients inside use-case functions; inject them

---

## Refactor Playbook (preferred approach)

1. **Identify responsibilities and seams:** what is domain logic vs infrastructure vs orchestration?
2. **Introduce abstractions only where they enable extension/testability:** define small interfaces (ports) at the boundary
3. **Replace conditionals with strategies/handlers:** keep stable logic unchanged; move variability behind an interface
4. **Push object creation outward:** create a composition root (factory/module) for wiring
5. **Lock behavior with tests before/after:** contract tests for interfaces, regression tests for prior behavior

## Output Contract (what to return)

Always return:

- Code + unit tests that demonstrate the contract.
- A short rationale explicitly answering:
  - **SRP:** what responsibilities were separated?
  - **OCP:** what is now extensible without edits to stable logic?
  - **LSP:** how substitutability is preserved (or why inheritance was avoided)?
  - **ISP:** how interfaces were slimmed/split?
  - **DIP:** what dependencies are injected and where is wiring done?
- If relevant: a minimal example showing how to extend via a new implementation.

## Red Flags (must call out and fix)

- Business logic instantiates concretes (`new`, direct client creation) rather than receiving deps.
- Large modules/classes growing in responsibilities.
- Type switches/if-chains driving behavior selection across the codebase.
- Inheritance hierarchies with special-case overrides or exceptions.
