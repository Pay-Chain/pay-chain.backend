package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
	domainerrors "pay-chain.backend/internal/domain/errors"
)

func TestPaymentRepository_BasicFlow(t *testing.T) {
	db := newTestDB(t)
	createPaymentTables(t, db)
	createChainTables(t, db)
	createTokenTable(t, db)
	repo := NewPaymentRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	merchantID := uuid.New()
	sourceChainID := uuid.New()
	destChainID := uuid.New()
	sourceTokenID := uuid.New()
	destTokenID := uuid.New()

	mustExec(t, db, `INSERT INTO chains(id,chain_id,name,type,is_active,created_at,updated_at) VALUES (?,?,?,?,?,?,?)`,
		sourceChainID.String(), "8453", "Base", "EVM", true, time.Now(), time.Now())
	mustExec(t, db, `INSERT INTO chains(id,chain_id,name,type,is_active,created_at,updated_at) VALUES (?,?,?,?,?,?,?)`,
		destChainID.String(), "42161", "Arbitrum", "EVM", true, time.Now(), time.Now())
	mustExec(t, db, `INSERT INTO tokens(id,chain_id,symbol,name,decimals,address,type,is_active,is_native,is_stablecoin,created_at,updated_at)
	VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, sourceTokenID.String(), sourceChainID.String(), "IDRX", "IDRX", 6, "0xsource", "ERC20", true, false, true, time.Now(), time.Now())
	mustExec(t, db, `INSERT INTO tokens(id,chain_id,symbol,name,decimals,address,type,is_active,is_native,is_stablecoin,created_at,updated_at)
	VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, destTokenID.String(), destChainID.String(), "USDC", "USDC", 6, "0xdest", "ERC20", true, false, true, time.Now(), time.Now())

	p := &entities.Payment{
		ID:            uuid.New(),
		SenderID:      &userID,
		MerchantID:    &merchantID,
		SourceChainID: sourceChainID,
		DestChainID:   destChainID,
		SourceTokenID: &sourceTokenID,
		DestTokenID:   &destTokenID,
		SourceAmount:  "100",
		FeeAmount:     "1",
		TotalCharged:  "101",
		SenderAddress: "0xsender",
		Status:        entities.PaymentStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	require.NoError(t, repo.Create(ctx, p))

	got, err := repo.GetByID(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, p.ID, got.ID)
	require.Equal(t, "0xsender", got.SenderAddress)

	byUser, totalUser, err := repo.GetByUserID(ctx, userID, 10, 0)
	require.NoError(t, err)
	require.Equal(t, 1, totalUser)
	require.Len(t, byUser, 1)

	byMerchant, totalMerchant, err := repo.GetByMerchantID(ctx, merchantID, 10, 0)
	require.NoError(t, err)
	require.Equal(t, 1, totalMerchant)
	require.Len(t, byMerchant, 1)

	require.NoError(t, repo.UpdateStatus(ctx, p.ID, entities.PaymentStatusProcessing))
	require.NoError(t, repo.UpdateDestTxHash(ctx, p.ID, "0xdtx"))
	require.NoError(t, repo.MarkRefunded(ctx, p.ID))

	updated, err := repo.GetByID(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, entities.PaymentStatusRefunded, updated.Status)
}

func TestPaymentRepository_NotFoundBranches(t *testing.T) {
	db := newTestDB(t)
	createPaymentTables(t, db)
	createChainTables(t, db)
	createTokenTable(t, db)
	repo := NewPaymentRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	require.ErrorIs(t, err, domainerrors.ErrNotFound)

	require.ErrorIs(t, repo.UpdateStatus(ctx, uuid.New(), entities.PaymentStatusCompleted), domainerrors.ErrNotFound)
	require.ErrorIs(t, repo.MarkRefunded(ctx, uuid.New()), domainerrors.ErrNotFound)
}
