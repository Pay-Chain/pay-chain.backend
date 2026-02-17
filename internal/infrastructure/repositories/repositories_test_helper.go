package repositories

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", t.Name(), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "open sqlite")
	return db
}

func mustExec(t *testing.T, db *gorm.DB, q string, args ...interface{}) {
	t.Helper()
	require.NoError(t, db.Exec(q, args...).Error, "exec failed: query=%s", q)
}

func createPaymentBridgeTable(t *testing.T, db *gorm.DB) {
	mustExec(t, db, `CREATE TABLE payment_bridge (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)
}

func createMerchantTable(t *testing.T, db *gorm.DB) {
	mustExec(t, db, `CREATE TABLE merchants (
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
		verified_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)
}

func createUserTable(t *testing.T, db *gorm.DB) {
	mustExec(t, db, `CREATE TABLE users (
		id TEXT PRIMARY KEY,
		email TEXT,
		name TEXT,
		role TEXT,
		kyc_status TEXT,
		kyc_verified_at DATETIME,
		password_hash TEXT,
		is_email_verified BOOLEAN,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)
}

func createAPIKeyTable(t *testing.T, db *gorm.DB) {
	mustExec(t, db, `CREATE TABLE api_keys (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		key_prefix TEXT NOT NULL,
		key_hash TEXT NOT NULL UNIQUE,
		secret_encrypted TEXT NOT NULL,
		secret_masked TEXT NOT NULL,
		permissions TEXT NOT NULL,
		is_active BOOLEAN NOT NULL,
		last_used_at DATETIME,
		expires_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)
}

func createChainTables(t *testing.T, db *gorm.DB) {
	mustExec(t, db, `CREATE TABLE chains (
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
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)
	mustExec(t, db, `CREATE TABLE chain_rpcs (
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
}

func createWalletTable(t *testing.T, db *gorm.DB) {
	mustExec(t, db, `CREATE TABLE wallets (
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
}

func createTokenTable(t *testing.T, db *gorm.DB) {
	mustExec(t, db, `CREATE TABLE tokens (
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

func createBridgeAndFeeTables(t *testing.T, db *gorm.DB) {
	mustExec(t, db, `CREATE TABLE bridge_configs (
		id TEXT PRIMARY KEY,
		bridge_id TEXT NOT NULL,
		source_chain_id TEXT NOT NULL,
		dest_chain_id TEXT NOT NULL,
		router_address TEXT,
		fee_percentage TEXT,
		config TEXT,
		is_active BOOLEAN,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)
	mustExec(t, db, `CREATE TABLE fee_configs (
		id TEXT PRIMARY KEY,
		chain_id TEXT NOT NULL,
		token_id TEXT NOT NULL,
		platform_fee_percent TEXT,
		fixed_base_fee TEXT,
		min_fee TEXT,
		max_fee TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)
}

func createRoutePolicyTables(t *testing.T, db *gorm.DB) {
	mustExec(t, db, `CREATE TABLE route_policies (
		id TEXT PRIMARY KEY,
		source_chain_id TEXT NOT NULL,
		dest_chain_id TEXT NOT NULL,
		default_bridge_type INTEGER,
		fallback_mode TEXT,
		fallback_order TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)
	mustExec(t, db, `CREATE TABLE layerzero_configs (
		id TEXT PRIMARY KEY,
		source_chain_id TEXT NOT NULL,
		dest_chain_id TEXT NOT NULL,
		dst_e_id INTEGER NOT NULL,
		dst_eid INTEGER,
		peer_hex TEXT NOT NULL,
		options_hex TEXT NOT NULL,
		is_active BOOLEAN,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)
}

func createPaymentTables(t *testing.T, db *gorm.DB) {
	mustExec(t, db, `CREATE TABLE payments (
		id TEXT PRIMARY KEY,
		sender_id TEXT NOT NULL,
		merchant_id TEXT,
		bridge_id TEXT,
		source_chain_id TEXT NOT NULL,
		dest_chain_id TEXT NOT NULL,
		source_token_id TEXT NOT NULL,
		dest_token_id TEXT NOT NULL,
		source_amount TEXT NOT NULL,
		dest_amount TEXT,
		fee_amount TEXT,
		total_charged TEXT,
		sender_address TEXT,
		dest_address TEXT,
		status TEXT NOT NULL,
		source_tx_hash TEXT,
		dest_tx_hash TEXT,
		refund_tx_hash TEXT,
		cross_chain_message_id TEXT,
		expires_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)
	mustExec(t, db, `CREATE TABLE payment_events (
		id TEXT PRIMARY KEY,
		payment_id TEXT NOT NULL,
		event_type TEXT NOT NULL,
		chain_id TEXT,
		chain TEXT,
		tx_hash TEXT,
		block_number INTEGER,
		metadata TEXT,
		created_at DATETIME
	);`)
}

func createSmartContractTable(t *testing.T, db *gorm.DB) {
	mustExec(t, db, `CREATE TABLE smart_contracts (
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
}

func createPaymentRequestTables(t *testing.T, db *gorm.DB) {
	mustExec(t, db, `CREATE TABLE payment_requests (
		id TEXT PRIMARY KEY,
		merchant_id TEXT NOT NULL,
		chain_id TEXT NOT NULL,
		token_id TEXT NOT NULL,
		wallet_address TEXT NOT NULL,
		token_address TEXT NOT NULL,
		amount TEXT NOT NULL,
		decimals INTEGER NOT NULL,
		description TEXT,
		status TEXT NOT NULL,
		expires_at DATETIME NOT NULL,
		tx_hash TEXT,
		payer_address TEXT,
		completed_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)
	mustExec(t, db, `CREATE TABLE background_jobs (
		id TEXT PRIMARY KEY,
		job_type TEXT NOT NULL,
		payload TEXT NOT NULL,
		status TEXT NOT NULL,
		attempts INTEGER DEFAULT 0,
		max_attempts INTEGER NOT NULL,
		scheduled_at DATETIME,
		started_at DATETIME,
		completed_at DATETIME,
		error_message TEXT,
		created_at DATETIME,
		updated_at DATETIME
	);`)
}
