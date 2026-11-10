package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"avibe-platform/backend/internal/config"
	"avibe-platform/backend/internal/domain"
	apcrypto "avibe-platform/backend/internal/pkg/crypto"
	"avibe-platform/backend/internal/pkg/storage"
	"avibe-platform/backend/internal/repository"
	"avibe-platform/backend/internal/service"
)

// DeviceHandler serves the certified store player API (WI-05 #20).
type DeviceHandler struct {
	Repos    *repository.Repos
	Services *service.Services
	Config   *config.Config
	Storage  storage.Storage
	Buckets  storage.Buckets
}

type claimDeviceRequest struct {
	PairingCode string `json:"pairingCode" binding:"required"`
}

// ClaimDevice exchanges a pairing code for a long-lived device token.
func (h *DeviceHandler) ClaimDevice(c *gin.Context) {
	var req claimDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	dev, token, err := h.Services.Store.ClaimDevice(c.Request.Context(), req.PairingCode)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"device": dev, "token": token})
}

// DeviceMe returns the caller's device record.
func (h *DeviceHandler) DeviceMe(c *gin.Context) {
	dev, err := h.Repos.Store.GetDeviceByID(c.Request.Context(), CurrentDeviceID(c))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, dev)
}

// DeviceQueue returns the schedule-aware auto queue.
func (h *DeviceHandler) DeviceQueue(c *gin.Context) {
	q, err := h.Services.Store.DeviceQueue(c.Request.Context(), CurrentDeviceID(c))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, q)
}

type deviceHeartbeatRequest struct {
	TrackID       string `json:"trackId" binding:"required"`
	DurationSecs  int    `json:"durationSecs"`
}

// DeviceHeartbeat records a play from the certified player.
func (h *DeviceHandler) DeviceHeartbeat(c *gin.Context) {
	var req deviceHeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	counted, err := h.Services.Store.Heartbeat(c.Request.Context(), CurrentDeviceID(c), req.TrackID, req.DurationSecs)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"counted": counted})
}

// DeviceStream issues a signed HLS URL for the player. Store licenses are
// commercial, so players always get the HQ master playlist.
func (h *DeviceHandler) DeviceStream(c *gin.Context) {
	track, err := h.Repos.Tracks.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		AbortError(c, err)
		return
	}
	if track.Status != domain.TrackReady || track.HLSPrefix == nil {
		AbortError(c, domain.ErrNotStreamable)
		return
	}
	base := strings.TrimRight(h.Config.Storage.CDNBaseURL, "/")
	if base == "" {
		if s3, isS3 := h.Storage.(*storage.S3Compat); isS3 {
			base = strings.TrimRight(s3.Endpoint(), "/") + "/" + h.Buckets.PublicHLS
		} else {
			AbortError(c, fmt.Errorf("%w: storage CDN base is not configured", domain.ErrInvalidInput))
			return
		}
	}
	playlistURL := base + "/" + strings.TrimPrefix(*track.HLSPrefix, "/") + "/master.m3u8"
	jti := uuid.NewString()
	exp := time.Now().Add(time.Duration(h.Config.HLS.URLTTLSecs) * time.Second)
	signed := apcrypto.BuildSignedPlaybackURL(playlistURL, track.ID, jti, exp, h.Config.HLS.SigningKey)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"url": signed, "expiresAt": exp.Format(time.RFC3339), "jti": jti, "quality": "HQ",
	}})
}
