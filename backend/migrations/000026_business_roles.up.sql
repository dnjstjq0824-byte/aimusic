-- WI-R10: BUSINESS role + multi-role grants + business profiles + store owner.
-- R0 decision (b): users.role stays the primary role; user_roles carries
-- additive capability grants so a store-owning PARTNER can also reach the
-- business console. Store authority moves from partner_profiles to the owning
-- user (owner_user_id); partner_id stays as a legacy link and turns nullable
-- for business-created stores. Local-only verification; prod apply needs
-- G-ROLE (dry-run report first, see scripts/role-migration-dryrun.sql).

-- 1. Primary role gains BUSINESS.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
  CHECK (role IN ('ADMIN', 'PARTNER', 'USER', 'BUSINESS'));

-- 2. Additive role grants (also the G-ROLE migration log source).
CREATE TABLE IF NOT EXISTS user_roles (
  user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  role TEXT NOT NULL CHECK (role IN ('ADMIN', 'PARTNER', 'USER', 'BUSINESS')),
  granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  source TEXT NOT NULL DEFAULT 'manual'
    CHECK (source IN ('backfill', 'migration', 'register', 'admin', 'manual')),
  CONSTRAINT user_roles_pkey PRIMARY KEY (user_id, role)
);
CREATE INDEX IF NOT EXISTS idx_user_roles_role ON user_roles (role);
-- Every account keeps its primary role as a grant.
INSERT INTO user_roles (user_id, role, source)
SELECT id, role, 'backfill' FROM users
ON CONFLICT (user_id, role) DO NOTHING;

-- 3. Business profiles (one per BUSINESS account).
CREATE TABLE IF NOT EXISTS business_profiles (
  user_id UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
  company_name TEXT NOT NULL,
  biz_no TEXT,
  country TEXT NOT NULL DEFAULT 'KR',
  industry TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'PENDING'
    CHECK (status IN ('PENDING', 'APPROVED', 'SUSPENDED')),
  verified_by UUID REFERENCES users (id) ON DELETE SET NULL,
  verified_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_business_profiles_status ON business_profiles (status);
DROP TRIGGER IF EXISTS trg_business_profiles_updated_at ON business_profiles;
CREATE TRIGGER trg_business_profiles_updated_at
  BEFORE UPDATE ON business_profiles FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 4. Store ownership moves to the owning user.
ALTER TABLE store_businesses
  ADD COLUMN IF NOT EXISTS owner_user_id UUID REFERENCES users (id) ON DELETE CASCADE;
UPDATE store_businesses b SET owner_user_id = p.user_id
FROM partner_profiles p
WHERE p.id = b.partner_id AND b.owner_user_id IS NULL;
ALTER TABLE store_businesses ALTER COLUMN owner_user_id SET NOT NULL;
ALTER TABLE store_businesses ALTER COLUMN partner_id DROP NOT NULL;
CREATE INDEX IF NOT EXISTS idx_store_businesses_owner ON store_businesses (owner_user_id);

-- 5. G-ROLE migration: store-owning PARTNERs gain the BUSINESS grant so both
-- consoles stay reachable after the split (R-3 acceptance).
INSERT INTO user_roles (user_id, role, source)
SELECT DISTINCT p.user_id, 'BUSINESS', 'migration'
FROM store_businesses b
JOIN partner_profiles p ON p.id = b.partner_id
ON CONFLICT (user_id, role) DO NOTHING;
