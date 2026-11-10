-- 000009: AI music generation jobs (WI-03 setup-only). Partners submit
-- prompts; the worker forwards them to an external generation backend
-- (GENERATION_BACKEND_URL, e.g. a home-PC ACE-Step server) and stages the
-- returned audio for the normal Finalize flow. No GPU runs in this repo.
CREATE TABLE IF NOT EXISTS generation_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  partner_id UUID NOT NULL REFERENCES partner_profiles(id) ON DELETE CASCADE,
  prompt TEXT NOT NULL,
  lyrics TEXT,
  model TEXT NOT NULL DEFAULT 'ace-step-v1',
  params_json TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'QUEUED'
    CHECK (status IN ('QUEUED','RUNNING','SUCCEEDED','FAILED')),
  output_object_key TEXT,
  duration_ms INTEGER CHECK (duration_ms IS NULL OR duration_ms > 0),
  error TEXT,
  attempts INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_generation_jobs_partner ON generation_jobs (partner_id);
CREATE INDEX IF NOT EXISTS idx_generation_jobs_status ON generation_jobs (status);
