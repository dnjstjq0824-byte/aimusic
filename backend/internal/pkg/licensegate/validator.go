// Package licensegate validates commercial eligibility before upload.
package licensegate

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	"avibe-platform/backend/internal/domain"
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
	PlanTierUsed                string
	AudioSHA256                 string
	GeneratedAt                 time.Time
	SubscriptionID              string
	RequiresActiveSubAtCreation bool
}

// Policy is the license-gate view of an AI provider policy row.
type Policy struct {
	Provider                  string
	AllowsStandaloneStreaming bool
	AllowedPlanTiers          []string
	BlockedPlanTiers          []string
	SyncOnlyPlanTiers         []string
	MinBitrateKbps            int
	BlockedMetadataKeywords   []string
	RejectionMessageKo        string
	// MaxGeneratedAt rejects tracks generated after the cutoff (nil = no cutoff).
	MaxGeneratedAt *time.Time
}

func tierMatches(tier string, list []string) bool {
	t := strings.ToLower(strings.TrimSpace(tier))
	if t == "" {
		return false
	}
	for _, item := range list {
		candidate := strings.ToLower(strings.TrimSpace(item))
		if candidate != "" && (t == candidate || strings.Contains(t, candidate)) {
			return true
		}
	}
	return false
}

func isBuiltInNonCommercialTier(tier string) bool {
	t := strings.ToLower(strings.TrimSpace(tier))
	switch {
	case t == "free", t == "basic", t == "free (basic)", t == "trial",
		strings.Contains(t, "cc-by-nc"), strings.Contains(t, "non-commercial"):
		return true
	}
	return false
}

func isBuiltInSyncOnlyTier(provider, tier string) bool {
	p := strings.ToUpper(strings.TrimSpace(provider))
	t := strings.ToLower(strings.TrimSpace(tier))
	if p == domain.ProviderAiva && t == "standard" {
		return true
	}
	if (p == domain.ProviderSoundraw || p == domain.ProviderMubert) &&
		(t == "creator" || strings.Contains(t, "video bgm")) {
		return true
	}
	return false
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

// BlockedSourceModels are known non-commercial / research-only model families
// (CC-BY-NC or research-only licenses). Source-model declarations matching
// these substrings are refused at certification even when the provider row
// would otherwise pass.
var BlockedSourceModels = []string{"musicgen", "audiogen", "musiclm", "stable-audio-open"}

// ValidateSourceModel refuses non-commercial model families (WI-02 model
// whitelist). Empty (undeclared) models pass; the provider gate still applies.
func ValidateSourceModel(sourceModel string) error {
	m := strings.ToLower(strings.TrimSpace(sourceModel))
	if m == "" {
		return nil
	}
	for _, blocked := range BlockedSourceModels {
		if strings.Contains(m, blocked) {
			return &domain.GateError{
				Code:    domain.CodeNonCommercialTier,
				Message: fmt.Sprintf("비상업 라이선스 모델(%s) 생성물은 인증할 수 없습니다", strings.TrimSpace(sourceModel)),
			}
		}
	}
	return nil
}

// ValidAICreditParts are the DDEX-style AI-generation scope declarations:
// which parts of the audio are AI-generated (all three = full AI audio).
var ValidAICreditParts = []string{"LYRICS", "VOCALS", "INSTRUMENTAL"}

// NormalizeAICreditParts upper-cases, dedupes, and validates AI-credit
// parts. Empty input yields nil without error (credits are optional).
func NormalizeAICreditParts(parts []string) ([]string, error) {
	if len(parts) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		up := strings.ToUpper(strings.TrimSpace(p))
		valid := false
		for _, v := range ValidAICreditParts {
			if up == v {
				valid = true
				break
			}
		}
		if !valid {
			return nil, fmt.Errorf("%w: aiCreditParts %q", domain.ErrInvalidInput, p)
		}
		if !seen[up] {
			seen[up] = true
			out = append(out, up)
		}
	}
	return out, nil
}

// PolicyVersion pins the certification semantics a track snapshot was taken
// under. Bump when gate rules change so old snapshots stay interpretable.
const PolicyVersion = "2026-11-11"

// CertSnapshot is the track-level commercial-certification record: the exact
// inputs and gate outcomes frozen at decision time, independent of later
// policy-row or subscription edits.
type CertSnapshot struct {
	PolicyVersion         string     `json:"policyVersion"`
	Provider              string     `json:"provider"`
	PlanTierUsed          *string    `json:"planTierUsed,omitempty"`
	SubscriptionID        *string    `json:"subscriptionId,omitempty"`
	GeneratedAt           *time.Time `json:"generatedAt,omitempty"`
	SourceModel           *string    `json:"sourceModel,omitempty"`
	HumanContributions    []string   `json:"humanContributions,omitempty"`
	CollectiveTrustStatus string     `json:"collectiveTrustStatus,omitempty"`
	Gate1PolicyPassed     bool       `json:"gate1PolicyPassed"`
	Gate2DatePassed       bool       `json:"gate2DatePassed"`
	DecidedAt             time.Time  `json:"decidedAt"`
}

// BuildCertSnapshot serializes a certification decision. Provider is required;
// DecidedAt defaults to now and PolicyVersion is pinned automatically.
func BuildCertSnapshot(s CertSnapshot) (string, error) {
	s.Provider = strings.ToUpper(strings.TrimSpace(s.Provider))
	if s.Provider == "" {
		return "", fmt.Errorf("cert snapshot requires provider: %w", domain.ErrInvalidInput)
	}
	s.PolicyVersion = PolicyVersion
	if s.DecidedAt.IsZero() {
		s.DecidedAt = time.Now().UTC()
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("cert snapshot marshal: %w", err)
	}
	return string(raw), nil
}

// ParseCertSnapshot decodes a stored certification record.
func ParseCertSnapshot(raw string) (*CertSnapshot, error) {
	var s CertSnapshot
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return nil, fmt.Errorf("cert snapshot parse: %w", err)
	}
	if s.Provider == "" {
		return nil, fmt.Errorf("cert snapshot missing provider: %w", domain.ErrInvalidInput)
	}
	return &s, nil
}

// ValidateCommercialEligibility enforces the license gate:
//  0. provider allowlist (known provider + registered policy row),
//  1. duplicate SHA-256 check,
//  2. ai_provider_policies lookup (allows_standalone_streaming must be true),
//     2b. provider generation cutoff (max_generated_at, e.g. Udio 2025-10-30),
//  3. when RequiresActiveSubAtCreation, subscription window check
//     (start <= generated <= end, status not REJECTED).
//
// Failures map to 403 gate codes (duplicate maps to 409).
func ValidateCommercialEligibility(ctx context.Context, store Store, req PreCheckRequest) (*EligibilityResult, error) {
	provider := strings.ToUpper(strings.TrimSpace(req.Provider))
	if provider == "" {
		return nil, fmt.Errorf("%w: provider", domain.ErrInvalidInput)
	}
	if !domain.ValidProvider(provider) {
		return nil, &domain.GateError{
			Code:    domain.CodeProviderNotAllowlisted,
			Message: fmt.Sprintf("지원하지 않는 AI 제공자입니다: %s", strings.TrimSpace(req.Provider)),
		}
	}
	if req.AudioSHA256 != "" {
		exists, err := store.AudioHashExists(ctx, req.AudioSHA256)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, fmt.Errorf("%w: sha256 already registered", domain.ErrDuplicateAudioFile)
		}
	}

	policy, err := store.GetPolicy(ctx, provider)
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return nil, &domain.GateError{
			Code:    domain.CodeProviderNotAllowlisted,
			Message: fmt.Sprintf("%s 제공자 정책이 등록되어 있지 않습니다", provider),
		}
	}
	msg := "provider license does not allow standalone streaming"
	if policy != nil && policy.RejectionMessageKo != "" {
		msg = policy.RejectionMessageKo
	}
	if req.PlanTierUsed != "" {
		if (policy != nil && tierMatches(req.PlanTierUsed, policy.SyncOnlyPlanTiers)) || isBuiltInSyncOnlyTier(req.Provider, req.PlanTierUsed) {
			return nil, &domain.GateError{Code: domain.CodeSyncOnlyLicense, Message: msg}
		}
		if (policy != nil && tierMatches(req.PlanTierUsed, policy.BlockedPlanTiers)) || isBuiltInNonCommercialTier(req.PlanTierUsed) {
			return nil, &domain.GateError{Code: domain.CodeNonCommercialTier, Message: msg}
		}
	}
	explicitAllowedTier := policy != nil && req.PlanTierUsed != "" && tierMatches(req.PlanTierUsed, policy.AllowedPlanTiers)
	if policy == nil || (!policy.AllowsStandaloneStreaming && !explicitAllowedTier) {
		return nil, &domain.GateError{Code: domain.CodeSyncOnlyLicense, Message: msg}
	}
	if req.PlanTierUsed != "" && len(policy.AllowedPlanTiers) > 0 && !tierMatches(req.PlanTierUsed, policy.AllowedPlanTiers) {
		return nil, &domain.GateError{Code: domain.CodeNonCommercialTier, Message: msg}
	}
	// Provider generation cutoff (e.g. Udio 2025-10-30 download halt):
	// tracks generated after the cutoff cannot be distributed here.
	if policy.MaxGeneratedAt != nil && !policy.MaxGeneratedAt.IsZero() &&
		!req.GeneratedAt.IsZero() && req.GeneratedAt.After(*policy.MaxGeneratedAt) {
		return nil, &domain.GateError{
			Code:    domain.CodeNonCommercialTier,
			Message: fmt.Sprintf("%s generations after %s are not eligible for distribution", provider, policy.MaxGeneratedAt.Format("2006-01-02")),
		}
	}
	result := &EligibilityResult{Policy: policy}

	if req.RequiresActiveSubAtCreation {
		if req.GeneratedAt.IsZero() {
			return nil, fmt.Errorf("generated_at is required: %w", domain.ErrInvalidInput)
		}
		sub, err := store.GetSubscription(ctx, req.PartnerID, provider, req.SubscriptionID)
		if err != nil {
			return nil, err
		}
		if sub == nil {
			return nil, &domain.GateError{Code: domain.CodeNonCommercialTier, Message: "no verified subscription for provider"}
		}
		if tierMatches(sub.PlanName, policy.SyncOnlyPlanTiers) || isBuiltInSyncOnlyTier(req.Provider, sub.PlanName) {
			return nil, &domain.GateError{Code: domain.CodeSyncOnlyLicense, Message: msg}
		}
		if tierMatches(sub.PlanName, policy.BlockedPlanTiers) || isBuiltInNonCommercialTier(sub.PlanName) {
			return nil, &domain.GateError{Code: domain.CodeNonCommercialTier, Message: msg}
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
