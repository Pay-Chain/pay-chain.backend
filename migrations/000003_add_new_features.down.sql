-- Rollback: 000003_add_new_features

-- Drop background jobs
DROP TABLE IF EXISTS background_jobs;

-- Remove wallet KYC columns
ALTER TABLE wallets DROP COLUMN IF EXISTS kyc_verified;
ALTER TABLE wallets DROP COLUMN IF EXISTS kyc_verified_at;
ALTER TABLE wallets DROP COLUMN IF EXISTS kyc_required;

-- Drop payment requests
DROP TABLE IF EXISTS payment_requests;
DROP TYPE IF EXISTS payment_request_status;

-- Drop RPC endpoints
DROP TABLE IF EXISTS rpc_endpoints;
