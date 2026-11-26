-- 000028: hard license cap on store devices.
-- The service pre-check (live devices < license_count) races under
-- concurrent creates; this trigger is the backstop. It is BEFORE (not
-- AFTER): locking the business row before the insert serializes writers
-- without deadlocking against the FK's own row lock (an AFTER trigger
-- deadlocks: two writers each hold the FK KeyShare while requesting
-- FOR UPDATE). Only license_count live (non-REVOKED) devices per business.
CREATE OR REPLACE FUNCTION enforce_store_license_limit() RETURNS trigger AS $$
DECLARE
  live INT;
  cap  INT;
BEGIN
  SELECT license_count INTO cap FROM store_businesses WHERE id = NEW.business_id FOR UPDATE;
  IF cap IS NULL THEN
    RETURN NEW;
  END IF;
  SELECT COUNT(*) INTO live FROM store_devices
    WHERE business_id = NEW.business_id AND status <> 'REVOKED';
  IF live + 1 > cap THEN
    RAISE EXCEPTION 'store_license_limit_exceeded' USING ERRCODE = 'P0001';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_store_devices_license ON store_devices;
CREATE TRIGGER trg_store_devices_license BEFORE INSERT ON store_devices
  FOR EACH ROW EXECUTE FUNCTION enforce_store_license_limit();
