package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"avibe-platform/backend/internal/config"
	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/pkg/storage"
	"avibe-platform/backend/internal/repository"
	"avibe-platform/backend/internal/service"
)

// TestEmailVerifyRoutesRegisterAndFailClosed guards the A2 route table: Gin
// panics on static/param conflicts at registration, and every guard path
// fails closed without credentials. Confirm logic itself is covered by
// service tests (it needs a database).
func TestEmailVerifyRoutesRegisterAndFailClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	repos := &repository.Repos{}
	services := service.New(repos, nil, storage.Buckets{}, cfg)
	r := SetupRouter(&Deps{Config: cfg, Repos: repos, Services: services})

	post := func(target, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}

	// Guest request/locale fail at JWT (401) before touching any service.
	if w := post("/api/v1/auth/email/verify/request", "{}"); w.Code != http.StatusUnauthorized {
		t.Fatalf("guest request status = %d, want 401", w.Code)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/me/locale", strings.NewReader(`{"locale":"en"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("guest locale status = %d, want 401", w.Code)
	}

	// Confirm validates the body before touching the token store.
	if w := post("/api/v1/auth/email/verify/confirm", "{}"); w.Code != http.StatusBadRequest {
		t.Fatalf("missing token status = %d, want 400", w.Code)
	}
	if body := post("/api/v1/auth/email/verify/confirm", "{}").Body.String(); !strings.Contains(body, "ERR_INVALID_INPUT") {
		t.Fatalf("missing token body = %s", body)
	}
}

// TestEmailNotVerifiedErrorMapping pins the A2 commerce-gate contract:
// domain.ErrEmailNotVerified renders 403 EMAIL_NOT_VERIFIED.
func TestEmailNotVerifiedErrorMapping(t *testing.T) {
	if got := errCodeOfDomain(domain.ErrEmailNotVerified); got != "EMAIL_NOT_VERIFIED" {
		t.Fatalf("code = %q", got)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/gated", func(c *gin.Context) {
		AbortError(c, domain.ErrEmailNotVerified)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/gated", nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, "EMAIL_NOT_VERIFIED") {
		t.Fatalf("body = %s", body)
	}
}
