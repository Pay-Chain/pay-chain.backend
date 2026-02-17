package repositories

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
)

func TestWalletRepository_CRUDAndPrimary(t *testing.T) {
	db := newTestDB(t)
	createChainTables(t, db)
	createWalletTable(t, db)
	repo := NewWalletRepository(db)
	ctx := context.Background()

	chainID := uuid.New()
	seedChain(t, db, chainID.String(), "8453", "Base", "EVM", true)
	userID := uuid.New()

	w1 := &entities.Wallet{ID: uuid.New(), UserID: &userID, ChainID: chainID, Address: "0xabc", IsPrimary: true, CreatedAt: time.Now()}
	w2 := &entities.Wallet{ID: uuid.New(), UserID: &userID, ChainID: chainID, Address: "0xdef", IsPrimary: false, CreatedAt: time.Now()}
	merchantID := uuid.New()
	w3 := &entities.Wallet{ID: uuid.New(), MerchantID: &merchantID, ChainID: chainID, Address: "0xmerchant", IsPrimary: false, CreatedAt: time.Now()}
	require.NoError(t, repo.Create(ctx, w1))
	require.NoError(t, repo.Create(ctx, w2))
	require.NoError(t, repo.Create(ctx, w3))

	got, err := repo.GetByID(ctx, w1.ID)
	require.NoError(t, err)
	require.Equal(t, w1.ID, got.ID)

	list, err := repo.GetByUserID(ctx, userID)
	require.NoError(t, err)
	require.Len(t, list, 2)

	byAddr, err := repo.GetByAddress(ctx, chainID, "0xabc")
	require.NoError(t, err)
	require.Equal(t, w1.ID, byAddr.ID)

	require.NoError(t, repo.SetPrimary(ctx, userID, w2.ID))

	require.NoError(t, repo.SoftDelete(ctx, w1.ID))
	_, err = repo.GetByID(ctx, w1.ID)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestWalletRepository_NotFoundBranches(t *testing.T) {
	db := newTestDB(t)
	createChainTables(t, db)
	createWalletTable(t, db)
	repo := NewWalletRepository(db)
	ctx := context.Background()

	id := uuid.New()
	_, err := repo.GetByID(ctx, id)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	_, err = repo.GetByAddress(ctx, uuid.New(), "0xmissing")
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.SetPrimary(ctx, uuid.New(), uuid.New())
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.SoftDelete(ctx, id)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestWalletRepository_SetPrimary_DBErrorBranch(t *testing.T) {
	db := newTestDB(t)
	// intentionally do not create wallets table
	repo := NewWalletRepository(db)
	ctx := context.Background()

	err := repo.SetPrimary(ctx, uuid.New(), uuid.New())
	require.Error(t, err)
}

func TestWalletRepository_SetPrimary_SecondUpdateErrorBranch(t *testing.T) {
	db := newTestDB(t)
	createChainTables(t, db)
	createWalletTable(t, db)
	repo := NewWalletRepository(db)
	ctx := context.Background()

	chainID := uuid.New()
	seedChain(t, db, chainID.String(), "8453", "Base", "EVM", true)
	userID := uuid.New()
	w1 := &entities.Wallet{ID: uuid.New(), UserID: &userID, ChainID: chainID, Address: "0xabc", IsPrimary: true, CreatedAt: time.Now()}
	w2 := &entities.Wallet{ID: uuid.New(), UserID: &userID, ChainID: chainID, Address: "0xdef", IsPrimary: false, CreatedAt: time.Now()}
	require.NoError(t, repo.Create(ctx, w1))
	require.NoError(t, repo.Create(ctx, w2))

	callbackName := "test:setprimary_fail_on_true"
	db.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		// Fail only on the second statement (set is_primary=true),
		// letting the first unset-all update pass.
		if tx.Statement != nil && tx.Statement.Dest != nil {
			if setMap, ok := tx.Statement.Dest.(map[string]interface{}); ok {
				if v, exists := setMap["is_primary"]; exists {
					if b, ok := v.(bool); ok && b {
						tx.AddError(errors.New("set primary failed"))
					}
				}
			}
		}
	})
	defer db.Callback().Update().Remove(callbackName)

	err := repo.SetPrimary(ctx, userID, w2.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "set primary failed")
}

func TestWalletRepository_OtherDBErrorBranches(t *testing.T) {
	db := newTestDB(t)
	// intentionally skip table creation
	repo := NewWalletRepository(db)
	ctx := context.Background()

	err := repo.Create(ctx, &entities.Wallet{
		ID:      uuid.New(),
		UserID:  ptrUUID(uuid.New()),
		ChainID: uuid.New(),
		Address: "0xabc",
	})
	require.Error(t, err)

	_, err = repo.GetByID(ctx, uuid.New())
	require.Error(t, err)

	_, err = repo.GetByAddress(ctx, uuid.New(), "0xabc")
	require.Error(t, err)

	_, err = repo.GetByUserID(ctx, uuid.New())
	require.Error(t, err)

	err = repo.SoftDelete(ctx, uuid.New())
	require.Error(t, err)
}

func ptrUUID(v uuid.UUID) *uuid.UUID {
	return &v
}
