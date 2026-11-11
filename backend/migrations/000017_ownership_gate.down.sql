-- 000017 down: restore pre-ownership-gate policy matrix.
ALTER TABLE tracks DROP COLUMN IF EXISTS ai_credit_parts;

DELETE FROM ai_provider_policies WHERE provider = 'ELEVENLABS';

UPDATE ai_provider_policies
SET allowed_plan_tiers = ARRAY['Pro', 'Creator', 'Enterprise'], updated_at = NOW()
WHERE provider = 'STABLE_AUDIO';

UPDATE ai_provider_policies
SET allowed_plan_tiers = ARRAY['Artist Distribute Plan', 'Enterprise'],
  rejection_message_ko = 'Soundraw 기본 요금제(Free/Creator/Video BGM 전용)는 영상·게임 배경음악(Sync) 용도만 허용되며 단독 음원 스트리밍 배포가 금지됩니다.',
  updated_at = NOW()
WHERE provider = 'SOUNDRAW';

UPDATE ai_provider_policies
SET allowed_plan_tiers = ARRAY['Artist Distribute Plan', 'Enterprise'],
  rejection_message_ko = 'Mubert 기본 요금제(Free/Creator)는 영상·앱 배경음악(Sync) 용도에 한정되며 단독 음원 스트리밍 등록이 금지됩니다.',
  updated_at = NOW()
WHERE provider = 'MUBERT';

ALTER TABLE ai_provider_policies DROP CONSTRAINT IF EXISTS ai_provider_policies_provider_check;
ALTER TABLE ai_provider_policies ADD CONSTRAINT ai_provider_policies_provider_check
  CHECK (provider IN ('SUNO', 'UDIO', 'AIVA', 'STABLE_AUDIO', 'SOUNDRAW', 'MUBERT', 'CUSTOM_OPEN_MODEL'));
ALTER TABLE partner_ai_subscriptions DROP CONSTRAINT IF EXISTS partner_ai_subscriptions_provider_check;
ALTER TABLE partner_ai_subscriptions ADD CONSTRAINT partner_ai_subscriptions_provider_check
  CHECK (provider IN ('SUNO', 'UDIO', 'AIVA', 'STABLE_AUDIO', 'SOUNDRAW', 'MUBERT', 'CUSTOM_OPEN_MODEL'));
ALTER TABLE tracks DROP CONSTRAINT IF EXISTS tracks_ai_provider_check;
ALTER TABLE tracks ADD CONSTRAINT tracks_ai_provider_check
  CHECK (ai_provider IN ('SUNO', 'UDIO', 'AIVA', 'STABLE_AUDIO', 'SOUNDRAW', 'MUBERT', 'CUSTOM_OPEN_MODEL'));
