-- A2 email verification + user locale: users.locale (CHECK 5 locales,
-- default ko), users.email_verified_at (existing accounts count as
-- verified), email_verifications (token hash, 24h TTL, single-use).
-- purpose VERIFY covers registered accounts; OAUTH covers pending social
-- signups that have no user row yet (A1 complete-email flow).
ALTER TABLE users ADD COLUMN locale TEXT NOT NULL DEFAULT 'ko';
ALTER TABLE users ADD CONSTRAINT users_locale_check
  CHECK (locale IN ('ko', 'en', 'ja', 'zh-cn', 'th'));
ALTER TABLE users ADD COLUMN email_verified_at TIMESTAMPTZ;
-- Accounts created before verification existed count as verified.
UPDATE users SET email_verified_at = NOW() WHERE email_verified_at IS NULL;

CREATE TABLE IF NOT EXISTS email_verifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID,
  email TEXT NOT NULL,
  purpose TEXT NOT NULL,
  token_hash TEXT NOT NULL,
  ref_token TEXT,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT email_verifications_user_fk FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  CONSTRAINT email_verifications_purpose_check CHECK (purpose IN ('VERIFY', 'OAUTH')),
  CONSTRAINT email_verifications_hash_unique UNIQUE (token_hash)
);
CREATE INDEX IF NOT EXISTS idx_email_verifications_user ON email_verifications (user_id);
CREATE INDEX IF NOT EXISTS idx_email_verifications_email ON email_verifications (email);
