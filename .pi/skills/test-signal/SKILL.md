---
name: test-signal
description: Push a test signal to the production trader engine via NATS to open or close a position. Use when you want to verify the signal pipeline end-to-end on the paper-transformer account.
allowed-tools: Bash
---

# Test Signal

Publish a test trading signal to the production trader engine via Synadia NGS NATS.

## Prerequisites

- Production Cloud SQL proxy must be running on port 5434 (`task proxy:prod`)
- NATS credentials are fetched from GCP Secret Manager at runtime

## Key Facts

- **Exchange:** `binance` (not `coinbase` — the ml_transformer_1h strategy runs on binance data)
- **Account:** `paper-transformer`
- **Subject format:** `signals.{exchange}.{product}.{granularity}.{strategy}`
- **NATS server:** `tls://connect.ngs.global`
- **Confidence threshold:** must be ≥ 0.5 for BUY/SHORT actions
- **Timestamp:** must be within 10 minutes of now

## Step 1 — Fetch NATS credentials

```bash
gcloud secrets versions access latest \
  --secret=signalngn-prod-nats-creds \
  --project=signalngn-prod \
  --account=anssip@gmail.com \
  > /tmp/prod-nats.creds
```

## Step 2 — Check for open positions

Before sending a BUY, confirm there's no open position (engine ignores duplicate entries):

```bash
psql "$(gcloud secrets versions access latest --secret=signalngn-prod-db-password --project=signalngn-prod --account=anssip@gmail.com | xargs -I{} echo 'postgres://spot:{}@localhost:5434/spot_canvas?sslmode=disable')" \
  -c "SELECT id, symbol, side, status FROM ledger_positions WHERE account_id = 'paper-transformer' AND status = 'open';"
```

Or more concisely, reuse the password already retrieved:

```bash
DB="postgres://spot:$(gcloud secrets versions access latest --secret=signalngn-prod-db-password --project=signalngn-prod --account=anssip@gmail.com)@localhost:5434/spot_canvas?sslmode=disable"
psql "$DB" -c "SELECT id, symbol, side, status FROM ledger_positions WHERE account_id = 'paper-transformer' AND status = 'open';"
```

## Step 3 — Publish signal

### Open a long position (BUY)

```bash
NOW=$(date +%s)
nats pub \
  --creds /tmp/prod-nats.creds \
  --server tls://connect.ngs.global \
  "signals.binance.ADA-USD.ONE_HOUR.ml_transformer_1h" \
  "{
    \"strategy\": \"ml_transformer_1h\",
    \"product\": \"ADA-USD\",
    \"exchange\": \"binance\",
    \"account_id\": \"paper-transformer\",
    \"action\": \"BUY\",
    \"market\": \"futures\",
    \"leverage\": 2,
    \"price\": 0.264,
    \"confidence\": 0.75,
    \"reason\": \"test signal - open position\",
    \"stop_loss\": 0.245,
    \"take_profit\": 0.302,
    \"risk_reasoning\": \"test\",
    \"position_pct\": 0.1,
    \"is_exit\": false,
    \"indicators\": {\"rsi\": 42.0, \"macd_hist\": 0.002, \"sma50\": 0.260, \"sma200\": 0.255},
    \"timestamp\": $NOW
  }"
```

### Close the position (SELL)

```bash
NOW=$(date +%s)
nats pub \
  --creds /tmp/prod-nats.creds \
  --server tls://connect.ngs.global \
  "signals.binance.ADA-USD.ONE_HOUR.ml_transformer_1h" \
  "{
    \"strategy\": \"ml_transformer_1h\",
    \"product\": \"ADA-USD\",
    \"exchange\": \"binance\",
    \"account_id\": \"paper-transformer\",
    \"action\": \"SELL\",
    \"market\": \"futures\",
    \"leverage\": 2,
    \"price\": 0.264,
    \"confidence\": 0.75,
    \"reason\": \"test signal - close position\",
    \"stop_loss\": 0,
    \"take_profit\": 0,
    \"risk_reasoning\": \"test\",
    \"position_pct\": 1.0,
    \"is_exit\": true,
    \"indicators\": {\"rsi\": 60.0, \"macd_hist\": -0.001, \"sma50\": 0.260, \"sma200\": 0.255},
    \"timestamp\": $NOW
  }"
```

## Step 4 — Verify

Wait ~5 seconds, then check the DB:

```bash
sleep 5 && psql "$DB" -c "
SELECT trade_id, side, quantity, price, ingested_at
FROM ledger_trades
WHERE account_id = 'paper-transformer'
AND ingested_at > NOW() - INTERVAL '2 minutes'
ORDER BY ingested_at DESC;

SELECT id, symbol, side, status FROM ledger_positions
WHERE account_id = 'paper-transformer' AND status = 'open';"
```

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| No trade recorded, no log | Wrong exchange or not in allowlist | Use `exchange: binance`; check trading configs |
| `signal too old, dropping` in logs | Stale timestamp | Always use `$(date +%s)` for the timestamp |
| Signal ignored silently | Open position already exists | Send SELL first to close, then BUY to reopen |
| `signal account_id not in managed accounts` | account not registered in engine | Check engine `ACCOUNTS` env var in Cloud Run |

## Checking trader logs

```bash
gcloud logging read 'resource.type="cloud_run_revision" AND resource.labels.service_name="signalngn-trader"' \
  --project=signalngn-prod \
  --account=anssip@gmail.com \
  --limit=20 \
  --format=json \
  --freshness=10m | python3 -c "
import sys, json
for e in json.load(sys.stdin):
    payload = e.get('jsonPayload') or {'msg': e.get('textPayload','')}
    print(e['timestamp'][:19], json.dumps(payload))
"
```
