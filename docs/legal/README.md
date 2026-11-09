# Legal Index (v1.0, effective 2026-11-06)

Canonical Korean source texts for `legal_terms` + `user_term_consents`
(see `backend/migrations/seed.sql`, `backend/internal/repository/legal_dispute.go`,
`GET /legal/terms*`, `POST /legal/consents`).

| Code | Version | File | Title | Required | Seed body |
|------|---------|------|-------|----------|-----------|
| `TOS` | `v1.0` | `terms-of-service.md` | 이용약관 | TRUE | `# 이용약관 v1.0` + pointer to this file |
| `PARTNER_CONTRACT` | `v1.0` | `partner-contract.md` | 파트너 계약 | TRUE | `# 파트너 계약 v1.0` + pointer |
| `PRIVACY` | `v1.0` | `privacy-policy.md` | 개인정보처리방침 | TRUE | `# 개인정보처리방침 v1.0` + pointer |
| `COPYRIGHT_POLICY` | `v1.0` | `copyright-policy.md` | 저작권 정책 | TRUE | `# 저작권 정책 v1.0` + pointer |

> **출시 전 필독:** [LAUNCH_COMPLIANCE.md](LAUNCH_COMPLIANCE.md) — 서비스 오픈 법무 컴플라이언스
> (출시 차단 항목, AI 제공자 약관 매트릭스, OSP 면책 요건, 표시광고·전자상거래·개인정보·세무, 개발 반영 체크리스트).
> 아래 v1.0 약관 문서들은 이 문서 12장 D1–D4에 따라 v1.1 개정이 필요합니다.

## What each doc covers

- `terms-of-service.md`: AI 표시 의무(인공지능기본법 제31조), 금지행위(보이스클로닝·침해·메타조작·어뷰징),
  OSP 임시조치, 3-strike(경고/30일/영구+몰수 가능), 3.3% 원천징수, 준거법.
- `partner-contract.md`: 유료 구독 증빙(무료/Sync-only 거부), 권리선언 3종, AI 표시,
  보이스클로닝 금지, 월 정산+3.3%, 부정행위 몰수, OSP 7일 소명, 해지.
- `privacy-policy.md`: 수집 항목(회원/파트너/AES-256 계좌/이용기록/증빙), 보유·파기(탈퇴 즉시,
  세무 5년), 제3자 제공(세무), 열람·정정·삭제 10일, bcrypt/TLS/최소권한.
- `copyright-policy.md`: 라이선스 게이트(standalone 불가·무료·차단키워드·192kbps 거부+한국어 통지),
  보이스클로닝 블라인드, OSP 30일 임시조치+소명 복원, 3-strike(`user_blocks`), 몰수+3.3%.

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
