# API Reference (Phase 5)

Base URL: `http://localhost:8080/api/v1` (local; see `docker-compose.local.yml` `API_PORT`).
Health: `GET /healthz` → `{"status":"ok","time":...}` (liveness; no auth, no `/api/v1` prefix).
Readiness: `GET /readyz` → `{"status":"ok","checks":{"db":"up","redis":"up","queueDepth":N,"jobsFailed24h":M}}` (503 `degraded` when a dep is down).

Source of truth: `backend/internal/handler/router.go` (+ `auth.go`, `legal.go`,
`partner.go`, `track.go`, `playback.go`, `playlist.go`, `dispute.go`, `admin.go`,
`middleware.go`). Web (`web/src/api/endpoints.ts`) and mobile
(`mobile/lib/services/api_service.dart`) mirror these URIs exactly.

## Conventions

- Success envelope: `{"data": <payload>}` (`200 OK` via `ok()`, `201 Created` via `created()`).
- Error envelope: `{"error": {"code": "<ERR_*>", "message": "<human>"}}`.
- Auth: `Authorization: Bearer <accessToken>`. Access TTL 15 min, refresh 14 days
  (`JWT_ACCESS_TTL_MINUTES`, `JWT_REFRESH_TTL_DAYS`; see `backend/internal/service/auth.go`).
- Roles: `ADMIN`, `PARTNER`, `USER` (`RequireRole` middleware).
- Pagination: `?limit=&offset=`; default limit 20, max 100 (`clampLimit`).
- Rate limits: global 100 rps / burst 200; `/auth/*` 10 rps / burst 20 (`429 ERR_RATE_LIMITED`).
- IDs are UUID strings (`id::text` in SQL).

## Error codes

| Code | HTTP | When |
|------|------|------|
| `ERR_INVALID_INPUT` | 400 | validation failure (handler `ShouldBindJSON`, service `ErrInvalidInput`) |
| `ERR_UNAUTHORIZED` | 401 | missing/invalid Bearer, bad login (`ErrInvalidCredentials`) |
| `ERR_FORBIDDEN` | 403 | wrong role (`RequireRole`), non-owner playlist/counter, `ErrForbidden` |
| `ERR_USER_INACTIVE` | 403 | `users.is_active=false` |
| `ERR_NOT_STREAMABLE` | 403 | track not `READY` or no `hls_prefix` (stream/heartbeat/playlist-add) |
| `ERR_NON_COMMERCIAL_TIER` | 403 | license gate: no/unverified/rejected/expired subscription, or generated after expiry |
| `ERR_RETROACTIVE_GENERATION_DATE` | 403 | license gate: `generatedAt` before subscription window start |
| `ERR_SYNC_ONLY_LICENSE` | 403 | license gate: `allows_standalone_streaming=false` (seed: `SOUNDRAW`, `MUBERT`) |
| `ERR_PROVIDER_NOT_ALLOWED` | 403 | license gate: unknown provider or no policy row (WI-02 allowlist) |
| (blocked source model) | 403 | `ERR_NON_COMMERCIAL_TIER`: `sourceModel` matches `musicgen/audiogen/musiclm/stable-audio-open` (WI-02) |
| `ERR_NOT_FOUND` | 404 | missing row (`ErrNotFound`) |
| `ERR_DUPLICATE_AUDIO_FILE` | 409 | `audioSha256` already in `tracks.original_object_key` (LIKE match) |
| `ERR_EMAIL_TAKEN` | 409 | duplicate registration email |
| `ERR_RATE_LIMITED` | 429 | token bucket empty |
| `ERR_PAYMENT_UNAVAILABLE` | 503 | billing called with `TOSS_SECRET_KEY` unset |
| `ERR_INTERNAL` | 500 | unmapped error |

Gate mapping: `domain.GateError.StatusCode()` → 409 only for duplicate, else 403
(`backend/internal/domain/errors.go`, `handler/middleware.go:AbortError`).

## Auth (`/auth`)

| Method | Path | Auth | Body → Data |
|--------|------|------|-------------|
| POST | `/auth/register` | – | `{email,password(min8),displayName,birthYear?,ageConfirmed:true,inviteCode?}` → `User` (C8 invite gate when `REGISTRATION_INVITE_CODE` set; C9 14+ required) |
| POST | `/auth/register/partner` | – | `{email,password,displayName,company,contactEmail?,birthYear?,ageConfirmed:true,inviteCode?}` → `{user,partner(PENDING)}` |
| POST | `/auth/login` | – | `{email,password,totpCode?}` → `{tokens:{accessToken,refreshToken,expiresIn},user}` (2FA accounts: `403 ERR_TOTP_REQUIRED` without code, `401 ERR_TOTP_INVALID` on wrong code; recovery codes accepted single-use; 10 failures/15 min → `429 LOGIN_LOCKED`) |
| POST | `/auth/refresh` | – | `{refreshToken}` → `{tokens}` |
| GET | `/auth/me` | JWT | – → `{userId,role,ageConfirmedAt}` |
| POST | `/auth/age-confirm` | JWT | `{ageConfirmed:true,birthYear?}` → `{confirmed:true}` (social signups must confirm 14+; commerce gated → `403 ERR_AGE_CONFIRM_REQUIRED`) |
| GET | `/auth/export` | JWT | personal-data bundle download (profile sans secrets, consents, playlists, playback, logs, disputes, orders, partner data) |
| POST | `/auth/logout-all` | JWT | – → `{revoked:true}` (bumps `users.token_version`; all prior access/refresh tokens 401) |
| POST | `/auth/totp/enroll` | JWT | – → `{secret,provisioningUri}` (staged; previous staged secret replaced) |
| POST | `/auth/totp/confirm` | JWT | `{code}` → `{recoveryCodes[10]}` (activates 2FA; plaintext shown once, bcrypt hashes stored) |
| POST | `/auth/totp/disable` | JWT | `{password}` → `{disabled:true}` (re-verifies password; revokes all sessions) |
| POST | `/auth/withdraw` | JWT | – → `{withdrawn:true}` (in-app immediate withdrawal) |
| POST | `/auth/password-reset/request` | – | `{email}` → `202 {requested:true}` (always; no enumeration) |
| POST | `/auth/password-reset/confirm` | – | `{token,newPassword(min8)}` → `{reset:true}` (single-use, 30 min TTL) |
| GET | `/auth/oauth/providers` | – | `[{provider,enabled}]` (drives frontend social buttons) |
| GET | `/auth/oauth/:provider/start` | – | `302` to Google/Kakao authorize (`:provider=google\|kakao`) |
| GET | `/auth/oauth/:provider/callback?code=&state=` | – | `302` to SPA `/oauth/callback?accessToken=&refreshToken=&role=` (failures → `/login?oauthError=<code>`) |

Social accounts always land on `USER`; partners onboard via email + business
verification. Kakao login requires account-email consent.

Example:

```bash
curl -s localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"partner@example.com","password":"..."}'
```

## Legal (`/legal`)

| Method | Path | Auth | Notes |
|--------|------|------|-------|
| GET | `/legal/terms` | – | all `legal_terms` ordered by code/version |
| GET | `/legal/terms/:code` | – | latest effective for `TOS\|PARTNER_CONTRACT\|PRIVACY\|COPYRIGHT_POLICY` |
| POST | `/legal/consents` | JWT | `{termsId}` → `{consented:true}` (idempotent per user+terms) |
| GET | `/legal/consents/me` | JWT | caller consent rows |
| GET | `/legal/consents/pending` | JWT | latest-effective required terms per code missing consent (new versions re-gate commerce → `403 ERR_CONSENT_REQUIRED` on stream/orders/pre-check until consented) |

## Partner (`/partner`, JWT + `PARTNER|ADMIN`)

| Method | Path | Body → Data |
|--------|------|-------------|
| GET | `/license-policies` | policy matrix (`ai_provider_policies`) |
| GET | `/license-policies/:provider` | one provider (`SUNO\|UDIO\|AIVA\|STABLE_AUDIO\|SOUNDRAW\|MUBERT\|CUSTOM_OPEN_MODEL`) |
| GET | `/subscriptions` | caller subscriptions |
| POST | `/subscriptions` | `{provider,planName,proofObjectKey?}` → `PENDING` row |
| POST | `/pre-check` | see below → `{eligible:true,stagingKey,uploadUrl,expiresInSecs:900}` (only on pass) |
| POST | `/finalize` | see below → `{trackId,jobId}` + enqueues `transcode:queue` |
| POST | `/albums` | `{title,releaseType?,coverObjectKey?}` → `Album` |
| GET | `/albums?limit=&offset=` | caller albums |
| GET | `/tracks?limit=&offset=` | caller tracks (all statuses) |
| PUT | `/settlement` | `{bank,accountNumber}` → AES-256-GCM encrypted at rest |
| POST | `/stores` | `{businessNumber,storeName,licenseCount?}` → `201 StoreBusiness PENDING` (WI-05 #19; biz-no checksum) |
| GET | `/stores` | caller store businesses |
| POST | `/stores/:id/devices` | `{deviceName}` → `201 StoreDevice` + 8-char pairing code (15min; ≤ license cap) |
| GET | `/stores/:id/devices` | store player devices |
| PUT | `/stores/:id/devices/:deviceId` | `{deviceName,volume 0..100}` (remote volume) |
| POST | `/stores/:id/devices/:deviceId/revoke` | deactivate player |
| GET/POST | `/stores/:id/schedules` | weekly schedule (`{weekday 0..6,openTime,closeTime,playlistId?,isClosed}`; playlist must be own/public/official) |
| GET | `/stores/:id/devices/:deviceId/logs?limit=` | certified play logs |
| POST | `/generations` | `{prompt,lyrics?,model?,licenseRef?,params?}` → `201 GenerationJob` + enqueues `generation:queue` (WI-03; needs `GENERATION_BACKEND_URL`, else job FAILED with message; WI-08: daily cap `GENERATION_DAILY_CAP` + banned-prompt filter with 24h suspension on 3rd strike) |
| GET | `/generations?limit=` | caller generation jobs, newest first |
| GET | `/generations/:id` | one job (`outputObjectKey` on SUCCEEDED → normal PreCheck/Finalize with its SHA-256) |

Pre-check request (`handler/partner.go:preCheckRequest`):

```json
{
  "provider": "SUNO",
  "audioSha256": "<lowercase hex>",
  "fileName": "track.wav",
  "generatedAt": "2026-11-06T12:00:00Z",
  "subscriptionId": "<uuid, optional>",
  "requiresActiveSub": true
}
```

Gate order (`pkg/licensegate/validator.go`): 1) duplicate SHA → 409,
2) policy `allows_standalone_streaming` → 403 `ERR_SYNC_ONLY_LICENSE`,
2b) provider generation cutoff (`max_generated_at`, e.g. Udio 2025-10-30) → 403 `ERR_NON_COMMERCIAL_TIER`,
3) subscription must be `VERIFIED`, `generatedAt` within
`[verified_at|created_at, expires_at]` → 403 `ERR_RETROACTIVE_GENERATION_DATE` /
`ERR_NON_COMMERCIAL_TIER`. Tier allow/block/sync-only lists and blocked metadata
keywords are enforced inside step 2. The window check is always enforced server-side
(the client `requiresActiveSub` flag is ignored). On pass the staging key
embeds the SHA: `staging/<partnerId>/<sha>/<basename>` and `PresignPut`
TTL is 15 min.

Finalize request (repeats the pre-check grant; the gate is re-validated and
the staging key must bind to the same partner + SHA, else 403):

```json
{
  "stagingKey": "staging/<partnerId>/<sha>/track.wav",
  "title": "My Track",
  "provider": "SUNO",
  "audioSha256": "<lowercase hex, must match stagingKey>",
  "generatedAt": "2026-11-06T12:00:00Z",
  "subscriptionId": "<uuid, optional>",
  "albumId": null, "durationMs": null, "bitrateKbps": null,
  "disclosure": "AI-generated with SUNO (commercial tier)",
  "rights": {
    "declaresOwnership": true, "declaresNoVoiceClone": true,
    "declaresTrainingConsent": true, "sourceModel": "suno-v4",
    "subscriptionId": "<uuid>",
    "collectiveTrustStatus": "NONE", "collectiveName": null
  }
}
```

The three warranties (`declaresOwnership/NoVoiceClone/TrainingConsent`) are
required server-side (`400` when any is false), and
`collectiveTrustStatus=TRUSTED` is refused (`400`, KOMCA-trusted tracks need a
society settlement path first). The worker re-validates
`allows_standalone_streaming`, the generation cutoff, and the subscription
window from the stored rights declaration before HLS packaging, so a track
that bypassed finalize can never reach transcode.

## Tracks (`/tracks`)

| Method | Path | Auth | Notes |
|--------|------|------|-------|
| GET | `/tracks?q=&provider=&genre=&mood=&sort=&limit=&offset=` | – | `READY` only page `{items,total,limit,offset}`; `sort=recent\|oldest\|popular\|title` (default `recent`), `q` = title substring (max 100 chars) |
| GET | `/tracks/:id` | – | one track (any status) |
| GET | `/tracks/:id/stream` | JWT (login required, anonymous 401) | `READY` + `hls_prefix` only → `{url,expiresAt,jti,quality}` signed (`exp,jti,sig`, TTL `HLS_URL_TTL_SECONDS` default 300). Active pass/admin → `master.m3u8` (`HQ` 256k); free → `stream_0.m3u8` (`SQ` 128k). Artist-blocked → `ERR_NOT_STREAMABLE` |

Stream URL: `<CDNBaseURL or S3 endpoint>/<public-hls>/<hls_prefix>/master.m3u8?exp=&jti=&sig=`
(HMAC-SHA256 over `trackId|jti|exp`, see `pkg/crypto/crypto.go`).

## Playback (`/playback`, JWT)

| Method | Path | Body → Data |
|--------|------|-------------|
| POST | `/heartbeat` | `{trackId,positionMs,playMs,jti?}` → `{countedAsPlay,fraudSuspected}` |
| GET | `/playback/state/:trackId` | resume row |
| PUT | `/playback/state/:trackId` | `{positionMs}` → resume row |

Rules (`service/playlist_playback_settlement.go`): `READY` only; counting is
cumulative — each `playMs` delta adds to the uncounted play time since the last
counted play for the user+track, and the delta crossing 30000ms counts once
(clients report ~10s deltas, so a single delta almost never counts alone).
Fraud windows count counted plays only: `>=120 counted/hour/user` or
`>=40 counted/hour/track` → `fraudSuspected`, not counted. Counted plays
`IncrementPlayCount` + `streaming_logs` row; position always upserted when `>=0`.

## Playlists (`/playlists`, JWT)

| Method | Path | Notes |
|--------|------|-------|
| GET | `/playlists` or `/playlists/me` | caller playlists |
| POST | `/playlists` | `{title,isPublic}` → `201` |
| GET/PUT/DELETE | `/playlists/:id` | owner or public-visible for GET; owner-only mutate |
| GET | `/playlists/:id/tracks` | ordered entries |
| POST | `/playlists/:id/tracks` | `{trackId}` (`READY` only, idempotent) → `201` |
| DELETE | `/playlists/:id/tracks/:trackId` | owner-only |
| PUT | `/playlists/:id/reorder` or `/:id/tracks/reorder` | `{trackIds:[...]}` must include every current id |

## UGC Blocks (`/users/blocks`, JWT)

| Method | Path | Notes |
|--------|------|-------|
| POST | `/users/blocks` | `{partnerId,reason?}` → records UGC artist block |
| GET | `/users/blocks` | caller's blocked artists |
| DELETE | `/users/blocks/:partnerId` | unblock artist |

## Store device (`/store/device`, device Bearer [REDACTED] except claim)

| Method | Path | Body → Data |
|--------|------|-------------|
| POST | `/store/device/claim` | `{pairingCode}` → `{device,token}` (WI-05 #20; business must be APPROVED) |
| GET | `/store/device/me` | caller device record |
| GET | `/store/device/queue` | schedule-aware auto queue `{open,reason,playlistId?,volume,tracks[]}` (READY+HLS only; off-hours/closed → empty + reason) |
| POST | `/store/device/heartbeat` | `{trackId,durationSecs}` → `{counted}` (30s+ counts, WI-05 #21) |
| GET | `/store/device/stream/:id` | signed HQ master playlist URL (commercial license; same TTL/signing as listener stream) |

Web player: `/store/player` (claim → auto-play → heartbeat; queue snapshot in localStorage for offline display; true offline segment cache is incompatible with 5-min signed URLs — documented limit).

## Disputes (`/disputes`, JWT)

| Method | Path | Body → Data |
|--------|------|-------------|
| POST | `/disputes/report` | `{trackId,reason,description?}` → `OPEN` (`reason ∈ AI_POLICY_VIOLATION\|VOICE_CLONE\|UNAUTHORIZED_SAMPLE\|OTHER`) |
| POST | `/disputes/:id/counter` | `{note}` (track owner only) → `UNDER_REVIEW`, appends `\n[COUNTER] note` |
| GET | `/disputes/blocks` | caller `user_blocks` (strikes) |

Resolve is admin-only (below). Uphold sets track `TAKEN_DOWN`, dispute
`UPHELD_TAKEDOWN` + `strike_applied=true`, and inserts `user_blocks`
(strike 1: +7d expiry, 2: +30d, 3: no expiry + partner `SUSPENDED`).
Dismiss sets `DISMISSED` with no strike.
Every dispute carries the §103 timeline: `takedown_requested_at` (report),
`partner_notified_at` (takedown/uphold), `resume_requested_at` +
`resume_scheduled_at` (+7d, counter), `resumed_at` (dismiss).

## Admin (`/admin`, JWT + `ADMIN`)

| Method | Path | Body → Data |
|--------|------|-------------|
| GET | `/admin/overview` | console snapshot `{users,tracks,reviews,disputesOpen,plays,queueDepth,jobsFailed24h,settlementsPending}` (`queueDepth=null` when Redis is down) |
| GET | `/admin/pending` or `/admin/tracks/pending` | `{tracks,licenseGateTracks,disputes,partners,subscriptions,rightsByTrack,subsById}` |
| POST | `/admin/review` or `/admin/tracks/:id/review` | `{trackId,approve,reason?}` → `READY` or `LICENSE_GATE_REJECTED` (+Korean reason) |
| PUT | `/admin/tracks/:id/isrc` | `{isrc}` → canonical 12-char assign (WI-05, audited `TRACK_ISRC_ASSIGN`; UNIQUE rejects dupes) |
| GET | `/admin/distribution/dsp-kit?trackIds=a,b&format=json\|csv` | distributor handoff kit (WI-05 #32, audited `DSP_KIT_EXPORT`): READY+ISRC only, others in `skipped[]` with reasons; CSV attachment when `format=csv`; master URLs TTL 1h |
| GET | `/admin/stores?status=` | store businesses by status (WI-05 #19) |
| POST | `/admin/stores/:id/verify` | `{approve}` → `APPROVED`/`SUSPENDED` (audited `STORE_VERIFY`) |
| POST | `/admin/review/partner` | `{partnerId,approve}` → `APPROVED` (+`approved_at`) or `SUSPENDED` |
| POST | `/admin/review/subscription` | `{subscriptionId,approve,expiresAt? RFC3339}` → `VERIFIED`/`REJECTED` (+`verified_at=NOW()`) |
| GET | `/admin/policies` | policy matrix |
| PUT | `/admin/policies/:provider` or `/admin/license-policies/:id` | `{allowsStandaloneStreaming,allowedPlanTiers[],blockedPlanTiers[],syncOnlyPlanTiers[],minBitrateKbps,blockedMetadataKeywords[],rejectionMessageKo,commercialTermsUrl?,maxGeneratedAt? RFC3339,termsSnapshotUrl?}` (every save stamps `terms_checked_at`) |
| GET | `/admin/disputes?limit=&offset=` | `OPEN`+`UNDER_REVIEW` oldest first |
| POST | `/admin/disputes/:id/resolve` or `/admin/disputes/:id/action` | `{uphold,action?}` → see Disputes |
| GET | `/admin/curation?limit=` | `READY` by `play_count` desc |
| POST | `/admin/curation/playlists` | `{title,description?,trackIds:[]}` → creates official curation playlist |
| POST | `/admin/curation/takedown` | `{trackId,reason?}` → immediate `TAKEN_DOWN` + staging/HLS byte purge (status kept on purge failure; retry resumes) |
| GET | `/admin/audit-events?action=&entityType=&limit=&offset=` | append-only admin trail `{id,actorUserId,action,entityType,entityId,metadata,ip,createdAt}` newest first |
| GET | `/admin/users?q=&role=&limit=&offset=` | account search newest first |
| POST | `/admin/users/:id/role` | `{role}` (self-change + last-admin demote refused) |
| POST | `/admin/users/:id/active` | `{active}` (locks revoke sessions; self-lock refused) |
| GET | `/admin/users/:id/blocks` | strike/block history of one account (admin console 제재 내역) |
| POST | `/admin/settlements/calculate` | `{partnerId,periodStart:YYYY-MM-DD,periodEnd:YYYY-MM-DD,ratePerPlayKrw?,incomeType?}` → upserted settlement (`BUSINESS_33` 3.3% default, `OTHER_88` 8.8% occasional income) |
| GET | `/admin/settlements/:partnerId` | settlements newest first |
| POST | `/admin/settlements/approve` | `{partnerId,periodStart,periodEnd}` → `APPROVED` (never from `FORFEITED`) |
| POST | `/admin/settlements/mark-paid` | `{partnerId,periodStart,periodEnd,reference?}` → `PAID` once (double-pay → 409) |
| GET | `/admin/partners/:partnerId/settlement` | `{bank,accountMasked}` (decrypt-then-mask; never plaintext) |

Settlement math (`SettlementService.Calculate`, default 10 KRW/play):
`gross=plays*rate`, `withholding=gross*33/1000` (3.3% business income) or `gross*88/1000` (8.8% occasional income), `net=gross-withholding`.
If partner `SUSPENDED|TERMINATED` or any `UPHELD_TAKEDOWN` on their tracks:
`status=FORFEITED`, `forfeitureReason=FORFEITED_VIOLATION`, `net=0`.
`plays` counts `streaming_logs` with `counted_as_play` in `[periodStart, periodEnd)`.

## Billing (`/billing`, Toss Payments)

Plans are single-term passes (1 month, no auto-renew). With `TOSS_SECRET_KEY`
unset, order/confirm/webhook endpoints fail closed with `503
ERR_PAYMENT_UNAVAILABLE`; the plan catalog still works.

| Method | Path | Auth | Body → Data |
|--------|------|------|-------------|
| GET | `/billing/plans` | – | active plans cheapest first |
| POST | `/billing/orders` | JWT | `{planCode,refundConsentAt RFC3339}` → `201 {order,plan}` (`PENDING`, merchant id `AVIBE-YYYYMMDD-XXXXXXXX`; consent ≤24h old, stored with `refund_terms_version`) |
| POST | `/billing/payments/confirm` | JWT | `{paymentKey,orderId,amount}` → `{subscription}` (Toss approve + `ACTIVE` period) |
| POST | `/billing/webhooks/toss` | – | Toss transmission → `{received:true}` (verified by re-query, idempotent) |
| GET | `/billing/subscriptions/me` | JWT | `{subscription}` (`null` when no active pass) |
| POST | `/billing/subscriptions/cancel` | JWT | – → `{cancelled:true}` |
| POST | `/billing/payments/refund` | JWT | `{orderId?,reason?}` → `{order:REFUNDED}` (empty `orderId` refunds the ACTIVE pass; Toss full void + pass ends immediately; `ALREADY_CANCELED` converges) |

Confirm and webhook both verify `orderId` + `amount` against the stored order
and against a live Toss re-query (`GET /v1/payments/{paymentKey}`); `DONE`
activates the period, `CANCELED` cancels it, duplicates are journaled once in
`billing_webhook_events` and acknowledged without re-applying.

## Clients

- Web: `web/src/api/client.ts` (`ERR_MESSAGES` Korean map, Bearer injection,
  401 refresh retry) + `web/src/api/endpoints.ts` (typed wrappers per section above).
- Mobile: `mobile/lib/services/api_service.dart` (`ApiService`: tracks/playback/
  playlists/disputes/legal; URIs match `router.go`; `StreamTicket` throws
  `ERR_NOT_STREAMABLE` for takedown/non-READY).
