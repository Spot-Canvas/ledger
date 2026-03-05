## Why

The trading engine has a single `BinanceFuturesExchange` for live mode, but signal configs include both futures strategies (`strategies_long`/`strategies_short`) and spot-like strategies (`strategies_spot`). Spot signals are already routed to the futures exchange — but two bugs make them dangerous or incorrect:

1. **Leverage not set for spot signals**: `strategies_spot` configs typically set `LongLeverage = 0`, so `req.Leverage` arrives as 0 and the `setLeverage` call is skipped entirely. The account retains whatever leverage was last set (e.g. 10×), silently applying leverage to what was intended as a 1× trade.
2. **Wrong fill quantity**: MARKET orders populate `executedQty` (actual fill), but the adapter reads `origQty` (requested qty). These differ when Binance rounds the quantity to lot-size precision.

Additionally, using the Futures API for spot-like strategies is intentional — funds remain in a single Futures wallet, funding rates are acceptable, and no real asset custody is needed. There is no plan to build a separate Binance Spot adapter.

## What Changes

- **Spot signals via futures at 1×**: When `req.Leverage` is 0 (spot-config signals), `BinanceFuturesExchange` SHALL explicitly set leverage to 1 before opening. This is a safety default — futures signals with an explicit leverage value are unaffected.
- **Fix `executedQty`**: Replace `origQty` with `executedQty` in the order response parser so recorded fill quantity matches the actual exchange fill.
- **Fix fee parsing**: Parse commission from the futures order acknowledgement where available; document that exact commission requires a separate `/fapi/v1/userTrades` lookup (deferred).

## Capabilities

### New Capabilities
- `binance-futures-adapter`: Specifies the behaviour of `BinanceFuturesExchange` for both futures (leveraged long/short) and spot-like (1× long-only) trading — covering leverage defaulting, fill quantity accuracy, and fee recording.

### Modified Capabilities
- None.

## Impact

- **`internal/engine/exchange.go`**: Change leverage guard from `if req.Leverage > 0` to always call `setLeverage`, defaulting to 1; swap `origQty` → `executedQty` in order response parser.
- **No new files, no config changes, no API/DB/NATS changes.**
