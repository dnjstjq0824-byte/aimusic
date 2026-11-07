package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"avibe-platform/backend/internal/service"
)

// PlaylistHandler serves playlist CRUD, entries, and reorder.
type PlaylistHandler struct {
	Playlists *service.PlaylistService
}

type playlistRequest struct {
	Title    string `json:"title" binding:"required"`
	IsPublic bool   `json:"isPublic"`
}

// Create makes a playlist for the caller.
func (h *PlaylistHandler) Create(c *gin.Context) {
	var req playlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	p, err := h.Playlists.Create(c.Request.Context(), CurrentUserID(c), req.Title, req.IsPublic)
	if err != nil {
		AbortError(c, err)
		return
	}
	created(c, p)
}

// ListMine returns the caller's playlists.
func (h *PlaylistHandler) ListMine(c *gin.Context) {
	playlists, err := h.Playlists.ListMine(c.Request.Context(), CurrentUserID(c))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, playlists)
}

// Get returns one visible playlist.
func (h *PlaylistHandler) Get(c *gin.Context) {
	p, err := h.Playlists.Get(c.Request.Context(), CurrentUserID(c), c.Param("id"))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, p)
}

// Update renames a playlist; owner only.
func (h *PlaylistHandler) Update(c *gin.Context) {
	var req playlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	p, err := h.Playlists.Update(c.Request.Context(), CurrentUserID(c), c.Param("id"), req.Title, req.IsPublic)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, p)
}

// Delete removes a playlist; owner only.
func (h *PlaylistHandler) Delete(c *gin.Context) {
	if err := h.Playlists.Delete(c.Request.Context(), CurrentUserID(c), c.Param("id")); err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}

type addTrackRequest struct {
	TrackID string `json:"trackId" binding:"required"`
}

// AddTrack appends a READY track; owner only.
func (h *PlaylistHandler) AddTrack(c *gin.Context) {
	var req addTrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	if err := h.Playlists.AddTrack(c.Request.Context(), CurrentUserID(c), c.Param("id"), req.TrackID); err != nil {
		AbortError(c, err)
		return
	}
	created(c, gin.H{"added": true})
}

// RemoveTrack removes an entry; owner only.
func (h *PlaylistHandler) RemoveTrack(c *gin.Context) {
	if err := h.Playlists.RemoveTrack(c.Request.Context(), CurrentUserID(c), c.Param("id"), c.Param("trackId")); err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"removed": true})
}

// ListTracks returns ordered entries.
func (h *PlaylistHandler) ListTracks(c *gin.Context) {
	entries, err := h.Playlists.ListTracks(c.Request.Context(), CurrentUserID(c), c.Param("id"))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, entries)
}

type reorderRequest struct {
	TrackIDs []string `json:"trackIds" binding:"required"`
}

// Reorder rewrites entry order; owner only.
func (h *PlaylistHandler) Reorder(c *gin.Context) {
	var req reorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	if err := h.Playlists.Reorder(c.Request.Context(), CurrentUserID(c), c.Param("id"), req.TrackIDs); err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"reordered": true})
}
