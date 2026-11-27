package service

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/repository"
)

// Listing-metadata bounds for creator edits.
const (
	studioGenreMax    = 40
	studioMoodMax     = 32
	studioMoodsMax    = 8
	studioLyricsMax   = 10000
	creatorWithdrawKo = "크리에이터 요청으로 삭제된 음원입니다."
)

// StudioTrackService lets a creator manage their own tracks after upload:
// edit discovery metadata and withdraw (delete) a track. Title, AI
// provenance and audio are certificate/licence evidence and stay immutable.
type StudioTrackService struct {
	repos    *repository.Repos
	disputes *DisputeService
}

// NewStudioTrackService builds the service; withdrawals reuse the takedown
// path (status flip + object purge).
func NewStudioTrackService(repos *repository.Repos, disputes *DisputeService) *StudioTrackService {
	return &StudioTrackService{repos: repos, disputes: disputes}
}

// TrackListingPatch is the editable subset. Nil fields stay unchanged.
type TrackListingPatch struct {
	Genre          *string
	MoodTags       *[]string
	Lyrics         *string
	IsPromptPublic *bool
}

func (s *StudioTrackService) ownedTrack(ctx context.Context, partnerID, trackID string) (*domain.Track, error) {
	t, err := s.repos.Tracks.GetByID(ctx, trackID)
	if err != nil {
		return nil, err
	}
	if t.PartnerID != partnerID {
		return nil, domain.ErrForbidden
	}
	return t, nil
}

func normalizeListingPatch(p TrackListingPatch) (TrackListingPatch, error) {
	if p.Genre != nil {
		g := strings.TrimSpace(*p.Genre)
		if utf8.RuneCountInString(g) > studioGenreMax {
			return p, fmt.Errorf("%w: genre too long", domain.ErrInvalidInput)
		}
		p.Genre = &g
	}
	if p.MoodTags != nil {
		seen := map[string]bool{}
		out := []string{}
		for _, m := range *p.MoodTags {
			m = strings.TrimSpace(m)
			if m == "" || seen[strings.ToLower(m)] {
				continue
			}
			if utf8.RuneCountInString(m) > studioMoodMax {
				return p, fmt.Errorf("%w: mood tag too long", domain.ErrInvalidInput)
			}
			seen[strings.ToLower(m)] = true
			out = append(out, m)
		}
		if len(out) > studioMoodsMax {
			return p, fmt.Errorf("%w: too many mood tags", domain.ErrInvalidInput)
		}
		p.MoodTags = &out
	}
	if p.Lyrics != nil {
		l := strings.TrimSpace(*p.Lyrics)
		if utf8.RuneCountInString(l) > studioLyricsMax {
			return p, fmt.Errorf("%w: lyrics too long", domain.ErrInvalidInput)
		}
		p.Lyrics = &l
	}
	return p, nil
}

// UpdateListing applies a metadata patch to the creator's own track.
func (s *StudioTrackService) UpdateListing(ctx context.Context, partnerID, trackID string, patch TrackListingPatch) (*domain.Track, error) {
	patch, err := normalizeListingPatch(patch)
	if err != nil {
		return nil, err
	}
	t, err := s.ownedTrack(ctx, partnerID, trackID)
	if err != nil {
		return nil, err
	}
	if t.Status == domain.TrackTakenDown {
		return nil, domain.ErrNotStreamable
	}
	genre, moods, lyrics, promptPublic := t.Genre, t.MoodTags, t.Lyrics, t.IsPromptPublic
	if patch.Genre != nil {
		genre = emptyToNil(*patch.Genre)
	}
	if patch.MoodTags != nil {
		moods = *patch.MoodTags
	}
	if patch.Lyrics != nil {
		lyrics = emptyToNil(*patch.Lyrics)
	}
	if patch.IsPromptPublic != nil {
		promptPublic = *patch.IsPromptPublic
	}
	if err := s.repos.Tracks.UpdateListingMetadata(ctx, trackID, genre, moods, lyrics, promptPublic); err != nil {
		return nil, err
	}
	return s.repos.Tracks.GetByID(ctx, trackID)
}

// Withdraw permanently removes the creator's own track from the service:
// status TAKEN_DOWN with a creator-withdrawal reason and the audio objects
// purged. Play history and settlements keep their rows.
func (s *StudioTrackService) Withdraw(ctx context.Context, partnerID, trackID string) (*domain.Track, error) {
	t, err := s.ownedTrack(ctx, partnerID, trackID)
	if err != nil {
		return nil, err
	}
	if t.Status == domain.TrackTakenDown {
		return t, nil
	}
	return s.disputes.Takedown(ctx, trackID, creatorWithdrawKo)
}

func emptyToNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
