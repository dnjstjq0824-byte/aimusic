# AVIBE Security Model

What protects what, enforced where. All claims trace to code.

## 1. Authentication

- Email+password (bcrypt) with 14+ gate (`ageConfirmed`, C9) and optional
  invite gate (`REGISTRATION_INVITE_CODE`, C8).
- Social (Google/Kakao) lands on `USER` and must complete 14+ via
  `POST /auth/age-confirm` before commerce (`RequireAgeConfirmed` →
  `403 ERR_AGE_CONFIRM_REQUIRED`).
- TOTP 2FA (RFC 6238, stdlib): enroll → confirm (10 single-use bcrypt-hashed
  recovery codes, shown once) → login gate (`403 ERR_TOTP_REQUIRED`,
  `401 ERR_TOTP_INVALID`). Disable re-verifies password and revokes sessions.
- JWT: short access (15 min) + 14-day refresh with rotation; `token_version`
  claim checked on every request — password change, withdraw, TOTP disable,
  lock, and `logout-all` revoke all sessions instantly.
- Login throttle: 10 failures / 15 min per email (Redis) → `429 LOGIN_LOCKED`.
- Refresh failure UX: clients clear tokens, toast once, and route to login.

## 2. Authorization

- Roles `USER|PARTNER|ADMIN` (`RequireRole`). Partners additionally pass the
  age + required-terms gates on every `/partner` route.
- Commercial surfaces (stream grant, billing orders/confirm, pre-check,
  finalize) require age confirmation AND current required-terms consent
  (`403 ERR_CONSENT_REQUIRED`; new versions re-gate automatically).
- Admin console additionally guards itself: no self role-change/lock, no
  last-admin demote.

## 3. Transport & edge

- TLS everywhere in prod; HSTS (prod only) + CSP via `SecurityHeaders`
  middleware, nginx template, and Worker headers (Toss `frame-src` allowed).
- CORS allowlist (`CORS_ALLOWED_ORIGINS`); trusted proxies must be set behind
  an LB or client-IP rate limits can be spoofed.
- HLS: private buckets; URLs carry `trackId|jti|exp` HMAC (`HLS_SIGNING_KEY`).
  Worker verifies, strips `sig`, caches immutable segments; API gates HQ
  (pass/admin → `master.m3u8`, else SQ `stream_0.m3u8`) and requires JWT.

## 4. Abuse controls

- Global `RateLimit(100, 200)` + auth group `RateLimit(10, 20)`; buckets are
  in-memory per instance with 5-min idle eviction (see `RateLimiter`).
- Upload cap 500 MiB; SHA-256 dedupe + license-gate recheck in worker;
  FFprobe/bitrate gates per provider policy.
- Heartbeat fraud flags; playback sessions bound to HLS `jti`.

## 5. Data protection

- Secrets at rest: payout accounts AES-256 (`AES256_KEY_BASE64`); TOTP
  secrets plaintext-in-row (needed for verification) with recovery hashes
  bcrypt; password hashes bcrypt; reset tokens stored hashed, single-use,
  30-min TTL.
- Withdrawal purges identity immediately (email alias-freed, name, hash,
  birth year, TOTP, OAuth links, notifications, reset tokens, partner
  contact/payout). Settlements, consents, and streaming logs stay
  pseudonymous for tax/legal retention.
- Self-service export (`GET /auth/export`) excludes all secrets (E2E asserts
  no `password_hash|totp_secret|settlement_account_encrypted`).

## 6. Auditability

- Every admin mutation + `adminctl` role/active change appends to
  `audit_events` (actor, action, entity, metadata, IP). Reads never audit.
- Audit writes never fail the action; failures log `audit record failed`.
- Request IDs on every response; structured JSON logs in API/worker.

## 7. Fail-closed defaults

- Prod boot refuses missing `SMTP_HOST`/short JWT secrets/missing
  `HLS_SIGNING_KEY` (`Config.Validate`).
- Empty `TOSS_SECRET_KEY` → billing 503s (`ERR_PAYMENT_UNAVAILABLE`), never
  fake-pays. Toss confirm/webhook always re-query the PG and verify
  order+amount.
- Billing/stream fail open only where documented (billing outage never
  silences playback grants per existing policy).

## 8. Known residual risks (honest)

- TOTP secrets are recoverable from a DB dump (by design for verification);
  mitigate with managed-DB encryption + least-privilege credentials.
- Rate limits are per-instance (no Redis sharing); put WAF rate rules in
  front for DDoS.
- `ageConfirmed` is self-attestation (no i-PIN/phone verification wired).
- Social-account email is trusted from the provider (Kakao requires
  account-email consent scope).
