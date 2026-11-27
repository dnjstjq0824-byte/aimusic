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
	Engagement    *EngagementService
	Comments      *CommentService
	StudioTracks  *StudioTrackService
	Business      *BusinessService
	Playback      *PlaybackService
	Settlement    *SettlementService
	PasswordReset *PasswordResetService
	EmailVerify   *EmailVerifyService
	OAuth         *OAuthService
	Billing       *BillingService
	Overview      *OverviewService
	Generation    *GenerationService
	Distribution  *DistributionService
	Store         *StoreService
	Certificates  *CertificateService
	SEO           *SEOService
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
	oauth := NewOAuthService(repos, auth, cfg, nil)
	disputes := &DisputeService{repos: repos, storage: st, stagingBucket: buckets.Staging, hlsBucket: buckets.PublicHLS}
	return &Services{
		Auth:          auth,
		Precheck:      &PrecheckService{repos: repos, storage: st, stagingBucket: buckets.Staging},
		Finalize:      &FinalizeService{repos: repos, stagingBucket: buckets.Staging, hlsBucket: buckets.PublicHLS},
		Review:        &ReviewService{repos: repos},
		Disputes:      disputes,
		Playlists:     &PlaylistService{repos: repos},
		Engagement:    &EngagementService{repos: repos},
		Comments:      NewCommentService(repos),
		StudioTracks:  NewStudioTrackService(repos, disputes),
		Business:      NewBusinessService(repos, mailer, cfg.Mail.NotifyEmail),
		Playback:      &PlaybackService{repos: repos},
		Settlement:    &SettlementService{repos: repos, cipher: settlementCipher},
		PasswordReset: NewPasswordResetService(repos, mailer, cfg),
		EmailVerify:   NewEmailVerifyService(repos, auth, mailer, oauth, cfg),
		OAuth:         oauth,
		Billing:       NewBillingService(repos, cfg),
		Overview:      NewOverviewService(repos),
		Generation:    NewGenerationService(repos, cfg.Generation.DailyCapPerPartner, cfg.Generation.BannedKeywordsExtra),
		Distribution:  NewDistributionService(repos, st, buckets.Staging),
		Store:         NewStoreService(repos),
		Certificates:  NewCertificateService(repos),
		SEO:           NewSEOService(repos, cfg.SEO, nil),
	}
}
