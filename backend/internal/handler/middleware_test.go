package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"ai-music-platform/backend/internal/config"
)

func testEngine(cfg *config.Config, mw ...gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(mw...)
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"requestId": CurrentRequestID(c)})
	})
	return r
}

func TestRequestIDAssignedAndEchoed(t *testing.T) {
	r := testEngine(nil, RequestID())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if id := w.Header().Get(RequestIDHeader); id == "" {
		t.Fatalf("missing %s response header", RequestIDHeader)
	}
}

func TestRequestIDPropagatesClientValue(t *testing.T) {
	r := testEngine(nil, RequestID())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set(RequestIDHeader, "trace-123")
	r.ServeHTTP(w, req)
	if got := w.Header().Get(RequestIDHeader); got != "trace-123" {
		t.Fatalf("propagated id = %q, want trace-123", got)
	}
}

func TestSecurityHeadersBaseline(t *testing.T) {
	r := testEngine(&config.Config{Env: "local"}, SecurityHeaders(&config.Config{Env: "local"}))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))
	want := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}
	for k, v := range want {
		if got := w.Header().Get(k); got != v {
			t.Fatalf("%s = %q, want %q", k, got, v)
		}
	}
	if hsts := w.Header().Get("Strict-Transport-Security"); hsts != "" {
		t.Fatalf("local env must not emit HSTS, got %q", hsts)
	}
}

func TestSecurityHeadersHSTSInProd(t *testing.T) {
	cfg := &config.Config{Env: "prod"}
	r := testEngine(cfg, SecurityHeaders(cfg))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if hsts := w.Header().Get("Strict-Transport-Security"); hsts == "" {
		t.Fatalf("prod env must emit HSTS")
	}
}

func TestCORSDeniesByDefault(t *testing.T) {
	r := testEngine(&config.Config{}, CORS(&config.Config{}))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://evil.example")
	r.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unlisted origin got ACAO %q", got)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("non-preflight must pass through, status = %d", w.Code)
	}
}

func TestCORSAllowsListedOrigin(t *testing.T) {
	cfg := &config.Config{}
	cfg.Security.CORSAllowedOrigins = []string{"https://app.example.com"}
	r := testEngine(cfg, CORS(cfg))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://app.example.com")
	r.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("ACAO = %q", got)
	}

	pre := httptest.NewRecorder()
	preReq := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	preReq.Header.Set("Origin", "https://app.example.com")
	r.ServeHTTP(pre, preReq)
	if pre.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", pre.Code)
	}
}

func TestRateLimitBurstsThen429(t *testing.T) {
	r := testEngine(nil, RateLimit(1000, 3))
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("request %d status = %d, want 200", i, w.Code)
		}
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("burst-exceeded status = %d, want 429", w.Code)
	}
}
