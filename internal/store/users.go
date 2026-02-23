package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuthUser holds the user fields the ledger needs after resolving an API key.
type AuthUser struct {
	TenantID uuid.UUID
}

// UserRepository provides read-only access to the shared users table.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository backed by the given pool.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// GetByAPIKey looks up a user by API key and returns their tenant ID.
// Returns nil, nil if the key is not found.
// Returns nil, nil for uuid.Nil (zero UUID) without hitting the database.
func (r *UserRepository) GetByAPIKey(ctx context.Context, apiKey uuid.UUID) (*AuthUser, error) {
	if apiKey == uuid.Nil {
		return nil, nil
	}

	var tenantID uuid.UUID
	err := r.pool.QueryRow(ctx,
		"SELECT tenant_id FROM users WHERE api_key = $1",
		apiKey,
	).Scan(&tenantID)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by api key: %w", err)
	}

	return &AuthUser{TenantID: tenantID}, nil
}
