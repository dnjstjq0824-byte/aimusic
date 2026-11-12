-- Reverts 000018. Prorated refunds collapse to REFUNDED so the old CHECK holds.
UPDATE billing_orders SET status = 'REFUNDED' WHERE status = 'PARTIALLY_REFUNDED';
ALTER TABLE billing_orders DROP CONSTRAINT IF EXISTS billing_orders_status_check;
ALTER TABLE billing_orders ADD CONSTRAINT billing_orders_status_check
  CHECK (status IN ('PENDING', 'PAID', 'FAILED', 'CANCELLED', 'REFUNDED'));
ALTER TABLE billing_orders DROP COLUMN IF EXISTS refunded_at;
ALTER TABLE billing_orders DROP COLUMN IF EXISTS refunded_amount_krw;
ALTER TABLE billing_subscriptions DROP COLUMN IF EXISTS cancel_requested_at;
