DROP INDEX IF EXISTS idx_disputes_sla_pending;
ALTER TABLE copyright_disputes DROP COLUMN IF EXISTS sla_notified_at;
