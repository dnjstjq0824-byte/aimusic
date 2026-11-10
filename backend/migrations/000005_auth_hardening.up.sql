-- Batch 5 schema: session revocation + TOTP two-factor.
-- All additive and idempotent; safe to run on existing volumes.

-- token_version is minted into JWTs; password change / withdraw / logout-all
-- bumps it and instantly invalidates every outstanding token.
ALTER TABLE users ADD COLUMN IF NOT EXISTS token_version INT NOT NULL DEFAULT 0;

-- TOTP second factor (RFC 6238, verified with stdlib HMAC-SHA1).
ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_secret TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_enrolled_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_recovery_hashes TEXT[] NOT NULL DEFAULT '{}';
