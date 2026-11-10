DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM partner_settlements WHERE status = 'HELD') THEN
    RAISE EXCEPTION 'cannot narrow settlement status: HELD rows exist';
  END IF;
END $$;
ALTER TABLE partner_settlements DROP CONSTRAINT IF EXISTS partner_settlements_status_check;
ALTER TABLE partner_settlements ADD CONSTRAINT partner_settlements_status_check
  CHECK (status IN ('PENDING', 'APPROVED', 'PAID', 'FORFEITED'));
ALTER TABLE partner_settlements
  DROP COLUMN IF EXISTS hold_reason,
  DROP COLUMN IF EXISTS held_at,
  DROP COLUMN IF EXISTS held_by;
