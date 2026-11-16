-- A2 rollback: drop verification rows/table and the users columns.
DROP TABLE IF EXISTS email_verifications;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified_at;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_locale_check;
ALTER TABLE users DROP COLUMN IF EXISTS locale;
