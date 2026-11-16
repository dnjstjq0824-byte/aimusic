-- WI-G0 rollback. Fails closed when AVIBE-gated data exists: the provider
-- CHECK restore errors while any AVIBE subscription or track row exists —
-- clear those rows first, then retry. The AVIBE policy row is seed-owned and
-- removed here (start-local re-seeds it on the next up-apply).
DROP TRIGGER IF EXISTS trg_credit_no_update ON credit_ledger;
DROP TABLE IF EXISTS credit_ledger;
DROP TRIGGER IF EXISTS trg_manifests_no_update ON dataset_manifests;
DROP TABLE IF EXISTS dataset_manifests;
DROP TRIGGER IF EXISTS trg_provenance_no_update ON data_provenance_ledger;
DROP TABLE IF EXISTS data_provenance_ledger;
DROP FUNCTION IF EXISTS forbid_provenance_mutation();
DROP TABLE IF EXISTS training_consents;
ALTER TABLE tracks DROP COLUMN IF EXISTS origin;
DROP INDEX IF EXISTS idx_generation_jobs_output_key;
ALTER TABLE generation_jobs
  DROP COLUMN IF EXISTS user_id,
  DROP COLUMN IF EXISTS watermark_id,
  DROP COLUMN IF EXISTS safety_result,
  DROP COLUMN IF EXISTS seed,
  DROP COLUMN IF EXISTS model_version;
DROP TABLE IF EXISTS generation_models;
DELETE FROM ai_provider_policies WHERE provider = 'AVIBE';
ALTER TABLE ai_provider_policies DROP CONSTRAINT IF EXISTS ai_provider_policies_provider_check;
ALTER TABLE ai_provider_policies ADD CONSTRAINT ai_provider_policies_provider_check
  CHECK (provider IN ('SUNO', 'UDIO', 'AIVA', 'STABLE_AUDIO', 'SOUNDRAW', 'MUBERT', 'ELEVENLABS', 'CUSTOM_OPEN_MODEL'));
ALTER TABLE partner_ai_subscriptions DROP CONSTRAINT IF EXISTS partner_ai_subscriptions_provider_check;
ALTER TABLE partner_ai_subscriptions ADD CONSTRAINT partner_ai_subscriptions_provider_check
  CHECK (provider IN ('SUNO', 'UDIO', 'AIVA', 'STABLE_AUDIO', 'SOUNDRAW', 'MUBERT', 'ELEVENLABS', 'CUSTOM_OPEN_MODEL'));
ALTER TABLE tracks DROP CONSTRAINT IF EXISTS tracks_ai_provider_check;
ALTER TABLE tracks ADD CONSTRAINT tracks_ai_provider_check
  CHECK (ai_provider IN ('SUNO', 'UDIO', 'AIVA', 'STABLE_AUDIO', 'SOUNDRAW', 'MUBERT', 'ELEVENLABS', 'CUSTOM_OPEN_MODEL'));
