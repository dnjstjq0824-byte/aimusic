package repository

import (
	"context"
	"database/sql"

	"ai-music-platform/backend/internal/pkg/licensegate"
)

// GateStore implements licensegate.Store against PostgreSQL.
type GateStore struct{ db *sql.DB }

// AudioHashExists reuses tracks.original_object_key, which embeds the
// client-declared SHA-256 in staging keys.
func (s *GateStore) AudioHashExists(ctx context.Context, sha256hex string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM tracks WHERE original_object_key LIKE '%' || $1 || '%')`,
		sha256hex).Scan(&exists)
	return exists, err
}

// GetPolicy loads the license-gate view of a provider policy.
func (s *GateStore) GetPolicy(ctx context.Context, provider string) (*licensegate.Policy, error) {
	p := &licensegate.Policy{}
	var joined string
	err := s.db.QueryRowContext(ctx, `
		SELECT provider, allows_standalone_streaming, min_bitrate_kbps,
			COALESCE(array_to_string(blocked_metadata_keywords, chr(31)), ''),
			rejection_message_ko
		FROM ai_provider_policies WHERE provider = $1`, provider,
	).Scan(&p.Provider, &p.AllowsStandaloneStreaming, &p.MinBitrateKbps, &joined, &p.RejectionMessageKo)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.BlockedMetadataKeywords = splitKeywords(joined)
	return p, nil
}

// GetSubscription loads the license-gate view of a subscription by ID,
// or by partner+provider when subscriptionID is empty.
func (s *GateStore) GetSubscription(ctx context.Context, partnerID, provider, subscriptionID string) (*licensegate.Subscription, error) {
	sub := &licensegate.Subscription{}
	var verifiedAt, expiresAt sql.NullTime
	var row *sql.Row
	if subscriptionID != "" {
		row = s.db.QueryRowContext(ctx, `
			SELECT id::text, partner_id::text, provider, plan_name, status,
				created_at, verified_at, expires_at
			FROM partner_ai_subscriptions WHERE id = $1::uuid`, subscriptionID)
	} else {
		row = s.db.QueryRowContext(ctx, `
			SELECT id::text, partner_id::text, provider, plan_name, status,
				created_at, verified_at, expires_at
			FROM partner_ai_subscriptions WHERE partner_id = $1::uuid AND provider = $2`, partnerID, provider)
	}
	err := row.Scan(&sub.ID, &sub.PartnerID, &sub.Provider, &sub.PlanName,
		&sub.Status, &sub.CreatedAt, &verifiedAt, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sub.VerifiedAt = timePtr(verifiedAt)
	sub.ExpiresAt = timePtr(expiresAt)
	return sub, nil
}

var _ licensegate.Store = (*GateStore)(nil)
