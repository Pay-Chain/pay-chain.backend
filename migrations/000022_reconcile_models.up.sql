-- Add api_keys table for Dual Auth system
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id),
    name VARCHAR(100) NOT NULL,
    key_prefix VARCHAR(20) NOT NULL,
    key_hash VARCHAR(64) UNIQUE NOT NULL,
    secret_encrypted TEXT NOT NULL,
    secret_masked VARCHAR(20) NOT NULL,
    permissions JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_used_at TIMESTAMP,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Add support for soft deletes in chain_rpcs
ALTER TABLE chain_rpcs ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;
