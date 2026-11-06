// Package licensegate validates commercial eligibility before upload.
package licensegate

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"ai-music-platform/backend/internal/domain"
)

// StagingKeyPrefix is the object-key prefix for pre-check staging uploads.
const StagingKeyPrefix = "staging"

// BuildStagingKey returns the canonical staging object key that binds a
// pre-check grant to one partner and one audio SHA-256.
func BuildStagingKey(partnerID, audioSHA256, fileName string) string {
	name := path.Base(strings.TrimSpace(fileName))
	return fmt.Sprintf("%s/%s/%s/%s", StagingKeyPrefix, partnerID, strings.ToLower(strings.TrimSpace(audioSHA256)), name)
}

// ParseStagingKey splits a canonical staging key into partner ID, audio
// SHA-256, and file name. ok is false for any non-canonical key.
func ParseStagingKey(key string) (partnerID, audioSHA256, fileName string, ok bool) {
	parts := strings.Split(strings.Trim(key, "/"), "/")
	if len(parts) != 4 || parts[0] != StagingKeyPrefix {
		return "", "", "", false
	}
	if parts[1] == "" || parts[2] == "" || parts[3] == "" || parts[3] == "." || parts[3] == "/" {
		return "", "", "", false
	}
	if strings.Contains(parts[3], "..") {
		return "", "", "", false
	}
	return parts[1], parts[2], parts[3], true
}

// PreCheckRequest carries the inputs for commercial eligibility validation.
type PreCheckRequest struct {
	PartnerID                   string
	Provider                    string
	AudioSHA256                 string
	GeneratedAt                 time.Time
	SubscriptionID              string
	RequiresActiveSubAtCreation bool
}

// Policy is the license-gate view of an AI provider policy row.
type Policy struct {
	Provider                  string
	AllowsStandaloneStreaming bool
	MinBitrateKbps            int
	BlockedMetadataKeywords   []string
	RejectionMessageKo        string
}

// Subscription is the license-gate view of a partner subscription row.
type Subscription struct {
	ID         string
	PartnerID  string
	Provider   string
	PlanName   string
	Status     string
	CreatedAt  time.Time
	VerifiedAt *time.Time
	ExpiresAt  *time.Time
}

// Store abstracts the lookups needed for eligibility validation.
type Store interface {
	AudioHashExists(ctx context.Context, sha256hex string) (bool, error)
	GetPolicy(ctx context.Context, provider string) (*Policy, error)
	GetSubscription(ctx context.Context, partnerID, provider, subscriptionID string) (*Subscription, error)
}

// EligibilityResult carries the rows that justified a pass.
type EligibilityResult struct {
	Policy       *Policy
	Subscription *Subscription
}

// ValidateCommercialEligibility enforces the license gate:
//  1. duplicate SHA-256 check,
//  2. ai_provider_policies lookup (allows_standalone_streaming must be true),
//  3. when RequiresActiveSubAtCreation, subscription window check
//     (start <= generated <= end, status not REJECTED).
//
// Failures map to 403 gate codes (duplicate maps to 409).
func ValidateCommercialEligibility(ctx context.Context, store Store, req PreCheckRequest) (*EligibilityResult, error) {
	if req.AudioSHA256 != "" {
		exists, err := store.AudioHashExists(ctx, req.AudioSHA256)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, fmt.Errorf("%w: sha256 already registered", domain.ErrDuplicateAudioFile)
		}
	}

	policy, err := store.GetPolicy(ctx, req.Provider)
	if err != nil {
		return nil, err
	}
	if policy == nil || !policy.AllowsStandaloneStreaming {
		msg := "provider license does not allow standalone streaming"
		if policy != nil && policy.RejectionMessageKo != "" {
			msg = policy.RejectionMessageKo
		}
		return nil, &domain.GateError{Code: domain.CodeSyncOnlyLicense, Message: msg}
	}
	result := &EligibilityResult{Policy: policy}

	if req.RequiresActiveSubAtCreation {
		if req.GeneratedAt.IsZero() {
			return nil, fmt.Errorf("generated_at is required: %w", domain.ErrInvalidInput)
		}
		sub, err := store.GetSubscription(ctx, req.PartnerID, req.Provider, req.SubscriptionID)
		if err != nil {
			return nil, err
		}
		if sub == nil {
			return nil, &domain.GateError{Code: domain.CodeNonCommercialTier, Message: "no verified subscription for provider"}
		}
		if sub.Status == domain.SubRejected {
			return nil, &domain.GateError{Code: domain.CodeNonCommercialTier, Message: "subscription was rejected"}
		}
		if sub.Status != domain.SubVerified {
			return nil, &domain.GateError{Code: domain.CodeNonCommercialTier, Message: "subscription is not verified"}
		}
		start := sub.CreatedAt
		if sub.VerifiedAt != nil && !sub.VerifiedAt.IsZero() {
			start = *sub.VerifiedAt
		}
		if req.GeneratedAt.Before(start) {
			return nil, &domain.GateError{
				Code:    domain.CodeRetroactiveGenerationDate,
				Message: "generated before subscription window",
			}
		}
		if sub.ExpiresAt != nil && !sub.ExpiresAt.IsZero() && req.GeneratedAt.After(*sub.ExpiresAt) {
			return nil, &domain.GateError{
				Code:    domain.CodeNonCommercialTier,
				Message: "generated outside subscription window",
			}
		}
		result.Subscription = sub
	}
	return result, nil
}
