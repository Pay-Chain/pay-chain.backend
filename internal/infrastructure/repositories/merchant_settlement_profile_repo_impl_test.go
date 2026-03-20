package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	domainentities "payment-kita.backend/internal/domain/entities"
	domainerrors "payment-kita.backend/internal/domain/errors"
)

func TestMerchantSettlementProfileRepository_UpsertAndGet(t *testing.T) {
	db := newTestDB(t)
	createMerchantTable(t, db)
	createMerchantSettlementProfileTable(t, db)

	ctx := context.Background()
	merchantID := uuid.New()
	now := time.Now().UTC()
	mustExec(t, db, `INSERT INTO merchants (id, user_id, business_name, business_email, merchant_type, status, documents, fee_discount_percent, webhook_metadata, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, merchantID.String(), uuid.NewString(), "Merchant", "merchant@example.com", "PARTNER", "ACTIVE", "{}", "0", "{}", "{}", now, now)

	repo := NewMerchantSettlementProfileRepository(db)
	profile := &domainentities.MerchantSettlementProfile{
		MerchantID:        merchantID,
		InvoiceCurrency:   "IDRX",
		DestChain:         "eip155:8453",
		DestToken:         "0xidrxtoken",
		DestWallet:        "0xmerchantwallet",
		BridgeTokenSymbol: "USDC",
	}
	require.NoError(t, repo.Upsert(ctx, profile))

	got, err := repo.GetByMerchantID(ctx, merchantID)
	require.NoError(t, err)
	require.Equal(t, "IDRX", got.InvoiceCurrency)
	require.Equal(t, "eip155:8453", got.DestChain)

	profile.InvoiceCurrency = "USDT"
	profile.DestToken = "0xusdttoken"
	require.NoError(t, repo.Upsert(ctx, profile))

	got, err = repo.GetByMerchantID(ctx, merchantID)
	require.NoError(t, err)
	require.Equal(t, "USDT", got.InvoiceCurrency)
	require.Equal(t, "0xusdttoken", got.DestToken)
}

func TestMerchantSettlementProfileRepository_NotFound(t *testing.T) {
	db := newTestDB(t)
	createMerchantSettlementProfileTable(t, db)
	repo := NewMerchantSettlementProfileRepository(db)
	_, err := repo.GetByMerchantID(context.Background(), uuid.New())
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
}

func TestMerchantSettlementProfileRepository_ListMissingAndHasProfiles(t *testing.T) {
	db := newTestDB(t)
	createMerchantTable(t, db)
	createMerchantSettlementProfileTable(t, db)

	ctx := context.Background()
	now := time.Now().UTC()
	merchantA := uuid.New()
	merchantB := uuid.New()
	mustExec(t, db, `INSERT INTO merchants (id, user_id, business_name, business_email, merchant_type, status, documents, fee_discount_percent, webhook_metadata, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, merchantA.String(), uuid.NewString(), "A", "a@example.com", "PARTNER", "ACTIVE", "{}", "0", "{}", "{}", now, now)
	mustExec(t, db, `INSERT INTO merchants (id, user_id, business_name, business_email, merchant_type, status, documents, fee_discount_percent, webhook_metadata, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, merchantB.String(), uuid.NewString(), "B", "b@example.com", "PARTNER", "ACTIVE", "{}", "0", "{}", "{}", now, now)

	repo := NewMerchantSettlementProfileRepository(db)
	require.NoError(t, repo.Upsert(ctx, &domainentities.MerchantSettlementProfile{
		MerchantID:        merchantA,
		InvoiceCurrency:   "IDRX",
		DestChain:         "eip155:8453",
		DestToken:         "0xidrxtoken",
		DestWallet:        "0xwallet",
		BridgeTokenSymbol: "USDC",
	}))

	hasMap, err := repo.HasProfilesByMerchantIDs(ctx, []uuid.UUID{merchantA, merchantB})
	require.NoError(t, err)
	require.True(t, hasMap[merchantA])
	require.False(t, hasMap[merchantB])

	missing, err := repo.ListMissingMerchantIDs(ctx)
	require.NoError(t, err)
	require.Contains(t, missing, merchantB)
	require.NotContains(t, missing, merchantA)
}
