-- 000014: P2-2 (#12) settlement hold for confirmed fraud.
-- A PENDING settlement can be HELD with a reason + actor; release returns
-- it to PENDING. Approve/pay never move a HELD row.
ALTER TABLE partner_settlements
  ADD COLUMN IF NOT EXISTS hold_reason TEXT,
  ADD COLUMN IF NOT EXISTS held_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS held_by UUID REFERENCES users(id) ON DELETE SET NULL;
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'partner_settlements_status_check') THEN
    ALTER TABLE partner_settlements DROP CONSTRAINT partner_settlements_status_check;
  END IF;
END $$;
ALTER TABLE partner_settlements ADD CONSTRAINT partner_settlements_status_check
  CHECK (status IN ('PENDING', 'APPROVED', 'PAID', 'FORFEITED', 'HELD'));
