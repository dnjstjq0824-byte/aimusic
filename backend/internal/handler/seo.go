package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"avibe-platform/backend/internal/config"
	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/repository"
	"avibe-platform/backend/internal/service"
)

// SEOHandler serves the WI-S1~S5 endpoints: canonical resolution, sitemap
// cursors, landing gate, creator profiles, oEmbed, analytics events, and
// the IndexNow/purge ops hooks.
type SEOHandler struct {
	SEO    *service.SEOService
	Repos  *repository.Repos
	Config *config.Config
}

// Resolve answers GET /seo/resolve?path= with the canonical target (S1).
func (h *SEOHandler) Resolve(c *gin.Context) {
	canonical, redirect, err := h.SEO.Resolve(c.Request.Context(), c.Query("path"))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"canonical": canonical, "redirect": redirect})
}

// Sitemap answers GET /seo/sitemap/:type?cursor=&limit= (S3, 1h cached).
func (h *SEOHandler) Sitemap(c *gin.Context) {
	typ := strings.ToLower(strings.TrimSpace(c.Param("type")))
	if !domain.ValidSitemapType(typ) {
		AbortError(c, fmt.Errorf("%w: unknown sitemap type", domain.ErrInvalidInput))
		return
	}
	limit := 500
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 || n > 2000 {
			AbortError(c, fmt.Errorf("%w: limit must be 1-2000", domain.ErrInvalidInput))
			return
		}
		limit = n
	}
	items, next, err := h.SEO.Sitemap(c.Request.Context(), typ, strings.TrimSpace(c.Query("cursor")), limit)
	if err != nil {
		AbortError(c, err)
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	ok(c, gin.H{"items": items, "nextCursor": next})
}

// Landing answers GET /seo/landing?mood=&genre=&use= (S4 12-track gate).
func (h *SEOHandler) Landing(c *gin.Context) {
	n, err := h.SEO.LandingCount(c.Request.Context(),
		strings.TrimSpace(c.Query("mood")),
		strings.TrimSpace(c.Query("genre")),
		strings.TrimSpace(c.Query("use")))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"count": n, "ok": n >= 12})
}

// Creators answers GET /creators?limit=&offset= (public creator directory).
func (h *SEOHandler) Creators(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "24"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	items, err := h.Repos.SEO.ListCreators(c.Request.Context(), limit, offset)
	if err != nil {
		AbortError(c, err)
		return
	}
	c.Header("Cache-Control", "public, max-age=300")
	ok(c, gin.H{"items": items})
}

// Creator answers GET /creators/:handle (S4).
func (h *SEOHandler) Creator(c *gin.Context) {
	detail, err := h.SEO.Creator(c.Request.Context(), strings.TrimSpace(c.Param("handle")))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, detail)
}

// notifySearchAsync pushes changed page paths to IndexNow and the CDN purge
// in the background (S3/S5). Failures only surface in the audit logs; the
// caller response never waits for them.
func notifySearchAsync(svc *service.SEOService, paths []string) {
	if svc == nil || len(paths) == 0 {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(context.Background()), 30*time.Second)
		defer cancel()
		_, _, _ = svc.SubmitIndexNow(ctx, paths)
		_, _ = svc.PurgeCache(ctx, svc.ExpandLocales(paths))
	}()
}

// ClaimHandle answers POST /users/handle (S1, authed creator handle claim).
func (h *SEOHandler) ClaimHandle(c *gin.Context) {
	var req struct {
		Handle string `json:"handle"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortError(c, fmt.Errorf("%w: handle required", domain.ErrInvalidInput))
		return
	}
	handle := strings.TrimSpace(req.Handle)
	if err := domain.ValidateHandle(handle); err != nil {
		AbortError(c, fmt.Errorf("%w: %s", domain.ErrInvalidInput, err.Error()))
		return
	}
	if err := h.Repos.SEO.ClaimHandle(c.Request.Context(), CurrentUserID(c), handle); err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"handle": handle})
}

// OEmbed answers GET /oembed?url=&maxwidth= with the bare oEmbed JSON (S4).
// The response is intentionally unenveloped: oEmbed consumers expect the
// spec shape, not the AVIBE {data} envelope.
func (h *SEOHandler) OEmbed(c *gin.Context) {
	rawURL := strings.TrimSpace(c.Query("url"))
	if rawURL == "" || len(rawURL) > 2048 {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "url required")
		return
	}
	maxWidth := 0
	if raw := strings.TrimSpace(c.Query("maxwidth")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", "maxwidth must be a number")
			return
		}
		maxWidth = n
	}
	payload, err := h.SEO.OEmbed(c.Request.Context(), rawURL, maxWidth)
	if err != nil {
		AbortError(c, err)
		return
	}
	c.JSON(http.StatusOK, payload)
}

type eventsRequest struct {
	Events []domain.AnalyticsEvent `json:"events"`
}

// SubmitEvents answers POST /events with a 202 accept (S5, cookieless).
func (h *SEOHandler) SubmitEvents(c *gin.Context) {
	var req eventsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortError(c, fmt.Errorf("%w: events required", domain.ErrInvalidInput))
		return
	}
	n, err := h.SEO.SubmitEvents(c.Request.Context(), req.Events, c.Request.URL.Path)
	if err != nil {
		AbortError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"data": gin.H{"accepted": n}})
}

type indexNowRequest struct {
	Paths []string `json:"paths"`
}

// IndexNowStatus answers GET /admin/seo/status (ops only, no key material).
func (h *SEOHandler) IndexNowStatus(c *gin.Context) {
	sent, err := h.Repos.SEO.IndexNowSentSince(c.Request.Context(), time.Now().Add(-24*time.Hour))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{
		"indexNowEnabled":  h.Config.SEO.IndexNowKey != "",
		"sent24h":          sent,
		"dailyCap":         h.Config.SEO.IndexNowDailyCap,
		"cachePurgeReady":  h.Config.SEO.CachePurgeToken != "" && h.Config.SEO.CachePurgeZone != "",
		"consoleVerifying": h.Config.SEO.GoogleVerify != "" || h.Config.SEO.BingVerify != "" || h.Config.SEO.NaverVerify != "" || h.Config.SEO.DaumVerify != "",
	})
}

// IndexNowSubmit answers POST /admin/seo/indexnow {paths} (manual push, S3).
func (h *SEOHandler) IndexNowSubmit(c *gin.Context) {
	var req indexNowRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Paths) == 0 || len(req.Paths) > 100 {
		AbortError(c, fmt.Errorf("%w: paths must hold 1-100 entries", domain.ErrInvalidInput))
		return
	}
	queued, sent, err := h.SEO.SubmitIndexNow(c.Request.Context(), req.Paths)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"queued": queued, "sent": sent})
}

type purgeRequest struct {
	URLs []string `json:"urls"`
}

// PurgeCache answers POST /admin/seo/purge {urls} (S5 manual purge).
func (h *SEOHandler) PurgeCache(c *gin.Context) {
	var req purgeRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.URLs) == 0 || len(req.URLs) > 100 {
		AbortError(c, fmt.Errorf("%w: urls must hold 1-100 entries", domain.ErrInvalidInput))
		return
	}
	for _, u := range req.URLs {
		if len(u) > 2048 || (!strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://")) {
			AbortError(c, fmt.Errorf("%w: urls must be absolute http(s)", domain.ErrInvalidInput))
			return
		}
	}
	purged, err := h.SEO.PurgeCache(c.Request.Context(), req.URLs)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"purged": purged})
}
