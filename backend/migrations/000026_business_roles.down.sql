-- WI-R10 rollback. Fails closed when business-owned data exists:
-- partner_id SET NOT NULL errors if any business-created (partner_id NULL)
-- store row exists — delete or re-link those rows first, then retry.
-- user_roles/business_profiles rows are dropped with their tables.
DROP TRIGGER IF EXISTS trg_business_profiles_updated_at ON business_profiles;
DROP TABLE IF EXISTS business_profiles;
DROP TABLE IF EXISTS user_roles;
ALTER TABLE store_businesses DROP COLUMN IF EXISTS owner_user_id;
ALTER TABLE store_businesses ALTER COLUMN partner_id SET NOT NULL;
UPDATE users SET role = 'USER' WHERE role = 'BUSINESS';
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
  CHECK (role IN ('ADMIN', 'PARTNER', 'USER'));
