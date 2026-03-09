-- Rollback for Phase 5 registry sync:
-- 1) Disable the target addresses from 000034 up migration.
-- 2) Reactivate latest previous address per type on Base (if any).

WITH base_chain AS (
    SELECT id
    FROM chains
    WHERE deleted_at IS NULL
      AND chain_id IN ('8453', 'eip155:8453')
    ORDER BY updated_at DESC NULLS LAST, created_at DESC NULLS LAST
    LIMIT 1
),
target_contracts AS (
    SELECT *
    FROM (
        VALUES
            ('ROUTER', '0x304185d7B5Eb9790Dc78805D2095612F7a43A291'),
            ('GATEWAY', '0xBaB8d97Fbdf6788BF40B01C096CFB2cC661ba642'),
            ('ADAPTER_HYPERBRIDGE', '0x6709C0dF1a2a015B3C34d6C7a04a185fbAc4740a')
    ) AS t(type, address)
),
disable_target AS (
    UPDATE smart_contracts sc
    SET is_active = FALSE,
        updated_at = NOW()
    FROM base_chain bc, target_contracts tc
    WHERE sc.chain_id = bc.id
      AND sc.deleted_at IS NULL
      AND sc.type = tc.type
      AND LOWER(sc.address) = LOWER(tc.address)
),
candidate_prev AS (
    SELECT DISTINCT ON (sc.type)
        sc.id,
        sc.type
    FROM smart_contracts sc
    JOIN base_chain bc ON sc.chain_id = bc.id
    JOIN target_contracts tc ON sc.type = tc.type
    WHERE sc.deleted_at IS NULL
      AND LOWER(sc.address) <> LOWER(tc.address)
    ORDER BY sc.type, sc.updated_at DESC NULLS LAST, sc.created_at DESC NULLS LAST, sc.id DESC
)
UPDATE smart_contracts sc
SET is_active = TRUE,
    updated_at = NOW()
FROM candidate_prev cp
WHERE sc.id = cp.id;
