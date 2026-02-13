-- Remove is_stablecoin column from tokens table
ALTER TABLE tokens DROP COLUMN IF EXISTS is_stablecoin;
