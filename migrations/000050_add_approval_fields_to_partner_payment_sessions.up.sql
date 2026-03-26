ALTER TABLE partner_payment_sessions 
ADD COLUMN instruction_approval_to VARCHAR(128) NULL,
ADD COLUMN instruction_approval_data_hex TEXT NULL;
