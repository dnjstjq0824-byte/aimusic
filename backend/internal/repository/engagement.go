package repository

import (
	"context"
	"database/sql"
	"time"

	"ai-music-platform/backend/internal/domain"
)

// PlaylistRepo accesses playlists and playlist_tracks.
type PlaylistRepo struct{ db *sql.DB }

const playlistColumns = `id::text, owner_user_id::text, title, description, is_public, is_official_curation, created_at, updated_at`

func scanPlaylist(row interface{ Scan(...any) error }) (*domain.Playlist, error) {
	p := &domain.Playlist{}
	var desc sql.NullString
	err := row.Scan(&p.ID, &p.OwnerUserID, &p.Title, &desc, &p.IsPublic, &p.IsOfficialCuration, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.Description = strPtr(desc)
	return p, nil
}

func (r *PlaylistRepo) Create(ctx context.Context, ownerID, title string, isPublic bool) (*domain.Playlist, error) {
	return r.CreateFull(ctx, ownerID, title, nil, isPublic, false)
}

func (r *PlaylistRepo) CreateFull(ctx context.Context, ownerID, title string, description *string, isPublic, isOfficial bool) (*domain.Playlist, error) {
	return scanPlaylist(r.db.QueryRowContext(ctx, `
		INSERT INTO playlists (owner_user_id, title, description, is_public, is_official_curation)
		VALUES ($1::uuid, $2, $3, $4, $5)
		RETURNING `+playlistColumns,
		ownerID, title, nullStr(description), isPublic, isOfficial))
}

func (r *PlaylistRepo) GetByID(ctx context.Context, id string) (*domain.Playlist, error) {
	p, err := scanPlaylist(r.db.QueryRowContext(ctx, `
		SELECT `+playlistColumns+` FROM playlists WHERE id = $1::uuid`, id))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return p, err
}

// ListByOwner returns a user's playlists, newest first.
func (r *PlaylistRepo) ListByOwner(ctx context.Context, ownerID string) ([]*domain.Playlist, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+playlistColumns+`
		FROM playlists WHERE owner_user_id = $1::uuid ORDER BY created_at DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Playlist
	for rows.Next() {
		p, err := scanPlaylist(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListOfficialCuration returns public official curation playlists.
func (r *PlaylistRepo) ListOfficialCuration(ctx context.Context) ([]*domain.Playlist, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+playlistColumns+`
		FROM playlists WHERE is_official_curation = TRUE ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Playlist
	for rows.Next() {
		p, err := scanPlaylist(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Update renames a playlist or flips visibility.
func (r *PlaylistRepo) Update(ctx context.Context, id, title string, isPublic bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE playlists SET title = $2, is_public = $3 WHERE id = $1::uuid`, id, title, isPublic)
	return err
}

// Delete removes a playlist and its entries via cascade.
func (r *PlaylistRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM playlists WHERE id = $1::uuid`, id)
	return err
}

// AddTrack appends a track at the next position (idempotent per playlist+track).
func (r *PlaylistRepo) AddTrack(ctx context.Context, playlistID, trackID string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO playlist_tracks (playlist_id, track_id, position)
		SELECT $1::uuid, $2::uuid, COALESCE(MAX(position) + 1, 0)
		FROM playlist_tracks WHERE playlist_id = $1::uuid
		ON CONFLICT (playlist_id, track_id) DO NOTHING`, playlistID, trackID)
	return err
}

// RemoveTrack deletes one entry from a playlist.
func (r *PlaylistRepo) RemoveTrack(ctx context.Context, playlistID, trackID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM playlist_tracks WHERE playlist_id = $1::uuid AND track_id = $2::uuid`,
		playlistID, trackID)
	return err
}

// ListTracks returns entries ordered by position.
func (r *PlaylistRepo) ListTracks(ctx context.Context, playlistID string) ([]*domain.PlaylistTrack, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, playlist_id::text, track_id::text, position, added_at
		FROM playlist_tracks WHERE playlist_id = $1::uuid ORDER BY position ASC`, playlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.PlaylistTrack
	for rows.Next() {
		t := &domain.PlaylistTrack{}
		if err := rows.Scan(&t.ID, &t.PlaylistID, &t.TrackID, &t.Position, &t.AddedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Reorder rewrites positions for the given track order in one transaction.
func (r *PlaylistRepo) Reorder(ctx context.Context, playlistID string, trackIDs []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for i, trackID := range trackIDs {
		if _, err := tx.ExecContext(ctx,
			`UPDATE playlist_tracks SET position = $3 WHERE playlist_id = $1::uuid AND track_id = $2::uuid`,
			playlistID, trackID, i); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// PlaybackRepo accesses user_playback_states.
type PlaybackRepo struct{ db *sql.DB }

// UpsertState stores the resume position for a user+track.
func (r *PlaybackRepo) UpsertState(ctx context.Context, userID, trackID string, positionMs int) (*domain.UserPlaybackState, error) {
	s := &domain.UserPlaybackState{}
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO user_playback_states (user_id, track_id, position_ms)
		VALUES ($1::uuid, $2::uuid, $3)
		ON CONFLICT (user_id, track_id) DO UPDATE SET position_ms = EXCLUDED.position_ms
		RETURNING id::text, user_id::text, track_id::text, position_ms, created_at, updated_at`,
		userID, trackID, positionMs,
	).Scan(&s.ID, &s.UserID, &s.TrackID, &s.PositionMs, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// GetState returns the resume position for a user+track.
func (r *PlaybackRepo) GetState(ctx context.Context, userID, trackID string) (*domain.UserPlaybackState, error) {
	s := &domain.UserPlaybackState{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id::text, user_id::text, track_id::text, position_ms, created_at, updated_at
		FROM user_playback_states WHERE user_id = $1::uuid AND track_id = $2::uuid`,
		userID, trackID,
	).Scan(&s.ID, &s.UserID, &s.TrackID, &s.PositionMs, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return s, err
}

// StreamingRepo accesses streaming_logs.
type StreamingRepo struct{ db *sql.DB }

// Insert appends a streaming log row.
func (r *StreamingRepo) Insert(ctx context.Context, l *domain.StreamingLog) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO streaming_logs (user_id, track_id, partner_id, play_ms, counted_as_play, ip, user_agent, hls_token_jti)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8)`,
		nullStr(l.UserID), nullStr(l.TrackID), nullStr(l.PartnerID),
		l.PlayMs, l.CountedAsPlay, nullStr(l.IP), nullStr(l.UserAgent), nullStr(l.HLSTokenJTI))
	return err
}

// CountUserPlaysSince counts a user's logs since a time (fraud window).
func (r *StreamingRepo) CountUserPlaysSince(ctx context.Context, userID string, since time.Time) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM streaming_logs WHERE user_id = $1::uuid AND played_at >= $2`,
		userID, since).Scan(&n)
	return n, err
}

// CountUserTrackPlaysSince counts a user's logs for one track since a time.
func (r *StreamingRepo) CountUserTrackPlaysSince(ctx context.Context, userID, trackID string, since time.Time) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM streaming_logs WHERE user_id = $1::uuid AND track_id = $2::uuid AND played_at >= $3`,
		userID, trackID, since).Scan(&n)
	return n, err
}

// CountPartnerPlaysInWindow counts counted plays for settlement.
func (r *StreamingRepo) CountPartnerPlaysInWindow(ctx context.Context, partnerID string, start, end time.Time) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM streaming_logs
		WHERE partner_id = $1::uuid AND counted_as_play = TRUE AND played_at >= $2 AND played_at < $3`,
		partnerID, start, end).Scan(&n)
	return n, err
}

// SettlementRepo accesses partner_settlements.
type SettlementRepo struct{ db *sql.DB }

func scanSettlement(row interface{ Scan(...any) error }) (*domain.PartnerSettlement, error) {
	s := &domain.PartnerSettlement{}
	var reason sql.NullString
	err := row.Scan(&s.ID, &s.PartnerID, &s.PeriodStart, &s.PeriodEnd, &s.GrossAmountKRW,
		&s.WithholdingKRW, &s.NetAmountKRW, &s.PlayCount, &s.Status, &reason, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	s.ForfeitureReason = strPtr(reason)
	return s, nil
}

const settlementColumns = `id::text, partner_id::text, period_start, period_end, gross_amount_krw,
	withholding_krw, net_amount_krw, play_count, status, forfeiture_reason, created_at, updated_at`

// Upsert writes a settlement row per partner+period.
func (r *SettlementRepo) Upsert(ctx context.Context, s *domain.PartnerSettlement) (*domain.PartnerSettlement, error) {
	return scanSettlement(r.db.QueryRowContext(ctx, `
		INSERT INTO partner_settlements (partner_id, period_start, period_end, gross_amount_krw,
			withholding_krw, net_amount_krw, play_count, status, forfeiture_reason)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (partner_id, period_start, period_end) DO UPDATE SET
			gross_amount_krw = EXCLUDED.gross_amount_krw,
			withholding_krw = EXCLUDED.withholding_krw,
			net_amount_krw = EXCLUDED.net_amount_krw,
			play_count = EXCLUDED.play_count,
			status = EXCLUDED.status,
			forfeiture_reason = EXCLUDED.forfeiture_reason
		RETURNING `+settlementColumns,
		s.PartnerID, s.PeriodStart, s.PeriodEnd, s.GrossAmountKRW,
		s.WithholdingKRW, s.NetAmountKRW, s.PlayCount, s.Status, nullStr(s.ForfeitureReason)))
}

// GetByPartnerPeriod returns one settlement row.
func (r *SettlementRepo) GetByPartnerPeriod(ctx context.Context, partnerID string, start, end time.Time) (*domain.PartnerSettlement, error) {
	s, err := scanSettlement(r.db.QueryRowContext(ctx,
		`SELECT `+settlementColumns+` FROM partner_settlements
		WHERE partner_id = $1::uuid AND period_start = $2 AND period_end = $3`,
		partnerID, start, end))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return s, err
}

// ListByPartner returns a partner's settlements, newest period first.
func (r *SettlementRepo) ListByPartner(ctx context.Context, partnerID string) ([]*domain.PartnerSettlement, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+settlementColumns+`
		FROM partner_settlements WHERE partner_id = $1::uuid ORDER BY period_start DESC`, partnerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.PartnerSettlement
	for rows.Next() {
		s, err := scanSettlement(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// UpdateStatus transitions a settlement with an optional forfeiture reason.
func (r *SettlementRepo) UpdateStatus(ctx context.Context, id, status string, forfeitReason *string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE partner_settlements SET status = $2, forfeiture_reason = $3 WHERE id = $1::uuid`,
		id, status, nullStr(forfeitReason))
	return err
}
