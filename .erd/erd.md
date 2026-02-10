ENUM user_role_enum {
  ADMIN
  SUB_ADMIN
  PARTNER
  USER
}

ENUM merchant_type_enum {
  PARTNER
  CORPORATE
  UMKM
  RETAIL
}

ENUM merchant_status_enum {
  PENDING
  ACTIVE
  SUSPENDED
  REJECTED
}

ENUM kyc_status_enum {
  NOT_STARTED
  ID_CARD_VERIFIED
  FACE_VERIFIED
  LIVENESS_VERIFIED
  FULLY_VERIFIED
}

ENUM chain_type_enum {
  EVM
  SVM
  MoveVM
  PolkaVM
  COSMOS
}

ENUM token_type_enum {
  NATIVE
  ERC20
  SPL
  COIN
}

ENUM payment_status_enum {
  PENDING
  PROCESSING
  COMPLETED
  FAILED
  REFUNDED
}

ENUM payment_request_status_enum {
  PENDING
  COMPLETED
  EXPIRED
  CANCELLED
}

ENUM job_status_enum {
  PENDING
  PROCESSING
  COMPLETED
  FAILED
}

Table schema_migrations {
  version numeric [pk]
  dirty boolean
  updated_at timestamp
}

Table users {
  id uuid [pk]
  name string
  email string
  password_hash string
  role user_role_enum
  kyc_status kyc_status_enum
  kyc_verivied_at timestamp
  deleted_at timestap
  created_at timestamp
  updated_at timestamp
}

Table email_verifications {
  id uuid [pk]
  user_id uuid [ref:> users.id]
  token string
  used_at timestap
  expires_at timestap
  deleted_at timestap
  created_at timestamp
  updated_at timestamp
}

Table password_resets {
  id uuid [pk]
  user_id uuid [ref:> users.id]
  token string
  used_at timestap
  expires_at timestap
  deleted_at timestap
  created_at timestamp
  updated_at timestamp
}

Table merchants {
  id uuid [pk]
  user_id uuid [ref:> users.id]
  business_name string
  business_email string
  merchant_type merchant_type_enum
  status merchant_status_enum
  tax_id string
  business_address string
  documents jsonb
  fee_discount_percent decimal
  verified_at timestamp
  deleted_at timestap
  created_at timestamp
  updated_at timestamp
}

Table merchant_applications {
  id uuid [pk]
  performed_by uuid [ref:> users.id]
  merchant_id uuid [ref:> merchants.id]
  action string
  reason string
  metadata string
  deleted_at timestap
  created_at timestamp
  updated_at timestamp
}

Table merchant_fee_tiers {
  id uuid [pk]
  merchant_type merchant_type_enum
  fee_discount_percent decimal
  min_monthly_volume decimal
  deleted_at timestap
  created_at timestamp
  updated_at timestamp
}

Table wallets {
  id uuid [pk]
  user_id uuid [ref:> users.id]
  merchant_id uuid [ref:> merchants.id]
  chain_id uuid [ref:> chains.id]
  address string
  is_primary boolean [default: false]
  kyc_verified boolean [default: false]
  kyc_required boolean [default: true]
  kyc_verified_at timestamp
  deleted_at timestap
  created_at timestamp
  updated_at timestamp
}

Table chains {
  id uuid [pk]
  chain_id string
  namespace string
  name string
  symbol string
  logo_url string
  chain_type chain_type_enum
  rpc_url string
  explorer_url string
  contract_address string
  ccip_router_address string
  hyperbridge_gateway string
  is_active boolean [default: true]
  deleted_at timestap
  created_at timestamp
  updated_at timestamp
}

Table chain_rpcs {
  id uuid [pk]
  chain_id uuid [ref:> chains.id]
  url string
  priority boolean
  is_active boolean
  error_count numeric
  last_error_at timestap
  deleted_at timestap
  created_at timestamp
  updated_at timestamp
}

Table tokens {
  id uuid [pk]
  chain_id uuid [ref:> chains.id]
  symbol string
  name string
  logo_url string
  type token_type_enum
  contract_address string
  decimal numeric
  min_amount numeric
  max_amount numeric
  deleted_at timestap
  created_at timestamp
  updated_at timestamp
}

Table fee_configs {
  id uuid [pk]
  chain_id uuid [ref:> chains.id]
  token_id uuid [ref:> tokens.id]
  platform_fee_percent decimal
  fixed_base_fee numeric
  min_fee numeric
  max_fee numeric
  deleted_at timestap
  created_at timestamp
  updated_at timestamp
}

Table smart_contracts_type {
  id uuid [pk]
  name string
  is_active true
  deleted_at timestap
  created_at timestamp
  updated_at timestamp
}

Table smart_contracts {
  id uuid [pk]
  chain_id uuid [ref:> chains.id]
  type_id uuid [ref: > smart_contracts_type.id]
  name string
  contract_address string
  abi jsonb
  start_block numeric
  deployer_address string
  version string [default: "0.0.1"]
  metadata jsonb
  hook_address string
  token0_address string
  token1_address string
  fee_tier numeric
  is_active boolean [default: true]
  deleted_at timestap
  created_at timestamp
  updated_at timestamp
}

Table payment_bridge {
  id uuid [pk]
  name string
  deleted_at timestap
  created_at timestamp
  updated_at timestamp
}

Table payment_requests {
  id uuid [pk]
  merchant_id uuid [ref:> merchants.id]
  chain_id uuid [ref:> chains.id]
  token_id uuid [ref:> tokens.id]
  wallet_address string
  amount numeric
  decimal numeric
  description string
  tx_hash string
  status payment_request_status_enum
  completed_at timestap
  expires_at timestap
  deleted_at timestap
  created_at timestamp
  updated_at timestamp
}

Table payments {
  id uuid [pk]
  sender_id uuid [ref:> users.id]
  merchant_id uuid [ref:> merchants.id]
  bridge_id uuid [ref:> payment_bridge.id]
  source_chain_id uuid [ref:> chains.id]
  dest_chain_id uuid [ref:> chains.id]
  source_token_id uuid [ref:> tokens.id]
  dest_token_id uuid [ref:> tokens.id]
  cross_chain_message_id string
  sender_address string
  dest_address string
  source_amount numeric
  dest_amount numeric
  fee_amount numeric
  total_charged numeric
  status payment_status_enum
  source_tx_hash string
  dest_tx_hash string
  refund_tx_hash string
  expires_at timestap
  deleted_at timestap
  created_at timestamp
  updated_at timestamp
}

Table payment_events {
  id uuid [pk]
  payment_id uuid [ref:> payments.id]
  chain_id uuid [ref:> chains.id]
  event_type string
  tx_hash string
  metadata jsonb
  deleted_at timestap
  created_at timestamp
  updated_at timestamp
}

Table bridge_configs {
  id uuid [pk]
  bridge_id uuid [ref:> payment_bridge.id]
  source_chain_id uuid [ref:> chains.id]
  dest_chain_id uuid [ref:> chains.id]
  router_address string
  fee_percentage decimal
  config jsonb
  is_active boolean [default: true]
  deleted_at timestap
  created_at timestamp
  updated_at timestamp
}

Table background_jobs {
  id uuid [pk]
  job_type string
  payload jsonb
  attempts integer
  error_message string
  status job_status_enum
  scheduled_at timestap
  started_at timestap
  deleted_at timestap
  created_at timestamp
  updated_at timestamp
}