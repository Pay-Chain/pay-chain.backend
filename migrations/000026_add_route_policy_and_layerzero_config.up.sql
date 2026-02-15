CREATE TABLE IF NOT EXISTS route_policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    source_chain_id UUID NOT NULL REFERENCES chains(id) ON DELETE CASCADE,
    dest_chain_id UUID NOT NULL REFERENCES chains(id) ON DELETE CASCADE,
    default_bridge_type SMALLINT NOT NULL DEFAULT 0,
    fallback_mode VARCHAR(32) NOT NULL DEFAULT 'strict',
    fallback_order JSONB NOT NULL DEFAULT '[0]'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_route_policies_unique_active
    ON route_policies (source_chain_id, dest_chain_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_route_policies_source_chain_id ON route_policies(source_chain_id);
CREATE INDEX IF NOT EXISTS idx_route_policies_dest_chain_id ON route_policies(dest_chain_id);
CREATE INDEX IF NOT EXISTS idx_route_policies_default_bridge_type ON route_policies(default_bridge_type);
CREATE INDEX IF NOT EXISTS idx_route_policies_deleted_at ON route_policies(deleted_at);

CREATE TABLE IF NOT EXISTS layerzero_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    source_chain_id UUID NOT NULL REFERENCES chains(id) ON DELETE CASCADE,
    dest_chain_id UUID NOT NULL REFERENCES chains(id) ON DELETE CASCADE,
    dst_eid INTEGER NOT NULL,
    peer_hex VARCHAR(66) NOT NULL,
    options_hex TEXT NOT NULL DEFAULT '0x',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_layerzero_configs_unique_active
    ON layerzero_configs (source_chain_id, dest_chain_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_layerzero_configs_source_chain_id ON layerzero_configs(source_chain_id);
CREATE INDEX IF NOT EXISTS idx_layerzero_configs_dest_chain_id ON layerzero_configs(dest_chain_id);
CREATE INDEX IF NOT EXISTS idx_layerzero_configs_is_active ON layerzero_configs(is_active);
CREATE INDEX IF NOT EXISTS idx_layerzero_configs_deleted_at ON layerzero_configs(deleted_at);
