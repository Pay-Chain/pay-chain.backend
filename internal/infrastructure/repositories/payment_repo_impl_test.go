package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/volatiletech/null/v8"
	"gorm.io/gorm"
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

func TestPaymentRepository_List_DBErrorBranches(t *testing.T) {
	db := newTestDB(t)
	// Intentionally skip creating payment table.
	repo := NewPaymentRepository(db)
	ctx := context.Background()

	_, _, err := repo.GetByUserID(ctx, uuid.New(), 10, 0)
	require.Error(t, err)

	_, _, err = repo.GetByMerchantID(ctx, uuid.New(), 10, 0)
	require.Error(t, err)
}

func TestPaymentRepository_DBErrorBranches_SingleAndUpdates(t *testing.T) {
	db := newTestDB(t)
	// Intentionally skip creating required tables.
	repo := NewPaymentRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	require.Error(t, err)

	err = repo.UpdateStatus(ctx, uuid.New(), entities.PaymentStatusCompleted)
	require.Error(t, err)

	err = repo.UpdateDestTxHash(ctx, uuid.New(), "0xhash")
	require.Error(t, err)

	err = repo.MarkRefunded(ctx, uuid.New())
	require.Error(t, err)

	senderID := uuid.New()
	sourceTokenID := uuid.New()
	destTokenID := uuid.New()
	err = repo.Create(ctx, &entities.Payment{
		ID:            uuid.New(),
		SenderID:      &senderID,
		SourceChainID: uuid.New(),
		DestChainID:   uuid.New(),
		SourceTokenID: &sourceTokenID,
		DestTokenID:   &destTokenID,
		SourceAmount:  "1",
		FeeAmount:     "0",
		TotalCharged:  "1",
		Status:        entities.PaymentStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	})
	require.Error(t, err)
}

func TestPaymentRepository_Create_WithTxAndDestAmountBranch(t *testing.T) {
	db := newTestDB(t)
	createPaymentTables(t, db)
	createChainTables(t, db)
	createTokenTable(t, db)
	repo := NewPaymentRepository(db)

	sourceChainID := uuid.New()
	destChainID := uuid.New()
	sourceTokenID := uuid.New()
	destTokenID := uuid.New()
	senderID := uuid.New()

	mustExec(t, db, `INSERT INTO chains(id,chain_id,name,type,is_active,created_at,updated_at) VALUES (?,?,?,?,?,?,?)`,
		sourceChainID.String(), "8453", "Base", "EVM", true, time.Now(), time.Now())
	mustExec(t, db, `INSERT INTO chains(id,chain_id,name,type,is_active,created_at,updated_at) VALUES (?,?,?,?,?,?,?)`,
		destChainID.String(), "42161", "Arbitrum", "EVM", true, time.Now(), time.Now())
	mustExec(t, db, `INSERT INTO tokens(id,chain_id,symbol,name,decimals,address,type,is_active,is_native,is_stablecoin,created_at,updated_at)
	VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, sourceTokenID.String(), sourceChainID.String(), "IDRX", "IDRX", 6, "0xsource", "ERC20", true, false, true, time.Now(), time.Now())
	mustExec(t, db, `INSERT INTO tokens(id,chain_id,symbol,name,decimals,address,type,is_active,is_native,is_stablecoin,created_at,updated_at)
	VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, destTokenID.String(), destChainID.String(), "USDC", "USDC", 6, "0xdest", "ERC20", true, false, true, time.Now(), time.Now())

	tx := db.Begin()
	ctx := context.WithValue(context.Background(), txKey, tx)

	p := &entities.Payment{
		ID:            uuid.New(),
		SenderID:      &senderID,
		SourceChainID: sourceChainID,
		DestChainID:   destChainID,
		SourceTokenID: &sourceTokenID,
		DestTokenID:   &destTokenID,
		SourceAmount:  "1000",
		DestAmount:    null.StringFrom("947"),
		FeeAmount:     "53",
		TotalCharged:  "1053",
		SenderAddress: "0xsender",
		Status:        entities.PaymentStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	require.NoError(t, repo.Create(ctx, p))
	require.NoError(t, tx.Commit().Error)

	got, err := repo.GetByID(context.Background(), p.ID)
	require.NoError(t, err)
	require.True(t, got.DestAmount.Valid)
	require.Equal(t, "947", got.DestAmount.String)
}

func TestPaymentRepository_List_FindErrorBranches(t *testing.T) {
	db := newTestDB(t)
	createPaymentTables(t, db)
	createChainTables(t, db)
	createTokenTable(t, db)
	repo := NewPaymentRepository(db)
	ctx := context.Background()

	cbName := "test:payment_repo_fail_find"
	queryCount := 0
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register(cbName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "payments" {
			queryCount++
			if queryCount > 1 {
				tx.AddError(gorm.ErrInvalidDB)
			}
		}
	}))
	t.Cleanup(func() {
		_ = db.Callback().Query().Remove(cbName)
	})

	_, _, err := repo.GetByUserID(ctx, uuid.New(), 10, 0)
	require.Error(t, err)

	queryCount = 0
	_, _, err = repo.GetByMerchantID(ctx, uuid.New(), 10, 0)
	require.Error(t, err)
}
