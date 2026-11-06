-- Phase 1 seed: 1 admin, 1 partner (Suno Pro proof), 1 user;
-- AI provider policy matrix (7 providers with plan-tier rules); 4 legal_terms rows.
-- Idempotent schema sync for existing volumes:
ALTER TABLE ai_provider_policies ADD COLUMN IF NOT EXISTS allowed_plan_tiers TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE ai_provider_policies ADD COLUMN IF NOT EXISTS blocked_plan_tiers TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE ai_provider_policies ADD COLUMN IF NOT EXISTS sync_only_plan_tiers TEXT[] NOT NULL DEFAULT '{}';

ALTER TABLE tracks ADD COLUMN IF NOT EXISTS genre TEXT;
ALTER TABLE tracks ADD COLUMN IF NOT EXISTS mood_tags TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE tracks ADD COLUMN IF NOT EXISTS ai_generation_type TEXT NOT NULL DEFAULT 'FULL_AI';
ALTER TABLE tracks ADD COLUMN IF NOT EXISTS ai_model_version TEXT;
ALTER TABLE tracks ADD COLUMN IF NOT EXISTS ai_prompt TEXT;
ALTER TABLE tracks ADD COLUMN IF NOT EXISTS is_prompt_public BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE tracks ADD COLUMN IF NOT EXISTS is_instrumental BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE tracks ADD COLUMN IF NOT EXISTS lyrics TEXT;
ALTER TABLE tracks ADD COLUMN IF NOT EXISTS audio_fingerprint_hash TEXT;
ALTER TABLE tracks ADD COLUMN IF NOT EXISTS waveform_json TEXT;

ALTER TABLE track_rights_declarations ADD COLUMN IF NOT EXISTS confirmed_no_unauthorized_sampling BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE track_rights_declarations ADD COLUMN IF NOT EXISTS confirmed_indemnification BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE track_rights_declarations ADD COLUMN IF NOT EXISTS plan_tier_used TEXT;
ALTER TABLE track_rights_declarations ADD COLUMN IF NOT EXISTS provider_track_url TEXT;
ALTER TABLE track_rights_declarations ADD COLUMN IF NOT EXISTS human_contribution_types TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE track_rights_declarations ADD COLUMN IF NOT EXISTS human_contribution_desc TEXT;
ALTER TABLE track_rights_declarations ADD COLUMN IF NOT EXISTS gate1_policy_passed BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE track_rights_declarations ADD COLUMN IF NOT EXISTS gate2_date_passed BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE track_rights_declarations ADD COLUMN IF NOT EXISTS gate3_ffprobe_passed BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE track_rights_declarations ADD COLUMN IF NOT EXISTS ffprobe_report_json TEXT;
ALTER TABLE track_rights_declarations ADD COLUMN IF NOT EXISTS signed_ip TEXT;

ALTER TABLE playlists ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE playlists ADD COLUMN IF NOT EXISTS is_official_curation BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS user_artist_blocks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  blocker_user_id UUID NOT NULL,
  blocked_partner_id UUID NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT user_artist_blocks_unique UNIQUE (blocker_user_id, blocked_partner_id),
  CONSTRAINT fk_uab_blocker FOREIGN KEY (blocker_user_id) REFERENCES users (id) ON DELETE CASCADE,
  CONSTRAINT fk_uab_partner FOREIGN KEY (blocked_partner_id) REFERENCES partner_profiles (id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_uab_blocker ON user_artist_blocks (blocker_user_id);

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

-- AI provider policy matrix (Section 3.1 Commercial Gatekeeper)
INSERT INTO ai_provider_policies (
  provider, allows_standalone_streaming, allowed_plan_tiers, blocked_plan_tiers, sync_only_plan_tiers,
  min_bitrate_kbps, blocked_metadata_keywords, rejection_message_ko, commercial_terms_url
) VALUES
  ('SUNO', TRUE,
   ARRAY['Pro', 'Premier'],
   ARRAY['Free', 'Basic', 'Free (Basic)', 'Trial'],
   ARRAY[]::TEXT[],
   192,
   ARRAY['free', 'non-commercial', 'trial', 'demo', 'evaluation', 'personal use only', 'watermarked', 'free-trial', 'cover', 'remix', 'unofficial', 'AI Drake', 'feat.'],
   'Suno 무료 플랜(Free/Basic) 및 소급 생성 음원은 단독 스트리밍이 불가합니다. 유료 구독 기간 내 생성된 Pro/Premier 음원만 허용됩니다.',
   'https://suno.com/pricing'),
  ('UDIO', TRUE,
   ARRAY['Standard', 'Pro'],
   ARRAY['Free', 'Basic', 'Trial'],
   ARRAY[]::TEXT[],
   192,
   ARRAY['free', 'non-commercial', 'trial', 'demo', 'evaluation', 'personal use only', 'watermarked', 'free-trial', 'cover', 'remix', 'unofficial', 'AI Weeknd'],
   'Udio 무료 플랜 및 외부 유통 제한 기간 생성 음원은 단독 스트리밍이 불가합니다. 상업 배포 허용 유료 플랜 증빙을 제출해 주세요.',
   'https://udio.com/terms'),
  ('AIVA', TRUE,
   ARRAY['Pro'],
   ARRAY['Free', 'Trial'],
   ARRAY['Standard'],
   192,
   ARRAY['free', 'non-commercial', 'trial', 'demo', 'evaluation', 'personal use only', 'watermarked', 'free-trial', 'cover', 'remix', 'unofficial'],
   'AIVA Free(비상업) 및 Standard(유튜브·SNS 채널 한정 수익화, 마스터 권리 AIVA 귀속) 플랜은 단독 음원 스트리밍 유통이 불가합니다. AIVA Pro만 허용됩니다.',
   'https://www.aiva.ai/legal/1'),
  ('STABLE_AUDIO', TRUE,
   ARRAY['Pro', 'Creator', 'Enterprise'],
   ARRAY['Free', 'Trial'],
   ARRAY[]::TEXT[],
   192,
   ARRAY['free', 'non-commercial', 'trial', 'demo', 'evaluation', 'personal use only', 'watermarked', 'free-trial', 'cover', 'remix', 'unofficial'],
   'Stable Audio 무료 플랜 음원은 단독 스트리밍이 불가합니다. Pro/Creator/Enterprise 구독 기간 내 생성된 음원만 허용됩니다.',
   'https://stability.ai/stable-audio-terms'),
  ('SOUNDRAW', FALSE,
   ARRAY['Artist Distribute Plan', 'Enterprise'],
   ARRAY['Free', 'Trial'],
   ARRAY['Creator', 'Pro', 'Pro (Video BGM 전용)'],
   192,
   ARRAY['free', 'non-commercial', 'trial', 'demo', 'evaluation', 'personal use only', 'watermarked', 'free-trial', 'cover', 'remix', 'unofficial', 'standalone'],
   'Soundraw 기본 요금제(Free/Creator/Video BGM 전용)는 영상·게임 배경음악(Sync) 용도만 허용되며 단독 음원 스트리밍 배포가 금지됩니다.',
   'https://soundraw.io/terms'),
  ('MUBERT', FALSE,
   ARRAY['Artist Distribute Plan', 'Enterprise'],
   ARRAY['Free', 'Trial'],
   ARRAY['Creator', 'Pro', 'Pro (Video BGM 전용)'],
   192,
   ARRAY['free', 'non-commercial', 'trial', 'demo', 'evaluation', 'personal use only', 'watermarked', 'free-trial', 'cover', 'remix', 'unofficial', 'standalone'],
   'Mubert 기본 요금제(Free/Creator)는 영상·앱 배경음악(Sync) 용도에 한정되며 단독 음원 스트리밍 등록이 금지됩니다.',
   'https://mubert.com/terms'),
  ('CUSTOM_OPEN_MODEL', TRUE,
   ARRAY['Apache-2.0', 'MIT', 'Commercial'],
   ARRAY['CC-BY-NC', 'CC-BY-NC-4.0', 'CC-BY-NC-SA', 'Non-Commercial', 'Free'],
   ARRAY[]::TEXT[],
   192,
   ARRAY['free', 'non-commercial', 'trial', 'demo', 'evaluation', 'personal use only', 'watermarked', 'free-trial', 'cc-by-nc', 'cover', 'remix', 'voice clone', '보이스클론', 'unofficial'],
   '비상업 라이선스 가중치 모델(CC-BY-NC 계열) 또는 보이스클로닝 포함 음원은 상업적 스트리밍 유통이 불가합니다.',
   'https://example.com/open-model-policy')
ON CONFLICT (provider) DO UPDATE SET
  allows_standalone_streaming = EXCLUDED.allows_standalone_streaming,
  allowed_plan_tiers = EXCLUDED.allowed_plan_tiers,
  blocked_plan_tiers = EXCLUDED.blocked_plan_tiers,
  sync_only_plan_tiers = EXCLUDED.sync_only_plan_tiers,
  min_bitrate_kbps = EXCLUDED.min_bitrate_kbps,
  blocked_metadata_keywords = EXCLUDED.blocked_metadata_keywords,
  rejection_message_ko = EXCLUDED.rejection_message_ko,
  commercial_terms_url = EXCLUDED.commercial_terms_url,
  updated_at = NOW();

-- Partner Suno Pro subscription proof (verified by admin seed)
INSERT INTO partner_ai_subscriptions (id, partner_id, provider, plan_name, proof_object_key, status, verified_by, verified_at, expires_at) VALUES
  ('c0000000-0000-4000-8000-000000000001', 'b0000000-0000-4000-8000-000000000001', 'SUNO', 'Pro', 'proofs/b0000000-0000-4000-8000-000000000001/suno-pro-2026-11.png', 'VERIFIED', 'a0000000-0000-4000-8000-000000000001', NOW() - INTERVAL '7 days', NOW() + INTERVAL '30 days')
ON CONFLICT (partner_id, provider) DO UPDATE SET
  plan_name = EXCLUDED.plan_name,
  proof_object_key = EXCLUDED.proof_object_key,
  status = EXCLUDED.status,
  verified_by = EXCLUDED.verified_by,
  verified_at = EXCLUDED.verified_at,
  expires_at = EXCLUDED.expires_at,
  updated_at = NOW();

-- Legal terms (4 mandatory policies, v1.0)
INSERT INTO legal_terms (code, version, title_ko, body_md, is_required) VALUES
  ('TOS', 'v1.0', '서비스 이용약관', '# 서비스 이용약관 (Terms of Service) v1.0\n\n본 약관은 인공지능기본법 제31조(AI 생성물 투명성 표시 의무) 및 저작권법 제102조~제104조(OSP 책임 제한 및 임시조치), 제133조의2(삼진아웃 계정 정지)를 준수하는 AI 음원 스트리밍 서비스 이용 조건을 규정합니다.\n\n1. 플랫폼에 유통되는 모든 AI 생성 음원에는 `[AI 생성 음원]` 또는 `[AI+인간 공동창작]` 배지와 오디오 스트림 워터마크가 부착됩니다.\n2. 비상업 플랜 음원, 소급 생성 음원, 무단 보이스클로닝 음원의 업로드는 엄격히 금지됩니다.\n3. 전문은 `docs/legal/terms-of-service.md`를 참조하세요.', TRUE),
  ('PARTNER_CONTRACT', 'v1.0', '파트너 음원유통 및 면책 계약서', '# 파트너 음원유통 및 면책 계약서 v1.0\n\n1. 파트너는 단독 음원 스트리밍 유통 및 마스터 권리가 부여된 유료 상업용 플랜(Suno Pro/Premier, AIVA Pro, Stable Audio Pro 등) 유효 구독 기간 중에 생성된 음원만 공급해야 합니다.\n2. 무료 플랜 생성 후 사후 구독한 소급 불가 음원 및 영상 BGM 전용(Sync-only) 라이선스 음원은 업로드를 금지합니다.\n3. 허위 권리 보증 또는 저작권/보이스클로닝 위반 적발 시 스트라이크 부과 및 정산 수익 전액 몰수(`FORFEITED_VIOLATION`)에 동의합니다.\n4. 개인 파트너 정산 시 소득세법에 따라 3.3% 원천징수 후 지급되며, 계좌정보는 `AES-256-GCM`으로 암호화 보관됩니다.\n5. 전문은 `docs/legal/partner-contract.md`를 참조하세요.', TRUE),
  ('PRIVACY', 'v1.0', '개인정보처리방침', '# 개인정보처리방침 (Privacy Policy) v1.0\n\n1. 개인정보보호법 제24조/제29조에 따라 비밀번호는 bcrypt 해시로, 정산 계좌번호 및 식별번호는 `AES-256-GCM`으로 암호화하여 저장합니다.\n2. 약관 동의 시 분쟁 대비 감사 로그(IP, User-Agent, 동의 시각)를 기록하며, 회원 탈퇴 요청 시 세무/법정 보존 의무 항목을 제외한 개인식별 정보를 즉시 파기합니다.\n3. 전문은 `docs/legal/privacy-policy.md`를 참조하세요.', TRUE),
  ('COPYRIGHT_POLICY', 'v1.0', '저작권 및 AI 보이스클로닝 보호정책', '# 저작권 및 AI 보이스클로닝 보호정책 v1.0\n\n1. 저작권법 제103조에 따라 권리침해 또는 비상업 음원 부정 유통 신고 접수 시 즉시 임시조치(`TAKEDOWN_TEMP` / `TAKEN_DOWN`)를 시행합니다.\n2. 파트너는 소명 자료(Counter-Notice)를 제출할 수 있으며, 위반 확정 3회 누적(`strike_count >= 3`) 시 계정이 자동 영구 정지됩니다.\n3. 실존 가수·성우·타인의 음성을 동의 없이 합성한 보이스클로닝(퍼블리시티권 침해) 음원은 즉시 차단 및 영구 삭제됩니다.\n4. 전문은 `docs/legal/copyright-policy.md`를 참조하세요.', TRUE)
ON CONFLICT (code, version) DO UPDATE SET
  title_ko = EXCLUDED.title_ko,
  body_md = EXCLUDED.body_md,
  is_required = EXCLUDED.is_required;
