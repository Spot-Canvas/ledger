## Why

Backtesting revealed that the 1h transformer model on futures-short caused liquidation events on at least two products (XLM drawdown 1330%, SUI drawdown 126%) because the engine has no hard price-level protection — it relies entirely on conviction-drop exit signals, which do not fire fast enough when a position moves adversely by 50%+ at 2× leverage. Adding a layered exit system with a circuit-breaker hard stop, trailing stop, and time-based kill switch closes these gaps without impairing the ML models' ability to run winners.

## What Changes

- **Engine hard stop (new):** A circuit-breaker stop is computed by the engine at entry time, independent of the signal's SL, using `max_adverse_pct = 30% / leverage` (15% at 2×, 10% at 3×, 6% at 5×, 7% flat for spot). This stop is always active and cannot be disabled by strategy configuration.
- **Trailing stop (new):** For ML strategies, once a position profits by 1× the signal SL distance, the engine moves the stop to breakeven. Once profitable by 2×, the stop trails 1× SL distance behind the running best price. Disabled for rule-based strategies.
- **Time-based exit (new):** The engine enforces a per-strategy-type max candle hold: 30 candles for ml_xgboost 5m, 24 candles for ml_transformer 5m, 12 candles for ml_transformer_1h, 48 candles for rule-based 5m, 24 candles for rule-based 1h. When the hold limit is reached, the position is closed at market.
- **Exit check frequency (new):** The engine goroutines responsible for hard stop and trailing stop SHALL evaluate exit conditions on every price tick received, not only on candle close. Checking only at candle close means a position can overshoot the stop by the full intra-candle range before being closed, compounding losses. Time-based exit checks at candle granularity (one check per completed candle) is sufficient since the unit is candle count.
- **Exit priority ordering (new):** When multiple exit conditions fire simultaneously the engine resolves them in strict priority: conviction-drop signal → engine hard stop → signal SL → trailing stop → time-based exit → signal TP.
- **ML vs rule-based TP treatment (clarified):** ML strategies use no fixed TP; rule-based strategies keep the signal TP. The engine derives strategy type from the strategy name prefix (`ml_` = ML, otherwise rule-based).

## Capabilities

### New Capabilities
- `engine-hard-stop`: Circuit-breaker stop computed at entry from leverage (`30% / leverage`), checked against candle high/low every candle, always active regardless of signal SL presence.
- `trailing-stop`: ML-strategy trailing stop that activates at 1× signal SL profit (breakeven move), then trails at 1× SL distance once profit exceeds 2× signal SL. Disabled for rule-based strategies.
- `time-based-exit`: Max candle hold limits keyed by strategy type and granularity; closes stale positions to prevent funding drain and capital lock.
- `exit-orchestration`: Priority resolution when multiple exit conditions are simultaneously satisfied; defines the canonical exit type enum used in exit_reason fields.

### Modified Capabilities
- `trade-ingestion`: When the engine closes a position due to a risk management rule, the trade event SHALL carry an `exit_reason` string that identifies both the layer and the specific trigger. Format: `"Layer <N>: <label> — <detail>"`. Examples:
  - `"Layer 2: hard stop — 15.3% adverse move at 2× leverage"`
  - `"Layer 4: trailing stop — breakeven triggered at +1× SL distance"`
  - `"Layer 4: trailing stop — trailing at +2× SL distance, best price $0.4821"`
  - `"Layer 5: time exit — 12-candle hold limit reached"`
  - `"Layer 1: signal SL — price hit stop at $0.4210"`
  - `"Layer 6: signal TP — price hit take-profit at $0.5400"`
  Strategy-emitted conviction-drop exits (Layer 3) do not set an engine exit_reason; the strategy may populate `exit_reason` itself. If a conviction-drop fires with no strategy-supplied reason, the engine SHALL write `"Layer 3: conviction drop"`.

## Impact

- **New Go library (`github.com/Signal-ngn/risk`):** All exit-decision logic extracted into a standalone zero-dependency module. Both the trading engine and the backtester import it, guaranteeing identical behaviour in live and simulation.
- **Trading engine (server):** Core position management loop delegates to the library; gains hard stop, SL-distance trailing stop, per-strategy time limits, and exit priority. No signal format changes required.
- **Trade event publisher:** Engine-generated exits will populate `exit_reason` with a structured string from the library.
- **Backtester (`spot-canvas-app`):** Wired to the same library so drawdown and liquidation events are correctly simulated. This fixes the core gap that motivated this change — the 1h futures backtest results will now reflect the hard stop protection.
- **No API or CLI changes required** — exit reasons are observable through existing trade event fields.
