-- 000007: dispute SLA tracking. The worker sweeps OPEN disputes older than
-- DISPUTE_SLA_HOURS and notifies admins once (sla_notified_at stamp).
ALTER TABLE copyright_disputes ADD COLUMN IF NOT EXISTS sla_notified_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_disputes_sla_pending ON copyright_disputes (created_at)
  WHERE sla_notified_at IS NULL AND status IN ('OPEN', 'UNDER_REVIEW');
