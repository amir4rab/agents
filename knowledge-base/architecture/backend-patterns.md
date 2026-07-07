# Backend Design Patterns (Go)

## 1. Module Architecture

The codebase uses a flat model structure under `internal/model/`. Each domain
is represented by a single package containing its entities, persistence,
business logic, and transport layer.

```
internal/
  model/
    auth/              # Session entity
    provider/          # Provider interface and kind enum
    space/             # Space entity (terminal-agent environments)
    user/              # User accounts and admin management
    internal/
      password/        # Argon2id password hashing
      sqlite/          # SQLite connection setup
      types/           # Shared domain types (Repository, CursorPagination, Optional)
```

Domains are self-contained packages. New domains can be added as siblings
under `model/`.

## 2. Dependency Direction

Dependencies are hierarchical — shared types flow downward, domain packages
depend only on `internal/` for infrastructure and types.

```
model/{auth,provider,space,user} -> model/internal/{types,sqlite,password}
```

No cyclic dependencies are permitted.

## 3. Cross-Module Communication

Constructor injection is the wiring mechanism. No global state, no service
locators, no `init()`.

```go
type Service struct {
    db   *sql.DB
    user Repository
}

func NewService(sql *sql.DB, repo Repository) *Service {
    return &Service{sql, repo}
}
```

A central `main.go` wires all dependencies explicitly:

```go
func main() {
    dbConfig := sqlite.NewConfig("./data.db")
    db, err := sqlite.Conn(dbConfig)
    userRepo := user.NewRepository()
    userSvc := user.NewService(db, userRepo)
}
```

Interfaces are defined alongside the code that implements them. The generic
`Repository` interface is defined in `model/internal/types/repository.go` and
imported by each domain.

## 4. Repository Pattern

The codebase uses a generic repository interface with **transaction injection**.
All repository methods receive `*sql.Tx`, never `*sql.DB`. The service layer
manages transactions explicitly.

Each domain aliases the generic repository to its concrete types:

```go
type Repository = domaintypes.Repository[User, int64, Query]
```

### Transaction-per-operation pattern

Every service method opens its own transaction, defers rollback, and commits
on success:

```go
func (s *Service) Get(ctx context.Context, id int64) (*User, error) {
    tx, err := s.db.BeginTx(ctx, readonlyTx)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()

    user, err := s.user.Get(tx, id)
    if err != nil {
        return nil, err
    }

    if err = tx.Commit(); err != nil {
        return nil, err
    }
    return user, nil
}
```

This pattern is consistent across all operations: Create, Get, Update, Delete,
and Find.

### Cursor-based pagination

List queries embed cursor pagination for stable, efficient bidirectional
navigation. Results include page metadata (start cursor, end cursor, next/previous
page presence).

## 5. Error Handling

Error handling is ad-hoc at this stage. Domain packages define their own errors
as needed. Service methods return errors directly; the transport layer maps
them to HTTP status codes.

A shared sentinel errors package is added when the need arises — not before.

## 6. Testing Patterns

| Layer | Approach |
|---|---|
| **Service** | Integration tests against an in-memory SQLite database |
| **Domain** | Pure unit tests for JSON serialization, optional field handling |

Tests live alongside the code they test, following Go convention:

```
internal/model/
  provider/kind_test.go
  user/model_test.go
  user/service_test.go
  internal/types/optional_test.go
  internal/password/create_test.go
  internal/password/compare_test.go
  internal/sqlite/conn_test.go
```

White-box testing is preferred. Repository implementations are tested through
the service layer.

## 7. Scalability & Replaceability

| Replacement | Surface area | Notes |
|---|---|---|
| SQLite -> Postgres | Update repository implementations | Interface stays the same |
| New provider | New package, register at startup | No changes to existing domains |
| HTTP -> gRPC | Add transport layer per domain | No changes to service layer |
