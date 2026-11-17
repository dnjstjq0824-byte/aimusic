package repository

import (
	"context"
	"database/sql"
	"time"

	"avibe-platform/backend/internal/domain"
)

// EmailVerifyRepo accesses email_verifications (WI-A2).
type EmailVerifyRepo struct{ db *sql.DB }

func scanEmailVerification(row interface{ Scan(...any) error }) (*domain.EmailVerification, error) {
	v := &domain.EmailVerification{}
	var userID, refToken sql.NullString
	var used sql.NullTime
	err := row.Scan(&v.ID, &userID, &v.Email, &v.Purpose, &v.TokenHash, &refToken,
		&v.ExpiresAt, &used, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	if userID.Valid {
		v.UserID = &userID.String
	}
	if refToken.Valid {
		v.RefToken = &refToken.String
	}
	v.UsedAt = timePtr(used)
	return v, nil
}

const emailVerifyColumns = `id::text, user_id::text, email, purpose, token_hash, ref_token, expires_at, used_at, created_at`

// Create stores a hashed single-use verification token. Empty userID/refToken
// store NULL (OAUTH rows have no user row yet; VERIFY rows have no ref).
func (r *EmailVerifyRepo) Create(ctx context.Context, userID, email, purpose, tokenHash, refToken string, expiresAt time.Time) (*domain.EmailVerification, error) {
	v, err := scanEmailVerification(r.db.QueryRowContext(ctx, `
		INSERT INTO email_verifications (user_id, email, purpose, token_hash, ref_token, expires_at)
		VALUES (NULLIF($1, '')::uuid, $2, $3, $4, NULLIF($5, ''), $6)
		RETURNING `+emailVerifyColumns, userID, email, purpose, tokenHash, refToken, expiresAt))
	return v, err
}

// GetByHash returns a verification by its token hash.
func (r *EmailVerifyRepo) GetByHash(ctx context.Context, tokenHash string) (*domain.EmailVerification, error) {
	v, err := scanEmailVerification(r.db.QueryRowContext(ctx,
		`SELECT `+emailVerifyColumns+` FROM email_verifications WHERE token_hash = $1`, tokenHash))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return v, err
}

// MarkUsed consumes a verification exactly once.
func (r *EmailVerifyRepo) MarkUsed(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE email_verifications SET used_at = NOW() WHERE id = $1::uuid AND used_at IS NULL`, id)
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

// DeletePendingForUser removes a user's outstanding verifications (a fresh
// request supersedes earlier links).
func (r *EmailVerifyRepo) DeletePendingForUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM email_verifications WHERE user_id = $1::uuid AND used_at IS NULL`, userID)
	return err
}

// DeletePendingForEmail removes outstanding verifications for an email +
// purpose (OAUTH re-submits supersede earlier links).
func (r *EmailVerifyRepo) DeletePendingForEmail(ctx context.Context, email, purpose string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM email_verifications WHERE email = $1 AND purpose = $2 AND used_at IS NULL`, email, purpose)
	return err
}
