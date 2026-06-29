# Backend Design Patterns (Go)

## 1. Module Architecture

Each domain is a self-contained package with its own entities, persistence, business logic, and transport layer. This structure enables individual modules to be extracted or replaced without affecting others.

The modules listed below are the initial set identified from the architecture. This layout is not exhaustive — new modules can be added as the system grows and existing modules can be split further when their complexity justifies it. Similarly, sub-packages like `transport/` appear here as a convention; not every module needs one initially.

```
internal/
  user/              # User accounts and admin management
    user.go          # User entity
    repository.go    # Repository interface + SQLite implementation
    service.go       # Business logic (CRUD, admin operations)
    transport/       # HTTP handlers
      handler.go
      routes.go      # RegisterRoutes(router, svc)
  auth/              # Authentication and session management
    session.go       # Session entity
    repository.go    # Session store interface + SQLite implementation
    service.go       # Login, logout, session validation
    middleware.go    # Auth middleware (cookie validation, role check)
    transport/
      handler.go
      routes.go
  agent/             # Terminal agent lifecycle
    agent.go         # Agent entity, creation options
    repository.go    # Repository interface + SQLite implementation
    service.go       # Orchestrator logic (create, destroy, list)
    transport/
      handler.go
      routes.go
  session/           # Active terminal sessions (in-memory)
    session.go       # Active session entity
    manager.go       # In-memory SessionManager, ring buffer
    transport/
      handler.go     # WebSocket handler (connect, reconnect)
      routes.go
  provider/          # Provider interface and registry
    interface.go     # Provider interface
    registry.go      # In-memory registry
    discovery.go     # Health-check scanning
    docker/          # Docker provider implementation
      provider.go
    firecracker/     # Firecracker provider implementation
      provider.go
  shared/            # Shared kernel
    config.go        # Configuration loading
    db.go            # SQLite connection, pragmas, migration runner
    errors.go        # Sentinel errors (ErrNotFound, ErrUnauthorized, etc.)
```

## 2. Dependency Direction

Dependencies flow inward. No cyclic dependencies are permitted.

```
transport → service → domain (entities + repository interface)
                           ← infrastructure (repository impl)
```

- `transport/` depends on `service` — calls use cases, validates input, maps errors to HTTP status codes
- `service` depends on `repository` interface — pure business logic, no HTTP or DB concerns
- `repository` implementation (same package) depends on `shared/db.go` for SQLite access
- `provider/` is consumed by `agent/` and `session/` via interface only
- Cross-module calls go through interfaces, never concrete types

## 3. Cross-Module Communication

Constructor injection is the only wiring mechanism. No global state, no service locators, no `init()`.

```go
// agent/service.go
type Service struct {
    repo     Repository
    registry *provider.Registry
}

func NewService(repo Repository, registry *provider.Registry) *Service {
    return &Service{repo: repo, registry: registry}
}
```

A central `main.go` or `wire.go` wires all dependencies explicitly:

```go
func main() {
    db := shared.OpenDB(cfg)
    userRepo := user.NewRepository(db)
    userSvc := user.NewService(userRepo)
    authRepo := auth.NewRepository(db)
    authSvc := auth.NewService(authRepo)
    // ... wire remaining modules ...
    user.RegisterRoutes(mux, userSvc)
    auth.RegisterRoutes(mux, authSvc)
    // ...
}
```

Interfaces are defined in the consuming package — the producer imports the interface, not the other way around.

```go
// agent/repository.go — defines what agent/service.go needs
type Repository interface {
    FindByID(ctx context.Context, id int64) (*Agent, error)
    Save(ctx context.Context, a *Agent) error
    Delete(ctx context.Context, id int64) error
    CountByUser(ctx context.Context, userID int64) (int, error)
}

// agent/repository.go — SQLite implementation in the same file or package
type SQLiteRepository struct {
    db *sql.DB
}
```

## 4. Route Registration

Each `transport/` package exports a single function to register its routes:

```go
type routeRegistrar interface {
    Handle(pattern string, handler http.Handler)
}

func RegisterRoutes(router routeRegistrar, svc *Service, mw ...func(http.Handler) http.Handler) {
    h := &Handler{svc: svc}
    router.Handle("GET /api/agents", middleware.Chain(mw, h.List))
    router.Handle("POST /api/agents", middleware.Chain(mw, h.Create))
    router.Handle("DELETE /api/agents/{id}", middleware.Chain(mw, h.Delete))
}
```

The concrete router is chosen by the application entry point — any router that satisfies the `Handle(pattern, handler)` contract works (standard library, chi, gorilla/mux, etc.). The `transport/` package remains decoupled from the router implementation.

The `main.go` calls `RegisterRoutes` for each module, passing the auth middleware chain:

```go
authMW := auth.RequireSession(authSvc)
adminMW := auth.RequireAdmin(authSvc)

agent.RegisterRoutes(router, agentSvc, authMW)
session.RegisterRoutes(router, sessionMgr, authMW)
```

## 5. Strategic Interfaces

Every interface in the codebase must satisfy one of these three criteria (per the project's abstraction rules):

| Criterion | Example |
|---|---|
| **Domain boundary** | Provider interface separates environment management from orchestration |
| **Supporting testing** | Repository interfaces allow mock-based unit tests |
| **Multiple implementations** | Docker and Firecracker both implement Provider |

Concretely:

| Interface | Defined in | Implementations | Justification |
|---|---|---|---|
| `provider.Provider` | `provider/interface.go` | Docker, Firecracker | Domain boundary + multiple impls |
| `agent.Repository` | `agent/repository.go` | SQLite, mock | Testing |
| `session.Repository` | `session/repository.go` | SQLite, mock | Testing |
| `auth.Repository` | `auth/repository.go` | SQLite, mock | Testing |
| `user.Repository` | `user/repository.go` | SQLite, mock | Testing |

Repository interfaces are not extracted to a separate package — they live alongside the entity they persist. This keeps the module cohesive and avoids a leaky `interfaces/` bucket.

## 6. Error Handling

### Domain-level errors

Defined in `shared/errors.go` as sentinel errors:

```go
var (
    ErrNotFound      = errors.New("not found")
    ErrUnauthorized  = errors.New("unauthorized")
    ErrForbidden     = errors.New("forbidden")
    ErrConflict      = errors.New("conflict")
    ErrCapReached    = errors.New("cap limit reached")
    ErrValidation    = errors.New("validation error")
)
```

### Module-specific errors

Each module can define its own errors:

```go
// agent/agent.go
var ErrAgentNotRunning = errors.New("agent not running")
```

### Propagation

- Service layer wraps errors with context: `fmt.Errorf("create agent: %w", err)`
- Transport layer maps domain errors to HTTP status codes in a single `respondError` helper:

```go
func respondError(w http.ResponseWriter, err error) {
    switch {
    case errors.Is(err, shared.ErrNotFound):
        http.Error(w, err.Error(), http.StatusNotFound)
    case errors.Is(err, shared.ErrUnauthorized):
        http.Error(w, err.Error(), http.StatusUnauthorized)
    case errors.Is(err, shared.ErrForbidden):
        http.Error(w, err.Error(), http.StatusForbidden)
    case errors.Is(err, shared.ErrCapReached):
        http.Error(w, err.Error(), http.StatusTooManyRequests)
    case errors.Is(err, shared.ErrValidation):
        http.Error(w, err.Error(), http.StatusBadRequest)
    default:
        log.Printf("unexpected error: %v", err)
        http.Error(w, "internal error", http.StatusInternalServerError)
    }
}
```

## 7. Testing Patterns

| Layer | Approach |
|---|---|
| **Domain** | Pure unit tests — no dependencies, no interfaces, no mocks |
| **Service** | Interface-based — mock the repository interface with `minimock` or hand-written stubs |
| **Repository** | Integration tests against a real SQLite in-memory database |
| **Transport** | `net/http/httptest` with mocked service; verify status codes, response bodies, header values |

Repository interfaces are designed for mocking — each method takes a `context.Context` as the first parameter and returns concrete types, never `interface{}`.

### Test file placement

Tests live alongside the code they test, following Go convention:

```
internal/
  agent/
    agent_test.go
    service_test.go
    repository_test.go
    transport/
      handler_test.go
```

No separate `_test` package suffix is required — `white-box` testing is preferred for coverage of unexported helpers, except in `transport/` where `black-box` (`agent_test`) may be used to test only the exported `RegisterRoutes` and handler behavior.

## 8. Replaceability & Future Scaling

The module structure guarantees replacement without spillover:

| Replacement | Surface area | Other modules affected |
|---|---|---|
| SQLite → Postgres | `agent/repository.go`, `auth/repository.go`, `user/repository.go` | None — all callers depend on the interface |
| In-memory sessions → Redis | `session/manager.go` behind a `SessionStore` interface | None |
| Docker → Podman | New `provider/podman/` package, register in `main.go` | None |
| HTTP → gRPC | Replace `transport/` package only | None |

Each module exposes only a `Service` struct with exported methods and a `RegisterRoutes` function. All internal details (repository implementation, transport encoding, private helpers) are unexported and invisible to other modules.
