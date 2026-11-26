// Package service SEO use cases (WI-S1~S5): canonical resolution, sitemap
// cursors, landing gate, creator profiles, oEmbed, analytics events,
// IndexNow submission, and Cloudflare cache purge.
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"avibe-platform/backend/internal/config"
	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/repository"
	"avibe-platform/backend/internal/search"
)

// cloudflareAPIBase is a var so tests can point purge at a stub server.
var cloudflareAPIBase = "https://api.cloudflare.com/client/v4"

// SEOService implements the search-exposure use cases.
type SEOService struct {
	repos      *repository.Repos
	cfg        config.SEOConfig
	httpClient *http.Client
}

// NewSEOService builds the service. A nil httpClient gets a 10s default.
func NewSEOService(repos *repository.Repos, cfg config.SEOConfig, httpClient *http.Client) *SEOService {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &SEOService{repos: repos, cfg: cfg, httpClient: httpClient}
}

// StartupWarnings names SEO variables that are still unset (values never).
func (s *SEOService) StartupWarnings() []string {
	var out []string
	if s.cfg.IndexNowKey == "" {
		out = append(out, "seo: INDEXNOW_KEY unset (index submission disabled, sitemap only)")
	}
	if s.cfg.CachePurgeToken == "" {
		out = append(out, "seo: CF_CACHE_PURGE_TOKEN unset (relying on SWR expiry)")
	}
	return out
}

// Resolve maps a legacy or locale-prefixed page path to its canonical path.
// ok=false means no redirect target (already canonical or unknown).
// Unknown detail params yield domain.ErrNotFound so callers can 404.
func (s *SEOService) Resolve(ctx context.Context, rawPath string) (canonical string, redirect bool, err error) {
	p := strings.TrimSpace(rawPath)
	if p == "" || !strings.HasPrefix(p, "/") {
		return "", false, fmt.Errorf("%w: path must start with /", domain.ErrInvalidInput)
	}
	// Strip one locale prefix; canonical keeps the caller's locale form.
	localePrefix := ""
	rest := p
	for _, l := range domain.SEOLocales {
		if l == "ko" {
			continue
		}
		if rest == "/"+l || strings.HasPrefix(rest, "/"+l+"/") {
			localePrefix = "/" + l
			rest = strings.TrimPrefix(rest, "/"+l)
			if rest == "" {
				rest = "/"
			}
			break
		}
	}
	withLocale := func(path string) string {
		if path == "/" {
			return localePrefix + "/"
		}
		return localePrefix + path
	}
	switch {
	case rest == "/my-library":
		return withLocale("/library"), true, nil
	case rest == "/disputes":
		return withLocale("/rights-center"), true, nil
	case rest == "/tracks" || rest == "/playlists" || rest == "/creators":
		return "", false, nil
	}
	if param, ok := strings.CutPrefix(rest, "/tracks/"); ok {
		param = cutQuery(param)
		if strings.Contains(param, "/") || param == "" {
			return "", false, domain.ErrNotFound
		}
		_, canonicalPath, moved, err := s.repos.SEO.TrackByParam(ctx, param)
		if err != nil {
			return "", false, err
		}
		if !moved {
			return "", false, nil
		}
		return withLocale(canonicalPath), true, nil
	}
	if param, ok := strings.CutPrefix(rest, "/playlists/"); ok {
		param = cutQuery(param)
		if strings.Contains(param, "/") || param == "" {
			return "", false, domain.ErrNotFound
		}
		_, canonicalPath, moved, err := s.repos.SEO.PlaylistByParam(ctx, param)
		if err != nil {
			return "", false, err
		}
		if !moved {
			return "", false, nil
		}
		return withLocale(canonicalPath), true, nil
	}
	return "", false, nil
}

func cutQuery(s string) string {
	if i := strings.IndexByte(s, '?'); i >= 0 {
		return s[:i]
	}
	return s
}

// Guide entries for the sitemap + oEmbed allowlist. Lastmod is the last
// content edit (bump when guides change).
var guideEntries = []domain.SitemapItem{
	{Loc: "/guides/store-music-license", Lastmod: "2026-11-16T00:00:00Z"},
	{Loc: "/guides/suno-paid-commercial-use", Lastmod: "2026-11-16T00:00:00Z"},
	{Loc: "/guides/ai-music-youtube-checklist", Lastmod: "2026-11-16T00:00:00Z"},
}

// GuideSlugs lists the guide addresses (web + tests share this contract).
func GuideSlugs() []string {
	return []string{"store-music-license", "suno-paid-commercial-use", "ai-music-youtube-checklist"}
}

var staticEntries = []domain.SitemapItem{
	{Loc: "/", Lastmod: "2026-11-16T00:00:00Z"},
	{Loc: "/explore", Lastmod: "2026-11-16T00:00:00Z"},
	{Loc: "/charts", Lastmod: "2026-11-16T00:00:00Z"},
	{Loc: "/playlists", Lastmod: "2026-11-16T00:00:00Z"},
	{Loc: "/pricing", Lastmod: "2026-11-16T00:00:00Z"},
	{Loc: "/business", Lastmod: "2026-11-16T00:00:00Z"},
	{Loc: "/for-creators", Lastmod: "2026-11-16T00:00:00Z"},
	{Loc: "/trust", Lastmod: "2026-11-16T00:00:00Z"},
	{Loc: "/create", Lastmod: "2026-11-16T00:00:00Z"},
	{Loc: "/guides", Lastmod: "2026-11-16T00:00:00Z"},
	{Loc: "/about", Lastmod: "2026-11-26T00:00:00Z"},
	{Loc: "/help", Lastmod: "2026-11-26T00:00:00Z"},
	{Loc: "/contact", Lastmod: "2026-11-26T00:00:00Z"},
	{Loc: "/creators", Lastmod: "2026-11-26T00:00:00Z"},
	{Loc: "/certificate", Lastmod: "2026-11-26T00:00:00Z"},
	{Loc: "/models", Lastmod: "2026-11-26T00:00:00Z"},
	{Loc: "/models/avibe", Lastmod: "2026-11-26T00:00:00Z"},
	{Loc: "/models/suno", Lastmod: "2026-11-26T00:00:00Z"},
	{Loc: "/models/udio", Lastmod: "2026-11-26T00:00:00Z"},
	{Loc: "/models/aiva", Lastmod: "2026-11-26T00:00:00Z"},
	{Loc: "/models/stable-audio", Lastmod: "2026-11-26T00:00:00Z"},
	{Loc: "/models/elevenlabs", Lastmod: "2026-11-26T00:00:00Z"},
	{Loc: "/models/open-model", Lastmod: "2026-11-26T00:00:00Z"},
	{Loc: "/models/soundraw", Lastmod: "2026-11-26T00:00:00Z"},
	{Loc: "/models/mubert", Lastmod: "2026-11-26T00:00:00Z"},
}

// Sitemap returns one cursor page for a sitemap type. Static/guide/landing
// types are single-page (nextCursor empty); DB types paginate.
func (s *SEOService) Sitemap(ctx context.Context, typ, cursor string, limit int) ([]domain.SitemapItem, string, error) {
	switch typ {
	case domain.SitemapTracks:
		return s.repos.SEO.SitemapTracks(ctx, cursor, limit)
	case domain.SitemapPlaylists:
		return s.repos.SEO.SitemapPlaylists(ctx, cursor, limit)
	case domain.SitemapCreators:
		return s.repos.SEO.SitemapCreators(ctx, cursor, limit)
	case domain.SitemapGuides:
		return guideEntries, "", nil
	case domain.SitemapStatic:
		return staticEntries, "", nil
	case domain.SitemapLandings:
		return s.landingEntries(ctx)
	default:
		return nil, "", fmt.Errorf("%w: unknown sitemap type", domain.ErrInvalidInput)
	}
}

// landingEntries lists only facet landings with >= 12 READY tracks (S4 gate).
func (s *SEOService) landingEntries(ctx context.Context) ([]domain.SitemapItem, string, error) {
	today := time.Now().UTC().Format("2006-01-02") + "T00:00:00Z"
	var out []domain.SitemapItem
	seen := map[string]bool{}
	add := func(loc, mood, genre, use string) error {
		if seen[loc] {
			return nil
		}
		seen[loc] = true
		n, err := s.repos.SEO.LandingCount(ctx, mood, genre, use)
		if err != nil {
			return err
		}
		if n >= 12 {
			out = append(out, domain.SitemapItem{Loc: loc, Lastmod: today})
		}
		return nil
	}
	for _, tag := range search.Registry {
		for _, f := range tag.Facets {
			switch f {
			case search.FacetMood:
				if err := add("/explore/mood/"+tag.Key, tag.Key, "", ""); err != nil {
					return nil, "", err
				}
			case search.FacetGenre:
				if err := add("/explore/genre/"+tag.Key, "", tag.Key, ""); err != nil {
					return nil, "", err
				}
			case search.FacetUseCase:
				if err := add("/explore/use/"+tag.Key, "", "", tag.Key); err != nil {
					return nil, "", err
				}
			}
		}
	}
	if out == nil {
		out = []domain.SitemapItem{}
	}
	return out, "", nil
}

// LandingCount returns the READY count for one facet (S4 404 gate input).
func (s *SEOService) LandingCount(ctx context.Context, mood, genre, useCase string) (int64, error) {
	set := 0
	for _, v := range []string{mood, genre, useCase} {
		if strings.TrimSpace(v) != "" {
			set++
		}
	}
	if set != 1 {
		return 0, fmt.Errorf("%w: exactly one of mood|genre|use required", domain.ErrInvalidInput)
	}
	key := strings.ToLower(strings.TrimSpace(mood + genre + useCase))
	if !search.IsKey(key) {
		return 0, fmt.Errorf("%w: unknown facet key", domain.ErrInvalidInput)
	}
	return s.repos.SEO.LandingCount(ctx, strings.ToLower(mood), strings.ToLower(genre), strings.ToLower(useCase))
}

// CreatorDetail is the /creators/{handle} aggregate.
type CreatorDetail struct {
	Profile   *domain.CreatorProfile `json:"profile"`
	Tracks    []*domain.Track        `json:"tracks"`
	Playlists []*domain.Playlist     `json:"playlists"`
}

// Creator returns the public creator page data (READY tracks only).
func (s *SEOService) Creator(ctx context.Context, handle string) (*CreatorDetail, error) {
	profile, err := s.repos.SEO.CreatorByHandle(ctx, handle)
	if err != nil {
		return nil, err
	}
	all, err := s.repos.Tracks.ListByPartner(ctx, profile.PartnerID, 50, 0)
	if err != nil {
		return nil, err
	}
	tracks := []*domain.Track{}
	for _, t := range all {
		if t.Status == domain.TrackReady {
			tracks = append(tracks, t.PublicView())
		}
		if len(tracks) >= 10 {
			break
		}
	}
	mine, err := s.repos.Playlists.ListByOwner(ctx, profile.UserID)
	if err != nil {
		return nil, err
	}
	playlists := []*domain.Playlist{}
	for _, p := range mine {
		if p.IsPublic || p.IsOfficialCuration {
			playlists = append(playlists, p)
		}
		if len(playlists) >= 6 {
			break
		}
	}
	return &CreatorDetail{Profile: profile, Tracks: tracks, Playlists: playlists}, nil
}

// OEmbedResponse is the oEmbed rich payload for a track/playlist URL (S4).
type OEmbedResponse struct {
	Version      string `json:"version"`
	Type         string `json:"type"`
	ProviderName string `json:"provider_name"`
	ProviderURL  string `json:"provider_url"`
	Title        string `json:"title"`
	AuthorName   string `json:"author_name,omitempty"`
	AuthorURL    string `json:"author_url,omitempty"`
	HTML         string `json:"html"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
}

// OEmbed builds the embed payload for a canonical track/playlist page URL.
func (s *SEOService) OEmbed(ctx context.Context, rawURL string, maxWidth int) (*OEmbedResponse, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed == nil {
		return nil, fmt.Errorf("%w: bad url", domain.ErrInvalidInput)
	}
	path := parsed.Path
	if parsed.Host == "" && !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("%w: bad url", domain.ErrInvalidInput)
	}
	if parsed.Host != "" && !strings.EqualFold(parsed.Host, siteHost(s.cfg.SiteBaseURL)) {
		return nil, fmt.Errorf("%w: url must be an AVIBE page", domain.ErrInvalidInput)
	}
	// Strip locale prefix.
	for _, l := range domain.SEOLocales {
		if l == "ko" {
			continue
		}
		if path == "/"+l || strings.HasPrefix(path, "/"+l+"/") {
			path = strings.TrimPrefix(path, "/"+l)
			if path == "" {
				path = "/"
			}
			break
		}
	}
	width := 480
	if maxWidth >= 200 && maxWidth <= 800 {
		width = maxWidth
	}
	base := strings.TrimRight(s.cfg.SiteBaseURL, "/")
	if param, ok := strings.CutPrefix(path, "/tracks/"); ok {
		t, _, _, err := s.repos.SEO.TrackByParam(ctx, cutQuery(param))
		if err != nil {
			return nil, err
		}
		if t.Status != domain.TrackReady {
			return nil, domain.ErrNotFound
		}
		embed := base + "/embed/track/" + t.ShortID
		author := ""
		if h := creatorHandle(ctx, s.repos, t.PartnerID); h != "" {
			author = base + "/creators/" + h
		}
		return &OEmbedResponse{
			Version: "1.0", Type: "rich", ProviderName: "AVIBE", ProviderURL: base,
			Title: t.Title, AuthorName: t.CreatorName, AuthorURL: author,
			HTML:  fmt.Sprintf(`<iframe src="%s" width="%d" height="180" frameborder="0" allow="autoplay" title="%s"></iframe>`, embed, width, htmlEscape(t.Title)),
			Width: width, Height: 180, ThumbnailURL: base + "/favicon.svg",
		}, nil
	}
	if param, ok := strings.CutPrefix(path, "/playlists/"); ok {
		p, _, _, err := s.repos.SEO.PlaylistByParam(ctx, cutQuery(param))
		if err != nil {
			return nil, err
		}
		if !p.IsPublic && !p.IsOfficialCuration {
			return nil, domain.ErrNotFound
		}
		embed := base + "/embed/playlist/" + p.ShortID
		return &OEmbedResponse{
			Version: "1.0", Type: "rich", ProviderName: "AVIBE", ProviderURL: base,
			Title: p.Title, AuthorName: p.OwnerName,
			HTML:  fmt.Sprintf(`<iframe src="%s" width="%d" height="360" frameborder="0" allow="autoplay" title="%s"></iframe>`, embed, width, htmlEscape(p.Title)),
			Width: width, Height: 360, ThumbnailURL: base + "/favicon.svg",
		}, nil
	}
	return nil, domain.ErrNotFound
}

func siteHost(base string) string {
	u, err := url.Parse(base)
	if err != nil {
		return "avibemusic.com"
	}
	return u.Host
}

func creatorHandle(ctx context.Context, repos *repository.Repos, partnerID string) string {
	pp, err := repos.Partners.GetByID(ctx, partnerID)
	if err != nil || pp == nil {
		return ""
	}
	u, err := repos.Users.GetByID(ctx, pp.UserID)
	if err != nil || u == nil || u.Handle == nil {
		return ""
	}
	return *u.Handle
}

func htmlEscape(s string) string {
	r := strings.ReplaceAll(s, "&", "&amp;")
	r = strings.ReplaceAll(r, "<", "&lt;")
	r = strings.ReplaceAll(r, ">", "&gt;")
	return strings.ReplaceAll(r, `"`, "&quot;")
}

// SubmitEvents validates and stores analytics events (S5). Invalid entries
// fail the batch with ERR_INVALID_INPUT; forbidden props are stripped.
func (s *SEOService) SubmitEvents(ctx context.Context, events []domain.AnalyticsEvent, path string) (int, error) {
	if len(events) == 0 || len(events) > 50 {
		return 0, fmt.Errorf("%w: batch must hold 1-50 events", domain.ErrInvalidInput)
	}
	clean := make([]domain.AnalyticsEvent, 0, len(events))
	for _, e := range events {
		name := strings.TrimSpace(e.Name)
		if !domain.ValidAnalyticsEvent(name) {
			return 0, fmt.Errorf("%w: unknown event %q", domain.ErrInvalidInput, name)
		}
		if e.Locale != "" && !domain.IsSEOLocale(e.Locale) {
			return 0, fmt.Errorf("%w: bad locale", domain.ErrInvalidInput)
		}
		if len(e.Path) > 512 || len(e.SessionHash) > 64 {
			return 0, fmt.Errorf("%w: event field too long", domain.ErrInvalidInput)
		}
		e.Name = name
		e.Props = domain.SanitizeEventProps(e.Props)
		clean = append(clean, e)
	}
	if err := s.repos.SEO.InsertEvents(ctx, clean); err != nil {
		return 0, err
	}
	return len(clean), nil
}

// ExpandLocales returns the 5 locale absolute URLs for page paths.
func (s *SEOService) ExpandLocales(paths []string) []string {
	base := strings.TrimRight(s.cfg.SiteBaseURL, "/")
	var out []string
	for _, p := range paths {
		if !strings.HasPrefix(p, "/") {
			continue
		}
		out = append(out, base+p)
		for _, l := range domain.SEOLocales {
			if l == "ko" {
				continue
			}
			out = append(out, base+"/"+l+p)
		}
	}
	return out
}

// SubmitIndexNow sends URLs to IndexNow (S3): daily-cap guarded, audit-logged,
// retried once. Without INDEXNOW_KEY it logs locally and reports queued=false.
func (s *SEOService) SubmitIndexNow(ctx context.Context, paths []string) (queued bool, sent int, err error) {
	urls := s.ExpandLocales(paths)
	if len(urls) == 0 {
		return false, 0, fmt.Errorf("%w: no urls", domain.ErrInvalidInput)
	}
	if s.cfg.IndexNowKey == "" {
		if logErr := s.repos.SEO.IndexNowLog(ctx, urls, "skipped_no_key"); logErr != nil {
			return false, 0, logErr
		}
		return false, 0, nil
	}
	sentToday, err := s.repos.SEO.IndexNowSentSince(ctx, time.Now().Add(-24*time.Hour))
	if err != nil {
		return false, 0, err
	}
	if sentToday+int64(len(urls)) > int64(s.cfg.IndexNowDailyCap) {
		if logErr := s.repos.SEO.IndexNowLog(ctx, urls, "skipped_cap"); logErr != nil {
			return false, 0, logErr
		}
		return false, 0, fmt.Errorf("%w: indexnow daily cap reached", domain.ErrInvalidInput)
	}
	if err := s.repos.SEO.IndexNowLog(ctx, urls, "queued"); err != nil {
		return false, 0, err
	}
	body, _ := json.Marshal(map[string]any{
		"host":        siteHost(s.cfg.SiteBaseURL),
		"key":         s.cfg.IndexNowKey,
		"keyLocation": strings.TrimRight(s.cfg.SiteBaseURL, "/") + "/" + s.cfg.IndexNowKey + ".txt",
		"urlList":     urls,
	})
	var lastErr string
	ok := false
	for attempt := 0; attempt < 2 && !ok; attempt++ {
		if attempt > 0 {
			time.Sleep(2 * time.Second)
		}
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.IndexNowEndpoint, bytes.NewReader(body))
		if reqErr != nil {
			lastErr = reqErr.Error()
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, doErr := s.httpClient.Do(req)
		if doErr != nil {
			lastErr = doErr.Error()
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			ok = true
		} else {
			lastErr = fmt.Sprintf("indexnow status %d", resp.StatusCode)
		}
	}
	if markErr := s.repos.SEO.IndexNowMarkSent(ctx, urls, ok, lastErr); markErr != nil {
		return true, 0, markErr
	}
	if !ok {
		return true, 0, fmt.Errorf("indexnow submission failed: %s", lastErr)
	}
	return true, len(urls), nil
}

// PurgeCache purges absolute URLs from Cloudflare (S5). Without a token it
// reports purged=false so callers fall back to SWR expiry.
func (s *SEOService) PurgeCache(ctx context.Context, absoluteURLs []string) (purged bool, err error) {
	if s.cfg.CachePurgeToken == "" || s.cfg.CachePurgeZone == "" || len(absoluteURLs) == 0 {
		return false, nil
	}
	body, _ := json.Marshal(map[string]any{"files": absoluteURLs})
	endpoint := cloudflareAPIBase + "/zones/" + s.cfg.CachePurgeZone + "/purge_cache"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.CachePurgeToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	var decoded struct {
		Success bool `json:"success"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&decoded)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || !decoded.Success {
		return false, fmt.Errorf("cache purge failed: status %d", resp.StatusCode)
	}
	return true, nil
}
