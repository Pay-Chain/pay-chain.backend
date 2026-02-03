-- Undo Seeding
DELETE FROM smart_contracts WHERE chain_id IN ('eip155:11155111', 'solana:103');
DELETE FROM chains WHERE id IN (11155111, 103);
