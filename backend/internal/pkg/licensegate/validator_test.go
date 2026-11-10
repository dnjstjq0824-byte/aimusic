package licensegate

import (
	"context"
	"errors"
	"testing"
	"time"

	"avibe-platform/backend/internal/domain"
)

func TestStagingKeyRoundTrip(t *testing.T) {
	key := BuildStagingKey("partner-1", "ABCDEF0123", "song.wav")
	partner, sha, name, ok := ParseStagingKey(key)
	if !ok {
		t.Fatalf("canonical key rejected: %q", key)
	}
	if partner != "partner-1" || sha != "abcdef0123" || name != "song.wav" {
		t.Fatalf("parse mismatch: %q %q %q", partner, sha, name)
	}
}

func TestStagingKeyRejectsNonCanonical(t *testing.T) {
	bad := []string{
		"",
		"staging/only-two",
		"other/p/s/f.wav",
		"staging//sha/f.wav",
		"staging/p//f.wav",
		"staging/p/sha/",
		"staging/p/sha/a/b.wav",
	}
	for _, key := range bad {
		if _, _, _, ok := ParseStagingKey(key); ok {
			t.Fatalf("non-canonical key accepted: %q", key)
		}
	}
}

// stubStore is an offline licensegate.Store for validator tests.
type stubStore struct {
	hashExists bool
	policy     *Policy
	sub        *Subscription
}

func (s *stubStore) AudioHashExists(_ context.Context, _ string) (bool, error) {
	return s.hashExists, nil
}

func (s *stubStore) GetPolicy(_ context.Context, _ string) (*Policy, error) {
	return s.policy, nil
}

func (s *stubStore) GetSubscription(_ context.Context, _, _, _ string) (*Subscription, error) {
	return s.sub, nil
}

func verifiedSub() *Subscription {
	verified := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	expires := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	return &Subscription{
		ID: "sub-1", PartnerID: "partner-1", Provider: "SUNO", PlanName: "Pro",
		Status: domain.SubVerified, CreatedAt: verified, VerifiedAt: &verified, ExpiresAt: &expires,
	}
}

func TestValidateCommercialEligibility(t *testing.T) {
	ctx := context.Background()
	allowed := &Policy{
		Provider:                  "SUNO",
		AllowsStandaloneStreaming: true,
		AllowedPlanTiers:          []string{"Pro", "Premier"},
		BlockedPlanTiers:          []string{"Free", "Basic"},
	}
	aivaPolicy := &Policy{
		Provider:                  "AIVA",
		AllowsStandaloneStreaming: true,
		AllowedPlanTiers:          []string{"Pro"},
		BlockedPlanTiers:          []string{"Free"},
		SyncOnlyPlanTiers:         []string{"Standard"},
		RejectionMessageKo:        "AIVA Standard는 Sync 전용입니다.",
	}
	blocked := &Policy{Provider: "MUBERT", AllowsStandaloneStreaming: false, RejectionMessageKo: "차단"}
	udioCutoff := time.Date(2025, 10, 30, 0, 0, 0, 0, time.UTC)
	udioPolicy := &Policy{
		Provider:                  "UDIO",
		AllowsStandaloneStreaming: true,
		AllowedPlanTiers:          []string{"Standard", "Pro"},
		MaxGeneratedAt:            &udioCutoff,
	}
	inWindow := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	beforeWindow := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	afterCutoff := time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC)
	beforeCutoff := time.Date(2025, 10, 29, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name     string
		store    *stubStore
		req      PreCheckRequest
		wantErr  string
		wantCode string
	}{
		{
			name:  "pass",
			store: &stubStore{policy: allowed, sub: verifiedSub()},
			req:   PreCheckRequest{PartnerID: "partner-1", Provider: "SUNO", PlanTierUsed: "Pro", AudioSHA256: "aaa", GeneratedAt: inWindow, SubscriptionID: "sub-1", RequiresActiveSubAtCreation: true},
		},
		{
			name:    "duplicate",
			store:   &stubStore{hashExists: true, policy: allowed, sub: verifiedSub()},
			req:     PreCheckRequest{PartnerID: "partner-1", Provider: "SUNO", AudioSHA256: "aaa", GeneratedAt: inWindow, RequiresActiveSubAtCreation: true},
			wantErr: "sha256 already registered",
		},
		{
			name:     "suno free tier blocked",
			store:    &stubStore{policy: allowed, sub: verifiedSub()},
			req:      PreCheckRequest{PartnerID: "partner-1", Provider: "SUNO", PlanTierUsed: "Free", AudioSHA256: "aaa", GeneratedAt: inWindow, SubscriptionID: "sub-1", RequiresActiveSubAtCreation: true},
			wantErr:  "ERR_NON_COMMERCIAL_TIER",
			wantCode: domain.CodeNonCommercialTier,
		},
		{
			name:     "aiva standard sync only blocked",
			store:    &stubStore{policy: aivaPolicy, sub: verifiedSub()},
			req:      PreCheckRequest{PartnerID: "partner-1", Provider: "AIVA", PlanTierUsed: "Standard", AudioSHA256: "aaa", GeneratedAt: inWindow, SubscriptionID: "sub-1", RequiresActiveSubAtCreation: true},
			wantErr:  "AIVA Standard는 Sync 전용입니다.",
			wantCode: domain.CodeSyncOnlyLicense,
		},
		{
			name:     "sync only blocked",
			store:    &stubStore{policy: blocked, sub: verifiedSub()},
			req:      PreCheckRequest{PartnerID: "partner-1", Provider: "MUBERT", AudioSHA256: "aaa", GeneratedAt: inWindow, RequiresActiveSubAtCreation: true},
			wantErr:  "차단",
			wantCode: domain.CodeSyncOnlyLicense,
		},
		{
			name:     "retroactive generation blocked",
			store:    &stubStore{policy: allowed, sub: verifiedSub()},
			req:      PreCheckRequest{PartnerID: "partner-1", Provider: "SUNO", AudioSHA256: "aaa", GeneratedAt: beforeWindow, SubscriptionID: "sub-1", RequiresActiveSubAtCreation: true},
			wantErr:  "generated before subscription window",
			wantCode: domain.CodeRetroactiveGenerationDate,
		},
		{
			name:     "udio post-cutoff blocked",
			store:    &stubStore{policy: udioPolicy, sub: verifiedSub()},
			req:      PreCheckRequest{PartnerID: "partner-1", Provider: "UDIO", AudioSHA256: "aaa", GeneratedAt: afterCutoff, RequiresActiveSubAtCreation: false},
			wantErr:  "not eligible for distribution",
			wantCode: domain.CodeNonCommercialTier,
		},
		{
			name:  "udio pre-cutoff passes",
			store: &stubStore{policy: udioPolicy, sub: verifiedSub()},
			req:   PreCheckRequest{PartnerID: "partner-1", Provider: "UDIO", AudioSHA256: "aaa", GeneratedAt: beforeCutoff, RequiresActiveSubAtCreation: false},
		},
		{
			name:     "unknown provider allowlisted-out",
			store:    &stubStore{policy: nil, sub: verifiedSub()},
			req:      PreCheckRequest{PartnerID: "partner-1", Provider: "MUSICGEN", AudioSHA256: "aaa", GeneratedAt: inWindow, RequiresActiveSubAtCreation: false},
			wantErr:  "지원하지 않는 AI 제공자",
			wantCode: domain.CodeProviderNotAllowlisted,
		},
		{
			name:     "valid provider without policy row",
			store:    &stubStore{policy: nil, sub: verifiedSub()},
			req:      PreCheckRequest{PartnerID: "partner-1", Provider: "SUNO", AudioSHA256: "aaa", GeneratedAt: inWindow, RequiresActiveSubAtCreation: false},
			wantErr:  "정책이 등록되어 있지 않습니다",
			wantCode: domain.CodeProviderNotAllowlisted,
		},
		{
			name:  "lowercase provider normalized",
			store: &stubStore{policy: allowed, sub: verifiedSub()},
			req:   PreCheckRequest{PartnerID: "partner-1", Provider: "suno", PlanTierUsed: "Pro", AudioSHA256: "aaa", GeneratedAt: inWindow, SubscriptionID: "sub-1", RequiresActiveSubAtCreation: true},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := ValidateCommercialEligibility(ctx, tc.store, tc.req)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res == nil || res.Policy == nil {
					t.Fatal("expected policy in result")
				}
				if tc.req.RequiresActiveSubAtCreation && res.Subscription == nil {
					t.Fatal("expected subscription in result")
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q", tc.wantErr)
			}
			var gateErr *domain.GateError
			if !errors.As(err, &gateErr) && tc.name != "duplicate" {
				t.Fatalf("expected GateError, got %T: %v", err, err)
			}
			if tc.wantCode != "" && gateErr != nil && gateErr.Code != tc.wantCode {
				t.Fatalf("want code %q, got %q", tc.wantCode, gateErr.Code)
			}
		})
	}
}

func TestValidateSourceModel(t *testing.T) {
	blocked := []string{"MusicGen", "musicgen-small", "AudioGen", "MUSICLM", "stable-audio-open"}
	for _, m := range blocked {
		if err := ValidateSourceModel(m); err == nil {
			t.Errorf("non-commercial model %q accepted", m)
		} else {
			var gateErr *domain.GateError
			if !errors.As(err, &gateErr) || gateErr.Code != domain.CodeNonCommercialTier {
				t.Errorf("model %q: want ERR_NON_COMMERCIAL_TIER GateError, got %v", m, err)
			}
		}
	}
	allowed := []string{"", "suno-v4", "udio-v1.5", "chirp-v3", "ace-step-v1"}
	for _, m := range allowed {
		if err := ValidateSourceModel(m); err != nil {
			t.Errorf("model %q rejected: %v", m, err)
		}
	}
}

func TestBuildCertSnapshotRoundTrip(t *testing.T) {
	gen := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	tier := "PRO"
	sub := "sub-123"
	model := "suno-v4"
	raw, err := BuildCertSnapshot(CertSnapshot{
		Provider:            "SUNO",
		PlanTierUsed:        &tier,
		SubscriptionID:      &sub,
		GeneratedAt:         &gen,
		SourceModel:         &model,
		CollectiveTrustStatus: domain.CollectiveNone,
		Gate1PolicyPassed:   true,
		Gate2DatePassed:     true,
	})
	if err != nil {
		t.Fatalf("BuildCertSnapshot: %v", err)
	}
	got, err := ParseCertSnapshot(raw)
	if err != nil {
		t.Fatalf("ParseCertSnapshot: %v", err)
	}
	if got.PolicyVersion == "" || got.PolicyVersion != PolicyVersion {
		t.Errorf("policy version = %q, want pinned %q", got.PolicyVersion, PolicyVersion)
	}
	if got.Provider != "SUNO" || got.PlanTierUsed == nil || *got.PlanTierUsed != "PRO" {
		t.Errorf("round trip lost provider/tier: %+v", got)
	}
	if got.SubscriptionID == nil || *got.SubscriptionID != "sub-123" {
		t.Errorf("round trip lost subscription: %+v", got)
	}
	if got.SourceModel == nil || *got.SourceModel != "suno-v4" {
		t.Errorf("round trip lost source model: %+v", got)
	}
	if got.GeneratedAt == nil || !got.GeneratedAt.Equal(gen) {
		t.Errorf("round trip lost generatedAt: %+v", got)
	}
	if !got.Gate1PolicyPassed || !got.Gate2DatePassed {
		t.Errorf("round trip lost gate flags: %+v", got)
	}
	if got.DecidedAt.IsZero() {
		t.Error("DecidedAt must default to build time")
	}
}

func TestBuildCertSnapshotRequiresProvider(t *testing.T) {
	if _, err := BuildCertSnapshot(CertSnapshot{}); err == nil {
		t.Error("empty provider must fail")
	}
	if _, err := ParseCertSnapshot("not-json"); err == nil {
		t.Error("malformed snapshot must fail")
	}
}
