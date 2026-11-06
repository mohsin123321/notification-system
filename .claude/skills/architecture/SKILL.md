---
name: architecture
description: Hexagonal/layered architecture for this Go project. Use whenever creating or editing any module — controllers, services, repositories, domain entities, Wire wiring, or tests.
---

## Request Flow

```
HTTP Request
    ↓ Controller   — gin.Context only; parse request, format response
    ↓ Service      — business logic; calls domain interfaces only
    ↓ Infra        — GORM, Keycloak, Redis; implements domain interfaces
    ↑ domain entity returned up each layer → DTO → HTTP Response
```

## Folder Layout

`internal/` = shared across all projects. `projects/{name}/` = project-specific (same structure).

```
internal/
  domain/{module}/
    pure/entity.go            # Pure Go structs — NO gorm/json/binding tags, no external imports
    port/port.go              # Interfaces only (Repository, IdentityProvider…) — never mention MySQL/Keycloak
    value/value.go             # Value objects (optional)

  infra/
    keycloak/            # Implements domain.IdentityProvider + domain.TokenVerifier
    mysql/repository/    # Implements domain.UserRepository

  app/{module}/          # Flat layout (small modules)
    controller.go        # Defines ServiceIface + Controller struct
    service.go           # Service struct + methods
    dto.go               # Request/Response structs with json + binding tags
    routes.go            # RegisterRoutes(engine *gin.Engine, ctrl *Controller)
    provider.go          # ProviderSet + App struct + Register(engine)

  app/{module}/          # Sub-package layout (large modules)
    controller/controller.go
    service/service.go
    dto/dto.go
    {module}.go          # ProviderSet + wire.Bind + App struct
```

## Rules Per File

| File | Hard rules |
|---|---|
| `entity.go` | No struct tags, no external imports, stdlib only |
| `port.go` | Interfaces only; no Keycloak/MySQL/GORM mentions |
| `repository/*.go` | Unexported GORM model; `toDomain()`; `NewRepo()` returns **interface**, not `*struct` |
| `service.go` | No `*gin.Context`, no `*gorm.DB`; accept/return domain types only |
| `controller.go` | Defines its own `ServiceIface` here (consumer owns the interface) |
| `provider.go` | `wire.NewSet(NewService, NewController, NewApp)` |

## Wire

```go
// Sub-package layout needs wire.Bind (interface + impl in different packages)
wire.Bind(new(controller.SubscriptionServicer), new(*service.Service))

// Flat layout does NOT need wire.Bind — Wire resolves *AuthService → AuthServicer automatically

// Infra ProviderSet example (keycloak/provider.go)
var ProviderSet = wire.NewSet(
    NewClient,
    wire.Bind(new(auth.IdentityProvider), new(*Client)),
    wire.Bind(new(auth.TokenVerifier), new(*Client)),
)
```

Regenerate after any change: `make wire-{project}`

## Testing

The controller defines `ServiceIface`; the service uses domain interfaces. This means **zero real infrastructure** needed for controller or service tests.

```go
// Controller test — fake the service interface defined in controller.go or controller folder inside the app
type fakeService struct { token *domain.TokenClaims; err error }
func (f *fakeService) Login(ctx context.Context, u, p string) (*domain.TokenClaims, error) {
    return f.token, f.err
}
ctrl := auth.NewController(&fakeService{token: &domain.TokenClaims{AccessToken: "abc"}})

// Service test — fake the domain interfaces
svc := auth.NewAuthService(&fakeIdentity{...}, &fakeRepo{...}, &fakeCache{...})
```

Repository tests are the **only** layer that needs a real DB or `sqlmock`.

## Adding a Module — Always 3 Steps

1. Create `internal/app/{name}/` with all 5 files (controller, service, dto, routes, provider)
2. Add `{name}.ProviderSet` to `cmd/{project}/wire.go` → `wire.Build(...)`
3. Add `*{name}.App` field to `ServerApps` in `cmd/{project}/main.go` and register in `NewApp`

## Shared vs Project-Specific

| Will other projects need it? | Location |
|---|---|
| Yes (auth, cms, media, settings) | `internal/` |
| No (team, player, match, sponsor) | `projects/sevilla/` |
| Unsure | `projects/` — easy to move later |