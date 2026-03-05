## Why

Backtesting revealed that the 1h transformer model on futures-short caused liquidation events on at least two products (XLM drawdown 1330%, SUI drawdown 126%) because the engine has no hard price-level protection — it relies entirely on conviction-drop exit signals, which do not fire fast enough when a position moves adversely by 50%+ at 2× leverage. Adding a layered exit system with a circuit-breaker hard stop, trailing stop, and time-based kill switch closes these gaps without impairing the ML models' ability to run winners.

## What Changes

- **Engine hard stop (new):** A circuit-breaker stop is computed by the engine at entry time, independent of the signal's SL, using `max_adverse_pct = 30% / leverage` (15% at 2×, 10% at 3×, 6% at 5×, 7% flat for spot). This stop is always active and cannot be disabled by strategy configuration.
- **Trailing stop (new):** For ML strategies, once a position profits by 1× the signal SL distance, the engine moves the stop to breakeven. Once profitable by 2×, the stop trails 1× SL distance behind the running best price. Disabled for rule-based strategies.
- **Time-based exit (new):** The engine enforces a per-strategy-type max candle hold: 30 candles for ml_xgboost 5m, 24 candles for ml_transformer 5m, 12 candles for ml_transformer_1h, 48 candles for rule-based 5m, 24 candles for rule-based 1h. When the hold limit is reached, the position is closed at market.
- **Exit priority ordering (new):** When multiple exit conditions fire simultaneously the engine resolves them in strict priority: conviction-drop signal → engine hard stop → signal SL → trailing stop → time-based exit → signal TP.
- **ML vs rule-based TP treatment (clarified):** ML strategies use no fixed TP; rule-based strategies keep the signal TP. The engine derives strategy type from the strategy name prefix (`ml_` = ML, otherwise rule-based).

## Capabilities

### New Capabilities
- `engine-hard-stop`: Circuit-breaker stop computed at entry from leverage (`30% / leverage`), checked against candle high/low every candle, always active regardless of signal SL presence.
- `trailing-stop`: ML-strategy trailing stop that activates at 1× signal SL profit (breakeven move), then trails at 1× SL distance once profit exceeds 2× signal SL. Disabled for rule-based strategies.
- `time-based-exit`: Max candle hold limits keyed by strategy type and granularity; closes stale positions to prevent funding drain and capital lock.
- `exit-orchestration`: Priority resolution when multiple exit conditions are simultaneously satisfied; defines the canonical exit type enum used in exit_reason fields.

### Modified Capabilities
- `trade-ingestion`: The `exit_reason` field SHALL accept a structured enum of engine-generated exit types (`conviction-drop`, `hard-stop`, `signal-sl`, `trailing-stop`, `time-exit`, `signal-tp`) in addition to arbitrary strategy-supplied strings, so exit analysis can distinguish engine-managed exits from model exits.

## Impact

- **Trading engine (server):** Core position management loop gains hard stop check, trailing stop state machine, candle counter, and exit priority resolver. All changes are engine-side; no signal format changes required.
- **Trade event publisher:** Engine-generated exits will populate `exit_reason` with one of the structured enum values above.
- **Backtester:** Should be updated to simulate the same layered exit logic so backtest results reflect live behaviour; this is out of scope for this change but tracked as follow-up.
- **No API or CLI changes required** for the initial implementation — exit reasons are observable through existing trade event fields.
