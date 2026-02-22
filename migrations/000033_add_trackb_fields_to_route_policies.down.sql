ALTER TABLE route_policies
    DROP COLUMN IF EXISTS max_fee,
    DROP COLUMN IF EXISTS min_fee,
    DROP COLUMN IF EXISTS overhead_bytes,
    DROP COLUMN IF EXISTS per_byte_rate;

