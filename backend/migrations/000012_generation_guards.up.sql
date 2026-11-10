-- 000012: generation license record + prompt-violation strikes (WI-08/E-3/E-4).
-- license_ref is the ops-declared model license basis per job; violations
-- drive the warn -> 24h generation suspension ladder (account termination
-- stays a manual admin suspension).
ALTER TABLE generation_jobs ADD COLUMN IF NOT EXISTS license_ref TEXT NOT NULL DEFAULT '';
CREATE TABLE IF NOT EXISTS generation_violations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  partner_id UUID NOT NULL REFERENCES partner_profiles(id) ON DELETE CASCADE,
  matched_keyword TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_generation_violations_partner ON generation_violations (partner_id, created_at DESC);
