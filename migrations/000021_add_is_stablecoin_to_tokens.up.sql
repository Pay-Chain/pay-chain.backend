-- Add is_stablecoin column to tokens table
ALTER TABLE tokens ADD COLUMN IF NOT EXISTS is_stablecoin BOOLEAN DEFAULT FALSE;
