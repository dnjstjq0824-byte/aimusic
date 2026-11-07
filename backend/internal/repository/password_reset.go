package repository

import (
	"context"
	"database/sql"
	"time"

	"avibe-platform/backend/internal/domain"
)

// PasswordResetRepo accesses password_reset_tokens.
type PasswordResetRepo struct{ db *sql.DB }

func scanResetToken(row interface{ Scan(...any) error }) (*domain.PasswordResetToken, error) {
	t := &domain.PasswordResetToken{}
	var used sql.NullTime
	err := row.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &used, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	t.UsedAt = timePtr(used)
	return t, nil
}

const resetTokenColumns = `id::text, user_id::text, token_hash, expires_at, used_at, created_at`

// Create stores a hashed single-use reset token.
func (r *PasswordResetRepo) Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*domain.PasswordResetToken, error) {
	t, err := scanResetToken(r.db.QueryRowContext(ctx, `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		VALUES ($1::uuid, $2, $3)
		RETURNING `+resetTokenColumns, userID, tokenHash, expiresAt))
	return t, err
}

// GetByHash returns a reset token by its hash.
func (r *PasswordResetRepo) GetByHash(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error) {
	t, err := scanResetToken(r.db.QueryRowContext(ctx,
		`SELECT `+resetTokenColumns+` FROM password_reset_tokens WHERE token_hash = $1`, tokenHash))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return t, err
}

// MarkUsed consumes a reset token exactly once.
func (r *PasswordResetRepo) MarkUsed(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE password_reset_tokens SET used_at = NOW() WHERE id = $1::uuid AND used_at IS NULL`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// DeleteForUser removes a user's outstanding reset tokens (issued anew on each request).
func (r *PasswordResetRepo) DeleteForUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM password_reset_tokens WHERE user_id = $1::uuid AND used_at IS NULL`, userID)
	return err
}
