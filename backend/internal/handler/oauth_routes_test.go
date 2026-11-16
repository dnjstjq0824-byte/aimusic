package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"avibe-platform/backend/internal/config"
	"avibe-platform/backend/internal/pkg/storage"
	"avibe-platform/backend/internal/repository"
	"avibe-platform/backend/internal/service"
)

// TestOAuthRoutesRegisterAndFailClosed guards the A1 route table: Gin panics
// on static/param conflicts at registration, and every OAuth guard path must
// fail closed without credentials.
func TestOAuthRoutesRegisterAndFailClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	repos := &repository.Repos{}
	services := service.New(repos, nil, storage.Buckets{}, cfg)
	r := SetupRouter(&Deps{Config: cfg, Repos: repos, Services: services})

	get := func(target string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
		return w
	}

	// Provider catalog: 5 codes in ko display order, all disabled.
	w := get("/api/v1/auth/oauth/providers?locale=ko")
	if w.Code != http.StatusOK {
		t.Fatalf("providers status = %d", w.Code)
	}
	var catalog struct {
		Data []struct {
			Provider string `json:"provider"`
			Enabled  bool   `json:"enabled"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &catalog); err != nil {
		t.Fatalf("providers body: %v", err)
	}
	if len(catalog.Data) != 5 || catalog.Data[0].Provider != "KAKAO" {
		t.Fatalf("catalog = %+v", catalog.Data)
	}
	for _, p := range catalog.Data {
		if p.Enabled {
			t.Fatalf("provider %s enabled without credentials", p.Provider)
		}
	}

	// Unknown + disabled providers fail closed at start.
	if w := get("/api/v1/auth/oauth/wechat/start"); w.Code != http.StatusBadRequest {
		t.Fatalf("wechat start status = %d, want 400", w.Code)
	}
	if w := get("/api/v1/auth/oauth/google/start"); w.Code != http.StatusBadRequest {
		t.Fatalf("disabled start status = %d, want 400", w.Code)
	}

	// Callback failures redirect with an error code and never leak tokens.
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/api/v1/auth/oauth/google/callback?code=x&state=y", nil)
		if method == http.MethodPost {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		r.ServeHTTP(w, req)
		if w.Code != http.StatusFound {
			t.Fatalf("%s callback status = %d, want 302", method, w.Code)
		}
		loc := w.Header().Get("Location")
		if !strings.Contains(loc, "oauthError=") {
			t.Fatalf("%s callback location = %q", method, loc)
		}
		if strings.Contains(loc, "accessToken=") || strings.Contains(loc, "refreshToken=") {
			t.Fatalf("%s callback leaks tokens: %q", method, loc)
		}
	}

	// Exchange with a bogus code is rejected without touching stores.
	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/exchange", strings.NewReader(`{"code":"bogus"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("exchange status = %d, want 401", w.Code)
	}

	// Connection endpoints require auth.
	if w := get("/api/v1/auth/oauth/connections"); w.Code != http.StatusUnauthorized {
		t.Fatalf("connections status = %d, want 401", w.Code)
	}
}

// TestOAuthAppErrorReturnRoutesToAppLink guards the WI-R12 error path:
// a callback failure whose state carries the .app marker redirects to
// the fixed avibe:// app link, never to the web frontend.
func TestOAuthAppErrorReturnRoutesToAppLink(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	repos := &repository.Repos{}
	services := service.New(repos, nil, storage.Buckets{}, cfg)
	r := SetupRouter(&Deps{Config: cfg, Repos: repos, Services: services})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
		"/api/v1/auth/oauth/google/callback?code=x&state=v1.n.e.GOOGLE.app.deadbeef", nil))
	if w.Code != http.StatusFound {
		t.Fatalf("app callback status = %d, want 302", w.Code)
	}
	loc := w.Header().Get("Location")
	if !strings.HasPrefix(loc, MobileOAuthCallbackBase+"?oauthError=") {
		t.Fatalf("app callback location = %q", loc)
	}
}
