-- Migration 003 DOWN: reverse tenant_id addition

-- Orders: restore old index, drop tenant_id
DROP INDEX IF EXISTS idx_ledger_orders_tenant_account_status_created;

CREATE INDEX idx_ledger_orders_account_status_created
    ON ledger_orders (account_id, status, created_at DESC);

ALTER TABLE ledger_orders DROP COLUMN tenant_id;

-- Positions: restore old indexes, drop tenant_id
DROP INDEX IF EXISTS idx_ledger_positions_open_unique;
DROP INDEX IF EXISTS idx_ledger_positions_tenant_account_status;

CREATE UNIQUE INDEX idx_ledger_positions_open_unique
    ON ledger_positions (account_id, symbol, market_type)
    WHERE status = 'open';

CREATE INDEX idx_ledger_positions_account_status
    ON ledger_positions (account_id, status);

ALTER TABLE ledger_positions DROP COLUMN tenant_id;

-- Trades: restore old indexes, drop tenant_id
DROP INDEX IF EXISTS idx_ledger_trades_tenant_account_timestamp;

CREATE INDEX idx_ledger_trades_account_timestamp
    ON ledger_trades (account_id, timestamp DESC);

CREATE INDEX idx_ledger_trades_account_symbol_timestamp
    ON ledger_trades (account_id, symbol, timestamp DESC);

ALTER TABLE ledger_trades DROP COLUMN tenant_id;

-- Accounts: restore simple primary key, drop tenant_id
ALTER TABLE ledger_accounts DROP CONSTRAINT ledger_accounts_pkey;
ALTER TABLE ledger_accounts ADD PRIMARY KEY (id);

ALTER TABLE ledger_accounts DROP COLUMN tenant_id;

-- Restore FK constraints
ALTER TABLE ledger_trades
    ADD CONSTRAINT ledger_trades_account_id_fkey
    FOREIGN KEY (account_id) REFERENCES ledger_accounts(id);

ALTER TABLE ledger_positions
    ADD CONSTRAINT ledger_positions_account_id_fkey
    FOREIGN KEY (account_id) REFERENCES ledger_accounts(id);

ALTER TABLE ledger_orders
    ADD CONSTRAINT ledger_orders_account_id_fkey
    FOREIGN KEY (account_id) REFERENCES ledger_accounts(id);
