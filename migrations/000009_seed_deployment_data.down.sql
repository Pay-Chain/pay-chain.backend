-- 000009_seed_deployment_data.down.sql

DELETE FROM chain_rpcs WHERE chain_id IN (8453, 42161);
DELETE FROM smart_contracts WHERE chain_id IN (8453, 42161);
DELETE FROM supported_tokens WHERE chain_id IN (8453, 42161);
-- We do not delete tokens (USDE, etc.) as they might be used elsewhere or expected to persist.
-- We do not delete chains (8453, 42161) if we want to be safe, but strictly we could.
-- Let's delete the chains if they have no other data, but cleaner to just leave them or delete them.
DELETE FROM chains WHERE id IN (8453, 42161);
