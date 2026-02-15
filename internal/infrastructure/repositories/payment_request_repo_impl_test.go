package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
)

func TestPaymentRequestRepository_FullFlow(t *testing.T) {
	db := newTestDB(t)
	createPaymentRequestTables(t, db)
	repo := NewPaymentRequestRepository(db)
	ctx := context.Background()

	id := uuid.New()
	merchantID := uuid.New()
	chainID := uuid.New()
	tokenID := uuid.New()
	expires := time.Now().Add(time.Hour)

	err := repo.Create(ctx, &entities.PaymentRequest{
		ID:            id,
		MerchantID:    merchantID,
		ChainID:       chainID,
		TokenID:       tokenID,
		WalletAddress: "0xwallet",
		TokenAddress:  "0xtoken",
		Amount:        "10",
		Decimals:      6,
		Description:   "test",
		Status:        entities.PaymentRequestStatusPending,
		ExpiresAt:     expires,
	})
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, merchantID, got.MerchantID)

	items, total, err := repo.GetByMerchantID(ctx, merchantID, 10, 0)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, items, 1)

	require.NoError(t, repo.UpdateStatus(ctx, id, entities.PaymentRequestStatusCancelled))
	require.NoError(t, repo.UpdateTxHash(ctx, id, "0xtx", "0xpayer"))
	require.NoError(t, repo.MarkCompleted(ctx, id, "0xtx2"))
}

func TestPaymentRequestRepository_ExpiredAndBulkExpire(t *testing.T) {
	db := newTestDB(t)
	createPaymentRequestTables(t, db)
	repo := NewPaymentRequestRepository(db)
	ctx := context.Background()

	id := uuid.New()
	mustExec(t, db, `INSERT INTO payment_requests(
		id,merchant_id,chain_id,token_id,wallet_address,token_address,amount,decimals,description,status,expires_at,created_at,updated_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id.String(), uuid.NewString(), uuid.NewString(), uuid.NewString(), "0xw", "0xt", "1", 6, "",
		string(entities.PaymentRequestStatusPending), time.Now().Add(-time.Hour), time.Now(), time.Now())

	expired, err := repo.GetExpiredPending(ctx, 10)
	require.NoError(t, err)
	require.Len(t, expired, 1)

	require.NoError(t, repo.ExpireRequests(ctx, []uuid.UUID{id}))
	require.NoError(t, repo.ExpireRequests(ctx, nil))
}

func TestBackgroundJobRepository_FullFlow(t *testing.T) {
	db := newTestDB(t)
	createPaymentRequestTables(t, db)
	repo := NewBackgroundJobRepository(db)
	ctx := context.Background()

	id := uuid.New()
	err := repo.Create(ctx, &entities.BackgroundJob{
		ID:          id,
		JobType:     "EXPIRE_PAYMENT_REQUESTS",
		Payload:     map[string]any{"limit": 100},
		Status:      entities.JobStatusPending,
		MaxAttempts: 3,
		ScheduledAt: time.Now().Add(-time.Minute),
	})
	require.NoError(t, err)

	pending, err := repo.GetPending(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pending, 1)

	require.NoError(t, repo.MarkProcessing(ctx, id))
	require.NoError(t, repo.MarkFailed(ctx, id, "boom"))
	require.NoError(t, repo.MarkCompleted(ctx, id))
}
