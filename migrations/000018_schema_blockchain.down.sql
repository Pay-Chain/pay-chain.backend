-- Drop new tables
DROP TABLE IF EXISTS smart_contracts;
DROP TABLE IF EXISTS wallets;
DROP TABLE IF EXISTS tokens;
DROP TABLE IF EXISTS chain_rpcs;
DROP TABLE IF EXISTS chains;

-- Restore old tables
DO $$
BEGIN
    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'smart_contracts_old') THEN
        ALTER TABLE smart_contracts_old RENAME TO smart_contracts;
    END IF;

    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'wallets_old') THEN
        ALTER TABLE wallets_old RENAME TO wallets;
    END IF;

    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'supported_tokens_old') THEN
        ALTER TABLE supported_tokens_old RENAME TO supported_tokens;
    END IF;

    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'tokens_old') THEN
        ALTER TABLE tokens_old RENAME TO tokens;
    END IF;

    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'chains_old') THEN
        ALTER TABLE chains_old RENAME TO chains;
    END IF;
END $$;
