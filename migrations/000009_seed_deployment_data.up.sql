-- 000009_seed_deployment_data.up.sql

-- =================================================================================
-- 1. Insert Chains: Base (8453) and Arbitrum (42161)
-- =================================================================================
INSERT INTO chains (id, namespace, name, chain_type, rpc_url, explorer_url, is_active)
VALUES
  (8453, 'eip155', 'Base Mainnet', 'EVM', 'https://mainnet.base.org', 'https://basescan.org', true),
  (42161, 'eip155', 'Arbitrum One', 'EVM', 'https://arb1.arbitrum.io/rpc', 'https://arbiscan.io', true)
ON CONFLICT (id) DO UPDATE SET 
    name = EXCLUDED.name, 
    rpc_url = EXCLUDED.rpc_url, 
    explorer_url = EXCLUDED.explorer_url;

-- =================================================================================
-- 2. Insert Additional RPCs (Base & Arbitrum)
-- =================================================================================
INSERT INTO chain_rpcs (chain_id, url, priority, is_active) VALUES
-- Base (8453)
(8453, 'https://base-mainnet.g.alchemy.com/v2/K9PzwLloeXxcOuFEx_fgR', 10, true),
(8453, 'https://base.llamarpc.com', 9, true),
(8453, 'https://base.meowrpc.com', 8, true),
(8453, 'https://1rpc.io/base', 8, true),
(8453, 'https://base-mainnet.public.blastapi.io', 7, true),
(8453, 'https://base.drpc.org', 7, true),
(8453, 'https://base-rpc.publicnode.com', 6, true),
(8453, 'https://api.zan.top/base-mainnet', 6, true),

-- Arbitrum (42161)
(42161, 'https://arb-mainnet.g.alchemy.com/v2/K9PzwLloeXxcOuFEx_fgR', 10, true),
(42161, 'https://arb-one.api.pocket.network', 9, true),
(42161, 'https://arbitrum.rpc.subquery.network/public', 8, true),
(42161, 'https://arbitrum.meowrpc.com', 8, true),
(42161, 'https://rpc.sentio.xyz/arbitrum-one', 7, true),
(42161, 'https://arb1.lava.build', 7, true),
(42161, 'https://arbitrum-one-rpc.publicnode.com', 6, true);

-- =================================================================================
-- 3. Insert Tokens (Ensure they exist)
-- =================================================================================
INSERT INTO tokens (symbol, name, decimals, is_stablecoin) 
SELECT 'USDE', 'Ethena USDe', 18, true WHERE NOT EXISTS (SELECT 1 FROM tokens WHERE symbol = 'USDE');

INSERT INTO tokens (symbol, name, decimals, is_stablecoin) 
SELECT 'WETH', 'Wrapped Ether', 18, false WHERE NOT EXISTS (SELECT 1 FROM tokens WHERE symbol = 'WETH');

INSERT INTO tokens (symbol, name, decimals, is_stablecoin) 
SELECT 'CBETH', 'Coinbase Wrapped Staked ETH', 18, false WHERE NOT EXISTS (SELECT 1 FROM tokens WHERE symbol = 'CBETH');

INSERT INTO tokens (symbol, name, decimals, is_stablecoin) 
SELECT 'CBBTC', 'Coinbase Wrapped BTC', 8, false WHERE NOT EXISTS (SELECT 1 FROM tokens WHERE symbol = 'CBBTC');

INSERT INTO tokens (symbol, name, decimals, is_stablecoin) 
SELECT 'WBTC', 'Wrapped BTC', 8, false WHERE NOT EXISTS (SELECT 1 FROM tokens WHERE symbol = 'WBTC');

INSERT INTO tokens (symbol, name, decimals, is_stablecoin) 
SELECT 'IDRX', 'IDRX', 2, true WHERE NOT EXISTS (SELECT 1 FROM tokens WHERE symbol = 'IDRX');

-- =================================================================================
-- 4. Seed Smart Contracts (Base)
-- =================================================================================
INSERT INTO smart_contracts (chain_id, contract_address, name, type, version, deployer_address, is_active, abi)
VALUES
(8453, '0xcbaEf496b284fBD5145E4E4f25d39BA90E773Dfd', 'TokenRegistry', 'TOKEN_REGISTRY', '1.0.0', '0x0000000000000000000000000000000000000000', true, '{}'::jsonb),
(8453, '0x4CD0C58C998ADaFb8c477191bAE7013436126628', 'PayChainVault', 'VAULT', '1.0.0', '0x0000000000000000000000000000000000000000', true, '{}'::jsonb),
(8453, '0xaC3C22fcFE2E4875DCB900d8c41817E7909BBfA3', 'PayChainRouter', 'ROUTER', '1.0.0', '0x0000000000000000000000000000000000000000', true, '{}'::jsonb),
(8453, '0x03De9D5B750a7a74002F4BBd75526c4d83C56020', 'PayChainGateway', 'GATEWAY', '1.0.0', '0x0000000000000000000000000000000000000000', true, '{}'::jsonb),
(8453, '0x31bAE445d57Bf34A6b5587BDdC16a6591EB84A73', 'TokenSwapper', 'SWAPPER', '1.0.0', '0x0000000000000000000000000000000000000000', true, '{}'::jsonb),
(8453, '0xE9B21247Ea01B48b219e6413e9E93C554AeEbc29', 'CCIPSender', 'ADAPTER_CCIP_SENDER', '1.0.0', '0x0000000000000000000000000000000000000000', true, '{}'::jsonb),
(8453, '0xD41F527D8386C9a637878380711a28B2239f90AA', 'CCIPReceiverAdapter', 'ADAPTER_CCIP_RECEIVER', '1.0.0', '0x0000000000000000000000000000000000000000', true, '{}'::jsonb),
(8453, '0x5Ce455a77aEcE956ed6AbDbaEa93B69E5aC1DC20', 'HyperbridgeSender', 'ADAPTER_HYPERBRIDGE_SENDER', '1.0.0', '0x0000000000000000000000000000000000000000', true, '{}'::jsonb),
(8453, '0x723D8f4A1C6954b862D045a2eaDA2306d6B60Fd1', 'HyperbridgeReceiver', 'ADAPTER_HYPERBRIDGE_RECEIVER', '1.0.0', '0x0000000000000000000000000000000000000000', true, '{}'::jsonb)
ON CONFLICT (chain_id, contract_address) DO NOTHING;

-- =================================================================================
-- 5. Seed Smart Contracts (Arbitrum)
-- =================================================================================
INSERT INTO smart_contracts (chain_id, contract_address, name, type, version, deployer_address, is_active, abi)
VALUES
(42161, '0x903095a079a0B32D7B2B1C55eD3535Bc0B1C16D9', 'TokenRegistry', 'TOKEN_REGISTRY', '1.0.0', '0x0000000000000000000000000000000000000000', true, '{}'::jsonb),
(42161, '0x6D350c3e11c3fA7eA8c35d86d14eFae117eaDDC2', 'PayChainVault', 'VAULT', '1.0.0', '0x0000000000000000000000000000000000000000', true, '{}'::jsonb),
(42161, '0x8397b59E44CC0A0e4bda6609F80fe7C55143C638', 'PayChainRouter', 'ROUTER', '1.0.0', '0x0000000000000000000000000000000000000000', true, '{}'::jsonb),
(42161, '0xd2C69EA4968e9F7cc8C0F447eB9b6DFdFFb1F8D7', 'PayChainGateway', 'GATEWAY', '1.0.0', '0x0000000000000000000000000000000000000000', true, '{}'::jsonb),
(42161, '0x6CFc15C526B8d06e7D192C18B5A2C5e3E10F7D8c', 'TokenSwapper', 'SWAPPER', '1.0.0', '0x0000000000000000000000000000000000000000', true, '{}'::jsonb),
(42161, '0xF043b0b91C8F5b6C2DC63897f1632D6D15e199A9', 'CCIPSender', 'ADAPTER_CCIP_SENDER', '1.0.0', '0x0000000000000000000000000000000000000000', true, '{}'::jsonb),
(42161, '0xd0F705AE978A3f0faF383c47C7205257d6A1A9e3', 'CCIPReceiverAdapter', 'ADAPTER_CCIP_RECEIVER', '1.0.0', '0x0000000000000000000000000000000000000000', true, '{}'::jsonb),
(42161, '0xdf6c1dFEf6A16315F6Be460114fB090Aea4dE500', 'HyperbridgeSender', 'ADAPTER_HYPERBRIDGE_SENDER', '1.0.0', '0x0000000000000000000000000000000000000000', true, '{}'::jsonb),
(42161, '0xbC75055BdF937353721BFBa9Dd1DCCFD0c70B8dd', 'HyperbridgeReceiver', 'ADAPTER_HYPERBRIDGE_RECEIVER', '1.0.0', '0x0000000000000000000000000000000000000000', true, '{}'::jsonb)
ON CONFLICT (chain_id, contract_address) DO NOTHING;

-- =================================================================================
-- 6. Seed Supported Tokens
-- =================================================================================

-- Base
INSERT INTO supported_tokens (chain_id, token_id, contract_address)
SELECT 8453, id, '0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913' FROM tokens WHERE symbol = 'USDC'
ON CONFLICT (chain_id, token_id) DO UPDATE SET contract_address = EXCLUDED.contract_address;

INSERT INTO supported_tokens (chain_id, token_id, contract_address)
SELECT 8453, id, '0x5d3a1Ff2b6BAb83b63cd9AD0787074081a52ef34' FROM tokens WHERE symbol = 'USDE'
ON CONFLICT (chain_id, token_id) DO UPDATE SET contract_address = EXCLUDED.contract_address;

INSERT INTO supported_tokens (chain_id, token_id, contract_address)
SELECT 8453, id, '0x4200000000000000000000000000000000000006' FROM tokens WHERE symbol = 'WETH'
ON CONFLICT (chain_id, token_id) DO UPDATE SET contract_address = EXCLUDED.contract_address;

INSERT INTO supported_tokens (chain_id, token_id, contract_address)
SELECT 8453, id, '0x2Ae3F1Ec7F1F5012CFEab0185bfc7aa3cf0DEc22' FROM tokens WHERE symbol = 'CBETH'
ON CONFLICT (chain_id, token_id) DO UPDATE SET contract_address = EXCLUDED.contract_address;

INSERT INTO supported_tokens (chain_id, token_id, contract_address)
SELECT 8453, id, '0xcbB7C0000aB88B473b1f5aFd9ef808440eed33Bf' FROM tokens WHERE symbol = 'CBBTC'
ON CONFLICT (chain_id, token_id) DO UPDATE SET contract_address = EXCLUDED.contract_address;

INSERT INTO supported_tokens (chain_id, token_id, contract_address)
SELECT 8453, id, '0x0555E30da8f98308EdB960aa94C0Db47230d2B9c' FROM tokens WHERE symbol = 'WBTC'
ON CONFLICT (chain_id, token_id) DO UPDATE SET contract_address = EXCLUDED.contract_address;

INSERT INTO supported_tokens (chain_id, token_id, contract_address)
SELECT 8453, id, '0x18Bc5bcC660cf2B9cE3cd51a404aFe1a0cBD3C22' FROM tokens WHERE symbol = 'IDRX'
ON CONFLICT (chain_id, token_id) DO UPDATE SET contract_address = EXCLUDED.contract_address;


-- Arbitrum
INSERT INTO supported_tokens (chain_id, token_id, contract_address)
SELECT 42161, id, '0xaf88d065e77c8cC2239327C5EDb3A432268e5831' FROM tokens WHERE symbol = 'USDC'
ON CONFLICT (chain_id, token_id) DO UPDATE SET contract_address = EXCLUDED.contract_address;

INSERT INTO supported_tokens (chain_id, token_id, contract_address)
SELECT 42161, id, '0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9' FROM tokens WHERE symbol = 'USDT'
ON CONFLICT (chain_id, token_id) DO UPDATE SET contract_address = EXCLUDED.contract_address;
