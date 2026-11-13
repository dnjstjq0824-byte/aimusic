package handler

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
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
// 50 downloads per rolling 24h, masters capped at 500 MiB total. Only the
// head window is buffered for tag planning; the rest streams (B-07), so a
// download holds ~KBs regardless of master size.
const (
	maxDownloadsPerDay  = 50
	maxDownloadBytes    = 500 << 20
	downloadWindowHours = 24
	downloadHeadWindow  = 64 << 10
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
	items := make([]*domain.Track, 0, len(page.Items))
	for _, t := range page.Items {
		items = append(items, t.PublicView())
	}
	ok(c, gin.H{
		"items": items, "total": page.Total,
		"limit": page.Limit, "offset": page.Offset,
	})
}

// GetTrack returns one track. Anonymous callers and listeners only see
// READY tracks through the public view (B-05); the owning partner and
// admins see every status with full fields. Requires JWTOptional.
func (h *TrackHandler) GetTrack(c *gin.Context) {
	track, err := h.Repos.Tracks.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		AbortError(c, err)
		return
	}
	if h.canSeeFullTrack(c, track) {
		ok(c, track)
		return
	}
	if track.Status != domain.TrackReady {
		AbortError(c, domain.ErrNotFound)
		return
	}
	ok(c, track.PublicView())
}

// canSeeFullTrack reports whether the caller is an admin or the partner
// that owns the track.
func (h *TrackHandler) canSeeFullTrack(c *gin.Context, track *domain.Track) bool {
	userID := CurrentUserID(c)
	if userID == "" {
		return false
	}
	if CurrentRole(c) == domain.RoleAdmin {
		return true
	}
	if h.Repos == nil || h.Repos.Partners == nil {
		return false
	}
	p, err := h.Repos.Partners.GetByUserID(c.Request.Context(), userID)
	return err == nil && p.ID == track.PartnerID
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
	if err := h.Repos.Grants.RecordGrant(c.Request.Context(), jti, userID, track.ID); err != nil {
		AbortError(c, err)
		return
	}
	signed := apcrypto.BuildSignedPlaybackURL(playlistURL, track.ID, jti, exp, quality, h.Config.HLS.SigningKey)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"url": signed, "expiresAt": exp.Format(time.RFC3339), "jti": jti, "quality": quality,
	}})
}

// Download serves the tagged master file (P2-1 #22). Login required;
// paid passes only (track owner + admin exempt), 50/day quota, READY
// tracks only. WAV masters get a LIST-INFO disclosure chunk, MP3s an
// ID3v2 comment; unknown formats serve raw with the disclosure header.
// Only the head is buffered for tag planning; the body streams (B-07).
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
	ctx := c.Request.Context()
	head, total, err := h.Storage.GetRange(ctx, h.Buckets.Staging, track.OriginalObjectKey, 0, downloadHeadWindow)
	if err != nil || total <= 0 || total > maxDownloadBytes {
		abort(c, http.StatusBadGateway, "ERR_DOWNLOAD_UNAVAILABLE", "master file unavailable")
		return
	}
	disclosure := downloadDisclosureText(track, h.Config.OAuth.FrontendBaseURL)
	var plan *audio.DownloadPlan
	var ext string
	switch detectDownloadFormat(head) {
	case "wav":
		fetch := func(offset, length int64) ([]byte, error) {
			b, _, err := h.Storage.GetRange(ctx, h.Buckets.Staging, track.OriginalObjectKey, offset, length)
			return b, err
		}
		plan, err = audio.PlanWAVDownload(head, total, track.Title, "AVIBE", disclosure, fetch)
		ext = "wav"
	case "mp3":
		meta := &audio.AudioMetadata{Title: track.Title, Artist: "AVIBE", Comment: disclosure, AIProvider: track.AIProvider}
		if track.AIModelVersion != nil {
			meta.AIModelVersion = *track.AIModelVersion
		}
		if track.ISRC != nil {
			meta.ISRC = *track.ISRC
		}
		plan, err = audio.PlanMP3Download(head, total, meta)
		ext = "mp3"
	default:
		plan, ext = audio.PlanRawDownload(total), "bin"
	}
	if err != nil {
		abort(c, http.StatusBadGateway, "ERR_DOWNLOAD_UNAVAILABLE", "master file unavailable")
		return
	}
	if err := h.Repos.Downloads.Log(ctx, userID, track.ID, plan.TotalSize); err != nil {
		AbortError(c, err)
		return
	}
	c.Header("Content-Disposition", "attachment; filename=\""+downloadFilename(track.ID, track.Title, ext)+"\"")
	c.Header("X-AVIBE-AI-Disclosure", disclosure)
	c.Header("Content-Type", plan.ContentType)
	c.Header("Content-Length", strconv.FormatInt(plan.TotalSize, 10))
	c.Status(http.StatusOK)
	open := func(offset, length int64) (io.ReadCloser, error) {
		rc, _, err := h.Storage.OpenRange(ctx, h.Buckets.Staging, track.OriginalObjectKey, offset, length)
		return rc, err
	}
	// Headers are already sent: a mid-stream failure can only truncate the
	// body, so log it for operations and let the client retry.
	if err := plan.Stream(ctx, c.Writer, open); err != nil {
		slog.Error("download stream aborted", "track_id", track.ID, "user_id", userID, "error", err)
	}
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
