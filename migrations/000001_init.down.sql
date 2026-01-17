-- 000001_init.down.sql
-- Rollback Pay-Chain Database Schema

-- Drop tables in reverse order
DROP TABLE IF EXISTS merchant_fee_tiers;
DROP TABLE IF EXISTS fee_configs;
DROP TABLE IF EXISTS bridge_configs;
DROP TABLE IF EXISTS payment_events;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS wallets;
DROP TABLE IF EXISTS supported_tokens;
DROP TABLE IF EXISTS tokens;
DROP TABLE IF EXISTS chains;
DROP TABLE IF EXISTS merchant_applications;
DROP TABLE IF EXISTS merchants;
DROP TABLE IF EXISTS password_resets;
DROP TABLE IF EXISTS email_verifications;
DROP TABLE IF EXISTS users;

-- Drop ENUM types
DROP TYPE IF EXISTS chain_type_enum;
DROP TYPE IF EXISTS payment_status_enum;
DROP TYPE IF EXISTS merchant_status_enum;
DROP TYPE IF EXISTS merchant_type_enum;
DROP TYPE IF EXISTS kyc_status_enum;
DROP TYPE IF EXISTS user_role_enum;
