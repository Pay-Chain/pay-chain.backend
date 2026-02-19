-- Normalize chains.chain_id values that are stored as CAIP-2 (e.g. eip155:8453 -> 8453)
-- Skip rows that would conflict with an existing normalized chain_id.
WITH candidates AS (
    SELECT
        c.id,
        c.chain_id AS old_chain_id,
        split_part(c.chain_id, ':', 2) AS normalized_chain_id
    FROM chains c
    WHERE c.deleted_at IS NULL
      AND position(':' in c.chain_id) > 0
      AND split_part(c.chain_id, ':', 2) <> ''
),
safe_updates AS (
    SELECT cand.id, cand.normalized_chain_id
    FROM candidates cand
    WHERE NOT EXISTS (
        SELECT 1
        FROM chains c2
        WHERE c2.deleted_at IS NULL
          AND c2.id <> cand.id
          AND c2.chain_id = cand.normalized_chain_id
    )
)
UPDATE chains c
SET chain_id = su.normalized_chain_id
FROM safe_updates su
WHERE c.id = su.id;

-- Bootstrap route policies for all active source->destination pairs that do not have one yet.
-- Default bridge type uses current heuristic:
--   - EVM -> EVM : CCIP (1)
--   - Others     : Hyperbridge (0)
INSERT INTO route_policies (
    id,
    source_chain_id,
    dest_chain_id,
    default_bridge_type,
    fallback_mode,
    fallback_order,
    created_at,
    updated_at
)
SELECT
    uuid_generate_v7(),
    src.id,
    dst.id,
    CASE
        WHEN src.type::text = 'EVM' AND dst.type::text = 'EVM' THEN 1
        ELSE 0
    END AS default_bridge_type,
    'strict',
    CASE
        WHEN src.type::text = 'EVM' AND dst.type::text = 'EVM' THEN '[1]'::jsonb
        ELSE '[0]'::jsonb
    END AS fallback_order,
    NOW(),
    NOW()
FROM chains src
CROSS JOIN chains dst
WHERE src.deleted_at IS NULL
  AND dst.deleted_at IS NULL
  AND src.is_active = TRUE
  AND dst.is_active = TRUE
  AND src.id <> dst.id
  AND NOT EXISTS (
      SELECT 1
      FROM route_policies rp
      WHERE rp.source_chain_id = src.id
        AND rp.dest_chain_id = dst.id
        AND rp.deleted_at IS NULL
  );
