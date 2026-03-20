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

func TestPaymentQuoteRepository_BasicFlow(t *testing.T) {
	db := newTestDB(t)
	createPartnerFlowTables(t, db)
	repo := NewPaymentQuoteRepository(db)
	ctx := context.Background()

	quote := &domainentities.PaymentQuote{
		ID:                    uuid.New(),
		MerchantID:            uuid.New(),
		InvoiceCurrency:       "IDRX",
		InvoiceAmount:         "50000000000",
		SelectedChainID:       "eip155:8453",
		SelectedTokenAddress:  "0x8335",
		SelectedTokenSymbol:   "USDC",
		SelectedTokenDecimals: 6,
		QuotedAmount:          "2950000",
		QuoteRate:             "0.000059",
		PriceSource:           "uniswap",
		Route:                 "IDRX->USDC",
		SlippageBps:           100,
		RateTimestamp:         time.Now(),
		ExpiresAt:             time.Now().Add(time.Hour),
		Status:                domainentities.PaymentQuoteStatusActive,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
	require.NoError(t, repo.Create(ctx, quote))

	got, err := repo.GetByID(ctx, quote.ID)
	require.NoError(t, err)
	require.Equal(t, quote.ID, got.ID)
	require.Equal(t, domainentities.PaymentQuoteStatusActive, got.Status)

	require.NoError(t, repo.MarkUsed(ctx, quote.ID))
	updated, err := repo.GetByID(ctx, quote.ID)
	require.NoError(t, err)
	require.Equal(t, domainentities.PaymentQuoteStatusUsed, updated.Status)
	require.NotNil(t, updated.UsedAt)
}

func TestPaymentQuoteRepository_ExpiryFlow(t *testing.T) {
	db := newTestDB(t)
	createPartnerFlowTables(t, db)
	repo := NewPaymentQuoteRepository(db)
	ctx := context.Background()

	activeExpired := &domainentities.PaymentQuote{
		ID:                    uuid.New(),
		MerchantID:            uuid.New(),
		InvoiceCurrency:       "IDRX",
		InvoiceAmount:         "1",
		SelectedChainID:       "eip155:8453",
		SelectedTokenAddress:  "0x1",
		SelectedTokenSymbol:   "USDC",
		SelectedTokenDecimals: 6,
		QuotedAmount:          "1",
		QuoteRate:             "1",
		PriceSource:           "src",
		Route:                 "A->B",
		SlippageBps:           0,
		RateTimestamp:         time.Now().Add(-2 * time.Hour),
		ExpiresAt:             time.Now().Add(-time.Minute),
		Status:                domainentities.PaymentQuoteStatusActive,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
	require.NoError(t, repo.Create(ctx, activeExpired))

	expired, err := repo.GetExpiredActive(ctx, 10)
	require.NoError(t, err)
	require.Len(t, expired, 1)
	require.Equal(t, activeExpired.ID, expired[0].ID)

	require.NoError(t, repo.ExpireQuotes(ctx, []uuid.UUID{activeExpired.ID}))
	got, err := repo.GetByID(ctx, activeExpired.ID)
	require.NoError(t, err)
	require.Equal(t, domainentities.PaymentQuoteStatusExpired, got.Status)
	require.NoError(t, repo.ExpireQuotes(ctx, nil))
}

func TestPaymentQuoteRepository_NotFoundAndDBErrors(t *testing.T) {
	db := newTestDB(t)
	createPartnerFlowTables(t, db)
	repo := NewPaymentQuoteRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
	require.ErrorIs(t, repo.MarkUsed(ctx, uuid.New()), domainerrors.ErrNotFound)
	require.ErrorIs(t, repo.UpdateStatus(ctx, uuid.New(), domainentities.PaymentQuoteStatusCancelled), domainerrors.ErrNotFound)

	db2 := newTestDB(t)
	repo2 := NewPaymentQuoteRepository(db2)
	_, err = repo2.GetExpiredActive(ctx, 10)
	require.Error(t, err)
}

func TestPartnerPaymentSessionRepository_BasicFlow(t *testing.T) {
	db := newTestDB(t)
	createPartnerFlowTables(t, db)
	repo := NewPartnerPaymentSessionRepository(db)
	ctx := context.Background()

	session := &domainentities.PartnerPaymentSession{
		ID:                    uuid.New(),
		MerchantID:            uuid.New(),
		PaymentRequestID:      partnerPtrUUID(uuid.New()),
		InvoiceCurrency:       "IDRX",
		InvoiceAmount:         "50000000000",
		SelectedChainID:       "eip155:8453",
		SelectedTokenAddress:  "0x8335",
		SelectedTokenSymbol:   "USDC",
		SelectedTokenDecimals: 6,
		DestChain:             "eip155:8453",
		DestToken:             "0x8335",
		DestWallet:            "0xmerchant",
		PaymentAmount:         "2950000",
		PaymentAmountDecimals: 6,
		Status:                domainentities.PartnerPaymentSessionStatusPending,
		PaymentCode:           "eyJ",
		PaymentURL:            "https://pay/payment/1",
		InstructionTo:         "0xreceiver",
		InstructionValue:      "0x0",
		InstructionDataHex:    "0xabc",
		ExpiresAt:             time.Now().Add(time.Hour),
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
	require.NoError(t, repo.Create(ctx, session))

	got, err := repo.GetByID(ctx, session.ID)
	require.NoError(t, err)
	require.Equal(t, session.ID, got.ID)
	require.Equal(t, domainentities.PartnerPaymentSessionStatusPending, got.Status)

	byPaymentRequest, err := repo.GetByPaymentRequestID(ctx, *session.PaymentRequestID)
	require.NoError(t, err)
	require.Equal(t, session.ID, byPaymentRequest.ID)

	require.NoError(t, repo.MarkCompleted(ctx, session.ID, "0xpaid"))
	updated, err := repo.GetByID(ctx, session.ID)
	require.NoError(t, err)
	require.Equal(t, domainentities.PartnerPaymentSessionStatusCompleted, updated.Status)
	require.NotNil(t, updated.CompletedAt)
	require.NotNil(t, updated.PaidTxHash)
	require.Equal(t, "0xpaid", *updated.PaidTxHash)
}

func TestPartnerPaymentSessionRepository_ExpiryFlow(t *testing.T) {
	db := newTestDB(t)
	createPartnerFlowTables(t, db)
	repo := NewPartnerPaymentSessionRepository(db)
	ctx := context.Background()

	session := &domainentities.PartnerPaymentSession{
		ID:                    uuid.New(),
		MerchantID:            uuid.New(),
		InvoiceCurrency:       "IDRX",
		InvoiceAmount:         "1",
		SelectedChainID:       "eip155:8453",
		SelectedTokenAddress:  "0x8335",
		SelectedTokenSymbol:   "USDC",
		SelectedTokenDecimals: 6,
		DestChain:             "eip155:8453",
		DestToken:             "0x8335",
		DestWallet:            "0xmerchant",
		PaymentAmount:         "1",
		PaymentAmountDecimals: 6,
		Status:                domainentities.PartnerPaymentSessionStatusPending,
		PaymentCode:           "eyJ",
		PaymentURL:            "https://pay/payment/1",
		ExpiresAt:             time.Now().Add(-time.Minute),
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
	require.NoError(t, repo.Create(ctx, session))

	expired, err := repo.GetExpiredPending(ctx, 10)
	require.NoError(t, err)
	require.Len(t, expired, 1)
	require.Equal(t, session.ID, expired[0].ID)

	require.NoError(t, repo.ExpireSessions(ctx, []uuid.UUID{session.ID}))
	got, err := repo.GetByID(ctx, session.ID)
	require.NoError(t, err)
	require.Equal(t, domainentities.PartnerPaymentSessionStatusExpired, got.Status)
	require.NoError(t, repo.ExpireSessions(ctx, nil))
}

func TestPartnerPaymentSessionRepository_NotFoundAndDBErrors(t *testing.T) {
	db := newTestDB(t)
	createPartnerFlowTables(t, db)
	repo := NewPartnerPaymentSessionRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
	_, err = repo.GetByPaymentRequestID(ctx, uuid.New())
	require.ErrorIs(t, err, domainerrors.ErrNotFound)
	require.ErrorIs(t, repo.MarkCompleted(ctx, uuid.New(), "0xpaid"), domainerrors.ErrNotFound)
	require.ErrorIs(t, repo.UpdateStatus(ctx, uuid.New(), domainentities.PartnerPaymentSessionStatusCancelled), domainerrors.ErrNotFound)

	db2 := newTestDB(t)
	repo2 := NewPartnerPaymentSessionRepository(db2)
	_, err = repo2.GetExpiredPending(ctx, 10)
	require.Error(t, err)
}

func partnerPtrUUID(v uuid.UUID) *uuid.UUID {
	return &v
}
