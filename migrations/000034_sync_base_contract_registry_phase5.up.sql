-- Phase 5 registry sync:
-- Ensure Base active contracts for ROUTER/GATEWAY/ADAPTER_HYPERBRIDGE
-- point to latest deployment addresses and ABI required-function maps.

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
END $$;

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
            (
                'ROUTER',
                'PaymentKitaRouter',
                '2.0.0',
                '0x304185d7B5Eb9790Dc78805D2095612F7a43A291',
                $$[
                  {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct BridgeMessage","name":"msgData","type":"tuple"}],"name":"quotePaymentFeeSafe","outputs":[{"internalType":"bool","name":"ok","type":"bool"},{"internalType":"uint256","name":"fee","type":"uint256"},{"internalType":"string","name":"reason","type":"string"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct BridgeMessage","name":"msgData","type":"tuple"}],"name":"quotePaymentFee","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"hasAdapter","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"getAdapter","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"internalType":"address","name":"adapter","type":"address"}],"name":"registerAdapter","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct BridgeMessage","name":"msgData","type":"tuple"}],"name":"routePayment","outputs":[],"stateMutability":"payable","type":"function"}
                ]$$::jsonb
            ),
            (
                'GATEWAY',
                'PaymentKitaGateway',
                '2.0.0',
                '0xBaB8d97Fbdf6788BF40B01C096CFB2cC661ba642',
                $$[
                  {"inputs":[{"components":[{"internalType":"bytes","name":"destChainIdBytes","type":"bytes"},{"internalType":"bytes","name":"receiverBytes","type":"bytes"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"bridgeTokenSource","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amountInSource","type":"uint256"},{"internalType":"uint256","name":"minBridgeAmountOut","type":"uint256"},{"internalType":"uint256","name":"minDestAmountOut","type":"uint256"},{"internalType":"uint8","name":"mode","type":"uint8"},{"internalType":"uint8","name":"bridgeOption","type":"uint8"}],"internalType":"struct PaymentRequestV2","name":"request","type":"tuple"}],"name":"createPayment","outputs":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"}],"stateMutability":"payable","type":"function"},
                  {"inputs":[{"components":[{"internalType":"bytes","name":"destChainIdBytes","type":"bytes"},{"internalType":"bytes","name":"receiverBytes","type":"bytes"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"bridgeTokenSource","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amountInSource","type":"uint256"},{"internalType":"uint256","name":"minBridgeAmountOut","type":"uint256"},{"internalType":"uint256","name":"minDestAmountOut","type":"uint256"},{"internalType":"uint8","name":"mode","type":"uint8"},{"internalType":"uint8","name":"bridgeOption","type":"uint8"}],"internalType":"struct PaymentRequestV2","name":"request","type":"tuple"}],"name":"createPaymentDefaultBridge","outputs":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"}],"stateMutability":"payable","type":"function"},
                  {"inputs":[{"components":[{"internalType":"bytes","name":"destChainIdBytes","type":"bytes"},{"internalType":"bytes","name":"receiverBytes","type":"bytes"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"bridgeTokenSource","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amountInSource","type":"uint256"},{"internalType":"uint256","name":"minBridgeAmountOut","type":"uint256"},{"internalType":"uint256","name":"minDestAmountOut","type":"uint256"},{"internalType":"uint8","name":"mode","type":"uint8"},{"internalType":"uint8","name":"bridgeOption","type":"uint8"}],"internalType":"struct PaymentRequestV2","name":"request","type":"tuple"},{"components":[{"internalType":"bytes32","name":"intentId","type":"bytes32"},{"internalType":"address","name":"stealthReceiver","type":"address"}],"internalType":"struct PrivateRouting","name":"privacy","type":"tuple"}],"name":"createPaymentPrivate","outputs":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"}],"stateMutability":"payable","type":"function"},
                  {"inputs":[{"components":[{"internalType":"bytes","name":"destChainIdBytes","type":"bytes"},{"internalType":"bytes","name":"receiverBytes","type":"bytes"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"bridgeTokenSource","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amountInSource","type":"uint256"},{"internalType":"uint256","name":"minBridgeAmountOut","type":"uint256"},{"internalType":"uint256","name":"minDestAmountOut","type":"uint256"},{"internalType":"uint8","name":"mode","type":"uint8"},{"internalType":"uint8","name":"bridgeOption","type":"uint8"}],"internalType":"struct PaymentRequestV2","name":"request","type":"tuple"}],"name":"previewApproval","outputs":[{"internalType":"address","name":"approvalToken","type":"address"},{"internalType":"uint256","name":"approvalAmount","type":"uint256"},{"internalType":"uint256","name":"requiredNativeFee","type":"uint256"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"setDefaultBridgeType","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"defaultBridgeTypes","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"components":[{"internalType":"bytes","name":"destChainIdBytes","type":"bytes"},{"internalType":"bytes","name":"receiverBytes","type":"bytes"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"bridgeTokenSource","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amountInSource","type":"uint256"},{"internalType":"uint256","name":"minBridgeAmountOut","type":"uint256"},{"internalType":"uint256","name":"minDestAmountOut","type":"uint256"},{"internalType":"uint8","name":"mode","type":"uint8"},{"internalType":"uint8","name":"bridgeOption","type":"uint8"}],"internalType":"struct PaymentRequestV2","name":"request","type":"tuple"}],"name":"quotePaymentCost","outputs":[{"internalType":"uint256","name":"platformFeeToken","type":"uint256"},{"internalType":"uint256","name":"bridgeFeeNative","type":"uint256"},{"internalType":"uint256","name":"totalSourceTokenRequired","type":"uint256"},{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"internalType":"bool","name":"bridgeQuoteOK","type":"bool"},{"internalType":"string","name":"bridgeQuoteReason","type":"string"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"string","name":"reason","type":"string"}],"name":"adapterFailAndRefund","outputs":[],"stateMutability":"nonpayable","type":"function"}
                ]$$::jsonb
            ),
            (
                'ADAPTER_HYPERBRIDGE',
                'HyperbridgeSender',
                '2.0.0',
                '0x6709C0dF1a2a015B3C34d6C7a04a185fbAc4740a',
                $$[
                  {"inputs":[{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct BridgeMessage","name":"message","type":"tuple"}],"name":"quoteFee","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct BridgeMessage","name":"message","type":"tuple"}],"name":"sendMessage","outputs":[],"stateMutability":"payable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"bytes","name":"stateMachineId","type":"bytes"}],"name":"setStateMachineId","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"bytes","name":"destination","type":"bytes"}],"name":"setDestinationContract","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"isChainConfigured","outputs":[{"internalType":"bool","name":"configured","type":"bool"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"stateMachineIds","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationContracts","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"address","name":"router","type":"address"}],"name":"setSwapRouter","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"swapper","type":"address"}],"name":"setSwapper","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"uint64","name":"timeout","type":"uint64"}],"name":"setDefaultTimeout","outputs":[],"stateMutability":"nonpayable","type":"function"}
                ]$$::jsonb
            )
    ) AS t(type, name, version, address, abi)
),
deactivate_by_type AS (
    UPDATE smart_contracts sc
    SET is_active = FALSE,
        updated_at = NOW()
    FROM base_chain bc, target_contracts tc
    WHERE sc.chain_id = bc.id
      AND sc.deleted_at IS NULL
      AND sc.type = tc.type
),
reactivate_existing AS (
    UPDATE smart_contracts sc
    SET name = tc.name,
        version = tc.version,
        abi = tc.abi,
        is_active = TRUE,
        updated_at = NOW()
    FROM base_chain bc, target_contracts tc
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
    '{}',
    NOW(),
    NOW()
FROM base_chain bc
CROSS JOIN target_contracts tc
WHERE NOT EXISTS (
    SELECT 1
    FROM reactivate_existing re
    WHERE re.chain_id = bc.id
      AND re.type = tc.type
      AND re.address_lc = LOWER(tc.address)
);
