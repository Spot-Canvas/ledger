## Context

The trading engine (`internal/engine/`) manages open positions through two goroutines: a signal loop that processes NGS signals in real time, and a risk loop that fires on a 5-minute ticker (`riskLoopInterval = 5 * time.Minute`). The risk loop calls `evaluatePosition` for each position in `posState`, checking max hold time, SL, TP, and a trailing stop based on fixed percentages (`trailingActivatePct = 3%`, `trailingTrailPct = 2%`).

The current implementation has four gaps that this change closes:

1. **No circuit-breaker hard stop.** The fallback SL (`defaultSLPct = 4%`) is applied only when the signal SL is absent or trivially close to entry. There is no leveraged-aware floor that prevents liquidation on a large adverse move.
2. **5-minute risk loop cadence.** A position can overshoot any stop by the full price distance accumulated in 5 minutes before the engine reacts. At 2× leverage on a volatile asset this is material.
3. **Trailing stop ignores signal SL distance.** Fixed percentage activation and trail width are decoupled from the position's actual SL, so the trailing stop can fire prematurely on normal noise or miss profits entirely.
4. **Single fixed hold limit for all strategies.** `maxHoldDuration = 48h` applies uniformly; it does not distinguish 5m ML strategies (which should exit within 2–3 hours) from 1h rule-based strategies (which can legitimately hold for a day).

## Goals / Non-Goals

**Goals:**
- Extract all exit-decision logic into a standalone, zero-dependency Go library (`github.com/Signal-ngn/risk`) usable by both the trading engine and the backtester.
- Add a leverage-scaled hard stop (Layer 2) that always fires, independent of signal SL.
- Reduce risk evaluation latency to tick level — fire on every incoming price update, not just on the 5-minute ticker.
- Replace fixed-percentage trailing stop with SL-distance-based trailing stop; disable trailing stop for rule-based strategies.
- Replace the single hold limit with per-strategy/granularity candle-count limits.
- Enforce strict exit priority within `evaluatePosition`.
- Write structured `exit_reason` strings (`"Layer N: label — detail"`) to every engine-generated trade close so ledger analysis can identify which risk layer fired.
- Wire the backtester (`spot-canvas-app`) to use the same library so backtest results reflect live behaviour.

**Non-Goals:**
- Changing signal formats or adding new API endpoints.
- Adjusting position sizing or leverage.

## Decisions

### Decision 0 — Extract risk logic into a dedicated Go module: `github.com/Signal-ngn/risk`

**Decision:** All exit-decision logic (hard stop, trailing stop, time-based exit, priority ordering, exit_reason formatting) is implemented in a new standalone Go module at `github.com/Signal-ngn/risk`. The module has **zero non-stdlib dependencies** (`math`, `strings`, `fmt`, `time` only). Both the trader engine and the backtester (`spot-canvas-app`) import this module.

**Public API:**

```go
package risk

// Position holds the mutable risk state for a single open position.
// Static fields are set at entry and never mutate. Mutable fields
// (PeakPrice, TrailingStop) are updated in-place by Evaluate.
type Position struct {
    // Static — set at entry
    EntryPrice  float64
    Side        string  // "long" | "short"
    StopLoss    float64 // 0 = absent
    TakeProfit  float64 // 0 = absent
    HardStop    float64 // pre-computed via ComputeHardStop
    Leverage    int
    Strategy    string
    Granularity string  // e.g. "FIVE_MINUTES", "ONE_HOUR"
    MarketType  string  // "spot" | "futures"
    OpenedAt    time.Time

    // Mutable — updated by Evaluate as trailing stop advances
    PeakPrice    float64
    TrailingStop float64
}

// ExitDecision describes why a position should be closed.
type ExitDecision struct {
    Layer      int
    Label      string
    Detail     string
    ExitReason string // "Layer N: label — detail"
}

// Evaluate checks all exit layers in priority order for the given price range.
// high/low are the candle's intra-bar extremes (use currentPrice for both in
// tick mode). close is the candle close (used for trailing stop advancement).
// now is the candle timestamp (used for time-based exit).
//
// Returns (decision, true) when an exit should fire. Also mutates pos.PeakPrice
// and pos.TrailingStop in-place when the trailing stop advances without firing.
// The caller is responsible for persisting the updated state.
func Evaluate(pos *Position, high, low, close float64, now time.Time) (ExitDecision, bool)

// ComputeHardStop returns the hard stop price for an entry.
// Formula: max_adverse_pct = max(30% / leverage, 7%) applied to entryPrice.
func ComputeHardStop(entryPrice float64, side string, leverage int, marketType string) float64

// MaxHoldDuration returns the max hold duration for a strategy/granularity pair.
func MaxHoldDuration(strategy, granularity string) time.Duration

// IsMLStrategy returns true if the strategy has the "ml_" prefix.
func IsMLStrategy(strategy string) bool
```

**How the trading engine uses it:**
- At entry: `ps.HardStop = risk.ComputeHardStop(...)`, stored in `PositionState` and DB.
- Per tick in `evaluatePosition`: convert `PositionState` → `risk.Position`, call `risk.Evaluate(riskPos, price, price, price, time.Now())`, act on the result, write back updated `PeakPrice`/`TrailingStop` if the trailing stop advanced.

**How the backtester uses it:**
- At entry: compute and store `HardStop` alongside `stopLoss`/`takeProfit`.
- Per candle in `checkSLTP` (renamed/extended to `checkExits`): call `risk.Evaluate(riskPos, candle.High, candle.Low, candle.Close, candle.Timestamp)`. This checks hard stop and signal SL against the intra-candle High/Low range, trailing stop against Close. Returns exit decision with layer and reason.

**Why a separate repository over `github.com/Signal-ngn/trader/pkg/risk`:**
The signal engine (`spot-canvas-app`) uses module path `spot-canvas-app` (local). Importing the trader module there would be direction-backwards (the signal engine is upstream of the trader) and would pull in pgx, NATS client, zerolog, cobra/viper, etc. into the signal engine's go.sum for no benefit. A zero-dep dedicated module avoids all of that and makes the library independently versioned and testable.

---

### Decision 1 — Risk evaluation cadence: trigger on price update, not on a timer

**Decision:** Call `evaluatePosition` immediately inside `handleSignal` every time a new price is cached for a symbol (`signal.Price > 0`), in addition to keeping the periodic ticker as a fallback. Reduce the ticker from 5 minutes to 30 seconds.

**Rationale:** The signal loop already runs per-message. Hooking evaluation there gives tick-level latency for active products at zero additional infrastructure cost. The 30-second ticker ensures that products with infrequent signals (1h strategies between candles) still get regular checks via the SN price API fallback. A pure timer-driven approach would require either a very short interval (CPU waste) or an external price-tick subscription.

**Alternative considered:** Subscribe to a dedicated exchange price stream (WebSocket) for each open position. Rejected — this would add exchange adapter complexity and another connection to manage. The signal stream is already a real-time feed; hooking into it is sufficient for the 5m strategy case and good enough (30s) for the 1h case.

**Sequence:** In `handleSignal`, after caching the price, call `go e.evaluateOpenPositionsForSymbol(ctx, product)`. This must be a goroutine to avoid blocking the NATS message handler. `evaluateOpenPositionsForSymbol` iterates posState and calls `evaluatePosition` for each state matching the product.

---

### Decision 2 — Hard stop: pre-compute at entry, store in PositionState

**Decision:** Compute the hard stop price at entry time using `max_adverse_pct = 30% / leverage` (minimum 7% for spot or when leverage ≤ 1). Store it as `HardStop float64` in `PositionState` and `store.EnginePositionState`. Check `currentPrice <= ps.HardStop` (long) or `currentPrice >= ps.HardStop` (short) in `evaluatePosition` as the first price-level check.

```
leverage=1 (spot): max_adverse_pct = 7%   (capped minimum)
leverage=2:        max_adverse_pct = 15%
leverage=3:        max_adverse_pct = 10%
leverage=5:        max_adverse_pct = 6%
```

**Rationale:** Pre-computing avoids re-deriving the threshold on every tick. The hard stop is immutable after entry; storing it in the DB means it survives restarts correctly without re-deriving from config (which could change). Storing in `PositionState` means the check is a single float comparison per tick.

**Alternative considered:** Compute from `ps.Leverage` on every tick. This works but leaves the 7% floor for spot implicit; storing makes the effective threshold auditable in the DB.

---

### Decision 3 — Trailing stop: SL-distance-based, ML strategies only

**Decision:** Replace the fixed-percentage trailing stop with:
- **Activation threshold:** `1 × slDistance` profit, where `slDistance = abs(ps.EntryPrice - ps.StopLoss)`. If `ps.StopLoss == 0`, fall back to `ps.EntryPrice × defaultSLPct`.
- **Breakeven move:** When profit ≥ 1× slDistance, move trailing stop to entry price.
- **Active trailing:** When profit ≥ 2× slDistance, trail at `1 × slDistance` behind `ps.PeakPrice`.
- **Disable for rule-based strategies:** `isMLStrategy(strategy string) bool` returns true when `strings.HasPrefix(strategy, "ml_")`. For non-ML strategies, skip trailing stop entirely.

**Rationale:** Tying trailing stop activation to the signal SL distance makes it proportional to the strategy's own risk/reward framing. A wide ATR-based SL (1h model) won't trigger the trailing stop on small favourable moves; a tight SL (5m model) will activate it sooner.

**Alternative considered:** Keep fixed percentages but make them configurable per strategy via `strategy_params`. Rejected — the SL-distance approach is self-calibrating and requires no manual tuning.

---

### Decision 4 — Per-strategy time limits: store granularity in PositionState

**Decision:** Add `Granularity string` to `PositionState` and `store.EnginePositionState`. Populate it at entry time from the matched `TradingConfig.Granularity`. Replace `maxHoldDuration` constant with `maxHoldDuration(strategy, granularity string) time.Duration` that returns:

| Strategy prefix | Granularity | Candles | Duration |
|---|---|---|---|
| `ml_xgboost` | `FIVE_MINUTES` | 30 | 2.5 h |
| `ml_transformer` | `FIVE_MINUTES` | 24 | 2 h |
| `ml_transformer` | `ONE_HOUR` | 12 | 12 h |
| rule-based | `FIVE_MINUTES` | 48 | 4 h |
| rule-based | `ONE_HOUR` | 24 | 24 h |
| (default) | any | 48 | 48 h |

Duration is compared against `time.Since(ps.OpenedAt)`, making this restart-safe without a separate candle counter.

**Rationale:** Storing granularity at entry time avoids a trading config lookup on every risk tick (which would require an HTTP call or a cached config that can drift). The position knows its own context.

---

### Decision 5 — Exit priority within evaluatePosition

**Decision:** Within `evaluatePosition`, check in this order and return after the first exit fires:

1. Hard stop (Layer 2) — checked before signal SL because signal SL can be 0 during ATR warmup.
2. Signal SL (Layer 1)
3. Trailing stop (Layer 4) — only if `isMLStrategy`
4. Time-based exit (Layer 5)
5. Signal TP (Layer 1) — only if `!isMLStrategy`

Conviction-drop exits (Layer 3) run on the signal goroutine via `handleCloseSignal`, which closes the position and deletes its `posState` entry before the risk loop can see it. This gives Layer 3 natural priority without any coordination needed.

---

### Decision 6 — Structured exit_reason strings

**Decision:** Define a helper:
```go
func engineExitReason(layer int, label, detail string) string {
    return fmt.Sprintf("Layer %d: %s — %s", layer, label, detail)
}
```

All `executeCloseTrade` calls from risk management paths use this helper:

| Trigger | exit_reason |
|---|---|
| Hard stop | `"Layer 2: hard stop — 15.3% adverse move at 2× leverage"` |
| Signal SL | `"Layer 1: signal SL — price $0.4210 hit stop $0.4215"` |
| Trailing stop (breakeven) | `"Layer 4: trailing stop — breakeven triggered, stop moved to entry $0.4500"` |
| Trailing stop (active trail) | `"Layer 4: trailing stop — trailing at $0.4380, best price $0.4480"` |
| Time exit | `"Layer 5: time exit — 12-candle hold limit reached (held 13h4m)"` |
| Signal TP | `"Layer 6: signal TP — price $0.5400 hit take-profit $0.5395"` |
| Conviction drop (fallback) | `"Layer 3: conviction drop"` (only when strategy supplies no reason) |

The existing close-by-signal path (`handleCloseSignal`) sets `exitReason = "signal"` for conviction drops. If the signal payload's `Reason` field is non-empty, use that as the detail. If empty, use `"Layer 3: conviction drop"`.

---

### Decision 7 — DB schema: two new columns on engine_position_state

**Decision:** Add `hard_stop NUMERIC` and `granularity TEXT` to `engine_position_state`. Both are nullable (existing rows default to NULL; hard_stop will be evaluated via fallback formula when NULL). Apply via a migration.

No change to `ledger_trades` schema — `exit_reason` is already a `TEXT` column; the structured format is a convention, not a schema constraint.

## Risks / Trade-offs

**[Risk] evaluatePosition called concurrently from signal goroutine and risk ticker** → Both paths read `posState` under `posStateMu.RLock()` then call `executeCloseTrade`, which deletes the state entry. If two goroutines simultaneously evaluate the same position and both decide to close it, `executeCloseTrade` will attempt a double-close. **Mitigation:** Add a `closing` flag to `PositionState` (guarded by `posStateMu`) set atomically before calling `executeCloseTrade`. Any goroutine that sees `closing == true` skips the position. Alternatively, use a per-position `sync.Once`.

**[Risk] Tick-triggered evaluation floods the risk loop on high-signal-volume symbols** → With 20 active products at 5m granularity, `evaluateOpenPositionsForSymbol` is called roughly every 5 minutes per symbol, matching the current ticker cadence. No flood risk under normal conditions. During signal bursts (multiple strategies on the same product) it could be called several times per minute per symbol. **Mitigation:** Each call is a few float comparisons and an HTTP call only if price is stale; the hot path is O(1) and sub-microsecond.

**[Risk] HardStop pre-computed at entry may mismatch current leverage if config changes** → Leverage is set per trading config, not per signal. If the config is updated after a position opens, the stored hard stop remains based on the original leverage. **Mitigation:** Acceptable — the position was entered under the original leverage and the hard stop should reflect that. Document this as intended behaviour.

**[Risk] SL-distance trailing stop breaks for positions with SL=0 (ATR not yet computed at signal time)** → Fall back to `defaultSLPct × entryPrice` as the SL distance. This is the same value used for the fallback SL in the current code. **Mitigation:** The proposal also recommends widening the ATR SL to 2.5–3× (tracked separately); reducing SL=0 cases at source is the long-term fix.

**[Risk] Library versioning drift between trader and backtester** → If one consumer updates the library and the other doesn't, their exit behaviour diverges — defeating the purpose. **Mitigation:** Pin both to the same version in CI. Document this constraint in both repos' README. Consider a shared renovate/dependabot config.

**[Risk] Granularity not stored in existing open positions after migration** → After deploying, existing `engine_position_state` rows will have `granularity = NULL`. The `maxHoldDuration` function defaults to 48h for NULL granularity, which is safe (no premature close). **Mitigation:** The default is intentionally conservative.

## Migration Plan

1. **Create `github.com/Signal-ngn/risk` repository.** Implement the library with full test coverage (pure unit tests, no external deps). Tag `v0.1.0`.
2. **Update trader:** `go get github.com/Signal-ngn/risk@v0.1.0`. Refactor `internal/engine/risk.go` to delegate exit logic to the library. Add DB migration: `ALTER TABLE engine_position_state ADD COLUMN hard_stop NUMERIC, ADD COLUMN granularity TEXT`.
3. **Deploy trader.** Existing positions have `hard_stop = NULL` and `granularity = NULL`; both fall back to safe defaults.
4. **Update spot-canvas-app:** `go get github.com/Signal-ngn/risk@v0.1.0`. Wire into `internal/backtest/backtest.go`'s `checkSLTP` path.
5. No rollback concern for the DB migration — columns are nullable and additive.

## Open Questions

- Should the hard stop floor for spot be configurable (currently hardcoded 7%), or is 7% universal enough?
- Should the ATR multiplier widening (2→2.5×) be tackled in this change or a separate one? The proposal mentions it but it touches the strategy/model side, not the engine. Currently scoped out.
