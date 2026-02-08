-- Migration: 000010_add_state_machine_id_to_chains (Down)
-- Description: Remove state_machine_id column from chains table

ALTER TABLE chains DROP COLUMN IF EXISTS state_machine_id;
