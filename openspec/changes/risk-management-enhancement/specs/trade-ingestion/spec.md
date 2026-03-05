## MODIFIED Requirements

### Requirement: Strategy metadata fields in trade events
The system SHALL accept optional strategy metadata fields in trade events: `strategy` (string), `entry_reason` (string), `exit_reason` (string), `confidence` (float64, 0–1), `stop_loss` (float64), and `take_profit` (float64). All fields SHALL be nullable and omitting them SHALL NOT affect validation.

When a trade event is published by the engine as a result of a risk management exit (hard stop, signal SL, trailing stop, time-based exit, or signal TP), the `exit_reason` field SHALL contain a structured string in the format `"Layer <N>: <label> — <detail>"` identifying which risk layer triggered the close. When a trade event is published as a result of a conviction-drop signal (SELL/COVER) the `exit_reason` field SHALL contain either the strategy-supplied reason string or `"Layer 3: conviction drop"` if no reason was provided.

The system SHALL persist the `exit_reason` value exactly as received without modification or validation of the layer format.

#### Scenario: Trade event with all metadata fields
- **WHEN** a trade event includes strategy, entry_reason, confidence, stop_loss, and take_profit fields
- **THEN** the system SHALL persist all metadata fields alongside the base trade fields

#### Scenario: Trade event without metadata fields
- **WHEN** a trade event is received without any of the new metadata fields
- **THEN** the system SHALL persist the trade with all metadata fields set to NULL

#### Scenario: Trade event with partial metadata
- **WHEN** a trade event includes strategy and confidence but omits entry_reason, stop_loss, and take_profit
- **THEN** the system SHALL persist the provided fields and set omitted fields to NULL

#### Scenario: Engine hard stop exit recorded
- **WHEN** a trade event is received with exit_reason `"Layer 2: hard stop — 15.3% adverse move at 2× leverage"`
- **THEN** the system SHALL persist the exit_reason exactly as provided

#### Scenario: Engine trailing stop exit recorded
- **WHEN** a trade event is received with exit_reason `"Layer 4: trailing stop — trailing at $0.4380, best price $0.4480"`
- **THEN** the system SHALL persist the exit_reason exactly as provided

#### Scenario: Engine time exit recorded
- **WHEN** a trade event is received with exit_reason `"Layer 5: time exit — 12-candle hold limit reached (held 12h4m)"`
- **THEN** the system SHALL persist the exit_reason exactly as provided

#### Scenario: Conviction-drop exit with fallback reason
- **WHEN** a trade event is received with exit_reason `"Layer 3: conviction drop"`
- **THEN** the system SHALL persist the exit_reason exactly as provided

#### Scenario: Confidence value validation
- **WHEN** a trade event includes a confidence value
- **THEN** the system SHALL accept any float64 value (validation of 0–1 range is the publisher's responsibility)
