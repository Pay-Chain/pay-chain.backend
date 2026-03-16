-- 000041_sync_stargate_rescuable_adapters.up.sql
-- 1. Rename layerzero_configs to stargate_configs if it exists
-- 2. Normalize smart_contract types from LAYERZERO to STARGATE
-- 3. Synchronize the smart contract registry with the newly deployed Stargate adapters

DO $$
DECLARE
    base_chain_id UUID;
    polygon_chain_id UUID;
    arbitrum_chain_id UUID;
    sender_abi JSONB := $abi$[
        {"inputs":[{"internalType":"address","name":"_vault","type":"address"},{"internalType":"address","name":"_gateway","type":"address"},{"internalType":"address","name":"_router","type":"address"}],"stateMutability":"nonpayable","type":"constructor"},
        {"inputs":[{"internalType":"string","name":"destChainId","type":"string"}],"name":"RouteNotConfigured","type":"error"},
        {"inputs":[{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct BridgeMessage","name":"message","type":"tuple"}],"name":"quoteFee","outputs":[{"internalType":"uint256","name":"fee","type":"uint256"}],"stateMutability":"view","type":"function"},
        {"inputs":[{"components":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"receiver","type":"address"},{"internalType":"address","name":"sourceToken","type":"address"},{"internalType":"address","name":"destToken","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"uint256","name":"minAmountOut","type":"uint256"},{"internalType":"address","name":"payer","type":"address"}],"internalType":"struct BridgeMessage","name":"message","type":"tuple"}],"name":"sendMessage","outputs":[{"internalType":"bytes32","name":"messageId","type":"bytes32"}],"stateMutability":"payable","type":"function"},
        {"inputs":[{"internalType":"string","name":"destChainId","type":"string"},{"internalType":"address","name":"stargate","type":"address"},{"internalType":"uint32","name":"dstEid","type":"uint32"},{"internalType":"bytes32","name":"destinationAdapter","type":"bytes32"}],"name":"setRoute","outputs":[],"stateMutability":"nonpayable","type":"function"},
        {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"rescueToken","outputs":[],"stateMutability":"nonpayable","type":"function"},
        {"inputs":[{"internalType":"address payable","name":"to","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"rescueNative","outputs":[],"stateMutability":"nonpayable","type":"function"}
    ]$abi$::jsonb;
    receiver_abi JSONB := $abi$[
        {"inputs":[{"internalType":"address","name":"_endpoint","type":"address"},{"internalType":"address","name":"_gateway","type":"address"},{"internalType":"address","name":"_vault","type":"address"}],"stateMutability":"nonpayable","type":"constructor"},
        {"inputs":[{"internalType":"uint32","name":"srcEid","type":"uint32"},{"internalType":"address","name":"stargate","type":"address"},{"internalType":"address","name":"receivedToken","type":"address"}],"name":"setRoute","outputs":[],"stateMutability":"nonpayable","type":"function"},
        {"inputs":[{"internalType":"uint32","name":"srcEid","type":"uint32"},{"internalType":"bool","name":"allowed","type":"bool"}],"name":"setSourceEidAllowed","outputs":[],"stateMutability":"nonpayable","type":"function"},
        {"inputs":[{"internalType":"bytes32","name":"guid","type":"bytes32"}],"name":"retryFailedCompose","outputs":[],"stateMutability":"nonpayable","type":"function"},
        {"inputs":[{"internalType":"bytes32","name":"guid","type":"bytes32"}],"name":"getFailedComposeStatus","outputs":[{"internalType":"bool","name":"exists","type":"bool"},{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"bytes","name":"reason","type":"bytes"},{"internalType":"uint256","name":"retryCount","type":"uint256"}],"stateMutability":"view","type":"function"},
        {"inputs":[{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"rescueToken","outputs":[],"stateMutability":"nonpayable","type":"function"},
        {"inputs":[{"internalType":"address payable","name":"to","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"rescueNative","outputs":[],"stateMutability":"nonpayable","type":"function"}
    ]$abi$::jsonb;
BEGIN
    -- 1. Rename table if old name exists
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'layerzero_configs') THEN
        ALTER TABLE layerzero_configs RENAME TO stargate_configs;
    END IF;

    -- 2. Normalize contract types
    UPDATE smart_contracts SET type = 'ADAPTER_STARGATE' WHERE type = 'ADAPTER_LAYERZERO';
    UPDATE smart_contracts SET type = 'RECEIVER_STARGATE' WHERE type = 'RECEIVER_LAYERZERO';

    -- 3. Sync addresses
    -- Resolve chain IDs
    SELECT id INTO base_chain_id FROM chains WHERE chain_id IN ('8453', 'eip155:8453') AND deleted_at IS NULL LIMIT 1;
    SELECT id INTO polygon_chain_id FROM chains WHERE chain_id IN ('137', 'eip155:137') AND deleted_at IS NULL LIMIT 1;
    SELECT id INTO arbitrum_chain_id FROM chains WHERE chain_id IN ('42161', 'eip155:42161') AND deleted_at IS NULL LIMIT 1;

    -- Deactivate current Stargate adapters for these chains
    UPDATE smart_contracts 
    SET is_active = FALSE, updated_at = NOW() 
    WHERE chain_id IN (base_chain_id, polygon_chain_id, arbitrum_chain_id) 
    AND type IN ('ADAPTER_STARGATE', 'RECEIVER_STARGATE');

    -- Base Stargate
    INSERT INTO smart_contracts (id, name, chain_id, address, abi, type, version, is_active, metadata, created_at, updated_at)
    VALUES 
        (uuid_generate_v7(), 'StargateSenderAdapter', base_chain_id, '0x1F746d1130d161413e0BC5598801798c402331d7', sender_abi, 'ADAPTER_STARGATE', '2.2.0', TRUE, '{"source": "CHAIN_BASE.md", "rescuable": true, "sync_reason": "rescuable_v2_migration"}'::jsonb, NOW(), NOW()),
        (uuid_generate_v7(), 'StargateReceiverAdapter', base_chain_id, '0xE09ed3D37ac311F9ef4aCF8927C27495Cc0D291A', receiver_abi, 'RECEIVER_STARGATE', '2.2.0', TRUE, '{"source": "CHAIN_BASE.md", "rescuable": true, "sync_reason": "rescuable_v2_migration"}'::jsonb, NOW(), NOW());

    -- Polygon Stargate
    INSERT INTO smart_contracts (id, name, chain_id, address, abi, type, version, is_active, metadata, created_at, updated_at)
    VALUES 
        (uuid_generate_v7(), 'StargateSenderAdapter', polygon_chain_id, '0x838Ba4E44E24f4d9A655698df535F404448aA2A9', sender_abi, 'ADAPTER_STARGATE', '2.2.0', TRUE, '{"source": "CHAIN_POLYGON.md", "rescuable": true, "sync_reason": "rescuable_v2_migration"}'::jsonb, NOW(), NOW()),
        (uuid_generate_v7(), 'StargateReceiverAdapter', polygon_chain_id, '0x1808bD03899C80D3C9619AD9740E8db04F32b471', receiver_abi, 'RECEIVER_STARGATE', '2.2.0', TRUE, '{"source": "CHAIN_POLYGON.md", "rescuable": true, "sync_reason": "rescuable_v2_migration"}'::jsonb, NOW(), NOW());

    -- Arbitrum Stargate
    INSERT INTO smart_contracts (id, name, chain_id, address, abi, type, version, is_active, metadata, created_at, updated_at)
    VALUES 
        (uuid_generate_v7(), 'StargateSenderAdapter', arbitrum_chain_id, '0x64976A3cDE870507B269FD4A8aC2dC9993bc3F3A', sender_abi, 'ADAPTER_STARGATE', '2.2.0', TRUE, '{"source": "CHAIN_ARBITRUM.md", "rescuable": true, "sync_reason": "rescuable_v2_migration"}'::jsonb, NOW(), NOW()),
        (uuid_generate_v7(), 'StargateReceiverAdapter', arbitrum_chain_id, '0xA0502C041AAE8Ae1A4141D7E7937b34A01510fcf', receiver_abi, 'RECEIVER_STARGATE', '2.2.0', TRUE, '{"source": "CHAIN_ARBITRUM.md", "rescuable": true, "sync_reason": "rescuable_v2_migration"}'::jsonb, NOW(), NOW());

    -- Update stargate_configs peer_hex for all mesh routes
    -- peer_hex is the receiver on the destination chain
    
    -- Base destinations
    UPDATE stargate_configs SET peer_hex = '0x0000000000000000000000001808bd03899c80d3c9619ad9740e8db04f32b471', updated_at = NOW() 
    WHERE source_chain_id = base_chain_id AND dest_chain_id = polygon_chain_id AND deleted_at IS NULL;
    
    UPDATE stargate_configs SET peer_hex = '0x000000000000000000000000a0502c041aae8ae1a4141d7e7937b34a01510fcf', updated_at = NOW() 
    WHERE source_chain_id = base_chain_id AND dest_chain_id = arbitrum_chain_id AND deleted_at IS NULL;

    -- Polygon destinations
    UPDATE stargate_configs SET peer_hex = '0x000000000000000000000000e09ed3d37ac311f9ef4acf8927c27495cc0d291a', updated_at = NOW() 
    WHERE source_chain_id = polygon_chain_id AND dest_chain_id = base_chain_id AND deleted_at IS NULL;
    
    UPDATE stargate_configs SET peer_hex = '0x000000000000000000000000a0502c041aae8ae1a4141d7e7937b34a01510fcf', updated_at = NOW() 
    WHERE source_chain_id = polygon_chain_id AND dest_chain_id = arbitrum_chain_id AND deleted_at IS NULL;

    -- Arbitrum destinations
    UPDATE stargate_configs SET peer_hex = '0x000000000000000000000000e09ed3d37ac311f9ef4acf8927c27495cc0d291a', updated_at = NOW() 
    WHERE source_chain_id = arbitrum_chain_id AND dest_chain_id = base_chain_id AND deleted_at IS NULL;
    
    UPDATE stargate_configs SET peer_hex = '0x0000000000000000000000001808bd03899c80d3c9619ad9740e8db04f32b471', updated_at = NOW() 
    WHERE source_chain_id = arbitrum_chain_id AND dest_chain_id = polygon_chain_id AND deleted_at IS NULL;

END $$;
