-- 000045_sync_current_stargate_senders.down.sql
-- Revert active Stargate sender registry rows to the previous sender addresses from 000041.

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
        address = '0x1F746d1130d161413e0BC5598801798c402331d7',
        version = '2.2.0',
        updated_at = NOW()
    WHERE chain_id = base_chain_id
      AND type = 'ADAPTER_STARGATE'
      AND is_active = TRUE;

    UPDATE smart_contracts
    SET
        address = '0x838Ba4E44E24f4d9A655698df535F404448aA2A9',
        version = '2.2.0',
        updated_at = NOW()
    WHERE chain_id = polygon_chain_id
      AND type = 'ADAPTER_STARGATE'
      AND is_active = TRUE;

    UPDATE smart_contracts
    SET
        address = '0x64976A3cDE870507B269FD4A8aC2dC9993bc3F3A',
        version = '2.2.0',
        updated_at = NOW()
    WHERE chain_id = arbitrum_chain_id
      AND type = 'ADAPTER_STARGATE'
      AND is_active = TRUE;
END $$;
