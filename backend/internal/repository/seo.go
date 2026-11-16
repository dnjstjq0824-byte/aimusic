package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"avibe-platform/backend/internal/domain"
)

// SEORepo serves the S1/S3/S4/S5 read paths: slug resolution, sitemap
// cursors, landing counts, creator profiles, analytics events, IndexNow log.
type SEORepo struct{ db DBTX }

// TrackByShortID returns a READY-or-owned track by its short ID.
func (r *SEORepo) TrackByShortID(ctx context.Context, shortID string) (*domain.Track, error) {
	t, err := scanTrackWithCreator(r.db.QueryRowContext(ctx,
		`SELECT `+trackCreatorSelect+` FROM tracks`+trackCreatorJoin+` WHERE tracks.short_id = $1`, shortID))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return t, err
}

// TrackByParam resolves a /tracks/:param value (uuid, short ID, slug-shortID).
// It returns the track, its canonical path, and whether the param differs
// from canonical (caller issues the 301).
func (r *SEORepo) TrackByParam(ctx context.Context, param string) (*domain.Track, string, bool, error) {
	_, shortID, kind := domain.SplitSlugID(param)
	var t *domain.Track
	var err error
	switch kind {
	case domain.SlugKindShortID, domain.SlugKindSlugShort:
		t, err = r.TrackByShortID(ctx, shortID)
	case domain.SlugKindUUID:
		t, err = scanTrackWithCreator(r.db.QueryRowContext(ctx,
			`SELECT `+trackCreatorSelect+` FROM tracks`+trackCreatorJoin+` WHERE tracks.id = $1::uuid`, param))
		if err == sql.ErrNoRows {
			return nil, "", false, domain.ErrNotFound
		}
	default:
		return nil, "", false, domain.ErrNotFound
	}
	if err != nil {
		return nil, "", false, err
	}
	canonical := domain.CanonicalTrackPath(t.Slug, t.ShortID)
	return t, canonical, "/tracks/"+param != canonical, nil
}

// PlaylistByShortID returns a playlist by its short ID (with owner name).
func (r *SEORepo) PlaylistByShortID(ctx context.Context, shortID string) (*domain.Playlist, error) {
	p := &domain.Playlist{}
	var desc, owner sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT p.id::text, p.owner_user_id::text, u.display_name, p.title, p.short_id, p.slug, p.description,
			p.is_public, p.is_official_curation, p.created_at, p.updated_at
		FROM playlists p JOIN users u ON u.id = p.owner_user_id
		WHERE p.short_id = $1`, shortID).Scan(
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
	return p, nil
}

// PlaylistByParam resolves a /playlists/:param value like TrackByParam.
func (r *SEORepo) PlaylistByParam(ctx context.Context, param string) (*domain.Playlist, string, bool, error) {
	_, shortID, kind := domain.SplitSlugID(param)
	var p *domain.Playlist
	var err error
	switch kind {
	case domain.SlugKindShortID, domain.SlugKindSlugShort:
		p, err = r.PlaylistByShortID(ctx, shortID)
	case domain.SlugKindUUID:
		p = &domain.Playlist{}
		var desc, owner sql.NullString
		err = r.db.QueryRowContext(ctx, `
			SELECT p.id::text, p.owner_user_id::text, u.display_name, p.title, p.short_id, p.slug, p.description,
				p.is_public, p.is_official_curation, p.created_at, p.updated_at
			FROM playlists p JOIN users u ON u.id = p.owner_user_id
			WHERE p.id = $1::uuid`, param).Scan(
			&p.ID, &p.OwnerUserID, &owner, &p.Title, &p.ShortID, &p.Slug, &desc,
			&p.IsPublic, &p.IsOfficialCuration, &p.CreatedAt, &p.UpdatedAt)
		if err == sql.ErrNoRows {
			return nil, "", false, domain.ErrNotFound
		}
		if err == nil {
			p.Description = strPtr(desc)
			if owner.Valid {
				p.OwnerName = owner.String
			}
		}
	default:
		return nil, "", false, domain.ErrNotFound
	}
	if err != nil {
		return nil, "", false, err
	}
	canonical := domain.CanonicalPlaylistPath(p.Slug, p.ShortID)
	return p, canonical, "/playlists/"+param != canonical, nil
}

// CreatorByHandle returns the public creator profile for a handle.
func (r *SEORepo) CreatorByHandle(ctx context.Context, handle string) (*domain.CreatorProfile, error) {
	out := &domain.CreatorProfile{}
	var plays sql.NullInt64
	err := r.db.QueryRowContext(ctx, `
		SELECT u.handle, u.display_name, u.id::text, pp.id::text,
			COUNT(t.id) FILTER (WHERE t.status = 'READY'),
			COALESCE(SUM(t.play_count) FILTER (WHERE t.status = 'READY'), 0)
		FROM users u
		JOIN partner_profiles pp ON pp.user_id = u.id
		LEFT JOIN tracks t ON t.partner_id = pp.id
		WHERE u.handle = $1 AND u.is_active = TRUE
		GROUP BY u.handle, u.display_name, u.id, pp.id`, handle).Scan(
		&out.Handle, &out.DisplayName, &out.UserID, &out.PartnerID, &out.TrackCount, &plays)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	out.TotalPlays = plays.Int64
	return out, nil
}

// ClaimHandle sets a user's creator handle (validated + unique).
func (r *SEORepo) ClaimHandle(ctx context.Context, userID, handle string) error {
	if err := domain.ValidateHandle(handle); err != nil {
		return err
	}
	var owner string
	err := r.db.QueryRowContext(ctx, `SELECT id::text FROM users WHERE handle = $1`, handle).Scan(&owner)
	switch {
	case err == nil:
		if owner != userID {
			return domain.ErrHandleTaken
		}
		return nil
	case !errors.Is(err, sql.ErrNoRows):
		return err
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE users SET handle = $2, updated_at = NOW() WHERE id = $1::uuid`, userID, handle); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrHandleTaken
		}
		return err
	}
	return nil
}

// SitemapTracks returns READY tracks ordered by (updated_at, id) after cursor.
// Cursor format: "<rfc3339-nano>|<uuid>", empty for the first page.
func (r *SEORepo) SitemapTracks(ctx context.Context, cursor string, limit int) ([]domain.SitemapItem, string, error) {
	since, afterID := splitCursor(cursor)
	rows, err := r.db.QueryContext(ctx, `
		SELECT slug || '-' || short_id, updated_at, id::text FROM tracks
		WHERE status = 'READY' AND (updated_at, id::text) > ($1::timestamptz, $2)
		ORDER BY updated_at ASC, id::text ASC LIMIT $3`,
		since, afterID, clampLimit(limit, 100, 2000))
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	return scanSitemap(rows, "/tracks/")
}

// SitemapPlaylists returns public/official playlists ordered by (updated_at, id).
func (r *SEORepo) SitemapPlaylists(ctx context.Context, cursor string, limit int) ([]domain.SitemapItem, string, error) {
	since, afterID := splitCursor(cursor)
	rows, err := r.db.QueryContext(ctx, `
		SELECT slug || '-' || short_id, updated_at, id::text FROM playlists
		WHERE (is_public = TRUE OR is_official_curation = TRUE)
		  AND (updated_at, id::text) > ($1::timestamptz, $2)
		ORDER BY updated_at ASC, id::text ASC LIMIT $3`,
		since, afterID, clampLimit(limit, 100, 2000))
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	return scanSitemap(rows, "/playlists/")
}

// SitemapCreators returns active partner handles ordered by user updated_at.
func (r *SEORepo) SitemapCreators(ctx context.Context, cursor string, limit int) ([]domain.SitemapItem, string, error) {
	since, afterID := splitCursor(cursor)
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.handle, u.updated_at, u.id::text FROM users u
		JOIN partner_profiles pp ON pp.user_id = u.id
		WHERE u.handle IS NOT NULL AND u.is_active = TRUE
		  AND (u.updated_at, u.id::text) > ($1::timestamptz, $2)
		ORDER BY u.updated_at ASC, u.id::text ASC LIMIT $3`,
		since, afterID, clampLimit(limit, 100, 2000))
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	return scanSitemap(rows, "/creators/")
}

func splitCursor(cursor string) (time.Time, string) {
	since := time.Unix(0, 0).UTC()
	afterID := ""
	if cursor == "" {
		return since, afterID
	}
	parts := splitOnce(cursor, "|")
	if t, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
		since = t
	}
	if len(parts) == 2 {
		afterID = parts[1]
	}
	return since, afterID
}

func splitOnce(s, sep string) []string {
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			return []string{s[:i], s[i+len(sep):]}
		}
	}
	return []string{s}
}

func scanSitemap(rows *sql.Rows, prefix string) ([]domain.SitemapItem, string, error) {
	items := []domain.SitemapItem{}
	next := ""
	for rows.Next() {
		var slug, id string
		var updated time.Time
		if err := rows.Scan(&slug, &updated, &id); err != nil {
			return nil, "", err
		}
		items = append(items, domain.SitemapItem{
			Loc:     prefix + slug,
			Lastmod: updated.UTC().Format(time.RFC3339),
		})
		next = updated.UTC().Format(time.RFC3339Nano) + "|" + id
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	return items, next, nil
}

// LandingCount counts READY tracks matching one landing facet (S4 12-track gate).
// Exactly one of mood/genre/useCase should be set; useCase shares the mood pool.
func (r *SEORepo) LandingCount(ctx context.Context, mood, genre, useCase string) (int64, error) {
	var n int64
	query := `SELECT COUNT(*) FROM tracks WHERE status = 'READY'`
	args := []any{}
	switch {
	case mood != "":
		query += ` AND $1 = ANY(mood_tags)`
		args = append(args, mood)
	case genre != "":
		query += ` AND genre = $1`
		args = append(args, genre)
	case useCase != "":
		query += ` AND $1 = ANY(mood_tags)`
		args = append(args, useCase)
	}
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// InsertEvents stores validated analytics events (S5, append-only).
func (r *SEORepo) InsertEvents(ctx context.Context, events []domain.AnalyticsEvent) error {
	if len(events) == 0 {
		return nil
	}
	for _, e := range events {
		props, err := json.Marshal(e.Props)
		if err != nil {
			return err
		}
		if _, err := r.db.ExecContext(ctx, `
			INSERT INTO analytics_events (name, locale, path, session_hash, props)
			VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), NULLIF($4, ''), $5::jsonb)`,
			e.Name, e.Locale, e.Path, e.SessionHash, string(props)); err != nil {
			return err
		}
	}
	return nil
}

// SignupsByLocale counts users per locale (S5 admin overview).
func (r *SEORepo) SignupsByLocale(ctx context.Context) (map[string]int64, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT locale, COUNT(*) FROM users GROUP BY locale`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var locale string
		var n int64
		if err := rows.Scan(&locale, &n); err != nil {
			return nil, err
		}
		out[locale] = n
	}
	return out, rows.Err()
}

// InquiryCounts counts business inquiries per status (S5 conversion).
func (r *SEORepo) InquiryCounts(ctx context.Context) (map[string]int64, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM business_inquiries GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var status string
		var n int64
		if err := rows.Scan(&status, &n); err != nil {
			return nil, err
		}
		out[status] = n
	}
	return out, rows.Err()
}

// IndexNowLog inserts submission rows (S3 audit trail).
func (r *SEORepo) IndexNowLog(ctx context.Context, urls []string, status string) error {
	for _, u := range urls {
		if _, err := r.db.ExecContext(ctx,
			`INSERT INTO indexnow_log (url, status) VALUES ($1, $2)`, u, status); err != nil {
			return err
		}
	}
	return nil
}

// IndexNowMarkSent marks the newest pending row per URL as sent/failed.
func (r *SEORepo) IndexNowMarkSent(ctx context.Context, urls []string, ok bool, errMsg string) error {
	status := "sent"
	if !ok {
		status = "failed"
	}
	for _, u := range urls {
		if _, err := r.db.ExecContext(ctx, `
			UPDATE indexnow_log SET status = $2, attempts = attempts + 1,
				last_error = NULLIF($3, ''), sent_at = CASE WHEN $2 = 'sent' THEN NOW() ELSE sent_at END
			WHERE id = (SELECT id FROM indexnow_log WHERE url = $1 ORDER BY created_at DESC LIMIT 1)`,
			u, status, errMsg); err != nil {
			return err
		}
	}
	return nil
}

// IndexNowSentSince counts sent rows since t (daily cap accounting).
func (r *SEORepo) IndexNowSentSince(ctx context.Context, t time.Time) (int64, error) {
	var n int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM indexnow_log WHERE status = 'sent' AND sent_at >= $1`, t).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}
