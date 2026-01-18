-- Migration: 000003_add_new_features
-- Description: Add RPC endpoints, payment requests, and wallet KYC fields

-- RPC Endpoints for chain fallback
CREATE TABLE rpc_endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chain_id INT NOT NULL REFERENCES chains(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    priority INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_error_at TIMESTAMP,
    error_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(chain_id, url)
);

-- Index for quick lookup by chain
CREATE INDEX idx_rpc_endpoints_chain_priority ON rpc_endpoints(chain_id, priority) WHERE is_active = true;

-- Payment Requests (merchant creates, payer pays)
CREATE TYPE payment_request_status AS ENUM ('pending', 'completed', 'expired', 'cancelled');

CREATE TABLE payment_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id UUID NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    chain_id TEXT NOT NULL,  -- CAIP-2 format: namespace:chainId
    token_address TEXT NOT NULL,
    amount TEXT NOT NULL,  -- Amount in smallest unit
    decimals INT NOT NULL,
    description TEXT,
    status payment_request_status NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMP NOT NULL,
    tx_hash TEXT,
    payer_address TEXT,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Indexes for payment requests
CREATE INDEX idx_payment_requests_merchant ON payment_requests(merchant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_payment_requests_status ON payment_requests(status) WHERE deleted_at IS NULL AND status = 'pending';
CREATE INDEX idx_payment_requests_expires_at ON payment_requests(expires_at) WHERE status = 'pending';

-- Add KYC fields to wallets table
ALTER TABLE wallets ADD COLUMN IF NOT EXISTS kyc_verified BOOLEAN DEFAULT false;
ALTER TABLE wallets ADD COLUMN IF NOT EXISTS kyc_verified_at TIMESTAMP;
ALTER TABLE wallets ADD COLUMN IF NOT EXISTS kyc_required BOOLEAN DEFAULT true;

-- Update existing wallets to not require KYC (grandfathered in)
UPDATE wallets SET kyc_verified = true, kyc_required = false WHERE kyc_verified IS NULL;

-- Background job tracking for payment request expiry
CREATE TABLE background_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending', -- pending, processing, completed, failed
    attempts INT DEFAULT 0,
    max_attempts INT DEFAULT 3,
    scheduled_at TIMESTAMP NOT NULL,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_background_jobs_scheduled ON background_jobs(scheduled_at) WHERE status = 'pending';
CREATE INDEX idx_background_jobs_type ON background_jobs(job_type, status);
