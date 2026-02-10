-- Drop new ENUMs
DROP TYPE IF EXISTS job_status_enum;
DROP TYPE IF EXISTS payment_request_status_enum;
DROP TYPE IF EXISTS payment_status_enum;
DROP TYPE IF EXISTS token_type_enum;
DROP TYPE IF EXISTS chain_type_enum;
DROP TYPE IF EXISTS kyc_status_enum;
DROP TYPE IF EXISTS merchant_status_enum;
DROP TYPE IF EXISTS merchant_type_enum;
DROP TYPE IF EXISTS user_role_enum;

-- Rename old ENUMs back
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role_enum_old') THEN
        ALTER TYPE user_role_enum_old RENAME TO user_role_enum;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'merchant_type_enum_old') THEN
        ALTER TYPE merchant_type_enum_old RENAME TO merchant_type_enum;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'merchant_status_enum_old') THEN
        ALTER TYPE merchant_status_enum_old RENAME TO merchant_status_enum;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'kyc_status_enum_old') THEN
        ALTER TYPE kyc_status_enum_old RENAME TO kyc_status_enum;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'chain_type_enum_old') THEN
        ALTER TYPE chain_type_enum_old RENAME TO chain_type_enum;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'token_type_enum_old') THEN
        ALTER TYPE token_type_enum_old RENAME TO token_type_enum;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_status_enum_old') THEN
        ALTER TYPE payment_status_enum_old RENAME TO payment_status_enum;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_request_status_enum_old') THEN
        ALTER TYPE payment_request_status_enum_old RENAME TO payment_request_status_enum;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'job_status_enum_old') THEN
        ALTER TYPE job_status_enum_old RENAME TO job_status_enum;
    END IF;
END$$;

-- Drop function
DROP FUNCTION IF EXISTS uuid_generate_v7();
