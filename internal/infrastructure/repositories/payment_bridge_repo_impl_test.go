package repositories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
	"pay-chain.backend/pkg/utils"
)

func TestPaymentBridgeRepository_CRUDAndList(t *testing.T) {
	db := newTestDB(t)
	createPaymentBridgeTable(t, db)
	repo := NewPaymentBridgeRepository(db)
	ctx := context.Background()

	bridge := &entities.PaymentBridge{ID: uuid.New(), Name: "CCIP"}
	require.NoError(t, repo.Create(ctx, bridge))

	gotByName, err := repo.GetByName(ctx, "ccip")
	require.NoError(t, err)
	require.Equal(t, "CCIP", gotByName.Name)

	gotByID, err := repo.GetByID(ctx, gotByName.ID)
	require.NoError(t, err)
	require.Equal(t, gotByName.ID, gotByID.ID)

	items, total, err := repo.List(ctx, utils.PaginationParams{Page: 1, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)

	gotByID.Name = "Hyperbridge"
	require.NoError(t, repo.Update(ctx, gotByID))

	require.NoError(t, repo.Delete(ctx, gotByID.ID))
	_, err = repo.GetByID(ctx, gotByID.ID)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestPaymentBridgeRepository_NotFoundBranches(t *testing.T) {
	db := newTestDB(t)
	createPaymentBridgeTable(t, db)
	repo := NewPaymentBridgeRepository(db)
	ctx := context.Background()

	id := uuid.New()
	_, err := repo.GetByID(ctx, id)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	_, err = repo.GetByName(ctx, "missing")
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.Update(ctx, &entities.PaymentBridge{ID: id, Name: "x"})
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	err = repo.Delete(ctx, id)
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestPaymentBridgeRepository_Create_AssignsIDWhenNil(t *testing.T) {
	db := newTestDB(t)
	createPaymentBridgeTable(t, db)
	repo := NewPaymentBridgeRepository(db)
	ctx := context.Background()

	bridge := &entities.PaymentBridge{Name: "LayerZero"}
	require.NoError(t, repo.Create(ctx, bridge))
	require.NotEqual(t, uuid.Nil, bridge.ID)
}

func TestPaymentBridgeRepository_DBErrorBranches(t *testing.T) {
	db := newTestDB(t)
	// intentionally skip table creation
	repo := NewPaymentBridgeRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	require.Error(t, err)

	_, err = repo.GetByName(ctx, "ccip")
	require.Error(t, err)

	_, _, err = repo.List(ctx, utils.PaginationParams{Page: 1, Limit: 10})
	require.Error(t, err)

	err = repo.Update(ctx, &entities.PaymentBridge{ID: uuid.New(), Name: "x"})
	require.Error(t, err)

	err = repo.Delete(ctx, uuid.New())
	require.Error(t, err)
}

func TestPaymentBridgeRepository_List_FindErrorAfterCount(t *testing.T) {
	db := newTestDB(t)
	createPaymentBridgeTable(t, db)
	repo := NewPaymentBridgeRepository(db)
	ctx := context.Background()

	cbName := "test:payment_bridge_list_find_error"
	queryCount := 0
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register(cbName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "payment_bridge" {
			queryCount++
			if queryCount > 1 {
				tx.AddError(gorm.ErrInvalidDB)
			}
		}
	}))
	t.Cleanup(func() {
		_ = db.Callback().Query().Remove(cbName)
	})

	_, _, err := repo.List(ctx, utils.PaginationParams{Page: 1, Limit: 10})
	require.Error(t, err)
}
