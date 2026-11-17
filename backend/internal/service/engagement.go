package service

import (
	"context"

	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/repository"
)

// EngagementService owns likes, playlist saves, and follows (R6).
type EngagementService struct {
	repos *repository.Repos
}

func (s *EngagementService) LikeTrack(ctx context.Context, userID, trackID string) (bool, error) {
	if _, err := s.repos.Tracks.GetByID(ctx, trackID); err != nil {
		return false, err
	}
	return s.repos.Engagement.LikeTrack(ctx, userID, trackID)
}

func (s *EngagementService) UnlikeTrack(ctx context.Context, userID, trackID string) (bool, error) {
	return s.repos.Engagement.UnlikeTrack(ctx, userID, trackID)
}

func (s *EngagementService) IsLiked(ctx context.Context, userID, trackID string) (bool, error) {
	return s.repos.Engagement.IsLiked(ctx, userID, trackID)
}

func (s *EngagementService) LikedTracks(ctx context.Context, userID string, limit int) ([]*domain.Track, error) {
	return s.repos.Tracks.LikedTracks(ctx, userID, limit)
}

func (s *EngagementService) SavePlaylist(ctx context.Context, userID, playlistID string) (bool, error) {
	if err := s.requireVisiblePlaylist(ctx, userID, playlistID); err != nil {
		return false, err
	}
	return s.repos.Engagement.SavePlaylist(ctx, userID, playlistID)
}

func (s *EngagementService) UnsavePlaylist(ctx context.Context, userID, playlistID string) (bool, error) {
	return s.repos.Engagement.UnsavePlaylist(ctx, userID, playlistID)
}

func (s *EngagementService) IsSaved(ctx context.Context, userID, playlistID string) (bool, error) {
	return s.repos.Engagement.IsSaved(ctx, userID, playlistID)
}

func (s *EngagementService) SavedPlaylists(ctx context.Context, userID string, limit int) ([]*domain.Playlist, error) {
	return s.repos.Playlists.SavedPlaylists(ctx, userID, limit)
}

func (s *EngagementService) Follow(ctx context.Context, userID, targetType, targetID string) (bool, error) {
	switch targetType {
	case "creator":
		if _, err := s.repos.Partners.GetByID(ctx, targetID); err != nil {
			return false, err
		}
	case "playlist":
		if err := s.requireVisiblePlaylist(ctx, userID, targetID); err != nil {
			return false, err
		}
	default:
		return false, domain.ErrInvalidInput
	}
	return s.repos.Engagement.Follow(ctx, userID, targetType, targetID)
}

func (s *EngagementService) Unfollow(ctx context.Context, userID, targetType, targetID string) (bool, error) {
	if targetType != "creator" && targetType != "playlist" {
		return false, domain.ErrInvalidInput
	}
	return s.repos.Engagement.Unfollow(ctx, userID, targetType, targetID)
}

func (s *EngagementService) ListFollows(ctx context.Context, userID, targetType string) ([]*domain.Follow, error) {
	if targetType != "" && targetType != "creator" && targetType != "playlist" {
		return nil, domain.ErrInvalidInput
	}
	return s.repos.Engagement.ListFollows(ctx, userID, targetType)
}

func (s *EngagementService) requireVisiblePlaylist(ctx context.Context, userID, playlistID string) error {
	p, err := s.repos.Playlists.GetByID(ctx, playlistID)
	if err != nil {
		return err
	}
	if p.OwnerUserID != userID && !p.IsPublic && !p.IsOfficialCuration {
		return domain.ErrForbidden
	}
	return nil
}
