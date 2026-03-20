package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"payment-kita.backend/internal/infrastructure/repositories"
)

func TestAdminMerchantSettlementHandler_GetAndUpsert(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testDB := repositoriesTestDBForAdminSettlement(t)
	repositoriesTestCreateAdminSettlementTables(t, testDB)
	now := time.Now().UTC()
	merchantID := uuid.New()
	chainID := uuid.New()
	mustExecAdminSettlement(t, testDB, `INSERT INTO merchants (id, user_id, business_name, business_email, merchant_type, status, documents, fee_discount_percent, webhook_metadata, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, merchantID.String(), uuid.NewString(), "Merchant", "merchant@example.com", "PARTNER", "ACTIVE", "{}", "0", "{}", "{}", now, now)
	mustExecAdminSettlement(t, testDB, `INSERT INTO chains (id, chain_id, name, type, rpc_url, explorer_url, currency_symbol, image_url, is_active, state_machine_id, ccip_chain_selector, stargate_eid, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, chainID.String(), "8453", "Base", "EVM", "", "", "ETH", "", true, "", "", 0, now, now)
	mustExecAdminSettlement(t, testDB, `INSERT INTO chain_rpcs (id, chain_id, url, priority, is_active, error_count, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, uuid.NewString(), chainID.String(), "https://rpc.base.example", 1, true, 0, now, now)
	mustExecAdminSettlement(t, testDB, `INSERT INTO tokens (id, chain_id, symbol, name, decimals, address, type, logo_url, is_active, is_native, is_stablecoin, min_amount, max_amount, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, uuid.NewString(), chainID.String(), "IDRX", "IDRX", 2, "0xidrxtoken", "ERC20", "", true, false, false, "0", nil, now, now)
	mustExecAdminSettlement(t, testDB, `INSERT INTO tokens (id, chain_id, symbol, name, decimals, address, type, logo_url, is_active, is_native, is_stablecoin, min_amount, max_amount, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, uuid.NewString(), chainID.String(), "USDC", "USDC", 6, "0xusdctoken", "ERC20", "", true, false, true, "0", nil, now, now)

	merchantRepo := repositories.NewMerchantRepository(testDB)
	settlementRepo := repositories.NewMerchantSettlementProfileRepository(testDB)
	chainRepo := repositories.NewChainRepository(testDB)
	tokenRepo := repositories.NewTokenRepository(testDB, chainRepo)
	h := NewAdminMerchantSettlementHandler(merchantRepo, settlementRepo, chainRepo, tokenRepo)

	r := gin.New()
	r.GET("/admin/merchants/:id/settlement-profile", h.GetSettlementProfile)
	r.PUT("/admin/merchants/:id/settlement-profile", h.UpsertSettlementProfile)

	getReq := httptest.NewRequest(http.MethodGet, "/admin/merchants/"+merchantID.String()+"/settlement-profile", nil)
	getRec := httptest.NewRecorder()
	r.ServeHTTP(getRec, getReq)
	require.Equal(t, http.StatusOK, getRec.Code)
	require.Contains(t, getRec.Body.String(), `"configured":false`)

	putReq := httptest.NewRequest(http.MethodPut, "/admin/merchants/"+merchantID.String()+"/settlement-profile", bytes.NewBufferString(`{
		"invoice_currency":"IDRX",
		"dest_chain":"eip155:8453",
		"dest_token":"0xidrxtoken",
		"dest_wallet":"0xmerchantwallet",
		"bridge_token_symbol":"USDC"
	}`))
	putReq.Header.Set("Content-Type", "application/json")
	putRec := httptest.NewRecorder()
	r.ServeHTTP(putRec, putReq)
	require.Equal(t, http.StatusOK, putRec.Code, putRec.Body.String())
	require.Contains(t, putRec.Body.String(), `"configured":true`)

	getReq = httptest.NewRequest(http.MethodGet, "/admin/merchants/"+merchantID.String()+"/settlement-profile", nil)
	getRec = httptest.NewRecorder()
	r.ServeHTTP(getRec, getReq)
	require.Equal(t, http.StatusOK, getRec.Code)
	require.Contains(t, getRec.Body.String(), `"invoice_currency":"IDRX"`)
	require.Contains(t, getRec.Body.String(), `"dest_wallet":"0xmerchantwallet"`)
}

func repositoriesTestDBForAdminSettlement(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", t.Name(), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func mustExecAdminSettlement(t *testing.T, db *gorm.DB, q string, args ...interface{}) {
	t.Helper()
	require.NoError(t, db.Exec(q, args...).Error, "exec failed: %s", q)
}

func repositoriesTestCreateAdminSettlementTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	mustExecAdminSettlement(t, db, `CREATE TABLE merchants (
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
	mustExecAdminSettlement(t, db, `CREATE TABLE merchant_settlement_profiles (
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
	mustExecAdminSettlement(t, db, `CREATE TABLE chains (
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
	mustExecAdminSettlement(t, db, `CREATE TABLE chain_rpcs (
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
	mustExecAdminSettlement(t, db, `CREATE TABLE tokens (
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
}
