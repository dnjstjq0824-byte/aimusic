-- 000027: Track G data model (WI-G0, AVIBE_자체생성AI_준비계획서 §3/§4/§7).
-- Model registry, generation job provenance fields, track origin, training
-- consents (feature OFF), append-only data provenance ledger + dataset
-- manifests, append-only credit ledger (unused until pricing is decided).
-- AVIBE joins the provider CHECKs; the AVIBE policy row lands via seed.sql
-- (ON CONFLICT-safe) so local and migrated stacks converge.
ALTER TABLE ai_provider_policies DROP CONSTRAINT IF EXISTS ai_provider_policies_provider_check;
ALTER TABLE ai_provider_policies ADD CONSTRAINT ai_provider_policies_provider_check
  CHECK (provider IN ('SUNO', 'UDIO', 'AIVA', 'STABLE_AUDIO', 'SOUNDRAW', 'MUBERT', 'ELEVENLABS', 'CUSTOM_OPEN_MODEL', 'AVIBE'));
ALTER TABLE partner_ai_subscriptions DROP CONSTRAINT IF EXISTS partner_ai_subscriptions_provider_check;
ALTER TABLE partner_ai_subscriptions ADD CONSTRAINT partner_ai_subscriptions_provider_check
  CHECK (provider IN ('SUNO', 'UDIO', 'AIVA', 'STABLE_AUDIO', 'SOUNDRAW', 'MUBERT', 'ELEVENLABS', 'CUSTOM_OPEN_MODEL', 'AVIBE'));
ALTER TABLE tracks DROP CONSTRAINT IF EXISTS tracks_ai_provider_check;
ALTER TABLE tracks ADD CONSTRAINT tracks_ai_provider_check
  CHECK (ai_provider IN ('SUNO', 'UDIO', 'AIVA', 'STABLE_AUDIO', 'SOUNDRAW', 'MUBERT', 'ELEVENLABS', 'CUSTOM_OPEN_MODEL', 'AVIBE'));

-- 1. Model registry: every servable weight set, pinned to a manifest hash.
CREATE TABLE IF NOT EXISTS generation_models (
  id TEXT PRIMARY KEY,
  version TEXT NOT NULL,
  weights_hash TEXT NOT NULL,
  license TEXT NOT NULL DEFAULT '',
  manifest_hash TEXT NOT NULL DEFAULT '',
  model_card_url TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'DRAFT'
    CHECK (status IN ('DRAFT', 'EVALUATING', 'APPROVED', 'RETIRED')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_generation_models_status ON generation_models (status);

-- 2. Job provenance: reproducibility (model_version/seed) + safety/watermark
-- evidence the AVIBE license-gate branch requires. user_id anticipates
-- non-partner creators; partner_id stays the current owner.
ALTER TABLE generation_jobs
  ADD COLUMN IF NOT EXISTS model_version TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS seed BIGINT CHECK (seed IS NULL OR seed >= 0),
  ADD COLUMN IF NOT EXISTS safety_result TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS watermark_id TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users (id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_generation_jobs_output_key ON generation_jobs (output_object_key)
  WHERE output_object_key IS NOT NULL;

-- 3. Track origin: external AI output vs AVIBE-model "Born Verified" output.
ALTER TABLE tracks
  ADD COLUMN IF NOT EXISTS origin TEXT NOT NULL DEFAULT 'EXTERNAL_AI'
    CHECK (origin IN ('EXTERNAL_AI', 'AVIBE_MODEL'));
CREATE INDEX IF NOT EXISTS idx_tracks_origin ON tracks (origin);

-- 4. Creator training opt-in (separate consent; default off; unenforced).
CREATE TABLE IF NOT EXISTS training_consents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  partner_id UUID NOT NULL REFERENCES partner_profiles (id) ON DELETE CASCADE,
  scope TEXT NOT NULL DEFAULT 'CATALOG'
    CHECK (scope IN ('CATALOG', 'TRACK')),
  track_id UUID REFERENCES tracks (id) ON DELETE CASCADE,
  consented BOOLEAN NOT NULL DEFAULT FALSE,
  consented_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ,
  revenue_share_bps INTEGER CHECK (revenue_share_bps IS NULL OR (revenue_share_bps >= 0 AND revenue_share_bps <= 10000)),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT training_consents_track_scope CHECK (
    (scope = 'CATALOG' AND track_id IS NULL) OR (scope = 'TRACK' AND track_id IS NOT NULL)
  )
);
CREATE INDEX IF NOT EXISTS idx_training_consents_partner ON training_consents (partner_id);

-- 5. Data provenance ledger (append-only) + dataset manifests.
CREATE TABLE IF NOT EXISTS data_provenance_ledger (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  file_hash TEXT NOT NULL,
  fingerprint TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL,
  rights_holder TEXT NOT NULL DEFAULT '',
  license_doc_id TEXT NOT NULL DEFAULT '',
  consent_id UUID REFERENCES training_consents (id) ON DELETE SET NULL,
  allowed_uses TEXT[] NOT NULL DEFAULT '{}',
  region TEXT NOT NULL DEFAULT '',
  valid_from TIMESTAMPTZ,
  valid_until TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ,
  supersedes UUID REFERENCES data_provenance_ledger (id) ON DELETE SET NULL,
  entry_hash TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_provenance_file_hash ON data_provenance_ledger (file_hash);
CREATE INDEX IF NOT EXISTS idx_provenance_source ON data_provenance_ledger (source);
CREATE OR REPLACE FUNCTION forbid_provenance_mutation() RETURNS TRIGGER AS $$
BEGIN
  RAISE EXCEPTION '% is append-only', TG_TABLE_NAME;
  RETURN NULL;
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS trg_provenance_no_update ON data_provenance_ledger;
CREATE TRIGGER trg_provenance_no_update
  BEFORE UPDATE OR DELETE ON data_provenance_ledger FOR EACH ROW EXECUTE FUNCTION forbid_provenance_mutation();

CREATE TABLE IF NOT EXISTS dataset_manifests (
  manifest_hash TEXT PRIMARY KEY,
  entry_ids UUID[] NOT NULL DEFAULT '{}',
  entry_count INTEGER NOT NULL DEFAULT 0 CHECK (entry_count >= 0),
  notes TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
DROP TRIGGER IF EXISTS trg_manifests_no_update ON dataset_manifests;
CREATE TRIGGER trg_manifests_no_update
  BEFORE UPDATE OR DELETE ON dataset_manifests FOR EACH ROW EXECUTE FUNCTION forbid_provenance_mutation();

-- 6. Credit ledger (append-only, unused: amounts post to paid plans only
-- after PO prices them; failed generations refund automatically).
CREATE TABLE IF NOT EXISTS credit_ledger (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  delta_credits INTEGER NOT NULL CHECK (delta_credits <> 0),
  reason TEXT NOT NULL
    CHECK (reason IN ('PLAN_GRANT', 'GENERATION_HOLD', 'GENERATION_REFUND', 'GENERATION_SPEND', 'ADMIN_ADJUST')),
  job_id UUID REFERENCES generation_jobs (id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_credit_ledger_user ON credit_ledger (user_id, created_at DESC);
DROP TRIGGER IF EXISTS trg_credit_no_update ON credit_ledger;
CREATE TRIGGER trg_credit_no_update
  BEFORE UPDATE OR DELETE ON credit_ledger FOR EACH ROW EXECUTE FUNCTION forbid_provenance_mutation();
