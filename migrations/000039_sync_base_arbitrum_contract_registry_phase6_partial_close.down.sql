-- Rollback for phase 6 partial-close registry sync:
-- disable the target addresses from 000039 and reactivate the latest previous
-- address per type for Base and Arbitrum.

WITH base_chain AS (
    SELECT id
    FROM chains
    WHERE deleted_at IS NULL
      AND chain_id IN ('8453', 'eip155:8453')
    ORDER BY updated_at DESC NULLS LAST, created_at DESC NULLS LAST
    LIMIT 1
),
base_target_contracts AS (
    SELECT *
    FROM (
        VALUES
            ('TOKEN_REGISTRY', '0x140fbAA1e8BE387082aeb6088E4Ffe1bf3Ba4d4f'),
            ('VAULT', '0x67d0af7f163F45578679eDa4BDf9042E3E5FEc60'),
            ('ROUTER', '0x1b91B56aD3aA6B35e5EAe18EE19A42574A545802'),
            ('GATEWAY', '0x08409b0fa63b0bCEb4c4B49DBf286ff943b60011'),
            ('TOKEN_SWAPPER', '0x8B6c7770D4B8AaD2d600e0cf5df3Eea5Bc0EB0fe'),
            ('ADAPTER_CCIP', '0x47FEA6C20aC5F029BAB99Ec2ed756d94c54707DE'),
            ('ADAPTER_HYPERBRIDGE', '0xB9F0429D420571923EeC57E8b7025d063E361329'),
            ('ADAPTER_STARGATE', '0x11bfD843dCEbF421d2f2A07D2C8BA5Db85E501E9'),
            ('RECEIVER_STARGATE', '0xc4c28aeeE5bb312970a7266461838565E1eEEc1a'),
            ('PRIVACY_MODULE', '0xd8a6818468eBB65527118308B48c1A969977A086'),
            ('STEALTH_ESCROW_FACTORY', '0x882A5d22d27C2e60dA7356DCdEA49bE3bCFbcBA3'),
            ('FEE_POLICY_MANAGER', '0x1443C7D4dbB86035739A69fBB39Ebb76Ba7590fc'),
            ('FEE_STRATEGY_DEFAULT_V1', '0x53689F9119345480C7b16B085b27F93A826b65CA')
    ) AS t(type, address)
),
base_disable AS (
    UPDATE smart_contracts sc
    SET is_active = FALSE,
        updated_at = NOW()
    FROM base_chain bc, base_target_contracts tc
    WHERE sc.chain_id = bc.id
      AND sc.deleted_at IS NULL
      AND sc.type = tc.type
      AND LOWER(sc.address) = LOWER(tc.address)
),
base_candidate_prev AS (
    SELECT DISTINCT ON (sc.type)
        sc.id,
        sc.type
    FROM smart_contracts sc
    JOIN base_chain bc ON sc.chain_id = bc.id
    JOIN base_target_contracts tc ON sc.type = tc.type
    WHERE sc.deleted_at IS NULL
      AND LOWER(sc.address) <> LOWER(tc.address)
    ORDER BY sc.type, sc.updated_at DESC NULLS LAST, sc.created_at DESC NULLS LAST, sc.id DESC
)
UPDATE smart_contracts sc
SET is_active = TRUE,
    updated_at = NOW()
FROM base_candidate_prev cp
WHERE sc.id = cp.id;

WITH arbitrum_chain AS (
    SELECT id
    FROM chains
    WHERE deleted_at IS NULL
      AND chain_id IN ('42161', 'eip155:42161')
    ORDER BY updated_at DESC NULLS LAST, created_at DESC NULLS LAST
    LIMIT 1
),
arbitrum_target_contracts AS (
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
            ('ADAPTER_STARGATE', '0x64505be2844d35284ab58984f93dceb21bc77464'),
            ('RECEIVER_STARGATE', '0x0c6c2cc9c2fb42d2fe591f2c3fee4db428090ad4'),
            ('PRIVACY_MODULE', '0x678fa4e50ed898e2c5694399651ea80894164766'),
            ('STEALTH_ESCROW_FACTORY', '0x703d53d548ef860902057226079bc842bf077d1c'),
            ('FEE_POLICY_MANAGER', '0x5bd6093f455534dfd5c0220f5ba6660d5dbb30a8'),
            ('FEE_STRATEGY_DEFAULT_V1', '0x62ccb9fbbd975d41210b367f5bc1b6da00f71610')
    ) AS t(type, address)
),
arbitrum_disable AS (
    UPDATE smart_contracts sc
    SET is_active = FALSE,
        updated_at = NOW()
    FROM arbitrum_chain ac, arbitrum_target_contracts tc
    WHERE sc.chain_id = ac.id
      AND sc.deleted_at IS NULL
      AND sc.type = tc.type
      AND LOWER(sc.address) = LOWER(tc.address)
),
arbitrum_candidate_prev AS (
    SELECT DISTINCT ON (sc.type)
        sc.id,
        sc.type
    FROM smart_contracts sc
    JOIN arbitrum_chain ac ON sc.chain_id = ac.id
    JOIN arbitrum_target_contracts tc ON sc.type = tc.type
    WHERE sc.deleted_at IS NULL
      AND LOWER(sc.address) <> LOWER(tc.address)
    ORDER BY sc.type, sc.updated_at DESC NULLS LAST, sc.created_at DESC NULLS LAST, sc.id DESC
)
UPDATE smart_contracts sc
SET is_active = TRUE,
    updated_at = NOW()
FROM arbitrum_candidate_prev cp
WHERE sc.id = cp.id;
