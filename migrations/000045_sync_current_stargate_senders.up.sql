-- 000045_sync_current_stargate_senders.up.sql
-- Sync active Stargate sender registry rows to current runtime addresses.

DO $$
DECLARE
    base_chain_id UUID;
    polygon_chain_id UUID;
    arbitrum_chain_id UUID;
BEGIN
    SELECT id INTO base_chain_id FROM chains WHERE chain_id IN ('8453', 'eip155:8453') AND deleted_at IS NULL LIMIT 1;
    SELECT id INTO polygon_chain_id FROM chains WHERE chain_id IN ('137', 'eip155:137') AND deleted_at IS NULL LIMIT 1;
    SELECT id INTO arbitrum_chain_id FROM chains WHERE chain_id IN ('42161', 'eip155:42161') AND deleted_at IS NULL LIMIT 1;

    UPDATE smart_contracts
    SET
        address = '0x44D10404d8e078af761e71c03d97cec30EE0a2A3',
        version = '2.3.0',
        metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object(
            'source', 'CHAIN_BASE.md',
            'runtime', 'current',
            'sync_reason', 'current_stargate_sender_runtime'
        ),
        updated_at = NOW()
    WHERE chain_id = base_chain_id
      AND type = 'ADAPTER_STARGATE'
      AND is_active = TRUE;

    UPDATE smart_contracts
    SET
        address = '0x8B6c7770D4B8AaD2d600e0cf5df3Eea5Bc0EB0fe',
        version = '2.3.0',
        metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object(
            'source', 'CHAIN_POLYGON.md',
            'runtime', 'current',
            'sync_reason', 'current_stargate_sender_runtime'
        ),
        updated_at = NOW()
    WHERE chain_id = polygon_chain_id
      AND type = 'ADAPTER_STARGATE'
      AND is_active = TRUE;

    UPDATE smart_contracts
    SET
        address = '0x2843e9880D7a29499e025C6E4ce749f127f6bD8e',
        version = '2.3.0',
        metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object(
            'source', 'CHAIN_ARBITRUM.md',
            'runtime', 'current',
            'sync_reason', 'current_stargate_sender_runtime'
        ),
        updated_at = NOW()
    WHERE chain_id = arbitrum_chain_id
      AND type = 'ADAPTER_STARGATE'
      AND is_active = TRUE;
END $$;
