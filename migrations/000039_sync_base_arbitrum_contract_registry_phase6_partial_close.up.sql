-- Phase 6 partial-close registry sync:
-- Align Base and Arbitrum active contract registry entries with the latest
-- chain documentation used for Phase 6 conditional sign-off.

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
                'TOKEN_REGISTRY',
                'TokenRegistry',
                '2.1.0',
                '0x140fbAA1e8BE387082aeb6088E4Ffe1bf3Ba4d4f',
                $$[
                  {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"bool","name":"supported","type":"bool"}],"name":"setTokenSupport","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"uint8","name":"decimals","type":"uint8"}],"name":"setTokenDecimals","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"token","type":"address"}],"name":"isTokenSupported","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"address","name":"token","type":"address"}],"name":"tokenDecimals","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"}
                ]$$::jsonb
            ),
            (
                'VAULT',
                'PaymentKitaVault',
                '2.1.0',
                '0x67d0af7f163F45578679eDa4BDf9042E3E5FEc60',
                $$[
                  {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"spender","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"approveSpender","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"spender","type":"address"},{"internalType":"bool","name":"authorized","type":"bool"}],"name":"setAuthorizedSpender","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"from","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"pullTokens","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"pushTokens","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"","type":"address"}],"name":"authorizedSpenders","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
                ]$$::jsonb
            ),
            (
                'ROUTER',
                'PaymentKitaRouter',
                '2.1.0',
                '0x1b91B56aD3aA6B35e5EAe18EE19A42574A545802',
                $$[
                  {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"internalType":"address","name":"adapter","type":"address"}],"name":"registerAdapter","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"getAdapter","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"hasAdapter","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"internalType":"uint8","name":"mode","type":"uint8"}],"name":"setBridgeMode","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"internalType":"bool","name":"enabled","type":"bool"}],"name":"setTokenBridgeDestSwapCapability","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"internalType":"bool","name":"enabled","type":"bool"}],"name":"setTokenBridgePrivacySettlementCapability","outputs":[],"stateMutability":"nonpayable","type":"function"}
                ]$$::jsonb
            ),
            (
                'GATEWAY',
                'PaymentKitaGateway',
                '2.1.0',
                '0x08409b0fa63b0bCEb4c4B49DBf286ff943b60011',
                $$[
                  {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"setDefaultBridgeType","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"adapter","type":"address"},{"internalType":"bool","name":"authorized","type":"bool"}],"name":"setAuthorizedAdapter","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"swapper","type":"address"}],"name":"setSwapper","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"bool","name":"enabled","type":"bool"}],"name":"setEnableSourceSideSwap","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[],"name":"router","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
                  {"inputs":[],"name":"vault","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
                  {"inputs":[],"name":"swapper","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
                  {"inputs":[],"name":"tokenRegistry","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
                  {"inputs":[],"name":"privacyModule","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
                  {"inputs":[],"name":"feePolicyManager","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"defaultBridgeTypes","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"}
                ]$$::jsonb
            ),
            (
                'TOKEN_SWAPPER',
                'TokenSwapper',
                '2.1.0',
                '0x8B6c7770D4B8AaD2d600e0cf5df3Eea5Bc0EB0fe',
                $$[
                  {"inputs":[{"internalType":"address","name":"vault_","type":"address"}],"name":"setVault","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"router","type":"address"}],"name":"setV3Router","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"caller","type":"address"},{"internalType":"bool","name":"allowed","type":"bool"}],"name":"setAuthorizedCaller","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"tokenIn","type":"address"},{"internalType":"address","name":"tokenOut","type":"address"},{"internalType":"uint24","name":"fee","type":"uint24"}],"name":"setV3Pool","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"tokenIn","type":"address"},{"internalType":"address","name":"tokenOut","type":"address"},{"internalType":"uint24","name":"fee","type":"uint24"},{"internalType":"int24","name":"tickSpacing","type":"int24"},{"internalType":"address","name":"hooks","type":"address"},{"internalType":"bytes","name":"hookData","type":"bytes"}],"name":"setDirectPool","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"tokenIn","type":"address"},{"internalType":"address","name":"tokenOut","type":"address"},{"internalType":"address[]","name":"path","type":"address[]"}],"name":"setMultiHopPath","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[],"name":"swapRouterV3","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"address","name":"","type":"address"}],"name":"authorizedCallers","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
                ]$$::jsonb
            ),
            (
                'ADAPTER_CCIP',
                'CCIPSender',
                '2.1.0',
                '0x47FEA6C20aC5F029BAB99Ec2ed756d94c54707DE',
                $$[
                  {"inputs":[{"internalType":"address","name":"caller","type":"address"},{"internalType":"bool","name":"allowed","type":"bool"}],"name":"setAuthorizedCaller","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct BridgeMessage","name":"message","type":"tuple"}],"name":"quoteFee","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct BridgeMessage","name":"message","type":"tuple"}],"name":"sendMessage","outputs":[],"stateMutability":"payable","type":"function"}
                ]$$::jsonb
            ),
            (
                'ADAPTER_HYPERBRIDGE',
                'HyperbridgeSender',
                '2.1.0',
                '0xB9F0429D420571923EeC57E8b7025d063E361329',
                $$[
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"bytes","name":"stateMachineId","type":"bytes"}],"name":"setStateMachineId","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"bytes","name":"destination","type":"bytes"}],"name":"setDestinationContract","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"swapper","type":"address"}],"name":"setSwapper","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct BridgeMessage","name":"message","type":"tuple"}],"name":"quoteFee","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct BridgeMessage","name":"message","type":"tuple"}],"name":"sendMessage","outputs":[],"stateMutability":"payable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"isChainConfigured","outputs":[{"internalType":"bool","name":"configured","type":"bool"}],"stateMutability":"view","type":"function"}
                ]$$::jsonb
            ),
            (
                'ADAPTER_LAYERZERO',
                'LayerZeroSenderAdapter',
                '2.1.0',
                '0x11bfD843dCEbF421d2f2A07D2C8BA5Db85E501E9',
                $$[
                  {"inputs":[],"name":"registerDelegate","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"uint32","name":"dstEid","type":"uint32"},{"internalType":"bytes32","name":"peer","type":"bytes32"}],"name":"setTrustedPeer","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"uint32","name":"dstEid","type":"uint32"},{"internalType":"uint256","name":"fee","type":"uint256"}],"name":"setMessageFee","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"uint32","name":"dstEid","type":"uint32"},{"internalType":"uint128","name":"gasLimit","type":"uint128"}],"name":"setEnforcedGas","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"isChainConfigured","outputs":[{"internalType":"bool","name":"configured","type":"bool"}],"stateMutability":"view","type":"function"}
                ]$$::jsonb
            ),
            (
                'RECEIVER_LAYERZERO',
                'LayerZeroReceiverAdapter',
                '2.1.0',
                '0xc4c28aeeE5bb312970a7266461838565E1eEEc1a',
                $$[
                  {"inputs":[{"internalType":"address","name":"swapper","type":"address"}],"name":"setSwapper","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"uint32","name":"srcEid","type":"uint32"},{"internalType":"bytes32","name":"peer","type":"bytes32"}],"name":"setTrustedPeer","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"uint32","name":"srcEid","type":"uint32"}],"name":"getTrustConfig","outputs":[{"internalType":"bool","name":"trusted","type":"bool"},{"internalType":"bytes32","name":"configuredPeer","type":"bytes32"},{"internalType":"uint64","name":"expectedNonce","type":"uint64"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"uint32","name":"","type":"uint32"}],"name":"peers","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"stateMutability":"view","type":"function"}
                ]$$::jsonb
            ),
            (
                'PRIVACY_MODULE',
                'GatewayPrivacyModule',
                '2.1.0',
                '0xd8a6818468eBB65527118308B48c1A969977A086',
                $$[
                  {"inputs":[{"internalType":"address","name":"gateway","type":"address"},{"internalType":"bool","name":"allowed","type":"bool"}],"name":"setAuthorizedGateway","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"gateway","type":"address"}],"name":"authorizedGateway","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"stealthReceiver","type":"address"},{"internalType":"address","name":"finalReceiver","type":"address"},{"internalType":"address","name":"token","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"address","name":"caller","type":"address"},{"internalType":"bool","name":"sameChain","type":"bool"}],"name":"forwardFromStealth","outputs":[],"stateMutability":"nonpayable","type":"function"}
                ]$$::jsonb
            ),
            (
                'STEALTH_ESCROW_FACTORY',
                'StealthEscrowFactory',
                '2.1.0',
                '0x882A5d22d27C2e60dA7356DCdEA49bE3bCFbcBA3',
                $$[
                  {"inputs":[{"internalType":"bytes32","name":"salt","type":"bytes32"},{"internalType":"address","name":"owner","type":"address"},{"internalType":"address","name":"forwarder","type":"address"}],"name":"deployEscrow","outputs":[{"internalType":"address","name":"escrow","type":"address"}],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"bytes32","name":"salt","type":"bytes32"},{"internalType":"address","name":"owner","type":"address"},{"internalType":"address","name":"forwarder","type":"address"}],"name":"predictEscrow","outputs":[{"internalType":"address","name":"predicted","type":"address"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"address","name":"candidate","type":"address"}],"name":"isEscrow","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
                ]$$::jsonb
            ),
            (
                'FEE_POLICY_MANAGER',
                'FeePolicyManager',
                '2.1.0',
                '0x1443C7D4dbB86035739A69fBB39Ebb76Ba7590fc',
                $$[
                  {"inputs":[{"internalType":"address","name":"strategy","type":"address"}],"name":"setDefaultStrategy","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"strategy","type":"address"}],"name":"setActiveStrategy","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[],"name":"clearActiveStrategy","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[],"name":"defaultStrategy","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
                  {"inputs":[],"name":"activeStrategy","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
                  {"inputs":[],"name":"resolveStrategy","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}
                ]$$::jsonb
            ),
            (
                'FEE_STRATEGY_DEFAULT_V1',
                'FeeStrategyDefaultV1',
                '2.1.0',
                '0x53689F9119345480C7b16B085b27F93A826b65CA',
                $$[
                  {"inputs":[{"internalType":"address","name":"registry","type":"address"}],"name":"setTokenRegistry","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[],"name":"tokenRegistry","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
                  {"inputs":[],"name":"FIXED_BASE_FEE","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
                  {"inputs":[],"name":"FEE_RATE_BPS","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}
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
        type = tc.type,
        version = tc.version,
        abi = tc.abi,
        is_active = TRUE,
        metadata = jsonb_build_object(
            'source', 'CHAIN_BASE.md',
            'sync_reason', 'phase6_partial_close',
            'phase6_status', 'conditionally_accepted'
        ),
        updated_at = NOW()
    FROM base_chain bc, base_target_contracts tc
    WHERE sc.chain_id = bc.id
      AND sc.deleted_at IS NULL
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
        'sync_reason', 'phase6_partial_close',
        'phase6_status', 'conditionally_accepted'
    ),
    NOW(),
    NOW()
FROM base_chain bc
CROSS JOIN base_target_contracts tc
WHERE NOT EXISTS (
    SELECT 1
    FROM base_reactivate re
    WHERE re.chain_id = bc.id
      AND re.type = tc.type
      AND re.address_lc = LOWER(tc.address)
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
            ('TOKEN_REGISTRY','TokenRegistry','2.1.0','0x53f1e35fea4b2cdc7e73feb4e36365c88569ebf0',$$[
              {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"bool","name":"supported","type":"bool"}],"name":"setTokenSupport","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"uint8","name":"decimals","type":"uint8"}],"name":"setTokenDecimals","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"address","name":"token","type":"address"}],"name":"isTokenSupported","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
              {"inputs":[{"internalType":"address","name":"token","type":"address"}],"name":"tokenDecimals","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"}
            ]$$::jsonb),
            ('VAULT','PaymentKitaVault','2.1.0','0x4a92d4079853c78df38b4bbd574aa88679adef93',$$[
              {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"spender","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"approveSpender","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"address","name":"spender","type":"address"},{"internalType":"bool","name":"authorized","type":"bool"}],"name":"setAuthorizedSpender","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"from","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"pullTokens","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"pushTokens","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"address","name":"","type":"address"}],"name":"authorizedSpenders","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
            ]$$::jsonb),
            ('ROUTER','PaymentKitaRouter','2.1.0','0x3722374b187e5400f4423dbc45ad73784604d275',$$[
              {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"internalType":"address","name":"adapter","type":"address"}],"name":"registerAdapter","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"getAdapter","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
              {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"hasAdapter","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
              {"inputs":[{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"internalType":"uint8","name":"mode","type":"uint8"}],"name":"setBridgeMode","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"internalType":"bool","name":"enabled","type":"bool"}],"name":"setTokenBridgeDestSwapCapability","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"internalType":"bool","name":"enabled","type":"bool"}],"name":"setTokenBridgePrivacySettlementCapability","outputs":[],"stateMutability":"nonpayable","type":"function"}
            ]$$::jsonb),
            ('GATEWAY','PaymentKitaGateway','2.1.0','0x259294aecdc0006b73b1281c30440a8179cff44c',$$[
              {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"setDefaultBridgeType","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"address","name":"adapter","type":"address"},{"internalType":"bool","name":"authorized","type":"bool"}],"name":"setAuthorizedAdapter","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"address","name":"swapper","type":"address"}],"name":"setSwapper","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"bool","name":"enabled","type":"bool"}],"name":"setEnableSourceSideSwap","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[],"name":"router","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
              {"inputs":[],"name":"vault","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
              {"inputs":[],"name":"swapper","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
              {"inputs":[],"name":"tokenRegistry","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
              {"inputs":[],"name":"privacyModule","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
              {"inputs":[],"name":"feePolicyManager","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
              {"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"defaultBridgeTypes","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"}
            ]$$::jsonb),
            ('TOKEN_SWAPPER','TokenSwapper','2.1.0','0x5d86bfd5a361bc652bc596dd2a77cd2bdba2bf35',$$[
              {"inputs":[{"internalType":"address","name":"vault_","type":"address"}],"name":"setVault","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"address","name":"router","type":"address"}],"name":"setV3Router","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"address","name":"caller","type":"address"},{"internalType":"bool","name":"allowed","type":"bool"}],"name":"setAuthorizedCaller","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"address","name":"tokenIn","type":"address"},{"internalType":"address","name":"tokenOut","type":"address"},{"internalType":"uint24","name":"fee","type":"uint24"}],"name":"setV3Pool","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"address","name":"tokenIn","type":"address"},{"internalType":"address","name":"tokenOut","type":"address"},{"internalType":"uint24","name":"fee","type":"uint24"},{"internalType":"int24","name":"tickSpacing","type":"int24"},{"internalType":"address","name":"hooks","type":"address"},{"internalType":"bytes","name":"hookData","type":"bytes"}],"name":"setDirectPool","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"address","name":"tokenIn","type":"address"},{"internalType":"address","name":"tokenOut","type":"address"},{"internalType":"address[]","name":"path","type":"address[]"}],"name":"setMultiHopPath","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[],"name":"swapRouterV3","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
              {"inputs":[{"internalType":"address","name":"","type":"address"}],"name":"authorizedCallers","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
            ]$$::jsonb),
            ('ADAPTER_CCIP','CCIPSender','2.1.0','0x5cce8cdfb77dccd28ed7cf0acf567f92d737abd9',$$[
              {"inputs":[{"internalType":"address","name":"caller","type":"address"},{"internalType":"bool","name":"allowed","type":"bool"}],"name":"setAuthorizedCaller","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct BridgeMessage","name":"message","type":"tuple"}],"name":"quoteFee","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
              {"inputs":[{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct BridgeMessage","name":"message","type":"tuple"}],"name":"sendMessage","outputs":[],"stateMutability":"payable","type":"function"}
            ]$$::jsonb),
            ('ADAPTER_HYPERBRIDGE','HyperbridgeSender','2.1.0','0xfdc7986e73f91ebc08130ba2325d32b23f844e26',$$[
              {"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"bytes","name":"stateMachineId","type":"bytes"}],"name":"setStateMachineId","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"bytes","name":"destination","type":"bytes"}],"name":"setDestinationContract","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"address","name":"swapper","type":"address"}],"name":"setSwapper","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct BridgeMessage","name":"message","type":"tuple"}],"name":"quoteFee","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
              {"inputs":[{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct BridgeMessage","name":"message","type":"tuple"}],"name":"sendMessage","outputs":[],"stateMutability":"payable","type":"function"},
              {"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"isChainConfigured","outputs":[{"internalType":"bool","name":"configured","type":"bool"}],"stateMutability":"view","type":"function"}
            ]$$::jsonb),
            ('ADAPTER_LAYERZERO','LayerZeroSenderAdapter','2.1.0','0x64505be2844d35284ab58984f93dceb21bc77464',$$[
              {"inputs":[],"name":"registerDelegate","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"uint32","name":"dstEid","type":"uint32"},{"internalType":"bytes32","name":"peer","type":"bytes32"}],"name":"setTrustedPeer","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"uint32","name":"dstEid","type":"uint32"},{"internalType":"uint256","name":"fee","type":"uint256"}],"name":"setMessageFee","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"uint32","name":"dstEid","type":"uint32"},{"internalType":"uint128","name":"gasLimit","type":"uint128"}],"name":"setEnforcedGas","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"isChainConfigured","outputs":[{"internalType":"bool","name":"configured","type":"bool"}],"stateMutability":"view","type":"function"}
            ]$$::jsonb),
            ('RECEIVER_LAYERZERO','LayerZeroReceiverAdapter','2.1.0','0x0c6c2cc9c2fb42d2fe591f2c3fee4db428090ad4',$$[
              {"inputs":[{"internalType":"address","name":"swapper","type":"address"}],"name":"setSwapper","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"uint32","name":"srcEid","type":"uint32"},{"internalType":"bytes32","name":"peer","type":"bytes32"}],"name":"setTrustedPeer","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"uint32","name":"srcEid","type":"uint32"}],"name":"getTrustConfig","outputs":[{"internalType":"bool","name":"trusted","type":"bool"},{"internalType":"bytes32","name":"configuredPeer","type":"bytes32"},{"internalType":"uint64","name":"expectedNonce","type":"uint64"}],"stateMutability":"view","type":"function"},
              {"inputs":[{"internalType":"uint32","name":"","type":"uint32"}],"name":"peers","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"stateMutability":"view","type":"function"}
            ]$$::jsonb),
            ('PRIVACY_MODULE','GatewayPrivacyModule','2.1.0','0x678fa4e50ed898e2c5694399651ea80894164766',$$[
              {"inputs":[{"internalType":"address","name":"gateway","type":"address"},{"internalType":"bool","name":"allowed","type":"bool"}],"name":"setAuthorizedGateway","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"address","name":"gateway","type":"address"}],"name":"authorizedGateway","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
              {"inputs":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"stealthReceiver","type":"address"},{"internalType":"address","name":"finalReceiver","type":"address"},{"internalType":"address","name":"token","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"address","name":"caller","type":"address"},{"internalType":"bool","name":"sameChain","type":"bool"}],"name":"forwardFromStealth","outputs":[],"stateMutability":"nonpayable","type":"function"}
            ]$$::jsonb),
            ('STEALTH_ESCROW_FACTORY','StealthEscrowFactory','2.1.0','0x703d53d548ef860902057226079bc842bf077d1c',$$[
              {"inputs":[{"internalType":"bytes32","name":"salt","type":"bytes32"},{"internalType":"address","name":"owner","type":"address"},{"internalType":"address","name":"forwarder","type":"address"}],"name":"deployEscrow","outputs":[{"internalType":"address","name":"escrow","type":"address"}],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"bytes32","name":"salt","type":"bytes32"},{"internalType":"address","name":"owner","type":"address"},{"internalType":"address","name":"forwarder","type":"address"}],"name":"predictEscrow","outputs":[{"internalType":"address","name":"predicted","type":"address"}],"stateMutability":"view","type":"function"},
              {"inputs":[{"internalType":"address","name":"candidate","type":"address"}],"name":"isEscrow","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
            ]$$::jsonb),
            ('FEE_POLICY_MANAGER','FeePolicyManager','2.1.0','0x5bd6093f455534dfd5c0220f5ba6660d5dbb30a8',$$[
              {"inputs":[{"internalType":"address","name":"strategy","type":"address"}],"name":"setDefaultStrategy","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[{"internalType":"address","name":"strategy","type":"address"}],"name":"setActiveStrategy","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[],"name":"clearActiveStrategy","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[],"name":"defaultStrategy","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
              {"inputs":[],"name":"activeStrategy","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
              {"inputs":[],"name":"resolveStrategy","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}
            ]$$::jsonb),
            ('FEE_STRATEGY_DEFAULT_V1','FeeStrategyDefaultV1','2.1.0','0x62ccb9fbbd975d41210b367f5bc1b6da00f71610',$$[
              {"inputs":[{"internalType":"address","name":"registry","type":"address"}],"name":"setTokenRegistry","outputs":[],"stateMutability":"nonpayable","type":"function"},
              {"inputs":[],"name":"tokenRegistry","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
              {"inputs":[],"name":"FIXED_BASE_FEE","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
              {"inputs":[],"name":"FEE_RATE_BPS","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}
            ]$$::jsonb)
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
        type = tc.type,
        version = tc.version,
        abi = tc.abi,
        is_active = TRUE,
        metadata = jsonb_build_object(
            'source', 'CHAIN_ARBITRUM.md',
            'sync_reason', 'phase6_partial_close',
            'phase6_status', 'conditionally_accepted'
        ),
        updated_at = NOW()
    FROM arbitrum_chain ac, arbitrum_target_contracts tc
    WHERE sc.chain_id = ac.id
      AND sc.deleted_at IS NULL
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
        'sync_reason', 'phase6_partial_close',
        'phase6_status', 'conditionally_accepted'
    ),
    NOW(),
    NOW()
FROM arbitrum_chain ac
CROSS JOIN arbitrum_target_contracts tc
WHERE NOT EXISTS (
    SELECT 1
    FROM arbitrum_reactivate re
    WHERE re.chain_id = ac.id
      AND re.type = tc.type
      AND re.address_lc = LOWER(tc.address)
);
