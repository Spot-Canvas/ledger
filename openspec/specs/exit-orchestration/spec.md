## ADDED Requirements

### Requirement: exit priority ordering
When multiple exit conditions are satisfied simultaneously, the system SHALL resolve them in the following strict priority order and apply only the highest-priority exit:

1. **Layer 3 — Conviction-drop signal** (`IsExit=true` from strategy): SELL or COVER signal received from the signal stream.
2. **Layer 2 — Engine hard stop**: price reached the leverage-scaled circuit-breaker level.
3. **Layer 1 — Signal SL**: price reached the stop-loss level from the entry signal.
4. **Layer 4 — Trailing stop**: price hit the trailing stop level (ML strategies only).
5. **Layer 5 — Time-based exit**: position has been held beyond the strategy/granularity hold limit.
6. **Layer 6 — Signal TP**: price reached the take-profit level from the entry signal (rule-based strategies only).

Layer 3 (conviction-drop) takes natural priority because it fires on the signal goroutine and removes the position from state before the risk evaluation loop can process it. Layers 2–6 are evaluated in the order above within a single `Evaluate` call; the first matching layer terminates evaluation.

#### Scenario: Hard stop and signal SL both triggered on same candle
- **WHEN** on a single candle the Low breaches both the signal SL ($0.920) and the hard stop ($0.850) for a long position
- **THEN** the system SHALL exit with Layer 2 (hard stop) reason, not Layer 1 (signal SL)

#### Scenario: Signal SL triggers while trailing stop is also active
- **WHEN** a position has an active trailing stop above the signal SL and the price hits only the signal SL
- **THEN** the system SHALL exit with Layer 1 (signal SL) reason

#### Scenario: Trailing stop triggers while position is near hold limit
- **WHEN** a position has exceeded 90% of its hold duration and the trailing stop fires on the same candle
- **THEN** the system SHALL exit with Layer 4 (trailing stop) reason, not Layer 5 (time exit)

#### Scenario: Conviction-drop signal preempts all price-level exits
- **WHEN** a SELL signal arrives for a long position while the hard stop has also been breached
- **THEN** the position SHALL have been closed by the conviction-drop path (Layer 3) and the risk loop SHALL find no open position to act on

#### Scenario: Signal TP only applies to rule-based strategies
- **WHEN** a rule-based strategy position reaches its take-profit level
- **THEN** the system SHALL exit with Layer 6 (signal TP) reason

#### Scenario: Signal TP is skipped for ML strategies
- **WHEN** an ML strategy position reaches the entry signal's take-profit price
- **THEN** the system SHALL NOT exit via the take-profit; the position remains open

---

### Requirement: exit reason format
Every engine-generated position close SHALL record an `exit_reason` string in the format `"Layer <N>: <label> — <detail>"`. The detail SHALL include enough numeric context to reconstruct why the exit fired from the ledger record alone, without cross-referencing engine logs.

#### Scenario: Hard stop exit reason
- **WHEN** a 2× futures long position is closed by the hard stop after a 15.3% adverse move
- **THEN** the exit reason SHALL be `"Layer 2: hard stop — 15.3% adverse move at 2× leverage"`

#### Scenario: Signal SL exit reason
- **WHEN** a long position is closed because the candle Low hit the signal stop-loss at $0.4210
- **THEN** the exit reason SHALL be `"Layer 1: signal SL — price $0.4207 hit stop $0.4210"`

#### Scenario: Trailing stop breakeven exit reason
- **WHEN** a long position is closed by the trailing stop that was at entry price $0.4500 (breakeven)
- **THEN** the exit reason SHALL be `"Layer 4: trailing stop — breakeven triggered, stop at entry $0.4500"`

#### Scenario: Trailing stop active trail exit reason
- **WHEN** a long position is closed by the trailing stop that was trailing at $0.4380 with peak $0.4480
- **THEN** the exit reason SHALL be `"Layer 4: trailing stop — trailing at $0.4380, best price $0.4480"`

#### Scenario: Time exit reason
- **WHEN** a position is closed after holding for 12 candles (12h4m elapsed) against a 12-candle limit
- **THEN** the exit reason SHALL be `"Layer 5: time exit — 12-candle hold limit reached (held 12h4m)"`

#### Scenario: Signal TP exit reason
- **WHEN** a rule-based strategy long position is closed by take-profit at $0.5395
- **THEN** the exit reason SHALL be `"Layer 6: signal TP — price $0.5397 hit take-profit $0.5395"`

#### Scenario: Conviction-drop fallback reason
- **WHEN** a SELL/COVER signal closes a position and the strategy did not provide a reason field
- **THEN** the exit reason SHALL be `"Layer 3: conviction drop"`

#### Scenario: Conviction-drop with strategy-supplied reason
- **WHEN** a SELL/COVER signal closes a position and the signal payload contains a non-empty `reason` field
- **THEN** the exit reason SHALL use the strategy-supplied reason string verbatim

---

### Requirement: evaluation uses intra-candle price range
In candle-based evaluation (backtester), the system SHALL check hard stop and signal SL against the candle High and Low (intra-candle extremes) rather than the candle Close. Trailing stop SHALL be evaluated against the candle Close. In tick-based evaluation (live engine), the current tick price SHALL be used for all checks.

#### Scenario: Backtester — hard stop detected mid-candle
- **WHEN** a candle has Close $0.870 but Low $0.842, and the hard stop is $0.850
- **THEN** the system SHALL detect the hard stop hit (Low ≤ hard stop) even though Close is above it

#### Scenario: Backtester — trailing stop evaluated at close only
- **WHEN** a candle has Low $0.990 (below trailing stop $0.995) but Close $1.010 (above trailing stop)
- **THEN** the system SHALL NOT exit via trailing stop (trailing stop uses Close, not Low)

#### Scenario: Live engine — tick price used for all checks
- **WHEN** the current tick price is $0.848 and the hard stop is $0.850
- **THEN** the system SHALL trigger the hard stop exit at the tick price
