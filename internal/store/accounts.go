package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"ledger/internal/domain"
)

// GetOrCreateAccount looks up an account by ID. If it doesn't exist, creates it.
func (r *Repository) GetOrCreateAccount(ctx context.Context, id string, accountType domain.AccountType) (*domain.Account, error) {
	var acct domain.Account
	var acctType string
	err := r.pool.QueryRow(ctx,
		"SELECT id, name, type, created_at FROM ledger_accounts WHERE id = $1", id,
	).Scan(&acct.ID, &acct.Name, &acctType, &acct.CreatedAt)

	if err == pgx.ErrNoRows {
		// Auto-create account
		name := id
		_, err := r.pool.Exec(ctx,
			"INSERT INTO ledger_accounts (id, name, type) VALUES ($1, $2, $3)",
			id, name, string(accountType),
		)
		if err != nil {
			return nil, fmt.Errorf("create account: %w", err)
		}

		return r.GetOrCreateAccount(ctx, id, accountType)
	}
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}

	acct.Type = domain.AccountType(acctType)
	return &acct, nil
}

// AccountExists checks if an account with the given ID exists.
func (r *Repository) AccountExists(ctx context.Context, id string) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM ledger_accounts WHERE id = $1", id,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check account: %w", err)
	}
	return count > 0, nil
}

// ListAccounts returns all accounts.
func (r *Repository) ListAccounts(ctx context.Context) ([]domain.Account, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT id, name, type, created_at FROM ledger_accounts ORDER BY created_at")
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()

	var accounts []domain.Account
	for rows.Next() {
		var acct domain.Account
		var acctType string
		if err := rows.Scan(&acct.ID, &acct.Name, &acctType, &acct.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		acct.Type = domain.AccountType(acctType)
		accounts = append(accounts, acct)
	}

	if accounts == nil {
		accounts = []domain.Account{}
	}
	return accounts, nil
}
