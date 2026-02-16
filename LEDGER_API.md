# Ledger API Integration

The Spot Canvas Ledger records trades, tracks positions, and computes P&L. It supports two integration paths:

- **NATS JetStream** — publish trade events in real time (preferred for live/paper trading)
- **REST API** — query portfolio state and bulk-import historic trades

## Base URL

```
https://spot-canvas-ledger-staging-970386657060.europe-west3.run.app
```

No authentication required. All responses are `Content-Type: application/json`.

---

## Accounts

The ledger supports multiple accounts simultaneously. Use the account ID to separate live and paper trading:

| Account ID | Type | Description |
|---|---|---|
| `"paper"` | paper | Paper/simulated trading |
| `"live"`, `"coinbase"`, etc. | live | Real trading (any ID other than `"paper"`) |

Accounts are auto-created on first trade. You can run both paper and live accounts at the same time — they are fully independent with separate positions and P&L.

**Symbol format**: use the exchange's native pair format, e.g. `"BTC-USD"`, `"ETH-USDT"`.

---

## Recording Trades via NATS (Real-Time)

For real-time trade recording, publish to NATS JetStream after each trade execution. This is fire-and-forget with durable at-least-once delivery — the ledger will process it even if it restarts.

### NATS Connection

- **URL**: same NATS cluster as the rest of spot-canvas (Synadia NGS)
- **Stream**: `LEDGER_TRADES`
- **Credentials**: use the same NATS credentials file as the rest of the system

### Subject Format

```
ledger.trades.<accountId>.<marketType>
```

Examples:
- `ledger.trades.paper.spot` — paper spot trade
- `ledger.trades.paper.futures` — paper futures trade
- `ledger.trades.live.spot` — live spot trade
- `ledger.trades.coinbase.spot` — live trade on Coinbase

### Message Format (JSON)

```json
{
  "trade_id": "unique-trade-id",
  "account_id": "paper",
  "symbol": "BTC-USD",
  "side": "buy",
  "quantity": 0.5,
  "price": 50000.00,
  "fee": 25.00,
  "fee_currency": "USD",
  "market_type": "spot",
  "timestamp": "2025-01-15T10:00:00Z"
}
```

For futures trades, include additional fields:

```json
{
  "trade_id": "unique-trade-id",
  "account_id": "paper",
  "symbol": "BTC-USD",
  "side": "buy",
  "quantity": 0.5,
  "price": 50000.00,
  "fee": 25.00,
  "fee_currency": "USD",
  "market_type": "futures",
  "timestamp": "2025-01-15T10:00:00Z",
  "leverage": 5,
  "margin": 5000.00,
  "liquidation_price": 42000.00,
  "funding_fee": 1.25
}
```

### Required Fields

| Field | Type | Description |
|---|---|---|
| `trade_id` | string | Unique ID (e.g. exchange order ID or UUID). Duplicates are safely skipped. |
| `account_id` | string | `"paper"` for paper trading, anything else for live |
| `symbol` | string | Trading pair, e.g. `"BTC-USD"` |
| `side` | string | `"buy"` or `"sell"` |
| `quantity` | float | Must be > 0 |
| `price` | float | Must be > 0 |
| `fee` | float | Trading fee (0 if none) |
| `fee_currency` | string | Fee currency, e.g. `"USD"` |
| `market_type` | string | `"spot"` or `"futures"` |
| `timestamp` | string | RFC3339 timestamp of the trade execution |

### Optional Fields (Futures Only)

| Field | Type | Description |
|---|---|---|
| `leverage` | int | Position leverage multiplier |
| `margin` | float | Margin amount |
| `liquidation_price` | float | Liquidation price |
| `funding_fee` | float | Funding fee (for perpetuals) |

### Delivery Guarantees

- JetStream durable consumer with at-least-once delivery
- Messages are retried up to 5 times on processing failure (30s ack timeout)
- Duplicate `trade_id` values are idempotently skipped — safe to re-publish
- Malformed or invalid messages are terminated (no redelivery)

---

## REST API — Querying Portfolio State

Use the REST API to read portfolio data. All query endpoints are GET.

### Health Check

```
GET /health
→ 200 {"status":"ok"}
→ 503 {"status":"error","error":"..."}
```

### List Accounts

```
GET /api/v1/accounts
→ 200 [{"id":"paper","name":"paper","type":"paper","created_at":"..."}]
```

### Portfolio Summary

```
GET /api/v1/accounts/{accountId}/portfolio
```

Response:

```json
{
  "positions": [
    {
      "id": "uuid",
      "account_id": "paper",
      "symbol": "BTC-USD",
      "market_type": "spot",
      "side": "long",
      "quantity": 0.5,
      "avg_entry_price": 50000,
      "cost_basis": 25025,
      "realized_pnl": 0,
      "status": "open",
      "opened_at": "2025-01-15T10:00:00Z"
    }
  ],
  "total_realized_pnl": 2450.00
}
```

Returns 404 if account not found.

### List Positions

```
GET /api/v1/accounts/{accountId}/positions?status=open
```

Query params: `status` = `open` (default) | `closed` | `all`

Response: array of Position objects (same shape as in portfolio summary).

### List Trades

```
GET /api/v1/accounts/{accountId}/trades?symbol=BTC-USD&side=buy&market_type=spot&start=2025-01-01T00:00:00Z&end=2025-12-31T23:59:59Z&limit=50&cursor=xxx
```

All query params optional. `limit` default 50, max 200. Cursor-based pagination.

Response:

```json
{
  "trades": [
    {
      "trade_id": "unique-id",
      "account_id": "paper",
      "symbol": "BTC-USD",
      "side": "buy",
      "quantity": 0.5,
      "price": 50000,
      "fee": 25,
      "fee_currency": "USD",
      "market_type": "spot",
      "timestamp": "2025-01-15T10:00:00Z",
      "ingested_at": "2025-01-15T10:00:01Z",
      "cost_basis": 25025,
      "realized_pnl": 0
    }
  ],
  "next_cursor": "base64-cursor-string"
}
```

`next_cursor` is omitted when there are no more results. Pass it as `cursor` param to get the next page.

### List Orders

```
GET /api/v1/accounts/{accountId}/orders?status=open&symbol=BTC-USD&limit=50&cursor=xxx
```

Response:

```json
{
  "orders": [
    {
      "order_id": "...",
      "account_id": "paper",
      "symbol": "BTC-USD",
      "side": "buy",
      "order_type": "market",
      "requested_qty": 0.5,
      "filled_qty": 0.5,
      "avg_fill_price": 50000,
      "status": "filled",
      "market_type": "spot",
      "created_at": "...",
      "updated_at": "..."
    }
  ],
  "next_cursor": "..."
}
```

---

## REST API — Bulk Import Historic Trades

Use the import endpoint to backfill historic trades (e.g. from exchange export or CSV).

```
POST /api/v1/import
Content-Type: application/json
```

Request body — max 1000 trades per request:

```json
{
  "trades": [
    {
      "trade_id": "t-001",
      "account_id": "paper",
      "symbol": "BTC-USD",
      "side": "buy",
      "quantity": 0.5,
      "price": 50000.00,
      "fee": 25.00,
      "fee_currency": "USD",
      "market_type": "spot",
      "timestamp": "2025-01-15T10:00:00Z"
    }
  ]
}
```

Response:

```json
{
  "total": 1,
  "inserted": 1,
  "duplicates": 0,
  "errors": 0,
  "results": [
    {"trade_id": "t-001", "status": "inserted"}
  ]
}
```

Status per trade: `"inserted"`, `"duplicate"`, or `"error"` (with `"error"` field). The entire batch is rejected with 400 if any trade fails validation. If all inserts fail, returns 422. Duplicate `trade_id` values are safely skipped.

---

## Error Shape

All errors:

```json
{"error": "human-readable message"}
```

HTTP status codes: 400 (bad request/validation), 404 (not found), 422 (all inserts failed), 500 (server error), 503 (unhealthy).

---

## Integration Summary

| Action | Method | When |
|---|---|---|
| Record a trade in real time | NATS publish to `ledger.trades.<accountId>.<marketType>` | After each trade execution |
| Backfill historic trades | `POST /api/v1/import` | One-time import of old trades |
| Check current positions | `GET /api/v1/accounts/{accountId}/portfolio` | Before trading decisions |
| Check if position is open | `GET /api/v1/accounts/{accountId}/positions?status=open` | Before opening new positions |
| Review trade history | `GET /api/v1/accounts/{accountId}/trades` | Reporting, debugging |
| List all accounts | `GET /api/v1/accounts` | Discovery |

The ledger handles all position tracking automatically — buys open/increase long positions, sells reduce/close them. Cost basis and realized P&L are computed server-side.
