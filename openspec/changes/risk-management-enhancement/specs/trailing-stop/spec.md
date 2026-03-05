## ADDED Requirements

### Requirement: trailing stop is ML-strategy-only
The trailing stop mechanism SHALL be active only for strategies whose name begins with the prefix `ml_`. For all other strategies (rule-based), the trailing stop SHALL be permanently disabled regardless of position profitability.

#### Scenario: Trailing stop active for ML strategy
- **WHEN** a position is open under strategy `ml_transformer_1h` and profit reaches 1× SL distance
- **THEN** the system SHALL activate the trailing stop

#### Scenario: Trailing stop disabled for rule-based strategy
- **WHEN** a position is open under strategy `rsi_divergence` and profit reaches any level
- **THEN** the system SHALL NOT activate a trailing stop

#### Scenario: Trailing stop active for ml_xgboost strategy
- **WHEN** a position is open under strategy `ml_xgboost_short`
- **THEN** the system SHALL treat it as an ML strategy (prefix `ml_`) and enable trailing stop

---

### Requirement: SL distance computation
The trailing stop activation threshold and trail width are both derived from the signal SL distance: `slDistance = abs(entryPrice − stopLoss)`. When the signal SL is zero or absent, the system SHALL fall back to `slDistance = entryPrice × 0.04` (4% of entry price).

#### Scenario: SL distance from signal SL
- **WHEN** a long position opens at $1.000 with signal SL $0.960
- **THEN** slDistance SHALL be $0.040

#### Scenario: SL distance fallback when SL is zero
- **WHEN** a long position opens at $1.000 with signal SL = 0
- **THEN** slDistance SHALL be $0.040 (4% × $1.000)

---

### Requirement: breakeven activation
When an ML-strategy position's unrealised profit (measured at the candle close or current tick price) first reaches or exceeds `1 × slDistance`, the system SHALL move the trailing stop to the entry price (breakeven). This prevents a winning position from becoming a losing one.

#### Scenario: Long position — breakeven triggered
- **WHEN** a long position opens at $1.000 with slDistance $0.040, and the close price reaches $1.040
- **THEN** the trailing stop SHALL be set to $1.000 (entry price)

#### Scenario: Short position — breakeven triggered
- **WHEN** a short position opens at $1.000 with slDistance $0.040, and the close price falls to $0.960
- **THEN** the trailing stop SHALL be set to $1.000 (entry price)

#### Scenario: Breakeven not triggered below threshold
- **WHEN** a long position opens at $1.000 with slDistance $0.040, and the close price is $1.039
- **THEN** the trailing stop SHALL remain unset (no breakeven activation)

---

### Requirement: active trailing
Once an ML-strategy position's unrealised profit first reaches or exceeds `2 × slDistance`, the trailing stop SHALL trail `1 × slDistance` behind the running peak price. As the peak price advances, the trailing stop SHALL be updated upward (long) or downward (short). The trailing stop SHALL never move against the position.

#### Scenario: Long position — active trailing activated
- **WHEN** a long position opens at $1.000 with slDistance $0.040, peak price reaches $1.080 (2× slDistance profit)
- **THEN** the trailing stop SHALL be set to $1.040 ($1.080 − $0.040)

#### Scenario: Long position — trailing stop advances with peak
- **WHEN** the peak price advances from $1.080 to $1.100
- **THEN** the trailing stop SHALL advance from $1.040 to $1.060

#### Scenario: Long position — trailing stop does not retreat
- **WHEN** the price falls from peak $1.100 to $1.090 (not hitting the trailing stop)
- **THEN** the trailing stop SHALL remain at $1.060 and SHALL NOT decrease

#### Scenario: Short position — active trailing activated
- **WHEN** a short position opens at $1.000 with slDistance $0.040, price falls to $0.920 (2× slDistance profit)
- **THEN** the trailing stop SHALL be set to $0.960 ($0.920 + $0.040)

#### Scenario: Short position — trailing stop advances with falling peak
- **WHEN** the peak price advances from $0.920 to $0.900
- **THEN** the trailing stop SHALL decrease from $0.960 to $0.940

---

### Requirement: trailing stop exit
The system SHALL exit a position when the close price (backtester) or tick price (live engine) reaches or crosses the trailing stop level.

#### Scenario: Long position — trailing stop hit
- **WHEN** a long position has trailing stop $1.040 and the close price falls to $1.038
- **THEN** the system SHALL exit the position with exit reason containing "Layer 4: trailing stop"

#### Scenario: Short position — trailing stop hit
- **WHEN** a short position has trailing stop $0.960 and the close price rises to $0.962
- **THEN** the system SHALL exit the position with exit reason containing "Layer 4: trailing stop"

#### Scenario: Trailing stop not hit
- **WHEN** a long position has trailing stop $1.040 and the close price is $1.041
- **THEN** the system SHALL NOT exit the position via the trailing stop

---

### Requirement: trailing stop state persistence
The current `PeakPrice` and `TrailingStop` values SHALL be persisted whenever they advance so that the state survives a service restart.

#### Scenario: State updated when trailing stop advances
- **WHEN** the peak price advances and the trailing stop moves to a new level
- **THEN** the updated `PeakPrice` and `TrailingStop` values SHALL be written to the persistent position state store

#### Scenario: State restored on restart
- **WHEN** the engine restarts with an open position that had an active trailing stop
- **THEN** the engine SHALL restore `PeakPrice` and `TrailingStop` from the store and continue trailing from where it left off
