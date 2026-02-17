package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUnitOfWork_DoCommitAndRollback(t *testing.T) {
	db := newTestDB(t)
	createPaymentBridgeTable(t, db)
	u := &UnitOfWorkImpl{db: db}

	// commit path
	err := u.Do(context.Background(), func(ctx context.Context) error {
		return GetDB(ctx, db).Exec("INSERT INTO payment_bridge(id,name) VALUES (?,?)", uuid.New().String(), "ccip").Error
	})
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Table("payment_bridge").Count(&count).Error)
	require.Equal(t, int64(1), count)

	// rollback path
	err = u.Do(context.Background(), func(ctx context.Context) error {
		if err := GetDB(ctx, db).Exec("INSERT INTO payment_bridge(id,name) VALUES (?,?)", uuid.New().String(), "hb").Error; err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	require.Error(t, err)

	require.NoError(t, db.Table("payment_bridge").Count(&count).Error)
	require.Equal(t, int64(1), count, "second insert must be rolled back")
}

func TestUnitOfWork_WithLockAndGetDB(t *testing.T) {
	db := newTestDB(t)
	u := &UnitOfWorkImpl{db: db}

	ctx := u.WithLock(context.Background())
	lockedDB := GetDB(ctx, db)
	require.NotNil(t, lockedDB)

	plainDB := u.GetDB(context.Background())
	require.Equal(t, db, plainDB)

	tx := db.Begin()
	txCtx := context.WithValue(context.Background(), txKey, tx)
	require.Equal(t, tx, u.GetDB(txCtx))
	tx.Rollback()
}

func TestUnitOfWork_DoBeginFailure(t *testing.T) {
	db := newTestDB(t)
	u := &UnitOfWorkImpl{db: db}

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	err = u.Do(context.Background(), func(ctx context.Context) error {
		_ = ctx
		return nil
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to begin transaction")
}

func TestUnitOfWork_DoCommitFailure_WithHook(t *testing.T) {
	db := newTestDB(t)
	createPaymentBridgeTable(t, db)
	u := &UnitOfWorkImpl{db: db}

	origCommit := commitTx
	t.Cleanup(func() { commitTx = origCommit })
	commitTx = func(tx *gorm.DB) error {
		_ = tx
		return errors.New("forced commit fail")
	}

	err := u.Do(context.Background(), func(ctx context.Context) error {
		return GetDB(ctx, db).Exec("INSERT INTO payment_bridge(id,name) VALUES (?,?)", uuid.New().String(), "ccip").Error
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to commit transaction")
}
