-- Phase 6 canary policy:
-- Enable Hyperbridge Token Gateway (bridge type 3) for Base <-> Arbitrum lanes.

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM chains
        WHERE deleted_at IS NULL
          AND chain_id IN ('8453', 'eip155:8453')
    ) THEN
        RAISE EXCEPTION 'Base chain (8453) not found in chains table';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM chains
        WHERE deleted_at IS NULL
          AND chain_id IN ('42161', 'eip155:42161')
    ) THEN
        RAISE EXCEPTION 'Arbitrum chain (42161) not found in chains table';
    END IF;
END $$;

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
INSERT INTO route_policies (
    id,
    source_chain_id,
    dest_chain_id,
    default_bridge_type,
    fallback_mode,
    fallback_order,
    supports_token_bridge,
    supports_dest_swap,
    supports_privacy_forward,
    bridge_token,
    status,
    created_at,
    updated_at
)
SELECT
    uuid_generate_v7(),
    tp.source_chain_id,
    tp.dest_chain_id,
    3,
    'strict',
    '[3,1,2,0]'::jsonb,
    TRUE,
    TRUE,
    TRUE,
    '0x8d010bf9C26881788b4e6bf5Fd1bdC358c8F90b8',
    'active',
    NOW(),
    NOW()
FROM target_policies tp
ON CONFLICT (source_chain_id, dest_chain_id) WHERE deleted_at IS NULL
DO UPDATE SET
    default_bridge_type = EXCLUDED.default_bridge_type,
    fallback_mode = EXCLUDED.fallback_mode,
    fallback_order = EXCLUDED.fallback_order,
    supports_token_bridge = EXCLUDED.supports_token_bridge,
    supports_dest_swap = EXCLUDED.supports_dest_swap,
    supports_privacy_forward = EXCLUDED.supports_privacy_forward,
    bridge_token = EXCLUDED.bridge_token,
    status = EXCLUDED.status,
    updated_at = NOW();
