package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository provides database access for the ledger service.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository with a connection pool.
func NewRepository(ctx context.Context, databaseURL string) (*Repository, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	return &Repository{pool: pool}, nil
}

// Pool returns the underlying connection pool (for migration runner).
func (r *Repository) Pool() *pgxpool.Pool {
	return r.pool
}

// Ping checks the database connection.
func (r *Repository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

// Close closes the connection pool.
func (r *Repository) Close() {
	r.pool.Close()
}
