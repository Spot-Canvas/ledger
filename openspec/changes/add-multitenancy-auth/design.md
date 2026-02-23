## Context

The ledger service is a standalone Go binary that ingests trades via NATS JetStream and exposes
a read-only REST API. It currently has no authentication or tenant isolation:

- All four ledger tables (`ledger_accounts`, `ledger_trades`, `ledger_positions`, `ledger_orders`)
  contain no `tenant_id` column — data is shared globally.
- Every HTTP handler works with account IDs like `"live"` or `"paper"` as plain strings with no
  namespace. Any caller can read any other user's portfolio.
- The NATS consumer accepts trade events with no `tenant_id` field; all writes are unscoped.

The ledger shares a PostgreSQL database with spot-canvas-app (same `DATABASE_URL`). The
`users` table with its `api_key → tenant_id` mapping already exists in that shared database,
established by the `auth` change in spot-canvas-app. The identity model is: one Google account =
one `tenant_id`; the trading bot authenticates with a Bearer API key issued on the settings page.

There is exactly one existing user (the owner). Data migration risk is low: we can backfill all
existing rows to `tenant_id = '00000000-0000-0000-0000-000000000001'` (the default tenant).

The `sn` CLI talks to the spot-canvas-app API server (not the ledger). It issues API keys via the
Google OAuth2 flow: `sn auth login` opens `<web_url>/oauth/start?cli_port=<port>` in the browser,
and after the Google callback the web server redirects to `http://localhost:<port>?api_key=<KEY>`.
The same API key is then used by the trading bot when calling the ledger — the bot reads it from
`~/.config/sn/config.yaml` or an environment variable. The ledger and spot-canvas-app share the
same `users` table, so the same key works for both.

---

## Goals / Non-Goals

**Goals:**
- Authenticate all `/api/v1/` requests via Bearer API key; resolve to a `tenant_id` using the
  shared `users` table; return 401 for missing or invalid credentials
- Scope all reads (accounts, trades, positions, orders) to the resolved `tenant_id`
- Scope all writes (NATS ingestion) to a `tenant_id` carried in the trade event
- Add `GET /auth/resolve` so the trading bot can self-identify its tenant before publishing events
- Add `tenant_id` to all ledger tables via a safe, additive migration
- `ENFORCE_AUTH=false` escape hatch for local development without a real `users` table entry

**Non-Goals:**
- Multiple users per tenant — one API key = one tenant, no org/team support
- Web-based login or session cookies — the ledger has no web UI
- HMAC-signed internal headers (X-Tenant-ID / X-Tenant-Hmac) — not needed; only the trading bot
  and direct API consumers talk to the ledger, never the spot-canvas-app web server
- Rate limiting or per-endpoint permissions
- Replacing the JetStream consumer with a push-based authenticated model — NATS auth is handled
  at the NATS credential level (unchanged); `tenant_id` is embedded in the event payload

---

## Decisions

### 1. Read `users` table directly — no new auth service

**Decision:** Add a `UserRepository` in `internal/store/users.go` that queries the `users` table
in the shared PostgreSQL database. Method: `GetByAPIKey(ctx, apiKey uuid.UUID) (*User, error)`.
The `User` struct carries only `TenantID` (the only field the ledger needs).

**Rationale:** The ledger already connects to the same PostgreSQL instance as spot-canvas-app.
Adding a `UserRepository` is ~30 lines of code. The alternative — an HTTP call to spot-canvas-app
on every request — introduces a network hop, a new failure mode, and a circular dependency
between two services that otherwise share only a database.

**Alternative considered:** Call spot-canvas-app's auth API to resolve keys — rejected (latency,
availability coupling, no clear API endpoint exists for this).

**Alternative considered:** Duplicate the `users` table in the ledger schema — rejected (two
sources of truth for API keys would require key sync on every regeneration).

---

### 2. `tenant_id` column: NOT NULL with migration-time default, then drop default

**Decision:** Migration 003 adds `tenant_id UUID NOT NULL DEFAULT
'00000000-0000-0000-0000-000000000001'` to all four tables, backfills existing rows (they already
have the default), then removes the column default so future inserts must supply an explicit
`tenant_id`. Indexes: composite indexes on `(tenant_id, account_id, ...)` replace single-column
`account_id` indexes where appropriate.

**Rationale:** PostgreSQL's `NOT NULL DEFAULT` allows adding the column and backfilling in a
single `ALTER TABLE` without a separate `UPDATE`. Removing the default afterward prevents future
code from accidentally omitting `tenant_id` and silently falling back to the seed value.

**Alternative considered:** Nullable `tenant_id` — rejected because all query paths would need
NULL-handling and the semantics are muddier.

**Alternative considered:** Separate per-tenant tables — rejected as over-engineering for this
scale.

---

### 3. Account ID namespace: `(tenant_id, account_id)` composite key

**Decision:** `ledger_accounts.id` remains a TEXT primary key (e.g. `"live"`, `"paper"`), but a
new `tenant_id` column is added and the primary key is changed to `(tenant_id, id)`. All
repository methods that currently accept `accountID string` gain a `tenantID uuid.UUID` parameter
and filter by both columns.

**Rationale:** This preserves the human-readable account IDs (`"live"`, `"paper"`) that the
trading bot already uses in NATS event payloads, while properly namespacing them per tenant.
No event format change for the `account_id` field.

**Alternative considered:** Use UUID account IDs — rejected because the bot already produces
`account_id: "live"` / `"paper"` and changing that is a separate concern.

**Migration note:** The existing `id TEXT PRIMARY KEY` becomes `PRIMARY KEY (tenant_id, id)`.
pgx handles composite PKs naturally; the `GetOrCreateAccount` query gains a `tenant_id` WHERE
clause.

---

### 4. `tenant_id` in NATS events: required field, terminate on missing

**Decision:** Add `TenantID string` (UUID string form) to `TradeEvent`. `Validate()` returns an
error if `TenantID` is empty or not a valid UUID. On validation failure the consumer calls
`msg.Term()` (no redelivery) and logs a warning — same pattern as today for other validation
failures.

**Rationale:** The trading bot is the only NATS publisher. Adding `tenant_id` to the payload is
a one-time change on the bot side. Making it required (rather than defaulting to a fallback)
ensures the bot is updated before events reach production; silent defaulting would mask a
misconfigured bot.

**NATS subject unchanged:** The existing `ledger.trades.>` wildcard and `LEDGER_TRADES` stream
name are preserved. The `tenant_id` lives in the message body, not the subject, so no stream
reconfiguration is needed.

---

### 5. `GET /auth/resolve` — tenant identity endpoint for the trading bot

**Decision:** Add `GET /auth/resolve` to the ledger router, protected by the same `AuthMiddleware`.
On success it returns `{"tenant_id": "<uuid>"}`. The bot calls this once at startup to confirm
its identity and obtain its `tenant_id` for embedding in subsequent NATS events.

**Rationale:** Without this endpoint the bot would need to hard-code its `tenant_id` in config,
which breaks if the tenant is recreated or the key is rotated. The resolve endpoint decouples
bot config from tenant UUIDs: the bot stores only its API key and resolves the tenant at runtime.

This is a ledger-specific endpoint — the sn CLI does not use it (sn talks to the spot-canvas-app
API server, not the ledger). The `sn` auth flow uses `GET /oauth/start?cli_port=<port>` on the
web server and `GET /auth/callback` to deliver the API key; the API key issued there is the same
one the bot presents to the ledger.

**Alternative considered:** Embed `tenant_id` in the API key JWT — rejected (API keys are plain
UUIDs, not JWTs; this is the established pattern from spot-canvas-app).

---

### 6. `AuthMiddleware` — Bearer API key only, no HMAC header support

**Decision:** The ledger `AuthMiddleware` supports only `Authorization: Bearer <api_key>`.
It does NOT support the `X-Tenant-ID` + `X-Tenant-Hmac` internal header pair used by
spot-canvas-app's web server.

**Rationale:** The ledger is not called by spot-canvas-app's web server. Supporting HMAC headers
adds code for a path that will never be exercised. If web-server → ledger calls are needed in
the future, HMAC support can be added then.

**`ENFORCE_AUTH` flag:** When `ENFORCE_AUTH=false` (env var, default `true`), the middleware
logs a warning and falls back to a hardcoded default tenant ID
(`00000000-0000-0000-0000-000000000001`). This enables local dev without a real `users` table
entry. Matches the pattern from spot-canvas-app's API server auth change.

---

### 7. Store layer: `tenantID` added to all repository signatures

**Decision:** Every repository method that reads or writes tenant-scoped data gains a
`tenantID uuid.UUID` parameter as its second argument (after `ctx`). Handlers extract tenant ID
from context via `TenantIDFromContext(r.Context())`. No global/package-level state.

**Affected methods:**
- `GetOrCreateAccount(ctx, tenantID, id, accountType)`
- `AccountExists(ctx, tenantID, id)`
- `ListAccounts(ctx, tenantID)`
- `GetPortfolioSummary(ctx, tenantID, accountID)`
- `ListPositions(ctx, tenantID, accountID, status)`
- `GetAvgEntryPrice(ctx, tenantID, accountID, symbol, marketType)`
- `InsertTradeAndUpdatePosition(ctx, tenantID, trade)` — `trade` struct also gains `TenantID`
- `InsertTrade(ctx, tx, trade)` — trade already has `TenantID` field
- `ListTrades(ctx, tenantID, accountID, filter)`
- `ListOrders(ctx, tenantID, accountID, filter)`

**`domain.Trade` struct:** gains `TenantID uuid.UUID` field.

---

## Data Model Changes

```sql
-- Migration 003: add tenant_id to all ledger tables

-- Accounts: composite PK
ALTER TABLE ledger_accounts
    ADD COLUMN tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001';
-- Drop old PK, add composite PK
ALTER TABLE ledger_accounts DROP CONSTRAINT ledger_accounts_pkey;
ALTER TABLE ledger_accounts ADD PRIMARY KEY (tenant_id, id);
ALTER TABLE ledger_accounts ALTER COLUMN tenant_id DROP DEFAULT;

-- Trades
ALTER TABLE ledger_trades
    ADD COLUMN tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE ledger_trades ALTER COLUMN tenant_id DROP DEFAULT;
CREATE INDEX idx_ledger_trades_tenant_account_timestamp
    ON ledger_trades (tenant_id, account_id, timestamp DESC);

-- Positions: composite unique constraint
ALTER TABLE ledger_positions
    ADD COLUMN tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE ledger_positions ALTER COLUMN tenant_id DROP DEFAULT;
-- Drop old unique index, add new one scoped to tenant
DROP INDEX idx_ledger_positions_open_unique;
CREATE UNIQUE INDEX idx_ledger_positions_open_unique
    ON ledger_positions (tenant_id, account_id, symbol, market_type)
    WHERE status = 'open';
CREATE INDEX idx_ledger_positions_tenant_account_status
    ON ledger_positions (tenant_id, account_id, status);

-- Orders
ALTER TABLE ledger_orders
    ADD COLUMN tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001';
ALTER TABLE ledger_orders ALTER COLUMN tenant_id DROP DEFAULT;
CREATE INDEX idx_ledger_orders_tenant_account_status_created
    ON ledger_orders (tenant_id, account_id, status, created_at DESC);
```

```go
// internal/store/users.go — new file
type User struct {
    TenantID uuid.UUID
}

type UserRepository struct { pool *pgxpool.Pool }

func (r *UserRepository) GetByAPIKey(ctx context.Context, apiKey uuid.UUID) (*User, error)
// SELECT tenant_id FROM users WHERE api_key = $1
```

---

## Request Flows

### API key issuance (via sn CLI / web dashboard)
```
sn auth login
  → opens browser: <web_url>/oauth/start?cli_port=<PORT>   ← actual route (not /auth/login)
  → Google OAuth2 flow → GET /auth/callback
  → redirect to http://localhost:<PORT>?api_key=<KEY>&email=<EMAIL>
  sn stores api_key in ~/.config/sn/config.yaml
  (same api_key is used for both sn → spot-canvas-app API and bot → ledger)
```

### Trading bot startup
```
bot starts (has api_key from sn config or env var)
  → GET /auth/resolve  Authorization: Bearer <api_key>   ← ledger endpoint
    AuthMiddleware: SELECT tenant_id FROM users WHERE api_key = $1  → tenant_id in ctx
    handler: return {"tenant_id": "<uuid>"}
  bot stores tenant_id in memory
  bot publishes NATS events with tenant_id in payload
```

### HTTP API request (authenticated)
```
bot/client → GET /api/v1/accounts  Authorization: Bearer <api_key>
  AuthMiddleware: GetByAPIKey → tenant_id in ctx
  handler: ListAccounts(ctx, tenantIDFromCtx)
  repo: SELECT ... FROM ledger_accounts WHERE tenant_id = $1
```

### NATS trade ingestion
```
bot publishes: ledger.trades.live  {"trade_id": "...", "tenant_id": "...", "account_id": "live", ...}
  consumer: Unmarshal → Validate (checks tenant_id non-empty, valid UUID)
  consumer: GetOrCreateAccount(ctx, tenantID, "live", "live")
  consumer: InsertTradeAndUpdatePosition(ctx, tenantID, trade)
```

### Development / no credentials
```
request → GET /api/v1/accounts  (no Authorization header)
  ENFORCE_AUTH=false: log warning, use default tenant_id
  handler: ListAccounts(ctx, defaultTenantID)
```

---

## Environment Variables

| Variable | Default | Purpose |
|---|---|---|
| `ENFORCE_AUTH` | `true` | Set `false` in dev to skip auth and fall back to default tenant |

No new infrastructure variables — `DATABASE_URL` already points to the shared DB containing `users`.

---

## Migration Plan

1. Deploy migration 003 (additive — default fills existing rows; existing queries still work
   until code is updated because the WHERE clause just widens to include the default tenant)
2. Deploy updated ledger binary with `ENFORCE_AUTH=false` — handlers now add `tenant_id` to
   queries but all existing data uses the default tenant ID, so results are unchanged
3. Update the trading bot to include `tenant_id` in NATS events and call `/auth/resolve` on
   startup
4. Switch ledger to `ENFORCE_AUTH=true`
5. Verify: unauthenticated request → 401; bot request with API key → correct tenant data

**Rollback:** Redeploy previous binary. Migration 003 is backward-compatible (old binary ignores
the new columns). The `ENFORCE_AUTH=false` flag allows instant fallback to open access if the
new binary causes issues.

---

## Risks / Trade-offs

**[Risk] `users` table cross-DB dependency** — if spot-canvas-app migrates the `users` table
schema (e.g. renames `api_key`), the ledger's `UserRepository` silently breaks.
→ Mitigation: `UserRepository` is a thin read-only query; the SELECT is explicit and will fail
fast at startup if the column is missing. The two services share a database intentionally and are
maintained together.

**[Risk] Composite PK migration on `ledger_accounts`** — `ALTER TABLE ... DROP CONSTRAINT ... ADD
PRIMARY KEY` acquires an exclusive lock. With few rows (dev data only at this point) the lock
duration is negligible.
→ Mitigation: run migration during a maintenance window (or during the next deploy, before
traffic resumes).

**[Risk] Bot not updated before `ENFORCE_AUTH=true`** — NATS events without `tenant_id` will be
terminated; trades will be lost.
→ Mitigation: step 3 (update bot) must complete before step 4 (`ENFORCE_AUTH=true`). Deploy
order enforces this.

**[Trade-off] API key lookup on every HTTP request** — no in-memory cache; every request hits
PostgreSQL with a `SELECT tenant_id FROM users WHERE api_key = $1` (indexed UUID lookup).
At current scale (single user, low QPS) this is fine. A short-lived LRU cache can be added later
if needed.
