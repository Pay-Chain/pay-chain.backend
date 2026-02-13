package repositories

import (
	"context"
)

// UnitOfWork defines the interface for atomic operations
type UnitOfWork interface {
	// Do executes the given function within a transaction scope
	Do(ctx context.Context, fn func(ctx context.Context) error) error
	// WithLock adds a locking clause to the context for subsequent repository calls
	WithLock(ctx context.Context) context.Context
}
