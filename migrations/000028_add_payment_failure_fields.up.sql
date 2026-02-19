ALTER TABLE payments
ADD COLUMN failure_reason TEXT,
ADD COLUMN revert_data TEXT;
