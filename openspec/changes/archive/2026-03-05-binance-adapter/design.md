## Context

The engine wires `BinanceFuturesExchange` for all live trades. Signal routing already distinguishes spot vs. futures via `mapSignalToSide` — spot signals (`strategies_spot`) resolve to `marketType=Spot`, `positionSide=LONG`, and `leverage=tc.LongLeverage`. For a spot trading config, `LongLeverage` is typically 0.

The existing `OpenPosition` implementation:
```go
if req.Leverage > 0 {
    b.client.setLeverage(ctx, symbol, req.Leverage)
}
```

When `req.Leverage == 0`, `setLeverage` is skipped entirely. The Binance account retains whatever leverage was last configured — silently applying it to the new position. A spot signal on an account that previously ran a 10× futures trade would open at 10× leverage.

The second bug: Binance MARKET orders populate `executedQty` with the actual filled quantity. The adapter currently reads `origQty`, which is the requested quantity before lot-size rounding. For most liquid pairs the difference is negligible, but it can cause a fractional mismatch between the recorded trade and the real position on Binance.

The chosen approach (Futures API at 1× for spot-like trading) is intentional: funds stay in a single Futures USDT wallet, covering both spot-like longs and leveraged long/short strategies without any asset custody or separate account management.

## Goals / Non-Goals

**Goals:**
- Guarantee leverage=1 when no explicit leverage is requested, regardless of prior account state.
- Record the correct fill quantity from `executedQty`.
- Document fee behaviour accurately.

**Non-Goals:**
- A separate Binance Spot REST adapter (`/api/v3/`) — not needed.
- `MultiExchange` routing — not needed; single adapter handles all market types.
- Binance Futures fee lookup via `/fapi/v1/userTrades` — deferred.
- Any changes to signal routing, position logic, or config.

## Decisions

### 1. Always call `setLeverage`, default to 1

**Decision:** Replace the conditional leverage guard with an unconditional call, defaulting to 1 when `req.Leverage` is 0:

```go
leverage := req.Leverage
if leverage <= 0 {
    leverage = 1
}
b.client.setLeverage(ctx, symbol, leverage)
```

This means:
- Spot signals (`LongLeverage=0`) → `setLeverage(1)` — safe, explicit.
- Futures signals (`LongLeverage=5`) → `setLeverage(5)` — unchanged behaviour.
- Any account pre-configured at a different leverage is corrected before the order, not silently inherited.

**Considered:** Adding `MarketType` to `OpenPositionRequest` and branching on it. Rejected — unnecessary complexity. The leverage value already fully encodes the intent; defaulting 0 → 1 is the right invariant regardless of market type.

---

### 2. `executedQty` for fill quantity

**Decision:** Replace `origQty` with `executedQty` in the order response parser. MARKET orders on Binance always fill completely (for liquid pairs), so `executedQty` equals the actual fill and `origQty` may differ by lot-size rounding.

```go
// Before
fmt.Sscanf(body.OrigQty, "%f", &result.Quantity)

// After
fmt.Sscanf(body.ExecutedQty, "%f", &result.Quantity)
```

---

### 3. Fee handling — no change for now

The Binance Futures order endpoint (`POST /fapi/v1/order`) does not return commission in its response body. The actual commission is available via `GET /fapi/v1/userTrades?orderId=...` as a separate request after the fill. Making a second API call on every trade adds latency and complexity for a low-priority field (fees are already recorded as 0 in paper mode and negligible at typical position sizes).

**Decision:** Leave `Fee: 0` for now. Document this limitation. A follow-up change can add an async post-fill lookup.

## Risks / Trade-offs

- **Funding rates**: Spot-like positions held overnight accumulate futures funding rate costs (typically ±0.01%/8h for BTC). This is acceptable per product decision and is not reflected in recorded fees. Operators should be aware.

- **`setLeverage` on every open**: Binance rate-limits leverage changes. Calling `setLeverage` even when the account is already at the correct leverage adds one extra signed request per open. In practice this is negligible; Binance returns quickly when the value is unchanged. The existing retry-on-429 handles any transient throttling.

- **Lot-size precision with `executedQty`**: If a MARKET order partially fills (extremely rare for BTC/ETH perpetuals), `executedQty` reflects only the filled portion. The position recorded in the ledger will match the real Binance position, which is the correct behaviour.

## Migration Plan

1. Deploy updated `exchange.go` — single file change, no config or DB migration.
2. Existing open positions are unaffected (close logic is unchanged).
3. Rollback: revert `exchange.go`. Paper mode has no regression path since `NoopExchange` is untouched.

## Open Questions

- None — scope is fully resolved.
