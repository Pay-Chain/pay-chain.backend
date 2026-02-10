package repositories

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	domainRepos "pay-chain.backend/internal/domain/repositories"
)

type contextKey string

const (
	txKey contextKey = "tx_db"
)

// UnitOfWorkImpl implements UnitOfWork using GORM
type UnitOfWorkImpl struct {
	db *gorm.DB
}

// NewUnitOfWork creates a new UnitOfWork
func NewUnitOfWork(db *gorm.DB) domainRepos.UnitOfWork {
	return &UnitOfWorkImpl{db: db}
}

// Do executes the given function within a transaction scope
func (u *UnitOfWorkImpl) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	// Check if transaction already exists (nested transaction not fully supported, but we can reuse)
	// For simplicity, we create a new transaction or savepoint if supported by driver,
	// but mostly we just want to ensure we are in a tx.

	// However, GORM supports nested transactions via SavePoint.
	// But let's start with basic:

	// If a tx is already in context, we could reuse it (nested Do calls).
	// But typically UoW is top-level.

	tx := u.GetDB(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	// Inject tx into context
	txCtx := context.WithValue(ctx, txKey, tx)

	// Execute function
	if err := fn(txCtx); err != nil {
		tx.Rollback()
		return err
	}

	// Commit
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetDB extracts the Transaction DB from context if present, otherwise returns standard DB
// This is a helper for Repositories.
// NOTE: We need the base DB fallback.
// Since this method is on UnitOfWorkImpl, Repositories don't have access to it easily unless they depend on UoW implementation?
// OR we make this a standalone helper function in this package.
// But repositories are in the same package `repositories` (infrastructure side).
// So they can call a package-level function `GetDB(ctx, fallbackDB)`.
func (u *UnitOfWorkImpl) GetDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey).(*gorm.DB); ok {
		return tx
	}
	return u.db
}

// Helper for other repositories in this package
func GetDB(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey).(*gorm.DB); ok {
		return tx
	}
	return fallback
}
