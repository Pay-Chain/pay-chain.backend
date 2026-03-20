-- 000045_sync_current_stargate_senders.up.sql
-- Sync active Stargate sender registry rows to current runtime addresses.

DO $$
DECLARE
    base_chain_id UUID;
    polygon_chain_id UUID;
    arbitrum_chain_id UUID;
    target_row_id UUID;
BEGIN
    SELECT id INTO base_chain_id FROM chains WHERE chain_id IN ('8453', 'eip155:8453') AND deleted_at IS NULL LIMIT 1;
    SELECT id INTO polygon_chain_id FROM chains WHERE chain_id IN ('137', 'eip155:137') AND deleted_at IS NULL LIMIT 1;
    SELECT id INTO arbitrum_chain_id FROM chains WHERE chain_id IN ('42161', 'eip155:42161') AND deleted_at IS NULL LIMIT 1;

    -- Base
    UPDATE smart_contracts
    SET
        is_active = FALSE,
        metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object(
            'superseded_by_runtime_sync', TRUE,
            'superseded_at', NOW()
        ),
        updated_at = NOW()
    WHERE chain_id = base_chain_id
      AND type = 'ADAPTER_STARGATE'
      AND deleted_at IS NULL
      AND is_active = TRUE
      AND LOWER(address) <> LOWER('0x44D10404d8e078af761e71c03d97cec30EE0a2A3');

    UPDATE smart_contracts
    SET
        is_active = TRUE,
        version = '2.3.0',
        metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object(
            'source', 'CHAIN_BASE.md',
            'runtime', 'current',
            'sync_reason', 'current_stargate_sender_runtime'
        ),
        updated_at = NOW()
    WHERE chain_id = base_chain_id
      AND type = 'ADAPTER_STARGATE'
      AND deleted_at IS NULL
      AND LOWER(address) = LOWER('0x44D10404d8e078af761e71c03d97cec30EE0a2A3');

    IF NOT FOUND THEN
        SELECT id INTO target_row_id
        FROM smart_contracts
        WHERE chain_id = base_chain_id
          AND type = 'ADAPTER_STARGATE'
          AND deleted_at IS NULL
        ORDER BY is_active DESC, updated_at DESC, created_at DESC
        LIMIT 1;

        IF target_row_id IS NOT NULL THEN
            UPDATE smart_contracts
            SET
                address = '0x44D10404d8e078af761e71c03d97cec30EE0a2A3',
                is_active = TRUE,
                version = '2.3.0',
                metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object(
                    'source', 'CHAIN_BASE.md',
                    'runtime', 'current',
                    'sync_reason', 'current_stargate_sender_runtime'
                ),
                updated_at = NOW()
            WHERE id = target_row_id;
        END IF;
    END IF;

    -- Polygon
    UPDATE smart_contracts
    SET
        is_active = FALSE,
        metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object(
            'superseded_by_runtime_sync', TRUE,
            'superseded_at', NOW()
        ),
        updated_at = NOW()
    WHERE chain_id = polygon_chain_id
      AND type = 'ADAPTER_STARGATE'
      AND deleted_at IS NULL
      AND is_active = TRUE
      AND LOWER(address) <> LOWER('0x8B6c7770D4B8AaD2d600e0cf5df3Eea5Bc0EB0fe');

    UPDATE smart_contracts
    SET
        is_active = TRUE,
        version = '2.3.0',
        metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object(
            'source', 'CHAIN_POLYGON.md',
            'runtime', 'current',
            'sync_reason', 'current_stargate_sender_runtime'
        ),
        updated_at = NOW()
    WHERE chain_id = polygon_chain_id
      AND type = 'ADAPTER_STARGATE'
      AND deleted_at IS NULL
      AND LOWER(address) = LOWER('0x8B6c7770D4B8AaD2d600e0cf5df3Eea5Bc0EB0fe');

    IF NOT FOUND THEN
        SELECT id INTO target_row_id
        FROM smart_contracts
        WHERE chain_id = polygon_chain_id
          AND type = 'ADAPTER_STARGATE'
          AND deleted_at IS NULL
        ORDER BY is_active DESC, updated_at DESC, created_at DESC
        LIMIT 1;

        IF target_row_id IS NOT NULL THEN
            UPDATE smart_contracts
            SET
                address = '0x8B6c7770D4B8AaD2d600e0cf5df3Eea5Bc0EB0fe',
                is_active = TRUE,
                version = '2.3.0',
                metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object(
                    'source', 'CHAIN_POLYGON.md',
                    'runtime', 'current',
                    'sync_reason', 'current_stargate_sender_runtime'
                ),
                updated_at = NOW()
            WHERE id = target_row_id;
        END IF;
    END IF;

    -- Arbitrum
    UPDATE smart_contracts
    SET
        is_active = FALSE,
        metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object(
            'superseded_by_runtime_sync', TRUE,
            'superseded_at', NOW()
        ),
        updated_at = NOW()
    WHERE chain_id = arbitrum_chain_id
      AND type = 'ADAPTER_STARGATE'
      AND deleted_at IS NULL
      AND is_active = TRUE
      AND LOWER(address) <> LOWER('0x2843e9880D7a29499e025C6E4ce749f127f6bD8e');

    UPDATE smart_contracts
    SET
        is_active = TRUE,
        version = '2.3.0',
        metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object(
            'source', 'CHAIN_ARBITRUM.md',
            'runtime', 'current',
            'sync_reason', 'current_stargate_sender_runtime'
        ),
        updated_at = NOW()
    WHERE chain_id = arbitrum_chain_id
      AND type = 'ADAPTER_STARGATE'
      AND deleted_at IS NULL
      AND LOWER(address) = LOWER('0x2843e9880D7a29499e025C6E4ce749f127f6bD8e');

    IF NOT FOUND THEN
        SELECT id INTO target_row_id
        FROM smart_contracts
        WHERE chain_id = arbitrum_chain_id
          AND type = 'ADAPTER_STARGATE'
          AND deleted_at IS NULL
        ORDER BY is_active DESC, updated_at DESC, created_at DESC
        LIMIT 1;

        IF target_row_id IS NOT NULL THEN
            UPDATE smart_contracts
            SET
                address = '0x2843e9880D7a29499e025C6E4ce749f127f6bD8e',
                is_active = TRUE,
                version = '2.3.0',
                metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object(
                    'source', 'CHAIN_ARBITRUM.md',
                    'runtime', 'current',
                    'sync_reason', 'current_stargate_sender_runtime'
                ),
                updated_at = NOW()
            WHERE id = target_row_id;
        END IF;
    END IF;
END $$;
