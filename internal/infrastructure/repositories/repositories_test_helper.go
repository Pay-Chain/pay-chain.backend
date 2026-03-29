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
}

func createMerchantSettlementProfileTable(t *testing.T, db *gorm.DB) {
	mustExec(t, db, `CREATE TABLE merchant_settlement_profiles (
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
		ccip_chain_selector TEXT,
		stargate_eid INTEGER,
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
		deleted_at DATETIME,
		CONSTRAINT fk_chains_rpcs FOREIGN KEY (chain_id) REFERENCES chains(id)
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
		supports_token_bridge BOOLEAN,
		supports_dest_swap BOOLEAN,
		supports_privacy_forward BOOLEAN,
		bridge_token TEXT,
		status TEXT,
		per_byte_rate TEXT,
		overhead_bytes TEXT,
		min_fee TEXT,
		max_fee TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`)
	mustExec(t, db, `CREATE TABLE stargate_configs (
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
		min_dest_amount TEXT,
		total_charged TEXT,
		sender_address TEXT,
		dest_address TEXT,
		status TEXT NOT NULL,
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

func createPartnerFlowTables(t *testing.T, db *gorm.DB) {
	mustExec(t, db, `CREATE TABLE payment_quotes (
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
	mustExec(t, db, `CREATE TABLE partner_payment_sessions (
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
		instruction_approval_to TEXT,
		instruction_approval_data_hex TEXT,
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
	mustExec(t, db, `CREATE TABLE chains (
		id TEXT PRIMARY KEY,
		chain_id TEXT NOT NULL,
		name TEXT,
		type TEXT,
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
	mustExec(t, db, `CREATE TABLE tokens (
		id TEXT PRIMARY KEY,
		chain_id TEXT NOT NULL,
		symbol TEXT,
		name TEXT,
		decimals INTEGER,
		address TEXT,
		type TEXT,
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
	mustExec(t, db, `CREATE TABLE payment_requests (
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

func createResolveAuditTable(t *testing.T, db *gorm.DB) {
	mustExec(t, db, `CREATE TABLE pk_resolve_audit (
		id TEXT PRIMARY KEY,
		merchant_id TEXT,
		user_id TEXT,
		action TEXT NOT NULL,
		details TEXT,
		ip_address TEXT,
		user_agent TEXT,
		created_at DATETIME
	);`)
}
