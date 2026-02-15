DROP INDEX IF EXISTS idx_layerzero_configs_deleted_at;
DROP INDEX IF EXISTS idx_layerzero_configs_is_active;
DROP INDEX IF EXISTS idx_layerzero_configs_dest_chain_id;
DROP INDEX IF EXISTS idx_layerzero_configs_source_chain_id;
DROP INDEX IF EXISTS idx_layerzero_configs_unique_active;
DROP TABLE IF EXISTS layerzero_configs;

DROP INDEX IF EXISTS idx_route_policies_deleted_at;
DROP INDEX IF EXISTS idx_route_policies_default_bridge_type;
DROP INDEX IF EXISTS idx_route_policies_dest_chain_id;
DROP INDEX IF EXISTS idx_route_policies_source_chain_id;
DROP INDEX IF EXISTS idx_route_policies_unique_active;
DROP TABLE IF EXISTS route_policies;
