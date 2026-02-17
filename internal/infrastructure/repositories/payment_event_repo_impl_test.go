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

func TestPaymentEventRepository_CreateAndReads(t *testing.T) {
	db := newTestDB(t)
	createPaymentTables(t, db)
	repo := NewPaymentEventRepository(db)
	ctx := context.Background()

	paymentID := uuid.New()
	eventID := uuid.New()
	chainID := uuid.New()
	now := time.Now()

	mustExec(t, db, `INSERT INTO payments(
		id,sender_id,source_chain_id,dest_chain_id,source_token_id,dest_token_id,source_amount,fee_amount,total_charged,status,created_at,updated_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		paymentID.String(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(),
		"1", "0", "1", "PENDING", now, now)

	err := repo.Create(ctx, &entities.PaymentEvent{
		ID:          eventID,
		PaymentID:   paymentID,
		EventType:   entities.PaymentEventTypeCreated,
		TxHash:      "0xtx",
		ChainID:     &chainID,
		BlockNumber: 12,
		CreatedAt:   now,
	})
	require.NoError(t, err)

	events, err := repo.GetByPaymentID(ctx, paymentID)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, entities.PaymentEventTypeCreated, events[0].EventType)

	latest, err := repo.GetLatestByPaymentID(ctx, paymentID)
	require.NoError(t, err)
	require.Equal(t, eventID, latest.ID)
}

func TestPaymentEventRepository_NotFoundLatest(t *testing.T) {
	db := newTestDB(t)
	createPaymentTables(t, db)
	repo := NewPaymentEventRepository(db)
	ctx := context.Background()

	_, err := repo.GetLatestByPaymentID(ctx, uuid.New())
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestPaymentEventRepository_DBErrorBranches(t *testing.T) {
	db := newTestDB(t)
	// intentionally skip table creation
	repo := NewPaymentEventRepository(db)
	ctx := context.Background()

	err := repo.Create(ctx, &entities.PaymentEvent{
		ID:        uuid.New(),
		PaymentID: uuid.New(),
		EventType: entities.PaymentEventTypeCreated,
		TxHash:    "0x",
	})
	require.Error(t, err)

	_, err = repo.GetByPaymentID(ctx, uuid.New())
	require.Error(t, err)

	_, err = repo.GetLatestByPaymentID(ctx, uuid.New())
	require.Error(t, err)
}

func TestPaymentEventRepository_ResolveLegacyChainValue(t *testing.T) {
	repo := NewPaymentEventRepository(nil)

	require.Equal(t, "UNKNOWN", repo.resolveLegacyChainValue(nil))

	chainID := uuid.MustParse("12345678-1234-1234-1234-1234567890ab")
	require.Equal(t, "chain-12345678", repo.resolveLegacyChainValue(&chainID))
}

func TestPaymentEventRepository_Create_ZeroCreatedAtAndNilChain(t *testing.T) {
	db := newTestDB(t)
	createPaymentTables(t, db)
	repo := NewPaymentEventRepository(db)
	ctx := context.Background()

	paymentID := uuid.New()
	now := time.Now()
	mustExec(t, db, `INSERT INTO payments(
		id,sender_id,source_chain_id,dest_chain_id,source_token_id,dest_token_id,source_amount,fee_amount,total_charged,status,created_at,updated_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		paymentID.String(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(),
		"1", "0", "1", "PENDING", now, now)

	err := repo.Create(ctx, &entities.PaymentEvent{
		ID:        uuid.New(),
		PaymentID: paymentID,
		EventType: entities.PaymentEventTypeCreated,
		TxHash:    "0xtx",
		// zero CreatedAt + nil ChainID to hit default branches
	})
	require.NoError(t, err)

	latest, err := repo.GetLatestByPaymentID(ctx, paymentID)
	require.NoError(t, err)
	require.False(t, latest.CreatedAt.IsZero())
}

func TestPaymentEventRepository_Create_WithinTxContext(t *testing.T) {
	db := newTestDB(t)
	createPaymentTables(t, db)
	repo := NewPaymentEventRepository(db)

	paymentID := uuid.New()
	now := time.Now()
	mustExec(t, db, `INSERT INTO payments(
		id,sender_id,source_chain_id,dest_chain_id,source_token_id,dest_token_id,source_amount,fee_amount,total_charged,status,created_at,updated_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		paymentID.String(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(),
		"1", "0", "1", "PENDING", now, now)

	tx := db.Begin()
	ctx := context.WithValue(context.Background(), txKey, tx)
	err := repo.Create(ctx, &entities.PaymentEvent{
		ID:        uuid.New(),
		PaymentID: paymentID,
		EventType: entities.PaymentEventTypeCreated,
		TxHash:    "0xabc",
	})
	require.NoError(t, err)
	require.NoError(t, tx.Commit().Error)
}
