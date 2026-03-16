-- 2.1.1: Merchant Profile Expansion
-- We add critical technical fields for external integration.
ALTER TABLE merchants 
ADD COLUMN callback_url TEXT,
ADD COLUMN webhook_secret VARCHAR(64) DEFAULT encode(gen_random_bytes(32), 'hex'),
ADD COLUMN webhook_is_active BOOLEAN DEFAULT false,
ADD COLUMN support_email VARCHAR(255),
ADD COLUMN logo_url TEXT,
ADD COLUMN webhook_metadata JSONB DEFAULT '{}',
ADD COLUMN metadata JSONB DEFAULT '{}';

-- 2.1.2: Webhook Delivery Engine States
-- A robust notification system needs defined lifecycle states.
CREATE TYPE webhook_delivery_status AS ENUM (
    'pending',      -- Log created but not yet attempted
    'delivering',   -- Currently being processed by a worker
    'delivered',    -- Received 2xx response
    'retrying',     -- Failed, scheduled for another attempt
    'failed',       -- Permanent failure (e.g., 404 or invalid URL)
    'dropped'       -- Reached max retries
);

-- 2.1.3: Webhook Dispatch Logs (Persistence Layer)
-- This table acts as both a queue for the worker and an audit trail.
CREATE TABLE webhook_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    merchant_id UUID NOT NULL REFERENCES merchants(id),
    payment_id UUID NOT NULL REFERENCES payments(id),
    event_type VARCHAR(50) NOT NULL, -- e.g., 'payment.completed', 'payment.failed'
    payload JSONB NOT NULL,
    delivery_status webhook_delivery_status DEFAULT 'pending',
    http_status INTEGER,
    response_body TEXT,
    retry_count INTEGER DEFAULT 0,
    next_retry_at TIMESTAMP,
    last_attempt_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 2.1.4: Optimization Indexes (Performance at Scale)
-- We index logs by merchant for the dashboard and status for the worker.
CREATE INDEX idx_webhook_logs_merchant_history ON webhook_logs(merchant_id, created_at DESC);
CREATE INDEX idx_webhook_logs_dispatcher_queue ON webhook_logs(delivery_status, next_retry_at) 
WHERE delivery_status IN ('pending', 'retrying');

-- 2.1.5: Payment Attribution (Linkages)
-- Ensure the payments table can track which business it belongs to.
DO $$ 
BEGIN 
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='payments' AND column_name='merchant_id') THEN
        ALTER TABLE payments ADD COLUMN merchant_id UUID REFERENCES merchants(id);
        CREATE INDEX idx_payments_merchant_id ON payments(merchant_id);
    END IF;
END $$;
