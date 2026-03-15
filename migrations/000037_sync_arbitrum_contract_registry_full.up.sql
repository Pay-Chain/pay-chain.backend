-- Arbitrum full registry sync:
-- Activate latest deployed addresses for runtime contract types.

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM chains
        WHERE deleted_at IS NULL
          AND chain_id IN ('42161', 'eip155:42161')
    ) THEN
        RAISE EXCEPTION 'Arbitrum chain (42161) not found in chains table';
    END IF;
END $$;

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
            (
                'TOKEN_REGISTRY',
                'TokenRegistry',
                '2.1.0',
                '0x53f1e35fea4b2cdc7e73feb4e36365c88569ebf0',
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
                '0x4a92d4079853c78df38b4bbd574aa88679adef93',
                $$[
                  {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"from","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"pullTokens","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"pushTokens","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"spender","type":"address"},{"internalType":"bool","name":"authorized","type":"bool"}],"name":"setAuthorizedSpender","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"","type":"address"}],"name":"authorizedSpenders","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
                ]$$::jsonb
            ),
            (
                'ROUTER',
                'PaymentKitaRouter',
                '2.1.0',
                '0x3722374b187e5400f4423dbc45ad73784604d275',
                $$[
                  {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"internalType":"address","name":"adapter","type":"address"}],"name":"registerAdapter","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"getAdapter","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"hasAdapter","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"internalType":"uint8","name":"mode","type":"uint8"}],"name":"setBridgeMode","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"internalType":"bool","name":"enabled","type":"bool"}],"name":"setTokenBridgeDestSwapCapability","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct BridgeMessage","name":"msgData","type":"tuple"}],"name":"quotePaymentFee","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct BridgeMessage","name":"msgData","type":"tuple"}],"name":"routePayment","outputs":[],"stateMutability":"payable","type":"function"}
                ]$$::jsonb
            ),
            (
                'GATEWAY',
                'PaymentKitaGateway',
                '2.1.0',
                '0x259294aecdc0006b73b1281c30440a8179cff44c',
                $$[
                  {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint8","name":"bridgeType","type":"uint8"}],"name":"setDefaultBridgeType","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"adapter","type":"address"},{"internalType":"bool","name":"authorized","type":"bool"}],"name":"setAuthorizedAdapter","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"defaultBridgeTypes","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"components":[{"internalType":"bytes","name":"destChainIdBytes","type":"bytes"},{"internalType":"bytes","name":"receiverBytes","type":"bytes"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"bridgeTokenSource","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amountInSource","type":"uint256"},{"internalType":"uint256","name":"minBridgeAmountOut","type":"uint256"},{"internalType":"uint256","name":"minDestAmountOut","type":"uint256"},{"internalType":"uint8","name":"mode","type":"uint8"},{"internalType":"uint8","name":"bridgeOption","type":"uint8"}],"internalType":"struct PaymentRequestV2","name":"request","type":"tuple"}],"name":"createPayment","outputs":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"}],"stateMutability":"payable","type":"function"},
                  {"inputs":[{"components":[{"internalType":"bytes","name":"destChainIdBytes","type":"bytes"},{"internalType":"bytes","name":"receiverBytes","type":"bytes"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"bridgeTokenSource","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amountInSource","type":"uint256"},{"internalType":"uint256","name":"minBridgeAmountOut","type":"uint256"},{"internalType":"uint256","name":"minDestAmountOut","type":"uint256"},{"internalType":"uint8","name":"mode","type":"uint8"},{"internalType":"uint8","name":"bridgeOption","type":"uint8"}],"internalType":"struct PaymentRequestV2","name":"request","type":"tuple"}],"name":"createPaymentDefaultBridge","outputs":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"}],"stateMutability":"payable","type":"function"},
                  {"inputs":[{"components":[{"internalType":"bytes","name":"destChainIdBytes","type":"bytes"},{"internalType":"bytes","name":"receiverBytes","type":"bytes"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"bridgeTokenSource","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amountInSource","type":"uint256"},{"internalType":"uint256","name":"minBridgeAmountOut","type":"uint256"},{"internalType":"uint256","name":"minDestAmountOut","type":"uint256"},{"internalType":"uint8","name":"mode","type":"uint8"},{"internalType":"uint8","name":"bridgeOption","type":"uint8"}],"internalType":"struct PaymentRequestV2","name":"request","type":"tuple"},{"components":[{"internalType":"bytes32","name":"intentId","type":"bytes32"},{"internalType":"address","name":"stealthReceiver","type":"address"}],"internalType":"struct PrivateRouting","name":"privacy","type":"tuple"}],"name":"createPaymentPrivate","outputs":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"}],"stateMutability":"payable","type":"function"},
                  {"inputs":[{"components":[{"internalType":"bytes","name":"destChainIdBytes","type":"bytes"},{"internalType":"bytes","name":"receiverBytes","type":"bytes"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"bridgeTokenSource","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amountInSource","type":"uint256"},{"internalType":"uint256","name":"minBridgeAmountOut","type":"uint256"},{"internalType":"uint256","name":"minDestAmountOut","type":"uint256"},{"internalType":"uint8","name":"mode","type":"uint8"},{"internalType":"uint8","name":"bridgeOption","type":"uint8"}],"internalType":"struct PaymentRequestV2","name":"request","type":"tuple"}],"name":"quotePaymentCost","outputs":[{"internalType":"uint256","name":"platformFeeToken","type":"uint256"},{"internalType":"uint256","name":"bridgeFeeNative","type":"uint256"},{"internalType":"uint256","name":"totalSourceTokenRequired","type":"uint256"},{"internalType":"uint8","name":"bridgeType","type":"uint8"},{"internalType":"bool","name":"bridgeQuoteOK","type":"bool"},{"internalType":"string","name":"bridgeQuoteReason","type":"string"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"components":[{"internalType":"bytes","name":"destChainIdBytes","type":"bytes"},{"internalType":"bytes","name":"receiverBytes","type":"bytes"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"bridgeTokenSource","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amountInSource","type":"uint256"},{"internalType":"uint256","name":"minBridgeAmountOut","type":"uint256"},{"internalType":"uint256","name":"minDestAmountOut","type":"uint256"},{"internalType":"uint8","name":"mode","type":"uint8"},{"internalType":"uint8","name":"bridgeOption","type":"uint8"}],"internalType":"struct PaymentRequestV2","name":"request","type":"tuple"}],"name":"previewApproval","outputs":[{"internalType":"address","name":"approvalToken","type":"address"},{"internalType":"uint256","name":"approvalAmount","type":"uint256"},{"internalType":"uint256","name":"requiredNativeFee","type":"uint256"}],"stateMutability":"view","type":"function"}
                ]$$::jsonb
            ),
            (
                'TOKEN_SWAPPER',
                'TokenSwapper',
                '2.1.0',
                '0x5d86bfd5a361bc652bc596dd2a77cd2bdba2bf35',
                $$[
                  {"inputs":[{"internalType":"address","name":"tokenIn","type":"address"},{"internalType":"address","name":"tokenOut","type":"address"},{"internalType":"uint256","name":"amountIn","type":"uint256"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"}],"name":"swap","outputs":[{"internalType":"uint256","name":"amountOut","type":"uint256"}],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"tokenIn","type":"address"},{"internalType":"address","name":"tokenOut","type":"address"},{"internalType":"uint256","name":"amountIn","type":"uint256"}],"name":"getRealQuote","outputs":[{"internalType":"uint256","name":"amountOut","type":"uint256"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"address","name":"tokenIn","type":"address"},{"internalType":"address","name":"tokenOut","type":"address"}],"name":"findRoute","outputs":[{"internalType":"bool","name":"exists","type":"bool"},{"internalType":"bool","name":"isDirect","type":"bool"},{"internalType":"address[]","name":"path","type":"address[]"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"address","name":"_vault","type":"address"}],"name":"setVault","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"_swapRouterV3","type":"address"}],"name":"setV3Router","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"caller","type":"address"},{"internalType":"bool","name":"allowed","type":"bool"}],"name":"setAuthorizedCaller","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"tokenA","type":"address"},{"internalType":"address","name":"tokenB","type":"address"},{"internalType":"uint24","name":"fee","type":"uint24"}],"name":"setV3Pool","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"tokenA","type":"address"},{"internalType":"address","name":"tokenB","type":"address"},{"internalType":"uint24","name":"fee","type":"uint24"},{"internalType":"int24","name":"tickSpacing","type":"int24"},{"internalType":"address","name":"hooks","type":"address"},{"internalType":"bytes","name":"hookData","type":"bytes"}],"name":"setDirectPool","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"tokenIn","type":"address"},{"internalType":"address","name":"tokenOut","type":"address"},{"internalType":"address[]","name":"path","type":"address[]"}],"name":"setMultiHopPath","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[],"name":"swapRouterV3","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
                  {"inputs":[],"name":"universalRouter","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}
                ]$$::jsonb
            ),
            (
                'ADAPTER_CCIP',
                'CCIPSender',
                '2.1.0',
                '0x5cce8cdfb77dccd28ed7cf0acf567f92d737abd9',
                $$[
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"uint64","name":"selector","type":"uint64"},{"internalType":"address","name":"destAdapter","type":"address"}],"name":"setChainConfig","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"uint64","name":"selector","type":"uint64"}],"name":"setChainSelector","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"bytes","name":"adapter","type":"bytes"}],"name":"setDestinationAdapter","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"uint256","name":"gasLimit","type":"uint256"}],"name":"setDestinationGasLimit","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"bytes","name":"extraArgs","type":"bytes"}],"name":"setDestinationExtraArgs","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"address","name":"feeToken","type":"address"}],"name":"setDestinationFeeToken","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct BridgeMessage","name":"message","type":"tuple"}],"name":"quoteFee","outputs":[{"internalType":"uint256","name":"fee","type":"uint256"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct BridgeMessage","name":"message","type":"tuple"}],"name":"sendMessage","outputs":[{"internalType":"bytes32","name":"messageId","type":"bytes32"}],"stateMutability":"payable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"chainSelectors","outputs":[{"internalType":"uint64","name":"","type":"uint64"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationAdapters","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationGasLimits","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationExtraArgs","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationFeeTokens","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}
                ]$$::jsonb
            ),
            (
                'ADAPTER_HYPERBRIDGE',
                'HyperbridgeSender',
                '2.1.0',
                '0xfdc7986e73f91ebc08130ba2325d32b23f844e26',
                $$[
                  {"inputs":[{"internalType":"address","name":"_router","type":"address"}],"name":"setSwapRouter","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"_swapper","type":"address"}],"name":"setSwapper","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"bytes","name":"stateMachineId","type":"bytes"}],"name":"setStateMachineId","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"},{"internalType":"bytes","name":"destination","type":"bytes"}],"name":"setDestinationContract","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"uint64","name":"newTimeout","type":"uint64"}],"name":"setDefaultTimeout","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct BridgeMessage","name":"message","type":"tuple"}],"name":"quoteFee","outputs":[{"internalType":"uint256","name":"fee","type":"uint256"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct BridgeMessage","name":"message","type":"tuple"}],"name":"sendMessage","outputs":[{"internalType":"bytes32","name":"commitment","type":"bytes32"}],"stateMutability":"payable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"isChainConfigured","outputs":[{"internalType":"bool","name":"configured","type":"bool"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"stateMachineIds","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"string","name":"chainId","type":"string"}],"name":"destinationContracts","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"}
                ]$$::jsonb
            ),
            (
                'ADAPTER_LAYERZERO',
                'LayerZeroSenderAdapter',
                '2.1.0',
                '0x64505be2844d35284ab58984f93dceb21bc77464',
                $$[
                  {"inputs":[{"internalType":"address","name":"_router","type":"address"}],"name":"setRouter","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint32","name":"dstEid","type":"uint32"},{"internalType":"bytes32","name":"peer","type":"bytes32"}],"name":"setRoute","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"bytes","name":"options","type":"bytes"}],"name":"setEnforcedOptions","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[],"name":"registerDelegate","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct BridgeMessage","name":"message","type":"tuple"}],"name":"quoteFee","outputs":[{"internalType":"uint256","name":"fee","type":"uint256"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct BridgeMessage","name":"message","type":"tuple"}],"name":"sendMessage","outputs":[{"internalType":"bytes32","name":"messageId","type":"bytes32"}],"stateMutability":"payable","type":"function"},
                  {"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"isRouteConfigured","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"dstEids","outputs":[{"internalType":"uint32","name":"","type":"uint32"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"peers","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"enforcedOptions","outputs":[{"internalType":"bytes","name":"","type":"bytes"}],"stateMutability":"view","type":"function"}
                ]$$::jsonb
            ),
            (
                'RECEIVER_LAYERZERO',
                'LayerZeroReceiverAdapter',
                '2.1.0',
                '0x0c6c2cc9c2fb42d2fe591f2c3fee4db428090ad4',
                $$[
                  {"inputs":[{"internalType":"address","name":"_swapper","type":"address"}],"name":"setSwapper","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"uint32","name":"eid","type":"uint32"},{"internalType":"bytes32","name":"peer","type":"bytes32"}],"name":"setPeer","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"uint32","name":"_srcEid","type":"uint32"},{"internalType":"bytes32","name":"_sender","type":"bytes32"}],"name":"getPathState","outputs":[{"internalType":"bool","name":"peerConfigured","type":"bool"},{"internalType":"bool","name":"trusted","type":"bool"},{"internalType":"bytes32","name":"configuredPeer","type":"bytes32"},{"internalType":"uint64","name":"expectedNonce","type":"uint64"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"uint32","name":"","type":"uint32"}],"name":"peers","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"stateMutability":"view","type":"function"}
                ]$$::jsonb
            ),
            (
                'PRIVACY_MODULE',
                'GatewayPrivacyModule',
                '2.1.0',
                '0x678fa4e50ed898e2c5694399651ea80894164766',
                $$[
                  {"inputs":[{"internalType":"address","name":"gateway","type":"address"},{"internalType":"bool","name":"allowed","type":"bool"}],"name":"setAuthorizedGateway","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"","type":"address"}],"name":"authorizedGateway","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"stealthReceiver","type":"address"},{"internalType":"address","name":"finalReceiver","type":"address"},{"internalType":"address","name":"token","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"address","name":"caller","type":"address"},{"internalType":"bool","name":"sameChain","type":"bool"}],"name":"forwardFromStealth","outputs":[],"stateMutability":"nonpayable","type":"function"}
                ]$$::jsonb
            ),
            (
                'STEALTH_ESCROW_FACTORY',
                'StealthEscrowFactory',
                '2.1.0',
                '0x703d53d548ef860902057226079bc842bf077d1c',
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
                '0x5bd6093f455534dfd5c0220f5ba6660d5dbb30a8',
                $$[
                  {"inputs":[{"internalType":"address","name":"strategy","type":"address"}],"name":"setDefaultStrategy","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"strategy","type":"address"}],"name":"setActiveStrategy","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[],"name":"clearActiveStrategy","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[],"name":"defaultStrategy","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
                  {"inputs":[],"name":"activeStrategy","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
                  {"inputs":[],"name":"resolveStrategy","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"bytes","name":"sourceChainId","type":"bytes"},{"internalType":"bytes","name":"destChainId","type":"bytes"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"uint256","name":"bridgeFeeNative","type":"uint256"},{"internalType":"uint256","name":"swapImpactBps","type":"uint256"},{"internalType":"bool","name":"policyEnabled","type":"bool"},{"internalType":"uint256","name":"payloadLength","type":"uint256"},{"internalType":"uint256","name":"policyOverheadBytes","type":"uint256"},{"internalType":"uint256","name":"policyPerByteRate","type":"uint256"},{"internalType":"uint256","name":"policyMinFee","type":"uint256"},{"internalType":"uint256","name":"policyMaxFee","type":"uint256"},{"internalType":"uint256","name":"fixedBaseFee","type":"uint256"},{"internalType":"uint256","name":"feeRateBps","type":"uint256"}],"name":"computePlatformFee","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}
                ]$$::jsonb
            ),
            (
                'FEE_STRATEGY_DEFAULT_V1',
                'FeeStrategyDefaultV1',
                '2.1.0',
                '0x62ccb9fbbd975d41210b367f5bc1b6da00f71610',
                $$[
                  {"inputs":[{"internalType":"address","name":"registry","type":"address"}],"name":"setTokenRegistry","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[],"name":"tokenRegistry","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
                  {"inputs":[],"name":"FIXED_BASE_FEE","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
                  {"inputs":[],"name":"FEE_RATE_BPS","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"bytes","name":"","type":"bytes"},{"internalType":"bytes","name":"","type":"bytes"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"","type":"address"},{"internalType":"uint256","name":"sourceAmount","type":"uint256"},{"internalType":"uint256","name":"","type":"uint256"},{"internalType":"uint256","name":"","type":"uint256"}],"name":"computePlatformFee","outputs":[{"internalType":"uint256","name":"platformFee","type":"uint256"}],"stateMutability":"view","type":"function"}
                ]$$::jsonb
            )
    ) AS t(type, name, version, address, abi)
),
deactivate_by_type AS (
    UPDATE smart_contracts sc
    SET is_active = FALSE,
        updated_at = NOW()
    FROM arbitrum_chain ac, target_contracts tc
    WHERE sc.chain_id = ac.id
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
    FROM arbitrum_chain ac, target_contracts tc
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
    '{}',
    NOW(),
    NOW()
FROM arbitrum_chain ac
CROSS JOIN target_contracts tc
WHERE NOT EXISTS (
    SELECT 1
    FROM reactivate_existing re
    WHERE re.chain_id = ac.id
      AND re.type = tc.type
      AND re.address_lc = LOWER(tc.address)
);
