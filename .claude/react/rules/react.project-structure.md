---
name: react-project-structure
description: React project layout, folder conventions, component organization, state architecture, and tooling based on react.dev recommendations and modern ecosystem standards.
applyTo: "**/*.{jsx,tsx,ts,json}"
---

# React Project Structure & Organization

## Recommended Layout

```
src/
├── app/                    # App-level setup: router, providers, global styles
│   ├── App.tsx
│   ├── router.tsx
│   └── providers.tsx
├── components/             # Shared, reusable UI components (design system)
│   ├── Button/
│   │   ├── Button.tsx
│   │   ├── Button.module.css
│   │   └── Button.test.tsx
│   └── Modal/
├── features/               # Feature-sliced: each feature owns its components, hooks, utils
│   └── orders/
│       ├── components/
│       │   └── OrderList.tsx
│       ├── hooks/
│       │   └── useOrders.ts
│       ├── api/
│       │   └── ordersApi.ts
│       └── index.ts        # Public API — only import from here outside the feature
├── hooks/                  # Shared custom hooks used across features
│   └── useDebounce.ts
├── lib/                    # Third-party wrappers, clients, utilities
│   └── httpClient.ts
├── types/                  # Shared TypeScript types and interfaces
│   └── api.ts
└── main.tsx                # Entry point — wires providers and mounts the app
```

## Component File Conventions
- One component per file; file name matches component name (`UserCard.tsx`)
- Co-locate test, styles, and stories with the component:
  ```
  Button/
  ├── Button.tsx
  ├── Button.module.css
  ├── Button.test.tsx
  └── Button.stories.tsx   # if using Storybook
  ```
- Export component as named export, not default, for better refactoring support

## Feature Slice Rules
- Each feature folder is a self-contained vertical slice
- Features import from `components/`, `hooks/`, `lib/` — never from sibling features directly
- Export only the public API through `features/<name>/index.ts`; keep internals private
- Keeps coupling low and makes features independently testable

## State Architecture — Choose the Right Tool

| State type | Tool |
|---|---|
| Local UI state | `useState` / `useReducer` |
| Shared UI state (few components) | Lift to common parent |
| Global UI state (theme, auth, locale) | `Context` + `useContext` |
| Server / async state | React Query / SWR |
| Complex client state | Zustand / Redux Toolkit |

- Default to `useState`; reach for external tools only when local state becomes painful
- Never store server data in global state — use a data-fetching library with caching

## Data Fetching
- Use React Query (`@tanstack/react-query`) or SWR for all server state
- Encapsulate fetch logic in custom hooks inside `features/<name>/hooks/`
- Never fetch directly in component bodies with raw `useEffect` — use a library
- Define API call functions in `features/<name>/api/` (plain async functions, no React)

## Routing
- Use React Router v6 (or Next.js App Router for SSR/SSG projects)
- Define routes in `app/router.tsx`; use lazy loading for page-level components:
  ```tsx
  const OrdersPage = React.lazy(() => import('../features/orders/OrdersPage'));
  ```
- Keep route components thin — delegate to feature components immediately

## Styling
- Prefer CSS Modules (`*.module.css`) for component-scoped styles
- Use Tailwind CSS for utility-first projects — avoid mixing both approaches in one codebase
- Never use inline styles for layout — reserve them for dynamic values only (`style={{ width: `${value}px` }}`)

## Tooling
| Tool | Purpose |
|---|---|
| `vite` | Build tool and dev server |
| `typescript` | Static typing |
| `eslint` + `react-hooks` plugin | Lint hooks rules and code quality |
| `prettier` | Formatting |
| `vitest` + React Testing Library | Unit and integration tests |
| `@tanstack/react-query` | Server state / data fetching |

## `package.json` / Dependency Conventions
- Separate `dependencies` (runtime) from `devDependencies` (build/test tooling)
- Pin exact versions for apps (`"react": "18.3.1"`); use ranges for libraries
- Use `npm ci` or equivalent in CI for reproducible installs