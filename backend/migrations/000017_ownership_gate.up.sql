-- 000017: ownership-gate policy matrix (PO 2026-11-11).
-- ELEVENLABS allow; STABLE_AUDIO tier refresh; SOUNDRAW/MUBERT full block
-- (no ownership transfer); terms recheck stamp; track AI-credit parts column.
INSERT INTO ai_provider_policies (
  provider, allows_standalone_streaming, allowed_plan_tiers, blocked_plan_tiers, sync_only_plan_tiers,
  min_bitrate_kbps, blocked_metadata_keywords, rejection_message_ko, commercial_terms_url, terms_checked_at
) VALUES (
  'ELEVENLABS', TRUE,
  ARRAY['Creator', 'Pro', 'Scale', 'Business', 'Enterprise'],
  ARRAY['Free', 'Starter', 'Trial'],
  ARRAY[]::TEXT[],
  192,
  ARRAY['free', 'non-commercial', 'trial', 'demo', 'evaluation', 'personal use only', 'watermarked', 'free-trial', 'cover', 'remix', 'unofficial', 'ai cover', 'aicover', 'voice clone', '보이스클론'],
  'ElevenLabs Music Free·Starter 플랜은 단독 스트리밍 배포가 불가합니다. Creator 이상 유료 플랜 구독 기간 내 생성·다운로드한 음원만 허용됩니다.',
  'https://elevenlabs.io/music-terms',
  NOW()
) ON CONFLICT (provider) DO UPDATE SET
  allows_standalone_streaming = EXCLUDED.allows_standalone_streaming,
  allowed_plan_tiers = EXCLUDED.allowed_plan_tiers,
  blocked_plan_tiers = EXCLUDED.blocked_plan_tiers,
  sync_only_plan_tiers = EXCLUDED.sync_only_plan_tiers,
  min_bitrate_kbps = EXCLUDED.min_bitrate_kbps,
  blocked_metadata_keywords = EXCLUDED.blocked_metadata_keywords,
  rejection_message_ko = EXCLUDED.rejection_message_ko,
  commercial_terms_url = EXCLUDED.commercial_terms_url,
  terms_checked_at = NOW(),
  updated_at = NOW();

UPDATE ai_provider_policies
SET allowed_plan_tiers = ARRAY['Solo', 'Session', 'Producer', 'Studio', 'Enterprise', 'Pro'],
  terms_checked_at = NOW(), updated_at = NOW()
WHERE provider = 'STABLE_AUDIO';

UPDATE ai_provider_policies
SET allowed_plan_tiers = ARRAY[]::TEXT[],
  rejection_message_ko = '작곡 저작권이 Soundraw에 귀속되어 생성자 소유권 요건을 충족하지 않습니다. SOUNDRAW 생성 음원은 유통할 수 없습니다.',
  terms_checked_at = NOW(), updated_at = NOW()
WHERE provider = 'SOUNDRAW';

UPDATE ai_provider_policies
SET allowed_plan_tiers = ARRAY[]::TEXT[],
  rejection_message_ko = 'MUBERT 생성 음원은 단독 스트리밍 배포가 금지된 라이선스로 생성자 소유권 요건을 충족하지 않습니다.',
  terms_checked_at = NOW(), updated_at = NOW()
WHERE provider = 'MUBERT';

UPDATE ai_provider_policies SET terms_checked_at = NOW(), updated_at = NOW() WHERE terms_checked_at IS NULL;

ALTER TABLE tracks ADD COLUMN IF NOT EXISTS ai_credit_parts TEXT[] NOT NULL DEFAULT '{}';
