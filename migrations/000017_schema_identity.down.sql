-- Drop new tables
DROP TABLE IF EXISTS merchant_applications;
DROP TABLE IF EXISTS merchants;
DROP TABLE IF EXISTS merchant_fee_tiers;
DROP TABLE IF EXISTS password_resets;
DROP TABLE IF EXISTS email_verifications;
DROP TABLE IF EXISTS users;

-- Rename old tables back (Idempotent check)
DO $$
BEGIN
    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'merchant_applications_old') THEN
        ALTER TABLE merchant_applications_old RENAME TO merchant_applications;
    END IF;

    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'merchants_old') THEN
        ALTER TABLE merchants_old RENAME TO merchants;
    END IF;
    
    -- Restore merchant_fee_tiers
    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'merchant_fee_tiers_old') THEN
        ALTER TABLE merchant_fee_tiers_old RENAME TO merchant_fee_tiers;
    END IF;

    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'password_resets_old') THEN
        ALTER TABLE password_resets_old RENAME TO password_resets;
    END IF;

    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'email_verifications_old') THEN
        ALTER TABLE email_verifications_old RENAME TO email_verifications;
    END IF;

    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'users_old') THEN
        ALTER TABLE users_old RENAME TO users;
    END IF;
END $$;
