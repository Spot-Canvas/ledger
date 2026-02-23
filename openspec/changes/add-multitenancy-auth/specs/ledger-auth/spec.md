## ADDED Requirements

### Requirement: UserRepository for API key resolution

The `internal/store/` package SHALL provide a `UserRepository` with a single method:
`GetByAPIKey(ctx context.Context, apiKey uuid.UUID) (*AuthUser, error)` — executes
`SELECT tenant_id FROM users WHERE api_key = $1` against the shared PostgreSQL database and
returns an `AuthUser{TenantID uuid.UUID}` or nil if not found. The `users` table is the same
table created and managed by spot-canvas-app; the ledger has read-only access to it.

#### Scenario: Known API key returns tenant ID
- **WHEN** `GetByAPIKey` is called with a UUID that exists in the `users.api_key` column
- **THEN** it returns an `AuthUser` with the corresponding `tenant_id`

#### Scenario: Unknown API key returns nil
- **WHEN** `GetByAPIKey` is called with a UUID that does not exist in `users`
- **THEN** it returns `nil, nil`

#### Scenario: Malformed UUID does not panic
- **WHEN** `GetByAPIKey` is called with `uuid.Nil`
- **THEN** it returns `nil, nil` (no match)

---

### Requirement: AuthMiddleware for Bearer API key authentication

The ledger HTTP server SHALL apply an `AuthMiddleware` to all routes under `/api/v1/` and to
`/auth/resolve`. The middleware SHALL:

1. Read the `Authorization` header; if it is of the form `Bearer <uuid>`, parse the UUID and
   call `UserRepository.GetByAPIKey`
2. On a successful lookup, place the resolved `tenant_id` in the request context under a typed
   `TenantIDKey` and call the next handler
3. If the header is absent, malformed, or the API key is not found, return HTTP 401 with
   `{"error": "unauthorized"}`

When the environment variable `ENFORCE_AUTH` is set to `false` (string), the middleware SHALL
log a warning and fall back to a hardcoded default tenant ID
(`00000000-0000-0000-0000-000000000001`) instead of returning 401. This enables local
development without a real entry in the `users` table.

`ENFORCE_AUTH` defaults to `true` if unset or set to any value other than `false`.

#### Scenario: Valid Bearer API key accepted
- **WHEN** a request carries `Authorization: Bearer <valid-uuid>` that matches a row in `users`
- **THEN** the tenant ID is placed in the request context and the handler is invoked

#### Scenario: Unknown API key returns 401
- **WHEN** a request carries `Authorization: Bearer <uuid>` that does not exist in `users`
- **THEN** HTTP 401 is returned with `{"error": "unauthorized"}`

#### Scenario: Missing Authorization header returns 401
- **WHEN** a request to `/api/v1/accounts` carries no `Authorization` header and `ENFORCE_AUTH=true`
- **THEN** HTTP 401 is returned with `{"error": "unauthorized"}`

#### Scenario: Non-Bearer scheme returns 401
- **WHEN** a request carries `Authorization: Basic dXNlcjpwYXNz`
- **THEN** HTTP 401 is returned with `{"error": "unauthorized"}`

#### Scenario: ENFORCE_AUTH=false falls back to default tenant
- **WHEN** `ENFORCE_AUTH=false` is set and a request carries no `Authorization` header
- **THEN** a warning is logged, the default tenant ID is used, and the handler is invoked (no 401)

#### Scenario: Health endpoint exempt from auth
- **WHEN** an unauthenticated request is made to `GET /health`
- **THEN** HTTP 200 is returned without requiring an API key

---

### Requirement: TenantIDFromContext helper

The `internal/api/` package SHALL expose a `TenantIDFromContext(ctx context.Context) uuid.UUID`
helper that reads the tenant ID placed by `AuthMiddleware`. All HTTP handlers SHALL use this
helper to obtain the tenant ID; no handler SHALL use a hardcoded tenant ID constant.

#### Scenario: Context carries tenant ID
- **WHEN** `TenantIDFromContext` is called on a context populated by `AuthMiddleware`
- **THEN** it returns the resolved `uuid.UUID`

#### Scenario: Context missing tenant ID returns zero value
- **WHEN** `TenantIDFromContext` is called on a context without a tenant ID (e.g. in tests)
- **THEN** it returns `uuid.Nil`

---

### Requirement: GET /auth/resolve endpoint

The ledger HTTP server SHALL expose `GET /auth/resolve`, protected by `AuthMiddleware`. On
success it SHALL return HTTP 200 with `{"tenant_id": "<uuid>"}`. This endpoint allows the
trading bot to resolve its `tenant_id` at startup using only its stored API key, without
hard-coding the UUID in its configuration.

#### Scenario: Authenticated request returns tenant ID
- **WHEN** `GET /auth/resolve` is called with a valid Bearer API key
- **THEN** HTTP 200 is returned with `{"tenant_id": "<uuid>"}`

#### Scenario: Unauthenticated request returns 401
- **WHEN** `GET /auth/resolve` is called without an `Authorization` header
- **THEN** HTTP 401 is returned with `{"error": "unauthorized"}`

---

### Requirement: ENFORCE_AUTH configuration

The ledger configuration SHALL support an `ENFORCE_AUTH` environment variable (string, default
`"true"`). When `"false"`, the `AuthMiddleware` bypasses key lookup and injects the default
tenant ID. The `Config` struct in `internal/config/` SHALL expose an `EnforceAuth bool` field
loaded from this variable.

#### Scenario: ENFORCE_AUTH unset defaults to true
- **WHEN** `ENFORCE_AUTH` is not set in the environment
- **THEN** `Config.EnforceAuth` is `true` and the middleware enforces authentication

#### Scenario: ENFORCE_AUTH=false disables enforcement
- **WHEN** `ENFORCE_AUTH=false` is set
- **THEN** `Config.EnforceAuth` is `false` and unauthenticated requests receive the default tenant ID
