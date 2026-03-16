-- 000041_sync_stargate_rescuable_adapters.down.sql
-- Rollback Stargate adapters to previous state (not fully possible to original without backup, but can mark as inactive)

UPDATE smart_contracts 
SET is_active = FALSE, updated_at = NOW() 
WHERE type IN ('ADAPTER_STARGATE', 'RECEIVER_STARGATE')
AND metadata->>'sync_reason' = 'rescuable_v2_migration';

-- Re-activation of old ones would require knowledge of their addresses, which is not easily retrievable in a stateless DP migration block.
