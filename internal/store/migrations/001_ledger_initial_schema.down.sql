-- Drop ledger tables in reverse order (respecting foreign keys)
DROP TABLE IF EXISTS ledger_orders;
DROP TABLE IF EXISTS ledger_positions;
DROP TABLE IF EXISTS ledger_trades;
DROP TABLE IF EXISTS ledger_accounts;
