package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ai-music-platform/backend/internal/domain"
	"ai-music-platform/backend/internal/pkg/licensegate"
	"ai-music-platform/backend/internal/pkg/storage"
	"ai-music-platform/backend/internal/repository"
)

// PrecheckInput carries partner upload intent.
//
// RequiresActiveSub is accepted for API compatibility but ignored: the
// subscription-window check (Gate 2) is always enforced server-side.
type PrecheckInput struct {
	PartnerID         string
	Provider          string
	PlanTierUsed      string
	AudioSHA256       string
	FileName          string
	GeneratedAt       time.Time
	SubscriptionID    string
	RequiresActiveSub bool
}

// PrecheckOutput carries the staging upload grant.
type PrecheckOutput struct {
	Eligible      bool   `json:"eligible"`
	StagingKey    string `json:"stagingKey"`
	UploadURL     string `json:"uploadUrl"`
	ExpiresInSecs int    `json:"expiresInSecs"`
}

// PrecheckService runs the license gate and issues staging URLs only on pass.
type PrecheckService struct {
	repos         *repository.Repos
	storage       storage.Storage
	stagingBucket string
}

// PreCheck validates commercial eligibility and, only on pass, returns a
// staging presigned PUT URL whose key embeds the audio SHA-256.
func (s *PrecheckService) PreCheck(ctx context.Context, in PrecheckInput) (*PrecheckOutput, error) {
	if in.PartnerID == "" || !domain.ValidProvider(in.Provider) || in.AudioSHA256 == "" || in.FileName == "" {
		return nil, fmt.Errorf("%w: partner/provider/sha256/fileName", domain.ErrInvalidInput)
	}
	sha := strings.ToLower(strings.TrimSpace(in.AudioSHA256))
	if _, err := licensegate.ValidateCommercialEligibility(ctx, s.repos.Gate, licensegate.PreCheckRequest{
		PartnerID:                   in.PartnerID,
		Provider:                    in.Provider,
		PlanTierUsed:                in.PlanTierUsed,
		AudioSHA256:                 sha,
		GeneratedAt:                 in.GeneratedAt,
		SubscriptionID:              in.SubscriptionID,
		RequiresActiveSubAtCreation: true,
	}); err != nil {
		return nil, err
	}
	key := licensegate.BuildStagingKey(in.PartnerID, sha, in.FileName)
	if _, _, _, ok := licensegate.ParseStagingKey(key); !ok {
		return nil, fmt.Errorf("%w: fileName", domain.ErrInvalidInput)
	}
	url, err := s.storage.PresignPut(ctx, s.stagingBucket, key, 15*time.Minute)
	if err != nil {
		return nil, err
	}
	return &PrecheckOutput{Eligible: true, StagingKey: key, UploadURL: url, ExpiresInSecs: 900}, nil
}

// RightsInput carries the rights declaration for finalize.
type RightsInput struct {
	DeclaresOwnership               bool
	DeclaresNoVoiceClone            bool
	DeclaresTrainingConsent         bool
	ConfirmedNoUnauthorizedSampling bool
	ConfirmedIndemnification        bool
	PlanTierUsed                    *string
	ProviderTrackURL                *string
	HumanContributionTypes          []string
	HumanContributionDesc           *string
	SignedIP                        *string
	SourceModel                     *string
	SubscriptionID                  *string
}

// FinalizeInput carries the completed staging upload to register.
//
// AudioSHA256 and GeneratedAt must repeat the values granted at pre-check;
// Finalize re-validates the license gate so pre-check cannot be skipped.
type FinalizeInput struct {
	PartnerID        string
	StagingKey       string
	Title            string
	Provider         string
	PlanTierUsed     string
	AudioSHA256      string
	GeneratedAt      time.Time
	SubscriptionID   string
	AlbumID          *string
	Genre            *string
	MoodTags         []string
	AIGenerationType string
	AIModelVersion   *string
	AIPrompt         *string
	IsPromptPublic   bool
	IsInstrumental   bool
	Lyrics           *string
	DurationMs       *int
	BitrateKbps      *int
	Disclosure       *string
	Rights           RightsInput
}

// FinalizeOutput carries created IDs.
type FinalizeOutput struct {
	TrackID string `json:"trackId"`
	JobID   string `json:"jobId"`
}

// FinalizeService registers uploaded tracks and enqueues transcode jobs.
type FinalizeService struct {
	repos         *repository.Repos
	stagingBucket string
	hlsBucket     string
}

// Finalize creates the track row, rights declaration, and transcode job, then
// enqueues the worker payload.
//
// Finalize binds the staging object to the pre-check grant (same partner,
// same audio SHA-256) and re-runs the license gate, so a client cannot skip
// pre-check by calling finalize directly with an arbitrary key.
func (s *FinalizeService) Finalize(ctx context.Context, in FinalizeInput) (*FinalizeOutput, error) {
	if in.PartnerID == "" || in.StagingKey == "" || in.Title == "" || !domain.ValidProvider(in.Provider) {
		return nil, fmt.Errorf("%w: partner/stagingKey/title/provider", domain.ErrInvalidInput)
	}
	if in.AudioSHA256 == "" || in.GeneratedAt.IsZero() {
		return nil, fmt.Errorf("%w: audioSha256/generatedAt", domain.ErrInvalidInput)
	}
	keyPartner, keySHA, _, ok := licensegate.ParseStagingKey(in.StagingKey)
	if !ok {
		return nil, fmt.Errorf("%w: stagingKey", domain.ErrInvalidInput)
	}
	sha := strings.ToLower(strings.TrimSpace(in.AudioSHA256))
	if keyPartner != in.PartnerID || !strings.EqualFold(keySHA, sha) {
		return nil, fmt.Errorf("%w: stagingKey does not match pre-check grant", domain.ErrForbidden)
	}
	if _, err := licensegate.ValidateCommercialEligibility(ctx, s.repos.Gate, licensegate.PreCheckRequest{
		PartnerID:                   in.PartnerID,
		Provider:                    in.Provider,
		PlanTierUsed:                in.PlanTierUsed,
		AudioSHA256:                 sha,
		GeneratedAt:                 in.GeneratedAt,
		SubscriptionID:              in.SubscriptionID,
		RequiresActiveSubAtCreation: true,
	}); err != nil {
		return nil, err
	}
	if in.AlbumID != nil {
		album, err := s.repos.Albums.GetByID(ctx, *in.AlbumID)
		if err != nil {
			return nil, err
		}
		if album.PartnerID != in.PartnerID {
			return nil, domain.ErrForbidden
		}
	}
	track, err := s.repos.Tracks.Create(ctx, &domain.Track{
		AlbumID:           in.AlbumID,
		PartnerID:         in.PartnerID,
		Title:             in.Title,
		Genre:             in.Genre,
		MoodTags:          in.MoodTags,
		AIGenerationType:  in.AIGenerationType,
		AIModelVersion:    in.AIModelVersion,
		AIPrompt:          in.AIPrompt,
		IsPromptPublic:    in.IsPromptPublic,
		IsInstrumental:    in.IsInstrumental,
		Lyrics:            in.Lyrics,
		DurationMs:        in.DurationMs,
		OriginalObjectKey: in.StagingKey,
		AIProvider:        in.Provider,
		AIGenerated:       true,
		AIDisclosureText:  in.Disclosure,
		BitrateKbps:       in.BitrateKbps,
		Status:            domain.TrackUploaded,
	})
	if err != nil {
		return nil, err
	}
	generatedAt := in.GeneratedAt
	planTierPtr := in.Rights.PlanTierUsed
	if (planTierPtr == nil || *planTierPtr == "") && in.PlanTierUsed != "" {
		pt := in.PlanTierUsed
		planTierPtr = &pt
	}
	if _, err := s.repos.Rights.Upsert(ctx, &domain.TrackRightsDeclaration{
		TrackID:                         track.ID,
		PartnerID:                       in.PartnerID,
		DeclaresOwnership:               in.Rights.DeclaresOwnership,
		DeclaresNoVoiceClone:            in.Rights.DeclaresNoVoiceClone,
		DeclaresTrainingConsent:         in.Rights.DeclaresTrainingConsent,
		ConfirmedNoUnauthorizedSampling: in.Rights.ConfirmedNoUnauthorizedSampling,
		ConfirmedIndemnification:        in.Rights.ConfirmedIndemnification,
		PlanTierUsed:                    planTierPtr,
		ProviderTrackURL:                in.Rights.ProviderTrackURL,
		HumanContributionTypes:          in.Rights.HumanContributionTypes,
		HumanContributionDesc:           in.Rights.HumanContributionDesc,
		Gate1PolicyPassed:               true,
		Gate2DatePassed:                 true,
		SignedIP:                        in.Rights.SignedIP,
		SourceModel:                     in.Rights.SourceModel,
		SubscriptionID:                  in.Rights.SubscriptionID,
		GeneratedAt:                     &generatedAt,
	}); err != nil {
		return nil, err
	}
	outputPrefix := "hls/" + track.ID
	job, err := s.repos.Jobs.Create(ctx, track.ID, in.StagingKey, outputPrefix)
	if err != nil {
		return nil, err
	}
	if err := s.repos.Queue.EnqueueTranscode(ctx, repository.TranscodePayload{
		JobID: job.ID, TrackID: track.ID,
		InputBucket: s.stagingBucket, InputKey: in.StagingKey, OutputPrefix: outputPrefix,
	}); err != nil {
		return nil, err
	}
	return &FinalizeOutput{TrackID: track.ID, JobID: job.ID}, nil
}

// CreateAlbum creates a partner album shell for later release.
func (s *FinalizeService) CreateAlbum(ctx context.Context, partnerID, title string, coverKey *string, releaseType string) (*domain.Album, error) {
	if partnerID == "" || title == "" {
		return nil, fmt.Errorf("%w: partner/title", domain.ErrInvalidInput)
	}
	return s.repos.Albums.Create(ctx, partnerID, title, coverKey, releaseType)
}
