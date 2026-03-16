-- migration: 000043_phase_8_audit.up.sql

-- 1. Table for Audit logging of search/resolve attempts
CREATE TABLE pk_resolve_audit (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL,
    wallet_address VARCHAR(64),
    risk_score INTEGER,
    risk_level VARCHAR(20), -- e.g., 'LOW', 'MEDIUM', 'HIGH'
    user_agent TEXT,
    ip_address INET,
    status VARCHAR(20), -- e.g., 'SUCCESS', 'BLOCKED', 'FAILED'
    reason TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 2. Index for faster lookups by session and wallet
CREATE INDEX idx_pk_resolve_audit_session_id ON pk_resolve_audit(session_id);
CREATE INDEX idx_pk_resolve_audit_wallet_address ON pk_resolve_audit(wallet_address);
CREATE INDEX idx_pk_resolve_audit_created_at ON pk_resolve_audit(created_at DESC);
