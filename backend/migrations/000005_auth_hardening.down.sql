-- Batch 5 rollback.
ALTER TABLE users DROP COLUMN IF EXISTS totp_recovery_hashes;
ALTER TABLE users DROP COLUMN IF EXISTS totp_enrolled_at;
ALTER TABLE users DROP COLUMN IF EXISTS totp_enabled;
ALTER TABLE users DROP COLUMN IF EXISTS totp_secret;
ALTER TABLE users DROP COLUMN IF EXISTS token_version;
