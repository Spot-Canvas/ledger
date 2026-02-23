## MODIFIED Requirements

### Requirement: Trade history storage

The system SHALL persist all ingested trades in the `ledger_trades` table with the following
fields: **tenant ID (UUID, NOT NULL)**, trade ID (unique within tenant), account ID, symbol,
side (buy/sell), quantity, price, fee, fee currency, market type (spot/futures), timestamp,
and ingestion timestamp. Futures trades SHALL additionally store leverage, margin, and
liquidation price. All trades SHALL additionally store optional metadata fields: strategy
(string), entry_reason (string), exit_reason (string), confidence (float64), stop_loss
(float64), and take_profit (float64).

#### Scenario: Trade persisted with all fields including tenant_id
- **WHEN** a valid trade event is ingested
- **THEN** the system SHALL store all trade fields including `tenant_id`, any provided metadata fields, and set the ingestion timestamp to the current time

#### Scenario: Trade fields queryable
- **WHEN** a trade has been persisted
- **THEN** all stored fields including `tenant_id` and metadata fields SHALL be retrievable via the query interface

#### Scenario: Trade persisted without metadata fields
- **WHEN** a valid trade event is ingested without metadata fields
- **THEN** the system SHALL store the trade with metadata fields set to NULL

---

### Requirement: Query trades by account

The system SHALL support querying trades filtered by **tenant ID and** account ID, with results
ordered by timestamp descending (most recent first). All queries SHALL be scoped to the
caller's `tenant_id`; trades belonging to other tenants SHALL never be returned.

#### Scenario: Query all trades for an account
- **WHEN** trades are queried for the authenticated tenant's "live" account with no additional filters
- **THEN** the system SHALL return all trades for that tenant's account ordered by timestamp descending

#### Scenario: Query trades for account with no trades
- **WHEN** trades are queried for an account that has no trades for the authenticated tenant
- **THEN** the system SHALL return an empty list

#### Scenario: Cross-tenant isolation in trade queries
- **WHEN** tenant A queries trades for account "live" and tenant B also has trades in a "live" account
- **THEN** the system SHALL return only tenant A's trades

---

### Requirement: Query orders by account

The system SHALL support querying orders filtered by **tenant ID and** account ID and status,
with results ordered by created timestamp descending. All queries SHALL be scoped to the
caller's `tenant_id`.

#### Scenario: Query open orders
- **WHEN** orders are queried for the authenticated tenant's "live" account with status "open"
- **THEN** the system SHALL return only open orders for that tenant's account

#### Scenario: Query all orders for an account
- **WHEN** orders are queried for the authenticated tenant's "live" account with no status filter
- **THEN** the system SHALL return all orders for that tenant ordered by created timestamp descending

#### Scenario: Cross-tenant isolation in order queries
- **WHEN** tenant A queries orders for account "live" and tenant B also has orders in a "live" account
- **THEN** the system SHALL return only tenant A's orders
