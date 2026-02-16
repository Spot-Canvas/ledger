-- Ledger initial schema
-- Tables are prefixed with ledger_ to avoid collisions in shared database

-- Accounts table
CREATE TABLE IF NOT EXISTS ledger_accounts (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL CHECK (type IN ('live', 'paper')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Trades table
CREATE TABLE IF NOT EXISTS ledger_trades (
    trade_id          TEXT PRIMARY KEY,
    account_id        TEXT NOT NULL REFERENCES ledger_accounts(id),
    symbol            TEXT NOT NULL,
    side              TEXT NOT NULL CHECK (side IN ('buy', 'sell')),
    quantity          DOUBLE PRECISION NOT NULL,
    price             DOUBLE PRECISION NOT NULL,
    fee               DOUBLE PRECISION NOT NULL DEFAULT 0,
    fee_currency      TEXT NOT NULL DEFAULT '',
    market_type       TEXT NOT NULL CHECK (market_type IN ('spot', 'futures')),
    timestamp         TIMESTAMPTZ NOT NULL,
    ingested_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cost_basis        DOUBLE PRECISION NOT NULL DEFAULT 0,
    realized_pnl      DOUBLE PRECISION NOT NULL DEFAULT 0,
    -- Futures-specific fields (nullable for spot trades)
    leverage          INTEGER,
    margin            DOUBLE PRECISION,
    liquidation_price DOUBLE PRECISION,
    funding_fee       DOUBLE PRECISION
);

CREATE INDEX IF NOT EXISTS idx_ledger_trades_account_timestamp
    ON ledger_trades (account_id, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_ledger_trades_account_symbol_timestamp
    ON ledger_trades (account_id, symbol, timestamp DESC);

-- Positions table
CREATE TABLE IF NOT EXISTS ledger_positions (
    id                TEXT PRIMARY KEY,
    account_id        TEXT NOT NULL REFERENCES ledger_accounts(id),
    symbol            TEXT NOT NULL,
    market_type       TEXT NOT NULL CHECK (market_type IN ('spot', 'futures')),
    side              TEXT NOT NULL CHECK (side IN ('long', 'short')),
    quantity          DOUBLE PRECISION NOT NULL DEFAULT 0,
    avg_entry_price   DOUBLE PRECISION NOT NULL DEFAULT 0,
    cost_basis        DOUBLE PRECISION NOT NULL DEFAULT 0,
    realized_pnl      DOUBLE PRECISION NOT NULL DEFAULT 0,
    leverage          INTEGER,
    margin            DOUBLE PRECISION,
    liquidation_price DOUBLE PRECISION,
    status            TEXT NOT NULL CHECK (status IN ('open', 'closed')) DEFAULT 'open',
    opened_at         TIMESTAMPTZ NOT NULL,
    closed_at         TIMESTAMPTZ
);

-- Unique constraint: only one open position per account/symbol/market_type
CREATE UNIQUE INDEX IF NOT EXISTS idx_ledger_positions_open_unique
    ON ledger_positions (account_id, symbol, market_type)
    WHERE status = 'open';

CREATE INDEX IF NOT EXISTS idx_ledger_positions_account_status
    ON ledger_positions (account_id, status);

-- Orders table
CREATE TABLE IF NOT EXISTS ledger_orders (
    order_id       TEXT PRIMARY KEY,
    account_id     TEXT NOT NULL REFERENCES ledger_accounts(id),
    symbol         TEXT NOT NULL,
    side           TEXT NOT NULL CHECK (side IN ('buy', 'sell')),
    order_type     TEXT NOT NULL CHECK (order_type IN ('market', 'limit')),
    requested_qty  DOUBLE PRECISION NOT NULL,
    filled_qty     DOUBLE PRECISION NOT NULL DEFAULT 0,
    avg_fill_price DOUBLE PRECISION NOT NULL DEFAULT 0,
    status         TEXT NOT NULL CHECK (status IN ('open', 'filled', 'partially_filled', 'cancelled')) DEFAULT 'open',
    market_type    TEXT NOT NULL CHECK (market_type IN ('spot', 'futures')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ledger_orders_account_status_created
    ON ledger_orders (account_id, status, created_at DESC);
