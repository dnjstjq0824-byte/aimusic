package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"avibe-platform/backend/internal/config"
	"avibe-platform/backend/internal/domain"
)

func testSEOService(cfg config.SEOConfig) *SEOService {
	cfg.SiteBaseURL = "https://avibemusic.com"
	return NewSEOService(nil, cfg, nil)
}

func TestExpandLocales(t *testing.T) {
	s := testSEOService(config.SEOConfig{})
	got := s.ExpandLocales([]string{"/tracks/a-Ab3xK9pQ", "nope"})
	if len(got) != 5 {
		t.Fatalf("len = %d, want 5", len(got))
	}
	want := []string{
		"https://avibemusic.com/tracks/a-Ab3xK9pQ",
		"https://avibemusic.com/en/tracks/a-Ab3xK9pQ",
		"https://avibemusic.com/ja/tracks/a-Ab3xK9pQ",
		"https://avibemusic.com/zh-cn/tracks/a-Ab3xK9pQ",
		"https://avibemusic.com/th/tracks/a-Ab3xK9pQ",
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestGuideSlugs(t *testing.T) {
	slugs := GuideSlugs()
	if len(slugs) != 3 {
		t.Fatalf("len = %d, want 3", len(slugs))
	}
}

func TestResolveValidation(t *testing.T) {
	s := testSEOService(config.SEOConfig{})
	ctx := context.Background()
	if _, _, err := s.Resolve(ctx, ""); err == nil {
		t.Error("empty path: want error")
	}
	if _, _, err := s.Resolve(ctx, "tracks/x"); err == nil {
		t.Error("relative path: want error")
	}
	if c, moved, err := s.Resolve(ctx, "/my-library"); err != nil || !moved || c != "/library" {
		t.Errorf("my-library = %q %v %v", c, moved, err)
	}
	if c, moved, err := s.Resolve(ctx, "/en/my-library"); err != nil || !moved || c != "/en/library" {
		t.Errorf("en my-library = %q %v %v", c, moved, err)
	}
	if c, moved, err := s.Resolve(ctx, "/disputes"); err != nil || !moved || c != "/rights-center" {
		t.Errorf("disputes = %q %v %v", c, moved, err)
	}
	if c, moved, err := s.Resolve(ctx, "/tracks"); err != nil || moved || c != "" {
		t.Errorf("index = %q %v %v", c, moved, err)
	}
}

func TestOEmbedValidation(t *testing.T) {
	s := testSEOService(config.SEOConfig{})
	ctx := context.Background()
	for _, bad := range []string{"", "notaurl", "https://evil.example.com/tracks/x-Ab3xK9pQ"} {
		if _, err := s.OEmbed(ctx, bad, 0); err == nil {
			t.Errorf("OEmbed(%q): want error", bad)
		}
	}
}

func TestSubmitEventsValidation(t *testing.T) {
	s := testSEOService(config.SEOConfig{})
	ctx := context.Background()
	if _, err := s.SubmitEvents(ctx, nil, "/"); err == nil {
		t.Error("empty batch: want error")
	}
	big := make([]domain.AnalyticsEvent, 51)
	if _, err := s.SubmitEvents(ctx, big, "/"); err == nil {
		t.Error("51 batch: want error")
	}
	bad := []domain.AnalyticsEvent{{Name: "keystroke"}}
	if _, err := s.SubmitEvents(ctx, bad, "/"); err == nil {
		t.Error("unknown event: want error")
	}
	badLocale := []domain.AnalyticsEvent{{Name: "play_started", Locale: "xx"}}
	if _, err := s.SubmitEvents(ctx, badLocale, "/"); err == nil {
		t.Error("bad locale: want error")
	}
}

func TestLandingCountValidation(t *testing.T) {
	s := testSEOService(config.SEOConfig{})
	ctx := context.Background()
	if _, err := s.LandingCount(ctx, "", "", ""); err == nil {
		t.Error("empty: want error")
	}
	if _, err := s.LandingCount(ctx, "cafe", "jazz", ""); err == nil {
		t.Error("two facets: want error")
	}
	if _, err := s.LandingCount(ctx, "nope", "", ""); err == nil {
		t.Error("unknown key: want error")
	}
}

func TestPurgeCacheNoToken(t *testing.T) {
	s := testSEOService(config.SEOConfig{})
	purged, err := s.PurgeCache(context.Background(), []string{"https://avibemusic.com/tracks/x"})
	if err != nil || purged {
		t.Errorf("no token = %v %v, want false nil", purged, err)
	}
}

func TestPurgeCacheRoundTrip(t *testing.T) {
	var gotAuth, gotPath string
	var gotFiles []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		var decoded struct {
			Files []string `json:"files"`
		}
		_ = json.NewDecoder(r.Body).Decode(&decoded)
		gotFiles = decoded.Files
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer srv.Close()
	old := cloudflareAPIBase
	cloudflareAPIBase = srv.URL
	defer func() { cloudflareAPIBase = old }()
	s := testSEOService(config.SEOConfig{CachePurgeToken: "tok", CachePurgeZone: "zone1"})
	purged, err := s.PurgeCache(context.Background(), []string{"https://avibemusic.com/tracks/x-Ab3xK9pQ"})
	if err != nil || !purged {
		t.Fatalf("purge = %v %v, want true nil", purged, err)
	}
	if gotAuth != "Bearer tok" {
		t.Errorf("auth = %q", gotAuth)
	}
	if gotPath != "/zones/zone1/purge_cache" {
		t.Errorf("path = %q", gotPath)
	}
	if len(gotFiles) != 1 || !strings.HasSuffix(gotFiles[0], "/tracks/x-Ab3xK9pQ") {
		t.Errorf("files = %v", gotFiles)
	}
}
