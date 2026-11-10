package repository

import (
	"context"
	"time"
)

// DownloadRepo accesses download_logs (P2-1 #22 quota + audit).
type DownloadRepo struct{ db DBTX }

// Log records one served download.
func (r *DownloadRepo) Log(ctx context.Context, userID, trackID string, size int64) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO download_logs (user_id, track_id, bytes) VALUES ($1::uuid, $2::uuid, $3)`,
		userID, trackID, size)
	return err
}

// CountSince counts a user's downloads at or after since (quota window).
func (r *DownloadRepo) CountSince(ctx context.Context, userID string, since time.Time) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM download_logs WHERE user_id = $1::uuid AND created_at >= $2`,
		userID, since).Scan(&n)
	return n, err
}
