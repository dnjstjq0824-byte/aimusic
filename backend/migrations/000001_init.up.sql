-- Phase 1 initial schema: AI music streaming platform.
-- Conventions: UUID PKs via gen_random_uuid(), TIMESTAMPTZ everywhere, CHECK-based enums.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Shared updated_at trigger.
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 1. users
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT 'USER'
    CHECK (role IN ('ADMIN', 'PARTNER', 'USER')),
  display_name TEXT NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT users_email_unique UNIQUE (email)
);
CREATE TRIGGER trg_users_updated_at
  BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 2. partner_profiles
CREATE TABLE partner_profiles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  company_name TEXT NOT NULL,
  business_number TEXT,
  settlement_bank TEXT,
  settlement_account_encrypted TEXT,
  contact_email TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'PENDING'
    CHECK (status IN ('PENDING', 'APPROVED', 'SUSPENDED', 'TERMINATED')),
  approved_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT partner_profiles_user_unique UNIQUE (user_id),
  CONSTRAINT fk_partner_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
CREATE TRIGGER trg_partner_profiles_updated_at
  BEFORE UPDATE ON partner_profiles FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 3. ai_provider_policies (license-gate matrix)
CREATE TABLE ai_provider_policies (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  provider TEXT NOT NULL
    CHECK (provider IN ('SUNO', 'UDIO', 'AIVA', 'STABLE_AUDIO', 'SOUNDRAW', 'MUBERT', 'CUSTOM_OPEN_MODEL')),
  allows_standalone_streaming BOOLEAN NOT NULL,
  allowed_plan_tiers TEXT[] NOT NULL DEFAULT '{}',
  blocked_plan_tiers TEXT[] NOT NULL DEFAULT '{}',
  sync_only_plan_tiers TEXT[] NOT NULL DEFAULT '{}',
  min_bitrate_kbps INT NOT NULL DEFAULT 192 CHECK (min_bitrate_kbps > 0),
  blocked_metadata_keywords TEXT[] NOT NULL DEFAULT '{}',
  rejection_message_ko TEXT NOT NULL,
  commercial_terms_url TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT ai_provider_policies_provider_unique UNIQUE (provider)
);
CREATE TRIGGER trg_ai_provider_policies_updated_at
  BEFORE UPDATE ON ai_provider_policies FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 4. partner_ai_subscriptions
CREATE TABLE partner_ai_subscriptions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  partner_id UUID NOT NULL,
  provider TEXT NOT NULL
    CHECK (provider IN ('SUNO', 'UDIO', 'AIVA', 'STABLE_AUDIO', 'SOUNDRAW', 'MUBERT', 'CUSTOM_OPEN_MODEL')),
  plan_name TEXT NOT NULL,
  proof_object_key TEXT,
  status TEXT NOT NULL DEFAULT 'PENDING'
    CHECK (status IN ('PENDING', 'VERIFIED', 'REJECTED', 'EXPIRED')),
  verified_by UUID,
  verified_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT partner_ai_subscriptions_partner_provider_unique UNIQUE (partner_id, provider),
  CONSTRAINT fk_sub_partner FOREIGN KEY (partner_id) REFERENCES partner_profiles (id) ON DELETE CASCADE,
  CONSTRAINT fk_sub_verifier FOREIGN KEY (verified_by) REFERENCES users (id) ON DELETE SET NULL
);
CREATE INDEX idx_subscriptions_partner ON partner_ai_subscriptions (partner_id);
CREATE TRIGGER trg_partner_ai_subscriptions_updated_at
  BEFORE UPDATE ON partner_ai_subscriptions FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 5. legal_terms
CREATE TABLE legal_terms (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  code TEXT NOT NULL
    CHECK (code IN ('TOS', 'PARTNER_CONTRACT', 'PRIVACY', 'COPYRIGHT_POLICY')),
  version TEXT NOT NULL,
  title_ko TEXT NOT NULL,
  body_md TEXT NOT NULL DEFAULT '',
  is_required BOOLEAN NOT NULL DEFAULT TRUE,
  effective_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT legal_terms_code_version_unique UNIQUE (code, version)
);

-- 6. user_term_consents
CREATE TABLE user_term_consents (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  terms_id UUID NOT NULL,
  consented_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ip TEXT,
  user_agent TEXT,
  CONSTRAINT user_term_consents_unique UNIQUE (user_id, terms_id),
  CONSTRAINT fk_consent_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  CONSTRAINT fk_consent_terms FOREIGN KEY (terms_id) REFERENCES legal_terms (id) ON DELETE RESTRICT
);
CREATE INDEX idx_consents_user ON user_term_consents (user_id);

-- 7. albums
CREATE TABLE albums (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  partner_id UUID NOT NULL,
  title TEXT NOT NULL,
  cover_object_key TEXT,
  release_type TEXT NOT NULL DEFAULT 'SINGLE'
    CHECK (release_type IN ('SINGLE', 'EP', 'ALBUM')),
  status TEXT NOT NULL DEFAULT 'DRAFT'
    CHECK (status IN ('DRAFT', 'PENDING_REVIEW', 'LIVE', 'TAKEN_DOWN')),
  released_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT fk_album_partner FOREIGN KEY (partner_id) REFERENCES partner_profiles (id) ON DELETE CASCADE
);
CREATE INDEX idx_albums_partner ON albums (partner_id);
CREATE TRIGGER trg_albums_updated_at
  BEFORE UPDATE ON albums FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 8. tracks
CREATE TABLE tracks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  album_id UUID,
  partner_id UUID NOT NULL,
  title TEXT NOT NULL,
  genre TEXT,
  mood_tags TEXT[] NOT NULL DEFAULT '{}',
  ai_generation_type TEXT NOT NULL DEFAULT 'FULL_AI'
    CHECK (ai_generation_type IN ('FULL_AI', 'AI_ASSISTED_HUMAN')),
  ai_model_version TEXT,
  ai_prompt TEXT,
  is_prompt_public BOOLEAN NOT NULL DEFAULT TRUE,
  is_instrumental BOOLEAN NOT NULL DEFAULT FALSE,
  lyrics TEXT,
  duration_ms INT CHECK (duration_ms IS NULL OR duration_ms > 0),
  original_object_key TEXT NOT NULL,
  audio_fingerprint_hash TEXT,
  hls_prefix TEXT,
  waveform_json TEXT,
  ai_provider TEXT NOT NULL
    CHECK (ai_provider IN ('SUNO', 'UDIO', 'AIVA', 'STABLE_AUDIO', 'SOUNDRAW', 'MUBERT', 'CUSTOM_OPEN_MODEL')),
  ai_generated BOOLEAN NOT NULL DEFAULT TRUE,
  ai_disclosure_text TEXT,
  bitrate_kbps INT CHECK (bitrate_kbps IS NULL OR bitrate_kbps > 0),
  status TEXT NOT NULL DEFAULT 'UPLOADED'
    CHECK (status IN ('UPLOADED', 'LICENSE_GATE_PENDING', 'LICENSE_GATE_REJECTED', 'TRANSCODING', 'READY', 'TAKEN_DOWN')),
  rejection_reason_ko TEXT,
  play_count BIGINT NOT NULL DEFAULT 0 CHECK (play_count >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT fk_track_album FOREIGN KEY (album_id) REFERENCES albums (id) ON DELETE SET NULL,
  CONSTRAINT fk_track_partner FOREIGN KEY (partner_id) REFERENCES partner_profiles (id) ON DELETE CASCADE
);
CREATE INDEX idx_tracks_partner ON tracks (partner_id);
CREATE INDEX idx_tracks_status ON tracks (status);
CREATE TRIGGER trg_tracks_updated_at
  BEFORE UPDATE ON tracks FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 9. track_rights_declarations
CREATE TABLE track_rights_declarations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  track_id UUID NOT NULL,
  partner_id UUID NOT NULL,
  declares_ownership BOOLEAN NOT NULL DEFAULT FALSE,
  declares_no_voice_clone BOOLEAN NOT NULL DEFAULT FALSE,
  declares_training_consent BOOLEAN NOT NULL DEFAULT FALSE,
  confirmed_no_unauthorized_sampling BOOLEAN NOT NULL DEFAULT FALSE,
  confirmed_indemnification BOOLEAN NOT NULL DEFAULT FALSE,
  plan_tier_used TEXT,
  provider_track_url TEXT,
  human_contribution_types TEXT[] NOT NULL DEFAULT '{}',
  human_contribution_desc TEXT,
  gate1_policy_passed BOOLEAN NOT NULL DEFAULT TRUE,
  gate2_date_passed BOOLEAN NOT NULL DEFAULT TRUE,
  gate3_ffprobe_passed BOOLEAN NOT NULL DEFAULT FALSE,
  ffprobe_report_json TEXT,
  signed_ip TEXT,
  source_model TEXT,
  subscription_id UUID,
  generated_at TIMESTAMPTZ,
  declared_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT track_rights_track_unique UNIQUE (track_id),
  CONSTRAINT fk_rights_track FOREIGN KEY (track_id) REFERENCES tracks (id) ON DELETE CASCADE,
  CONSTRAINT fk_rights_partner FOREIGN KEY (partner_id) REFERENCES partner_profiles (id) ON DELETE CASCADE,
  CONSTRAINT fk_rights_subscription FOREIGN KEY (subscription_id) REFERENCES partner_ai_subscriptions (id) ON DELETE SET NULL
);

-- 10. transcoding_jobs
CREATE TABLE transcoding_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  track_id UUID NOT NULL,
  status TEXT NOT NULL DEFAULT 'QUEUED'
    CHECK (status IN ('QUEUED', 'RUNNING', 'SUCCEEDED', 'FAILED')),
  input_object_key TEXT NOT NULL,
  output_prefix TEXT,
  bitrate_kbps INT CHECK (bitrate_kbps IS NULL OR bitrate_kbps > 0),
  error TEXT,
  attempts INT NOT NULL DEFAULT 0 CHECK (attempts >= 0),
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT fk_transcode_track FOREIGN KEY (track_id) REFERENCES tracks (id) ON DELETE CASCADE
);
CREATE INDEX idx_transcoding_track ON transcoding_jobs (track_id);
CREATE INDEX idx_transcoding_status ON transcoding_jobs (status);
CREATE TRIGGER trg_transcoding_jobs_updated_at
  BEFORE UPDATE ON transcoding_jobs FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 11. copyright_disputes
CREATE TABLE copyright_disputes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  track_id UUID NOT NULL,
  reporter_user_id UUID,
  reason TEXT NOT NULL
    CHECK (reason IN ('AI_POLICY_VIOLATION', 'VOICE_CLONE', 'UNAUTHORIZED_SAMPLE', 'OTHER')),
  description TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'OPEN'
    CHECK (status IN ('OPEN', 'UNDER_REVIEW', 'UPHELD_TAKEDOWN', 'DISMISSED')),
  strike_applied BOOLEAN NOT NULL DEFAULT FALSE,
  resolved_by UUID,
  resolved_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT fk_dispute_track FOREIGN KEY (track_id) REFERENCES tracks (id) ON DELETE CASCADE,
  CONSTRAINT fk_dispute_reporter FOREIGN KEY (reporter_user_id) REFERENCES users (id) ON DELETE SET NULL,
  CONSTRAINT fk_dispute_resolver FOREIGN KEY (resolved_by) REFERENCES users (id) ON DELETE SET NULL
);
CREATE INDEX idx_disputes_track ON copyright_disputes (track_id);
CREATE INDEX idx_disputes_status ON copyright_disputes (status);
CREATE TRIGGER trg_copyright_disputes_updated_at
  BEFORE UPDATE ON copyright_disputes FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 12. playlists
CREATE TABLE playlists (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_user_id UUID NOT NULL,
  title TEXT NOT NULL,
  description TEXT,
  is_public BOOLEAN NOT NULL DEFAULT FALSE,
  is_official_curation BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT fk_playlist_owner FOREIGN KEY (owner_user_id) REFERENCES users (id) ON DELETE CASCADE
);
CREATE INDEX idx_playlists_owner ON playlists (owner_user_id);
CREATE TRIGGER trg_playlists_updated_at
  BEFORE UPDATE ON playlists FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 13. playlist_tracks
CREATE TABLE playlist_tracks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  playlist_id UUID NOT NULL,
  track_id UUID NOT NULL,
  position INT NOT NULL CHECK (position >= 0),
  added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT playlist_tracks_playlist_track_unique UNIQUE (playlist_id, track_id),
  CONSTRAINT fk_pt_playlist FOREIGN KEY (playlist_id) REFERENCES playlists (id) ON DELETE CASCADE,
  CONSTRAINT fk_pt_track FOREIGN KEY (track_id) REFERENCES tracks (id) ON DELETE CASCADE
);
CREATE INDEX idx_pt_playlist ON playlist_tracks (playlist_id);

-- 14. user_playback_states
CREATE TABLE user_playback_states (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  track_id UUID NOT NULL,
  position_ms INT NOT NULL DEFAULT 0 CHECK (position_ms >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT user_playback_user_track_unique UNIQUE (user_id, track_id),
  CONSTRAINT fk_playback_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  CONSTRAINT fk_playback_track FOREIGN KEY (track_id) REFERENCES tracks (id) ON DELETE CASCADE
);
CREATE TRIGGER trg_user_playback_states_updated_at
  BEFORE UPDATE ON user_playback_states FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 15. streaming_logs (append-only)
CREATE TABLE streaming_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID,
  track_id UUID,
  partner_id UUID,
  played_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  play_ms INT NOT NULL DEFAULT 0 CHECK (play_ms >= 0),
  counted_as_play BOOLEAN NOT NULL DEFAULT FALSE,
  ip TEXT,
  user_agent TEXT,
  hls_token_jti TEXT,
  CONSTRAINT fk_stream_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL,
  CONSTRAINT fk_stream_track FOREIGN KEY (track_id) REFERENCES tracks (id) ON DELETE SET NULL,
  CONSTRAINT fk_stream_partner FOREIGN KEY (partner_id) REFERENCES partner_profiles (id) ON DELETE SET NULL
);
CREATE INDEX idx_streaming_played_at ON streaming_logs (played_at);
CREATE INDEX idx_streaming_track ON streaming_logs (track_id);
CREATE INDEX idx_streaming_partner_played ON streaming_logs (partner_id, played_at);

-- 16. partner_settlements
CREATE TABLE partner_settlements (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  partner_id UUID NOT NULL,
  period_start DATE NOT NULL,
  period_end DATE NOT NULL,
  gross_amount_krw BIGINT NOT NULL DEFAULT 0 CHECK (gross_amount_krw >= 0),
  withholding_krw BIGINT NOT NULL DEFAULT 0 CHECK (withholding_krw >= 0),
  net_amount_krw BIGINT NOT NULL DEFAULT 0 CHECK (net_amount_krw >= 0),
  play_count BIGINT NOT NULL DEFAULT 0 CHECK (play_count >= 0),
  status TEXT NOT NULL DEFAULT 'PENDING'
    CHECK (status IN ('PENDING', 'APPROVED', 'PAID', 'FORFEITED')),
  forfeiture_reason TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT partner_settlements_period_unique UNIQUE (partner_id, period_start, period_end),
  CONSTRAINT fk_settlement_partner FOREIGN KEY (partner_id) REFERENCES partner_profiles (id) ON DELETE CASCADE,
  CONSTRAINT chk_settlement_period CHECK (period_start <= period_end)
);
CREATE TRIGGER trg_partner_settlements_updated_at
  BEFORE UPDATE ON partner_settlements FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 17. user_blocks (3-strike enforcement)
CREATE TABLE user_blocks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  blocked_by UUID,
  reason TEXT NOT NULL,
  strike_number INT NOT NULL CHECK (strike_number BETWEEN 1 AND 3),
  expires_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT fk_block_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  CONSTRAINT fk_block_by FOREIGN KEY (blocked_by) REFERENCES users (id) ON DELETE SET NULL
);
CREATE INDEX idx_blocks_user ON user_blocks (user_id);

-- 18. user_artist_blocks (App Store / Google Play UGC compliance: user blocks partner/artist)
CREATE TABLE user_artist_blocks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  blocker_user_id UUID NOT NULL,
  blocked_partner_id UUID NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT user_artist_blocks_unique UNIQUE (blocker_user_id, blocked_partner_id),
  CONSTRAINT fk_uab_blocker FOREIGN KEY (blocker_user_id) REFERENCES users (id) ON DELETE CASCADE,
  CONSTRAINT fk_uab_partner FOREIGN KEY (blocked_partner_id) REFERENCES partner_profiles (id) ON DELETE CASCADE
);
CREATE INDEX idx_uab_blocker ON user_artist_blocks (blocker_user_id);
