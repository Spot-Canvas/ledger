## ADDED Requirements

### Requirement: per-strategy max hold duration
The system SHALL enforce a maximum hold duration for every open position. The limit is determined by the strategy name prefix and the candle granularity at entry time. When `time.Since(openedAt)` exceeds the limit, the system SHALL close the position regardless of profitability.

Hold limits:

| Strategy prefix | Granularity | Candle count | Wall-clock duration |
|---|---|---|---|
| `ml_xgboost` | `FIVE_MINUTES` | 30 | 2.5 hours |
| `ml_transformer` | `FIVE_MINUTES` | 24 | 2 hours |
| `ml_transformer` | `ONE_HOUR` | 12 | 12 hours |
| rule-based | `FIVE_MINUTES` | 48 | 4 hours |
| rule-based | `ONE_HOUR` | 24 | 24 hours |
| (default / unrecognised) | any | 48 | 48 hours |

Granularity is stored at entry time from the trading config and is immutable for the life of the position.

#### Scenario: ml_xgboost 5m position exceeds hold limit
- **WHEN** an `ml_xgboost` position on `FIVE_MINUTES` granularity has been open for 2 hours 31 minutes
- **THEN** the system SHALL close the position with exit reason containing "Layer 5: time exit"

#### Scenario: ml_transformer 5m position within hold limit
- **WHEN** an `ml_transformer` position on `FIVE_MINUTES` granularity has been open for 1 hour 55 minutes
- **THEN** the system SHALL NOT trigger a time-based exit

#### Scenario: ml_transformer 1h position exceeds hold limit
- **WHEN** an `ml_transformer` position on `ONE_HOUR` granularity has been open for 12 hours 5 minutes
- **THEN** the system SHALL close the position with exit reason containing "Layer 5: time exit"

#### Scenario: rule-based 5m position exceeds hold limit
- **WHEN** a rule-based strategy position on `FIVE_MINUTES` granularity has been open for 4 hours 1 minute
- **THEN** the system SHALL close the position with exit reason containing "Layer 5: time exit"

#### Scenario: rule-based 1h position within hold limit
- **WHEN** a rule-based strategy position on `ONE_HOUR` granularity has been open for 23 hours 55 minutes
- **THEN** the system SHALL NOT trigger a time-based exit

#### Scenario: unknown strategy/granularity defaults to 48h
- **WHEN** a position is open under an unrecognised strategy or granularity combination
- **THEN** the hold limit SHALL default to 48 hours

---

### Requirement: granularity stored at entry
The candle granularity SHALL be recorded in the position state at entry time from the matched trading config. It SHALL be immutable for the life of the position and SHALL survive engine restarts.

#### Scenario: Granularity recorded at entry
- **WHEN** a position opens under a trading config with granularity `ONE_HOUR`
- **THEN** the position state SHALL record `granularity = "ONE_HOUR"`

#### Scenario: Granularity survives restart
- **WHEN** the engine restarts with an open position
- **THEN** the granularity SHALL be restored from the persistent position state and used for hold limit evaluation

#### Scenario: Null granularity on legacy positions
- **WHEN** an existing position has no granularity stored (pre-migration row)
- **THEN** the system SHALL apply the default 48-hour hold limit
