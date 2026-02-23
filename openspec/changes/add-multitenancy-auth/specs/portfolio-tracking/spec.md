## MODIFIED Requirements

### Requirement: Account management

The system SHALL maintain a `ledger_accounts` table that stores trading accounts scoped by
tenant. The primary key SHALL be the composite `(tenant_id, id)` pair. Each account MUST have
a `tenant_id` (UUID), a unique ID within that tenant (e.g. `"live"`, `"paper"`), name, type
(live/paper), and creation timestamp. Account IDs are unique per tenant, not globally.

#### Scenario: Account referenced by trade within tenant
- **WHEN** a trade references an account ID that exists in `ledger_accounts` for the same tenant
- **THEN** the trade SHALL be associated with that account

#### Scenario: Trade for unknown account auto-creates within tenant
- **WHEN** a trade references an account ID that does not exist in `ledger_accounts` for the trade's tenant
- **THEN** the system SHALL auto-create the account scoped to that tenant, with the ID derived from the NATS event and type inferred from the account name

#### Scenario: Same account ID for different tenants is independent
- **WHEN** tenant A and tenant B both have trades for `account_id: "live"`
- **THEN** the system SHALL maintain two separate `ledger_accounts` rows, one per tenant, each with its own positions

---

### Requirement: Cross-account isolation

The system SHALL ensure that positions are isolated per account **and per tenant**. A trade in
one account or tenant SHALL NOT affect positions in another account or tenant, even for the same
symbol.

#### Scenario: Same symbol in different accounts within same tenant
- **WHEN** a buy trade for BTC-USD is ingested for the "live" account and a separate buy trade for BTC-USD is ingested for the "paper" account (same tenant)
- **THEN** the system SHALL maintain two independent positions, one per account, each with its own quantity and cost basis

#### Scenario: Same account ID and symbol in different tenants
- **WHEN** tenant A and tenant B both ingest a buy trade for BTC-USD in a "live" account
- **THEN** the system SHALL maintain two independent positions, one per tenant, with no cross-contamination

---

### Requirement: Position rebuild from trade history

The system SHALL support rebuilding all positions for an account by replaying the full trade
history in chronological order. The rebuilt positions MUST match the current materialized
positions. The rebuild operation SHALL be scoped to a specific `tenant_id`.

#### Scenario: Rebuild positions after data repair
- **WHEN** a position rebuild is triggered for a specific tenant's account
- **THEN** the system SHALL delete all existing positions for that tenant's account, replay all trades for that tenant in timestamp order, and produce positions identical to what incremental processing would have created
