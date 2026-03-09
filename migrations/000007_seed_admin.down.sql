-- 000007_seed_admin.down.sql

DELETE FROM merchants WHERE user_id IN (SELECT id FROM users WHERE email = 'admin@paymentkita.io');
DELETE FROM users WHERE email = 'admin@paymentkita.io';
