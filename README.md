# Trader

A Go trading engine for the Signal Ngn ecosystem. It ingests trade events via NATS JetStream, maintains portfolio state (positions, P&L), and exposes data through a REST API and CLI.

## Features

- **Trade Ingestion**: NATS JetStream durable consumer for reliable, at-least-once trade processing
- **Portfolio Tracking**: Positions derived from trade history with real-time updates
- **Multi-Tenancy**: Bearer API key auth; each user's data is fully isolated
- **Multi-Account**: Support for live and paper trading accounts
- **Spot & Futures**: Handles both spot trades and leveraged futures positions
- **REST API**: HTTP endpoints for querying portfolio state and importing historic trades
- **CLI**: `trader` command-line tool for humans and trading bots
- **Tax Data**: Captures cost basis, realized P&L, fees, and holding periods for tax reporting
- **Idempotent**: Duplicate trades are safely discarded (dedup by trade ID)
- **Rebuildable**: Positions can be rebuilt from trade history for audit/repair

---

## CLI

### Installation

**1. Install the `sn` CLI** (needed for login):

```bash
# go install
go install github.com/Signal-ngn/sn/cmd/sn@latest

# Homebrew (macOS)
brew install Signal-ngn/sn/sn
```

**2. Install the `trader` CLI:**

```bash
# go install
go install github.com/Signal-ngn/trader/cmd/trader@latest

# Homebrew (macOS)
brew install --cask Signal-ngn/trader/ledger
```

### Authentication

```bash
sn auth login          # opens browser — logs you in and stores your API key
trader accounts list   # works immediately; picks up the key from ~/.config/sn/config.yaml
```

For trading bots or CI, skip `sn` entirely and set `TRADER_API_KEY` directly:

```bash
export TRADER_API_KEY=your-api-key
trader accounts list
```

The tenant ID is resolved automatically on first use (via `GET /auth/resolve`) and cached in `~/.config/trader/config.yaml`.

### Commands

#### Accounts

```bash
trader accounts list           # list all accounts
trader accounts list --json    # JSON output
```

#### Portfolio

```bash
trader portfolio live          # open positions + total realized P&L
trader portfolio paper --json
```

#### Positions

```bash
trader positions live                    # open positions (default)
trader positions live --status closed    # closed positions
trader positions live --status all       # all positions
trader positions live --json
```

#### Trades

```bash
# List trades
trader trades list live                         # 50 most recent trades
trader trades list live --symbol BTC-USD        # filter by symbol
trader trades list live --side buy              # filter by side
trader trades list live --market-type futures   # filter by market type
trader trades list live --start 2025-01-01T00:00:00Z --end 2025-02-01T00:00:00Z
trader trades list live --limit 200             # up to 200 results
trader trades list live --limit 0               # all trades (follows all pages)
trader trades list live --json

# Record a trade
trader trades add live --symbol BTC-USD --side buy --quantity 0.1 --price 95000

# With fees and strategy metadata
trader trades add live \
  --symbol BTC-USD --side buy --quantity 0.1 --price 95000 \
  --fee 9.50 --strategy macd_momentum --confidence 0.78 \
  --stop-loss 93000 --take-profit 99000

# Futures long with leverage
trader trades add live \
  --symbol BTC-USD --side buy --quantity 0.5 --price 95000 \
  --market-type futures --leverage 10 --margin 4750
```

#### Orders

```bash
trader orders live                       # 50 most recent orders
trader orders live --status open         # open orders only
trader orders live --symbol BTC-USD
trader orders live --limit 0 --json      # all orders as JSON
```

#### Import

```bash
trader import trades.json          # import historic trades from file
trader import trades.json --json   # show full response JSON
```

The file must be a JSON object with a `"trades"` array matching the [trade event format](#nats-trade-events). Prints `Total / Inserted / Duplicates / Errors` and exits non-zero if any errors occurred.

#### Config

```bash
trader config show                              # show all config values and sources
trader config set trader_url https://...        # override service URL
trader config get trader_url
```

Config file: `~/.config/trader/config.yaml`

| Key | Default | Env override |
|-----|---------|-------------|
| `trader_url` | `https://signalngn-trader-potbdcvufa-ew.a.run.app` | `TRADER_URL` |
| `api_key` | _(from `~/.config/sn/config.yaml`)_ | `TRADER_API_KEY` |
| `tenant_id` | _(resolved automatically)_ | `TRADER_TENANT_ID` |

#### Global flags

```bash
trader --trader-url http://localhost:8080 accounts list   # one-off URL override
trader --json accounts list                               # JSON output (any command)
```

---

## Agent Skill

The ledger ships an [agent skill](https://agentskills.io) that gives AI coding agents full knowledge of the `trader` CLI — commands, flags, trade event format, and bot patterns. Install it so your agent can record trades and query portfolio state without needing to look anything up.

Works with Claude Code, Cursor, pi, Windsurf, Codex, and [many more](https://github.com/vercel-labs/skills).

### Install

```bash
npx skills add Signal-ngn/trader
```

For global installation (available in all projects):

```bash
npx skills add Signal-ngn/trader -g
```

### Usage

Once installed the skill is available as **`trader`** in your agent. Invoke it in any conversation where the agent needs to interact with the ledger:

```
Use the ledger skill to check my open positions before placing this trade.
```

The skill covers: accounts, portfolio, positions, trades, orders, import, config, NATS event format, and common trading bot patterns.

---

## Local Development

### Prerequisites

- Go 1.24+
- [Task](https://taskfile.dev/)
- Docker & Docker Compose

### Quick start

```bash
task setup   # creates .env, installs deps, starts infra
task dev     # run service with go run
```

### Tasks

```bash
task build            # build server binary → bin/ledger
task build:cli        # build CLI binary → bin/ledger
task build:all        # build both
task dev              # run server with go run
task test             # unit tests
task test:v           # verbose
task test:race        # with race detector
task test:integration # requires running infra
task test:all         # unit + integration + race
task test:cover       # coverage report
task fmt              # gofmt
task lint             # go vet + staticcheck
task infra:up         # start PostgreSQL + NATS
task infra:down       # stop containers
task infra:reset      # stop + delete volumes
task infra:status     # show running containers
task docker:build     # build Docker image
task deploy:production # deploy to Cloud Run
```

### Architecture

```
cmd/trader/          # CLI entry point
cmd/traderd/         # Server entry point
internal/
├── config/          # Configuration loading
├── domain/          # Core types: Trade, Position, Account, Order
├── store/           # PostgreSQL repository, migrations
├── ingest/          # NATS JetStream consumer, trade processing
└── api/             # REST handlers, routing, middleware
migrations/          # SQL migration files
```

Data flow: `Trading Bot → NATS → Ingestion → PostgreSQL ← REST API ← CLI / Dashboard`

---

## NATS Trade Events

The service subscribes to `trader.trades.>`.

**Subject format:** `trader.trades.<account>.<market_type>`

```json
{
  "tenant_id": "c2899e28-2bbe-47c1-8d29-84ee1a04fd37",
  "trade_id": "unique-trade-id",
  "account_id": "live",
  "symbol": "BTC-USD",
  "side": "buy",
  "quantity": 0.5,
  "price": 50000,
  "fee": 25,
  "fee_currency": "USD",
  "market_type": "spot",
  "timestamp": "2025-01-15T10:00:00Z"
}
```

`tenant_id` is required. Futures trades additionally include `leverage`, `margin`, `liquidation_price`, and optionally `funding_fee`.

---

## REST API

**Production URL:** `https://signalngn-trader-potbdcvufa-ew.a.run.app`

All `/api/v1/` and `/auth/resolve` endpoints require `Authorization: Bearer <api-key>`.

### Auth

```
GET /auth/resolve
→ {"tenant_id": "<uuid>"}
```

### Health

```
GET /health
→ {"status": "ok"}
```

### Accounts

```
GET  /api/v1/accounts
GET  /api/v1/accounts/{accountId}/portfolio
GET  /api/v1/accounts/{accountId}/positions?status=open|closed|all
GET  /api/v1/accounts/{accountId}/trades?symbol=&side=&market_type=&start=&end=&cursor=&limit=
GET  /api/v1/accounts/{accountId}/orders?status=&symbol=&cursor=&limit=
POST /api/v1/import
```

#### Import request body

```json
{
  "trades": [
    {
      "tenant_id": "c2899e28-2bbe-47c1-8d29-84ee1a04fd37",
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
    }
  ]
}
```

Max 1000 trades per request. Duplicate trade IDs are skipped. Re-importing is safe.

#### Import response

```json
{
  "total": 1,
  "inserted": 1,
  "duplicates": 0,
  "errors": 0,
  "results": [{"trade_id": "t-001", "status": "inserted"}]
}
```

---

## Tax Reporting

The ledger captures all data needed for tax reporting: cost basis, realized P&L, fees, holding periods, and exit reasons. You can export your trade history to CSV and submit it directly to your tax authority or import it into accounting software.

> **Note:** Tax regulations vary by jurisdiction. The ledger provides the raw transaction data — consult a tax professional for advice on how to apply it.

### Export round-trip positions to CSV

Each row is one complete trade (entry + exit pair), which is typically what tax authorities require:

```bash
go run ./cmd/trader trades list live --json | jq -r '
  ["RESULT","SYMBOL","DIR","SIZE","ENTRY","EXIT","PNL","PNL%","OPENED","CLOSED","EXIT_REASON"],
  (.[] | [
    (if .status == "open" then "open" elif .realized_pnl > 0 then "win" else "loss" end),
    .symbol,
    .side,
    .cost_basis,
    .avg_entry_price,
    (.exit_price // ""),
    .realized_pnl,
    (if .cost_basis > 0 then (.realized_pnl / .cost_basis * 100 | . * 100 | round | . / 100 | tostring) + "%" else "" end),
    .opened_at,
    (.closed_at // ""),
    (.exit_reason // "")
  ])
  | @csv
' > positions.csv
```

Use `--limit 0` to export the full history (all pages):

```bash
go run ./cmd/trader trades list live --limit 0 --json | jq -r '...' > positions_full.csv
```

### Export raw individual trades to CSV

If your tax authority requires every individual buy/sell transaction:

```bash
go run ./cmd/trader trades list live --raw --limit 0 --json | jq -r '
  ["TRADE_ID","SYMBOL","SIDE","QTY","PRICE","FEE","MARKET","TIMESTAMP"],
  (.[] | [.trade_id, .symbol, .side, .quantity, .price, .fee, .market_type, .timestamp])
  | @csv
' > trades.csv
```
