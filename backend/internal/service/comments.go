package service

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/repository"
)

// CommentService handles listener comments on READY tracks.
type CommentService struct {
	repos *repository.Repos
}

// NewCommentService builds the comment service.
func NewCommentService(repos *repository.Repos) *CommentService {
	return &CommentService{repos: repos}
}

// CommentPage is one newest-first page plus the total visible count.
type CommentPage struct {
	Items []*domain.TrackComment `json:"items"`
	Total int                    `json:"total"`
	// NextBefore is the cursor for the next (older) page; empty at the end.
	NextBefore string `json:"nextBefore,omitempty"`
}

// normalizeCommentBody trims and bounds a comment body.
func normalizeCommentBody(body string) (string, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", fmt.Errorf("%w: comment is empty", domain.ErrInvalidInput)
	}
	if utf8.RuneCountInString(body) > domain.TrackCommentMax {
		return "", fmt.Errorf("%w: comment too long", domain.ErrInvalidInput)
	}
	return body, nil
}

// readyTrack resolves a READY track (comments exist only on public tracks).
func (s *CommentService) readyTrack(ctx context.Context, trackID string) (*domain.Track, error) {
	t, err := s.repos.Tracks.GetByID(ctx, trackID)
	if err != nil {
		return nil, err
	}
	if t.Status != domain.TrackReady {
		return nil, domain.ErrNotFound
	}
	return t, nil
}

// List returns a newest-first page. before is an RFC3339Nano cursor.
func (s *CommentService) List(ctx context.Context, trackID, before string, limit int) (*CommentPage, error) {
	if _, err := s.readyTrack(ctx, trackID); err != nil {
		return nil, err
	}
	var cursor time.Time
	if before != "" {
		t, err := time.Parse(time.RFC3339Nano, before)
		if err != nil {
			return nil, fmt.Errorf("%w: before", domain.ErrInvalidInput)
		}
		cursor = t
	}
	items, err := s.repos.Comments.ListByTrack(ctx, trackID, cursor, limit)
	if err != nil {
		return nil, err
	}
	total, err := s.repos.Comments.CountByTrack(ctx, trackID)
	if err != nil {
		return nil, err
	}
	page := &CommentPage{Items: items, Total: total}
	if n := len(items); n > 0 && n >= effectiveCommentLimit(limit) {
		page.NextBefore = items[n-1].CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return page, nil
}

func effectiveCommentLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 50 {
		return 50
	}
	return limit
}

// Create posts a comment on a READY track.
func (s *CommentService) Create(ctx context.Context, userID, trackID, body string) (*domain.TrackComment, error) {
	body, err := normalizeCommentBody(body)
	if err != nil {
		return nil, err
	}
	if _, err := s.readyTrack(ctx, trackID); err != nil {
		return nil, err
	}
	return s.repos.Comments.Create(ctx, trackID, userID, body)
}

// Delete removes a comment; its author or an admin only.
func (s *CommentService) Delete(ctx context.Context, userID string, isAdmin bool, trackID, commentID string) error {
	c, err := s.repos.Comments.GetByID(ctx, commentID)
	if err != nil {
		return err
	}
	if c.TrackID != trackID {
		return domain.ErrNotFound
	}
	if c.UserID != userID && !isAdmin {
		return domain.ErrForbidden
	}
	return s.repos.Comments.Delete(ctx, commentID)
}
