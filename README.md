# Trader

A Go trading engine for the Signal Ngn ecosystem. It ingests trade events via NATS JetStream, maintains portfolio state (positions, P&L), exposes data through a REST API and CLI, and runs a signal-driven trading engine that can execute paper or live trades autonomously.

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
- **Trading Engine**: Signal-driven engine goroutine — subscribes to Synadia NGS signals, executes paper or live (Binance Futures) trades, enforces risk rules (SL/TP, trailing stop, max hold time, daily loss limit, kill switch)
- **Live Trade Stream**: SSE endpoint streams trade events in real time to the CLI and dashboards

---

## CLI

### Installation

**Install the `trader` CLI:**

```bash
# go install
go install github.com/Signal-ngn/trader/cmd/trader@latest

# Homebrew (macOS)
brew install --cask Signal-ngn/trader/ledger
```

### Authentication

```bash
trader auth login      # opens browser — logs you in and stores your API key
trader accounts list   # works immediately
```

If you previously ran `sn auth login`, the key from `~/.config/sn/config.yaml` is picked up automatically — no re-login needed.

For trading bots or CI, set `TRADER_API_KEY` directly:

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

# Stream live trade events (SSE → JSONL to stdout)
trader trades watch live
trader trades watch paper
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

#### Auth

```bash
trader auth login      # open browser, complete OAuth, write api_key to config
trader auth logout     # remove api_key from ~/.config/trader/config.yaml
trader auth status     # show whether you are authenticated and which key is active
```

#### Auth login flow

`trader auth login` opens your browser to the platform login page, listens for the OAuth callback on a local port, writes the `api_key` to `~/.config/trader/config.yaml`, and prints `Authenticated as <email>`.

If the browser cannot open automatically the URL is printed for manual navigation. The command times out after 120 seconds.

#### Strategies

```bash
trader strategies list                          # list all built-in and user strategies
trader strategies list --active                 # only active user strategies + all built-ins
trader strategies list --json

trader strategies get <id>                      # detail view + source code (if any)
trader strategies get 42 --json

trader strategies validate --name my_strat --file strat.star   # validate source file
trader strategies create --name my_strat --file strat.star     # create user strategy
trader strategies create --name x --file x.star --description "My strat" --params '{"THRESHOLD":2.0}'
trader strategies update 42 --file updated.star                # update source
trader strategies activate 42
trader strategies deactivate 42
trader strategies delete 42

# Backtest a user strategy
trader strategies backtest 42 \
  --exchange coinbase --product BTC-USD --granularity ONE_HOUR
trader strategies backtest 42 \
  --exchange coinbase --product BTC-USD --granularity ONE_HOUR \
  --mode futures-long --leverage 5
```

`strategies list` output columns: `TYPE` (`builtin`/`user`), `NAME`, `DESCRIPTION`, `ACTIVE` (`yes`/`no` for user, `-` for built-in).

#### Trading

```bash
trader trading list                                  # all trading configs
trader trading list live                             # filtered to account
trader trading list --enabled                        # only enabled configs
trader trading list --json

trader trading get live coinbase BTC-USD             # detail view
trader trading get live coinbase BTC-USD --json

# Create or update a config (unset flags preserve existing values)
trader trading set live coinbase BTC-USD \
  --granularity ONE_HOUR --spot ml_xgboost --enable
trader trading set live coinbase BTC-USD \
  --params ml_xgboost:confidence=0.80 --params ml_xgboost:exit_confidence=0.40
trader trading set live coinbase BTC-USD --params ml_xgboost:clear   # remove all params
trader trading set live coinbase BTC-USD --disable

trader trading delete live coinbase BTC-USD          # delete a config
```

`trading set` flags: `--granularity`, `--long`, `--short`, `--spot`, `--long-leverage`, `--short-leverage`, `--trend-filter`, `--no-trend-filter`, `--enable`, `--disable`, `--params`.

#### Price

```bash
trader price BTC-USD                                # live price (default: coinbase, ONE_MINUTE)
trader price BTC-USD --exchange kraken --granularity ONE_HOUR
trader price BTC-USD --json

trader price --all                                  # all enabled products
trader price --all --json                           # JSON array of successful results
```

Output columns: `EXCHANGE`, `PRODUCT`, `PRICE`, `OPEN`, `HIGH`, `LOW`, `VOLUME`, `AGE`. The `AGE` column is prefixed with `!` when the price is more than 1 hour stale.

#### Backtest

```bash
# Submit and wait for result
trader backtest run \
  --exchange coinbase --product BTC-USD --strategy ml_xgboost --granularity ONE_HOUR
trader backtest run \
  --exchange coinbase --product BTC-USD --strategy ml_xgboost --granularity ONE_HOUR \
  --mode futures-long --leverage 5 --trend-filter \
  --params confidence=0.80 --params exit_confidence=0.40

# No-wait mode — print job ID and exit
trader backtest run --exchange coinbase --product BTC-USD \
  --strategy ml_xgboost --granularity ONE_HOUR --no-wait

# List and inspect results
trader backtest list                                # 20 most recent (default)
trader backtest list --strategy ml_xgboost --limit 50
trader backtest list --sort winrate
trader backtest list --limit 0                      # all results
trader backtest list --json

trader backtest get 123                             # full detail table
trader backtest get 123 --json

trader backtest job <job-id>                        # poll job status / fetch result
```

`backtest run` required flags: `--exchange`, `--product`, `--strategy`, `--granularity`. Optional: `--mode` (default `spot`), `--start`, `--end`, `--leverage`, `--trend-filter`, `--no-wait`, `--params`.

#### Signals

```bash
# Stream all signals from your enabled trading configs (Ctrl-C to stop)
trader signals

# Filter to specific exchange/product/granularity/strategy
trader signals --exchange coinbase --product BTC-USD
trader signals --exchange coinbase --product BTC-USD --granularity ONE_HOUR
trader signals --strategy ml_xgboost

# Machine-readable output (one JSON object per line)
trader signals --json
```

On startup `trader signals`:
1. Calls `GET /config/trading` to build an allowlist of your enabled strategy slots
2. Connects to NATS at `nats_url` using embedded read-only credentials
3. Subscribes to `signals.<exchange>.<product>.<granularity>.<strategy>` using wildcards for unset filters
4. Only prints signals that match your allowlist

Use `nats_creds_file` to override the embedded credentials (e.g., for custom NGS accounts).

#### Config

```bash
trader config show                              # show all config values and sources
trader config set trader_url https://...        # override service URL
trader config set api_url https://...           # override platform API URL
trader config get trader_url
```

Config file: `~/.config/trader/config.yaml`

| Key | Default | Env override |
|-----|---------|-------------|
| `trader_url` | `https://signalngn-trader-potbdcvufa-ew.a.run.app` | `TRADER_URL` |
| `api_key` | _(from `~/.config/sn/config.yaml`)_ | `TRADER_API_KEY` |
| `tenant_id` | _(resolved automatically)_ | `TRADER_TENANT_ID` |
| `api_url` | `https://signalngn-api-potbdcvufa-ew.a.run.app` | `TRADER_API_URL` |
| `web_url` | _(none)_ | `TRADER_WEB_URL` |
| `ingestion_url` | `https://signalngn-ingestion-potbdcvufa-ew.a.run.app` | `TRADER_INGESTION_URL` |
| `nats_url` | `tls://connect.ngs.global` | `TRADER_NATS_URL` |
| `nats_creds_file` | _(embedded credentials)_ | `TRADER_NATS_CREDS_FILE` |

#### Global flags

```bash
trader --trader-url http://localhost:8080 accounts list   # ledger URL override
trader --api-url http://localhost:9090 strategies list    # platform API URL override
trader --ingestion-url http://localhost:9091 strategies validate --name x --file x.star
trader --web-url http://localhost:3000 auth login         # web app URL override
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
├── engine/          # Trading engine goroutine (signals, positions, risk, exchange)
└── api/             # REST handlers, routing, middleware, SSE stream
migrations/          # SQL migration files
```

Data flows:

```
Trading Bot  → NATS JetStream → Ingestion → PostgreSQL ← REST API ← CLI / Dashboard
                                                 ↑
NGS Signals → Engine ──────────────────────────→┘
                  └→ Exchange (Binance Futures / Noop)
```

---

## Trading Engine

The `traderd` server includes an optional signal-driven trading engine. When enabled it subscribes to [Synadia NGS](https://www.synadia.com/ngs) signals, filters them against your configured trading strategies, and executes paper or live trades — writing every fill to the ledger through the same path as manual ingestion.

### Enable

Set `TRADING_ENABLED=true` in your environment or `.env` file. The engine starts as a goroutine inside `traderd` — no separate binary is needed.

### Modes

| Mode | `TRADING_MODE` | Exchange |
|---|---|---|
| Paper | `paper` (default) | Synthetic fills at signal price, zero fees |
| Live | `live` | Binance Futures (raw HTTP, HMAC-SHA256 signed) |

### Configuration

| Env var | Default | Description |
|---|---|---|
| `TRADING_ENABLED` | `false` | Set `true` to start the engine |
| `TRADING_MODE` | `paper` | `paper` or `live` |
| `TRADER_ACCOUNT` | `paper` | Account ID all engine trades are booked under |
| `STRATEGY_FILTER` | — | Optional prefix — only process signals whose strategy starts with this string |
| `PORTFOLIO_SIZE` | `10000` | Total portfolio size in USD |
| `POSITION_SIZE_PCT` | `10` | Default position size as % of portfolio (overridden per-signal by `position_pct`) |
| `MAX_POSITION_SIZE` | `0` | Max position size in USD (0 = no cap) |
| `MIN_POSITION_SIZE` | `0` | Min position size in USD (0 = no floor) |
| `MAX_POSITIONS` | `0` | Max concurrent open positions (0 = no cap) |
| `DAILY_LOSS_LIMIT` | `0` | Halt new opens once realised losses today exceed this USD amount (0 = disabled) |
| `KILL_SWITCH_FILE` | `/tmp/trader.kill` | Touch this file to halt all new opens immediately — existing positions are still risk-managed |
| `SN_API_KEY` | — | SignalNGN API key (required when `TRADING_ENABLED=true`) |
| `SN_API_URL` | `https://api.signal-ngn.com` | SignalNGN API base URL |
| `SN_NATS_CREDS_FILE` | — | Path to custom NGS NATS credentials file (embedded subscribe-only key used by default) |
| `BINANCE_API_KEY` | — | Binance API key (live mode only) |
| `BINANCE_API_SECRET` | — | Binance API secret (live mode only) |

### Signal pipeline

Every incoming NGS signal passes through these checks before a trade is placed:

1. **Allowlist** — fetched from `GET /config/trading` on the SN API, rebuilt every 5 minutes; only enabled trading configs are allowed
2. **Strategy filter** — optional `STRATEGY_FILTER` prefix match
3. **Staleness** — signals older than 2 minutes are dropped
4. **Confidence** — `BUY`/`SHORT` signals with `confidence < 0.5` are dropped
5. **Cooldown** — a 5-minute per-(symbol, action) cooldown prevents re-entering immediately after an open
6. **Kill switch** — if `KILL_SWITCH_FILE` exists, new opens are skipped (closes still execute)
7. **Daily loss limit** — queried live from the DB; counts all realised losses since midnight UTC
8. **Direction conflict** — won't open a new position in the opposite direction to an existing one
9. **Max positions** — won't exceed `MAX_POSITIONS` concurrent open positions

### Risk management

The risk loop runs every 5 minutes and evaluates every open position:

| Rule | Default | Notes |
|---|---|---|
| **Stop-loss** | −4% from entry | Uses signal `stop_loss` if provided and > 0.1% from entry |
| **Take-profit** | +10% from entry | Uses signal `take_profit` if provided and > 0.1% from entry |
| **Trailing stop** | Activates at +3% unrealised gain; trails 2% behind peak | Scaled by `1/leverage` for futures; never loosens |
| **Max hold time** | 48 hours | Position is closed regardless of P&L |

Price used for evaluation: last price seen in a received NGS signal → SN price API fallback → skip tick (warning logged).

### Live trade stream

```bash
# Stream all trades for an account to stdout as JSONL
trader trades watch live

# Pipe to jq for pretty-printing
trader trades watch paper | jq .
```

Reconnects automatically every 5 seconds on disconnect. Exit with `Ctrl-C`.

Each event is a JSON object:

```json
{
  "trade_id": "engine-live-BTC-USD-1740912345678901234",
  "account_id": "live",
  "symbol": "BTC-USD",
  "side": "sell",
  "quantity": 0.021,
  "price": 96800.0,
  "fee": 0,
  "market_type": "futures",
  "timestamp": "2026-03-02T11:00:00Z",
  "strategy": "ml_xgboost",
  "confidence": 0.82,
  "stop_loss": 93200.0,
  "take_profit": 104000.0,
  "exit_reason": "take profit"
}
```

The SSE endpoint is also available directly:

```
GET /api/v1/accounts/{accountId}/trades/stream
Authorization: Bearer <api-key>
Accept: text/event-stream
```

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
GET  /api/v1/accounts/{accountId}/trades/stream   (SSE — see Trading Engine section)
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
