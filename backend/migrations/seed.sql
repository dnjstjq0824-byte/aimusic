-- Phase 1 seed: 1 admin, 1 partner (Suno Pro proof), 1 user;
-- AI provider policy matrix (7 providers with plan-tier rules); 4 legal_terms rows.
--
-- LOCAL DEV ONLY. This file creates fixed demo accounts and must never run in
-- staging/prod. Guard: refuse unless the session sets app.allow_seed=true
-- (scripts/start-local.sh exports PGOPTIONS accordingly).
DO $$
BEGIN
  IF current_setting('app.allow_seed', true) IS DISTINCT FROM 'true' THEN
    RAISE EXCEPTION 'seed.sql refused: set app.allow_seed=true (local dev only)';
  END IF;
END
$$;
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

-- Batch 3 idempotent sync for volumes seeded before 000003 (mirrors 000003 up).
ALTER TABLE ai_provider_policies ADD COLUMN IF NOT EXISTS max_generated_at TIMESTAMPTZ;
ALTER TABLE ai_provider_policies ADD COLUMN IF NOT EXISTS terms_snapshot_url TEXT;
ALTER TABLE ai_provider_policies ADD COLUMN IF NOT EXISTS terms_checked_at TIMESTAMPTZ;
ALTER TABLE track_rights_declarations ADD COLUMN IF NOT EXISTS collective_trust_status TEXT NOT NULL DEFAULT 'NONE';
ALTER TABLE track_rights_declarations ADD COLUMN IF NOT EXISTS collective_name TEXT;
ALTER TABLE copyright_disputes ADD COLUMN IF NOT EXISTS takedown_requested_at TIMESTAMPTZ;
ALTER TABLE copyright_disputes ADD COLUMN IF NOT EXISTS partner_notified_at TIMESTAMPTZ;
ALTER TABLE copyright_disputes ADD COLUMN IF NOT EXISTS resume_requested_at TIMESTAMPTZ;
ALTER TABLE copyright_disputes ADD COLUMN IF NOT EXISTS resume_scheduled_at TIMESTAMPTZ;
ALTER TABLE copyright_disputes ADD COLUMN IF NOT EXISTS resumed_at TIMESTAMPTZ;
ALTER TABLE billing_orders ADD COLUMN IF NOT EXISTS refund_consent_at TIMESTAMPTZ;
ALTER TABLE billing_orders ADD COLUMN IF NOT EXISTS refund_terms_version TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS birth_year INT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS age_confirmed_at TIMESTAMPTZ;
ALTER TABLE partner_settlements ADD COLUMN IF NOT EXISTS income_type TEXT NOT NULL DEFAULT 'BUSINESS_33';

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
   ARRAY['free', 'non-commercial', 'trial', 'demo', 'evaluation', 'personal use only', 'watermarked', 'free-trial', 'cover', 'remix', 'unofficial', 'AI Drake', 'feat.', 'ai cover', 'aicover', 'voice clone', '보이스클론'],
   'Suno 무료 플랜(Free/Basic) 및 소급 생성 음원은 단독 스트리밍이 불가합니다. 유료 구독 기간 내 생성된 Pro/Premier 음원만 허용됩니다.',
   'https://suno.com/pricing'),
  ('UDIO', TRUE,
   ARRAY['Standard', 'Pro'],
   ARRAY['Free', 'Basic', 'Trial'],
   ARRAY[]::TEXT[],
   192,
   ARRAY['free', 'non-commercial', 'trial', 'demo', 'evaluation', 'personal use only', 'watermarked', 'free-trial', 'cover', 'remix', 'unofficial', 'AI Weeknd', 'ai cover', 'aicover', 'voice clone', '보이스클론'],
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

-- B2: Udio generation cutoff 2025-10-30 (UMG deal download halt).
UPDATE ai_provider_policies
SET max_generated_at = '2025-10-30T00:00:00Z'::timestamptz
WHERE provider = 'UDIO' AND max_generated_at IS NULL;

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

-- Legal terms v1.1 (launch-compliance revision: §102/§103 OSP procedure,
-- "no rights warranty" notice, withdrawal/refund terms; see docs/legal/*.md).
INSERT INTO legal_terms (code, version, title_ko, body_md, is_required) VALUES
  ('TOS', 'v1.1', '서비스 이용약관', '# 서비스 이용약관 (Terms of Service) v1.1\n\n1. 본 플랫폼은 저작권법상 온라인서비스제공자(OSP)이며, 권리자의 복제·전송 중단 요구(저작권법 제103조)가 접수되면 지체 없이 해당 음원의 복제·전송을 중단하고 파트너에게 통보합니다.\n2. 플랫폼의 상업 이용 조건 확인 절차(라이선스 게이트)는 파트너 신고와 자동 검사를 거친 과정이며, 개별 음원의 권리 관계를 보증하지 않습니다.\n3. AI 생성 표시는 인공지능기본법 제31조의 취지에 따른 자율 표시입니다.\n4. 이용권은 1개월 단기 패스(자동연장 없음)이며, 결제 후 7일 이내 미사용 시 전액 환불됩니다. 디지털콘텐츠 제공 개시 후에는 청약철회가 제한될 수 있습니다.\n5. 실존 인물을 연상시키는 제목·커버·설명의 등록을 금지합니다.\n6. 전문은 `docs/legal/terms-of-service.md`를 참조하세요.', TRUE),
  ('PARTNER_CONTRACT', 'v1.1', '파트너 음원유통 및 면책 계약서', '# 파트너 음원유통 및 면책 계약서 v1.1\n\n1. 파트너는 플랫폼에 음원의 복제·전송·스트리밍 및 홍보 목적 일부 사용에 대한 비독점적 이용허락을 부여합니다(전 세계, 계약 존속 기간 + 종료 후 30일 정리 기간).\n2. 파트너는 AI 제공자의 상업 이용 조건(플랜·생성 시점·다운로드 적법성) 충족, 제3자 권리(음성·샘플·가사) 비침해, 신탁단체 미신탁(또는 신고)을 보증합니다.\n3. 보증 위반으로 발생한 제3자 청구는 파트너가 부담합니다(면책).\n4. 파트너는 생성 당시 구독 증빙과 생성 기록을 보관하고 요청 시 제출합니다.\n5. AI 제공자 약관 변경으로 유통 근거가 사라지면 플랫폼은 해당 음원을 비공개 처리할 수 있습니다.\n6. 부정행위가 확정되면 해당 부정행위로 발생한 정산금에 한해 지급 보류·상계할 수 있으며, 소명 기회와 산정 근거를 제공합니다.\n7. 중단 요구 접수 시 파트너에게 통보하며, 재개 요구는 접수 후 7영업일 내 검토를 원칙으로 합니다.\n8. 전문은 `docs/legal/partner-contract.md`를 참조하세요.', TRUE),
  ('PRIVACY', 'v1.1', '개인정보처리방침', '# 개인정보처리방침 (Privacy Policy) v1.1\n\n1. 처리위탁: 결제(Toss Payments), 호스팅·CDN(Cloudflare), 이메일 발송(도입 시) — 수탁자명·위탁 업무를 명시합니다.\n2. 국외 이전: Cloudflare 등 해외 인프라 이용 시 이전 국가·일시·방법·받는 자·항목·목적·보유 기간을 고지합니다.\n3. 개인정보 보호책임자: 서비스 푸터 및 본 방침에 실명·연락처를 공개합니다(사업자 등록 후 기재).\n4. 만 14세 미만은 가입할 수 없습니다(가입 시 만 14세 이상 확인).\n5. 정산·세무 기록은 5년, 전자상거래 계약·청약철회·대금결제 기록은 5년, 소비자 불만·분쟁 기록은 3년 보관합니다.\n6. 전문은 `docs/legal/privacy-policy.md`를 참조하세요.', TRUE),
  ('COPYRIGHT_POLICY', 'v1.1', '저작권 및 AI 보이스클로닝 보호정책', '# 저작권 및 AI 보이스클로닝 보호정책 v1.1\n\n1. 본 플랫폼은 저작권법 제102조의 책임 제한 요건을 준수합니다: 반복 침해자 계정 해지 방침(3-strike), 중단 요구 수령인 지정·공지, 중단 요구 접수 시 즉시 복제·전송 중단.\n2. 중단 요구 → 즉시 중단 + 파트너 통보 → 파트너 재개 요구 → 권리자에게 재개 예정일 통보 → 예정일 재개(소송 제기 통지 시 예외) 절차를 따릅니다(저작권법 제103조).\n3. 중단 요구 수령인: 서비스 푸터 및 `/legal/COPYRIGHT_POLICY`에 성명(직책)·이메일·주소·전화를 공지합니다(사업자 등록 후 기재).\n4. 실존 인물의 목소리를 동의 없이 복제·합성한 음원은 업로드·유통이 전면 금지됩니다.\n5. 전문은 `docs/legal/copyright-policy.md`를 참조하세요.', TRUE)
ON CONFLICT (code, version) DO UPDATE SET
  title_ko = EXCLUDED.title_ko,
  body_md = EXCLUDED.body_md,
  is_required = EXCLUDED.is_required;
