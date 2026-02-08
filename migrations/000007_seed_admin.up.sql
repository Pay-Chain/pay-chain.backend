-- 000007_seed_admin.up.sql

INSERT INTO users (id, email, name, password_hash, role, kyc_status, kyc_verified_at, created_at, updated_at)
VALUES (
    uuid_generate_v4(),
    'admin@paychain.io',
    'Admin PayChain',
    '$2a$10$TODO_GENERATE_BCRYPT_HASH_FOR_The.Conqueror-45______', -- TODO: REPLACE THIS with valid bcrypt hash for 'The.Conqueror-45'
    'admin',
    'fully_verified',
    NOW(),
    NOW(),
    NOW()
);

-- Seed admin merchant profile
INSERT INTO merchants (id, user_id, business_name, business_email, merchant_type, status, created_at, updated_at)
SELECT 
    uuid_generate_v4(),
    id,
    'PayChain Admin',
    'admin@paychain.io',
    'partner',
    'active',
    NOW(),
    NOW()
FROM users WHERE email = 'admin@paychain.io';
