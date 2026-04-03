---
name: golang-feature
description: Workflow skill for implementing new features in a Go project following idiomatic Go conventions, Clean Architecture, and SOLID principles. Use when adding new use cases, handlers, services, repositories, or domain logic.
---

# Go Feature Implementation Skill

This skill provides a step-by-step workflow for implementing new features in a Go project following idiomatic Go practices.

## Invocation
Use this skill when adding new functionality. Examples:
- _"add an endpoint to create an order"_
- _"implement a service to send notifications"_
- _"add a repository method to fetch users by email"_

## When to Use
- Adding new business logic or use cases
- Creating new HTTP/gRPC handlers
- Implementing repository methods
- Defining new domain types or value objects
- Writing new service layer orchestration

---

## Workflow: Implement a New Feature

### 1. Define the Domain Types (if needed) in `internal/domain/`
- Use structs for entities; design zero values to be valid where possible
- Validate invariants inside constructor functions (`NewOrder(...)`)
- Keep domain types free of external dependencies (no DB, no HTTP)
- Return `error` from constructors for invalid input

### 2. Define the Port (Interface) in the Consumer Package
- Define the interface in the package that _uses_ it, not the one that implements it
- Keep it small — only the methods this consumer actually needs
  ```go
  // internal/service/order.go
  type OrderRepository interface {
      Save(ctx context.Context, order domain.Order) error
      FindByID(ctx context.Context, id string) (domain.Order, error)
  }
  ```

### 3. Implement the Service in `internal/service/`
- Accept dependencies via constructor: `func NewOrderService(repo OrderRepository) *OrderService`
- Accept `context.Context` as the first parameter on all methods
- Add context with `fmt.Errorf("service.CreateOrder: %w", err)` when wrapping errors
- No direct calls to `new` on infra types inside service logic

### 4. Implement the Repository / Adapter in `internal/repository/`
- Implement the interface defined in the service package
- Map between domain types and DB/external models here
- All DB calls receive `context.Context`
- Wrap infra errors: `fmt.Errorf("repository.Save: %w", err)`

### 5. Add the Handler in `handler/` (HTTP) or `server/` (gRPC)
- Decode incoming request → call service → encode response
- No business logic in handlers
- Handle errors by mapping to appropriate HTTP status / gRPC status codes
  ```go
  if errors.Is(err, domain.ErrNotFound) {
      http.Error(w, "not found", http.StatusNotFound)
      return
  }
  ```
- Use `r.Context()` and pass it to the service

### 6. Wire in `cmd/` / `main.go`
- Instantiate dependencies bottom-up: DB → repository → service → handler
- Register routes / gRPC methods
- No business logic here — composition only

### 7. Write Tests

**Unit test the service** (mock the repository interface):
```go
func TestOrderService_Create(t *testing.T) {
    tests := []struct {
        name    string
        input   domain.CreateOrderRequest
        wantErr bool
    }{
        {"valid order", validRequest, false},
        {"missing item", emptyRequest, true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            repo := &mockOrderRepository{}
            svc := NewOrderService(repo)
            _, err := svc.Create(context.Background(), tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

**Integration test the repository** against a real (test) DB when possible.

**Handler tests** using `httptest.NewRecorder()` and `httptest.NewServer()`.

---

## Checklist Before Submitting

- [ ] `gofmt` / `goimports` clean
- [ ] All exported symbols have doc comments
- [ ] No errors discarded with `_`
- [ ] `context.Context` passed through all layers
- [ ] Interface defined in the consumer package
- [ ] Table-driven tests cover happy path and error cases
- [ ] `go test -race ./...` passes
- [ ] `go vet ./...` passes
- [ ] `go mod tidy` run if dependencies changed
