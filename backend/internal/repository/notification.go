package repository

import (
	"context"
	"database/sql"
)

// NotificationRepo writes in-app notifications (SLA alerts, takedown events).
type NotificationRepo struct{ db *sql.DB }

// Create inserts one notification row.
func (r *NotificationRepo) Create(ctx context.Context, userID, kind, titleKo, bodyKo, linkPath string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO notifications (user_id, kind, title_ko, body_ko, link_path)
		VALUES ($1::uuid, $2, $3, $4, $5)`,
		userID, kind, titleKo, bodyKo, linkPath)
	return err
}

// DeleteForUser removes a user's notifications (withdrawal cleanup).
func (r *NotificationRepo) DeleteForUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM notifications WHERE user_id = $1::uuid`, userID)
	return err
}
