-- Rollback Phase 6 canary route policy for Base <-> Arbitrum.
-- Revert to default EVM baseline (bridge type 1) and disable token-bridge flags.

WITH base_chain AS (
    SELECT id
    FROM chains
    WHERE deleted_at IS NULL
      AND chain_id IN ('8453', 'eip155:8453')
    ORDER BY updated_at DESC NULLS LAST, created_at DESC NULLS LAST
    LIMIT 1
),
arb_chain AS (
    SELECT id
    FROM chains
    WHERE deleted_at IS NULL
      AND chain_id IN ('42161', 'eip155:42161')
    ORDER BY updated_at DESC NULLS LAST, created_at DESC NULLS LAST
    LIMIT 1
),
target_policies AS (
    SELECT
        src.id AS source_chain_id,
        dst.id AS dest_chain_id
    FROM base_chain src
    CROSS JOIN arb_chain dst
    UNION ALL
    SELECT
        src.id AS source_chain_id,
        dst.id AS dest_chain_id
    FROM arb_chain src
    CROSS JOIN base_chain dst
)
UPDATE route_policies rp
SET
    default_bridge_type = 1,
    fallback_mode = 'strict',
    fallback_order = '[1]'::jsonb,
    supports_token_bridge = FALSE,
    supports_dest_swap = FALSE,
    supports_privacy_forward = FALSE,
    bridge_token = NULL,
    status = 'active',
    updated_at = NOW()
FROM target_policies tp
WHERE rp.deleted_at IS NULL
  AND rp.source_chain_id = tp.source_chain_id
  AND rp.dest_chain_id = tp.dest_chain_id;
