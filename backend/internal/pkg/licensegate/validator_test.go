package licensegate

import (
	"context"
	"errors"
	"testing"
	"time"

	"ai-music-platform/backend/internal/domain"
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
	allowed := &Policy{Provider: "SUNO", AllowsStandaloneStreaming: true}
	blocked := &Policy{Provider: "MUBERT", AllowsStandaloneStreaming: false, RejectionMessageKo: "차단"}
	inWindow := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	beforeWindow := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name    string
		store   *stubStore
		req     PreCheckRequest
		wantErr string
	}{
		{
			name:  "pass",
			store: &stubStore{policy: allowed, sub: verifiedSub()},
			req:   PreCheckRequest{PartnerID: "partner-1", Provider: "SUNO", AudioSHA256: "aaa", GeneratedAt: inWindow, SubscriptionID: "sub-1", RequiresActiveSubAtCreation: true},
		},
		{
			name:    "duplicate",
			store:   &stubStore{hashExists: true, policy: allowed, sub: verifiedSub()},
			req:     PreCheckRequest{PartnerID: "partner-1", Provider: "SUNO", AudioSHA256: "aaa", GeneratedAt: inWindow, RequiresActiveSubAtCreation: true},
			wantErr: "sha256 already registered",
		},
		{
			name:    "sync only blocked",
			store:   &stubStore{policy: blocked, sub: verifiedSub()},
			req:     PreCheckRequest{PartnerID: "partner-1", Provider: "MUBERT", AudioSHA256: "aaa", GeneratedAt: inWindow, RequiresActiveSubAtCreation: true},
			wantErr: "차단",
		},
		{
			name:    "retroactive generation blocked",
			store:   &stubStore{policy: allowed, sub: verifiedSub()},
			req:     PreCheckRequest{PartnerID: "partner-1", Provider: "SUNO", AudioSHA256: "aaa", GeneratedAt: beforeWindow, SubscriptionID: "sub-1", RequiresActiveSubAtCreation: true},
			wantErr: "generated before subscription window",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := ValidateCommercialEligibility(ctx, tc.store, tc.req)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res == nil || res.Policy == nil || res.Subscription == nil {
					t.Fatal("expected policy + subscription in result")
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
		})
	}
}
