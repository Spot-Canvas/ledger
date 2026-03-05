## ADDED Requirements

### Requirement: hard stop threshold computation
The system SHALL compute a hard stop price at position entry time using the formula `max_adverse_pct = max(30% / leverage, 7%)`. The resulting price level SHALL be stored alongside the position and used for all subsequent exit checks. The hard stop SHALL be immutable after entry — it is never updated or removed while the position is open.

Threshold table:

| Market type | Leverage | max_adverse_pct | Long hard stop | Short hard stop |
|---|---|---|---|---|
| Spot | 1 | 7% | entry × 0.93 | entry × 1.07 |
| Futures | 2× | 15% | entry × 0.85 | entry × 1.15 |
| Futures | 3× | 10% | entry × 0.90 | entry × 1.10 |
| Futures | 5× | 6% | entry × 0.94 | entry × 1.06 |

#### Scenario: Spot long hard stop computed at entry
- **WHEN** a long spot position opens at entry price $1.000
- **THEN** the hard stop SHALL be set to $0.930 (7% adverse)

#### Scenario: Futures 2× long hard stop computed at entry
- **WHEN** a long futures position opens at entry price $1.000 with leverage 2
- **THEN** the hard stop SHALL be set to $0.850 (15% adverse)

#### Scenario: Futures 2× short hard stop computed at entry
- **WHEN** a short futures position opens at entry price $1.000 with leverage 2
- **THEN** the hard stop SHALL be set to $1.150 (15% adverse)

#### Scenario: Futures 3× hard stop computed at entry
- **WHEN** a futures position opens at entry price $1.000 with leverage 3
- **THEN** the hard stop SHALL be set to $0.900 for long (10% adverse) and $1.100 for short

#### Scenario: Futures 5× hard stop computed at entry
- **WHEN** a futures position opens at entry price $1.000 with leverage 5
- **THEN** the hard stop SHALL be set to $0.940 for long (6% adverse) and $1.060 for short

#### Scenario: Hard stop is immutable after entry
- **WHEN** a position is open and the signal SL or any config value changes
- **THEN** the hard stop price SHALL remain unchanged at its entry-time value

---

### Requirement: hard stop exit for long positions
For a long position, the system SHALL trigger a hard stop exit when the evaluated price is at or below the hard stop level. In candle-based evaluation (backtester), the candle Low SHALL be used as the evaluated price. In tick-based evaluation (live engine), the current tick price SHALL be used.

#### Scenario: Long position — hard stop triggered on tick
- **WHEN** a long position has hard stop $0.850 and the current price is $0.848
- **THEN** the system SHALL exit the position with exit reason containing "Layer 2: hard stop"

#### Scenario: Long position — hard stop triggered on candle low
- **WHEN** a long position has hard stop $0.850 and the candle Low is $0.847 (even if Close is $0.860)
- **THEN** the system SHALL exit the position with exit reason containing "Layer 2: hard stop"

#### Scenario: Long position — hard stop not triggered
- **WHEN** a long position has hard stop $0.850 and the current price is $0.851
- **THEN** the system SHALL NOT trigger a hard stop exit

---

### Requirement: hard stop exit for short positions
For a short position, the system SHALL trigger a hard stop exit when the evaluated price is at or above the hard stop level. In candle-based evaluation, the candle High SHALL be used. In tick-based evaluation, the current tick price SHALL be used.

#### Scenario: Short position — hard stop triggered on tick
- **WHEN** a short position has hard stop $1.150 and the current price is $1.152
- **THEN** the system SHALL exit the position with exit reason containing "Layer 2: hard stop"

#### Scenario: Short position — hard stop triggered on candle high
- **WHEN** a short position has hard stop $1.150 and the candle High is $1.153 (even if Close is $1.140)
- **THEN** the system SHALL exit the position with exit reason containing "Layer 2: hard stop"

#### Scenario: Short position — hard stop not triggered
- **WHEN** a short position has hard stop $1.150 and the current price is $1.148
- **THEN** the system SHALL NOT trigger a hard stop exit

---

### Requirement: hard stop supersedes absent or zero signal SL
The hard stop SHALL fire even when the signal SL is zero or absent. It is the exit of last resort and its activation does not depend on any signal-provided stop level.

#### Scenario: Hard stop fires when signal SL is zero
- **WHEN** a position has signal SL = 0 (ATR warmup, SL unavailable) and price reaches the hard stop level
- **THEN** the system SHALL exit the position via the hard stop

#### Scenario: Hard stop fires when signal SL has not yet been reached
- **WHEN** a position has signal SL $0.820 (wider than hard stop $0.850) and price drops to $0.848
- **THEN** the system SHALL exit via the hard stop (Layer 2) and NOT wait for the signal SL (Layer 1)
