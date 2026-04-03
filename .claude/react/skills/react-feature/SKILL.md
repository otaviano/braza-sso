---
name: react-feature
description: Workflow skill for implementing new features in a React + TypeScript project following react.dev best practices. Use when adding new components, hooks, pages, API integrations, or UI flows.
---

# React Feature Implementation Skill

This skill provides a step-by-step workflow for implementing new features in a React + TypeScript project following official React best practices.

## Invocation
Use this skill when adding new functionality. Examples:
- _"add a page to list orders with filtering"_
- _"implement a form to create a new user"_
- _"add a custom hook to fetch product data"_
- _"create a reusable Modal component"_

## When to Use
- Adding new pages or routes
- Building new UI components
- Implementing data fetching hooks
- Creating forms with validation
- Adding shared/reusable components to the design system

---

## Workflow: Implement a New Feature

### 1. Define Types in `src/types/` or `features/<name>/`
- Define request/response shapes and domain types before writing components
- Use `interface` for object shapes; `type` for unions and aliases

```tsx
// features/orders/types.ts
export interface Order {
  id: string;
  customerId: string;
  status: 'pending' | 'shipped' | 'delivered';
  createdAt: string;
}

export interface CreateOrderPayload {
  customerId: string;
  items: OrderItem[];
}
```

### 2. Create the API Layer in `features/<name>/api/`
- Plain async functions — no React, no hooks
- All functions are typed; errors surface as thrown exceptions

```tsx
// features/orders/api/ordersApi.ts
import { httpClient } from '@/lib/httpClient';
import type { Order, CreateOrderPayload } from '../types';

export async function fetchOrders(): Promise<Order[]> {
  const response = await httpClient.get('/orders');
  return response.data;
}

export async function createOrder(payload: CreateOrderPayload): Promise<Order> {
  const response = await httpClient.post('/orders', payload);
  return response.data;
}
```

### 3. Create Data-Fetching Hooks in `features/<name>/hooks/`
- Use React Query (`useQuery`, `useMutation`) to manage server state
- Never raw `useEffect` + `useState` for fetching

```tsx
// features/orders/hooks/useOrders.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { fetchOrders, createOrder } from '../api/ordersApi';

export function useOrders() {
  return useQuery({ queryKey: ['orders'], queryFn: fetchOrders });
}

export function useCreateOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: createOrder,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['orders'] }),
  });
}
```

### 4. Build Components in `features/<name>/components/`
- One component per file; pure by default
- Derive all display values during render — no derived state
- Compose with smaller components; keep JSX readable

```tsx
// features/orders/components/OrderList.tsx
import { useOrders } from '../hooks/useOrders';
import { OrderCard } from './OrderCard';

export function OrderList() {
  const { data: orders, isLoading, isError } = useOrders();

  if (isLoading) return <p>Loading...</p>;
  if (isError) return <p>Failed to load orders.</p>;

  return (
    <ul>
      {orders?.map((order) => (
        <OrderCard key={order.id} order={order} />
      ))}
    </ul>
  );
}
```

### 5. Build Forms with Controlled Inputs
- Use `react-hook-form` for non-trivial forms — avoids re-render on every keystroke
- Validate with `zod` schema; integrate via `@hookform/resolvers/zod`
- Map domain exceptions to user-facing error messages in the `onError` handler

```tsx
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';

const schema = z.object({ customerId: z.string().min(1, 'Required') });
type FormValues = z.infer<typeof schema>;

export function CreateOrderForm() {
  const { register, handleSubmit, formState: { errors } } = useForm<FormValues>({
    resolver: zodResolver(schema),
  });
  const { mutate, isPending } = useCreateOrder();

  return (
    <form onSubmit={handleSubmit((data) => mutate(data))}>
      <label htmlFor="customerId">Customer ID</label>
      <input id="customerId" {...register('customerId')} />
      {errors.customerId && <span>{errors.customerId.message}</span>}
      <button type="submit" disabled={isPending}>Create</button>
    </form>
  );
}
```

### 6. Add the Page Component and Register Route
- Page component is thin: composes feature components, sets page title
- Register in `app/router.tsx` with `React.lazy` for code splitting

```tsx
// features/orders/OrdersPage.tsx
export function OrdersPage() {
  return (
    <main>
      <h1>Orders</h1>
      <OrderList />
    </main>
  );
}

// app/router.tsx
const OrdersPage = React.lazy(() =>
  import('../features/orders/OrdersPage').then((m) => ({ default: m.OrdersPage }))
);
```

### 7. Export Public API via `features/<name>/index.ts`
```ts
// features/orders/index.ts
export { OrdersPage } from './OrdersPage';
export { useOrders, useCreateOrder } from './hooks/useOrders';
export type { Order } from './types';
```

### 8. Write Tests

**Component tests** with React Testing Library — test behavior, not implementation:
```tsx
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CreateOrderForm } from './CreateOrderForm';

test('shows validation error when customer ID is empty', async () => {
  render(<CreateOrderForm />);
  await userEvent.click(screen.getByRole('button', { name: /create/i }));
  expect(screen.getByText('Required')).toBeInTheDocument();
});
```

**Hook tests** with `renderHook` from React Testing Library.

**API function tests** with `vitest` + `msw` (Mock Service Worker) for network mocking.

---

## Checklist Before Submitting

- [ ] No side effects in the render body
- [ ] No derived values stored in state
- [ ] All `useEffect` dependencies declared (no suppressed lint warnings)
- [ ] All list items have a stable, unique `key` (not array index)
- [ ] All props typed with `interface` / `type`
- [ ] No `any` in TypeScript
- [ ] Semantic HTML used (`<button>`, `<label htmlFor>`, `<main>`)
- [ ] Component is accessible (keyboard navigable, has `alt` / `aria-*` where needed)
- [ ] Feature exported only through `index.ts`
- [ ] Tests cover happy path and key error states
- [ ] `eslint` passes with no warnings
- [ ] `tsc --noEmit` passes
