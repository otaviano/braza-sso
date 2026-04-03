---
name: react-conventions
description: React coding conventions based on official React docs (react.dev). Component purity, hooks rules, state management, effects, props, performance, and TypeScript patterns.
applyTo: "**/*.{jsx,tsx}"
---

# General React Coding Conventions

## Component Design — Purity First
- Components must be pure functions: same props/state/context → same JSX output, every time
- Never mutate objects that existed before the component was called during render
- Side effects belong in event handlers or `useEffect` — never in the render body
- Local mutations *during* render are fine (building arrays, computing values)
- One responsibility per component; decompose when a component grows complex

## Naming
- Component names: `PascalCase` — React uses this to distinguish components from HTML elements
- Files: match the component name (`Button.tsx`, `UserCard.tsx`)
- Custom hooks: must start with `use` (`useAuth`, `useFetchOrders`)
- Event handler props: `onX` convention (`onClick`, `onSubmit`, `onValueChange`)
- Boolean props: `is`, `has`, `can` prefix (`isLoading`, `hasError`, `canSubmit`)

## State — Minimal and Intentional
- Store only the minimum state needed — apply DRY aggressively
- Never store derived values in state; compute them during render:
  ```tsx
  // Bad
  const [fullName, setFullName] = useState(`${firstName} ${lastName}`);
  // Good
  const fullName = `${firstName} ${lastName}`;
  ```
- Lift state to the closest common ancestor of all components that need it
- One-way data flow: props down, callback functions up (inverse data flow)
- Use `key` prop to reset component state on identity change — not an Effect

## Rules of Hooks — Non-Negotiable
- Call hooks only at the **top level** of a component or custom hook — never inside loops, conditions, or nested functions
- Call hooks only from React components or other hooks — never from plain functions
- Enable and never suppress the `react-hooks/exhaustive-deps` ESLint rule — missing deps hide real bugs

## `useEffect` — Only for External Synchronization
Use `useEffect` only to synchronize with something **external** to React (APIs, WebSockets, DOM APIs, timers).

| Don't use `useEffect` for | Use instead |
|---|---|
| Transforming data for rendering | Compute during render |
| Handling user events | Event handler functions |
| Expensive calculations | `useMemo` |
| Resetting state when a prop changes | `key` prop |
| Adjusting state based on other state | Calculate during render |

- Always declare all reactive values in the dependency array
- Always return a cleanup function for subscriptions and timers
- Ask: "Why does this code need to run?" → User clicked? → event handler. Staying in sync? → Effect.

## `useMemo` / `useCallback` — Profile Before Using
- `useMemo`: cache expensive calculations across renders
- `useCallback`: stabilize function references passed as props to memoized children
- Do not add these speculatively — profile first to confirm a real performance problem
- Wrap child components with `React.memo` only when prop stability is proven to matter

## Props and Composition
- Props are immutable snapshots — never mutate them
- Prefer composition (`children`, render props, component slots) over configuration props
- Avoid inline object/function literals in JSX props — they create new references every render:
  ```tsx
  // Bad — new object every render
  <Chart options={{ color: 'red' }} />
  // Good
  const options = useMemo(() => ({ color: 'red' }), []);
  <Chart options={options} />
  ```
- Always provide a unique, stable `key` when rendering lists — never use array index as key for dynamic lists

## TypeScript
- Type all props with an `interface` or `type`:
  ```tsx
  interface ButtonProps {
    label: string;
    onClick: () => void;
    disabled?: boolean;
  }
  ```
- Prefer `interface` for props; `type` for unions, tuples, or computed types
- Type event handlers explicitly: `(e: React.ChangeEvent<HTMLInputElement>) => void`
- Avoid `any`; use `unknown` and narrow with type guards
- Use `React.FC` sparingly — prefer plain typed function signatures
- Use `ComponentPropsWithoutRef<'button'>` to extend native element props safely

## Accessibility
- Always use semantic HTML elements (`<button>`, `<nav>`, `<main>`, `<label>`)
- Associate `<label>` with inputs via `htmlFor`/`id`
- Provide `alt` text on all `<img>` elements
- Manage focus explicitly when showing/hiding modals or dialogs
- Test with a keyboard and a screen reader before shipping

## Common Pitfalls — Must Avoid
- Mutating state directly (`state.items.push(x)`) — always produce new objects/arrays
- Side effects in the render body
- Calling hooks conditionally or inside loops
- Storing computed values in state
- Using array index as `key` in dynamic lists
- Suppressing `exhaustive-deps` lint warnings
- Fetching data inside `useEffect` without a cleanup / abort controller
- Infinite effect loops (effect updates state → triggers re-render → effect runs again)