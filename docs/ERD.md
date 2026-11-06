# ERD (Phase 5)

Source of truth: `backend/migrations/000001_init.up.sql` (17 tables),
`backend/internal/domain/entities.go` (Go mirrors), `backend/internal/repository/*.go`
(access), `backend/migrations/seed.sql` (policy matrix + demo rows).
Conventions: UUID PKs `gen_random_uuid()`, `TIMESTAMPTZ`, `CHECK` enums,
`set_updated_at()` trigger on mutable tables.

## Diagram (Mermaid)

```mermaid
erDiagram
  users ||--o| partner_profiles : "user_id unique, CASCADE"
  users ||--o{ user_term_consents : "CASCADE"
  legal_terms ||--o{ user_term_consents : "RESTRICT"
  partner_profiles ||--o{ partner_ai_subscriptions : "CASCADE"
  users ||--o{ partner_ai_subscriptions : "verified_by SET NULL"
  partner_profiles ||--o{ albums : "CASCADE"
  partner_profiles ||--o{ tracks : "CASCADE"
  albums ||--o{ tracks : "album_id SET NULL"
  tracks ||--o| track_rights_declarations : "track_id unique, CASCADE"
  partner_ai_subscriptions ||--o{ track_rights_declarations : "SET NULL"
  tracks ||--o{ transcoding_jobs : "CASCADE"
  tracks ||--o{ copyright_disputes : "CASCADE"
  users ||--o{ copyright_disputes : "reporter SET NULL"
  users ||--o{ copyright_disputes : "resolver SET NULL"
  users ||--o{ playlists : "owner CASCADE"
  playlists ||--o{ playlist_tracks : "CASCADE"
  tracks ||--o{ playlist_tracks : "CASCADE"
  users ||--o{ user_playback_states : "CASCADE"
  tracks ||--o{ user_playback_states : "CASCADE"
  users ||--o{ streaming_logs : "SET NULL"
  tracks ||--o{ streaming_logs : "SET NULL"
  partner_profiles ||--o{ streaming_logs : "SET NULL"
  partner_profiles ||--o{ partner_settlements : "CASCADE"
  users ||--o{ user_blocks : "CASCADE"
  users ||--o{ user_blocks : "blocked_by SET NULL"

  users {
    uuid id PK
    text email "unique"
    text password_hash "bcrypt"
    text role "ADMIN|PARTNER|USER"
    text display_name
    bool is_active
  }
  partner_profiles {
    uuid id PK
    uuid user_id FK "unique"
    text company_name
    text contact_email
    text status "PENDING|APPROVED|SUSPENDED|TERMINATED"
  }
  ai_provider_policies {
    uuid id PK
    text provider "unique, 7 values"
    bool allows_standalone_streaming
    int min_bitrate_kbps
    text_array blocked_metadata_keywords
    text rejection_message_ko
  }
  partner_ai_subscriptions {
    uuid id PK
    uuid partner_id FK
    text provider
    text plan_name
    text status "PENDING|VERIFIED|REJECTED|EXPIRED"
  }
  tracks {
    uuid id PK
    uuid partner_id FK
    uuid album_id FK "nullable"
    text title
    text original_object_key "embeds SHA-256"
    text hls_prefix "nullable, READY only"
    text ai_provider
    text status "UPLOADED|LICENSE_GATE_PENDING|LICENSE_GATE_REJECTED|TRANSCODING|READY|TAKEN_DOWN"
    bigint play_count
  }
  transcoding_jobs {
    uuid id PK
    uuid track_id FK
    text status "QUEUED|RUNNING|SUCCEEDED|FAILED"
    int attempts
  }
  copyright_disputes {
    uuid id PK
    uuid track_id FK
    text reason "AI_POLICY_VIOLATION|VOICE_CLONE|UNAUTHORIZED_SAMPLE|OTHER"
    text status "OPEN|UNDER_REVIEW|UPHELD_TAKEDOWN|DISMISSED"
    bool strike_applied
  }
  partner_settlements {
    uuid id PK
    uuid partner_id FK
    date period_start
    date period_end "unique(partner,period)"
    bigint gross_amount_krw
    bigint withholding_krw "3.3%"
    bigint net_amount_krw
    text status "PENDING|APPROVED|PAID|FORFEITED"
  }
```

## Tables (all 17)

### 1. `users`
`id, email UNIQUE, password_hash (bcrypt), role CHECK(ADMIN|PARTNER|USER) DEFAULT USER,`
`display_name, is_active DEFAULT TRUE, created_at, updated_at`.
Repo: `UserRepo` (`Create/GetByID/GetByEmail/SetActive`).

### 2. `partner_profiles`
`id, user_id UNIQUE FK users CASCADE, company_name, business_number?,`
`settlement_bank?, settlement_account_encrypted? (AES-256-GCM base64, see `pkg/crypto`),`
`contact_email, status CHECK(PENDING|APPROVED|SUSPENDED|TERMINATED) DEFAULT PENDING,`
`approved_at?, created_at, updated_at`.
Repo: `PartnerRepo` (`Create/GetByID/GetByUserID/ListByStatus/UpdateStatus/UpdateSettlement`).
`UpdateStatus(approved=true)` stamps `approved_at=NOW()`.

### 3. `ai_provider_policies`
`id, provider UNIQUE CHECK(SUNO|UDIO|AIVA|STABLE_AUDIO|SOUNDRAW|MUBERT|CUSTOM_OPEN_MODEL),`
`allows_standalone_streaming, min_bitrate_kbps DEFAULT 192 CHECK>0,`
`blocked_metadata_keywords TEXT[] DEFAULT '{}', rejection_message_ko, commercial_terms_url?`.
Seed: SUNO/UDIO/AIVA/STABLE/CUSTOM `TRUE`; SOUNDRAW/MUBERT `FALSE` (SYNC-only).
Repo: `PolicyRepo` (`Get/List/Update` via `string_to_array(chr(31))`).
Gate view: `licensegate.Policy`.

### 4. `partner_ai_subscriptions`
`id, partner_id FK CASCADE, provider CHECK(7), plan_name, proof_object_key?,`
`status CHECK(PENDING|VERIFIED|REJECTED|EXPIRED) DEFAULT PENDING,`
`verified_by? FK users SET NULL, verified_at?, expires_at?, created_at, updated_at,`
`UNIQUE(partner_id,provider)`, index `(partner_id)`.
Repo: `SubscriptionRepo` (`Create/GetByID/GetByPartnerProvider/ListByPartner/ListPending/UpdateVerification` sets `verified_at=NOW()`).
Seed: one `SUNO Pro VERIFIED` for the demo partner, `expires_at NOW()+30d`.

### 5. `legal_terms`
`id, code CHECK(TOS|PARTNER_CONTRACT|PRIVACY|COPYRIGHT_POLICY), version,`
`title_ko, body_md DEFAULT '', is_required DEFAULT TRUE, effective_at DEFAULT NOW(),`
`created_at, UNIQUE(code,version)`.
Seed: 4 rows `v1.0` pointing at `docs/legal/*.md`.

### 6. `user_term_consents`
`id, user_id FK CASCADE, terms_id FK RESTRICT, consented_at DEFAULT NOW(), ip?, user_agent?,`
`UNIQUE(user_id,terms_id)`, index `(user_id)`.
Repo: `LegalRepo.CreateConsent` is `ON CONFLICT DO NOTHING` (idempotent).

### 7. `albums`
`id, partner_id FK CASCADE, title, cover_object_key?,`
`release_type CHECK(SINGLE|EP|ALBUM) DEFAULT SINGLE,`
`status CHECK(DRAFT|PENDING_REVIEW|LIVE|TAKEN_DOWN) DEFAULT DRAFT,`
`released_at?, created_at, updated_at`, index `(partner_id)`.

### 8. `tracks`
`id, album_id? FK SET NULL, partner_id FK CASCADE, title, duration_ms? CHECK>0,`
`original_object_key NOT NULL (staging key embeds client SHA-256; duplicate check is`
`LIKE '%sha%' in `GateStore.AudioHashExists` / `TrackRepo.ExistsByHash`),`
`hls_prefix? (set on transcode, `hls/<trackId>`), ai_provider CHECK(7),`
`ai_generated DEFAULT TRUE, ai_disclosure_text?, bitrate_kbps? CHECK>0,`
`status CHECK(UPLOADED|LICENSE_GATE_PENDING|LICENSE_GATE_REJECTED|TRANSCODING|READY|TAKEN_DOWN) DEFAULT UPLOADED,`
`rejection_reason_ko?, play_count DEFAULT 0, created_at, updated_at,`
indexes `(partner_id)`, `(status)`.
Lifecycle: `Finalize→UPLOADED` → worker `TRANSCODING` → `READY` (`SetTranscoded`)
or `LICENSE_GATE_REJECTED`; admin `ReviewTrack` → `READY`/`LICENSE_GATE_REJECTED`;
dispute uphold / takedown → `TAKEN_DOWN`.

### 9. `track_rights_declarations`
`id, track_id UNIQUE FK CASCADE, partner_id FK CASCADE,`
`declares_ownership/no_voice_clone/training_consent DEFAULT FALSE,`
`source_model?, subscription_id? FK SET NULL, declared_at`.
Repo: `RightsRepo.Upsert` (`ON CONFLICT(track_id) DO UPDATE`).

### 10. `transcoding_jobs`
`id, track_id FK CASCADE, status CHECK(QUEUED|RUNNING|SUCCEEDED|FAILED) DEFAULT QUEUED,`
`input_object_key, output_prefix?, bitrate_kbps? CHECK>0, error?, attempts DEFAULT 0,`
`started_at?, finished_at?, created_at, updated_at`, indexes `(track_id)`, `(status)`.
Worker (`cmd/worker`): `MarkRunning` → ffprobe/blocked-keyword/bitrate gates →
ffmpeg HLS → `MarkSucceeded`+`Tracks.SetTranscoded`, else `MarkFailed` (attempts+1;
license rejects → `LICENSE_GATE_REJECTED` immediately, transient errors requeue to
`transcode:queue` until 3 attempts).

### 11. `copyright_disputes`
`id, track_id FK CASCADE, reporter_user_id? FK SET NULL,`
`reason CHECK(AI_POLICY_VIOLATION|VOICE_CLONE|UNAUTHORIZED_SAMPLE|OTHER),`
`description DEFAULT '', status CHECK(OPEN|UNDER_REVIEW|UPHELD_TAKEDOWN|DISMISSED) DEFAULT OPEN,`
`strike_applied DEFAULT FALSE, resolved_by? FK SET NULL, resolved_at?, created_at, updated_at,`
indexes `(track_id)`, `(status)`.
`AppendCounter` appends `\n[COUNTER] note` + `UNDER_REVIEW`; `Resolve` sets
status/strike/resolver/`resolved_at=NOW()`; `CountUpheldByPartner` drives settlement forfeiture.

### 12. `playlists`
`id, owner_user_id FK CASCADE, title, is_public DEFAULT FALSE, created_at, updated_at,`
index `(owner_user_id)`.

### 13. `playlist_tracks`
`id, playlist_id FK CASCADE, track_id FK CASCADE, position CHECK>=0, added_at,`
`UNIQUE(playlist_id,track_id)`, index `(playlist_id)`.
`AddTrack` is idempotent (`ON CONFLICT DO NOTHING`, `COALESCE(MAX(position)+1,0)`);
`Reorder` rewrites positions transactionally (must include every id).

### 14. `user_playback_states`
`id, user_id FK CASCADE, track_id FK CASCADE, position_ms DEFAULT 0 CHECK>=0,`
`created_at, updated_at, UNIQUE(user_id,track_id)`.
`UpsertState` is `ON CONFLICT(user_id,track_id) DO UPDATE`.

### 15. `streaming_logs` (append-only)
`id, user_id?/track_id?/partner_id? FK SET NULL, played_at DEFAULT NOW(),`
`play_ms DEFAULT 0 CHECK>=0, counted_as_play DEFAULT FALSE, ip?, user_agent?, hls_token_jti?,`
indexes `(played_at)`, `(track_id)`, `(partner_id,played_at)`.
`CountPartnerPlaysInWindow` (`counted_as_play` in `[start,end)`) drives settlement;
`CountUserPlaysSince` / `CountUserTrackPlaysSince` (1h) drive fraud flags.

### 16. `partner_settlements`
`id, partner_id FK CASCADE, period_start DATE, period_end DATE,`
`gross/withholding/net/play_count BIGINT DEFAULT 0 CHECK>=0,`
`status CHECK(PENDING|APPROVED|PAID|FORFEITED) DEFAULT PENDING, forfeiture_reason?,`
`created_at, updated_at, UNIQUE(partner_id,period_start,period_end), CHECK(start<=end)`.
`Upsert` per partner+period; `Approve` refuses `FORFEITED`.
Math: `gross=plays*rate`, `withholding=gross*33/1000`, `net=gross-withholding`,
violated (`SUSPENDED|TERMINATED` or upheld>0) → `FORFEITED/FORFEITED_VIOLATION/net=0`.

### 17. `user_blocks` (3-strike)
`id, user_id FK CASCADE, blocked_by? FK SET NULL, reason,`
`strike_number CHECK(1..3), expires_at? (NULL = permanent on #3), created_at,`
index `(user_id)`. `CountStrikes` caps display at 3; strike 3 suspends the partner.

## Seed (`seed.sql`, idempotent)

- `users`: fixed UUIDs `a000...001/002/003` (admin/partner/user, dummy bcrypt, local only)
  `ON CONFLICT(email) DO NOTHING`; `partner_profiles b000...001 APPROVED`.
- `ai_provider_policies`: 7 rows `ON CONFLICT(provider) DO UPDATE` (matrix above).
- `partner_ai_subscriptions c000...001`: `SUNO Pro VERIFIED`, `verified_at NOW()`,
  `expires_at NOW()+30d`, `ON CONFLICT(partner,provider) DO UPDATE`.
- `legal_terms`: 4×`v1.0` `ON CONFLICT(code,version) DO NOTHING`.
