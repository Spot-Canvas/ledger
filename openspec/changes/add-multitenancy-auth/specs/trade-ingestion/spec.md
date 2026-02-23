## MODIFIED Requirements

### Requirement: Trade event validation

The system SHALL validate each incoming trade event before persisting it. A valid trade event
MUST contain: **tenant ID (UUID string)**, trade ID, account ID, symbol, side (buy/sell),
quantity, price, fee, fee currency, timestamp, and market type (spot/futures).

#### Scenario: Valid trade event received
- **WHEN** a trade event with all required fields including a valid `tenant_id` UUID and valid values is received
- **THEN** the system SHALL persist it to the `ledger_trades` table and acknowledge the message

#### Scenario: Trade event missing required fields
- **WHEN** a trade event is missing one or more required fields
- **THEN** the system SHALL log the validation error, terminate the message (no redelivery), and NOT persist it

#### Scenario: Trade event with invalid market type
- **WHEN** a trade event has a market type other than "spot" or "futures"
- **THEN** the system SHALL log the validation error, terminate the message, and NOT persist it

#### Scenario: Trade event missing tenant_id is rejected
- **WHEN** a trade event is received without a `tenant_id` field
- **THEN** the system SHALL log the validation error, terminate the message, and NOT persist it

#### Scenario: Trade event with non-UUID tenant_id is rejected
- **WHEN** a trade event is received with `tenant_id` set to a non-UUID string (e.g. `"my-tenant"`)
- **THEN** the system SHALL log the validation error, terminate the message, and NOT persist it

---

## ADDED Requirements

### Requirement: tenant_id field in trade events

The `TradeEvent` struct SHALL include a required `TenantID` field (string, JSON key `tenant_id`)
representing the UUID of the tenant that owns this trade. The `Validate()` method SHALL return
an error if `TenantID` is empty or cannot be parsed as a valid UUID. The `ToDomain()` method
SHALL parse `TenantID` and set `Trade.TenantID` on the resulting domain object.

#### Scenario: Trade event with valid tenant_id converts to domain
- **WHEN** `ToDomain()` is called on a valid `TradeEvent` with `tenant_id: "00000000-0000-0000-0000-000000000001"`
- **THEN** the returned `Trade` has `TenantID` set to the corresponding `uuid.UUID`

#### Scenario: Validate rejects empty tenant_id
- **WHEN** `Validate()` is called on a `TradeEvent` with `tenant_id: ""`
- **THEN** an error is returned containing "tenant_id"

#### Scenario: Validate rejects non-UUID tenant_id
- **WHEN** `Validate()` is called on a `TradeEvent` with `tenant_id: "not-a-uuid"`
- **THEN** an error is returned

---

### Requirement: Tenant-scoped account creation and trade persistence

The NATS consumer SHALL use the `tenant_id` from the validated trade event for all database
operations: account creation, trade insertion, and position updates. All writes SHALL be scoped
to the event's `tenant_id`.

#### Scenario: Account auto-created within correct tenant namespace
- **WHEN** a trade event with `tenant_id: "aaaa-..."` and `account_id: "live"` is received and no such account exists for that tenant
- **THEN** the system SHALL create a `ledger_accounts` row with both `tenant_id = "aaaa-..."` and `id = "live"`

#### Scenario: Same account_id for different tenants is independent
- **WHEN** tenant A and tenant B both publish trades with `account_id: "live"`
- **THEN** the system SHALL maintain two independent account and position records, one per tenant
