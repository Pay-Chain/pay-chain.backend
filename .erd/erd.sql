CREATE TYPE "user_role_enum" AS ENUM (
  'ADMIN',
  'SUB_ADMIN',
  'PARTNER',
  'USER'
);

CREATE TYPE "merchant_type_enum" AS ENUM (
  'PARTNER',
  'CORPORATE',
  'UMKM',
  'RETAIL'
);

CREATE TYPE "merchant_status_enum" AS ENUM (
  'PENDING',
  'ACTIVE',
  'SUSPENDED',
  'REJECTED'
);

CREATE TYPE "kyc_status_enum" AS ENUM (
  'NOT_STARTED',
  'ID_CARD_VERIFIED',
  'FACE_VERIFIED',
  'LIVENESS_VERIFIED',
  'FULLY_VERIFIED'
);

CREATE TYPE "chain_type_enum" AS ENUM (
  'EVM',
  'SVM',
  'MoveVM',
  'PolkaVM',
  'COSMOS'
);

CREATE TYPE "token_type_enum" AS ENUM (
  'NATIVE',
  'ERC20',
  'SPL',
  'COIN'
);

CREATE TYPE "payment_status_enum" AS ENUM (
  'PENDING',
  'PROCESSING',
  'COMPLETED',
  'FAILED',
  'REFUNDED'
);

CREATE TYPE "payment_request_status_enum" AS ENUM (
  'PENDING',
  'COMPLETED',
  'EXPIRED',
  'CANCELLED'
);

CREATE TYPE "job_status_enum" AS ENUM (
  'PENDING',
  'PROCESSING',
  'COMPLETED',
  'FAILED'
);

CREATE TABLE "schema_migrations" (
  "version" numeric PRIMARY KEY,
  "dirty" boolean,
  "updated_at" timestamp
);

CREATE TABLE "users" (
  "id" uuid PRIMARY KEY,
  "name" string,
  "email" string,
  "password_hash" string,
  "role" user_role_enum,
  "kyc_status" kyc_status_enum,
  "kyc_verivied_at" timestamp,
  "deleted_at" timestap,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "email_verifications" (
  "id" uuid PRIMARY KEY,
  "user_id" uuid,
  "token" string,
  "used_at" timestap,
  "expires_at" timestap,
  "deleted_at" timestap,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "password_resets" (
  "id" uuid PRIMARY KEY,
  "user_id" uuid,
  "token" string,
  "used_at" timestap,
  "expires_at" timestap,
  "deleted_at" timestap,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "merchants" (
  "id" uuid PRIMARY KEY,
  "user_id" uuid,
  "business_name" string,
  "business_email" string,
  "merchant_type" merchant_type_enum,
  "status" merchant_status_enum,
  "tax_id" string,
  "business_address" string,
  "documents" jsonb,
  "fee_discount_percent" decimal,
  "verified_at" timestamp,
  "deleted_at" timestap,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "merchant_applications" (
  "id" uuid PRIMARY KEY,
  "performed_by" uuid,
  "merchant_id" uuid,
  "action" string,
  "reason" string,
  "metadata" string,
  "deleted_at" timestap,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "merchant_fee_tiers" (
  "id" uuid PRIMARY KEY,
  "merchant_type" merchant_type_enum,
  "fee_discount_percent" decimal,
  "min_monthly_volume" decimal,
  "deleted_at" timestap,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "wallets" (
  "id" uuid PRIMARY KEY,
  "user_id" uuid,
  "merchant_id" uuid,
  "chain_id" uuid,
  "address" string,
  "is_primary" boolean DEFAULT false,
  "kyc_verified" boolean DEFAULT false,
  "kyc_required" boolean DEFAULT true,
  "kyc_verified_at" timestamp,
  "deleted_at" timestap,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "chains" (
  "id" uuid PRIMARY KEY,
  "chain_id" string,
  "namespace" string,
  "name" string,
  "symbol" string,
  "logo_url" string,
  "chain_type" chain_type_enum,
  "rpc_url" string,
  "explorer_url" string,
  "contract_address" string,
  "ccip_router_address" string,
  "hyperbridge_gateway" string,
  "is_active" boolean DEFAULT true,
  "deleted_at" timestap,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "chain_rpcs" (
  "id" uuid PRIMARY KEY,
  "chain_id" uuid,
  "url" string,
  "priority" boolean,
  "is_active" boolean,
  "error_count" numeric,
  "last_error_at" timestap,
  "deleted_at" timestap,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "tokens" (
  "id" uuid PRIMARY KEY,
  "chain_id" uuid,
  "symbol" string,
  "name" string,
  "logo_url" string,
  "type" token_type_enum,
  "contract_address" string,
  "decimal" numeric,
  "min_amount" numeric,
  "max_amount" numeric,
  "deleted_at" timestap,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "fee_configs" (
  "id" uuid PRIMARY KEY,
  "chain_id" uuid,
  "token_id" uuid,
  "platform_fee_percent" decimal,
  "fixed_base_fee" numeric,
  "min_fee" numeric,
  "max_fee" numeric,
  "deleted_at" timestap,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "smart_contracts_type" (
  "id" uuid PRIMARY KEY,
  "name" string,
  "is_active" true,
  "deleted_at" timestap,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "smart_contracts" (
  "id" uuid PRIMARY KEY,
  "chain_id" uuid,
  "type_id" uuid,
  "name" string,
  "contract_address" string,
  "abi" jsonb,
  "start_block" numeric,
  "deployer_address" string,
  "version" string DEFAULT '0.0.1',
  "metadata" jsonb,
  "hook_address" string,
  "token0_address" string,
  "token1_address" string,
  "fee_tier" numeric,
  "is_active" boolean DEFAULT true,
  "deleted_at" timestap,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "payment_bridge" (
  "id" uuid PRIMARY KEY,
  "name" string,
  "deleted_at" timestap,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "payment_requests" (
  "id" uuid PRIMARY KEY,
  "merchant_id" uuid,
  "chain_id" uuid,
  "token_id" uuid,
  "wallet_address" string,
  "amount" numeric,
  "decimal" numeric,
  "description" string,
  "tx_hash" string,
  "status" payment_request_status_enum,
  "completed_at" timestap,
  "expires_at" timestap,
  "deleted_at" timestap,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "payments" (
  "id" uuid PRIMARY KEY,
  "sender_id" uuid,
  "merchant_id" uuid,
  "bridge_id" uuid,
  "source_chain_id" uuid,
  "dest_chain_id" uuid,
  "source_token_id" uuid,
  "dest_token_id" uuid,
  "cross_chain_message_id" string,
  "sender_address" string,
  "dest_address" string,
  "source_amount" numeric,
  "dest_amount" numeric,
  "fee_amount" numeric,
  "total_charged" numeric,
  "status" payment_status_enum,
  "source_tx_hash" string,
  "dest_tx_hash" string,
  "refund_tx_hash" string,
  "expires_at" timestap,
  "deleted_at" timestap,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "payment_events" (
  "id" uuid PRIMARY KEY,
  "payment_id" uuid,
  "chain_id" uuid,
  "event_type" string,
  "tx_hash" string,
  "metadata" jsonb,
  "deleted_at" timestap,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "bridge_configs" (
  "id" uuid PRIMARY KEY,
  "bridge_id" uuid,
  "source_chain_id" uuid,
  "dest_chain_id" uuid,
  "router_address" string,
  "fee_percentage" decimal,
  "config" jsonb,
  "is_active" boolean DEFAULT true,
  "deleted_at" timestap,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "background_jobs" (
  "id" uuid PRIMARY KEY,
  "job_type" string,
  "payload" jsonb,
  "attempts" integer,
  "error_message" string,
  "status" job_status_enum,
  "scheduled_at" timestap,
  "started_at" timestap,
  "deleted_at" timestap,
  "created_at" timestamp,
  "updated_at" timestamp
);

ALTER TABLE "email_verifications" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "password_resets" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "merchants" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "merchant_applications" ADD FOREIGN KEY ("performed_by") REFERENCES "users" ("id");

ALTER TABLE "merchant_applications" ADD FOREIGN KEY ("merchant_id") REFERENCES "merchants" ("id");

ALTER TABLE "wallets" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "wallets" ADD FOREIGN KEY ("merchant_id") REFERENCES "merchants" ("id");

ALTER TABLE "wallets" ADD FOREIGN KEY ("chain_id") REFERENCES "chains" ("id");

ALTER TABLE "chain_rpcs" ADD FOREIGN KEY ("chain_id") REFERENCES "chains" ("id");

ALTER TABLE "tokens" ADD FOREIGN KEY ("chain_id") REFERENCES "chains" ("id");

ALTER TABLE "fee_configs" ADD FOREIGN KEY ("chain_id") REFERENCES "chains" ("id");

ALTER TABLE "fee_configs" ADD FOREIGN KEY ("token_id") REFERENCES "tokens" ("id");

ALTER TABLE "smart_contracts" ADD FOREIGN KEY ("chain_id") REFERENCES "chains" ("id");

ALTER TABLE "smart_contracts" ADD FOREIGN KEY ("type_id") REFERENCES "smart_contracts_type" ("id");

ALTER TABLE "payment_requests" ADD FOREIGN KEY ("merchant_id") REFERENCES "merchants" ("id");

ALTER TABLE "payment_requests" ADD FOREIGN KEY ("chain_id") REFERENCES "chains" ("id");

ALTER TABLE "payment_requests" ADD FOREIGN KEY ("token_id") REFERENCES "tokens" ("id");

ALTER TABLE "payments" ADD FOREIGN KEY ("sender_id") REFERENCES "users" ("id");

ALTER TABLE "payments" ADD FOREIGN KEY ("merchant_id") REFERENCES "merchants" ("id");

ALTER TABLE "payments" ADD FOREIGN KEY ("bridge_id") REFERENCES "payment_bridge" ("id");

ALTER TABLE "payments" ADD FOREIGN KEY ("source_chain_id") REFERENCES "chains" ("id");

ALTER TABLE "payments" ADD FOREIGN KEY ("dest_chain_id") REFERENCES "chains" ("id");

ALTER TABLE "payments" ADD FOREIGN KEY ("source_token_id") REFERENCES "tokens" ("id");

ALTER TABLE "payments" ADD FOREIGN KEY ("dest_token_id") REFERENCES "tokens" ("id");

ALTER TABLE "payment_events" ADD FOREIGN KEY ("payment_id") REFERENCES "payments" ("id");

ALTER TABLE "payment_events" ADD FOREIGN KEY ("chain_id") REFERENCES "chains" ("id");

ALTER TABLE "bridge_configs" ADD FOREIGN KEY ("bridge_id") REFERENCES "payment_bridge" ("id");

ALTER TABLE "bridge_configs" ADD FOREIGN KEY ("source_chain_id") REFERENCES "chains" ("id");

ALTER TABLE "bridge_configs" ADD FOREIGN KEY ("dest_chain_id") REFERENCES "chains" ("id");
