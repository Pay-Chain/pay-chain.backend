-- 000001_init.up.sql
-- Pay-Chain Database Schema

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create ENUM types
CREATE TYPE user_role_enum AS ENUM (
    'admin',
    'sub_admin',
    'partner',
    'user'
);

CREATE TYPE kyc_status_enum AS ENUM (
    'not_started',
    'id_card_verified',
    'face_verified',
    'liveness_verified',
    'fully_verified'
);

CREATE TYPE merchant_type_enum AS ENUM (
    'partner',
    'corporate',
    'umkm',
    'retail'
);

CREATE TYPE merchant_status_enum AS ENUM (
    'pending',
    'active',
    'suspended',
    'rejected'
);

CREATE TYPE payment_status_enum AS ENUM (
    'pending',
    'processing',
    'completed',
    'failed',
    'refunded'
);

CREATE TYPE chain_type_enum AS ENUM (
    'EVM',
    'SVM',
    'SUBSTRATE'
);

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role user_role_enum DEFAULT 'user',
    kyc_status kyc_status_enum DEFAULT 'not_started',
    kyc_verified_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NULL;

-- Email verifications table
CREATE TABLE email_verifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    token VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    verified_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_email_verifications_token ON email_verifications(token);

-- Password resets table
CREATE TABLE password_resets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    token VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Merchants table
CREATE TABLE merchants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID UNIQUE NOT NULL REFERENCES users(id),
    business_name VARCHAR(255) NOT NULL,
    business_email VARCHAR(255) NOT NULL,
    merchant_type merchant_type_enum NOT NULL,
    status merchant_status_enum DEFAULT 'pending',
    tax_id VARCHAR(50),
    business_address TEXT,
    documents JSONB,
    fee_discount_percent DECIMAL(5,2) DEFAULT 0,
    verified_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_merchants_user_id ON merchants(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_merchants_status ON merchants(status) WHERE deleted_at IS NULL;

-- Merchant applications table (audit trail)
CREATE TABLE merchant_applications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    merchant_id UUID NOT NULL REFERENCES merchants(id),
    action VARCHAR(50) NOT NULL,
    performed_by UUID REFERENCES users(id),
    reason TEXT,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Chains table
CREATE TABLE chains (
    id INTEGER PRIMARY KEY,
    namespace VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,
    chain_type chain_type_enum NOT NULL,
    rpc_url TEXT NOT NULL,
    explorer_url TEXT,
    contract_address VARCHAR(66),
    ccip_router_address VARCHAR(66),
    hyperbridge_gateway VARCHAR(66),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_chains_active ON chains(is_active) WHERE deleted_at IS NULL;

-- Tokens table
CREATE TABLE tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    symbol VARCHAR(10) NOT NULL,
    name VARCHAR(100) NOT NULL,
    decimals INTEGER NOT NULL,
    logo_url TEXT,
    is_stablecoin BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Supported tokens table
CREATE TABLE supported_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chain_id INTEGER NOT NULL REFERENCES chains(id),
    token_id UUID NOT NULL REFERENCES tokens(id),
    contract_address VARCHAR(66) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    min_amount DECIMAL(36,18) DEFAULT 0,
    max_amount DECIMAL(36,18),
    created_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP,
    UNIQUE(chain_id, token_id)
);

CREATE INDEX idx_supported_tokens_chain ON supported_tokens(chain_id) WHERE deleted_at IS NULL;

-- Wallets table
CREATE TABLE wallets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id),
    merchant_id UUID REFERENCES merchants(id),
    chain_id INTEGER NOT NULL REFERENCES chains(id),
    address VARCHAR(66) NOT NULL,
    is_primary BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_wallets_user_id ON wallets(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_wallets_address ON wallets(address) WHERE deleted_at IS NULL;

-- Payments table
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sender_id UUID NOT NULL REFERENCES users(id),
    merchant_id UUID REFERENCES merchants(id),
    receiver_wallet_id UUID NOT NULL REFERENCES wallets(id),
    source_token_id UUID NOT NULL REFERENCES supported_tokens(id),
    dest_token_id UUID NOT NULL REFERENCES supported_tokens(id),
    source_amount DECIMAL(36,18) NOT NULL,
    dest_amount DECIMAL(36,18),
    fee_amount DECIMAL(36,18) NOT NULL,
    total_charged DECIMAL(36,18) NOT NULL,
    bridge_type VARCHAR(20) NOT NULL,
    status payment_status_enum NOT NULL DEFAULT 'pending',
    source_tx_hash VARCHAR(66),
    dest_tx_hash VARCHAR(66),
    refund_tx_hash VARCHAR(66),
    cross_chain_message_id VARCHAR(100),
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_payments_sender ON payments(sender_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_payments_merchant ON payments(merchant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_payments_status ON payments(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_payments_created ON payments(created_at DESC) WHERE deleted_at IS NULL;

-- Payment events table
CREATE TABLE payment_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    payment_id UUID NOT NULL REFERENCES payments(id),
    event_type VARCHAR(50) NOT NULL,
    chain VARCHAR(20) NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_payment_events_payment ON payment_events(payment_id);

-- Bridge configs table
CREATE TABLE bridge_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bridge_type VARCHAR(20) NOT NULL,
    source_chain_id INTEGER NOT NULL REFERENCES chains(id),
    dest_chain_id INTEGER NOT NULL REFERENCES chains(id),
    router_address VARCHAR(66) NOT NULL,
    fee_percentage DECIMAL(5,4) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    config JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP,
    UNIQUE(bridge_type, source_chain_id, dest_chain_id)
);

-- Fee configs table
CREATE TABLE fee_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chain_id INTEGER REFERENCES chains(id),
    token_id UUID REFERENCES tokens(id),
    platform_fee_percent DECIMAL(5,4) NOT NULL DEFAULT 0.003,
    fixed_base_fee DECIMAL(36,18) NOT NULL DEFAULT 0.5,
    min_fee DECIMAL(36,18) DEFAULT 0,
    max_fee DECIMAL(36,18),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Merchant fee tiers table
CREATE TABLE merchant_fee_tiers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    merchant_type merchant_type_enum NOT NULL UNIQUE,
    fee_discount_percent DECIMAL(5,2) NOT NULL,
    min_monthly_volume DECIMAL(36,18) NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Insert default merchant fee tiers
INSERT INTO merchant_fee_tiers (merchant_type, fee_discount_percent, min_monthly_volume) VALUES
    ('partner', 50.00, 100000),
    ('corporate', 30.00, 50000),
    ('umkm', 15.00, 5000),
    ('retail', 0.00, 0);

-- Insert Phase 1 testnet chains
INSERT INTO chains (id, namespace, name, chain_type, rpc_url, explorer_url, is_active) VALUES
    (84532, 'eip155', 'Base Sepolia', 'EVM', 'https://sepolia.base.org', 'https://sepolia.basescan.org', true),
    (97, 'eip155', 'BSC Sepolia', 'EVM', 'https://data-seed-prebsc-1-s1.binance.org:8545', 'https://testnet.bscscan.com', true);

-- Insert common stablecoins
INSERT INTO tokens (symbol, name, decimals, is_stablecoin) VALUES
    ('USDT', 'Tether USD', 6, true),
    ('USDC', 'USD Coin', 6, true),
    ('DAI', 'Dai Stablecoin', 18, true);
