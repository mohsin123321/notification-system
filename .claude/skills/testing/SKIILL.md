---
name: testing
description: Conventions for writing tests in this project. Apply these rules whenever writing or reviewing test code.
---

Follow these conventions when writing tests in this project:

## Subtest ordering
Always place success cases first, failure cases after.

## gomock argument matchers
Only use `gomock.Any()` for arguments you genuinely cannot know at test setup time (e.g. `ctx`, randomly generated secrets). For everything else pass the real value directly — this applies equally to success cases and error/failure cases. If asserting a dynamic value is needed (e.g. a map with some dynamic keys), prefer a custom `gomock.Matcher` or `DoAndReturn` over `gomock.Any()`.

## Comments
Do not add Arrange/Act/Assert section comments. Do not add numbered case lists (e.g. `// 1. success`, `// 2. fails`) at the top of test functions.

## Assertions
Use `require` for conditions that must pass before indexing or dereferencing (e.g. `require.NotEmpty(t, resp.Errors)` before `resp.Errors[0].Message`). Use `assert` for everything else. Never use `t.Fatalf` manually.

## Shared response types
Use `shared.APIResponse` (and `shared.APIError`) for unmarshalling HTTP responses in controller tests. Do not duplicate these structs locally.

## Service error tests
For tests that verify a service error propagates to an HTTP error response, use `assert.AnError` as the sentinel error — do not assert on specific error types or messages from the service layer.

## Mock generation — `//go:generate` directives

Every interface that has a mock **must** have a `//go:generate` directive placed directly above it in the source file. Never hand-maintain mock files.

### Which mode to use

| Situation | Mode | Directive |
|-----------|------|-----------|
| Interface is self-contained in one file, no generics | **source** | `-source=<file>.go` |
| Interface embeds interfaces from other files | **reflect** | package path only (no `-source`) |
| Interface uses generics (`PagedResult[any]`, etc.) | **reflect** | package path only (no `-source`) |

**Source mode** example:
```go
//go:generate mockgen -source=port.go -destination=mock/mock_foo.go -package=mock FooInterface
type FooInterface interface { ... }
```

**Reflect mode** example (cross-file embedding or generics):
```go
//go:generate mockgen -package mock -destination=mock/mock_foo.go . FooInterface
type FooInterface interface {
    BarInterface  // defined in another file in same package
}
```

In reflect mode, `.` means the current package and resolves correctly when `go generate` runs from the package directory.

### mockgen binary
Use `go.uber.org/mock/mockgen` (v0.5+). The old `github.com/golang/mock/mockgen` (v1.6) is incompatible — it generates wrong import paths. Install with:
```
go install go.uber.org/mock/mockgen@latest
```

### Running go generate
Run from the package directory or via `go generate ./path/to/pkg/...` from the repo root. Reflect mode requires the package to compile, so fix all compile errors first.

### Placement rule
The `//go:generate` directive goes directly above the **first** interface in the file it applies to. If multiple interfaces in one file share a single mock file, list them all in one directive:
```go
//go:generate mockgen -source=port.go -destination=mock/mock_port.go -package=mock InterfaceA,InterfaceB
type InterfaceA interface { ... }
type InterfaceB interface { ... }
```

### Do not use PagedResult type aliases
Do not create `PagedResultAny = PagedResult[any]` aliases to work around mockgen source-mode limitations. Switch to reflect mode instead and keep the original generic type in the interface.

## Controller test helpers
For HTTP controller tests, register routes using the production `routes.Register` function instead of manually wiring `router.POST(...)`. Middleware dependencies that are never reached in the test can be passed as `nil`. Example:
```go
func testAuthRouter(t *testing.T) (*gin.Engine, *mock.MockServicer) {
    t.Helper()
    gin.SetMode(gin.TestMode)
    ctrl := gomock.NewController(t)
    mockSvc := mock.NewMockServicer(ctrl)
    c := controller.New(mockSvc)
    router := gin.New()
    routes.Register(router, c, nil, nil, nil)
    return router, mockSvc
}
```

## Request builder helpers
Define request-building helpers as local function variables inside the test function, not as package-level functions. They should take all required parameters explicitly (no conditional `if param != ""` guards):
```go
func TestLogin(t *testing.T) {
    loginReq := func(username, password, otp, ip string) *http.Request {
        form := url.Values{}
        form.Set("username", username)
        form.Set("password", password)
        form.Set("otp", otp)
        req := httptest.NewRequest(http.MethodPost, "/admin/api/auth/login", strings.NewReader(form.Encode()))
        req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
        req.Header.Set("X-Client-IP", ip)
        return req
    }
    // subtests follow...
}
```
