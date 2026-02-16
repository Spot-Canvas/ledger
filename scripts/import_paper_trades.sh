#!/bin/bash
# Import paper trading data from the dashboard into the ledger.
# Timestamps on the dashboard are EET (UTC+2), converted to UTC below.
#
# Data source: https://spot-arb-dashboard.s3.eu-north-1.amazonaws.com/paper_trading.html
# Last updated: 2026-02-16 07:41:59 EET
#
# Trade mapping:
#   LONG open  → side: "buy"
#   LONG close → side: "sell"
#   SHORT open → side: "sell"
#   SHORT close→ side: "buy"
#
# Market types:
#   -USD pairs (Coinbase)  → "spot"
#   -USDT pairs (CoinDesk) → "futures"
#
# Fees:
#   Futures: 0.02% entry, 0.05% exit (per dashboard)
#   Spot: ~0.1% taker fee estimate

set -euo pipefail

API="http://localhost:8080/api/v1/import"

# Helper: compute quantity = size / price (8 decimal places)
qty() {
  echo "scale=8; $1 / $2" | bc
}

# Helper: compute fee = size * rate (4 decimal places)
fee() {
  echo "scale=4; $1 * $2" | bc
}

# Build the JSON payload with all trades
# Each closed position = 2 trades (entry + exit)
# Each open position = 1 trade (entry only)

cat <<'EOF' | curl -s -X POST "$API" -H "Content-Type: application/json" -d @-
{
  "trades": [
    
    {"trade_id":"paper-001-buy","account_id":"paper","symbol":"BTC-USD","side":"buy","quantity":0.01521128,"price":65749.80,"fee":1.00,"fee_currency":"USD","market_type":"spot","timestamp":"2026-02-12T19:24:00Z"},
    {"trade_id":"paper-001-sell","account_id":"paper","symbol":"BTC-USD","side":"sell","quantity":0.01521128,"price":69532.90,"fee":1.06,"fee_currency":"USD","market_type":"spot","timestamp":"2026-02-14T08:15:00Z"},

    {"trade_id":"paper-002-buy","account_id":"paper","symbol":"DOGE-USD","side":"buy","quantity":3264.41784,"price":0.0919,"fee":0.30,"fee_currency":"USD","market_type":"spot","timestamp":"2026-02-12T19:26:00Z"},
    {"trade_id":"paper-002-sell","account_id":"paper","symbol":"DOGE-USD","side":"sell","quantity":3264.41784,"price":0.0970,"fee":0.30,"fee_currency":"USD","market_type":"spot","timestamp":"2026-02-13T21:45:00Z"},

    {"trade_id":"paper-003-buy","account_id":"paper","symbol":"SOL-USD","side":"buy","quantity":6.40533436,"price":78.06,"fee":0.50,"fee_currency":"USD","market_type":"spot","timestamp":"2026-02-12T19:26:00Z"},
    {"trade_id":"paper-003-sell","account_id":"paper","symbol":"SOL-USD","side":"sell","quantity":6.40533436,"price":84.99,"fee":0.50,"fee_currency":"USD","market_type":"spot","timestamp":"2026-02-14T03:15:00Z"},

    {"trade_id":"paper-004-buy","account_id":"paper","symbol":"QRL-USDT","side":"buy","quantity":263.15789474,"price":1.9000,"fee":0.10,"fee_currency":"USDT","market_type":"futures","leverage":1,"timestamp":"2026-02-12T20:05:00Z"},
    {"trade_id":"paper-004-sell","account_id":"paper","symbol":"QRL-USDT","side":"sell","quantity":263.15789474,"price":1.7900,"fee":0.25,"fee_currency":"USDT","market_type":"futures","leverage":1,"timestamp":"2026-02-13T20:20:00Z"},

    {"trade_id":"paper-005-buy","account_id":"paper","symbol":"INJ-USD","side":"buy","quantity":166.27868,"price":3.0070,"fee":0.50,"fee_currency":"USD","market_type":"spot","timestamp":"2026-02-13T07:05:00Z"},
    {"trade_id":"paper-005-sell","account_id":"paper","symbol":"INJ-USD","side":"sell","quantity":166.27868,"price":3.1380,"fee":0.50,"fee_currency":"USD","market_type":"spot","timestamp":"2026-02-13T22:30:00Z"},

    {"trade_id":"paper-006-buy","account_id":"paper","symbol":"TAO-USDT","side":"buy","quantity":7.98487,"price":187.8600,"fee":0.30,"fee_currency":"USDT","market_type":"futures","leverage":1,"timestamp":"2026-02-15T13:30:00Z"},
    {"trade_id":"paper-006-sell","account_id":"paper","symbol":"TAO-USDT","side":"sell","quantity":7.98487,"price":182.3000,"fee":0.75,"fee_currency":"USDT","market_type":"futures","leverage":1,"timestamp":"2026-02-15T17:48:00Z"},

    {"trade_id":"paper-007-buy","account_id":"paper","symbol":"TRX-USDT","side":"buy","quantity":5353.31906,"price":0.2802,"fee":0.30,"fee_currency":"USDT","market_type":"futures","leverage":1,"timestamp":"2026-02-15T13:00:00Z"},
    {"trade_id":"paper-007-sell","account_id":"paper","symbol":"TRX-USDT","side":"sell","quantity":5353.31906,"price":0.2796,"fee":0.75,"fee_currency":"USDT","market_type":"futures","leverage":1,"timestamp":"2026-02-16T04:00:00Z"},

    {"trade_id":"paper-008-buy","account_id":"paper","symbol":"TAO-USDT","side":"buy","quantity":8.16301,"price":183.7600,"fee":0.30,"fee_currency":"USDT","market_type":"futures","leverage":1,"timestamp":"2026-02-15T19:30:00Z"},
    {"trade_id":"paper-008-sell","account_id":"paper","symbol":"TAO-USDT","side":"sell","quantity":8.16301,"price":185.9300,"fee":0.75,"fee_currency":"USDT","market_type":"futures","leverage":1,"timestamp":"2026-02-15T21:55:00Z"},

    {"trade_id":"paper-009-buy","account_id":"paper","symbol":"QRL-USDT","side":"buy","quantity":802.13904,"price":1.8700,"fee":0.30,"fee_currency":"USDT","market_type":"futures","leverage":1,"timestamp":"2026-02-15T22:35:00Z"},
    {"trade_id":"paper-009-sell","account_id":"paper","symbol":"QRL-USDT","side":"sell","quantity":802.13904,"price":1.8800,"fee":0.75,"fee_currency":"USDT","market_type":"futures","leverage":1,"timestamp":"2026-02-15T22:56:00Z"},

    {"trade_id":"paper-010-sell","account_id":"paper","symbol":"HYPE-USDT","side":"sell","quantity":49.85050,"price":30.0900,"fee":0.30,"fee_currency":"USDT","market_type":"futures","leverage":2,"timestamp":"2026-02-16T03:00:00Z"},
    {"trade_id":"paper-010-buy","account_id":"paper","symbol":"HYPE-USDT","side":"buy","quantity":49.85050,"price":30.1300,"fee":0.75,"fee_currency":"USDT","market_type":"futures","leverage":2,"timestamp":"2026-02-16T03:03:00Z"},

    {"trade_id":"paper-011-buy","account_id":"paper","symbol":"QRL-USDT","side":"buy","quantity":821.69269,"price":1.8255,"fee":0.30,"fee_currency":"USDT","market_type":"futures","leverage":1,"timestamp":"2026-02-16T02:36:00Z"},

    {"trade_id":"paper-012-buy","account_id":"paper","symbol":"TRX-USDT","side":"buy","quantity":5353.31906,"price":0.2802,"fee":0.30,"fee_currency":"USDT","market_type":"futures","leverage":1,"timestamp":"2026-02-16T05:00:00Z"}
  ]
}
EOF
