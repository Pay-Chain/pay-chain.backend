CREATE TABLE IF NOT EXISTS merchant_settlement_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    merchant_id UUID NOT NULL UNIQUE REFERENCES merchants(id),
    invoice_currency VARCHAR(32) NOT NULL,
    dest_chain VARCHAR(64) NOT NULL,
    dest_token VARCHAR(128) NOT NULL,
    dest_wallet VARCHAR(128) NOT NULL,
    bridge_token_symbol VARCHAR(32) NOT NULL DEFAULT 'USDC',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_merchant_settlement_profiles_deleted_at
    ON merchant_settlement_profiles (deleted_at);
