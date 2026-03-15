-- Rollback for Arbitrum full registry sync:
-- 1) Disable target addresses from 000037 up migration.
-- 2) Reactivate latest previous address per type on Arbitrum (if any).

WITH arbitrum_chain AS (
    SELECT id
    FROM chains
    WHERE deleted_at IS NULL
      AND chain_id IN ('42161', 'eip155:42161')
    ORDER BY updated_at DESC NULLS LAST, created_at DESC NULLS LAST
    LIMIT 1
),
target_contracts AS (
    SELECT *
    FROM (
        VALUES
            ('TOKEN_REGISTRY', '0x53f1e35fea4b2cdc7e73feb4e36365c88569ebf0'),
            ('VAULT', '0x4a92d4079853c78df38b4bbd574aa88679adef93'),
            ('ROUTER', '0x3722374b187e5400f4423dbc45ad73784604d275'),
            ('GATEWAY', '0x259294aecdc0006b73b1281c30440a8179cff44c'),
            ('TOKEN_SWAPPER', '0x5d86bfd5a361bc652bc596dd2a77cd2bdba2bf35'),
            ('ADAPTER_CCIP', '0x5cce8cdfb77dccd28ed7cf0acf567f92d737abd9'),
            ('ADAPTER_HYPERBRIDGE', '0xfdc7986e73f91ebc08130ba2325d32b23f844e26'),
            ('ADAPTER_LAYERZERO', '0x64505be2844d35284ab58984f93dceb21bc77464'),
            ('RECEIVER_LAYERZERO', '0x0c6c2cc9c2fb42d2fe591f2c3fee4db428090ad4'),
            ('PRIVACY_MODULE', '0x678fa4e50ed898e2c5694399651ea80894164766'),
            ('STEALTH_ESCROW_FACTORY', '0x703d53d548ef860902057226079bc842bf077d1c'),
            ('FEE_POLICY_MANAGER', '0x5bd6093f455534dfd5c0220f5ba6660d5dbb30a8'),
            ('FEE_STRATEGY_DEFAULT_V1', '0x62ccb9fbbd975d41210b367f5bc1b6da00f71610')
    ) AS t(type, address)
),
disable_target AS (
    UPDATE smart_contracts sc
    SET is_active = FALSE,
        updated_at = NOW()
    FROM arbitrum_chain ac, target_contracts tc
    WHERE sc.chain_id = ac.id
      AND sc.deleted_at IS NULL
      AND sc.type = tc.type
      AND LOWER(sc.address) = LOWER(tc.address)
),
candidate_prev AS (
    SELECT DISTINCT ON (sc.type)
        sc.id,
        sc.type
    FROM smart_contracts sc
    JOIN arbitrum_chain ac ON sc.chain_id = ac.id
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
