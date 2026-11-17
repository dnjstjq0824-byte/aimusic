package handler

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/service"
)

// EngagementHandler serves likes, playlist saves, and follows (R6).
type EngagementHandler struct {
	Engagement *service.EngagementService
}

func validUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

func badID(c *gin.Context) {
	AbortError(c, fmt.Errorf("%w: id must be a uuid", domain.ErrInvalidInput))
}

// LikeTrack likes a track (idempotent).
//
//	POST /likes/tracks/:id
func (h *EngagementHandler) LikeTrack(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		badID(c)
		return
	}
	added, err := h.Engagement.LikeTrack(c.Request.Context(), CurrentUserID(c), id)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"liked": true, "added": added})
}

// UnlikeTrack removes a like (idempotent).
//
//	DELETE /likes/tracks/:id
func (h *EngagementHandler) UnlikeTrack(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		badID(c)
		return
	}
	removed, err := h.Engagement.UnlikeTrack(c.Request.Context(), CurrentUserID(c), id)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"liked": false, "removed": removed})
}

// LikedState reports whether the caller likes a track.
//
//	GET /likes/tracks/:id
func (h *EngagementHandler) LikedState(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		badID(c)
		return
	}
	liked, err := h.Engagement.IsLiked(c.Request.Context(), CurrentUserID(c), id)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"liked": liked})
}

// LikedTracks lists the caller's liked tracks.
//
//	GET /likes/tracks?limit=
func (h *EngagementHandler) LikedTracks(c *gin.Context) {
	items, err := h.Engagement.LikedTracks(c.Request.Context(), CurrentUserID(c), queryInt(c, "limit", 50))
	if err != nil {
		AbortError(c, err)
		return
	}
	out := make([]*domain.Track, 0, len(items))
	for _, t := range items {
		out = append(out, t.PublicView())
	}
	okList(c, out)
}

// SavePlaylist saves a playlist (idempotent).
//
//	POST /playlists/:id/save
func (h *EngagementHandler) SavePlaylist(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		badID(c)
		return
	}
	added, err := h.Engagement.SavePlaylist(c.Request.Context(), CurrentUserID(c), id)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"saved": true, "added": added})
}

// UnsavePlaylist removes a save (idempotent).
//
//	DELETE /playlists/:id/save
func (h *EngagementHandler) UnsavePlaylist(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		badID(c)
		return
	}
	removed, err := h.Engagement.UnsavePlaylist(c.Request.Context(), CurrentUserID(c), id)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"saved": false, "removed": removed})
}

// SaveState reports whether the caller saved a playlist.
//
//	GET /playlists/:id/save
func (h *EngagementHandler) SaveState(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		badID(c)
		return
	}
	saved, err := h.Engagement.IsSaved(c.Request.Context(), CurrentUserID(c), id)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"saved": saved})
}

// SavedPlaylists lists the caller's saved playlists.
//
//	GET /saves/playlists?limit=
func (h *EngagementHandler) SavedPlaylists(c *gin.Context) {
	items, err := h.Engagement.SavedPlaylists(c.Request.Context(), CurrentUserID(c), queryInt(c, "limit", 50))
	if err != nil {
		AbortError(c, err)
		return
	}
	okList(c, items)
}

type followRequest struct {
	TargetType string `json:"targetType"`
	TargetID   string `json:"targetId"`
}

// Follow adds a creator/playlist follow (idempotent).
//
//	POST /follows {"targetType":"creator|playlist","targetId":"uuid"}
func (h *EngagementHandler) Follow(c *gin.Context) {
	var req followRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortError(c, fmt.Errorf("%w: targetType and targetId required", domain.ErrInvalidInput))
		return
	}
	if (req.TargetType != "creator" && req.TargetType != "playlist") || !validUUID(req.TargetID) {
		AbortError(c, fmt.Errorf("%w: targetType must be creator|playlist with a uuid targetId", domain.ErrInvalidInput))
		return
	}
	added, err := h.Engagement.Follow(c.Request.Context(), CurrentUserID(c), req.TargetType, req.TargetID)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"following": true, "added": added})
}

// Unfollow removes a follow (idempotent).
//
//	DELETE /follows/:type/:id
func (h *EngagementHandler) Unfollow(c *gin.Context) {
	targetType, targetID := c.Param("type"), c.Param("id")
	if (targetType != "creator" && targetType != "playlist") || !validUUID(targetID) {
		AbortError(c, fmt.Errorf("%w: type must be creator|playlist with a uuid id", domain.ErrInvalidInput))
		return
	}
	removed, err := h.Engagement.Unfollow(c.Request.Context(), CurrentUserID(c), targetType, targetID)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"following": false, "removed": removed})
}

// ListFollows lists the caller's follows (optional ?type=).
//
//	GET /follows?type=&limit=
func (h *EngagementHandler) ListFollows(c *gin.Context) {
	items, err := h.Engagement.ListFollows(c.Request.Context(), CurrentUserID(c), strings.TrimSpace(c.Query("type")))
	if err != nil {
		AbortError(c, err)
		return
	}
	limit := queryInt(c, "limit", 50)
	if limit >= 0 && len(items) > limit {
		items = items[:limit]
	}
	okList(c, items)
}
