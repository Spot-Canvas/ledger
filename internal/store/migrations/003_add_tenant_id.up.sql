-- Migration 003: add tenant_id to all ledger tables for multi-tenancy
-- Backfills existing rows to the default tenant '00000000-0000-0000-0000-000000000001'
-- then removes the DEFAULT so future inserts must supply an explicit tenant_id.
--
-- The FK constraints on ledger_trades/positions/orders that reference ledger_accounts(id)
-- must be dropped before changing the accounts primary key to (tenant_id, id).
-- We do not restore them because a composite PK cannot be trivially referenced by a
-- single-column FK; tenant_id filtering in queries provides the logical isolation instead.

-- Drop FK constraints that depend on ledger_accounts(id) PK
ALTER TABLE ledger_trades    DROP CONSTRAINT IF EXISTS ledger_trades_account_id_fkey;
ALTER TABLE ledger_positions DROP CONSTRAINT IF EXISTS ledger_positions_account_id_fkey;
ALTER TABLE ledger_orders    DROP CONSTRAINT IF EXISTS ledger_orders_account_id_fkey;

-- Accounts: add tenant_id, then migrate primary key to composite (tenant_id, id)
ALTER TABLE ledger_accounts
    ADD COLUMN tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001';

ALTER TABLE ledger_accounts DROP CONSTRAINT ledger_accounts_pkey;
ALTER TABLE ledger_accounts ADD PRIMARY KEY (tenant_id, id);

ALTER TABLE ledger_accounts ALTER COLUMN tenant_id DROP DEFAULT;

-- Trades: add tenant_id, drop old single-column indexes, add composite index
ALTER TABLE ledger_trades
    ADD COLUMN tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001';

ALTER TABLE ledger_trades ALTER COLUMN tenant_id DROP DEFAULT;

DROP INDEX IF EXISTS idx_ledger_trades_account_timestamp;
DROP INDEX IF EXISTS idx_ledger_trades_account_symbol_timestamp;

CREATE INDEX idx_ledger_trades_tenant_account_timestamp
    ON ledger_trades (tenant_id, account_id, timestamp DESC);

-- Positions: add tenant_id, replace unique index with tenant-scoped version
ALTER TABLE ledger_positions
    ADD COLUMN tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001';

ALTER TABLE ledger_positions ALTER COLUMN tenant_id DROP DEFAULT;

DROP INDEX IF EXISTS idx_ledger_positions_open_unique;
DROP INDEX IF EXISTS idx_ledger_positions_account_status;

CREATE UNIQUE INDEX idx_ledger_positions_open_unique
    ON ledger_positions (tenant_id, account_id, symbol, market_type)
    WHERE status = 'open';

CREATE INDEX idx_ledger_positions_tenant_account_status
    ON ledger_positions (tenant_id, account_id, status);

-- Orders: add tenant_id, replace single-column index with composite
ALTER TABLE ledger_orders
    ADD COLUMN tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001';

ALTER TABLE ledger_orders ALTER COLUMN tenant_id DROP DEFAULT;

DROP INDEX IF EXISTS idx_ledger_orders_account_status_created;

CREATE INDEX idx_ledger_orders_tenant_account_status_created
    ON ledger_orders (tenant_id, account_id, status, created_at DESC);
