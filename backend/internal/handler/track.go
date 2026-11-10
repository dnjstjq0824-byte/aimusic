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
	"avibe-platform/backend/internal/pkg/audio"
	apcrypto "avibe-platform/backend/internal/pkg/crypto"
	"avibe-platform/backend/internal/pkg/storage"
	"avibe-platform/backend/internal/repository"
)

// Download policy (P2-1 #22): paid passes only (owner + admin exempt),
// 50 downloads per rolling 24h, masters capped at 500 MiB per fetch.
const (
	maxDownloadsPerDay  = 50
	maxDownloadBytes    = 500 << 20
	downloadWindowHours = 24
)

// detectDownloadFormat sniffs the master bytes: "wav" for RIFF/WAVE,
// "mp3" for ID3v2 or an MPEG frame header, "" when unknown.
func detectDownloadFormat(data []byte) string {
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WAVE" {
		return "wav"
	}
	if len(data) >= 3 && string(data[:3]) == "ID3" {
		return "mp3"
	}
	if len(data) >= 2 && data[0] == 0xFF && data[1]&0xE0 == 0xE0 {
		return "mp3"
	}
	return ""
}

// downloadFilename builds a stable attachment name: AVIBE-<id8>-<slug>.<ext>.
func downloadFilename(trackID, title, ext string) string {
	slug := make([]rune, 0, 40)
	prevDash := true
	for _, r := range strings.ToLower(title) {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if !ok {
			if !prevDash {
				slug = append(slug, '-')
				prevDash = true
			}
			continue
		}
		prevDash = false
		slug = append(slug, r)
		if len(slug) >= 40 {
			break
		}
	}
	name := strings.Trim(string(slug), "-")
	if name == "" {
		name = "track"
	}
	short := trackID
	if len(short) > 8 {
		short = short[:8]
	}
	return "AVIBE-" + short + "-" + name + "." + ext
}

// downloadDisclosureText is the in-file AI notice embedded in downloads.
func downloadDisclosureText(track *domain.Track, verifyBase string) string {
	base := strings.TrimRight(strings.TrimSpace(verifyBase), "/")
	if base == "" {
		base = "https://avibemusic.com"
	}
	return "AI-generated music via " + track.AIProvider + ". Verify: " + base + "/tracks/" + track.ID
}

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

// Download serves the tagged master file (P2-1 #22). Login required;
// paid passes only (track owner + admin exempt), 50/day quota, READY
// tracks only. WAV masters get a LIST-INFO disclosure chunk, MP3s an
// ID3v2 comment; unknown formats serve raw with the disclosure header.
func (h *TrackHandler) Download(c *gin.Context) {
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
	if track.Status != domain.TrackReady {
		AbortError(c, domain.ErrNotStreamable)
		return
	}
	if h.Repos != nil && h.Repos.Blocks != nil {
		if blocked, err := h.Repos.Blocks.IsArtistBlocked(c.Request.Context(), userID, track.PartnerID); err == nil && blocked {
			AbortError(c, domain.ErrNotStreamable)
			return
		}
	}
	role := CurrentRole(c)
	owner := false
	if role != domain.RoleAdmin {
		if p, err := h.Repos.Partners.GetByUserID(c.Request.Context(), userID); err == nil && p.ID == track.PartnerID {
			owner = true
		}
	}
	if role != domain.RoleAdmin && !owner {
		sub, err := h.Repos.Billing.GetActiveSubscription(c.Request.Context(), userID)
		if err != nil || sub == nil {
			abort(c, http.StatusForbidden, "ERR_DOWNLOAD_REQUIRES_PASS", "downloads require an active pass")
			return
		}
		since := time.Now().Add(-downloadWindowHours * time.Hour)
		n, err := h.Repos.Downloads.CountSince(c.Request.Context(), userID, since)
		if err != nil {
			AbortError(c, err)
			return
		}
		if n >= maxDownloadsPerDay {
			abort(c, http.StatusTooManyRequests, "ERR_DOWNLOAD_QUOTA", "daily download limit reached")
			return
		}
	}
	master, err := h.Storage.Get(c.Request.Context(), h.Buckets.Staging, track.OriginalObjectKey, maxDownloadBytes)
	if err != nil {
		abort(c, http.StatusBadGateway, "ERR_DOWNLOAD_UNAVAILABLE", "master file unavailable")
		return
	}
	disclosure := downloadDisclosureText(track, h.Config.OAuth.FrontendBaseURL)
	var body []byte
	var contentType, ext string
	switch detectDownloadFormat(master) {
	case "wav":
		tagged, err := audio.InjectWAVInfo(master, track.Title, "AVIBE", disclosure)
		if err != nil {
			abort(c, http.StatusBadGateway, "ERR_DOWNLOAD_UNAVAILABLE", "master file unavailable")
			return
		}
		body, contentType, ext = tagged, "audio/wav", "wav"
	case "mp3":
		meta := &audio.AudioMetadata{Title: track.Title, Artist: "AVIBE", Comment: disclosure, AIProvider: track.AIProvider}
		if track.AIModelVersion != nil {
			meta.AIModelVersion = *track.AIModelVersion
		}
		if track.ISRC != nil {
			meta.ISRC = *track.ISRC
		}
		tagged, err := audio.InjectID3v24(master, meta)
		if err != nil {
			abort(c, http.StatusBadGateway, "ERR_DOWNLOAD_UNAVAILABLE", "master file unavailable")
			return
		}
		body, contentType, ext = tagged, "audio/mpeg", "mp3"
	default:
		body, contentType, ext = master, "application/octet-stream", "bin"
	}
	if err := h.Repos.Downloads.Log(c.Request.Context(), userID, track.ID, int64(len(body))); err != nil {
		AbortError(c, err)
		return
	}
	c.Header("Content-Disposition", "attachment; filename=\""+downloadFilename(track.ID, track.Title, ext)+"\"")
	c.Header("X-AVIBE-AI-Disclosure", disclosure)
	c.Data(http.StatusOK, contentType, body)
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
