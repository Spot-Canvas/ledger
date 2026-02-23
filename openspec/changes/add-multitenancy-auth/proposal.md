## Why

The ledger service is part of the SignalNgn infrastructure alongside spot-canvas-app, but currently has no authentication or tenant isolation — any caller can read or write any account's trading data. As the platform moves to real multi-tenant usage, the ledger must enforce the same identity model already established in spot-canvas-app: a Google OAuth2-backed `users` table mapping each user to a `tenant_id`, with API requests authenticated via a Bearer API key and trade ingestion scoped to that tenant.

## What Changes

- Add `tenant_id` column to all ledger tables (`ledger_accounts`, `ledger_trades`, `ledger_positions`, `ledger_orders`) so every row is scoped to a tenant
- Add an `GET /auth/resolve` endpoint that accepts a Bearer API key and returns the resolved `tenant_id` — this is how the trading bot authenticates itself and learns which tenant it belongs to
- Add `AuthMiddleware` to the HTTP router: Bearer API key → `UserRepository.GetByAPIKey` → `tenant_id` in context; all `/api/v1/` handlers scoped to the resolved tenant; unauthenticated requests return HTTP 401
- All NATS-ingested trade events must carry a `tenant_id`; the consumer validates and uses it to scope writes
- The `ledger_accounts` primary key changes from a plain text ID to a `(tenant_id, account_id)` pair — account IDs like `"live"` and `"paper"` are now namespaced per tenant
- Migrate existing single-tenant data to `tenant_id = '00000000-0000-0000-0000-000000000001'` (the existing default tenant in spot-canvas-app)
- **BREAKING**: all `/api/v1/` endpoints now require `Authorization: Bearer <api_key>` — unauthenticated requests return 401

## Capabilities

### New Capabilities

- `ledger-auth`: Bearer API key authentication middleware for the ledger HTTP server; `/auth/resolve` endpoint that exchanges an API key for a tenant ID; `ENFORCE_AUTH` escape hatch for dev mode; reads `users` table from the shared spot-canvas-app PostgreSQL database (same `DATABASE_URL`, different table prefix)

### Modified Capabilities

- `rest-api`: all endpoints now require auth; responses are scoped to the caller's `tenant_id`; account IDs are resolved within the tenant namespace; 401 returned on missing/invalid credentials
- `trade-ingestion`: NATS trade events must carry `tenant_id`; consumer validates presence and rejects events without it; all writes scoped to the event's `tenant_id`
- `portfolio-tracking`: positions and portfolio summary scoped by `tenant_id`; the `ledger_positions` unique constraint becomes `(tenant_id, account_id, symbol, market_type)` for open positions
- `order-history`: orders and trades scoped by `tenant_id`; queries filtered by resolved tenant

## Impact

- **Database**: new migration adding `tenant_id UUID NOT NULL` column (with default for migration safety) to all four ledger tables; updated indexes; backfill existing rows to default tenant
- **`internal/api/`**: `AuthMiddleware` added; `Server` struct gains `UserRepository` reference; all handlers read `tenant_id` from context instead of using it from path params alone
- **`internal/ingest/`**: `TradeEvent` struct gains `TenantID` field; consumer validates and uses it
- **`internal/config/`**: new env vars: `INTERNAL_HMAC_SECRET` (optional, for future web-server integration), `ENFORCE_AUTH` (default `true`)
- **`internal/store/`**: all repository methods gain `tenantID uuid.UUID` parameter; `UserRepository` added (read-only access to spot-canvas-app `users` table)
- **New dependency**: `github.com/google/uuid` (may already be present transitively)
- **Environment**: `DATABASE_URL` already points to the shared spot-canvas-app PostgreSQL database — no new database required; the `users` table is already there
