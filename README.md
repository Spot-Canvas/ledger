# Ledger Service

A Go service that acts as a trading ledger for the spot-canvas ecosystem. It ingests trade events via NATS JetStream, maintains portfolio state (positions, P&L), and exposes data through a REST API.

## Features

- **Trade Ingestion**: NATS JetStream durable consumer for reliable, at-least-once trade processing
- **Portfolio Tracking**: Positions derived from trade history with real-time updates
- **Multi-Account**: Support for live and paper trading accounts
- **Spot & Futures**: Handles both spot trades and leveraged futures positions
- **REST API**: HTTP endpoints for querying portfolio state and importing historic trades
- **Tax Data**: Captures cost basis, realized P&L, fees, and holding periods for tax reporting
- **Idempotent**: Duplicate trades are safely discarded (dedup by trade ID)
- **Rebuildable**: Positions can be rebuilt from trade history for audit/repair

## Prerequisites

- Go 1.24+
- [Task](https://taskfile.dev/) (task runner)
- Docker & Docker Compose (for local PostgreSQL and NATS)

## Quick Start

```bash
task setup
# wait a few seconds for containers to start, then:
task dev
```

This creates `.env` from the example, installs dependencies, starts PostgreSQL + NATS via Docker Compose, and runs the service.

## Local Development

### 1. Start infrastructure

```bash
task infra:up
```

Starts PostgreSQL and NATS with JetStream. If spot-canvas-app containers are already running, the ledger will reuse them automatically (shared database by design).

```bash
task infra:status    # check what's running
task infra:logs      # follow all container logs
task db:logs         # PostgreSQL logs only
task nats:logs       # NATS logs only
```

### 2. Configure environment

```bash
task env
# Edit .env if needed (defaults work with docker-compose)
```

### 3. Run the service

```bash
task dev
```

The service will:
- Apply database migrations automatically
- Connect to NATS and start consuming trade events
- Start the HTTP server on port 8080

For a compiled binary instead:

```bash
task run
```

### 4. Run tests

```bash
# Unit tests
task test

# Unit tests (verbose)
task test:v

# Unit tests with race detector
task test:race

# Integration tests (requires infra running)
task test:integration

# All tests (unit + integration + race detector)
task test:all

# Coverage report
task test:cover
```

### 5. Code quality

```bash
# Format all Go files
task fmt

# Run go vet + staticcheck
task lint

# All checks (fmt + vet + test)
task check
```

### 6. Infrastructure management

```bash
task infra:up       # Start PostgreSQL + NATS (skips if spot-canvas-app is running)
task infra:down     # Stop ledger containers (does not touch spot-canvas-app)
task infra:reset    # Stop + remove volumes (clean slate)
task infra:status   # Show which containers are running
task infra:logs     # Follow all container logs
task db:logs        # Follow PostgreSQL logs only
task nats:logs      # Follow NATS logs only
```

## Available Tasks

Run `task` or `task --list` to see all available tasks:

| Task | Description |
|------|-------------|
| `task setup` | Full local setup (env + deps + infra) |
| `task dev` | Run with `go run` |
| `task build` | Build binary to `bin/ledger` |
| `task run` | Build + run binary |
| `task test` | Unit tests |
| `task test:integration` | Integration tests |
| `task test:all` | Unit + integration + race |
| `task test:cover` | Tests with coverage report |
| `task check` | fmt + vet + test |
| `task infra:up` | Start PostgreSQL + NATS (reuses spot-canvas-app if running) |
| `task infra:down` | Stop ledger containers |
| `task infra:status` | Show running database & NATS containers |
| `task infra:reset` | Stop + delete volumes |
| `task docker:build` | Build Docker image |
| `task docker:run` | Run Docker image locally |
| `task deploy:staging` | Deploy to staging Cloud Run |
| `task deploy:production` | Deploy to production Cloud Run |
| `task clean` | Remove build artifacts |
| `task deps` | Download + tidy Go modules |

## API Endpoints

Query endpoints are read-only (GET). The import endpoint (POST) allows bulk-loading historic trades.

### Health

```
GET /health
```

Returns `{"status": "ok"}` when database and NATS are connected, `503` otherwise.

### Accounts

```
GET /api/v1/accounts
```

Returns all trading accounts.

### Portfolio Summary

```
GET /api/v1/accounts/{accountId}/portfolio
```

Returns open positions and aggregate realized P&L. Returns `404` if account not found.

### Positions

```
GET /api/v1/accounts/{accountId}/positions?status=open
```

Query params:
- `status`: `open` (default), `closed`, or `all`

### Trades

```
GET /api/v1/accounts/{accountId}/trades?symbol=BTC-USD&limit=50
```

Query params:
- `symbol`: Filter by trading pair
- `side`: `buy` or `sell`
- `market_type`: `spot` or `futures`
- `start`: Start time (RFC3339)
- `end`: End time (RFC3339)
- `cursor`: Pagination cursor
- `limit`: Results per page (default 50, max 200)

### Orders

```
GET /api/v1/accounts/{accountId}/orders?status=open
```

Query params:
- `status`: Filter by order status
- `symbol`: Filter by trading pair
- `cursor`: Pagination cursor
- `limit`: Results per page (default 50, max 200)

### Import Historic Trades

```
POST /api/v1/import
```

Bulk-import historic trades. Trades are validated up front (the entire batch is rejected if any trade is invalid), sorted by timestamp, inserted idempotently, and positions are rebuilt from the full trade history after import.

**Request body:**

```json
{
  "trades": [
    {
      "trade_id": "t-001",
      "account_id": "live",
      "symbol": "BTC-USD",
      "side": "buy",
      "quantity": 0.5,
      "price": 40000,
      "fee": 20,
      "fee_currency": "USD",
      "market_type": "spot",
      "timestamp": "2024-06-01T10:00:00Z"
    },
    {
      "trade_id": "t-002",
      "account_id": "live",
      "symbol": "BTC-USD",
      "side": "sell",
      "quantity": 0.5,
      "price": 45000,
      "fee": 22.50,
      "fee_currency": "USD",
      "market_type": "spot",
      "timestamp": "2024-07-01T10:00:00Z"
    }
  ]
}
```

**Response:**

```json
{
  "total": 2,
  "inserted": 2,
  "duplicates": 0,
  "errors": 0,
  "results": [
    {"trade_id": "t-001", "status": "inserted"},
    {"trade_id": "t-002", "status": "inserted"}
  ]
}
```

**Limits:** Max 1000 trades per request. Duplicate trade IDs are skipped (status `"duplicate"`). Re-importing the same trades is safe.

**Example with curl:**

```bash
curl -X POST http://localhost:8080/api/v1/import \
  -H "Content-Type: application/json" \
  -d '{"trades": [{"trade_id":"t-001","account_id":"live","symbol":"BTC-USD","side":"buy","quantity":0.5,"price":40000,"fee":20,"fee_currency":"USD","market_type":"spot","timestamp":"2024-06-01T10:00:00Z"}]}'
```

## NATS Trade Events

The service subscribes to `ledger.trades.>` for trade events.

### Subject Format

```
ledger.trades.<account>.<market_type>
```

Examples:
- `ledger.trades.live.spot`
- `ledger.trades.paper.futures`

### Message Format

```json
{
  "trade_id": "unique-trade-id",
  "account_id": "live",
  "symbol": "BTC-USD",
  "side": "buy",
  "quantity": 0.5,
  "price": 50000,
  "fee": 25,
  "fee_currency": "USD",
  "market_type": "spot",
  "timestamp": "2025-01-15T10:00:00Z",
  "leverage": null,
  "margin": null,
  "liquidation_price": null,
  "funding_fee": null
}
```

Futures trades include `leverage`, `margin`, `liquidation_price`, and optionally `funding_fee`.

## Deployment

### Staging

```bash
task deploy:staging
```

### Production

```bash
task deploy:production
```

The service deploys to Cloud Run (europe-west3) and shares the existing Cloud SQL instance with spot-canvas-app. All tables are prefixed with `ledger_` to avoid collisions.

### Docker

```bash
# Build image locally
task docker:build

# Run locally (connects to host network)
task docker:run
```

### Environment Variables

See [.env.example](.env.example) for all configuration options.

## Architecture

```
cmd/ledger/          # Entry point
internal/
├── config/          # Configuration loading
├── domain/          # Core types: Trade, Position, Account, Order
├── store/           # PostgreSQL repository, migrations
├── ingest/          # NATS JetStream consumer, trade processing
└── api/             # REST handlers, routing, middleware
migrations/          # SQL migration files
```

Data flow: `Trading Bot → NATS → Ingestion → PostgreSQL ← REST API ← Dashboard`
