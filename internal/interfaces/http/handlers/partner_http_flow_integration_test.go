package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	domainentities "payment-kita.backend/internal/domain/entities"
	"payment-kita.backend/internal/domain/services"
	"payment-kita.backend/internal/infrastructure/repositories"
	"payment-kita.backend/internal/interfaces/http/middleware"
	"payment-kita.backend/internal/usecases"
)

func TestPartnerHTTPFlow_QuoteSessionReadResolveWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newPartnerHTTPFlowTestDB(t)
	createPartnerHTTPFlowTables(t, db)

	ctx := context.Background()
	merchantID := uuid.New()
	chainID := uuid.New()
	idrxID := uuid.New()
	usdcID := uuid.New()
	now := time.Now().UTC()

	mustExecPartnerHTTP(t, db, `INSERT INTO chains (
		id, chain_id, name, type, rpc_url, explorer_url, currency_symbol, image_url, is_active, state_machine_id, ccip_chain_selector, stargate_eid, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		chainID.String(), "8453", "Base", "EVM", "https://rpc.base.example", "https://basescan.org", "ETH", "", true, "", "", 0, now, now,
	)
	mustExecPartnerHTTP(t, db, `INSERT INTO tokens (
		id, chain_id, symbol, name, decimals, address, type, logo_url, is_active, is_native, is_stablecoin, min_amount, max_amount, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		idrxID.String(), chainID.String(), "IDRX", "IDRX", 2, "0xidrxtoken", "ERC20", "", true, false, false, "0", nil, now, now,
	)
	mustExecPartnerHTTP(t, db, `INSERT INTO tokens (
		id, chain_id, symbol, name, decimals, address, type, logo_url, is_active, is_native, is_stablecoin, min_amount, max_amount, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		usdcID.String(), chainID.String(), "USDC", "USDC", 6, "0xusdctoken", "ERC20", "", true, false, true, "0", nil, now, now,
	)
	mustExecPartnerHTTP(t, db, `INSERT INTO smart_contracts (
		id, name, chain_id, address, abi, type, version, deployer_address, token0_address, token1_address, fee_tier, hook_address, start_block, metadata, is_active, destination_map, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), "Gateway", chainID.String(), "0xgateway0000000000000000000000000000000000", "[]", "GATEWAY", "1.0.0", "", "", "", 0, "", 0, "{}", true, "{}", now, now,
	)

	chainRepo := repositories.NewChainRepository(db)
	tokenRepo := repositories.NewTokenRepository(db, chainRepo)
	contractRepo := repositories.NewSmartContractRepository(db, chainRepo)
	quoteRepo := repositories.NewPaymentQuoteRepository(db)
	sessionRepo := repositories.NewPartnerPaymentSessionRepository(db)
	paymentRequestRepo := repositories.NewPaymentRequestRepository(db)
	uow := repositories.NewUnitOfWork(db)

	jweService, err := services.NewJWEService([]byte("12345678901234567890123456789012"))
	require.NoError(t, err)

	quoteUsecase := usecases.NewPartnerQuoteUsecase(quoteRepo, tokenRepo, chainRepo, nil)
	quoteUsecaseRoute := &usecases.TokenRouteSupportStatus{
		Exists:       true,
		IsDirect:     true,
		Path:         []string{"0xidrxtoken", "0xusdctoken"},
		Executable:   true,
		UniversalV4:  "0xuniversalrouter",
		SwapRouterV3: "",
	}
	quoteUsecaseCreateFns(quoteUsecase, quoteUsecaseRoute)

	merchantRepo := repositories.NewMerchantRepository(db)
	paymentRequestUsecase := usecases.NewPaymentRequestUsecase(paymentRequestRepo, merchantRepo, nil, chainRepo, contractRepo, tokenRepo, jweService)
	sessionUsecase := usecases.NewPartnerPaymentSessionUsecase(
		quoteRepo,
		sessionRepo,
		paymentRequestRepo,
		contractRepo,
		tokenRepo,
		chainRepo,
		merchantRepo,
		uow,
		jweService,
		paymentRequestUsecase,
		nil,
		"https://partner.pay.test/pay",
	)
	webhookUsecase := usecases.NewWebhookUsecase(nil, nil, paymentRequestRepo, sessionRepo, nil, nil, nil, nil)

	quoteHandler := NewPartnerQuoteHandler(quoteUsecase)
	sessionHandler := NewPartnerPaymentSessionHandler(sessionUsecase, nil, nil)
	webhookHandler := NewWebhookHandler(webhookUsecase)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.MerchantIDKey, merchantID)
		c.Next()
	})
	router.POST("/api/v1/partner/quotes", quoteHandler.CreateQuote)
	router.POST("/api/v1/partner/payment-sessions", sessionHandler.CreateSession)
	router.GET("/api/v1/partner/payment-sessions/:id", sessionHandler.GetSession)
	router.POST("/api/v1/partner/payment-sessions/resolve-code", sessionHandler.ResolvePaymentCode)
	router.POST("/api/v1/webhooks/indexer", webhookHandler.HandleIndexerWebhook)

	quoteReq := httptest.NewRequest(http.MethodPost, "/api/v1/partner/quotes", bytes.NewBufferString(`{
		"invoice_currency":"IDRX",
		"invoice_amount":"50000000000",
		"selected_chain":"eip155:8453",
		"selected_token":"0xusdctoken",
		"dest_wallet":"0xmerchantdestination"
	}`))
	quoteReq.Header.Set("Content-Type", "application/json")
	quoteRec := httptest.NewRecorder()
	router.ServeHTTP(quoteRec, quoteReq)
	require.Equal(t, http.StatusOK, quoteRec.Code, quoteRec.Body.String())

	var quoteResp struct {
		QuoteID string `json:"quote_id"`
	}
	require.NoError(t, json.Unmarshal(quoteRec.Body.Bytes(), &quoteResp))
	require.NotEmpty(t, quoteResp.QuoteID)

	sessionReq := httptest.NewRequest(http.MethodPost, "/api/v1/partner/payment-sessions", bytes.NewBufferString(fmt.Sprintf(`{
		"quote_id":"%s",
		"dest_wallet":"0xmerchantdestination"
	}`, quoteResp.QuoteID)))
	sessionReq.Header.Set("Content-Type", "application/json")
	sessionRec := httptest.NewRecorder()
	router.ServeHTTP(sessionRec, sessionReq)
	require.Equal(t, http.StatusOK, sessionRec.Code, sessionRec.Body.String())

	var sessionResp struct {
		PaymentID   string `json:"payment_id"`
		PaymentCode string `json:"payment_code"`
		Status      string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(sessionRec.Body.Bytes(), &sessionResp))
	require.NotEmpty(t, sessionResp.PaymentID)
	require.NotEmpty(t, sessionResp.PaymentCode)
	require.Equal(t, "PENDING", sessionResp.Status)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/partner/payment-sessions/"+sessionResp.PaymentID, nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	require.Equal(t, http.StatusOK, getRec.Code, getRec.Body.String())
	require.Contains(t, getRec.Body.String(), sessionResp.PaymentCode)

	resolveReq := httptest.NewRequest(http.MethodPost, "/api/v1/partner/payment-sessions/resolve-code", bytes.NewBufferString(fmt.Sprintf(`{
		"payment_code":"%s",
		"payer_wallet":"0xpayerwallet"
	}`, sessionResp.PaymentCode)))
	resolveReq.Header.Set("Content-Type", "application/json")
	resolveRec := httptest.NewRecorder()
	router.ServeHTTP(resolveRec, resolveReq)
	require.Equal(t, http.StatusOK, resolveRec.Code, resolveRec.Body.String())
	require.Contains(t, resolveRec.Body.String(), sessionResp.PaymentID)

	sessionID := uuid.MustParse(sessionResp.PaymentID)
	sessionEntity, err := sessionRepo.GetByID(ctx, sessionID)
	require.NoError(t, err)
	require.NotNil(t, sessionEntity.PaymentRequestID)

	webhookBody := fmt.Sprintf(`{
		"eventType":"REQUEST_PAYMENT_RECEIVED",
		"data":{
			"id":"%s",
			"payer":"0xpayerwallet",
			"txHash":"0xpartnerhttpdone"
		}
	}`, sessionEntity.PaymentRequestID.String())
	webhookReq := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/indexer", bytes.NewBufferString(webhookBody))
	webhookReq.Header.Set("Content-Type", "application/json")
	webhookRec := httptest.NewRecorder()
	router.ServeHTTP(webhookRec, webhookReq)
	require.Equal(t, http.StatusOK, webhookRec.Code, webhookRec.Body.String())

	sessionEntity, err = sessionRepo.GetByID(ctx, sessionID)
	require.NoError(t, err)
	require.Equal(t, domainentities.PartnerPaymentSessionStatusCompleted, sessionEntity.Status)
	require.NotNil(t, sessionEntity.PaidTxHash)
	require.Equal(t, "0xpartnerhttpdone", *sessionEntity.PaidTxHash)
}

func TestPartnerHTTPFlow_CreatePaymentReadResolveWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newPartnerHTTPFlowTestDB(t)
	createPartnerHTTPFlowTables(t, db)

	ctx := context.Background()
	merchantID := uuid.New()
	userID := uuid.New()
	chainID := uuid.New()
	idrxID := uuid.New()
	usdcID := uuid.New()
	now := time.Now().UTC()

	mustExecPartnerHTTP(t, db, `INSERT INTO merchants (
		id, user_id, business_name, business_email, merchant_type, status, tax_id, business_address, documents, fee_discount_percent, callback_url, webhook_secret, webhook_is_active, support_email, logo_url, webhook_metadata, metadata, verified_at, created_at, updated_at, deleted_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)`,
		merchantID.String(), userID.String(), "Merchant A", "merchant@example.com", "PARTNER", "ACTIVE", "", "", "{}", "0", "", "", false, "", "", "{}", `{}`, now, now, now,
	)
	mustExecPartnerHTTP(t, db, `INSERT INTO merchant_settlement_profiles (
		id, merchant_id, invoice_currency, dest_chain, dest_token, dest_wallet, bridge_token_symbol, created_at, updated_at, deleted_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)`,
		uuid.NewString(), merchantID.String(), "IDRX", "eip155:8453", "0xidrxtoken", "0xmerchantdestination", "USDC", now, now,
	)
	mustExecPartnerHTTP(t, db, `INSERT INTO chains (
		id, chain_id, name, type, rpc_url, explorer_url, currency_symbol, image_url, is_active, state_machine_id, ccip_chain_selector, stargate_eid, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		chainID.String(), "8453", "Base", "EVM", "https://rpc.base.example", "https://basescan.org", "ETH", "", true, "", "", 0, now, now,
	)
	mustExecPartnerHTTP(t, db, `INSERT INTO wallets (
		id, user_id, merchant_id, chain_id, address, is_primary, created_at, updated_at, deleted_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL)`,
		uuid.NewString(), userID.String(), merchantID.String(), chainID.String(), "0xmerchantdestination", true, now, now,
	)
	mustExecPartnerHTTP(t, db, `INSERT INTO tokens (
		id, chain_id, symbol, name, decimals, address, type, logo_url, is_active, is_native, is_stablecoin, min_amount, max_amount, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		idrxID.String(), chainID.String(), "IDRX", "IDRX", 2, "0xidrxtoken", "ERC20", "", true, false, false, "0", nil, now, now,
	)
	mustExecPartnerHTTP(t, db, `INSERT INTO tokens (
		id, chain_id, symbol, name, decimals, address, type, logo_url, is_active, is_native, is_stablecoin, min_amount, max_amount, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		usdcID.String(), chainID.String(), "USDC", "USDC", 6, "0xusdctoken", "ERC20", "", true, false, true, "0", nil, now, now,
	)
	mustExecPartnerHTTP(t, db, `INSERT INTO smart_contracts (
		id, name, chain_id, address, abi, type, version, deployer_address, token0_address, token1_address, fee_tier, hook_address, start_block, metadata, is_active, destination_map, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), "Gateway", chainID.String(), "0xgateway0000000000000000000000000000000000", "[]", "GATEWAY", "1.0.0", "", "", "", 0, "", 0, "{}", true, "{}", now, now,
	)

	chainRepo := repositories.NewChainRepository(db)
	tokenRepo := repositories.NewTokenRepository(db, chainRepo)
	contractRepo := repositories.NewSmartContractRepository(db, chainRepo)
	quoteRepo := repositories.NewPaymentQuoteRepository(db)
	sessionRepo := repositories.NewPartnerPaymentSessionRepository(db)
	paymentRequestRepo := repositories.NewPaymentRequestRepository(db)
	merchantRepo := repositories.NewMerchantRepository(db)
	settlementProfileRepo := repositories.NewMerchantSettlementProfileRepository(db)
	walletRepo := repositories.NewWalletRepository(db)
	uow := repositories.NewUnitOfWork(db)

	jweService, err := services.NewJWEService([]byte("12345678901234567890123456789012"))
	require.NoError(t, err)

	quoteUsecase := usecases.NewPartnerQuoteUsecase(quoteRepo, tokenRepo, chainRepo, nil)
	quoteUsecaseCreateFns(quoteUsecase, &usecases.TokenRouteSupportStatus{
		Exists:       true,
		IsDirect:     true,
		Path:         []string{"0xidrxtoken", "0xusdctoken"},
		Executable:   true,
		UniversalV4:  "0xuniversalrouter",
		SwapRouterV3: "",
	})
	paymentRequestUsecase := usecases.NewPaymentRequestUsecase(paymentRequestRepo, merchantRepo, nil, chainRepo, contractRepo, tokenRepo, jweService)
	sessionUsecase := usecases.NewPartnerPaymentSessionUsecase(
		quoteRepo,
		sessionRepo,
		paymentRequestRepo,
		contractRepo,
		tokenRepo,
		chainRepo,
		merchantRepo,
		uow,
		jweService,
		paymentRequestUsecase,
		nil,
		"https://partner.pay.test/pay",
	)
	createPaymentUsecase := usecases.NewCreatePaymentUsecase(
		merchantRepo,
		settlementProfileRepo,
		walletRepo,
		tokenRepo,
		chainRepo,
		quoteRepo,
		sessionRepo,
		quoteUsecase,
		sessionUsecase,
	)
	webhookUsecase := usecases.NewWebhookUsecase(nil, nil, paymentRequestRepo, sessionRepo, nil, nil, nil, nil)

	createPaymentHandler := NewCreatePaymentHandler(createPaymentUsecase)
	sessionHandler := NewPartnerPaymentSessionHandler(sessionUsecase, nil, nil)
	webhookHandler := NewWebhookHandler(webhookUsecase)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.MerchantIDKey, merchantID)
		c.Next()
	})
	router.POST("/api/v1/create-payment", createPaymentHandler.CreatePayment)
	router.GET("/api/v1/partner/payment-sessions/:id", sessionHandler.GetSession)
	router.POST("/api/v1/partner/payment-sessions/resolve-code", sessionHandler.ResolvePaymentCode)
	router.POST("/api/v1/webhooks/indexer", webhookHandler.HandleIndexerWebhook)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/create-payment", bytes.NewBufferString(`{
		"chain_id":"eip155:8453",
		"selected_token":"0xusdctoken",
		"pricing_type":"invoice_currency",
		"requested_amount":"50000"
	}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusOK, createRec.Code, createRec.Body.String())

	var createResp struct {
		PaymentID   string `json:"payment_id"`
		PaymentCode string `json:"payment_code"`
	}
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createResp))
	require.NotEmpty(t, createResp.PaymentID)
	require.NotEmpty(t, createResp.PaymentCode)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/partner/payment-sessions/"+createResp.PaymentID, nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	require.Equal(t, http.StatusOK, getRec.Code, getRec.Body.String())
	require.Contains(t, getRec.Body.String(), createResp.PaymentCode)

	resolveReq := httptest.NewRequest(http.MethodPost, "/api/v1/partner/payment-sessions/resolve-code", bytes.NewBufferString(fmt.Sprintf(`{
		"payment_code":"%s",
		"payer_wallet":"0xpayerwallet"
	}`, createResp.PaymentCode)))
	resolveReq.Header.Set("Content-Type", "application/json")
	resolveRec := httptest.NewRecorder()
	router.ServeHTTP(resolveRec, resolveReq)
	require.Equal(t, http.StatusOK, resolveRec.Code, resolveRec.Body.String())

	sessionID := uuid.MustParse(createResp.PaymentID)
	sessionEntity, err := sessionRepo.GetByID(ctx, sessionID)
	require.NoError(t, err)
	require.NotNil(t, sessionEntity.PaymentRequestID)

	webhookBody := fmt.Sprintf(`{
		"eventType":"REQUEST_PAYMENT_RECEIVED",
		"data":{
			"id":"%s",
			"payer":"0xpayerwallet",
			"txHash":"0xcreatepaymentdone"
		}
	}`, sessionEntity.PaymentRequestID.String())
	webhookReq := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/indexer", bytes.NewBufferString(webhookBody))
	webhookReq.Header.Set("Content-Type", "application/json")
	webhookRec := httptest.NewRecorder()
	router.ServeHTTP(webhookRec, webhookReq)
	require.Equal(t, http.StatusOK, webhookRec.Code, webhookRec.Body.String())

	sessionEntity, err = sessionRepo.GetByID(ctx, sessionID)
	require.NoError(t, err)
	require.Equal(t, domainentities.PartnerPaymentSessionStatusCompleted, sessionEntity.Status)
}

func quoteUsecaseCreateFns(uc *usecases.PartnerQuoteUsecase, route *usecases.TokenRouteSupportStatus) {
	ucValue := uc
	ucValueReflectRoute(ucValue, route)
}

func ucValueReflectRoute(uc *usecases.PartnerQuoteUsecase, route *usecases.TokenRouteSupportStatus) {
	uc.RouteSupportFnForTest(func(context.Context, uuid.UUID, string, string) (*usecases.TokenRouteSupportStatus, error) {
		return route, nil
	})
	uc.SwapQuoteFnForTest(func(context.Context, uuid.UUID, string, string, *big.Int) (*big.Int, error) {
		return big.NewInt(2950000), nil
	})
}

func newPartnerHTTPFlowTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", t.Name(), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func mustExecPartnerHTTP(t *testing.T, db *gorm.DB, q string, args ...interface{}) {
	t.Helper()
	require.NoError(t, db.Exec(q, args...).Error, "exec failed: %s", q)
}

func createPartnerHTTPFlowTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	mustExecPartnerHTTP(t, db, `CREATE TABLE merchants (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		business_name TEXT NOT NULL,
		business_email TEXT NOT NULL,
		merchant_type TEXT NOT NULL,
		status TEXT NOT NULL,
		tax_id TEXT,
		business_address TEXT,
		documents TEXT,
		fee_discount_percent TEXT,
		callback_url TEXT,
		webhook_secret TEXT,
		webhook_is_active BOOLEAN,
		support_email TEXT,
		logo_url TEXT,
		webhook_metadata TEXT,
		metadata TEXT,
		verified_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)
	mustExecPartnerHTTP(t, db, `CREATE TABLE merchant_settlement_profiles (
		id TEXT PRIMARY KEY,
		merchant_id TEXT NOT NULL UNIQUE,
		invoice_currency TEXT NOT NULL,
		dest_chain TEXT NOT NULL,
		dest_token TEXT NOT NULL,
		dest_wallet TEXT NOT NULL,
		bridge_token_symbol TEXT NOT NULL,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)
	mustExecPartnerHTTP(t, db, `CREATE TABLE chains (
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
	mustExecPartnerHTTP(t, db, `CREATE TABLE chain_rpcs (
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
	mustExecPartnerHTTP(t, db, `CREATE TABLE tokens (
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
	mustExecPartnerHTTP(t, db, `CREATE TABLE smart_contracts (
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
	mustExecPartnerHTTP(t, db, `CREATE TABLE wallets (
		id TEXT PRIMARY KEY,
		user_id TEXT,
		merchant_id TEXT,
		chain_id TEXT NOT NULL,
		address TEXT NOT NULL,
		is_primary BOOLEAN,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)
	mustExecPartnerHTTP(t, db, `CREATE TABLE payment_requests (
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
	mustExecPartnerHTTP(t, db, `CREATE TABLE payment_quotes (
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
	mustExecPartnerHTTP(t, db, `CREATE TABLE partner_payment_sessions (
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
