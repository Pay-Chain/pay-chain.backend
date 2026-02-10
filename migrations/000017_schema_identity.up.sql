-- Resilient Version of Phase 2 Migration
-- Handles renaming tables idempotently to crash-proof against dirty states.

-- 1. Rename old tables (Safety check)
DO $$
BEGIN
    -- users
    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'users') THEN
        IF NOT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'users_old') THEN
            ALTER TABLE users RENAME TO users_old;
        END IF;
    END IF;

    -- email_verifications
    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'email_verifications') THEN
        IF NOT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'email_verifications_old') THEN
            ALTER TABLE email_verifications RENAME TO email_verifications_old;
        END IF;
    END IF;

    -- password_resets
    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'password_resets') THEN
         IF NOT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'password_resets_old') THEN
            ALTER TABLE password_resets RENAME TO password_resets_old;
        END IF;
    END IF;

    -- merchants
    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'merchants') THEN
         IF NOT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'merchants_old') THEN
            ALTER TABLE merchants RENAME TO merchants_old;
        END IF;
    END IF;

    -- merchant_applications
    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'merchant_applications') THEN
         IF NOT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'merchant_applications_old') THEN
            ALTER TABLE merchant_applications RENAME TO merchant_applications_old;
        END IF;
    END IF;

    -- merchant_fee_tiers (THE FIX)
    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'merchant_fee_tiers') THEN
         IF NOT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'merchant_fee_tiers_old') THEN
            ALTER TABLE merchant_fee_tiers RENAME TO merchant_fee_tiers_old;
        END IF;
    END IF;
END $$;

-- 2. Create New Tables (IF NOT EXISTS to allow retries)

-- Users
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name VARCHAR(255),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    role user_role_enum NOT NULL DEFAULT 'USER',
    kyc_status kyc_status_enum NOT NULL DEFAULT 'NOT_STARTED',
    kyc_verified_at TIMESTAMP,
    deleted_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Email Verifications
CREATE TABLE IF NOT EXISTS email_verifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id),
    token VARCHAR(255),
    used_at TIMESTAMP,
    expires_at TIMESTAMP,
    deleted_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Password Resets
CREATE TABLE IF NOT EXISTS password_resets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id),
    token VARCHAR(255),
    used_at TIMESTAMP,
    expires_at TIMESTAMP,
    deleted_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Merchant Fee Tiers
CREATE TABLE IF NOT EXISTS merchant_fee_tiers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    merchant_type merchant_type_enum NOT NULL,
    fee_discount_percent DECIMAL(5,2),
    min_monthly_volume DECIMAL(36,18),
    deleted_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Merchants
CREATE TABLE IF NOT EXISTS merchants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id),
    business_name VARCHAR(255),
    business_email VARCHAR(255),
    merchant_type merchant_type_enum NOT NULL,
    status merchant_status_enum NOT NULL DEFAULT 'PENDING',
    tax_id VARCHAR(50),
    business_address TEXT,
    documents JSONB,
    fee_discount_percent DECIMAL(5,2),
    verified_at TIMESTAMP,
    deleted_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Merchant Applications
CREATE TABLE IF NOT EXISTS merchant_applications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    performed_by UUID REFERENCES users(id),
    merchant_id UUID REFERENCES merchants(id),
    action VARCHAR(255),
    reason VARCHAR(255),
    metadata JSONB,
    deleted_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 3. Migrate Data (ON CONFLICT DO NOTHING to allow retries)

-- Users Data
INSERT INTO users (id, name, email, password_hash, role, kyc_status, kyc_verified_at, deleted_at, created_at, updated_at)
SELECT
    id,
    name,
    email,
    password_hash,
    UPPER(role::text)::user_role_enum,
    UPPER(kyc_status::text)::kyc_status_enum,
    kyc_verified_at,
    deleted_at,
    created_at,
    updated_at
FROM users_old
ON CONFLICT (id) DO NOTHING;

-- Email Verifications Data
INSERT INTO email_verifications (id, user_id, token, used_at, expires_at, deleted_at, created_at)
SELECT id, user_id, token, verified_at, expires_at, deleted_at, created_at
FROM email_verifications_old
ON CONFLICT (id) DO NOTHING;

-- Password Resets Data
INSERT INTO password_resets (id, user_id, token, used_at, expires_at, deleted_at, created_at)
SELECT id, user_id, token, used_at, expires_at, deleted_at, created_at
FROM password_resets_old
ON CONFLICT (id) DO NOTHING;

-- Merchant Fee Tiers Data (Seed)
INSERT INTO merchant_fee_tiers (merchant_type, fee_discount_percent, min_monthly_volume) VALUES
    ('PARTNER', 50.00, 100000),
    ('CORPORATE', 30.00, 50000),
    ('UMKM', 15.00, 5000),
    ('RETAIL', 0.00, 0)
ON CONFLICT DO NOTHING; -- No primary key constraint on values, but good to be safe. Actually UUID PK determines uniqueness.
-- Wait, hardcoded seed does not have IDs. It will create new IDs every time this runs.
-- Let's use specific TRUNCATE for seed table or check existance.
-- Better to just leave it. If duplicates exist, it's fine for now, or we can clear table first.
TRUNCATE TABLE merchant_fee_tiers;
INSERT INTO merchant_fee_tiers (merchant_type, fee_discount_percent, min_monthly_volume) VALUES
    ('PARTNER', 50.00, 100000),
    ('CORPORATE', 30.00, 50000),
    ('UMKM', 15.00, 5000),
    ('RETAIL', 0.00, 0);

-- Merchants Data
INSERT INTO merchants (id, user_id, business_name, business_email, merchant_type, status, tax_id, business_address, documents, fee_discount_percent, verified_at, deleted_at, created_at, updated_at)
SELECT
    id,
    user_id,
    business_name,
    business_email,
    UPPER(merchant_type::text)::merchant_type_enum,
    UPPER(status::text)::merchant_status_enum,
    tax_id,
    business_address,
    documents,
    fee_discount_percent,
    verified_at,
    deleted_at,
    created_at,
    updated_at
FROM merchants_old
ON CONFLICT (id) DO NOTHING;

-- Merchant Applications Data
INSERT INTO merchant_applications (id, performed_by, merchant_id, action, reason, metadata, created_at)
SELECT id, performed_by, merchant_id, action, reason, metadata, created_at
FROM merchant_applications_old
ON CONFLICT (id) DO NOTHING;
