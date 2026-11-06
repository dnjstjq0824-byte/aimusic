// Package service implements application use cases over repositories.
package service

import (
	"ai-music-platform/backend/internal/config"
	apcrypto "ai-music-platform/backend/internal/pkg/crypto"
	"ai-music-platform/backend/internal/pkg/storage"
	"ai-music-platform/backend/internal/repository"
)

// Services aggregates all application services.
type Services struct {
	Auth       *AuthService
	Precheck   *PrecheckService
	Finalize   *FinalizeService
	Review     *ReviewService
	Disputes   *DisputeService
	Playlists  *PlaylistService
	Playback   *PlaybackService
	Settlement *SettlementService
}

// New builds the service aggregate.
//
// The settlement cipher is nil when AES256_KEY_BASE64 is unset; settlement
// account writes then fail closed instead of storing plaintext.
func New(repos *repository.Repos, st storage.Storage, buckets storage.Buckets, cfg *config.Config) *Services {
	var settlementCipher *apcrypto.Cipher
	if len(cfg.AESKey) == 32 {
		if c, err := apcrypto.NewCipher(cfg.AESKey); err == nil {
			settlementCipher = c
		}
	}
	return &Services{
		Auth:       NewAuthService(repos, cfg),
		Precheck:   &PrecheckService{repos: repos, storage: st, stagingBucket: buckets.Staging},
		Finalize:   &FinalizeService{repos: repos, stagingBucket: buckets.Staging, hlsBucket: buckets.PublicHLS},
		Review:     &ReviewService{repos: repos},
		Disputes:   &DisputeService{repos: repos},
		Playlists:  &PlaylistService{repos: repos},
		Playback:   &PlaybackService{repos: repos},
		Settlement: &SettlementService{repos: repos, defaultRatePerPlayKRW: 10, cipher: settlementCipher},
	}
}
