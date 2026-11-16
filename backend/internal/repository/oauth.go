package repository

import (
	"context"
	"database/sql"

	"avibe-platform/backend/internal/domain"
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

// OAuthProviderStat is one provider's link count (admin, R11).
type OAuthProviderStat struct {
	Provider string `json:"provider"`
	Links    int64  `json:"links"`
}

// OAuthStats is the social-connection overview (admin, R11).
type OAuthStats struct {
	ByProvider  []*OAuthProviderStat `json:"byProvider"`
	LinkedUsers int64                `json:"linkedUsers"`
	TotalLinks  int64                `json:"totalLinks"`
}

// Stats counts social links per provider plus linked-user totals.
func (r *OAuthRepo) Stats(ctx context.Context) (*OAuthStats, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT provider, COUNT(*) FROM oauth_accounts GROUP BY provider ORDER BY provider ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := &OAuthStats{ByProvider: []*OAuthProviderStat{}}
	for rows.Next() {
		s := &OAuthProviderStat{}
		if err := rows.Scan(&s.Provider, &s.Links); err != nil {
			return nil, err
		}
		out.ByProvider = append(out.ByProvider, s)
		out.TotalLinks += s.Links
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT user_id) FROM oauth_accounts`).Scan(&out.LinkedUsers); err != nil {
		return nil, err
	}
	return out, nil
}

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

// DeleteForUser unlinks every provider identity (withdrawal cleanup).
func (r *OAuthRepo) DeleteForUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM oauth_accounts WHERE user_id = $1::uuid`, userID)
	return err
}

// ListByUserID returns every provider link for a user (connections page).
func (r *OAuthRepo) ListByUserID(ctx context.Context, userID string) ([]*domain.OAuthAccount, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+oauthColumns+` FROM oauth_accounts WHERE user_id = $1::uuid ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.OAuthAccount
	for rows.Next() {
		a := &domain.OAuthAccount{}
		var email sql.NullString
		if err := rows.Scan(&a.ID, &a.UserID, &a.Provider, &a.ProviderUserID, &email, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		a.Email = strPtr(email)
		out = append(out, a)
	}
	return out, rows.Err()
}

// DeleteByUserProvider unlinks one provider for a user.
func (r *OAuthRepo) DeleteByUserProvider(ctx context.Context, userID, provider string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM oauth_accounts WHERE user_id = $1::uuid AND provider = $2`, userID, provider)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		if err != nil {
			return err
		}
		return domain.ErrNotFound
	}
	return nil
}
