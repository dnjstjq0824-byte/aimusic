package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"ai-music-platform/backend/internal/domain"
	"ai-music-platform/backend/internal/service"
)

// PlaybackSvc is the playback use-case surface used by handlers.
type PlaybackSvc interface {
	Heartbeat(ctx context.Context, in service.HeartbeatInput) (*service.HeartbeatOutput, error)
	GetState(ctx context.Context, userID, trackID string) (*domain.UserPlaybackState, error)
	SetPosition(ctx context.Context, userID, trackID string, positionMs int) (*domain.UserPlaybackState, error)
}

// Heartbeat logs playback progress and returns counting/fraud decisions.
func (h *PlaybackHandler) Heartbeat(c *gin.Context) {
	var req heartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	out, err := h.Playback.Heartbeat(c.Request.Context(), service.HeartbeatInput{
		UserID: CurrentUserID(c), TrackID: req.TrackID,
		PositionMs: req.PositionMs, PlayMs: req.PlayMs, JTI: req.JTI,
		IP: c.ClientIP(), UA: c.GetHeader("User-Agent"),
	})
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, out)
}

// GetState returns the resume position for a track.
func (h *PlaybackHandler) GetState(c *gin.Context) {
	state, err := h.Playback.GetState(c.Request.Context(), CurrentUserID(c), c.Param("trackId"))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, state)
}

type setStateRequest struct {
	TrackID    string `json:"trackId"`
	PositionMs int    `json:"positionMs" binding:"required"`
}

// SetState stores the resume position for a track.
func (h *PlaybackHandler) SetState(c *gin.Context) {
	var req setStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	trackID := c.Param("trackId")
	if trackID == "" {
		trackID = req.TrackID
	}
	if trackID == "" {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "trackId required")
		return
	}
	state, err := h.Playback.SetPosition(c.Request.Context(), CurrentUserID(c), trackID, req.PositionMs)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, state)
}
