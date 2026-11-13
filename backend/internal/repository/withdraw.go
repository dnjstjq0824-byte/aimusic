package repository

import (
	"context"
)

// WithdrawUserTx runs the whole withdrawal in one transaction: deactivate,
// purge user + partner PII, cancel the active subscription, delete reset /
// OAuth / notification rows, and bump the token version (revoking every
// outstanding token). Any failure rolls everything back, so an account can
// never strand half-withdrawn (deactivated but token-live, or purged but
// still subscribed).
//
// The statements mirror UserRepo.SetActive/PurgePII/BumpTokenVersion,
// PartnerRepo.PurgePIIByUser, BillingRepo.CancelActiveSubscription,
// PasswordResetRepo.DeleteForUser, OAuthRepo.DeleteForUser and
// NotificationRepo.DeleteForUser; keep them in sync.
func (r *Repos) WithdrawUserTx(ctx context.Context, userID string) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	steps := []struct {
		name string
		sql  string
	}{
		{"deactivate", `UPDATE users SET is_active = FALSE WHERE id = $1::uuid`},
		{"purge user PII", `
			UPDATE users SET
				email = 'deleted+' || id::text || '@deleted.avibe.local',
				display_name = '탈퇴한 사용자',
				password_hash = '!',
				birth_year = NULL,
				totp_secret = NULL, totp_enabled = FALSE, totp_enrolled_at = NULL,
				totp_recovery_hashes = '{}'
			WHERE id = $1::uuid`},
		{"purge partner PII", `
			UPDATE partner_profiles SET contact_email = 'deleted+' || user_id::text || '@deleted.avibe.local',
				business_number = NULL, settlement_bank = NULL, settlement_account_encrypted = NULL
			WHERE user_id = $1::uuid`},
		{"cancel subscription", `
			UPDATE billing_subscriptions SET status = 'CANCELLED'
			WHERE user_id = $1::uuid AND status = 'ACTIVE'`},
		{"delete reset tokens", `
			DELETE FROM password_reset_tokens WHERE user_id = $1::uuid AND used_at IS NULL`},
		{"delete oauth links", `DELETE FROM oauth_accounts WHERE user_id = $1::uuid`},
		{"delete notifications", `DELETE FROM notifications WHERE user_id = $1::uuid`},
		{"bump token version", `UPDATE users SET token_version = token_version + 1 WHERE id = $1::uuid`},
	}
	for _, st := range steps {
		if _, err := tx.ExecContext(ctx, st.sql, userID); err != nil {
			return err
		}
	}
	return tx.Commit()
}
