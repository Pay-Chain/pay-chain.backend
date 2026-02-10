-- Drop new tables
DROP TABLE IF EXISTS background_jobs;
DROP TABLE IF EXISTS fee_configs;
DROP TABLE IF EXISTS bridge_configs;
DROP TABLE IF EXISTS payment_events;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS payment_requests;
DROP TABLE IF EXISTS payment_bridge;

-- Restore old tables
DO $$
BEGIN
    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'fee_configs_old') THEN
        ALTER TABLE fee_configs_old RENAME TO fee_configs;
    END IF;

    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'bridge_configs_old') THEN
        ALTER TABLE bridge_configs_old RENAME TO bridge_configs;
    END IF;

    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'payment_requests_old') THEN
        ALTER TABLE payment_requests_old RENAME TO payment_requests;
    END IF;

    IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'payments_old') THEN
        ALTER TABLE payments_old RENAME TO payments;
    END IF;
END $$;
