package repository

import (
	"context"
	"database/sql"
	"time"

	"avibe-platform/backend/internal/domain"
)

// CommentRepo accesses track_comments.
type CommentRepo struct{ db *sql.DB }

const commentSelect = `SELECT c.id::text, c.track_id::text, c.user_id::text, u.display_name, c.body, c.created_at
	FROM track_comments c JOIN users u ON u.id = c.user_id`

func scanComment(row interface{ Scan(...any) error }) (*domain.TrackComment, error) {
	c := &domain.TrackComment{}
	var name sql.NullString
	if err := row.Scan(&c.ID, &c.TrackID, &c.UserID, &name, &c.Body, &c.CreatedAt); err != nil {
		return nil, err
	}
	c.AuthorName = name.String
	return c, nil
}

// Create inserts a comment and returns it with the author's display name.
func (r *CommentRepo) Create(ctx context.Context, trackID, userID, body string) (*domain.TrackComment, error) {
	var id string
	if err := r.db.QueryRowContext(ctx, `
		INSERT INTO track_comments (track_id, user_id, body) VALUES ($1::uuid, $2::uuid, $3)
		RETURNING id::text`, trackID, userID, body).Scan(&id); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

// GetByID returns one comment.
func (r *CommentRepo) GetByID(ctx context.Context, id string) (*domain.TrackComment, error) {
	c, err := scanComment(r.db.QueryRowContext(ctx, commentSelect+` WHERE c.id = $1::uuid`, id))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return c, err
}

// ListByTrack returns newest-first comments, keyset-paged by created_at
// (before == zero time means "from the newest"). Comments by deactivated
// accounts are hidden.
func (r *CommentRepo) ListByTrack(ctx context.Context, trackID string, before time.Time, limit int) ([]*domain.TrackComment, error) {
	var beforeArg any
	if !before.IsZero() {
		beforeArg = before
	}
	rows, err := r.db.QueryContext(ctx, commentSelect+`
		WHERE c.track_id = $1::uuid AND u.is_active = TRUE
			AND ($2::timestamptz IS NULL OR c.created_at < $2::timestamptz)
		ORDER BY c.created_at DESC LIMIT $3`, trackID, beforeArg, clampLimit(limit, 20, 50))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.TrackComment{}
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CountByTrack counts visible comments on a track.
func (r *CommentRepo) CountByTrack(ctx context.Context, trackID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM track_comments c JOIN users u ON u.id = c.user_id
		WHERE c.track_id = $1::uuid AND u.is_active = TRUE`, trackID).Scan(&n)
	return n, err
}

// Delete removes one comment.
func (r *CommentRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM track_comments WHERE id = $1::uuid`, id)
	return err
}
