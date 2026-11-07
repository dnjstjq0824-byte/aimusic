# API Reference (Phase 5)

Base URL: `http://localhost:8080/api/v1` (local; see `docker-compose.local.yml` `API_PORT`).
Health: `GET /healthz` → `{"status":"ok"}` (no auth, no `/api/v1` prefix).

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
| POST | `/auth/register` | – | `{email,password(min8),displayName}` → `User` |
| POST | `/auth/register/partner` | – | `{email,password,displayName,company,contactEmail?}` → `{user,partner(PENDING)}` |
| POST | `/auth/login` | – | `{email,password}` → `{tokens:{accessToken,refreshToken,expiresIn},user}` |
| POST | `/auth/refresh` | – | `{refreshToken}` → `{tokens}` |
| GET | `/auth/me` | JWT | – → `{userId,role}` |
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
3) subscription must be `VERIFIED`, `generatedAt` within
`[verified_at|created_at, expires_at]` → 403 `ERR_RETROACTIVE_GENERATION_DATE` /
`ERR_NON_COMMERCIAL_TIER`. The window check is always enforced server-side
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
    "subscriptionId": "<uuid>"
  }
}
```

The worker re-validates `allows_standalone_streaming` plus the subscription
window from the stored rights declaration before HLS packaging, so a track
that bypassed finalize can never reach transcode.

## Tracks (`/tracks`)

| Method | Path | Auth | Notes |
|--------|------|------|-------|
| GET | `/tracks?q=&provider=&genre=&mood=&sort=&limit=&offset=` | – | `READY` only page `{items,total,limit,offset}`; `sort=recent\|oldest\|popular\|title` (default `recent`), `q` = title substring (max 100 chars) |
| GET | `/tracks/:id` | – | one track (any status) |
| GET | `/tracks/:id/stream` | JWT | `READY` + `hls_prefix` only → `{url,expiresAt,jti}` signed (`exp,jti,sig`, TTL `HLS_URL_TTL_SECONDS` default 300) |

Stream URL: `<CDNBaseURL or S3 endpoint>/<public-hls>/<hls_prefix>/master.m3u8?exp=&jti=&sig=`
(HMAC-SHA256 over `trackId|jti|exp`, see `pkg/crypto/crypto.go`).

## Playback (`/playback`, JWT)

| Method | Path | Body → Data |
|--------|------|-------------|
| POST | `/heartbeat` | `{trackId,positionMs,playMs,jti?}` → `{countedAsPlay,fraudSuspected}` |
| GET | `/playback/state/:trackId` | resume row |
| PUT | `/playback/state/:trackId` | `{positionMs}` → resume row |

Rules (`service/playlist_playback_settlement.go`): `READY` only; `playMs>=30000`
counts unless fraud (`>=120 plays/hour/user` or `>=40/hour/track` → `fraudSuspected`,
not counted). Counted plays `IncrementPlayCount` + `streaming_logs` row; position
always upserted when `>=0`.

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

## Admin (`/admin`, JWT + `ADMIN`)

| Method | Path | Body → Data |
|--------|------|-------------|
| GET | `/admin/overview` | console snapshot `{users,tracks,reviews,disputesOpen,plays,queueDepth,jobsFailed24h,settlementsPending}` (`queueDepth=null` when Redis is down) |
| GET | `/admin/pending` or `/admin/tracks/pending` | `{tracks,licenseGateTracks,disputes,partners,subscriptions,rightsByTrack,subsById}` |
| POST | `/admin/review` or `/admin/tracks/:id/review` | `{trackId,approve,reason?}` → `READY` or `LICENSE_GATE_REJECTED` (+Korean reason) |
| POST | `/admin/review/partner` | `{partnerId,approve}` → `APPROVED` (+`approved_at`) or `SUSPENDED` |
| POST | `/admin/review/subscription` | `{subscriptionId,approve,expiresAt? RFC3339}` → `VERIFIED`/`REJECTED` (+`verified_at=NOW()`) |
| GET | `/admin/policies` | policy matrix |
| PUT | `/admin/policies/:provider` or `/admin/license-policies/:id` | `{allowsStandaloneStreaming,allowedPlanTiers[],blockedPlanTiers[],syncOnlyPlanTiers[],minBitrateKbps,blockedMetadataKeywords[],rejectionMessageKo,commercialTermsUrl?}` |
| GET | `/admin/disputes?limit=&offset=` | `OPEN`+`UNDER_REVIEW` oldest first |
| POST | `/admin/disputes/:id/resolve` or `/admin/disputes/:id/action` | `{uphold,action?}` → see Disputes |
| GET | `/admin/curation?limit=` | `READY` by `play_count` desc |
| POST | `/admin/curation/playlists` | `{title,description?,trackIds:[]}` → creates official curation playlist |
| POST | `/admin/curation/takedown` | `{trackId,reason?}` → immediate `TAKEN_DOWN` |
| POST | `/admin/settlements/calculate` | `{partnerId,periodStart:YYYY-MM-DD,periodEnd:YYYY-MM-DD,ratePerPlayKrw?}` → upserted settlement |
| GET | `/admin/settlements/:partnerId` | settlements newest first |
| POST | `/admin/settlements/approve` | `{partnerId,periodStart,periodEnd}` → `APPROVED` (never from `FORFEITED`) |
| GET | `/admin/partners/:partnerId/settlement` | `{bank,accountMasked}` (decrypt-then-mask; never plaintext) |

Settlement math (`SettlementService.Calculate`, default 10 KRW/play):
`gross=plays*rate`, `withholding=gross*33/1000` (3.3%), `net=gross-withholding`.
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
| POST | `/billing/orders` | JWT | `{planCode}` → `201 {order,plan}` (`PENDING`, merchant id `AIVIBE-YYYYMMDD-XXXXXXXX`) |
| POST | `/billing/payments/confirm` | JWT | `{paymentKey,orderId,amount}` → `{subscription}` (Toss approve + `ACTIVE` period) |
| POST | `/billing/webhooks/toss` | – | Toss transmission → `{received:true}` (verified by re-query, idempotent) |
| GET | `/billing/subscriptions/me` | JWT | `{subscription}` (`null` when no active pass) |
| POST | `/billing/subscriptions/cancel` | JWT | – → `{cancelled:true}` |

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
