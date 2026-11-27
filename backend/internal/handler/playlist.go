package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/service"
)

// PlaylistHandler serves playlist CRUD, entries, and reorder.
type PlaylistHandler struct {
	Playlists *service.PlaylistService
	SEO       *service.SEOService
}

type playlistRequest struct {
	Title string `json:"title" binding:"required"`
	// Description is optional; omitted on update = unchanged, "" = clear.
	Description *string `json:"description"`
	IsPublic    bool    `json:"isPublic"`
}

// Create makes a playlist for the caller.
func (h *PlaylistHandler) Create(c *gin.Context) {
	var req playlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	p, err := h.Playlists.Create(c.Request.Context(), CurrentUserID(c), req.Title, req.Description, req.IsPublic)
	if err != nil {
		AbortError(c, err)
		return
	}
	created(c, p)
}

// ListMine returns the caller's playlists.
//
//	GET /playlists?containsTrack=<trackId>  (optional: per-playlist containsTrack flag)
func (h *PlaylistHandler) ListMine(c *gin.Context) {
	contains := c.Query("containsTrack")
	if contains != "" && !validUUID(contains) {
		badID(c)
		return
	}
	playlists, err := h.Playlists.ListMine(c.Request.Context(), CurrentUserID(c), contains)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, playlists)
}

// ListPublic returns discoverable playlists (no auth).
//
//	GET /playlists/public?limit=
func (h *PlaylistHandler) ListPublic(c *gin.Context) {
	items, err := h.Playlists.ListPublic(c.Request.Context(), queryInt(c, "limit", 20))
	if err != nil {
		AbortError(c, err)
		return
	}
	okList(c, items)
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
	p, err := h.Playlists.Update(c.Request.Context(), CurrentUserID(c), c.Param("id"), req.Title, req.Description, req.IsPublic)
	if err != nil {
		AbortError(c, err)
		return
	}
	notifySearchAsync(h.SEO, []string{domain.CanonicalPlaylistPath(p.Slug, p.ShortID)})
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

// AddTrack appends a READY track; owner only. A track already in the
// playlist answers 200 {added:false, duplicate:true} (no second entry).
func (h *PlaylistHandler) AddTrack(c *gin.Context) {
	var req addTrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	if !validUUID(req.TrackID) || !validUUID(c.Param("id")) {
		badID(c)
		return
	}
	added, err := h.Playlists.AddTrack(c.Request.Context(), CurrentUserID(c), c.Param("id"), req.TrackID)
	if err != nil {
		AbortError(c, err)
		return
	}
	if !added {
		ok(c, gin.H{"added": false, "duplicate": true})
		return
	}
	created(c, gin.H{"added": true, "duplicate": false})
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
