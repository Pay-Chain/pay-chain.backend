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

	"payment-kita.backend/internal/domain/services"
	"payment-kita.backend/internal/infrastructure/repositories"
	"payment-kita.backend/internal/interfaces/http/middleware"
	"payment-kita.backend/internal/usecases"
)

func TestCreatePayment_SettlementProfileScenarios_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type scenario struct {
		name                string
		profileInvoice      string
		profileDestChain    string
		profileDestToken    string
		profileBridgeSymbol string
		requestChain        string
		requestToken        string
		pricingType         string
		requestedAmount     string
		expectDestChain     string
		expectDestToken     string
		expectQuotedSymbol  string
		expectQuotedAtomic  string
		expectPriceSource   string
	}

	scenarios := []scenario{
		{
			name:                "same-chain invoice currency",
			profileInvoice:      "IDRX",
			profileDestChain:    "eip155:8453",
			profileDestToken:    "0xbaseidrx",
			profileBridgeSymbol: "USDC",
			requestChain:        "eip155:8453",
			requestToken:        "0xbaseusdc",
			pricingType:         "invoice_currency",
			requestedAmount:     "50000",
			expectDestChain:     "eip155:8453",
			expectDestToken:     "0xbaseidrx",
			expectQuotedSymbol:  "USDC",
			expectQuotedAtomic:  "2950000",
			expectPriceSource:   "uniswap-v4-base-usdc-idrx",
		},
		{
			name:                "cross-chain bridge-token direct",
			profileInvoice:      "IDRT",
			profileDestChain:    "eip155:137",
			profileDestToken:    "0xpolygonidrt",
			profileBridgeSymbol: "USDC",
			requestChain:        "eip155:8453",
			requestToken:        "0xbaseusdc",
			pricingType:         "invoice_currency",
			requestedAmount:     "50000",
			expectDestChain:     "eip155:137",
			expectDestToken:     "0xpolygonidrt",
			expectQuotedSymbol:  "USDC",
			expectQuotedAtomic:  "2950000",
			expectPriceSource:   "cross-chain-bridge-token-direct-via-usdc",
		},
		{
			name:                "cross-chain normalized via non-bridge token",
			profileInvoice:      "IDRT",
			profileDestChain:    "eip155:137",
			profileDestToken:    "0xpolygonidrt",
			profileBridgeSymbol: "USDC",
			requestChain:        "eip155:8453",
			requestToken:        "0xbasexsgd",
			pricingType:         "invoice_currency",
			requestedAmount:     "50000",
			expectDestChain:     "eip155:137",
			expectDestToken:     "0xpolygonidrt",
			expectQuotedSymbol:  "XSGD",
			expectQuotedAtomic:  "3100000",
			expectPriceSource:   "cross-chain-normalized-via-usdc",
		},
	}

	for _, tc := range scenarios {
		t.Run(tc.name, func(t *testing.T) {
			db := newCreatePaymentScenarioDB(t)
			createPartnerHTTPFlowTables(t, db)
			now := time.Now().UTC()
			merchantID := uuid.New()
			userID := uuid.New()
			baseChainID := uuid.New()
			polygonChainID := uuid.New()

			mustExecPartnerHTTP(t, db, `INSERT INTO merchants (id, user_id, business_name, business_email, merchant_type, status, tax_id, business_address, documents, fee_discount_percent, callback_url, webhook_secret, webhook_is_active, support_email, logo_url, webhook_metadata, metadata, verified_at, created_at, updated_at, deleted_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)`, merchantID.String(), userID.String(), "Merchant", "merchant@example.com", "PARTNER", "ACTIVE", "", "", "{}", "0", "", "", false, "", "", "{}", `{}`, now, now, now)
			mustExecPartnerHTTP(t, db, `INSERT INTO merchant_settlement_profiles (id, merchant_id, invoice_currency, dest_chain, dest_token, dest_wallet, bridge_token_symbol, created_at, updated_at, deleted_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)`, uuid.NewString(), merchantID.String(), tc.profileInvoice, tc.profileDestChain, tc.profileDestToken, "0xmerchantdestination", tc.profileBridgeSymbol, now, now)
			mustExecPartnerHTTP(t, db, `INSERT INTO chains (id, chain_id, name, type, rpc_url, explorer_url, currency_symbol, image_url, is_active, state_machine_id, ccip_chain_selector, stargate_eid, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, baseChainID.String(), "8453", "Base", "EVM", "https://rpc.base.example", "", "ETH", "", true, "", "", 0, now, now)
			mustExecPartnerHTTP(t, db, `INSERT INTO chains (id, chain_id, name, type, rpc_url, explorer_url, currency_symbol, image_url, is_active, state_machine_id, ccip_chain_selector, stargate_eid, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, polygonChainID.String(), "137", "Polygon", "EVM", "https://rpc.polygon.example", "", "POL", "", true, "", "", 0, now, now)
			mustExecPartnerHTTP(t, db, `INSERT INTO wallets (id, user_id, merchant_id, chain_id, address, is_primary, created_at, updated_at, deleted_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL)`, uuid.NewString(), userID.String(), merchantID.String(), baseChainID.String(), "0xmerchantdestination", true, now, now)

			insertToken := func(id uuid.UUID, chain uuid.UUID, symbol, address string, decimals int, stable bool) {
				mustExecPartnerHTTP(t, db, `INSERT INTO tokens (id, chain_id, symbol, name, decimals, address, type, logo_url, is_active, is_native, is_stablecoin, min_amount, max_amount, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, id.String(), chain.String(), symbol, symbol, decimals, address, "ERC20", "", true, false, stable, "0", nil, now, now)
			}
			insertToken(uuid.New(), baseChainID, "IDRX", "0xbaseidrx", 2, false)
			insertToken(uuid.New(), baseChainID, "USDC", "0xbaseusdc", 6, true)
			insertToken(uuid.New(), baseChainID, "XSGD", "0xbasexsgd", 6, false)
			insertToken(uuid.New(), polygonChainID, "IDRT", "0xpolygonidrt", 2, false)
			insertToken(uuid.New(), polygonChainID, "USDC", "0xpolygonusdc", 6, true)

			mustExecPartnerHTTP(t, db, `INSERT INTO smart_contracts (id, name, chain_id, address, abi, type, version, deployer_address, token0_address, token1_address, fee_tier, hook_address, start_block, metadata, is_active, destination_map, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, uuid.NewString(), "Gateway", baseChainID.String(), "0xgateway0000000000000000000000000000000000", "[]", "GATEWAY", "1.0.0", "", "", "", 0, "", 0, "{}", true, "{}", now, now)

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
			quoteUsecase.RouteSupportFnForTest(func(_ context.Context, _ uuid.UUID, tokenIn string, tokenOut string) (*usecases.TokenRouteSupportStatus, error) {
				if tokenIn == "0xpolygonusdc" && tokenOut == "0xbasexsgd" {
					return &usecases.TokenRouteSupportStatus{Exists: true, Executable: true, IsDirect: false, Path: []string{"0xbaseusdc", "0xbasexsgd"}, UniversalV4: "0xuniversalrouter"}, nil
				}
				return &usecases.TokenRouteSupportStatus{Exists: true, Executable: true, IsDirect: true, Path: []string{tokenIn, tokenOut}, UniversalV4: "0xuniversalrouter"}, nil
			})
			quoteUsecase.SwapQuoteFnForTest(func(_ context.Context, _ uuid.UUID, tokenIn string, tokenOut string, _ *big.Int) (*big.Int, error) {
				switch fmt.Sprintf("%s->%s", tokenIn, tokenOut) {
				case "0xbaseidrx->0xbaseusdc":
					return big.NewInt(2950000), nil
				case "0xpolygonidrt->0xpolygonusdc":
					return big.NewInt(2950000), nil
				case "0xbaseusdc->0xbasexsgd":
					return big.NewInt(3100000), nil
				default:
					return big.NewInt(2950000), nil
				}
			})
			paymentRequestUsecase := usecases.NewPaymentRequestUsecase(paymentRequestRepo, merchantRepo, nil, chainRepo, contractRepo, tokenRepo, jweService)
			sessionUsecase := usecases.NewPartnerPaymentSessionUsecase(quoteRepo, sessionRepo, paymentRequestRepo, contractRepo, tokenRepo, chainRepo, merchantRepo, uow, jweService, paymentRequestUsecase, nil, "https://partner.pay.test/pay")
			createPaymentUsecase := usecases.NewCreatePaymentUsecase(merchantRepo, settlementProfileRepo, walletRepo, tokenRepo, chainRepo, quoteRepo, sessionRepo, quoteUsecase, sessionUsecase)
			handler := NewCreatePaymentHandler(createPaymentUsecase)

			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set(middleware.MerchantIDKey, merchantID)
				c.Next()
			})
			router.POST("/api/v1/create-payment", handler.CreatePayment)

			body := fmt.Sprintf(`{"chain_id":"%s","selected_token":"%s","pricing_type":"%s","requested_amount":"%s"}`, tc.requestChain, tc.requestToken, tc.pricingType, tc.requestedAmount)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/create-payment", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

			var got struct {
				PaymentID               string `json:"payment_id"`
				QuotedTokenSymbol       string `json:"quoted_token_symbol"`
				QuotedTokenAmountAtomic string `json:"quoted_token_amount_atomic"`
				QuoteSource             string `json:"quote_source"`
				DestChain               string `json:"dest_chain"`
				DestToken               string `json:"dest_token"`
				PaymentInstruction      struct {
					ChainID string `json:"chain_id"`
				} `json:"payment_instruction"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
			require.NotEmpty(t, got.PaymentID)
			require.Equal(t, tc.expectQuotedSymbol, got.QuotedTokenSymbol)
			require.Equal(t, tc.expectQuotedAtomic, got.QuotedTokenAmountAtomic)
			require.Equal(t, tc.expectPriceSource, got.QuoteSource)
			require.Equal(t, tc.expectDestChain, got.DestChain)
			require.Equal(t, tc.expectDestToken, got.DestToken)
			require.Equal(t, tc.requestChain, got.PaymentInstruction.ChainID)
		})
	}
}

func newCreatePaymentScenarioDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", t.Name(), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	return db
}
