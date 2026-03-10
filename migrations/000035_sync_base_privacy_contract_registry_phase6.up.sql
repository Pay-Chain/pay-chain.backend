-- Phase 6 registry sync:
-- Ensure Base active contracts for PRIVACY_MODULE and STEALTH_ESCROW_FACTORY
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
                'PRIVACY_MODULE',
                'GatewayPrivacyModule',
                '2.1.0',
                '0xd8a6818468eBB65527118308B48c1A969977A086',
                $$[
                  {"inputs":[{"internalType":"address","name":"gateway","type":"address"},{"internalType":"bool","name":"allowed","type":"bool"}],"name":"setAuthorizedGateway","outputs":[],"stateMutability":"nonpayable","type":"function"},
                  {"inputs":[{"internalType":"address","name":"gateway","type":"address"}],"name":"authorizedGateway","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},
                  {"inputs":[{"internalType":"bytes32","name":"paymentId","type":"bytes32"},{"internalType":"address","name":"token","type":"address"},{"internalType":"address","name":"stealthReceiver","type":"address"},{"internalType":"address","name":"finalReceiver","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"},{"internalType":"bool","name":"sameChain","type":"bool"}],"name":"forwardFromStealth","outputs":[],"stateMutability":"nonpayable","type":"function"}
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
                  {"inputs":[{"internalType":"address","name":"escrow","type":"address"}],"name":"isEscrow","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"}
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
