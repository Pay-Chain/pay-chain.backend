package repositories

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	domainRepos "pay-chain.backend/internal/domain/repositories"
)

type contextKey string

const (
	txKey   contextKey = "tx_db"
	lockKey contextKey = "lock"
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

// WithLock adds a locking clause to the context for subsequent repository calls
func (u *UnitOfWorkImpl) WithLock(ctx context.Context) context.Context {
	return context.WithValue(ctx, lockKey, true)
}

// GetDB extracts the Transaction DB from context if present, otherwise returns standard DB
func (u *UnitOfWorkImpl) GetDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey).(*gorm.DB); ok {
		return tx
	}
	return u.db
}

// Helper for other repositories in this package
func GetDB(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	db := fallback
	if tx, ok := ctx.Value(txKey).(*gorm.DB); ok {
		db = tx
	}

	// Check for lock request
	if lock, ok := ctx.Value(lockKey).(bool); ok && lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}

	return db
}
