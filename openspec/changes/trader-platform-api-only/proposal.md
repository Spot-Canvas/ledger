## Why

The trader engine currently writes trades and position state directly to the platform's PostgreSQL database (`ledger_trades`, `ledger_positions`, `engine_position_state`), creating a tight coupling between the engine and the platform's internal schema. This makes tenant deployments impossible without a database connection — defeating the goal of the tenant installer, which is to run a stateless Cloud Run container that only needs NATS and a Signal ngn API key.

## What Changes

- The trader engine stops writing to any database table directly
- All trade recording is done via `POST /api/v1/trades` on the Signal ngn platform API
- All portfolio and position reads (startup state, open position checks) are done via `GET /api/v1/accounts/{id}/portfolio` on the platform API
- The `DATABASE_URL` / Cloud SQL dependency is removed from the trader engine entirely
- The `internal/store` package's trade/position/account/order write paths are replaced by platform API calls
- `engine_position_state` (in-memory position cache) is hydrated from the platform API on startup instead of from the DB
- `ledger_trades`, `ledger_positions`, `ledger_accounts`, `ledger_account_balances`, `ledger_orders`, `ledger_schema_migrations` tables become safe to drop from the platform DB

## Capabilities

### New Capabilities
- `engine-platform-api-store`: A new store implementation backed by the platform API instead of direct DB access. Implements the same `Store` interface used by the trading engine but routes writes to `POST /api/v1/trades` and reads to `GET /api/v1/accounts/{id}/portfolio`.

### Modified Capabilities
- `trade-ingestion`: Trade recording no longer writes to `ledger_trades` — trades are submitted to the platform API. The ingestion contract (what fields are recorded, idempotency) stays the same, only the transport changes.
- `portfolio-tracking`: Position state is no longer sourced from `ledger_positions`. On engine startup and for open-position checks, the engine reads from the platform API portfolio endpoint.
- `platform-client`: The existing platform API client (used by the CLI) needs to expose the trade submission and portfolio query methods required by the engine store.

## Impact

- **Removed dependency**: `DATABASE_URL` env var and Cloud SQL (or any Postgres) are no longer required by the trader engine binary
- **New required config**: `SN_API_KEY` (already present in the tenant installer) and `TRADER_API_URL` (defaults to the Signal ngn platform API)
- **Affected packages**: `internal/store`, `internal/engine`, `internal/ingest`, `internal/config`
- **Platform API prerequisite**: The `ledger-rest-ingest` change in spot-canvas-app must be deployed and stable — the engine depends on `POST /api/v1/trades` and `GET /api/v1/accounts/{id}/portfolio`
- **`ledger_*` table removal**: Once the engine is deployed without DB writes, the `ledger_trades`, `ledger_positions`, `ledger_accounts`, `ledger_account_balances`, `ledger_orders`, and `ledger_schema_migrations` tables can be dropped from the platform DB in a follow-up migration
- **No change** to the signal pipeline, NATS subscription, strategy execution, or the trader CLI
