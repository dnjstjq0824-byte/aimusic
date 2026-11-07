package repository

import (
	"context"
	"database/sql"

	"ai-music-platform/backend/internal/domain"
)

// OAuthRepo accesses oauth_accounts.
type OAuthRepo struct{ db *sql.DB }

func scanOAuth(row interface{ Scan(...any) error }) (*domain.OAuthAccount, error) {
	a := &domain.OAuthAccount{}
	var email sql.NullString
	err := row.Scan(&a.ID, &a.UserID, &a.Provider, &a.ProviderUserID, &email, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	a.Email = strPtr(email)
	return a, nil
}

const oauthColumns = `id::text, user_id::text, provider, provider_user_id, email, created_at, updated_at`

// GetByProviderUID returns the link for a provider user ID.
func (r *OAuthRepo) GetByProviderUID(ctx context.Context, provider, providerUID string) (*domain.OAuthAccount, error) {
	a, err := scanOAuth(r.db.QueryRowContext(ctx,
		`SELECT `+oauthColumns+` FROM oauth_accounts WHERE provider = $1 AND provider_user_id = $2`,
		provider, providerUID))
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return a, err
}

// Upsert links a provider identity to a user (idempotent per provider+UID).
func (r *OAuthRepo) Upsert(ctx context.Context, userID, provider, providerUID string, email *string) (*domain.OAuthAccount, error) {
	return scanOAuth(r.db.QueryRowContext(ctx, `
		INSERT INTO oauth_accounts (user_id, provider, provider_user_id, email)
		VALUES ($1::uuid, $2, $3, $4)
		ON CONFLICT (provider, provider_user_id) DO UPDATE SET email = EXCLUDED.email
		RETURNING `+oauthColumns, userID, provider, providerUID, nullStr(email)))
}
