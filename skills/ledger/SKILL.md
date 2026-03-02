---
name: trader
description: Record trades, query positions, and inspect portfolio state using the `trader` CLI. Use this skill when a trading bot needs to persist executed trades, check open positions before sizing a new trade, query realized P&L, or import historic trade history.
allowed-tools: Bash
---

# trader — Spot Canvas Trading Ledger CLI

`trader` is the command-line interface for the Spot Canvas ledger service. Trading bots use it to record executed trades, query live portfolio state, and inspect trade history.

## Prerequisites

### Install

```bash
# Go toolchain
go install github.com/Signal-ngn/trader/cmd/trader@latest

# Homebrew (macOS)
brew install --cask Signal-ngn/trader/trader
```

### Authenticate

The `trader` CLI reads your API key from `~/.config/sn/config.yaml` (written by `sn auth login`):

```bash
sn auth login        # one-time browser login
trader accounts list # verify it works
```

For bots and CI, set the API key directly via environment variable — no config file needed:

```bash
export TRADER_API_KEY=your-api-key
```

The tenant ID is resolved automatically on first use and cached in `~/.config/trader/config.yaml`. Override with `TRADER_TENANT_ID`.

---

## Accounts

```bash
trader accounts list                        # list all accounts for this tenant
trader accounts list --json                 # JSON array
trader accounts show <account-id>           # aggregate stats: trades, win rate, P&L, balance
trader accounts show <account-id> --json    # raw JSON
```

**Response fields:** `id`, `name`, `type` (`live`/`paper`), `created_at`

Common account IDs: `live` (real trading), `paper` (simulated).

`accounts show` includes a `Balance (USD)` row when a balance has been set, or displays `not set`.

---

## Account Balance

The ledger tracks a cash balance per account. Set an initial balance once, then query it before sizing positions. **Balance is automatically adjusted by trade ingestion** — opening a position deducts the cost and closing a position credits the realised P&L.

```bash
trader accounts balance set live 50000                    # set USD balance (overwrites any existing value)
trader accounts balance set live 40000 --currency EUR     # set a non-USD balance
trader accounts balance get live                          # query current USD balance
trader accounts balance get live --currency EUR           # query non-USD balance
trader accounts balance get live --json                   # raw JSON
```

**Automatic balance adjustments during ingestion:**

| Trade event | Balance change |
|---|---|
| Spot buy (open / add to position) | − `quantity × price + fee` |
| Spot sell (partial or full close) | + realised P&L |
| Futures open | − margin (`margin` field; or `cost_basis / leverage`; skipped if neither available) |
| Futures close (partial or full) | + realised P&L (leverage- and fee-adjusted) |

Adjustments are a **no-op** when no balance row exists — the row is never auto-created. Position rebuild does not touch the balance.

`PUT /balance` always overwrites — use it to set an initial balance or to correct it after broker reconciliation.

---

## Portfolio

```bash
trader portfolio live          # open positions + total realized P&L + balance (when set)
trader portfolio paper
trader portfolio live --json
```

**Response fields:**
- `positions[]` — open positions (see Positions below)
- `total_realized_pnl` — sum of realized P&L across all positions (open + closed)
- `balance` — current USD balance (omitted when no balance has been set)

Use this before placing a new trade to check current exposure.

---

## Positions

```bash
trader positions live                    # open positions (default)
trader positions live --status closed    # closed positions
trader positions live --status all       # all positions
trader positions live --json
```

**Position fields:**

| Field | Description |
|-------|-------------|
| `id` | Position ID |
| `symbol` | e.g. `BTC-USD` |
| `market_type` | `spot` or `futures` |
| `side` | `long` or `short` |
| `quantity` | Current size |
| `avg_entry_price` | Volume-weighted average entry |
| `cost_basis` | Total cost of the position |
| `realized_pnl` | Realized P&L so far |
| `status` | `open` or `closed` |
| `opened_at` | When the position was opened |
| `closed_at` | When it was closed (if closed) |
| `stop_loss` | Stop-loss price (if set) |
| `take_profit` | Take-profit price (if set) |

**Pattern — check if already in a position before trading:**

```bash
# Is there an open BTC-USD position?
trader positions live --json | jq '.[] | select(.symbol == "BTC-USD" and .status == "open")'
```

---

## Trades

### List trades

```bash
trader trades list <account-id>                              # 50 most recent trades (default)
trader trades list live --symbol BTC-USD                     # filter by symbol
trader trades list live --side buy                           # filter by side: buy or sell
trader trades list live --market-type futures                # filter by market type
trader trades list live --start 2025-01-01T00:00:00Z --end 2025-02-01T00:00:00Z
trader trades list live --limit 200                          # up to 200 results
trader trades list live --limit 0                            # all trades (follows all cursor pages)
trader trades list live --json
```

### Record a single trade

```bash
trader trades add <account-id> --symbol BTC-USD --side buy --quantity 0.1 --price 95000
```

**Required flags:** `--symbol`, `--side`, `--quantity`, `--price`

**Optional flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--trade-id` | auto-generated UUID | Unique trade identifier |
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

**Examples:**

```bash
# Spot buy with strategy metadata
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

# With explicit trade ID and timestamp
trader trades add paper \
  --trade-id "bot-20250201-042" \
  --symbol ETH-USD --side buy --quantity 1.0 --price 3200 \
  --timestamp 2025-02-01T10:30:00Z

# JSON output
trader trades add live --symbol BTC-USD --side buy --quantity 0.1 --price 95000 --json
```

**Trade is idempotent** — re-submitting the same `--trade-id` is safe, returns duplicate status.

### Trade fields (list output)

| Field | Description |
|-------|-------------|
| `trade_id` | Unique trade identifier |
| `symbol` | Trading pair |
| `side` | `buy` or `sell` |
| `quantity` | Trade size |
| `price` | Fill price |
| `fee` | Fee paid |
| `fee_currency` | Fee currency (e.g. `USD`) |
| `market_type` | `spot` or `futures` |
| `timestamp` | Trade execution time (RFC3339) |
| `cost_basis` | Cost basis for this trade |
| `realized_pnl` | Realized P&L for this trade (sell side) |
| `strategy` | Strategy that generated the signal (if set) |
| `entry_reason` | Why the position was entered |
| `exit_reason` | Why the position was exited |
| `confidence` | Signal confidence at entry (0–1) |
| `stop_loss` | Stop-loss at time of trade |
| `take_profit` | Take-profit at time of trade |

**Pattern — get the last trade for a symbol:**

```bash
trader trades list live --symbol BTC-USD --limit 1 --json | jq '.[0]'
```

---

## Orders

```bash
trader orders live                       # 50 most recent orders
trader orders live --status open         # open orders only
trader orders live --status filled
trader orders live --symbol BTC-USD
trader orders live --limit 0 --json      # all orders as JSON
```

**Order fields:** `order_id`, `symbol`, `side`, `order_type` (`market`/`limit`), `requested_qty`, `filled_qty`, `avg_fill_price`, `status` (`open`/`filled`/`partially_filled`/`cancelled`), `market_type`, `created_at`

---

## Import Historic Trades

Use `trader import` to bulk-load past trades from a JSON file. The service validates all trades up front, inserts them idempotently, and rebuilds positions from the full trade history.

```bash
trader import trades.json          # import and print summary
trader import trades.json --json   # full response JSON
```

**File format** — a JSON object with a `"trades"` array:

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
      "timestamp": "2025-01-01T10:00:00Z",
      "strategy": "macd_momentum",
      "entry_reason": "MACD crossover with RSI < 60",
      "confidence": 0.78,
      "stop_loss": 93000,
      "take_profit": 99000
    }
  ]
}
```

**Output:**
```
Total: 1  Inserted: 1  Duplicates: 0  Errors: 0
```

Exits non-zero if any errors occurred. Re-running the same file is safe (duplicates are skipped).

**Required fields:** `tenant_id`, `trade_id`, `account_id`, `symbol`, `side`, `quantity`, `price`, `fee`, `fee_currency`, `market_type`, `timestamp`

**Optional fields:** `leverage`, `margin`, `liquidation_price`, `funding_fee`, `strategy`, `entry_reason`, `exit_reason`, `confidence`, `stop_loss`, `take_profit`

---

## Config

```bash
trader config show                              # all resolved values and sources
trader config set trader_url https://...        # override service URL
trader config get trader_url
```

| Key | Default | Env override |
|-----|---------|-------------|
| `trader_url` | `https://signalngn-trader-potbdcvufa-ew.a.run.app` | `TRADER_URL` |
| `api_key` | _(from `~/.config/sn/config.yaml`)_ | `TRADER_API_KEY` |
| `tenant_id` | _(resolved via `/auth/resolve` on first use)_ | `TRADER_TENANT_ID` |

---

## Global Flags

```bash
trader --trader-url http://localhost:8080 accounts list   # one-off URL override
trader --json <any-command>                               # JSON output
```

---

## Trading Bot Patterns

### Check available balance before sizing a position

```bash
# Get current USD balance
BALANCE=$(trader accounts balance get live --json | jq '.amount')
echo "Available balance: $BALANCE"

# Set initial balance (e.g. after funding the account)
trader accounts balance set live 50000
```

### Check exposure before entering a trade

```bash
# Get open position size for BTC-USD
SIZE=$(trader positions live --json | jq '[.[] | select(.symbol == "BTC-USD" and .status == "open")] | map(.quantity) | add // 0')
echo "Current BTC exposure: $SIZE"
```

### Record a trade immediately after execution

```bash
trader trades add live \
  --trade-id "$EXCHANGE_ORDER_ID" \
  --symbol BTC-USD --side buy --quantity 0.1 --price 95000 \
  --fee 9.50 --strategy macd_momentum --confidence 0.78 \
  --stop-loss 93000 --take-profit 99000
```

### Get realized P&L for the day

```bash
TODAY=$(date -u +%Y-%m-%dT00:00:00Z)
trader trades list live --start "$TODAY" --json | \
  jq '[.[] | select(.side == "sell")] | map(.realized_pnl) | add // 0'
```

### Verify a trade was recorded after execution

```bash
# After placing a trade with trade_id "cb-20250201-042"
trader trades list live --json | jq '.[] | select(.trade_id == "cb-20250201-042")'
```

### Import a day's trades from a file

```bash
trader import /tmp/trades-2025-02-01.json
# Total: 12  Inserted: 12  Duplicates: 0  Errors: 0
```

### Point at a local ledger instance for testing

```bash
TRADER_URL=http://localhost:8080 trader accounts list
# or permanently:
trader config set trader_url http://localhost:8080
```
