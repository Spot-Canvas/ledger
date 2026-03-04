---
name: trader
description: >
  Query and manage the Signal Ngn trading ledger using the `trader` CLI.
  Use this skill when you need to record trades, check open positions,
  inspect portfolio state, stream live trade events, or query trade history.
allowed-tools: Bash
---

# trader CLI

`trader` is the command-line interface for the Signal Ngn trader service. It lets you record executed trades, query live portfolio state, stream real-time trade events, and inspect trade history — from a terminal or a trading bot script.

**Production service URL:** `https://signalngn-trader-potbdcvufa-ew.a.run.app`

---

## Installation

```bash
# Homebrew (macOS)
brew install --cask Signal-ngn/trader/trader

# Go toolchain
go install github.com/Signal-ngn/trader/cmd/trader@latest
```

---

## Authentication

The CLI reads your API key from three places, in order:

1. `TRADER_API_KEY` env var
2. `api_key` in `~/.config/trader/config.yaml` (written by `trader auth login`)
3. `api_key` in `~/.config/sn/config.yaml` (fallback for users who logged in via sn)

```bash
# One-time browser login (primary path — no sn required)
trader auth login
trader accounts list   # works immediately

# For bots / CI — no login needed
export TRADER_API_KEY=your-api-key
trader accounts list
```

The tenant ID is resolved automatically on first use (via `GET /auth/resolve`) and cached in `~/.config/trader/config.yaml`. Override with `TRADER_TENANT_ID`.

---

## Global flags

These work on every command:

| Flag | Description |
|---|---|
| `--trader-url <url>` | Override ledger service URL for this invocation |
| `--api-url <url>` | Override platform API URL for this invocation |
| `--ingestion-url <url>` | Override ingestion server URL for this invocation |
| `--web-url <url>` | Override web app URL for this invocation |
| `--json` | Output raw JSON instead of the default table |
| `--version` | Print CLI version |

```bash
trader --trader-url http://localhost:8080 accounts list
trader --api-url http://localhost:9090 strategies list
trader --json portfolio live
```

---

## Commands

### accounts

```bash
trader accounts list                          # list all accounts for the tenant
trader accounts list --json

trader accounts show <account-id>             # aggregate stats: trades, win rate, P&L, balance
trader accounts show live --json

trader accounts balance set <account-id> <amount>          # set USD cash balance
trader accounts balance set live 50000
trader accounts balance set live 40000 --currency EUR

trader accounts balance get <account-id>                   # query current balance
trader accounts balance get live
trader accounts balance get live --currency EUR
trader accounts balance get live --json
```

`accounts show` response fields: `total_trades`, `closed_trades`, `win_count`, `loss_count`, `win_rate`, `total_realized_pnl`, `open_positions`, `balance` (omitted when not set).

The balance is adjusted automatically by trade ingestion: buys deduct cost, sells credit realised P&L. `balance set` always overwrites — use it to set an initial value or correct after broker reconciliation.

---

### portfolio

```bash
trader portfolio <account-id>       # open positions + total realized P&L
trader portfolio live
trader portfolio paper --json
```

---

### positions

```bash
trader positions <account-id>                     # open positions (default)
trader positions live
trader positions live --status closed             # closed positions
trader positions live --status all                # all positions
trader positions live --json
```

`--status` values: `open` (default), `closed`, `all`.

Position fields: `symbol`, `side` (`long`/`short`), `market_type` (`spot`/`futures`), `quantity`, `avg_entry_price`, `cost_basis`, `realized_pnl`, `status`.

**Check if already in a position before trading:**

```bash
trader positions live --json | jq '.[] | select(.symbol == "BTC-USD")'
```

---

### trades list

`trades list` has two modes:

**Round-trip view (default):** shows one row per complete trade cycle (entry + exit), with win/loss result, P&L, and P&L%. Best for reviewing performance.

**Raw view (`--raw`):** shows individual buy/sell legs. Required when filtering by symbol, side, or date range.

```bash
# Round-trip view (default)
trader trades list live
trader trades list live --limit 20
trader trades list live --long               # adds position ID, full timestamps, exit reason
trader trades list live --json               # JSON array of position objects

# Raw individual trades
trader trades list live --raw
trader trades list live --raw --symbol BTC-USD
trader trades list live --raw --side buy
trader trades list live --raw --market-type futures
trader trades list live --raw --start 2025-01-01T00:00:00Z --end 2025-02-01T00:00:00Z
trader trades list live --raw --limit 200
trader trades list live --raw --limit 0      # all pages
trader trades list live --raw --long         # adds trade ID column, full timestamps
trader trades list live --raw --json
```

Filters (`--symbol`, `--side`, `--market-type`, `--start`, `--end`) are **only applied in `--raw` mode**. A warning is printed if you use them without `--raw`.

`--limit 0` fetches all pages. Default is 50.

---

### trades add

Record a single trade immediately after execution.

```bash
trader trades add <account-id> [flags]
```

**Required flags:** `--symbol`, `--side`, `--quantity`, `--price`

```bash
# Minimal spot buy
trader trades add live --symbol BTC-USD --side buy --quantity 0.1 --price 95000

# With fee and strategy metadata
trader trades add live \
  --symbol BTC-USD --side buy --quantity 0.1 --price 95000 \
  --fee 9.50 --strategy macd_momentum --confidence 0.78 \
  --stop-loss 93000 --take-profit 99000

# Spot sell (exit)
trader trades add live \
  --symbol BTC-USD --side sell --quantity 0.1 --price 98000 \
  --fee 9.80 --exit-reason "take-profit hit"

# Futures long with leverage
trader trades add live \
  --symbol BTC-USD --side buy --quantity 0.5 --price 95000 \
  --market-type futures --leverage 10 --margin 4750

# Explicit trade ID and timestamp
trader trades add paper \
  --trade-id "bot-20250201-042" \
  --symbol ETH-USD --side buy --quantity 1.0 --price 3200 \
  --timestamp 2025-02-01T10:30:00Z

trader trades add live --symbol BTC-USD --side buy --quantity 0.1 --price 95000 --json
```

**All flags:**

| Flag | Default | Description |
|---|---|---|
| `--trade-id` | auto UUID | Unique trade identifier — resubmitting the same ID is safe (idempotent) |
| `--symbol` | *(required)* | Trading pair, e.g. `BTC-USD` |
| `--side` | *(required)* | `buy` or `sell` |
| `--quantity` | *(required)* | Trade size |
| `--price` | *(required)* | Fill price |
| `--fee` | `0` | Fee paid |
| `--fee-currency` | `USD` | Fee currency |
| `--market-type` | `spot` | `spot` or `futures` |
| `--timestamp` | now | Execution time (RFC3339) |
| `--strategy` | | Strategy name |
| `--entry-reason` | | Why the position was entered |
| `--exit-reason` | | Why the position was exited |
| `--confidence` | | Signal confidence (0–1) |
| `--stop-loss` | | Stop-loss price |
| `--take-profit` | | Take-profit price |
| `--leverage` | | Leverage multiplier (futures) |
| `--margin` | | Margin used (futures) |
| `--liquidation-price` | | Liquidation price (futures) |
| `--funding-fee` | | Funding fee (futures) |

---

### trades watch

Stream live trade events for an account to stdout as JSONL. Reconnects automatically every 5 s on disconnect. Exit with `Ctrl-C`.

```bash
trader trades watch <account-id>
trader trades watch live
trader trades watch paper | jq .       # pretty-print with jq
```

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

Optional fields (`strategy`, `confidence`, `stop_loss`, `take_profit`, `entry_reason`, `exit_reason`) are omitted when not set.

---

### trades delete

```bash
trader trades delete <trade-id> --confirm
trader trades delete abc-123 --confirm --json
```

Fails if the trade contributes to an open position. The `--confirm` flag is required.

---

### orders

```bash
trader orders <account-id>
trader orders live                           # 50 most recent orders
trader orders live --status open             # open orders only
trader orders live --status filled
trader orders live --symbol BTC-USD
trader orders live --limit 0 --json          # all orders as JSON
```

`--status` values: `open`, `filled`, `partially_filled`, `cancelled`.

Order fields: `order_id`, `symbol`, `side`, `order_type` (`market`/`limit`), `requested_qty`, `filled_qty`, `avg_fill_price`, `status`, `market_type`, `created_at`.

---

### import

Bulk-load historic trades from a JSON file. Validates all trades up front, inserts idempotently (duplicate IDs are skipped), rebuilds positions.

```bash
trader import trades.json
trader import trades.json --json    # full response JSON
```

Prints `Total: N  Inserted: N  Duplicates: N  Errors: N`. Exits non-zero if any errors occurred. Safe to re-run.

**File format** — JSON object with a `"trades"` array:

```json
{
  "trades": [
    {
      "tenant_id": "c2899e28-2bbe-47c1-8d29-84ee1a04fd37",
      "trade_id": "cb-20250101-001",
      "account_id": "live",
      "symbol": "BTC-USD",
      "side": "buy",
      "quantity": 0.1,
      "price": 95000,
      "fee": 9.50,
      "fee_currency": "USD",
      "market_type": "spot",
      "timestamp": "2025-01-01T10:00:00Z"
    }
  ]
}
```

Required fields: `tenant_id`, `trade_id`, `account_id`, `symbol`, `side`, `quantity`, `price`, `fee`, `fee_currency`, `market_type`, `timestamp`. Max 1000 trades per request.

---

### auth

```bash
trader auth login      # open browser OAuth flow; write api_key to ~/.config/trader/config.yaml
trader auth logout     # remove api_key from config
trader auth status     # show whether authenticated; print masked key
```

`auth login` builds the URL as `{web_url}/oauth/start?cli_port=<port>`. Falls back to `api_url` if `web_url` is not set. Times out after 120 seconds.

---

### strategies

```bash
trader strategies list                          # all built-in + user strategies
trader strategies list --active                 # only active user strategies + all built-ins
trader strategies list --json

trader strategies get <id>                      # detail + optional source code
trader strategies get 42 --json

trader strategies validate --name <name> --file <path>    # validate source file
trader strategies create --name <name> --file <path>      # create user strategy
trader strategies create --name x --file x.star \
  --description "..." --params '{"THRESHOLD":2.0}'
trader strategies update <id> --file <path>               # update source
trader strategies update 42 --file updated.star --description "new desc"
trader strategies activate <id>
trader strategies deactivate <id>
trader strategies delete <id>

# Backtest a user strategy (synchronous)
trader strategies backtest <id> \
  --exchange coinbase --product BTC-USD --granularity ONE_HOUR
trader strategies backtest 42 \
  --exchange coinbase --product BTC-USD --granularity ONE_HOUR \
  --mode futures-long --leverage 5 --json
```

`strategies list` output: `TYPE` (`builtin`/`user`), `NAME`, `DESCRIPTION`, `ACTIVE` (`-` for built-ins, `yes`/`no` for user).

`strategies backtest` flags: `--exchange` (req), `--product` (req), `--granularity` (req), `--mode` (default `spot`), `--start`, `--end`, `--leverage`.

---

### trading

```bash
trader trading list                             # all trading configs
trader trading list live                        # filter by account ID
trader trading list --enabled                   # only enabled configs
trader trading list --json

trader trading get <account> <exchange> <product>
trader trading get live coinbase BTC-USD
trader trading get live coinbase BTC-USD --json

# Create or update (unset flags preserve existing values)
trader trading set <account> <exchange> <product> [flags]
trader trading set live coinbase BTC-USD --granularity ONE_HOUR --spot ml_xgboost --enable
trader trading set live coinbase BTC-USD \
  --params ml_xgboost:confidence=0.80 --params ml_xgboost:exit_confidence=0.40
trader trading set live coinbase BTC-USD --params ml_xgboost:clear   # remove all params
trader trading set live coinbase BTC-USD --disable
trader trading set live coinbase BTC-USD --json

trader trading delete <account> <exchange> <product>
trader trading delete live coinbase BTC-USD
```

`trading set` flags:

| Flag | Description |
|---|---|
| `--granularity` | Candle granularity (e.g. `ONE_HOUR`) |
| `--long` | Long strategies (comma-separated) |
| `--short` | Short strategies (comma-separated) |
| `--spot` | Spot strategies (comma-separated) |
| `--long-leverage` | Long leverage multiplier |
| `--short-leverage` | Short leverage multiplier |
| `--trend-filter` | Enable trend filter |
| `--no-trend-filter` | Disable trend filter |
| `--enable` | Enable the config |
| `--disable` | Disable the config |
| `--params` | `<strategy>:<key>=<value>` or `<strategy>:clear` (repeatable) |

---

### price

```bash
trader price <product>                          # live price, default exchange=coinbase, granularity=ONE_MINUTE
trader price BTC-USD
trader price BTC-USD --exchange kraken --granularity ONE_HOUR
trader price BTC-USD --json

trader price --all                              # all enabled products (concurrent)
trader price --all --json                       # JSON array of successful results only
```

Output columns: `EXCHANGE`, `PRODUCT`, `PRICE`, `OPEN`, `HIGH`, `LOW`, `VOLUME`, `AGE`.  
`AGE` is prefixed with `!` when the price is stale (> 1 hour since `last_update`).

---

### backtest

```bash
# Submit and wait for result (synchronous)
trader backtest run \
  --exchange coinbase --product BTC-USD --strategy ml_xgboost --granularity ONE_HOUR
trader backtest run \
  --exchange coinbase --product BTC-USD --strategy ml_xgboost --granularity ONE_HOUR \
  --mode futures-long --leverage 5 --trend-filter \
  --params confidence=0.80 --params exit_confidence=0.40

# No-wait: print job ID and exit
trader backtest run --exchange coinbase --product BTC-USD \
  --strategy ml_xgboost --granularity ONE_HOUR --no-wait

# List results
trader backtest list                            # 20 most recent
trader backtest list --strategy ml_xgboost
trader backtest list --sort winrate             # highest win rate first
trader backtest list --limit 0                  # all results
trader backtest list --json

# Get full detail
trader backtest get <id>
trader backtest get 123 --json

# Poll a job
trader backtest job <job-id>                    # prints result or poll hint
```

`backtest run` required flags: `--exchange`, `--product`, `--strategy`, `--granularity`.  
Optional: `--mode` (default `spot`), `--start`, `--end`, `--leverage`, `--trend-filter`, `--no-wait`, `--params` (repeatable `key=value`).

---

### signals

```bash
# Stream all signals from your enabled trading configs
trader signals

# Filter
trader signals --exchange coinbase --product BTC-USD
trader signals --granularity ONE_HOUR
trader signals --strategy ml_xgboost

# Machine-readable
trader signals --json   # one JSON object per signal line
```

On startup the command:
1. Calls `GET /config/trading` to build an allowlist of your enabled strategy slots
2. Connects to NATS at `nats_url` using embedded read-only NGS credentials
3. Subscribes to `signals.<exchange>.<product>.<granularity>.<strategy>` with `*`/`>` wildcards for unset filters
4. Filters received signals to your allowlist (prefix-tolerant: `ml_xgboost_short` matches `ml_xgboost`)

Press `Ctrl-C` to stop. Prints `Unsubscribing...` on exit.

Use `nats_creds_file` config key to override the embedded credentials.

---

### config

```bash
trader config show                        # all resolved values and their sources
trader config set trader_url https://...  # write to ~/.config/trader/config.yaml
trader config set api_key sk-...
trader config set api_url https://my-api.example.com
trader config get trader_url
```

Config file: `~/.config/trader/config.yaml`

| Key | Default | Env override |
|---|---|---|
| `trader_url` | `https://signalngn-trader-potbdcvufa-ew.a.run.app` | `TRADER_URL` |
| `api_key` | *(from `~/.config/sn/config.yaml`)* | `TRADER_API_KEY` |
| `tenant_id` | *(resolved via `/auth/resolve` on first use)* | `TRADER_TENANT_ID` |
| `api_url` | `https://signalngn-api-potbdcvufa-ew.a.run.app` | `TRADER_API_URL` |
| `web_url` | *(none)* | `TRADER_WEB_URL` |
| `ingestion_url` | `https://signalngn-ingestion-potbdcvufa-ew.a.run.app` | `TRADER_INGESTION_URL` |
| `nats_url` | `tls://connect.ngs.global` | `TRADER_NATS_URL` |
| `nats_creds_file` | *(embedded credentials)* | `TRADER_NATS_CREDS_FILE` |

---

## Common bot patterns

### Check live price before sizing a position

```bash
# Get the current BTC-USD price
PRICE=$(trader price BTC-USD --json | jq '.close')

# Get price for a specific exchange/granularity
PRICE=$(trader price BTC-USD --exchange coinbase --granularity ONE_HOUR --json | jq '.close')

# Check all enabled products — useful for scanning before entering positions
trader price --all --json | jq '.[] | {product: .product_id, price: .close}'
```

### Check live price staleness before acting

```bash
# fmtAge returns "!2h 15m" when stale (> 1 hour). Use --json to get the raw timestamp.
LAST_UPDATE=$(trader price BTC-USD --json | jq -r '.last_update')
```

### List enabled trading configs

```bash
# See what the bot has active
trader trading list --enabled --json

# Check a specific config
trader trading get live coinbase BTC-USD --json
```

### Run a backtest before enabling a strategy

```bash
# Synchronous run — waits for result
trader backtest run \
  --exchange coinbase --product BTC-USD \
  --strategy ml_xgboost --granularity ONE_HOUR \
  --params confidence=0.80 --json

# List recent backtests sorted by win rate
trader backtest list --strategy ml_xgboost --sort winrate --json | jq '.results[:5]'
```

### Stream signals and act on them in a script

```bash
# Stream as JSON lines, pipe to jq for filtering
trader signals --json | jq --unbuffered 'select(.action == "BUY" and .confidence >= 0.8)'
```

### Check available balance before sizing a position

```bash
BALANCE=$(trader accounts balance get live --json | jq '.amount')
```

### Check current exposure before entering

```bash
# Is BTC-USD already open?
trader positions live --json | jq '.[] | select(.symbol == "BTC-USD" and .status == "open")'

# Total open position count
trader positions live --json | jq 'length'
```

### Record a trade immediately after execution

```bash
trader trades add live \
  --trade-id "$EXCHANGE_ORDER_ID" \
  --symbol BTC-USD --side buy --quantity 0.1 --price 95000 \
  --fee 9.50 --strategy macd_momentum --confidence 0.78 \
  --stop-loss 93000 --take-profit 99000
```

### Get realised P&L since midnight

```bash
TODAY=$(date -u +%Y-%m-%dT00:00:00Z)
trader trades list live --raw --start "$TODAY" --json | \
  jq '[.[] | select(.side == "sell")] | map(.realized_pnl) | add // 0'
```

### Verify a trade was recorded

```bash
trader trades list live --raw --json | jq '.[] | select(.trade_id == "my-order-id")'
```

### Point at a local instance for testing

```bash
TRADER_URL=http://localhost:8080 trader accounts list
# or permanently:
trader config set trader_url http://localhost:8080
```
