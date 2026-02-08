-- Migration: 000010_add_state_machine_id_to_chains
-- Description: Add state_machine_id column to chains table

ALTER TABLE chains ADD COLUMN IF NOT EXISTS state_machine_id VARCHAR(100) DEFAULT '';
