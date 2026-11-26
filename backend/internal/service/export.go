package service

import (
	"context"
	"errors"
	"time"

	"avibe-platform/backend/internal/domain"
)

// UserDataExport is the self-service personal-data bundle (privacy policy:
// access/portability). Secrets (password hash, TOTP secret, encrypted payout
// account) are never included.
type UserDataExport struct {
	Profile        *domain.User                `json:"profile"`
	Consents       []*domain.UserTermConsent   `json:"consents"`
	Playlists      []*ExportPlaylist           `json:"playlists"`
	PlaybackStates []*domain.UserPlaybackState `json:"playbackStates"`
	StreamingLogs  []*domain.StreamingLog      `json:"streamingLogs"`
	DisputesFiled  []*domain.CopyrightDispute  `json:"disputesFiled"`
	Blocks         []*domain.UserBlock         `json:"blocks"`
	ArtistBlocks   []*domain.UserArtistBlock   `json:"artistBlocks"`
	Orders         []*domain.BillingOrder      `json:"orders"`
	Subscription   *domain.BillingSubscription `json:"subscription,omitempty"`
	PartnerProfile *domain.PartnerProfile      `json:"partnerProfile,omitempty"`
	PartnerTracks  []*domain.Track             `json:"partnerTracks,omitempty"`
	ExportedAt     string                      `json:"exportedAt"`
	// Warnings names sections dropped by transient failures, so a partial
	// bundle is never mistaken for a complete one.
	Warnings []string `json:"warnings,omitempty"`
}

// ExportPlaylist pairs a playlist with its track IDs.
type ExportPlaylist struct {
	Playlist *domain.Playlist        `json:"playlist"`
	Tracks   []*domain.PlaylistTrack `json:"tracks"`
}

// ExportData assembles the caller's personal-data bundle.
func (s *AuthService) ExportData(ctx context.Context, userID string) (*UserDataExport, error) {
	if userID == "" {
		return nil, domain.ErrInvalidInput
	}
	u, err := s.repos.Users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	u.PasswordHash = ""
	u.TOTPSecret = ""
	out := &UserDataExport{
		Profile:        u,
		Consents:       []*domain.UserTermConsent{},
		Playlists:      []*ExportPlaylist{},
		PlaybackStates: []*domain.UserPlaybackState{},
		StreamingLogs:  []*domain.StreamingLog{},
		DisputesFiled:  []*domain.CopyrightDispute{},
		Blocks:         []*domain.UserBlock{},
		ArtistBlocks:   []*domain.UserArtistBlock{},
		Orders:         []*domain.BillingOrder{},
		ExportedAt:     time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	}
	warn := func(section string) { out.Warnings = append(out.Warnings, section) }
	if v, err := s.repos.Legal.ListConsentsByUser(ctx, userID); err == nil {
		out.Consents = v
	} else {
		warn("consents")
	}
	if pls, err := s.repos.Playlists.ListByOwner(ctx, userID); err == nil {
		for _, p := range pls {
			ep := &ExportPlaylist{Playlist: p, Tracks: []*domain.PlaylistTrack{}}
			if ts, err := s.repos.Playlists.ListTracks(ctx, p.ID); err == nil {
				ep.Tracks = ts
			} else {
				warn("playlists:" + p.ID)
			}
			out.Playlists = append(out.Playlists, ep)
		}
	} else {
		warn("playlists")
	}
	if v, err := s.repos.Playback.ListStatesByUser(ctx, userID); err == nil {
		out.PlaybackStates = v
	} else {
		warn("playbackStates")
	}
	if v, err := s.repos.Streaming.ListByUser(ctx, userID, 1000); err == nil {
		out.StreamingLogs = v
	} else {
		warn("streamingLogs")
	}
	if v, err := s.repos.Disputes.ListByReporter(ctx, userID, 200); err == nil {
		out.DisputesFiled = v
	} else {
		warn("disputesFiled")
	}
	if v, err := s.repos.Blocks.ListByUser(ctx, userID); err == nil {
		out.Blocks = v
	} else {
		warn("blocks")
	}
	if v, err := s.repos.Blocks.ListBlockedArtists(ctx, userID); err == nil {
		out.ArtistBlocks = v
	} else {
		warn("artistBlocks")
	}
	if v, err := s.repos.Billing.ListOrdersByUser(ctx, userID, 200); err == nil {
		out.Orders = v
	} else {
		warn("orders")
	}
	if sub, err := s.repos.Billing.GetActiveSubscription(ctx, userID); err == nil {
		out.Subscription = sub
	} else if !errors.Is(err, domain.ErrNotFound) {
		warn("subscription")
	}
	if prof, err := s.repos.Partners.GetByUserID(ctx, userID); err == nil {
		out.PartnerProfile = prof
		if ts, err := s.repos.Tracks.ListByPartner(ctx, prof.ID, 500, 0); err == nil {
			out.PartnerTracks = ts
		} else {
			warn("partnerTracks")
		}
	} else if !errors.Is(err, domain.ErrNotFound) {
		warn("partnerProfile")
	}
	return out, nil
}
