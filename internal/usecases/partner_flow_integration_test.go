package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	domainentities "payment-kita.backend/internal/domain/entities"
	"payment-kita.backend/internal/domain/services"
	infrarepos "payment-kita.backend/internal/infrastructure/repositories"
)

func TestPartnerFlow_QuoteSessionHostedReadResolveWebhookSync_Integration(t *testing.T) {
	db := newPartnerFlowIntegrationDB(t)
	createPartnerFlowIntegrationTables(t, db)

	ctx := context.Background()
	merchantID := uuid.New()
	chainID := uuid.New()
	idrxID := uuid.New()
	usdcID := uuid.New()

	mustExecIntegration(t, db, `INSERT INTO chains (
		id, chain_id, name, type, rpc_url, explorer_url, currency_symbol, image_url, is_active, state_machine_id, ccip_chain_selector, stargate_eid, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		chainID.String(), "8453", "Base", "EVM", "https://rpc.base.example", "https://basescan.org", "ETH", "", true, "", "", 0, time.Now().UTC(), time.Now().UTC(),
	)
	mustExecIntegration(t, db, `INSERT INTO tokens (
		id, chain_id, symbol, name, decimals, address, type, logo_url, is_active, is_native, is_stablecoin, min_amount, max_amount, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		idrxID.String(), chainID.String(), "IDRX", "IDRX", 2, "0xidrxtoken", "ERC20", "", true, false, false, "0", nil, time.Now().UTC(), time.Now().UTC(),
	)
	mustExecIntegration(t, db, `INSERT INTO tokens (
		id, chain_id, symbol, name, decimals, address, type, logo_url, is_active, is_native, is_stablecoin, min_amount, max_amount, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		usdcID.String(), chainID.String(), "USDC", "USDC", 6, "0xusdctoken", "ERC20", "", true, false, true, "0", nil, time.Now().UTC(), time.Now().UTC(),
	)
	mustExecIntegration(t, db, `INSERT INTO smart_contracts (
		id, name, chain_id, address, abi, type, version, deployer_address, token0_address, token1_address, fee_tier, hook_address, start_block, metadata, is_active, destination_map, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), "Gateway", chainID.String(), "0xgateway0000000000000000000000000000000000", "[]", "GATEWAY", "1.0.0", "", "", "", 0, "", 0, "{}", true, "{}", time.Now().UTC(), time.Now().UTC(),
	)

	chainRepo := infrarepos.NewChainRepository(db)
	tokenRepo := infrarepos.NewTokenRepository(db, chainRepo)
	contractRepo := infrarepos.NewSmartContractRepository(db, chainRepo)
	quoteRepo := infrarepos.NewPaymentQuoteRepository(db)
	sessionRepo := infrarepos.NewPartnerPaymentSessionRepository(db)
	paymentRequestRepo := infrarepos.NewPaymentRequestRepository(db)
	uow := infrarepos.NewUnitOfWork(db)

	jweService, err := services.NewJWEService([]byte("12345678901234567890123456789012"))
	require.NoError(t, err)

	quoteUsecase := NewPartnerQuoteUsecase(quoteRepo, tokenRepo, chainRepo, nil)
	quoteUsecase.routeSupportFn = func(context.Context, uuid.UUID, string, string) (*TokenRouteSupportStatus, error) {
		return &TokenRouteSupportStatus{
			Exists:      true,
			IsDirect:    true,
			Path:        []string{"0xidrxtoken", "0xusdctoken"},
			Executable:  true,
			UniversalV4: "0xuniversalrouter",
		}, nil
	}
	quoteUsecase.swapQuoteFn = func(context.Context, uuid.UUID, string, string, *big.Int) (*big.Int, error) {
		return big.NewInt(2950000), nil
	}

	paymentRequestUsecase := NewPaymentRequestUsecase(paymentRequestRepo, nil, nil, chainRepo, contractRepo, tokenRepo, jweService)
	sessionUsecase := NewPartnerPaymentSessionUsecase(
		quoteRepo,
		sessionRepo,
		paymentRequestRepo,
		contractRepo,
		tokenRepo,
		chainRepo,
		uow,
		jweService,
		paymentRequestUsecase,
		"https://partner.pay.test/checkout",
	)
	webhookUsecase := NewWebhookUsecase(nil, nil, paymentRequestRepo, sessionRepo, nil, nil, nil, nil)

	quoteOut, err := quoteUsecase.CreateQuote(ctx, &CreatePartnerQuoteInput{
		MerchantID:      merchantID,
		InvoiceCurrency: "IDRX",
		InvoiceAmount:   "50000000000",
		SelectedChain:   "eip155:8453",
		SelectedToken:   "0xusdctoken",
		DestWallet:      "0xmerchantdestination",
	})
	require.NoError(t, err)
	require.Equal(t, "IDRX", quoteOut.InvoiceCurrency)
	require.Equal(t, "2950000", quoteOut.QuotedAmount)

	quoteID := uuid.MustParse(quoteOut.QuoteID)
	quoteEntity, err := quoteRepo.GetByID(ctx, quoteID)
	require.NoError(t, err)
	require.Equal(t, domainentities.PaymentQuoteStatusActive, quoteEntity.Status)

	sessionOut, err := sessionUsecase.CreateSession(ctx, &CreatePartnerPaymentSessionInput{
		MerchantID: merchantID,
		QuoteID:    quoteID,
		DestWallet: "0xmerchantdestination",
	})
	require.NoError(t, err)
	require.Equal(t, "PENDING", sessionOut.Status)
	require.Contains(t, sessionOut.PaymentURL, "/checkout/")
	require.NotEmpty(t, sessionOut.PaymentCode)
	require.Equal(t, "0xgateway0000000000000000000000000000000000", sessionOut.PaymentInstruction.To)

	quoteEntity, err = quoteRepo.GetByID(ctx, quoteID)
	require.NoError(t, err)
	require.Equal(t, domainentities.PaymentQuoteStatusUsed, quoteEntity.Status)

	sessionID := uuid.MustParse(sessionOut.PaymentID)
	hostedRead, err := sessionUsecase.GetSession(ctx, sessionID)
	require.NoError(t, err)
	require.Equal(t, sessionOut.PaymentID, hostedRead.PaymentID)
	require.Equal(t, sessionOut.PaymentCode, hostedRead.PaymentCode)
	require.Equal(t, sessionOut.PaymentInstruction.Data, hostedRead.PaymentInstruction.Data)

	resolveOut, err := sessionUsecase.ResolvePaymentCode(ctx, &ResolvePartnerPaymentCodeInput{
		PaymentCode: sessionOut.PaymentCode,
		PayerWallet: "0xpayerwallet",
	})
	require.NoError(t, err)
	require.Equal(t, sessionOut.PaymentID, resolveOut.PaymentID)
	require.Equal(t, sessionOut.Amount, resolveOut.Amount)
	require.Equal(t, sessionOut.DestWallet, resolveOut.DestWallet)

	sessionEntity, err := sessionRepo.GetByID(ctx, sessionID)
	require.NoError(t, err)
	require.NotNil(t, sessionEntity.PaymentRequestID)
	require.Equal(t, domainentities.PartnerPaymentSessionStatusPending, sessionEntity.Status)

	raw, err := json.Marshal(map[string]string{
		"id":     sessionEntity.PaymentRequestID.String(),
		"payer":  "0xpayerwallet",
		"txHash": "0xpartnerflowcompleted",
	})
	require.NoError(t, err)
	require.NoError(t, webhookUsecase.ProcessIndexerWebhook(ctx, "REQUEST_PAYMENT_RECEIVED", raw))

	sessionEntity, err = sessionRepo.GetByID(ctx, sessionID)
	require.NoError(t, err)
	require.Equal(t, domainentities.PartnerPaymentSessionStatusCompleted, sessionEntity.Status)
	require.NotNil(t, sessionEntity.PaidTxHash)
	require.Equal(t, "0xpartnerflowcompleted", *sessionEntity.PaidTxHash)
	require.NotNil(t, sessionEntity.CompletedAt)

	paymentRequestEntity, err := paymentRequestRepo.GetByID(ctx, *sessionEntity.PaymentRequestID)
	require.NoError(t, err)
	require.Equal(t, domainentities.PaymentRequestStatusCompleted, paymentRequestEntity.Status)
	require.Equal(t, "0xpartnerflowcompleted", paymentRequestEntity.TxHash)
}

func newPartnerFlowIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", t.Name(), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func mustExecIntegration(t *testing.T, db *gorm.DB, q string, args ...interface{}) {
	t.Helper()
	require.NoError(t, db.Exec(q, args...).Error, "exec failed: %s", q)
}

func createPartnerFlowIntegrationTables(t *testing.T, db *gorm.DB) {
	t.Helper()

	mustExecIntegration(t, db, `CREATE TABLE chains (
		id TEXT PRIMARY KEY,
		chain_id TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		rpc_url TEXT,
		explorer_url TEXT,
		currency_symbol TEXT,
		image_url TEXT,
		is_active BOOLEAN,
		state_machine_id TEXT,
		ccip_chain_selector TEXT,
		stargate_eid INTEGER,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)
	mustExecIntegration(t, db, `CREATE TABLE chain_rpcs (
		id TEXT PRIMARY KEY,
		chain_id TEXT NOT NULL,
		url TEXT NOT NULL,
		priority INTEGER,
		is_active BOOLEAN,
		last_error_at DATETIME,
		error_count INTEGER,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)
	mustExecIntegration(t, db, `CREATE TABLE tokens (
		id TEXT PRIMARY KEY,
		chain_id TEXT NOT NULL,
		symbol TEXT NOT NULL,
		name TEXT NOT NULL,
		decimals INTEGER NOT NULL,
		address TEXT,
		type TEXT NOT NULL,
		logo_url TEXT,
		is_active BOOLEAN,
		is_native BOOLEAN,
		is_stablecoin BOOLEAN,
		min_amount TEXT,
		max_amount TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)
	mustExecIntegration(t, db, `CREATE TABLE smart_contracts (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		chain_id TEXT NOT NULL,
		address TEXT NOT NULL,
		abi TEXT NOT NULL,
		type TEXT NOT NULL,
		version TEXT NOT NULL,
		deployer_address TEXT,
		token0_address TEXT,
		token1_address TEXT,
		fee_tier INTEGER,
		hook_address TEXT,
		start_block INTEGER,
		metadata TEXT,
		is_active BOOLEAN,
		destination_map TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)
	mustExecIntegration(t, db, `CREATE TABLE payment_requests (
		id TEXT PRIMARY KEY,
		merchant_id TEXT NOT NULL,
		chain_id TEXT NOT NULL,
		token_id TEXT NOT NULL,
		wallet_address TEXT NOT NULL,
		amount TEXT NOT NULL,
		decimals INTEGER NOT NULL,
		description TEXT,
		status TEXT NOT NULL,
		expires_at DATETIME NOT NULL,
		tx_hash TEXT,
		payer_address TEXT,
		payment_code TEXT,
		completed_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)
	mustExecIntegration(t, db, `CREATE TABLE payment_quotes (
		id TEXT PRIMARY KEY,
		merchant_id TEXT NOT NULL,
		invoice_currency TEXT NOT NULL,
		invoice_amount TEXT NOT NULL,
		selected_chain_id TEXT NOT NULL,
		selected_token_address TEXT NOT NULL,
		selected_token_symbol TEXT NOT NULL,
		selected_token_decimals INTEGER NOT NULL,
		quoted_amount TEXT NOT NULL,
		quote_rate TEXT NOT NULL,
		price_source TEXT NOT NULL,
		route TEXT NOT NULL,
		slippage_bps INTEGER NOT NULL,
		rate_timestamp DATETIME NOT NULL,
		expires_at DATETIME NOT NULL,
		status TEXT NOT NULL,
		used_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME
	);`)
	mustExecIntegration(t, db, `CREATE TABLE partner_payment_sessions (
		id TEXT PRIMARY KEY,
		merchant_id TEXT NOT NULL,
		quote_id TEXT,
		payment_request_id TEXT,
		invoice_currency TEXT NOT NULL,
		invoice_amount TEXT NOT NULL,
		selected_chain_id TEXT NOT NULL,
		selected_token_address TEXT NOT NULL,
		selected_token_symbol TEXT NOT NULL,
		selected_token_decimals INTEGER NOT NULL,
		dest_chain TEXT NOT NULL,
		dest_token TEXT NOT NULL,
		dest_wallet TEXT NOT NULL,
		payment_amount TEXT NOT NULL,
		payment_amount_decimals INTEGER NOT NULL,
		status TEXT NOT NULL,
		channel_used TEXT,
		payment_code TEXT NOT NULL,
		payment_url TEXT NOT NULL,
		instruction_to TEXT,
		instruction_value TEXT,
		instruction_data_hex TEXT,
		instruction_data_base58 TEXT,
		instruction_data_base64 TEXT,
		quote_rate TEXT,
		quote_source TEXT,
		quote_route TEXT,
		quote_expires_at DATETIME,
		quote_snapshot_json TEXT,
		expires_at DATETIME NOT NULL,
		paid_tx_hash TEXT,
		paid_chain_id TEXT,
		paid_token_address TEXT,
		paid_amount TEXT,
		paid_sender_address TEXT,
		completed_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME
	);`)
}
