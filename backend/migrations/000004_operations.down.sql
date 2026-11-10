-- Batch 4 rollback.
DROP TABLE IF EXISTS admin_action_logs;
DROP TABLE IF EXISTS notifications;
ALTER TABLE partner_settlements DROP COLUMN IF EXISTS payout_reference;
ALTER TABLE partner_settlements DROP COLUMN IF EXISTS paid_at;
