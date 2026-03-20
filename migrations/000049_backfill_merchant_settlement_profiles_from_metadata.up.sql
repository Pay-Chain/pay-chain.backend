INSERT INTO merchant_settlement_profiles (
    id,
    merchant_id,
    invoice_currency,
    dest_chain,
    dest_token,
    dest_wallet,
    bridge_token_symbol,
    created_at,
    updated_at
)
SELECT
    uuid_generate_v7(),
    m.id,
    COALESCE(NULLIF(m.metadata ->> 'invoice_currency', ''), NULLIF(m.metadata ->> 'invoiceCurrency', '')),
    COALESCE(NULLIF(m.metadata ->> 'dest_chain', ''), NULLIF(m.metadata ->> 'destChain', '')),
    COALESCE(NULLIF(m.metadata ->> 'dest_token', ''), NULLIF(m.metadata ->> 'destToken', '')),
    COALESCE(
        NULLIF(m.metadata ->> 'dest_wallet', ''),
        NULLIF(m.metadata ->> 'destWallet', ''),
        NULLIF(m.metadata ->> 'wallet_address', '')
    ),
    COALESCE(NULLIF(m.metadata ->> 'bridge_token_symbol', ''), NULLIF(m.metadata ->> 'bridgeTokenSymbol', ''), 'USDC'),
    NOW(),
    NOW()
FROM merchants m
LEFT JOIN merchant_settlement_profiles msp ON msp.merchant_id = m.id AND msp.deleted_at IS NULL
WHERE m.deleted_at IS NULL
  AND msp.id IS NULL
  AND COALESCE(NULLIF(m.metadata ->> 'invoice_currency', ''), NULLIF(m.metadata ->> 'invoiceCurrency', '')) IS NOT NULL
  AND COALESCE(NULLIF(m.metadata ->> 'dest_chain', ''), NULLIF(m.metadata ->> 'destChain', '')) IS NOT NULL
  AND COALESCE(NULLIF(m.metadata ->> 'dest_token', ''), NULLIF(m.metadata ->> 'destToken', '')) IS NOT NULL
  AND COALESCE(
        NULLIF(m.metadata ->> 'dest_wallet', ''),
        NULLIF(m.metadata ->> 'destWallet', ''),
        NULLIF(m.metadata ->> 'wallet_address', '')
  ) IS NOT NULL;
