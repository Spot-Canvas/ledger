-- Prevent duplicate account names within a tenant
CREATE UNIQUE INDEX IF NOT EXISTS idx_ledger_accounts_tenant_name
    ON ledger_accounts (tenant_id, name);
