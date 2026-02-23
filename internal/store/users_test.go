package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// TestUserRepository_NilKey verifies that uuid.Nil returns (nil, nil) without a DB call.
func TestUserRepository_NilKey(t *testing.T) {
	repo := &UserRepository{pool: nil} // pool is nil — must not be called
	user, err := repo.GetByAPIKey(context.Background(), uuid.Nil)
	if err != nil {
		t.Fatalf("expected nil error for uuid.Nil, got: %v", err)
	}
	if user != nil {
		t.Fatalf("expected nil user for uuid.Nil, got: %+v", user)
	}
}
