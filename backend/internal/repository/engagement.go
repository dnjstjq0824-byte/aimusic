package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"avibe-platform/backend/internal/domain"
)

// PlaylistRepo accesses playlists and playlist_tracks.
type PlaylistRepo struct{ db *sql.DB }

const playlistColumns = `id::text, owner_user_id::text, title, short_id, slug, description, is_public, is_official_curation, created_at, updated_at`

func scanPlaylist(row interface{ Scan(...any) error }) (*domain.Playlist, error) {
	p := &domain.Playlist{}
	var desc sql.NullString
	err := row.Scan(&p.ID, &p.OwnerUserID, &p.Title, &p.ShortID, &p.Slug, &desc, &p.IsPublic, &p.IsOfficialCuration, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.Description = strPtr(desc)
	return p, nil
}

func (r *PlaylistRepo) Create(ctx context.Context, ownerID, title string, description *string, isPublic bool) (*domain.Playlist, error) {
	return r.CreateFull(ctx, ownerID, title, description, isPublic, false)
}

func (r *PlaylistRepo) CreateFull(ctx context.Context, ownerID, title string, description *string, isPublic, isOfficial bool) (*domain.Playlist, error) {
	shortID, err := domain.NewShortID()
	if err != nil {
		return nil, err
	}
	return scanPlaylist(r.db.QueryRowContext(ctx, `
		INSERT INTO playlists (owner_user_id, title, short_id, slug, description, is_public, is_official_curation)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7)
		RETURNING `+playlistColumns,
		ownerID, title, shortID, domain.Slugify(title, "playlist"), nullStr(description), isPublic, isOfficial))
}

func (r *PlaylistRepo) GetByID(ctx context.Context, id string) (*domain.Playlist, error) {
	p := &domain.Playlist{}
	var desc, owner sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT p.id::text, p.owner_user_id::text, u.display_name, p.title, p.short_id, p.slug, p.description,
			p.is_public, p.is_official_curation, p.created_at, p.updated_at
		FROM playlists p JOIN users u ON u.id = p.owner_user_id
		WHERE p.id = $1::uuid`, id).Scan(
		&p.ID, &p.OwnerUserID, &owner, &p.Title, &p.ShortID, &p.Slug, &desc,
		&p.IsPublic, &p.IsOfficialCuration, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p.Description = strPtr(desc)
	if owner.Valid {
		p.OwnerName = owner.String
	}
	return p, r.attachStats(ctx, []*domain.Playlist{p})
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
	out := []*domain.Playlist{}
	for rows.Next() {
		p, err := scanPlaylist(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, r.attachStats(ctx, out)
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
	out := []*domain.Playlist{}
	for rows.Next() {
		p, err := scanPlaylist(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, r.attachStats(ctx, out)
}

// ListPublic returns discoverable playlists (official first, then public),
// newest first. R6 builds full discovery on top of this.
func (r *PlaylistRepo) ListPublic(ctx context.Context, limit int) ([]*domain.Playlist, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+playlistColumns+`
		FROM playlists WHERE is_official_curation = TRUE OR is_public = TRUE
		ORDER BY is_official_curation DESC, created_at DESC LIMIT $1`, clampLimit(limit, 20, 100))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Playlist{}
	for rows.Next() {
		p, err := scanPlaylist(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, r.attachStats(ctx, out)
}

// ListPublicContaining returns discoverable playlists holding a track.
func (r *PlaylistRepo) ListPublicContaining(ctx context.Context, trackID string) ([]*domain.Playlist, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.id::text, p.owner_user_id::text, p.title, p.short_id, p.slug, p.description, p.is_public, p.is_official_curation, p.created_at, p.updated_at
		FROM playlists p JOIN playlist_tracks pt ON pt.playlist_id = p.id
		WHERE pt.track_id = $1::uuid AND (p.is_official_curation = TRUE OR p.is_public = TRUE)
		ORDER BY p.created_at DESC`, trackID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Playlist{}
	for rows.Next() {
		p, err := scanPlaylist(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, r.attachStats(ctx, out)
}

// SavedPlaylists returns playlists the user saved, most recently saved first.
func (r *PlaylistRepo) SavedPlaylists(ctx context.Context, userID string, limit int) ([]*domain.Playlist, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT playlists.id::text, playlists.owner_user_id::text, playlists.title, playlists.short_id, playlists.slug, playlists.description,
			playlists.is_public, playlists.is_official_curation, playlists.created_at, playlists.updated_at
		FROM playlist_saves s JOIN playlists ON playlists.id = s.playlist_id
		WHERE s.user_id = $1::uuid ORDER BY s.created_at DESC LIMIT $2`,
		userID, clampLimit(limit, 20, 100))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Playlist{}
	for rows.Next() {
		p, err := scanPlaylist(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, r.attachStats(ctx, out)
}

// Update edits title, visibility and (when description is non-nil) the
// description; an empty description clears it.
func (r *PlaylistRepo) Update(ctx context.Context, id, title string, description *string, isPublic bool) error {
	setDesc := description != nil
	var desc any
	if setDesc && *description != "" {
		desc = *description
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE playlists SET title = $2, is_public = $3,
			description = CASE WHEN $4 THEN $5::text ELSE description END
		WHERE id = $1::uuid`, id, title, isPublic, setDesc, desc)
	return err
}

// Delete removes a playlist and its entries via cascade.
func (r *PlaylistRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM playlists WHERE id = $1::uuid`, id)
	return err
}

// AddTrack appends a track at the next position. added=false means the
// track was already in the playlist (UNIQUE(playlist_id, track_id)).
func (r *PlaylistRepo) AddTrack(ctx context.Context, playlistID, trackID string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO playlist_tracks (playlist_id, track_id, position)
		SELECT $1::uuid, $2::uuid, COALESCE(MAX(position) + 1, 0)
		FROM playlist_tracks WHERE playlist_id = $1::uuid
		ON CONFLICT (playlist_id, track_id) DO NOTHING`, playlistID, trackID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

// CountTracks returns the number of entries (any track status).
func (r *PlaylistRepo) CountTracks(ctx context.Context, playlistID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM playlist_tracks WHERE playlist_id = $1::uuid`, playlistID).Scan(&n)
	return n, err
}

// ContainingTrack returns the IDs of ownerID's playlists that hold trackID.
func (r *PlaylistRepo) ContainingTrack(ctx context.Context, ownerID, trackID string) (map[string]bool, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT pt.playlist_id::text FROM playlist_tracks pt
		JOIN playlists p ON p.id = pt.playlist_id
		WHERE p.owner_user_id = $1::uuid AND pt.track_id = $2::uuid`, ownerID, trackID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

// attachStats fills TrackCount/TotalDurationMs/CoverSeeds for a page of
// playlists in one grouped query (READY tracks only: what listeners see).
func (r *PlaylistRepo) attachStats(ctx context.Context, list []*domain.Playlist) error {
	if len(list) == 0 {
		return nil
	}
	ids := make([]string, 0, len(list))
	byID := make(map[string]*domain.Playlist, len(list))
	for _, p := range list {
		p.CoverSeeds = []string{}
		ids = append(ids, p.ID)
		byID[p.ID] = p
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT pt.playlist_id::text, COUNT(*), COALESCE(SUM(t.duration_ms), 0),
			COALESCE(array_to_string((ARRAY_AGG(t.id::text ORDER BY pt.position, pt.added_at))[1:4], ','), '')
		FROM playlist_tracks pt JOIN tracks t ON t.id = pt.track_id
		WHERE pt.playlist_id = ANY(string_to_array($1, ',')::uuid[]) AND t.status = 'READY'
		GROUP BY pt.playlist_id`, strings.Join(ids, ","))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, seeds string
		var count int
		var total int64
		if err := rows.Scan(&id, &count, &total, &seeds); err != nil {
			return err
		}
		if p := byID[id]; p != nil {
			p.TrackCount = count
			p.TotalDurationMs = total
			if seeds != "" {
				p.CoverSeeds = strings.Split(seeds, ",")
			}
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return r.attachOwnerNames(ctx, ids, byID)
}

// attachOwnerNames fills OwnerName for list rows (list queries share the
// join-less playlistColumns with INSERT ... RETURNING).
func (r *PlaylistRepo) attachOwnerNames(ctx context.Context, ids []string, byID map[string]*domain.Playlist) error {
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.id::text, u.display_name FROM playlists p JOIN users u ON u.id = p.owner_user_id
		WHERE p.id = ANY(string_to_array($1, ',')::uuid[])`, strings.Join(ids, ","))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return err
		}
		if p := byID[id]; p != nil && p.OwnerName == "" && name.Valid {
			p.OwnerName = name.String
		}
	}
	return rows.Err()
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
		FROM playlist_tracks WHERE playlist_id = $1::uuid ORDER BY position ASC, added_at ASC`, playlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.PlaylistTrack{}
	for rows.Next() {
		t := &domain.PlaylistTrack{}
		if err := rows.Scan(&t.ID, &t.PlaylistID, &t.TrackID, &t.Position, &t.AddedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ListTracksWithMeta returns ordered entries with the joined public-view
// track in one query. readyOnly hides non-READY tracks (non-owner viewers).
func (r *PlaylistRepo) ListTracksWithMeta(ctx context.Context, playlistID string, readyOnly bool) ([]*domain.PlaylistTrack, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT pt.id::text, pt.playlist_id::text, pt.track_id::text, pt.position, pt.added_at, `+trackCreatorSelect+`
		FROM playlist_tracks pt JOIN tracks ON tracks.id = pt.track_id`+trackCreatorJoin+`
		WHERE pt.playlist_id = $1::uuid AND (NOT $2 OR tracks.status = 'READY')
		ORDER BY pt.position ASC, pt.added_at ASC`, playlistID, readyOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.PlaylistTrack{}
	for rows.Next() {
		e := &domain.PlaylistTrack{}
		t, err := scanTrackWithCreator(prefixScanner{row: rows, prefix: []any{&e.ID, &e.PlaylistID, &e.TrackID, &e.Position, &e.AddedAt}})
		if err != nil {
			return nil, err
		}
		e.Track = t.PublicView()
		out = append(out, e)
	}
	return out, rows.Err()
}

// prefixScanner scans leading extra columns before a shared column set.
type prefixScanner struct {
	row    interface{ Scan(...any) error }
	prefix []any
}

func (p prefixScanner) Scan(dest ...any) error {
	all := make([]any, 0, len(p.prefix)+len(dest))
	all = append(all, p.prefix...)
	return p.row.Scan(append(all, dest...)...)
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

// EngagementRepo accesses track likes, playlist saves, and follows.
// Writes are idempotent (re-like/re-save return added=false, removing a
// missing row returns removed=false) so UI toggles never 4xx/5xx.
type EngagementRepo struct{ db *sql.DB }

func (r *EngagementRepo) LikeTrack(ctx context.Context, userID, trackID string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO track_likes (user_id, track_id) VALUES ($1::uuid, $2::uuid) ON CONFLICT DO NOTHING`,
		userID, trackID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

func (r *EngagementRepo) UnlikeTrack(ctx context.Context, userID, trackID string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM track_likes WHERE user_id = $1::uuid AND track_id = $2::uuid`, userID, trackID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

func (r *EngagementRepo) IsLiked(ctx context.Context, userID, trackID string) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM track_likes WHERE user_id = $1::uuid AND track_id = $2::uuid`,
		userID, trackID).Scan(&n)
	return n == 1, err
}

func (r *EngagementRepo) SavePlaylist(ctx context.Context, userID, playlistID string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO playlist_saves (user_id, playlist_id) VALUES ($1::uuid, $2::uuid) ON CONFLICT DO NOTHING`,
		userID, playlistID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

func (r *EngagementRepo) UnsavePlaylist(ctx context.Context, userID, playlistID string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM playlist_saves WHERE user_id = $1::uuid AND playlist_id = $2::uuid`, userID, playlistID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

func (r *EngagementRepo) IsSaved(ctx context.Context, userID, playlistID string) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM playlist_saves WHERE user_id = $1::uuid AND playlist_id = $2::uuid`,
		userID, playlistID).Scan(&n)
	return n == 1, err
}

func (r *EngagementRepo) Follow(ctx context.Context, userID, targetType, targetID string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO follows (user_id, target_type, target_id) VALUES ($1::uuid, $2, $3::uuid) ON CONFLICT DO NOTHING`,
		userID, targetType, targetID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

func (r *EngagementRepo) Unfollow(ctx context.Context, userID, targetType, targetID string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM follows WHERE user_id = $1::uuid AND target_type = $2 AND target_id = $3::uuid`,
		userID, targetType, targetID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

// ListFollows returns resolved follows (creator display names, playlist
// titles); rows whose target vanished are skipped. Empty targetType lists all.
func (r *EngagementRepo) ListFollows(ctx context.Context, userID, targetType string) ([]*domain.Follow, error) {
	query := `
		SELECT f.target_type, f.target_id::text,
			COALESCE(u.display_name, p.title, ''),
			f.created_at, u.handle
		FROM follows f
		LEFT JOIN partner_profiles pp ON f.target_type = 'creator' AND pp.id = f.target_id
		LEFT JOIN users u ON u.id = pp.user_id
		LEFT JOIN playlists p ON f.target_type = 'playlist' AND p.id = f.target_id
		WHERE f.user_id = $1::uuid`
	args := []any{userID}
	if targetType != "" {
		query += ` AND f.target_type = $2`
		args = append(args, targetType)
	}
	query += ` ORDER BY f.created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Follow{}
	for rows.Next() {
		f := &domain.Follow{}
		var handle sql.NullString
		if err := rows.Scan(&f.TargetType, &f.TargetID, &f.DisplayName, &f.CreatedAt, &handle); err != nil {
			return nil, err
		}
		f.Handle = strPtr(handle)
		if f.DisplayName == "" {
			continue
		}
		out = append(out, f)
	}
	return out, rows.Err()
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

// ListStatesByUser returns every resume position for a user (data export).
func (r *PlaybackRepo) ListStatesByUser(ctx context.Context, userID string) ([]*domain.UserPlaybackState, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, user_id::text, track_id::text, position_ms, created_at, updated_at
		FROM user_playback_states WHERE user_id = $1::uuid ORDER BY updated_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.UserPlaybackState{}
	for rows.Next() {
		s := &domain.UserPlaybackState{}
		if err := rows.Scan(&s.ID, &s.UserID, &s.TrackID, &s.PositionMs, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// StreamingRepo accesses streaming_logs.
type StreamingRepo struct{ db *sql.DB }

// ListByUser returns a user's streaming log rows, newest first (data export).
func (r *StreamingRepo) ListByUser(ctx context.Context, userID string, limit int) ([]*domain.StreamingLog, error) {
	if limit <= 0 || limit > 5000 {
		limit = 1000
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, user_id::text, track_id::text, partner_id::text, played_at,
			play_ms, counted_as_play, ip, user_agent, hls_token_jti
		FROM streaming_logs WHERE user_id = $1::uuid ORDER BY played_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.StreamingLog{}
	for rows.Next() {
		l := &domain.StreamingLog{}
		var uid, tid, pid, ip, ua, jti sql.NullString
		if err := rows.Scan(&l.ID, &uid, &tid, &pid, &l.PlayedAt,
			&l.PlayMs, &l.CountedAsPlay, &ip, &ua, &jti); err != nil {
			return nil, err
		}
		l.UserID, l.TrackID, l.PartnerID = strPtr(uid), strPtr(tid), strPtr(pid)
		l.IP, l.UserAgent, l.HLSTokenJTI = strPtr(ip), strPtr(ua), strPtr(jti)
		out = append(out, l)
	}
	return out, rows.Err()
}

// Insert appends a streaming log row.
func (r *StreamingRepo) Insert(ctx context.Context, l *domain.StreamingLog) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO streaming_logs (user_id, track_id, partner_id, play_ms, counted_as_play, ip, user_agent, hls_token_jti)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8)`,
		nullStr(l.UserID), nullStr(l.TrackID), nullStr(l.PartnerID),
		l.PlayMs, l.CountedAsPlay, nullStr(l.IP), nullStr(l.UserAgent), nullStr(l.HLSTokenJTI))
	return err
}

// CountUserPlaysSince counts a user's COUNTED plays since a time (fraud window).
// Raw heartbeat rows are excluded: clients report every ~10s, so counting rows
// would flag every normal listener as fraud after a few minutes.
func (r *StreamingRepo) CountUserPlaysSince(ctx context.Context, userID string, since time.Time) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM streaming_logs WHERE user_id = $1::uuid AND counted_as_play = TRUE AND played_at >= $2`,
		userID, since).Scan(&n)
	return n, err
}

// CountUserTrackPlaysSince counts a user's COUNTED plays for one track since a time.
func (r *StreamingRepo) CountUserTrackPlaysSince(ctx context.Context, userID, trackID string, since time.Time) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM streaming_logs WHERE user_id = $1::uuid AND track_id = $2::uuid AND counted_as_play = TRUE AND played_at >= $3`,
		userID, trackID, since).Scan(&n)
	return n, err
}

// SumPlayMsSinceLastCounted returns the uncounted play_ms accumulated for a
// user+track since their last counted play (0 when none). Heartbeat clients
// send small deltas (~10s), so counting requires accumulation: a single delta
// almost never reaches the 30s valid-stream threshold on its own.
func (r *StreamingRepo) SumPlayMsSinceLastCounted(ctx context.Context, userID, trackID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(play_ms), 0) FROM streaming_logs
		WHERE user_id = $1::uuid AND track_id = $2::uuid AND played_at > COALESCE(
			(SELECT MAX(played_at) FROM streaming_logs
				WHERE user_id = $1::uuid AND track_id = $2::uuid AND counted_as_play = TRUE),
			'1970-01-01'::timestamptz)`,
		userID, trackID).Scan(&n)
	return n, err
}

// LastHeartbeatAt returns the newest heartbeat receive time for a user+track.
// ok=false when the pair never reported (first heartbeat has no elapsed
// basis). played_at is server receive time, so elapsed is client-clock-free.
func (r *StreamingRepo) LastHeartbeatAt(ctx context.Context, userID, trackID string) (at time.Time, ok bool, err error) {
	var last sql.NullTime
	err = r.db.QueryRowContext(ctx, `
		SELECT MAX(played_at) FROM streaming_logs
		WHERE user_id = $1::uuid AND track_id = $2::uuid`,
		userID, trackID).Scan(&last)
	if err != nil || !last.Valid {
		return time.Time{}, false, err
	}
	return last.Time, true, nil
}

// PlaybackGrantTTL bounds heartbeat authorization per stream grant. It
// deliberately outlives the signed URL TTL: long mixes keep sending
// heartbeats after the playlist URL expires, and the jti check proves issuance, not
// freshness. Claimed time stays bounded by the per-heartbeat elapsed cap
// plus the hourly fraud windows.
const PlaybackGrantTTL = 24 * time.Hour

// PlaybackGrantStore records stream-grant ids so heartbeats prove they
// follow a real grant (B-06). Values bind user and track ids with a unit separator.
type PlaybackGrantStore struct{ rdb *redis.Client }

func playbackGrantKey(jti string) string { return "playback:grant:" + jti }

// RecordGrant stores a grant issued to userID for trackID.
func (g *PlaybackGrantStore) RecordGrant(ctx context.Context, jti, userID, trackID string) error {
	if g == nil || g.rdb == nil {
		return errors.New("playback grant store is not configured")
	}
	return g.rdb.Set(ctx, playbackGrantKey(jti), userID+"\x1f"+trackID, PlaybackGrantTTL).Err()
}

// VerifyGrant reports whether jti was issued to userID for trackID.
func (g *PlaybackGrantStore) VerifyGrant(ctx context.Context, jti, userID, trackID string) (bool, error) {
	if g == nil || g.rdb == nil {
		return false, errors.New("playback grant store is not configured")
	}
	got, err := g.rdb.Get(ctx, playbackGrantKey(jti)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}
	return got == userID+"\x1f"+trackID, nil
}

// CountCountedSince counts counted plays platform-wide since a time.
func (r *StreamingRepo) CountCountedSince(ctx context.Context, since time.Time) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM streaming_logs WHERE counted_as_play = TRUE AND played_at >= $1`,
		since).Scan(&n)
	return n, err
}

// CountCountedTotal counts all counted plays platform-wide.
func (r *StreamingRepo) CountCountedTotal(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM streaming_logs WHERE counted_as_play = TRUE`).Scan(&n)
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
	var reason, payoutRef, holdReason, heldBy sql.NullString
	var paidAt, heldAt sql.NullTime
	err := row.Scan(&s.ID, &s.PartnerID, &s.PeriodStart, &s.PeriodEnd, &s.GrossAmountKRW,
		&s.WithholdingKRW, &s.NetAmountKRW, &s.PlayCount, &s.StorePoolKRW, &s.DirectKRW,
		&s.StorePlaySecs, &s.TotalStorePlaySecs, &s.Status, &holdReason, &heldAt, &heldBy,
		&reason, &s.IncomeType, &paidAt, &payoutRef, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	s.HoldReason = strPtr(holdReason)
	s.HeldAt = timePtr(heldAt)
	s.HeldBy = strPtr(heldBy)
	s.ForfeitureReason = strPtr(reason)
	s.PaidAt = timePtr(paidAt)
	s.PayoutReference = strPtr(payoutRef)
	return s, nil
}

const settlementColumns = `id::text, partner_id::text, period_start, period_end, gross_amount_krw,
	withholding_krw, net_amount_krw, play_count, store_pool_krw, direct_krw,
	store_play_secs, total_store_play_secs, status, hold_reason, held_at, held_by::text,
	forfeiture_reason, income_type, paid_at, payout_reference, created_at, updated_at`

// Upsert writes a settlement row per partner+period. Recalculation only
// touches PENDING rows: the status predicate makes a concurrent approve win
// the race (the decided row is left untouched and sql.ErrNoRows surfaces,
// like Hold/Release/MarkPaid).
func (r *SettlementRepo) Upsert(ctx context.Context, s *domain.PartnerSettlement) (*domain.PartnerSettlement, error) {
	return scanSettlement(r.db.QueryRowContext(ctx, `
		INSERT INTO partner_settlements (partner_id, period_start, period_end, gross_amount_krw,
			withholding_krw, net_amount_krw, play_count, store_pool_krw, direct_krw,
			store_play_secs, total_store_play_secs, status, forfeiture_reason, income_type)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (partner_id, period_start, period_end) DO UPDATE SET
			gross_amount_krw = EXCLUDED.gross_amount_krw,
			withholding_krw = EXCLUDED.withholding_krw,
			net_amount_krw = EXCLUDED.net_amount_krw,
			play_count = EXCLUDED.play_count,
			store_pool_krw = EXCLUDED.store_pool_krw,
			direct_krw = EXCLUDED.direct_krw,
			store_play_secs = EXCLUDED.store_play_secs,
			total_store_play_secs = EXCLUDED.total_store_play_secs,
			status = EXCLUDED.status,
			forfeiture_reason = EXCLUDED.forfeiture_reason,
			income_type = EXCLUDED.income_type
		WHERE partner_settlements.status = 'PENDING'
		RETURNING `+settlementColumns,
		s.PartnerID, s.PeriodStart, s.PeriodEnd, s.GrossAmountKRW,
		s.WithholdingKRW, s.NetAmountKRW, s.PlayCount, s.StorePoolKRW, s.DirectKRW,
		s.StorePlaySecs, s.TotalStorePlaySecs, s.Status, nullStr(s.ForfeitureReason), s.IncomeType))
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
	out := []*domain.PartnerSettlement{}
	for rows.Next() {
		s, err := scanSettlement(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Approve transitions a PENDING settlement to APPROVED (re-approving an
// APPROVED row stays idempotent). The status predicate makes a concurrent
// MarkPaid win the race: a PAID row is never moved back to APPROVED
// (sql.ErrNoRows surfaces, like Hold/Release/MarkPaid).
func (r *SettlementRepo) Approve(ctx context.Context, id string) error {
	tag, err := r.db.ExecContext(ctx,
		`UPDATE partner_settlements SET status = 'APPROVED'
		WHERE id = $1::uuid AND status IN ('PENDING', 'APPROVED')`,
		id)
	if err != nil {
		return err
	}
	if n, err := tag.RowsAffected(); err != nil || n == 0 {
		if err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	return nil
}

// Hold suspends a PENDING settlement exactly once, recording reason + actor.
// The status predicate makes double-holds safe (sql.ErrNoRows).
func (r *SettlementRepo) Hold(ctx context.Context, id, reason, actorID string) error {
	tag, err := r.db.ExecContext(ctx,
		`UPDATE partner_settlements
		SET status = 'HELD', hold_reason = $2, held_at = NOW(), held_by = $3::uuid
		WHERE id = $1::uuid AND status = 'PENDING'`,
		id, reason, actorID)
	if err != nil {
		return err
	}
	if n, err := tag.RowsAffected(); err != nil || n == 0 {
		if err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	return nil
}

// Release returns a HELD settlement to PENDING. Hold history (reason, actor,
// time) stays on the row as the audit trail.
func (r *SettlementRepo) Release(ctx context.Context, id string) error {
	tag, err := r.db.ExecContext(ctx,
		`UPDATE partner_settlements SET status = 'PENDING'
		WHERE id = $1::uuid AND status = 'HELD'`, id)
	if err != nil {
		return err
	}
	if n, err := tag.RowsAffected(); err != nil || n == 0 {
		if err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	return nil
}

// MarkPaid transitions APPROVED -> PAID exactly once and records the payout
// reference (bank transfer slip). The status predicate makes double-pays safe.
func (r *SettlementRepo) MarkPaid(ctx context.Context, partnerID string, start, end time.Time, reference string) (*domain.PartnerSettlement, error) {
	s, err := scanSettlement(r.db.QueryRowContext(ctx, `
		UPDATE partner_settlements
		SET status = 'PAID', paid_at = NOW(), payout_reference = NULLIF($4, '')
		WHERE partner_id = $1::uuid AND period_start = $2 AND period_end = $3
		  AND status = 'APPROVED'
		RETURNING `+settlementColumns, partnerID, start, end, reference))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return s, err
}

// PendingTotals aggregates PENDING settlements for the admin overview.
func (r *SettlementRepo) PendingTotals(ctx context.Context) (count, netKRW int64, err error) {
	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(net_amount_krw), 0)
		FROM partner_settlements WHERE status = 'PENDING'`).Scan(&count, &netKRW)
	return count, netKRW, err
}
