package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"avibe-platform/backend/internal/repository"
	"avibe-platform/backend/internal/service"
)

// DisputeHandler serves user-facing report/counter/blocks endpoints.
type DisputeHandler struct {
	Repos    *repository.Repos
	Disputes *service.DisputeService
}

type reportRequest struct {
	TrackID     string `json:"trackId" binding:"required"`
	Reason      string `json:"reason" binding:"required"`
	Description string `json:"description"`
}

// Report files a copyright dispute.
func (h *DisputeHandler) Report(c *gin.Context) {
	var req reportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	reporter := CurrentUserID(c)
	d, err := h.Disputes.Report(c.Request.Context(), req.TrackID, &reporter, req.Reason, req.Description)
	if err != nil {
		AbortError(c, err)
		return
	}
	created(c, d)
}

type counterRequest struct {
	Note string `json:"note" binding:"required"`
}

// Counter records a partner counter-claim on a dispute.
func (h *DisputeHandler) Counter(c *gin.Context) {
	var req counterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	d, err := h.Disputes.Counter(c.Request.Context(), c.Param("id"), CurrentUserID(c), req.Note)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, d)
}

// MyBlocks returns the caller's strike blocks.
func (h *DisputeHandler) MyBlocks(c *gin.Context) {
	blocks, err := h.Repos.Blocks.ListByUser(c.Request.Context(), CurrentUserID(c))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, blocks)
}

type blockArtistRequest struct {
	PartnerID string `json:"partnerId" binding:"required"`
	Reason    string `json:"reason"`
}

// BlockArtist blocks a partner/artist for the calling user (UGC store compliance).
func (h *DisputeHandler) BlockArtist(c *gin.Context) {
	var req blockArtistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	b, err := h.Disputes.BlockArtist(c.Request.Context(), CurrentUserID(c), req.PartnerID, req.Reason)
	if err != nil {
		AbortError(c, err)
		return
	}
	created(c, b)
}

// ListArtistBlocks returns the caller's blocked artists.
func (h *DisputeHandler) ListArtistBlocks(c *gin.Context) {
	blocks, err := h.Disputes.ListBlockedArtists(c.Request.Context(), CurrentUserID(c))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, blocks)
}

// UnblockArtist removes an artist block for the calling user.
func (h *DisputeHandler) UnblockArtist(c *gin.Context) {
	if err := h.Disputes.UnblockArtist(c.Request.Context(), CurrentUserID(c), c.Param("partnerId")); err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"unblocked": true})
}
