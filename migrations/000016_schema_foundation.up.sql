-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enable pgcrypto for gen_random_bytes
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Function to generate UUID v7
CREATE OR REPLACE FUNCTION uuid_generate_v7()
RETURNS uuid
AS $$
DECLARE
  unix_ts_ms bytea;
  uuid_bytes bytea;
BEGIN
  unix_ts_ms = substring(int8send(floor(extract(epoch from clock_timestamp()) * 1000)::bigint) from 3);
  uuid_bytes = unix_ts_ms || gen_random_bytes(10);
  uuid_bytes = set_byte(uuid_bytes, 6, (get_byte(uuid_bytes, 6) & x'0f'::int) | x'70'::int);
  uuid_bytes = set_byte(uuid_bytes, 8, (get_byte(uuid_bytes, 8) & x'3f'::int) | x'80'::int);
  RETURN encode(uuid_bytes, 'hex')::uuid;
END;
$$ LANGUAGE plpgsql;

-- Rename existing types to allow new strict types
DO $$
BEGIN
    -- user_role_enum
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role_enum') THEN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role_enum_old') THEN
            ALTER TYPE user_role_enum RENAME TO user_role_enum_old;
        END IF;
    END IF;
    -- merchant_type_enum
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'merchant_type_enum') THEN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'merchant_type_enum_old') THEN
            ALTER TYPE merchant_type_enum RENAME TO merchant_type_enum_old;
        END IF;
    END IF;
    -- merchant_status_enum
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'merchant_status_enum') THEN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'merchant_status_enum_old') THEN
            ALTER TYPE merchant_status_enum RENAME TO merchant_status_enum_old;
        END IF;
    END IF;
    -- kyc_status_enum
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'kyc_status_enum') THEN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'kyc_status_enum_old') THEN
            ALTER TYPE kyc_status_enum RENAME TO kyc_status_enum_old;
        END IF;
    END IF;
    -- chain_type_enum
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'chain_type_enum') THEN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'chain_type_enum_old') THEN
            ALTER TYPE chain_type_enum RENAME TO chain_type_enum_old;
        END IF;
    END IF;
    -- token_type_enum
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'token_type_enum') THEN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'token_type_enum_old') THEN
            ALTER TYPE token_type_enum RENAME TO token_type_enum_old;
        END IF;
    END IF;
    -- payment_status_enum
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_status_enum') THEN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_status_enum_old') THEN
            ALTER TYPE payment_status_enum RENAME TO payment_status_enum_old;
        END IF;
    END IF;
    -- payment_request_status_enum
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_request_status_enum') THEN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_request_status_enum_old') THEN
            ALTER TYPE payment_request_status_enum RENAME TO payment_request_status_enum_old;
        END IF;
    END IF;
    -- job_status_enum
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'job_status_enum') THEN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'job_status_enum_old') THEN
            ALTER TYPE job_status_enum RENAME TO job_status_enum_old;
        END IF;
    END IF;
END$$;

-- Create New ENUMs (Idempotent)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role_enum') THEN
        CREATE TYPE user_role_enum AS ENUM ('ADMIN', 'SUB_ADMIN', 'PARTNER', 'USER');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'merchant_type_enum') THEN
        CREATE TYPE merchant_type_enum AS ENUM ('PARTNER', 'CORPORATE', 'UMKM', 'RETAIL');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'merchant_status_enum') THEN
        CREATE TYPE merchant_status_enum AS ENUM ('PENDING', 'ACTIVE', 'SUSPENDED', 'REJECTED');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'kyc_status_enum') THEN
        CREATE TYPE kyc_status_enum AS ENUM ('NOT_STARTED', 'ID_CARD_VERIFIED', 'FACE_VERIFIED', 'LIVENESS_VERIFIED', 'FULLY_VERIFIED');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'chain_type_enum') THEN
        CREATE TYPE chain_type_enum AS ENUM ('EVM', 'SVM', 'MoveVM', 'PolkaVM', 'COSMOS');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'token_type_enum') THEN
        CREATE TYPE token_type_enum AS ENUM ('NATIVE', 'ERC20', 'SPL', 'COIN');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_status_enum') THEN
        CREATE TYPE payment_status_enum AS ENUM ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED', 'REFUNDED');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_request_status_enum') THEN
        CREATE TYPE payment_request_status_enum AS ENUM ('PENDING', 'COMPLETED', 'EXPIRED', 'CANCELLED');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'job_status_enum') THEN
        CREATE TYPE job_status_enum AS ENUM ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED');
    END IF;
END$$;
