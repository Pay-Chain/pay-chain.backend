package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/volatiletech/null/v8"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"payment-kita.backend/internal/domain/entities"
	"payment-kita.backend/internal/infrastructure/models"
	"payment-kita.backend/internal/infrastructure/repositories"
	servicesimpl "payment-kita.backend/internal/infrastructure/services"
	"payment-kita.backend/internal/usecases"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Manually create tables with SQLite compatible schema
	queries := []string{
		`CREATE TABLE users (
			id TEXT PRIMARY KEY, email TEXT, name TEXT, password_hash TEXT, role TEXT, 
			kyc_status TEXT, kyc_verified_at DATETIME, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE merchants (
			id TEXT PRIMARY KEY, user_id TEXT, business_name TEXT, business_email TEXT, status TEXT, 
			merchant_type TEXT, callback_url TEXT, webhook_secret TEXT, webhook_is_active BOOLEAN, 
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE payments (
			id TEXT PRIMARY KEY, 
			sender_id TEXT, 
			merchant_id TEXT, 
			bridge_id TEXT,
			source_chain_id TEXT, 
			dest_chain_id TEXT, 
			source_token_id TEXT, 
			dest_token_id TEXT, 
			source_amount TEXT, 
			dest_amount TEXT,
			fee_amount TEXT,
			total_charged TEXT,
			sender_address TEXT,
			dest_address TEXT,
			status TEXT, 
			source_tx_hash TEXT,
			dest_tx_hash TEXT,
			refund_tx_hash TEXT,
			cross_chain_message_id TEXT,
			failure_reason TEXT,
			revert_data TEXT,
			expires_at DATETIME,
			created_at DATETIME, 
			updated_at DATETIME, 
			deleted_at DATETIME
		)`,
		`CREATE TABLE webhook_logs (
			id TEXT PRIMARY KEY, merchant_id TEXT, payment_id TEXT, event_type TEXT, payload TEXT, 
			delivery_status TEXT, http_status INTEGER, response_body TEXT, retry_count INTEGER DEFAULT 0, 
			next_retry_at DATETIME, last_attempt_at DATETIME, created_at DATETIME, updated_at DATETIME
		)`,
		`CREATE TABLE chains (
			id TEXT PRIMARY KEY, chain_id TEXT, name TEXT, currency_symbol TEXT, image_url TEXT, 
			rpc_url TEXT, type TEXT, is_active BOOLEAN, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE chain_rpcs (
			id TEXT PRIMARY KEY, chain_id TEXT, url TEXT, priority INTEGER, is_active BOOLEAN, 
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE tokens (
			id TEXT PRIMARY KEY, chain_id TEXT, symbol TEXT, name TEXT, address TEXT, 
			decimals INTEGER, type TEXT, is_active BOOLEAN, is_native BOOLEAN, is_stablecoin BOOLEAN, 
			min_amount TEXT, max_amount TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE payment_events (
			id TEXT PRIMARY KEY, payment_id TEXT, event_type TEXT, chain_id TEXT, chain TEXT,
			tx_hash TEXT, block_number BIGINT, metadata TEXT, created_at DATETIME
		)`,
		`CREATE TABLE api_keys (
			id TEXT PRIMARY KEY, user_id TEXT, name TEXT, key_hash TEXT, is_active BOOLEAN, 
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE smart_contracts (
			id TEXT PRIMARY KEY, chain_id TEXT, name TEXT, type TEXT, address TEXT, 
			version INTEGER, is_active BOOLEAN, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE payment_bridge (
			id TEXT PRIMARY KEY, name TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE bridge_configs (
			id TEXT PRIMARY KEY, bridge_id TEXT, source_chain_id TEXT, dest_chain_id TEXT, 
			router_address TEXT, fee_percentage TEXT, config TEXT, is_active BOOLEAN, 
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE fee_configs (
			id TEXT PRIMARY KEY, chain_id TEXT, token_id TEXT, platform_fee_percent TEXT, 
			fixed_base_fee TEXT, min_fee TEXT, max_fee TEXT, 
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE route_policies (
			id TEXT PRIMARY KEY, source_chain_id TEXT, dest_chain_id TEXT, 
			default_bridge_type INTEGER, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE route_errors (
			id TEXT PRIMARY KEY, payment_id TEXT, source_chain_id TEXT, dest_chain_id TEXT,
			source_token_id TEXT, dest_token_id TEXT, amount TEXT, error_message TEXT,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
	}

	for _, q := range queries {
		err := db.Exec(q).Error
		require.NoError(t, err, "Failed to execute query: %s", q)
	}

	return db
}

func TestPaymentAttribution(t *testing.T) {
	db := setupTestDB(t)

	// Initialize all required repositories
	chainRepo := repositories.NewChainRepository(db)
	tokenRepo := repositories.NewTokenRepository(db, chainRepo)
	paymentRepo := repositories.NewPaymentRepository(db)
	paymentEventRepo := repositories.NewPaymentEventRepository(db)
	walletRepo := repositories.NewWalletRepository(db)
	merchantRepo := repositories.NewMerchantRepository(db)
	contractRepo := repositories.NewSmartContractRepository(db, chainRepo)
	bridgeConfigRepo := repositories.NewBridgeConfigRepository(db)
	feeConfigRepo := repositories.NewFeeConfigRepository(db)
	routePolicyRepo := repositories.NewRoutePolicyRepository(db)
	uow := repositories.NewUnitOfWork(db)

	// Setup Usecase
	uc := usecases.NewPaymentUsecase(
		paymentRepo,
		paymentEventRepo,
		walletRepo,
		merchantRepo,
		contractRepo,
		chainRepo,
		tokenRepo,
		bridgeConfigRepo,
		feeConfigRepo,
		routePolicyRepo,
		uow,
		nil, // clientFactory
	)

	userID := uuid.New()
	merchantID := uuid.New()

	// Create required data for attribution
	db.Exec("INSERT INTO users (id, email, name, password_hash, role) VALUES (?, ?, ?, ?, ?)", userID, "test@user.com", "Test", "hash", "user")
	db.Exec("INSERT INTO merchants (id, user_id, business_name, business_email, status, merchant_type) VALUES (?, ?, ?, ?, ?, ?)", merchantID, userID, "Test Merchant", "test@merchant.com", "active", "individual")

	chainUUID := uuid.New()
	db.Exec("INSERT INTO chains (id, chain_id, name, type, is_active) VALUES (?, ?, ?, ?, ?)", chainUUID, "eip155:1", "Ethereum", "evm", true)
	db.Exec("INSERT INTO tokens (id, chain_id, symbol, name, address, decimals, is_active) VALUES (?, ?, ?, ?, ?, ?, ?)", uuid.New(), chainUUID, "USDC", "USD Coin", "0xUSDC", 6, true)

	ctx := context.WithValue(context.Background(), "MerchantID", merchantID)

	input := &entities.CreatePaymentInput{
		Amount:             "100",
		SourceChainID:      "eip155:1",
		DestChainID:        "eip155:1",
		SourceTokenAddress: "0xUSDC",
		DestTokenAddress:   "0xUSDC",
		ReceiverAddress:    "0xReceiver",
		Decimals:           6,
	}

	result, err := uc.CreatePayment(ctx, userID, input)
	require.NoError(t, err)

	// Verify in DB
	var dbPayment models.Payment
	err = db.First(&dbPayment, "id = ?", result.PaymentID).Error
	require.NoError(t, err)
	assert.NotNil(t, dbPayment.MerchantID)
	assert.Equal(t, merchantID, *dbPayment.MerchantID)
}

func TestWebhookSignature(t *testing.T) {
	db := setupTestDB(t)
	merchantRepo := repositories.NewMerchantRepository(db)
	webhookRepo := repositories.NewGormWebhookLogRepository(db)
	hmacService := servicesimpl.NewHMACService()

	secret := "test-secret-123"
	merchantID := uuid.New()
	db.Exec("INSERT INTO merchants (id, user_id, business_name, business_email, webhook_secret, webhook_is_active, merchant_type) VALUES (?, ?, ?, ?, ?, ?, ?)",
		merchantID, uuid.New(), "Test Merchant", "test@merchant.com", secret, true, "individual")

	// Setup mock server
	receivedHeaders := make(http.Header)
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		for k, v := range r.Header {
			receivedHeaders[k] = v
		}
		receivedBody, _ = io.ReadAll(r.Body)
	}))
	defer server.Close()

	// Update merchant with mock server URL
	db.Exec("UPDATE merchants SET callback_url = ? WHERE id = ?", server.URL, merchantID)

	dispatcher := usecases.NewWebhookDispatcher(webhookRepo, merchantRepo, hmacService)

	payload := map[string]interface{}{"status": "success"}
	payloadBytes, _ := json.Marshal(payload)
	delivery := &entities.WebhookDelivery{
		ID:             uuid.New(),
		MerchantID:     merchantID,
		PaymentID:      uuid.New(),
		EventType:      "payment.completed",
		Payload:        null.JSONFrom(payloadBytes),
		DeliveryStatus: entities.WebhookDeliveryStatusPending,
		CreatedAt:      time.Now(),
	}

	err := dispatcher.Dispatch(context.Background(), delivery)
	require.NoError(t, err)

	// Verify headers
	signature := receivedHeaders.Get("X-Webhook-Signature")
	legacySignature := receivedHeaders.Get("X-Webhook-Signature-Legacy")
	timestamp := receivedHeaders.Get("X-Webhook-Timestamp")
	deliveryID := receivedHeaders.Get("X-Webhook-Delivery-Id")
	eventType := receivedHeaders.Get("X-Webhook-Event")
	assert.NotEmpty(t, signature)
	assert.NotEmpty(t, legacySignature)
	assert.NotEmpty(t, timestamp)
	assert.Equal(t, delivery.ID.String(), deliveryID)
	assert.Equal(t, "payment.completed", eventType)

	// Verify signature manually
	expected := generateHmac(secret, timestamp, receivedBody)
	expectedLegacy := generateLegacyHmac(secret, timestamp, receivedBody)
	assert.Equal(t, expected, signature)
	assert.Equal(t, expectedLegacy, legacySignature)

	// Verify delivery status updated to delivered
	var dbLog models.WebhookLog
	db.First(&dbLog, "id = ?", delivery.ID)
	assert.Equal(t, "delivered", dbLog.DeliveryStatus)
	assert.Equal(t, 200, dbLog.HttpStatus)
}

func TestWebhookRetryOnFailure(t *testing.T) {
	db := setupTestDB(t)
	merchantRepo := repositories.NewMerchantRepository(db)
	webhookRepo := repositories.NewGormWebhookLogRepository(db)
	hmacService := servicesimpl.NewHMACService()

	merchantID := uuid.New()
	db.Exec("INSERT INTO merchants (id, user_id, business_name, business_email, webhook_is_active, merchant_type) VALUES (?, ?, ?, ?, ?, ?)",
		merchantID, uuid.New(), "Fail Merchant", "fail@merchant.com", true, "individual")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	db.Exec("UPDATE merchants SET callback_url = ? WHERE id = ?", server.URL, merchantID)

	dispatcher := usecases.NewWebhookDispatcher(webhookRepo, merchantRepo, hmacService)

	payload := map[string]interface{}{"status": "fail"}
	payloadBytes, _ := json.Marshal(payload)
	delivery := &entities.WebhookDelivery{
		ID:             uuid.New(),
		MerchantID:     merchantID,
		PaymentID:      uuid.New(),
		EventType:      "payment.completed",
		Payload:        null.JSONFrom(payloadBytes),
		DeliveryStatus: entities.WebhookDeliveryStatusPending,
		CreatedAt:      time.Now(),
	}

	err := dispatcher.Dispatch(context.Background(), delivery)
	require.NoError(t, err)

	// Verify in DB
	var dbLog models.WebhookLog
	db.First(&dbLog, "id = ?", delivery.ID)
	assert.Equal(t, "retrying", dbLog.DeliveryStatus)
	assert.Equal(t, 500, dbLog.HttpStatus)
	assert.Equal(t, 1, dbLog.RetryCount)
}

func generateHmac(secret, timestamp string, payload []byte) string {
	hmacService := servicesimpl.NewHMACService()
	return hmacService.Generate(timestamp+"."+string(payload), secret)
}

func generateLegacyHmac(secret, timestamp string, payload []byte) string {
	hmacService := servicesimpl.NewHMACService()
	return hmacService.Generate(timestamp+string(payload), secret)
}
