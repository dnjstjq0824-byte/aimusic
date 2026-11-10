# Legal Index (v1.1, effective 2026-11-10)

Canonical Korean source texts for `legal_terms` + `user_term_consents`
(see `backend/migrations/seed.sql`, `backend/internal/repository/legal_dispute.go`,
`GET /legal/terms*`, `POST /legal/consents`).

| Code | Version | File | Title | Required | Seed body |
|------|---------|------|-------|----------|-----------|
| `TOS` | `v1.1` | `terms-of-service.md` | 서비스 이용약관 | TRUE | v1.1 summary + pointer to this file |
| `PARTNER_CONTRACT` | `v1.1` | `partner-contract.md` | 파트너 음원유통 및 면책 계약서 | TRUE | v1.1 summary + pointer |
| `PRIVACY` | `v1.1` | `privacy-policy.md` | 개인정보처리방침 | TRUE | v1.1 summary + pointer |
| `COPYRIGHT_POLICY` | `v1.1` | `copyright-policy.md` | 저작권 및 AI 보이스클로닝 보호정책 | TRUE | v1.1 summary + pointer |

> **출시 전 필독:** [LAUNCH_COMPLIANCE.md](LAUNCH_COMPLIANCE.md) — 서비스 오픈 법무 컴플라이언스
> (출시 차단 항목, AI 제공자 약관 매트릭스, OSP 면책 요건, 표시광고·전자상거래·개인정보·세무, 개발 반영 체크리스트).
> v1.0 → v1.1 개정으로 이 문서 12장 D1–D4가 반영되었습니다(수령인·보호책임자 실명 기재는 사업자 등록 후).

## What each doc covers

- `terms-of-service.md`: AI 생성 표시(제31조 취지)+확인 절차 한계 고지, 금지행위(실존 인물 연상 포함),
  저작권법 §103 중단 절차, 3-strike, 3.3%/8.8% 원천징수, 이용권 청약철회, 만 14세 제한, 준거법.
- `partner-contract.md`: 이용허락 범위(전 세계·존속+30일), 구독 증빙+생성 기록 보관+약관 변경 비공개,
  보증 6종(신탁 신고 포함)+면책, AI 표시, 월 정산+3.3%/8.8%, 지급 보류·상계, §103 통보, 해지.
- `privacy-policy.md`: 수집 항목(연령 확인·결제 포함), 보유·파기(세무 5년·전자상거래 5/3년),
  처리위탁(Toss·Cloudflare)+국외이전 고지, 권리 10일, 만 14세, bcrypt/TLS, 보호책임자란.
- `copyright-policy.md`: 라이선스 게이트(컷오프 포함)+한국어 통지, 보이스클로닝 블라인드,
  §103 중단 절차+§103 타임라인 보관, 수령인 공지란, 3-strike(`user_blocks`), §102 반복 침해자 해지.

## API & DB wiring

- `GET /legal/terms` (all, `ORDER BY code, version DESC`), `GET /legal/terms/:code`
  (latest `effective_at`), `POST /legal/consents {termsId}` (`ON CONFLICT DO NOTHING`),
  `GET /legal/consents/me`. Web: `legalApi`; mobile: `ApiService.listTerms/getTerm/createConsent`.
- Enforcement mapping: TOS §4/§6 + copyright §§2–5 → pre-check gate (`ERR_*`),
  worker metadata/bitrate reject, dispute `UPHELD_TAKEDOWN` + `user_blocks`,
  settlement `FORFEITED_VIOLATION`; TOS §7 + partner §§6–7 + copyright §6 → 3.3%
  (`*33/1000`) + forfeiture; privacy §§1/5 → `users`/`partner_profiles`/`streaming_logs`.

## Updating terms

1. Edit the `.md` file(s) here, bump version (e.g. `v1.1`) + `시행일`.
2. `INSERT INTO legal_terms (code,version,title_ko,body_md,is_required)` with the new
   version (keep `UNIQUE(code,version)`; `GetTermByCode` picks latest `effective_at`).
3. If re-consent is needed, gate features on `ListConsentsByUser` for the new `terms_id`.
4. Update `seed.sql` only for fresh-local defaults; never rewrite history in prod.
