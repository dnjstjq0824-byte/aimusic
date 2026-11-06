package repository

import (
	"context"
	"database/sql"

	"ai-music-platform/backend/internal/domain"
)

// AlbumRepo accesses albums.
type AlbumRepo struct{ db *sql.DB }

func scanAlbum(row interface{ Scan(...any) error }) (*domain.Album, error) {
	a := &domain.Album{}
	var cover sql.NullString
	var released sql.NullTime
	err := row.Scan(&a.ID, &a.PartnerID, &a.Title, &cover, &a.ReleaseType, &a.Status, &released, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	a.CoverObjectKey = strPtr(cover)
	a.ReleasedAt = timePtr(released)
	return a, nil
}

const albumColumns = `id::text, partner_id::text, title, cover_object_key, release_type, status, released_at, created_at, updated_at`

func (r *AlbumRepo) Create(ctx context.Context, partnerID, title string, coverKey *string, releaseType string) (*domain.Album, error) {
	if releaseType == "" {
		releaseType = "SINGLE"
	}
	a, err := scanAlbum(r.db.QueryRowContext(ctx, `
		INSERT INTO albums (partner_id, title, cover_object_key, release_type)
		VALUES ($1::uuid, $2, $3, $4)
		RETURNING `+albumColumns, partnerID, title, nullStr(coverKey), releaseType))
	return a, err
}

func (r *AlbumRepo) GetByID(ctx context.Context, id string) (*domain.Album, error) {
	a, err := scanAlbum(r.db.QueryRowContext(ctx,
		`SELECT `+albumColumns+` FROM albums WHERE id = $1::uuid`, id))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return a, err
}

// ListByPartner returns a partner's albums, newest first.
func (r *AlbumRepo) ListByPartner(ctx context.Context, partnerID string, limit, offset int) ([]*domain.Album, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+albumColumns+`
		FROM albums WHERE partner_id = $1::uuid ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		partnerID, clampLimit(limit, 20, 100), offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Album
	for rows.Next() {
		a, err := scanAlbum(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// UpdateStatus transitions an album.
func (r *AlbumRepo) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE albums SET status = $2 WHERE id = $1::uuid`, id, status)
	return err
}

// TrackRepo accesses tracks.
type TrackRepo struct{ db *sql.DB }

func scanTrack(row interface{ Scan(...any) error }) (*domain.Track, error) {
	t := &domain.Track{}
	var album, genre, modelVer, prompt, lyrics, fp, hls, waveform, disclosure, rejection sql.NullString
	var moodsJoined string
	var duration, bitrate sql.NullInt64
	err := row.Scan(&t.ID, &album, &t.PartnerID, &t.Title,
		&genre, &moodsJoined, &t.AIGenerationType, &modelVer, &prompt, &t.IsPromptPublic, &t.IsInstrumental, &lyrics,
		&duration, &t.OriginalObjectKey, &fp, &hls, &waveform,
		&t.AIProvider, &t.AIGenerated, &disclosure, &bitrate, &t.Status,
		&rejection, &t.PlayCount, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	t.AlbumID = strPtr(album)
	t.Genre = strPtr(genre)
	t.MoodTags = splitKeywords(moodsJoined)
	t.AIModelVersion = strPtr(modelVer)
	t.AIPrompt = strPtr(prompt)
	t.Lyrics = strPtr(lyrics)
	t.DurationMs = intPtr(duration)
	t.AudioFingerprintHash = strPtr(fp)
	t.HLSPrefix = strPtr(hls)
	t.WaveformJSON = strPtr(waveform)
	t.AIDisclosureText = strPtr(disclosure)
	t.BitrateKbps = intPtr(bitrate)
	t.RejectionReasonKo = strPtr(rejection)
	return t, nil
}

const trackColumns = `id::text, album_id::text, partner_id::text, title,
	genre, COALESCE(array_to_string(mood_tags, chr(31)), ''), ai_generation_type,
	ai_model_version, ai_prompt, is_prompt_public, is_instrumental, lyrics,
	duration_ms, original_object_key, audio_fingerprint_hash, hls_prefix, waveform_json,
	ai_provider, ai_generated, ai_disclosure_text,
	bitrate_kbps, status, rejection_reason_ko, play_count, created_at, updated_at`

func (r *TrackRepo) Create(ctx context.Context, t *domain.Track) (*domain.Track, error) {
	if t.Status == "" {
		t.Status = domain.TrackUploaded
	}
	genType := t.AIGenerationType
	if genType != "AI_ASSISTED_HUMAN" {
		genType = "FULL_AI"
	}
	return scanTrack(r.db.QueryRowContext(ctx, `
		INSERT INTO tracks (album_id, partner_id, title, genre, mood_tags, ai_generation_type,
			ai_model_version, ai_prompt, is_prompt_public, is_instrumental, lyrics,
			duration_ms, original_object_key, ai_provider, ai_generated, ai_disclosure_text, bitrate_kbps, status)
		VALUES ($1::uuid, $2::uuid, $3, $4, string_to_array($5, chr(31)), $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16, $17, $18)
		RETURNING `+trackColumns,
		nullStr(t.AlbumID), t.PartnerID, t.Title, nullStr(t.Genre), joinKeywords(t.MoodTags), genType,
		nullStr(t.AIModelVersion), nullStr(t.AIPrompt), t.IsPromptPublic, t.IsInstrumental, nullStr(t.Lyrics),
		nullInt(t.DurationMs), t.OriginalObjectKey, t.AIProvider, t.AIGenerated, nullStr(t.AIDisclosureText), nullInt(t.BitrateKbps), t.Status))
}

func (r *TrackRepo) GetByID(ctx context.Context, id string) (*domain.Track, error) {
	t, err := scanTrack(r.db.QueryRowContext(ctx,
		`SELECT `+trackColumns+` FROM tracks WHERE id = $1::uuid`, id))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return t, err
}

func (r *TrackRepo) listTracks(ctx context.Context, query string, args ...any) ([]*domain.Track, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Track
	for rows.Next() {
		t, err := scanTrack(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ListReady returns READY tracks for public listing.
func (r *TrackRepo) ListReady(ctx context.Context, limit, offset int) ([]*domain.Track, error) {
	return r.listTracks(ctx, `SELECT `+trackColumns+`
		FROM tracks WHERE status = 'READY' ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		clampLimit(limit, 20, 100), offset)
}

// ListByPartner returns a partner's tracks, newest first.
func (r *TrackRepo) ListByPartner(ctx context.Context, partnerID string, limit, offset int) ([]*domain.Track, error) {
	return r.listTracks(ctx, `SELECT `+trackColumns+`
		FROM tracks WHERE partner_id = $1::uuid ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		partnerID, clampLimit(limit, 20, 100), offset)
}

// ListByStatus returns tracks filtered by status.
func (r *TrackRepo) ListByStatus(ctx context.Context, status string, limit, offset int) ([]*domain.Track, error) {
	return r.listTracks(ctx, `SELECT `+trackColumns+`
		FROM tracks WHERE status = $1 ORDER BY created_at ASC LIMIT $2 OFFSET $3`,
		status, clampLimit(limit, 20, 100), offset)
}

// TopByPlays returns READY tracks ordered by play count for curation.
func (r *TrackRepo) TopByPlays(ctx context.Context, limit int) ([]*domain.Track, error) {
	return r.listTracks(ctx, `SELECT `+trackColumns+`
		FROM tracks WHERE status = 'READY' ORDER BY play_count DESC, created_at DESC LIMIT $1`,
		clampLimit(limit, 20, 100))
}

// UpdateStatus transitions a track with an optional Korean rejection reason.
func (r *TrackRepo) UpdateStatus(ctx context.Context, id, status string, rejectionReason *string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tracks SET status = $2, rejection_reason_ko = $3 WHERE id = $1::uuid`,
		id, status, nullStr(rejectionReason))
	return err
}

// SetTranscoded marks a track LICENSE_GATE_PENDING (awaiting Gate 4 admin review)
// with its HLS prefix, probe metadata, waveform JSON, and audio fingerprint.
func (r *TrackRepo) SetTranscoded(ctx context.Context, id, hlsPrefix string, bitrate int, durationMs *int, waveformJSON, fingerprint *string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE tracks SET status = 'LICENSE_GATE_PENDING', hls_prefix = $2, bitrate_kbps = $3,
			duration_ms = COALESCE($4, duration_ms),
			waveform_json = COALESCE($5, waveform_json),
			audio_fingerprint_hash = COALESCE($6, audio_fingerprint_hash),
			rejection_reason_ko = NULL
		WHERE id = $1::uuid`, id, hlsPrefix, bitrate, nullInt(durationMs), nullStr(waveformJSON), nullStr(fingerprint))
	return err
}

// IncrementPlayCount adds one counted play.
func (r *TrackRepo) IncrementPlayCount(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tracks SET play_count = play_count + 1 WHERE id = $1::uuid`, id)
	return err
}

// ExistsByHash reports whether any track's object key embeds the given SHA-256.
// Staging keys embed the client-declared hash, so LIKE reuse avoids new columns.
func (r *TrackRepo) ExistsByHash(ctx context.Context, sha string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM tracks WHERE original_object_key LIKE '%' || $1 || '%')`, sha).Scan(&exists)
	return exists, err
}

// RightsRepo accesses track_rights_declarations.
type RightsRepo struct{ db *sql.DB }

const rightsColumns = `id::text, track_id::text, partner_id::text, declares_ownership,
	declares_no_voice_clone, declares_training_consent,
	confirmed_no_unauthorized_sampling, confirmed_indemnification,
	plan_tier_used, provider_track_url,
	COALESCE(array_to_string(human_contribution_types, chr(31)), ''),
	human_contribution_desc, gate1_policy_passed, gate2_date_passed, gate3_ffprobe_passed,
	ffprobe_report_json, signed_ip, source_model,
	subscription_id::text, generated_at, declared_at`

func scanRights(row interface{ Scan(...any) error }) (*domain.TrackRightsDeclaration, error) {
	out := &domain.TrackRightsDeclaration{}
	var planTier, trackURL, humanTypesJoined, humanDesc, ffprobeRep, signedIP, model, sub sql.NullString
	var generated sql.NullTime
	err := row.Scan(&out.ID, &out.TrackID, &out.PartnerID, &out.DeclaresOwnership,
		&out.DeclaresNoVoiceClone, &out.DeclaresTrainingConsent,
		&out.ConfirmedNoUnauthorizedSampling, &out.ConfirmedIndemnification,
		&planTier, &trackURL, &humanTypesJoined, &humanDesc,
		&out.Gate1PolicyPassed, &out.Gate2DatePassed, &out.Gate3FFprobePassed,
		&ffprobeRep, &signedIP, &model, &sub, &generated, &out.DeclaredAt)
	if err != nil {
		return nil, err
	}
	out.PlanTierUsed = strPtr(planTier)
	out.ProviderTrackURL = strPtr(trackURL)
	if humanTypesJoined.Valid {
		out.HumanContributionTypes = splitKeywords(humanTypesJoined.String)
	}
	out.HumanContributionDesc = strPtr(humanDesc)
	out.FFprobeReportJSON = strPtr(ffprobeRep)
	out.SignedIP = strPtr(signedIP)
	out.SourceModel = strPtr(model)
	out.SubscriptionID = strPtr(sub)
	out.GeneratedAt = timePtr(generated)
	return out, nil
}

// Upsert writes the rights declaration for a track.
func (r *RightsRepo) Upsert(ctx context.Context, d *domain.TrackRightsDeclaration) (*domain.TrackRightsDeclaration, error) {
	return scanRights(r.db.QueryRowContext(ctx, `
		INSERT INTO track_rights_declarations (track_id, partner_id, declares_ownership,
			declares_no_voice_clone, declares_training_consent,
			confirmed_no_unauthorized_sampling, confirmed_indemnification,
			plan_tier_used, provider_track_url, human_contribution_types, human_contribution_desc,
			gate1_policy_passed, gate2_date_passed, signed_ip,
			source_model, subscription_id, generated_at)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, string_to_array($10, chr(31)), $11, $12, $13, $14, $15, $16::uuid, $17)
		ON CONFLICT (track_id) DO UPDATE SET
			declares_ownership = EXCLUDED.declares_ownership,
			declares_no_voice_clone = EXCLUDED.declares_no_voice_clone,
			declares_training_consent = EXCLUDED.declares_training_consent,
			confirmed_no_unauthorized_sampling = EXCLUDED.confirmed_no_unauthorized_sampling,
			confirmed_indemnification = EXCLUDED.confirmed_indemnification,
			plan_tier_used = EXCLUDED.plan_tier_used,
			provider_track_url = EXCLUDED.provider_track_url,
			human_contribution_types = EXCLUDED.human_contribution_types,
			human_contribution_desc = EXCLUDED.human_contribution_desc,
			gate1_policy_passed = EXCLUDED.gate1_policy_passed,
			gate2_date_passed = EXCLUDED.gate2_date_passed,
			signed_ip = EXCLUDED.signed_ip,
			source_model = EXCLUDED.source_model,
			subscription_id = EXCLUDED.subscription_id,
			generated_at = EXCLUDED.generated_at
		RETURNING `+rightsColumns,
		d.TrackID, d.PartnerID, d.DeclaresOwnership, d.DeclaresNoVoiceClone, d.DeclaresTrainingConsent,
		d.ConfirmedNoUnauthorizedSampling, d.ConfirmedIndemnification,
		nullStr(d.PlanTierUsed), nullStr(d.ProviderTrackURL), joinKeywords(d.HumanContributionTypes), nullStr(d.HumanContributionDesc),
		d.Gate1PolicyPassed, d.Gate2DatePassed, nullStr(d.SignedIP),
		nullStr(d.SourceModel), nullStr(d.SubscriptionID), nullTime(d.GeneratedAt),
	))
}

// SetGate3Report records the worker's FFprobe forensic scan result.
func (r *RightsRepo) SetGate3Report(ctx context.Context, trackID string, passed bool, reportJSON string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE track_rights_declarations
		SET gate3_ffprobe_passed = $2, ffprobe_report_json = $3
		WHERE track_id = $1::uuid`, trackID, passed, reportJSON)
	return err
}

// GetByTrack returns the rights declaration for a track.
func (r *RightsRepo) GetByTrack(ctx context.Context, trackID string) (*domain.TrackRightsDeclaration, error) {
	out, err := scanRights(r.db.QueryRowContext(ctx, `
		SELECT `+rightsColumns+`
		FROM track_rights_declarations WHERE track_id = $1::uuid`, trackID))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return out, err
}

// JobRepo accesses transcoding_jobs.
type JobRepo struct{ db *sql.DB }

func scanJob(row interface{ Scan(...any) error }) (*domain.TranscodingJob, error) {
	j := &domain.TranscodingJob{}
	var prefix, errmsg sql.NullString
	var bitrate sql.NullInt64
	var started, finished sql.NullTime
	err := row.Scan(&j.ID, &j.TrackID, &j.Status, &j.InputObjectKey, &prefix,
		&bitrate, &errmsg, &j.Attempts, &started, &finished, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return nil, err
	}
	j.OutputPrefix = strPtr(prefix)
	j.BitrateKbps = intPtr(bitrate)
	j.Error = strPtr(errmsg)
	j.StartedAt = timePtr(started)
	j.FinishedAt = timePtr(finished)
	return j, nil
}

const jobColumns = `id::text, track_id::text, status, input_object_key, output_prefix,
	bitrate_kbps, error, attempts, started_at, finished_at, created_at, updated_at`

func (r *JobRepo) Create(ctx context.Context, trackID, inputKey, outputPrefix string) (*domain.TranscodingJob, error) {
	j, err := scanJob(r.db.QueryRowContext(ctx, `
		INSERT INTO transcoding_jobs (track_id, input_object_key, output_prefix)
		VALUES ($1::uuid, $2, $3)
		RETURNING `+jobColumns, trackID, inputKey, outputPrefix))
	return j, err
}

func (r *JobRepo) GetByID(ctx context.Context, id string) (*domain.TranscodingJob, error) {
	j, err := scanJob(r.db.QueryRowContext(ctx,
		`SELECT `+jobColumns+` FROM transcoding_jobs WHERE id = $1::uuid`, id))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return j, err
}

// MarkRunning transitions a job to RUNNING.
func (r *JobRepo) MarkRunning(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE transcoding_jobs SET status = 'RUNNING', started_at = NOW() WHERE id = $1::uuid`, id)
	return err
}

// MarkSucceeded transitions a job to SUCCEEDED.
func (r *JobRepo) MarkSucceeded(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE transcoding_jobs SET status = 'SUCCEEDED', finished_at = NOW() WHERE id = $1::uuid`, id)
	return err
}

// MarkFailed transitions a job to FAILED and bumps attempts.
func (r *JobRepo) MarkFailed(ctx context.Context, id, errMsg string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE transcoding_jobs SET status = 'FAILED', error = $2,
			attempts = attempts + 1, finished_at = NOW() WHERE id = $1::uuid`, id, errMsg)
	return err
}
