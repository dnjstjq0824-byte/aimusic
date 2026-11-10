-- Batch 4 schema: settlement payout record, notifications, admin audit.
-- All additive and idempotent; safe to run on existing volumes.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Settlement payout execution record (records the bank transfer; the transfer
-- itself stays manual until a payout provider is integrated).
ALTER TABLE partner_settlements ADD COLUMN IF NOT EXISTS paid_at TIMESTAMPTZ;
ALTER TABLE partner_settlements ADD COLUMN IF NOT EXISTS payout_reference TEXT;

-- In-app + mail-event notifications (takedown/counter/resolve/settlement).
CREATE TABLE IF NOT EXISTS notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  title_ko TEXT NOT NULL,
  body_ko TEXT NOT NULL DEFAULT '',
  link_path TEXT NOT NULL DEFAULT '',
  is_read BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT notifications_kind_check CHECK (
    kind IN ('DISPUTE_TAKEDOWN', 'DISPUTE_COUNTER', 'DISPUTE_RESOLVED',
             'SETTLEMENT_CALCULATED', 'SETTLEMENT_APPROVED', 'SETTLEMENT_PAID',
             'SUBSCRIPTION_VERIFIED', 'SUBSCRIPTION_REJECTED', 'NOTICE')
  )
);
CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications (user_id, created_at DESC);

-- Tamper-evident journal of admin decisions (actor + reason recorded).
CREATE TABLE IF NOT EXISTS admin_action_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  admin_user_id UUID REFERENCES users (id) ON DELETE SET NULL,
  action TEXT NOT NULL,
  target_type TEXT NOT NULL,
  target_id TEXT NOT NULL DEFAULT '',
  reason TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT admin_action_logs_action_check CHECK (
    action IN ('REVIEW_TRACK', 'REVIEW_PARTNER', 'REVIEW_SUBSCRIPTION',
               'POLICY_UPDATE', 'DISPUTE_RESOLVE', 'TAKEDOWN',
               'SETTLEMENT_CALCULATE', 'SETTLEMENT_APPROVE', 'SETTLEMENT_MARK_PAID',
               'USER_SET_ACTIVE', 'USER_SET_ROLE', 'CURATION_CREATE')
  )
);
CREATE INDEX IF NOT EXISTS idx_admin_action_logs_target
  ON admin_action_logs (target_type, target_id, created_at DESC);
