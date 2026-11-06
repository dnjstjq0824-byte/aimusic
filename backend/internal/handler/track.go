package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ai-music-platform/backend/internal/config"
	"ai-music-platform/backend/internal/domain"
	apcrypto "ai-music-platform/backend/internal/pkg/crypto"
	"ai-music-platform/backend/internal/pkg/storage"
	"ai-music-platform/backend/internal/repository"
)

// TrackHandler serves public track listing and signed streaming URLs.
type TrackHandler struct {
	Repos   *repository.Repos
	Config  *config.Config
	Storage storage.Storage
	Buckets storage.Buckets
}

// ListTracks returns READY tracks.
func (h *TrackHandler) ListTracks(c *gin.Context) {
	tracks, err := h.Repos.Tracks.ListReady(c.Request.Context(), queryInt(c, "limit", 20), queryInt(c, "offset", 0))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, tracks)
}

// GetTrack returns one track.
func (h *TrackHandler) GetTrack(c *gin.Context) {
	track, err := h.Repos.Tracks.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, track)
}

// Stream issues a short-lived signed HLS URL for READY tracks only.
func (h *TrackHandler) Stream(c *gin.Context) {
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
		if s3, ok := h.Storage.(*storage.S3Compat); ok {
			base = strings.TrimRight(s3.Endpoint(), "/") + "/" + h.Buckets.PublicHLS
		} else {
			base = "https://cdn.example.com/" + h.Buckets.PublicHLS
		}
	}
	playlistURL := base + "/" + strings.TrimPrefix(*track.HLSPrefix, "/") + "/master.m3u8"
	jti := uuid.NewString()
	exp := time.Now().Add(time.Duration(h.Config.HLS.URLTTLSecs) * time.Second)
	signed := apcrypto.BuildSignedPlaybackURL(playlistURL, track.ID, jti, exp, h.Config.HLS.SigningKey)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"url": signed, "expiresAt": exp.Format(time.RFC3339), "jti": jti,
	}})
}

// PlaybackHandler serves heartbeat and resume state.
type PlaybackHandler struct {
	Playback PlaybackSvc
}

type heartbeatRequest struct {
	TrackID    string `json:"trackId" binding:"required"`
	PositionMs int    `json:"positionMs"`
	PlayMs     int    `json:"playMs"`
	JTI        string `json:"jti"`
}
