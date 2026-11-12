-- 000018: refund policy (2026-11-12): 청약철회 / 중도해지 / 일반해지.
-- Additive only.

ALTER TABLE billing_orders ADD COLUMN IF NOT EXISTS refunded_amount_krw INT
  CHECK (refunded_amount_krw IS NULL OR refunded_amount_krw >= 0);
ALTER TABLE billing_orders ADD COLUMN IF NOT EXISTS refunded_at TIMESTAMPTZ;

ALTER TABLE billing_orders DROP CONSTRAINT IF EXISTS billing_orders_status_check;
ALTER TABLE billing_orders ADD CONSTRAINT billing_orders_status_check
  CHECK (status IN ('PENDING', 'PAID', 'FAILED', 'CANCELLED', 'REFUNDED', 'PARTIALLY_REFUNDED'));

-- 일반해지: access continues until current_period_end.
ALTER TABLE billing_subscriptions ADD COLUMN IF NOT EXISTS cancel_requested_at TIMESTAMPTZ;
