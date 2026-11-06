-- Phase 1 seed: 1 admin, 1 partner (Suno Pro proof), 1 user;
-- AI provider policy matrix (7 providers); 4 legal_terms rows.
-- Password hashes are dummy bcrypt values for local dev only.

-- Users (fixed UUIDs for reproducible local dev)
INSERT INTO users (id, email, password_hash, role, display_name, is_active) VALUES
  ('a0000000-0000-4000-8000-000000000001', 'admin@example.com', '$2a$10$012345678901234567890uSeEdDuMmYhAsHfOrLoCaLdEvOnLy01234', 'ADMIN', '관리자', TRUE),
  ('a0000000-0000-4000-8000-000000000002', 'partner@example.com', '$2a$10$012345678901234567890uSeEdDuMmYhAsHfOrLoCaLdEvOnLy01234', 'PARTNER', '파트너뮤직', TRUE),
  ('a0000000-0000-4000-8000-000000000003', 'user@example.com', '$2a$10$012345678901234567890uSeEdDuMmYhAsHfOrLoCaLdEvOnLy01234', 'USER', '일반청취자', TRUE)
ON CONFLICT (email) DO NOTHING;

-- Partner profile
INSERT INTO partner_profiles (id, user_id, company_name, business_number, settlement_bank, settlement_account_encrypted, contact_email, status, approved_at) VALUES
  ('b0000000-0000-4000-8000-000000000001', 'a0000000-0000-4000-8000-000000000002', '파트너뮤직', '123-45-67890', '테스트은행', 'ENC(LOCAL-SEED-NOT-REAL)', 'partner@example.com', 'APPROVED', NOW())
ON CONFLICT (user_id) DO NOTHING;

-- AI provider policy matrix
INSERT INTO ai_provider_policies (provider, allows_standalone_streaming, min_bitrate_kbps, blocked_metadata_keywords, rejection_message_ko, commercial_terms_url) VALUES
  ('SUNO', TRUE, 192,
   ARRAY['cover', 'remix', 'unofficial', 'AI Drake', 'feat.'],
   'Suno 무료 플랜 음원은 단독 스트리밍이 불가합니다. Pro 이상 구독 증빙을 제출해 주세요.',
   'https://suno.com/terms'),
  ('UDIO', TRUE, 192,
   ARRAY['cover', 'remix', 'unofficial', 'AI Weeknd'],
   'Udio 무료 플랜 음원은 단독 스트리밍이 불가합니다. 유료 구독 증빙을 제출해 주세요.',
   'https://udio.com/terms'),
  ('AIVA', TRUE, 192,
   ARRAY['cover', 'remix', 'unofficial'],
   'AIVA 무료 플랜 음원은 단독 스트리밍이 불가합니다. Pro 구독 증빙을 제출해 주세요.',
   'https://aiva.ai/terms'),
  ('STABLE_AUDIO', TRUE, 192,
   ARRAY['cover', 'remix', 'unofficial'],
   'Stable Audio 무료 플랜 음원은 단독 스트리밍이 불가합니다. Pro 구독 증빙을 제출해 주세요.',
   'https://stability.ai/stable-audio-terms'),
  ('SOUNDRAW', FALSE, 192,
   ARRAY['cover', 'remix', 'unofficial', 'standalone'],
   'Soundraw 라이선스는 본 플랫폼의 단독 스트리밍 배포를 허용하지 않습니다. 다른 제공자를 이용하거나 권리 증빙을 제출해 주세요.',
   'https://soundraw.io/terms'),
  ('MUBERT', FALSE, 192,
   ARRAY['cover', 'remix', 'unofficial', 'standalone'],
   'Mubert 라이선스는 API 기반 이용에 한정되며 단독 스트리밍 배포가 불가합니다.',
   'https://mubert.com/terms'),
  ('CUSTOM_OPEN_MODEL', TRUE, 192,
   ARRAY['cover', 'remix', 'voice clone', '보이스클론', 'unofficial'],
   '오픈모델 음원은 학습데이터 적법성 및 보이스클로닝 아님에 대한 권리선언이 필요합니다.',
   'https://example.com/open-model-policy')
ON CONFLICT (provider) DO UPDATE SET
  allows_standalone_streaming = EXCLUDED.allows_standalone_streaming,
  min_bitrate_kbps = EXCLUDED.min_bitrate_kbps,
  blocked_metadata_keywords = EXCLUDED.blocked_metadata_keywords,
  rejection_message_ko = EXCLUDED.rejection_message_ko,
  commercial_terms_url = EXCLUDED.commercial_terms_url,
  updated_at = NOW();

-- Partner Suno Pro subscription proof (verified by admin seed)
INSERT INTO partner_ai_subscriptions (id, partner_id, provider, plan_name, proof_object_key, status, verified_by, verified_at, expires_at) VALUES
  ('c0000000-0000-4000-8000-000000000001', 'b0000000-0000-4000-8000-000000000001', 'SUNO', 'Pro', 'proofs/b0000000-0000-4000-8000-000000000001/suno-pro-2026-11.png', 'VERIFIED', 'a0000000-0000-4000-8000-000000000001', NOW(), NOW() + INTERVAL '30 days')
ON CONFLICT (partner_id, provider) DO UPDATE SET
  plan_name = EXCLUDED.plan_name,
  proof_object_key = EXCLUDED.proof_object_key,
  status = EXCLUDED.status,
  verified_by = EXCLUDED.verified_by,
  verified_at = NOW(),
  expires_at = EXCLUDED.expires_at,
  updated_at = NOW();

-- Legal terms (4 rows, v1.0)
INSERT INTO legal_terms (code, version, title_ko, body_md, is_required) VALUES
  ('TOS', 'v1.0', '이용약관', '# 이용약관 v1.0\n\n전문은 docs/legal/terms-of-service.md를 참조하세요.', TRUE),
  ('PARTNER_CONTRACT', 'v1.0', '파트너 계약', '# 파트너 계약 v1.0\n\n전문은 docs/legal/partner-contract.md를 참조하세요.', TRUE),
  ('PRIVACY', 'v1.0', '개인정보처리방침', '# 개인정보처리방침 v1.0\n\n전문은 docs/legal/privacy-policy.md를 참조하세요.', TRUE),
  ('COPYRIGHT_POLICY', 'v1.0', '저작권 정책', '# 저작권 정책 v1.0\n\n전문은 docs/legal/copyright-policy.md를 참조하세요.', TRUE)
ON CONFLICT (code, version) DO NOTHING;
