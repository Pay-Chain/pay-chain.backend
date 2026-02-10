-- Phase 3: Blockchain & Tokens
-- Refactor Chains, Tokens, and Wallets to use UUID v7 and new schema.
-- Correctly handles the state from Migration 000014 (where chains.id is UUID and network_id is the integer ID).
-- Update 1: chain_id is VARCHAR to support Solana/SVM as per ERD.
-- Update 2: smart_contracts_old uses contract_address, not address.

-- 1. Rename existing tables (Idempotent)
DO $$
BEGIN
    -- chains
    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'chains') THEN
        IF NOT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'chains_old') THEN
            ALTER TABLE chains RENAME TO chains_old;
        END IF;
    END IF;

    -- tokens
    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'tokens') THEN
        IF NOT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'tokens_old') THEN
            ALTER TABLE tokens RENAME TO tokens_old;
        END IF;
    END IF;

    -- supported_tokens
    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'supported_tokens') THEN
        IF NOT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'supported_tokens_old') THEN
            ALTER TABLE supported_tokens RENAME TO supported_tokens_old;
        END IF;
    END IF;

    -- wallets
    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'wallets') THEN
        IF NOT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'wallets_old') THEN
            ALTER TABLE wallets RENAME TO wallets_old;
        END IF;
    END IF;
    
    -- smart_contracts (if exists)
    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'smart_contracts') THEN
         IF NOT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'smart_contracts_old') THEN
            ALTER TABLE smart_contracts RENAME TO smart_contracts_old;
        END IF;
    END IF;
END $$;

-- 2. Create New Tables

-- Chains
CREATE TABLE IF NOT EXISTS chains (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    chain_id VARCHAR(255) UNIQUE NOT NULL, -- String ID from network (e.g. "1", "5eykt...")
    name VARCHAR(255) NOT NULL,
    type chain_type_enum NOT NULL,
    image_url VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    is_testnet BOOLEAN DEFAULT FALSE,
    currency_symbol VARCHAR(20),
    explorer_url VARCHAR(255),
    rpc_url VARCHAR(255), -- Main RPC
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Chain RPCs
CREATE TABLE IF NOT EXISTS chain_rpcs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    chain_id UUID NOT NULL REFERENCES chains(id),
    url VARCHAR(255) NOT NULL,
    priority INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Tokens (Merged Concept)
CREATE TABLE IF NOT EXISTS tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    chain_id UUID NOT NULL REFERENCES chains(id),
    name VARCHAR(255) NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    decimals INTEGER NOT NULL,
    type token_type_enum NOT NULL DEFAULT 'ERC20',
    address VARCHAR(255),
    logo_url VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    is_native BOOLEAN DEFAULT FALSE,
    min_amount DECIMAL(36, 18) DEFAULT 0,
    max_amount DECIMAL(36, 18),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Wallets
CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID REFERENCES users(id),
    merchant_id UUID REFERENCES merchants(id),
    chain_id UUID NOT NULL REFERENCES chains(id),
    address VARCHAR(255) NOT NULL,
    type VARCHAR(50) DEFAULT 'EOA',
    is_primary BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Smart Contracts
CREATE TABLE IF NOT EXISTS smart_contracts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    chain_id UUID NOT NULL REFERENCES chains(id),
    name VARCHAR(255) NOT NULL,
    address VARCHAR(255) NOT NULL,
    type VARCHAR(50),
    abi JSONB,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);


-- 3. Data Migration

-- Migrate Chains
-- Map old 'id' (UUID due to mig 14) to new 'id'.
-- Map old 'network_id' (string) to new 'chain_id' (String).
INSERT INTO chains (id, chain_id, name, type, rpc_url, explorer_url, is_active, created_at)
SELECT 
    id, -- Preserve exiting UUID PK
    network_id, -- Keep as String
    name, 
    UPPER(chain_type::text)::chain_type_enum, 
    rpc_url, 
    explorer_url, 
    is_active, 
    created_at
FROM chains_old
ON CONFLICT (chain_id) DO NOTHING;

-- Populate Chain RPCs
INSERT INTO chain_rpcs (chain_id, url)
SELECT id, rpc_url FROM chains_old
ON CONFLICT DO NOTHING;

-- Migrate Tokens
INSERT INTO tokens (chain_id, name, symbol, decimals, type, address, logo_url, is_active, min_amount, max_amount, created_at, is_native)
SELECT
    st.chain_id, -- Direct UUID copy
    t.name,
    t.symbol,
    t.decimals,
    'ERC20'::token_type_enum,
    st.contract_address,
    t.logo_url,
    st.is_active,
    st.min_amount,
    st.max_amount,
    st.created_at,
    CASE WHEN t.symbol IN ('ETH', 'MATIC', 'BNB', 'SOL') THEN TRUE ELSE FALSE END
FROM supported_tokens_old st
JOIN tokens_old t ON st.token_id = t.id
ON CONFLICT DO NOTHING;

-- Update Token Types
UPDATE tokens SET type = 'NATIVE' WHERE is_native = TRUE;

-- Migrate Wallets
INSERT INTO wallets (id, user_id, merchant_id, chain_id, address, is_primary, created_at)
SELECT
    w.id, -- Preserve UUID
    w.user_id,
    w.merchant_id,
    w.chain_id, -- Direct UUID copy
    w.address,
    w.is_primary,
    w.created_at
FROM wallets_old w
ON CONFLICT (id) DO NOTHING;

-- Migrate Smart Contracts
INSERT INTO smart_contracts (chain_id, name, address, created_at)
SELECT
    -- Attempt to find the matching Chain ID. 
    -- Old `sc.chain_id` is "eip155:84532". New `chains.chain_id` is "84532".
    -- We need to split the string or match vaguely.
    c.id, 
    sc.name,
    sc.contract_address, -- Fixed column name
    sc.created_at
FROM smart_contracts_old sc
JOIN chains c ON sc.chain_id LIKE '%' || c.chain_id -- Vague match: "eip155:84532" LIKE "%84532"
ON CONFLICT DO NOTHING;
