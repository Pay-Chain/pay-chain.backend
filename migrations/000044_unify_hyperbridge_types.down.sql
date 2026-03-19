-- Reverse unification of Hyperbridge contract types (This is complex because we lose the original specific type)
-- This is a partial reversal if we assume specific logic or if we just want to revert the status.
-- However, since the goal is unification, a true 'down' would likely be manual recovery if needed.
-- For now, we'll leave it as a placeholder as the migration is intended to be permanent.
-- If needed, one could use original contract addresses to restore types, but that's out of scope for a simple down migration.
SELECT 1;
