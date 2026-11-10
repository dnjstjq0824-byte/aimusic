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
)

// TrackHandler serves public track listing and signed streaming URLs.
type TrackHandler struct {
	Repos   *repository.Repos
	Config  *config.Config
	Storage storage.Storage
	Buckets storage.Buckets
}

// ListTracks returns one page of READY tracks with server-side search
// (q), filters (provider/genre/mood), sort, and a total count:
//
//	GET /tracks?q=&provider=&genre=&mood=&sort=recent|oldest|popular|title&limit=&offset=
func (h *TrackHandler) ListTracks(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if len([]rune(q)) > 100 {
		AbortError(c, fmt.Errorf("%w: q too long (max 100 chars)", domain.ErrInvalidInput))
		return
	}
	sort := strings.TrimSpace(c.Query("sort"))
	if _, valid := repository.TrackSortOrderBy(sort); !valid {
		AbortError(c, fmt.Errorf("%w: sort must be recent|oldest|popular|title", domain.ErrInvalidInput))
		return
	}
	page, err := h.Repos.Tracks.SearchReady(c.Request.Context(), repository.TrackFilter{
		Q:        q,
		Provider: c.Query("provider"),
		Genre:    c.Query("genre"),
		Mood:     c.Query("mood"),
		Sort:     sort,
		Limit:    queryInt(c, "limit", 50),
		Offset:   queryInt(c, "offset", 0),
	})
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{
		"items": page.Items, "total": page.Total,
		"limit": page.Limit, "offset": page.Offset,
	})
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

// Stream issues a short-lived signed HLS URL for READY tracks (or
// LICENSE_GATE_PENDING for Admin review preview). Login is required; pass
// holders (and admins) get the master playlist (128k+256k), everyone else
// gets the 128k standard-quality variant playlist.
func (h *TrackHandler) Stream(c *gin.Context) {
	userID := CurrentUserID(c)
	if userID == "" {
		abort(c, http.StatusUnauthorized, "ERR_UNAUTHORIZED", "login required")
		return
	}
	track, err := h.Repos.Tracks.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		AbortError(c, err)
		return
	}
	isAdminPreview := CurrentRole(c) == domain.RoleAdmin && track.Status == domain.TrackLicenseGatePending
	if (track.Status != domain.TrackReady && !isAdminPreview) || track.HLSPrefix == nil {
		AbortError(c, domain.ErrNotStreamable)
		return
	}
	if h.Repos != nil && h.Repos.Blocks != nil {
		if blocked, err := h.Repos.Blocks.IsArtistBlocked(c.Request.Context(), userID, track.PartnerID); err == nil && blocked {
			AbortError(c, domain.ErrNotStreamable)
			return
		}
	}
	base := strings.TrimRight(h.Config.Storage.CDNBaseURL, "/")
	if base == "" {
		if s3, ok := h.Storage.(*storage.S3Compat); ok {
			base = strings.TrimRight(s3.Endpoint(), "/") + "/" + h.Buckets.PublicHLS
		} else {
			AbortError(c, fmt.Errorf("%w: storage CDN base is not configured", domain.ErrInvalidInput))
			return
		}
	}
	// HQ gate: active pass (or admin role) unlocks the master playlist with
	// the 256k variant; lookup failures fail open to standard quality so a
	// billing outage never silences playback.
	playlist := "stream_0.m3u8"
	quality := "SQ"
	if CurrentRole(c) == domain.RoleAdmin {
		playlist, quality = "master.m3u8", "HQ"
	} else if h.Repos != nil && h.Repos.Billing != nil {
		if sub, err := h.Repos.Billing.GetActiveSubscription(c.Request.Context(), userID); err == nil && sub != nil {
			playlist, quality = "master.m3u8", "HQ"
		}
	}
	playlistURL := base + "/" + strings.TrimPrefix(*track.HLSPrefix, "/") + "/" + playlist
	jti := uuid.NewString()
	exp := time.Now().Add(time.Duration(h.Config.HLS.URLTTLSecs) * time.Second)
	signed := apcrypto.BuildSignedPlaybackURL(playlistURL, track.ID, jti, exp, h.Config.HLS.SigningKey)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"url": signed, "expiresAt": exp.Format(time.RFC3339), "jti": jti, "quality": quality,
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
