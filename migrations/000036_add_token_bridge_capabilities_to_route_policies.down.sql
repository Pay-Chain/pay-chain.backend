DROP INDEX IF EXISTS idx_route_policies_status;

ALTER TABLE route_policies
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS bridge_token,
    DROP COLUMN IF EXISTS supports_privacy_forward,
    DROP COLUMN IF EXISTS supports_dest_swap,
    DROP COLUMN IF EXISTS supports_token_bridge;
