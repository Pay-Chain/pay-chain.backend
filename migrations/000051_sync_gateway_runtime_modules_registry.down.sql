-- Roll back gateway runtime module contract registry sync

WITH base_chain AS (
    SELECT id
    FROM chains
    WHERE deleted_at IS NULL
      AND chain_id IN ('8453', 'eip155:8453')
    ORDER BY updated_at DESC NULLS LAST, created_at DESC NULLS LAST
    LIMIT 1
),
base_targets AS (
    SELECT *
    FROM (
        VALUES
            ('GATEWAY_VALIDATOR_MODULE', '0xb7A893672189B46632109CC15De8986e2B8be1c6'),
            ('GATEWAY_QUOTE_MODULE', '0xfc70c24D9dC932572A067349E4D3A2eeb0280b31'),
            ('GATEWAY_EXECUTION_MODULE', '0x5852Bec7f3Ce38Ffdd8d1c9F48c88a620a9e6078')
    ) AS t(type, address)
)
UPDATE smart_contracts sc
SET is_active = FALSE,
    metadata = COALESCE(sc.metadata, '{}'::jsonb) || jsonb_build_object(
        'rolled_back_by', '000051_sync_gateway_runtime_modules_registry.down.sql',
        'rolled_back_at', NOW()
    ),
    updated_at = NOW()
FROM base_chain bc, base_targets bt
WHERE sc.chain_id = bc.id
  AND sc.deleted_at IS NULL
  AND sc.type = bt.type
  AND LOWER(sc.address) = LOWER(bt.address);

WITH arbitrum_chain AS (
    SELECT id
    FROM chains
    WHERE deleted_at IS NULL
      AND chain_id IN ('42161', 'eip155:42161')
    ORDER BY updated_at DESC NULLS LAST, created_at DESC NULLS LAST
    LIMIT 1
),
arbitrum_targets AS (
    SELECT *
    FROM (
        VALUES
            ('GATEWAY_VALIDATOR_MODULE', '0xaf65342c8d1b42650b88d737ce5b630f5487f7f0'),
            ('GATEWAY_QUOTE_MODULE', '0x6917d003add05eef125f3630fdae759c47f308bb'),
            ('GATEWAY_EXECUTION_MODULE', '0x62763108cd44c86c9b588f4defc2c66790fef34b')
    ) AS t(type, address)
)
UPDATE smart_contracts sc
SET is_active = FALSE,
    metadata = COALESCE(sc.metadata, '{}'::jsonb) || jsonb_build_object(
        'rolled_back_by', '000051_sync_gateway_runtime_modules_registry.down.sql',
        'rolled_back_at', NOW()
    ),
    updated_at = NOW()
FROM arbitrum_chain ac, arbitrum_targets at
WHERE sc.chain_id = ac.id
  AND sc.deleted_at IS NULL
  AND sc.type = at.type
  AND LOWER(sc.address) = LOWER(at.address);
