// Package service implements application use cases over repositories.
package service

import (
	"strings"

	"avibe-platform/backend/internal/config"
	apcrypto "avibe-platform/backend/internal/pkg/crypto"
	"avibe-platform/backend/internal/pkg/mail"
	"avibe-platform/backend/internal/pkg/storage"
	"avibe-platform/backend/internal/repository"
)

// Services aggregates all application services.
type Services struct {
	Auth          *AuthService
	Precheck      *PrecheckService
	Finalize      *FinalizeService
	Review        *ReviewService
	Disputes      *DisputeService
	Playlists     *PlaylistService
	Playback      *PlaybackService
	Settlement    *SettlementService
	PasswordReset *PasswordResetService
	OAuth         *OAuthService
	Billing       *BillingService
	Overview      *OverviewService
}

// New builds the service aggregate.
//
// The settlement cipher is nil when AES256_KEY_BASE64 is unset; settlement
// account writes then fail closed instead of storing plaintext. Mail falls
// back to log-only delivery when SMTP_HOST is unset.
func New(repos *repository.Repos, st storage.Storage, buckets storage.Buckets, cfg *config.Config) *Services {
	var settlementCipher *apcrypto.Cipher
	if len(cfg.AESKey) == 32 {
		if c, err := apcrypto.NewCipher(cfg.AESKey); err == nil {
			settlementCipher = c
		}
	}
	var mailer mail.Mailer = mail.LogMailer{}
	if strings.TrimSpace(cfg.Mail.SMTPHost) != "" {
		mailer = mail.NewSMTPMailer(cfg.Mail.SMTPHost, cfg.Mail.SMTPPort,
			cfg.Mail.SMTPUser, cfg.Mail.SMTPPass, cfg.Mail.From, cfg.Mail.FromName)
	}
	auth := NewAuthService(repos, cfg)
	return &Services{
		Auth:          auth,
		Precheck:      &PrecheckService{repos: repos, storage: st, stagingBucket: buckets.Staging},
		Finalize:      &FinalizeService{repos: repos, stagingBucket: buckets.Staging, hlsBucket: buckets.PublicHLS},
		Review:        &ReviewService{repos: repos},
		Disputes:      &DisputeService{repos: repos},
		Playlists:     &PlaylistService{repos: repos},
		Playback:      &PlaybackService{repos: repos},
		Settlement:    &SettlementService{repos: repos, defaultRatePerPlayKRW: 10, cipher: settlementCipher},
		PasswordReset: NewPasswordResetService(repos, mailer, cfg),
		OAuth:         NewOAuthService(repos, auth, cfg, nil),
		Billing:       NewBillingService(repos, cfg),
		Overview:      NewOverviewService(repos),
	}
}
