-- G-ROLE dry-run: what 000026_business_roles WOULD migrate (read-only).
-- Run against the target DB BEFORE applying the migration, attach the
-- output to the progress log, and only then apply. Production execution is
-- forbidden in this package (continuous mode keeps G-ROLE for ops).
--
-- Usage (local stack):
--   docker compose -f docker-compose.local.yml exec -T postgres \
--     psql -U avibe -d avibe -f scripts/role-migration-dryrun.sql

-- 1. Accounts per primary role.
SELECT role, COUNT(*) AS accounts FROM users GROUP BY role ORDER BY role;

-- 2. Store-owning PARTNERs that would gain the BUSINESS grant.
SELECT u.email, u.display_name, COUNT(b.id) AS stores
FROM users u
JOIN partner_profiles p ON p.user_id = u.id
JOIN store_businesses b ON b.partner_id = p.id
WHERE u.role = 'PARTNER'
GROUP BY u.email, u.display_name
ORDER BY u.email;

-- 3. Count for the migration INSERT (step 5 of 000026).
SELECT COUNT(DISTINCT p.user_id) AS partners_gaining_business
FROM store_businesses b
JOIN partner_profiles p ON p.id = b.partner_id;

-- 4. Stores whose owner backfill (step 4) would FAIL (orphan partner link).
SELECT b.id, b.partner_id
FROM store_businesses b
LEFT JOIN partner_profiles p ON p.id = b.partner_id
WHERE p.id IS NULL;

-- 5. Business-number duplicates that would collide (informational only;
-- the migration does not touch business_number uniqueness).
SELECT business_number, COUNT(*) AS n
FROM store_businesses
GROUP BY business_number HAVING COUNT(*) > 1;
