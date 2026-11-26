package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"avibe-platform/backend/internal/domain"
)

// DownloadRepo accesses download_logs (P2-1 #22 quota + audit).
type DownloadRepo struct{ db DBTX }

// Log records one served download, enforcing the rolling quota atomically:
// the user row is locked in its own statement inside an explicit
// transaction, so concurrent downloads serialize and the window count can
// never overshoot max. A lost race is ErrQuotaExceeded (maps to 429), never
// a silent over-log.
func (r *DownloadRepo) Log(ctx context.Context, userID, trackID string, size int64, windowStart time.Time, max int64) error {
	// Downloads never runs inside WithTx, so the handle is always the pool
	// in practice; the fallback keeps transactional callers serial-correct.
	if sqldb, ok := r.db.(*sql.DB); ok {
		tx, err := sqldb.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		var one int
		if err := tx.QueryRowContext(ctx,
			`SELECT 1 FROM users WHERE id = $1::uuid FOR UPDATE`, userID).Scan(&one); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrNotFound
			}
			return err
		}
		var id string
		err = tx.QueryRowContext(ctx, `
			INSERT INTO download_logs (user_id, track_id, bytes)
			SELECT $1::uuid, $2::uuid, $3
			WHERE (SELECT COUNT(*) FROM download_logs
			       WHERE user_id = $1::uuid AND created_at >= $4) < $5
			RETURNING id::text`,
			userID, trackID, size, windowStart, max).Scan(&id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrQuotaExceeded
			}
			return err
		}
		return tx.Commit()
	}
	var id string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO download_logs (user_id, track_id, bytes)
		SELECT $1::uuid, $2::uuid, $3
		WHERE (SELECT COUNT(*) FROM download_logs
		       WHERE user_id = $1::uuid AND created_at >= $4) < $5
		RETURNING id::text`,
		userID, trackID, size, windowStart, max).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrQuotaExceeded
		}
		return err
	}
	return nil
}

// CountSince counts a user's downloads at or after since (quota window).
func (r *DownloadRepo) CountSince(ctx context.Context, userID string, since time.Time) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM download_logs WHERE user_id = $1::uuid AND created_at >= $2`,
		userID, since).Scan(&n)
	return n, err
}
