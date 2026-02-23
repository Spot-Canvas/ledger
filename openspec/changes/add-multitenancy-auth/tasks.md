## 1. Database Migration

- [x] 1.1 Write migration `003_add_tenant_id.up.sql`: add `tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'` to `ledger_accounts`, `ledger_trades`, `ledger_positions`, `ledger_orders`
- [x] 1.2 Drop old `ledger_accounts` primary key and add composite PK `(tenant_id, id)` in the same migration
- [x] 1.3 Drop old `idx_ledger_positions_open_unique` and recreate as `(tenant_id, account_id, symbol, market_type) WHERE status = 'open'`
- [x] 1.4 Drop old single-column indexes on `account_id` and add composite indexes: `(tenant_id, account_id, timestamp DESC)` on trades, `(tenant_id, account_id, status)` on positions, `(tenant_id, account_id, status, created_at DESC)` on orders
- [x] 1.5 Remove the `DEFAULT` clause from `tenant_id` on all four tables after backfill (so future inserts must supply an explicit value)
- [x] 1.6 Write `003_add_tenant_id.down.sql` to reverse the migration
- [x] 1.7 Run migration on local dev database and verify schema with `\d` on each table
- [x] 1.8 Copy migration file to `migrations/` (root-level copy used by Dockerfile / Cloud Run)

## 2. Domain Model

- [x] 2.1 Add `TenantID uuid.UUID` field to `domain.Trade` struct in `internal/domain/types.go`
- [x] 2.2 Verify JSON tag is `tenant_id` and that existing serialisation tests still pass

## 3. Storage — UserRepository

- [x] 3.1 Create `internal/store/users.go` with `AuthUser struct { TenantID uuid.UUID }` and `UserRepository struct { pool *pgxpool.Pool }`
- [x] 3.2 Implement `NewUserRepository(pool *pgxpool.Pool) *UserRepository`
- [x] 3.3 Implement `GetByAPIKey(ctx, apiKey uuid.UUID) (*AuthUser, error)` — `SELECT tenant_id FROM users WHERE api_key = $1`
- [x] 3.4 Write unit test: known key returns correct tenant ID; unknown key returns nil; uuid.Nil returns nil

## 4. Storage — Tenant-scoped repository methods

- [x] 4.1 Update `GetOrCreateAccount(ctx, tenantID uuid.UUID, id string, accountType)` — add `tenant_id` to SELECT and INSERT; update composite PK lookup
- [x] 4.2 Update `AccountExists(ctx, tenantID uuid.UUID, id string)` — filter by `tenant_id`
- [x] 4.3 Update `ListAccounts(ctx, tenantID uuid.UUID)` — filter by `tenant_id`
- [x] 4.4 Update `GetPortfolioSummary(ctx, tenantID uuid.UUID, accountID string)` — filter by `tenant_id`
- [x] 4.5 Update `ListPositions(ctx, tenantID uuid.UUID, accountID string, status string)` — filter by `tenant_id`
- [x] 4.6 Update `GetAvgEntryPrice(ctx, tenantID uuid.UUID, accountID, symbol string, marketType)` — filter by `tenant_id`
- [x] 4.7 Update `InsertTrade(ctx, tx, trade *domain.Trade)` — include `tenant_id` in INSERT column list (reads from `trade.TenantID`)
- [x] 4.8 Update `InsertTradeAndUpdatePosition(ctx, tenantID uuid.UUID, trade *domain.Trade)` — pass `tenantID` through to all sub-calls; include `tenant_id` in position INSERT/UPDATE
- [x] 4.9 Update `ListTrades(ctx, tenantID uuid.UUID, accountID string, filter TradeFilter)` — add `tenant_id = $N` condition
- [x] 4.10 Update `ListOrders(ctx, tenantID uuid.UUID, accountID string, filter OrderFilter)` — add `tenant_id = $N` condition

## 5. Config

- [x] 5.1 Add `EnforceAuth bool` field to `Config` struct in `internal/config/config.go`
- [x] 5.2 Load `EnforceAuth` from `ENFORCE_AUTH` env var: parse as bool, default `true` when unset or any value other than `"false"`

## 6. Auth Middleware

- [x] 6.1 Create `internal/api/middleware/auth.go` with typed `tenantIDKey` context key and `TenantIDFromContext(ctx) uuid.UUID` helper
- [x] 6.2 Implement `NewAuthMiddleware(userRepo *store.UserRepository, enforceAuth bool, defaultTenantID uuid.UUID) func(http.Handler) http.Handler`
- [x] 6.3 Middleware logic: parse `Authorization: Bearer <uuid>` → call `GetByAPIKey` → set tenant ID in context; return 401 on missing/invalid/unknown key when `enforceAuth=true`; fall back to `defaultTenantID` with warning log when `enforceAuth=false`
- [x] 6.4 Write unit tests: valid key accepted; unknown key → 401; missing header → 401; non-Bearer scheme → 401; `ENFORCE_AUTH=false` uses default tenant

## 7. HTTP Router and Handlers

- [x] 7.1 Instantiate `UserRepository` in `cmd/ledger/main.go` and pass to `NewAuthMiddleware`
- [x] 7.2 Mount `AuthMiddleware` on `/api/v1/` subrouter and on `/auth/resolve` in `internal/api/router.go`; leave `/health` exempt
- [x] 7.3 Update `Server` struct to hold `*store.UserRepository`; pass `enforceAuth` and `defaultTenantID` from config
- [x] 7.4 Add `GET /auth/resolve` handler: reads tenant ID from context via `TenantIDFromContext`, returns `{"tenant_id": "<uuid>"}` with HTTP 200
- [x] 7.5 Update `handleListAccounts`: extract `tenantID` from context, pass to `repo.ListAccounts(ctx, tenantID)`
- [x] 7.6 Update `handlePortfolioSummary`: pass `tenantID` to `repo.AccountExists` and `repo.GetPortfolioSummary`
- [x] 7.7 Update `handleListPositions`: pass `tenantID` to `repo.ListPositions`
- [x] 7.8 Update `handleListTrades`: pass `tenantID` to `repo.ListTrades`
- [x] 7.9 Update `handleListOrders`: pass `tenantID` to `repo.ListOrders`

## 8. NATS Ingestion — TradeEvent

- [x] 8.1 Add `TenantID string` field with JSON tag `tenant_id` to `TradeEvent` struct in `internal/ingest/event.go`
- [x] 8.2 Update `Validate()`: return error if `TenantID` is empty or cannot be parsed by `uuid.Parse`
- [x] 8.3 Update `ToDomain()`: parse `TenantID` with `uuid.Parse` and set `trade.TenantID`
- [x] 8.4 Write unit tests for `Validate()`: missing tenant_id → error; non-UUID tenant_id → error; valid UUID → passes

## 9. NATS Ingestion — Consumer

- [x] 9.1 Update `handleMessage` in `consumer.go`: after `ToDomain()`, use `trade.TenantID` as `tenantID` for all repository calls
- [x] 9.2 Update `GetOrCreateAccount` call to pass `trade.TenantID`
- [x] 9.3 Update `InsertTradeAndUpdatePosition` call to pass `trade.TenantID`
- [x] 9.4 Update log fields to include `tenant_id` on ingestion log lines

## 10. Dependency

- [x] 10.1 Run `go get github.com/google/uuid` if not already a direct dependency; verify `go.mod` lists it directly
- [x] 10.2 Run `go build ./...` and fix any compilation errors

## 11. Tests and Verification

- [x] 11.1 Run existing integration tests; fix any broken tests caused by new `tenantID` parameters
- [x] 11.2 Write integration test for `GET /auth/resolve`: valid API key returns correct tenant ID; no key returns 401
- [x] 11.3 Write integration test for `GET /api/v1/accounts`: returns only accounts for the authenticated tenant
- [x] 11.4 Write integration test for tenant isolation: insert trades for two tenants; verify each can only see its own data
- [x] 11.5 Manually verify end-to-end: run `sn auth login`, copy API key, call `GET /auth/resolve` on the ledger, confirm returned `tenant_id` matches the one in spot-canvas-app `users` table

## 12. Deployment

- [x] 12.1 Run migration 003 on staging database
- [x] 12.2 Deploy ledger with `ENFORCE_AUTH=false` initially; verify all existing functionality works with default tenant fallback
- [ ] 12.3 Update trading bot to call `GET /auth/resolve` at startup and include `tenant_id` in all NATS trade events
- [x] 12.4 Switch ledger to `ENFORCE_AUTH=true` on staging; verify bot requests are authenticated and 401 is returned for unauthenticated callers
- [x] 12.5 Deploy to production following the same sequence (migration → `ENFORCE_AUTH=false` → bot update → `ENFORCE_AUTH=true`)
