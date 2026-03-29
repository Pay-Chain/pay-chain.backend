-- Sync gateway runtime module contracts into smart_contracts registry
-- Sources:
-- - CHAIN_BASE.md
-- - CHAIN_ARBITRUM.md

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM chains
        WHERE deleted_at IS NULL AND chain_id IN ('8453', 'eip155:8453')
    ) THEN
        RAISE EXCEPTION 'Base chain (8453) not found in chains table';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM chains
        WHERE deleted_at IS NULL AND chain_id IN ('42161', 'eip155:42161')
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
base_target_contracts AS (
    SELECT *
    FROM (
        VALUES
            (
                'GATEWAY_VALIDATOR_MODULE',
                'GatewayValidatorModule',
                '2.1.0',
                '0xb7A893672189B46632109CC15De8986e2B8be1c6',
                $$[
                  {"inputs":[{"internalType":"address","name":"tokenRegistry","type":"address"},{"internalType":"bytes","name":"receiverBytes","type":"bytes"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"bool","name":"requireSourceTokenSupported","type":"bool"},{"internalType":"bool","name":"requireDestTokenSupported","type":"bool"}],"name":"validateCreate","outputs":[{"internalType":"address","name":"receiver","type":"address"}],"stateMutability":"view","type":"function"}
                ]$$::jsonb
            ),
            (
                'GATEWAY_QUOTE_MODULE',
                'GatewayQuoteModule',
                '2.1.0',
                '0xfc70c24D9dC932572A067349E4D3A2eeb0280b31',
                $$[
                  {"inputs":[{"internalType":"address","name":"router","type":"address"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct IBridgeAdapter.BridgeMessage","name":"message","type":"tuple"}],"name":"quotePaymentFeeSafe","outputs":[{"internalType":"bool","name":"ok","type":"bool"},{"internalType":"uint256","name":"feeNative","type":"uint256"},{"internalType":"string","name":"reason","type":"string"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"address","name":"router","type":"address"},{"internalType":"address","name":"swapper","type":"address"},{"internalType":"bool","name":"enableSourceSideSwap","type":"bool"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"bytes","name":"receiverBytes","type":"bytes"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amountInSource","type":"uint256"},{"internalType":"uint256","name":"minDestAmountOut","type":"uint256"},{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"internalType":"address","name":"bridgeTokenSource","type":"address"}],"name":"quoteBridgeForV2","outputs":[{"internalType":"bool","name":"ok","type":"bool"},{"internalType":"uint256","name":"feeNative","type":"uint256"},{"internalType":"string","name":"reason","type":"string"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"address","name":"router","type":"address"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"bytes","name":"receiverBytes","type":"bytes"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"quoteBridgeForV1","outputs":[{"internalType":"bool","name":"ok","type":"bool"},{"internalType":"uint256","name":"feeNative","type":"uint256"},{"internalType":"string","name":"reason","type":"string"}],"stateMutability":"view","type":"function"}
                ]$$::jsonb
            ),
            (
                'GATEWAY_EXECUTION_MODULE',
                'GatewayExecutionModule',
                '2.1.0',
                '0x5852Bec7f3Ce38Ffdd8d1c9F48c88a620a9e6078',
                $$[
                  {"inputs":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"internalType":"uint256","name":"providedNativeFee","type":"uint256"},{"internalType":"uint256","name":"requiredNativeFee","type":"uint256"}],"name":"beforeRoute","outputs":[],"stateMutability":"pure","type":"function"},
                  {"inputs":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"token","type":"address"},{"internalType":"uint256","name":"settledAmount","type":"uint256"}],"name":"onSameChainSettled","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"onIncomingFinalized","outputs":[],"stateMutability":"nonpayable","type":"function"}
                ]$$::jsonb
            )
    ) AS t(type, name, version, address, abi)
),
base_deactivate AS (
    UPDATE smart_contracts sc
    SET is_active = FALSE,
        updated_at = NOW()
    FROM base_chain bc, base_target_contracts tc
    WHERE sc.chain_id = bc.id
      AND sc.deleted_at IS NULL
      AND sc.type = tc.type
),
base_reactivate AS (
    UPDATE smart_contracts sc
    SET name = tc.name,
        version = tc.version,
        abi = tc.abi,
        is_active = TRUE,
        metadata = jsonb_build_object(
            'source', 'CHAIN_BASE.md',
            'sync_reason', 'gateway_runtime_modules_registry_sync',
            'phase', 'post_phase6'
        ),
        updated_at = NOW()
    FROM base_chain bc, base_target_contracts tc
    WHERE sc.chain_id = bc.id
      AND sc.deleted_at IS NULL
      AND sc.type = tc.type
      AND LOWER(sc.address) = LOWER(tc.address)
    RETURNING sc.chain_id, sc.type, LOWER(sc.address) AS address_lc
)
INSERT INTO smart_contracts (
    id,
    name,
    chain_id,
    address,
    abi,
    type,
    version,
    deployer_address,
    is_active,
    metadata,
    created_at,
    updated_at
)
SELECT
    uuid_generate_v7(),
    tc.name,
    bc.id,
    tc.address,
    tc.abi,
    tc.type,
    tc.version,
    '',
    TRUE,
    jsonb_build_object(
        'source', 'CHAIN_BASE.md',
        'sync_reason', 'gateway_runtime_modules_registry_sync',
        'phase', 'post_phase6'
    ),
    NOW(),
    NOW()
FROM base_chain bc
CROSS JOIN base_target_contracts tc
WHERE NOT EXISTS (
    SELECT 1
    FROM base_reactivate br
    WHERE br.chain_id = bc.id
      AND br.type = tc.type
      AND br.address_lc = LOWER(tc.address)
);

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
            (
                'GATEWAY_VALIDATOR_MODULE',
                'GatewayValidatorModule',
                '2.1.0',
                '0xaf65342c8d1b42650b88d737ce5b630f5487f7f0',
                $$[
                  {"inputs":[{"internalType":"address","name":"tokenRegistry","type":"address"},{"internalType":"bytes","name":"receiverBytes","type":"bytes"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"bool","name":"requireSourceTokenSupported","type":"bool"},{"internalType":"bool","name":"requireDestTokenSupported","type":"bool"}],"name":"validateCreate","outputs":[{"internalType":"address","name":"receiver","type":"address"}],"stateMutability":"view","type":"function"}
                ]$$::jsonb
            ),
            (
                'GATEWAY_QUOTE_MODULE',
                'GatewayQuoteModule',
                '2.1.0',
                '0x6917d003add05eef125f3630fdae759c47f308bb',
                $$[
                  {"inputs":[{"internalType":"address","name":"router","type":"address"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct IBridgeAdapter.BridgeMessage","name":"message","type":"tuple"}],"name":"quotePaymentFeeSafe","outputs":[{"internalType":"bool","name":"ok","type":"bool"},{"internalType":"uint256","name":"feeNative","type":"uint256"},{"internalType":"string","name":"reason","type":"string"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"address","name":"router","type":"address"},{"internalType":"address","name":"swapper","type":"address"},{"internalType":"bool","name":"enableSourceSideSwap","type":"bool"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"bytes","name":"receiverBytes","type":"bytes"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amountInSource","type":"uint256"},{"internalType":"uint256","name":"minDestAmountOut","type":"uint256"},{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"internalType":"address","name":"bridgeTokenSource","type":"address"}],"name":"quoteBridgeForV2","outputs":[{"internalType":"bool","name":"ok","type":"bool"},{"internalType":"uint256","name":"feeNative","type":"uint256"},{"internalType":"string","name":"reason","type":"string"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"address","name":"router","type":"address"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"bytes","name":"receiverBytes","type":"bytes"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"quoteBridgeForV1","outputs":[{"internalType":"bool","name":"ok","type":"bool"},{"internalType":"uint256","name":"feeNative","type":"uint256"},{"internalType":"string","name":"reason","type":"string"}],"stateMutability":"view","type":"function"}
                ]$$::jsonb
            ),
            (
                'GATEWAY_EXECUTION_MODULE',
                'GatewayExecutionModule',
                '2.1.0',
                '0x62763108cd44c86c9b588f4defc2c66790fef34b',
                $$[
                  {"inputs":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"internalType":"uint256","name":"providedNativeFee","type":"uint256"},{"internalType":"uint256","name":"requiredNativeFee","type":"uint256"}],"name":"beforeRoute","outputs":[],"stateMutability":"pure","type":"function"},
                  {"inputs":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"token","type":"address"},{"internalType":"uint256","name":"settledAmount","type":"uint256"}],"name":"onSameChainSettled","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"onIncomingFinalized","outputs":[],"stateMutability":"nonpayable","type":"function"}
                ]$$::jsonb
            )
    ) AS t(type, name, version, address, abi)
),
arbitrum_deactivate AS (
    UPDATE smart_contracts sc
    SET is_active = FALSE,
        updated_at = NOW()
    FROM arbitrum_chain ac, arbitrum_target_contracts tc
    WHERE sc.chain_id = ac.id
      AND sc.deleted_at IS NULL
      AND sc.type = tc.type
),
arbitrum_reactivate AS (
    UPDATE smart_contracts sc
    SET name = tc.name,
        version = tc.version,
        abi = tc.abi,
        is_active = TRUE,
        metadata = jsonb_build_object(
            'source', 'CHAIN_ARBITRUM.md',
            'sync_reason', 'gateway_runtime_modules_registry_sync',
            'phase', 'post_phase6'
        ),
        updated_at = NOW()
    FROM arbitrum_chain ac, arbitrum_target_contracts tc
    WHERE sc.chain_id = ac.id
      AND sc.deleted_at IS NULL
      AND sc.type = tc.type
      AND LOWER(sc.address) = LOWER(tc.address)
    RETURNING sc.chain_id, sc.type, LOWER(sc.address) AS address_lc
)
INSERT INTO smart_contracts (
    id,
    name,
    chain_id,
    address,
    abi,
    type,
    version,
    deployer_address,
    is_active,
    metadata,
    created_at,
    updated_at
)
SELECT
    uuid_generate_v7(),
    tc.name,
    ac.id,
    tc.address,
    tc.abi,
    tc.type,
    tc.version,
    '',
    TRUE,
    jsonb_build_object(
        'source', 'CHAIN_ARBITRUM.md',
        'sync_reason', 'gateway_runtime_modules_registry_sync',
        'phase', 'post_phase6'
    ),
    NOW(),
    NOW()
FROM arbitrum_chain ac
CROSS JOIN arbitrum_target_contracts tc
WHERE NOT EXISTS (
    SELECT 1
    FROM arbitrum_reactivate ar
    WHERE ar.chain_id = ac.id
      AND ar.type = tc.type
      AND ar.address_lc = LOWER(tc.address)
);
