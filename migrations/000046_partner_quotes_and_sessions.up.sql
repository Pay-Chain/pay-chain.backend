CREATE TABLE payment_quotes (
    id UUID PRIMARY KEY,
    merchant_id UUID NOT NULL,
    invoice_currency VARCHAR(32) NOT NULL,
    invoice_amount NUMERIC(78, 0) NOT NULL,
    selected_chain_id VARCHAR(64) NOT NULL,
    selected_token_address VARCHAR(128) NOT NULL,
    selected_token_symbol VARCHAR(32) NOT NULL,
    selected_token_decimals INTEGER NOT NULL,
    quoted_amount NUMERIC(78, 0) NOT NULL,
    quote_rate NUMERIC(78, 18) NOT NULL,
    price_source VARCHAR(128) NOT NULL,
    route TEXT NOT NULL,
    slippage_bps INTEGER NOT NULL DEFAULT 0,
    rate_timestamp TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    status VARCHAR(32) NOT NULL,
    used_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_payment_quotes_status_expires_at
    ON payment_quotes(status, expires_at);

CREATE INDEX idx_payment_quotes_merchant_created_at
    ON payment_quotes(merchant_id, created_at DESC);

CREATE TABLE partner_payment_sessions (
    id UUID PRIMARY KEY,
    merchant_id UUID NOT NULL,
    quote_id UUID NULL,
    invoice_currency VARCHAR(32) NOT NULL,
    invoice_amount NUMERIC(78, 0) NOT NULL,
    selected_chain_id VARCHAR(64) NOT NULL,
    selected_token_address VARCHAR(128) NOT NULL,
    selected_token_symbol VARCHAR(32) NOT NULL,
    selected_token_decimals INTEGER NOT NULL,
    dest_chain VARCHAR(64) NOT NULL,
    dest_token VARCHAR(128) NOT NULL,
    dest_wallet VARCHAR(128) NOT NULL,
    payment_amount NUMERIC(78, 0) NOT NULL,
    payment_amount_decimals INTEGER NOT NULL,
    status VARCHAR(32) NOT NULL,
    channel_used VARCHAR(32) NULL,
    payment_code TEXT NOT NULL,
    payment_url TEXT NOT NULL,
    instruction_to VARCHAR(128) NULL,
    instruction_value VARCHAR(128) NULL,
    instruction_data_hex TEXT NULL,
    instruction_data_base58 TEXT NULL,
    instruction_data_base64 TEXT NULL,
    quote_rate NUMERIC(78, 18) NULL,
    quote_source VARCHAR(128) NULL,
    quote_route TEXT NULL,
    quote_expires_at TIMESTAMP NULL,
    quote_snapshot_json JSONB NULL,
    expires_at TIMESTAMP NOT NULL,
    paid_tx_hash VARCHAR(128) NULL,
    paid_chain_id VARCHAR(64) NULL,
    paid_token_address VARCHAR(128) NULL,
    paid_amount NUMERIC(78, 0) NULL,
    paid_sender_address VARCHAR(128) NULL,
    completed_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_partner_payment_sessions_status_expires_at
    ON partner_payment_sessions(status, expires_at);

CREATE INDEX idx_partner_payment_sessions_merchant_created_at
    ON partner_payment_sessions(merchant_id, created_at DESC);

CREATE INDEX idx_partner_payment_sessions_quote_id
    ON partner_payment_sessions(quote_id);
