// Package handler exposes the Gin REST API with JWT, RBAC, rate-limit, and audit middleware.
package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ai-music-platform/backend/internal/config"
	"ai-music-platform/backend/internal/domain"
	"ai-music-platform/backend/internal/service"
)

// Context keys for the authenticated principal and request tracing.
const (
	ctxUserIDKey    = "auth.userID"
	ctxRoleKey      = "auth.role"
	ctxRequestIDKey = "http.requestID"
)

// RequestIDHeader is echoed on every response for log correlation.
const RequestIDHeader = "X-Request-ID"

// CurrentUserID returns the authenticated user ID (empty when anonymous).
func CurrentUserID(c *gin.Context) string {
	if v, ok := c.Get(ctxUserIDKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// CurrentRole returns the authenticated role.
func CurrentRole(c *gin.Context) string {
	if v, ok := c.Get(ctxRoleKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// JWTAuth validates Bearer access tokens and stores the principal.
func JWTAuth(auth *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		token := ""
		if parts := strings.SplitN(header, " ", 2); len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			token = strings.TrimSpace(parts[1])
		}
		if token == "" {
			abort(c, http.StatusUnauthorized, "ERR_UNAUTHORIZED", "missing bearer token")
			return
		}
		claims, err := auth.ParseAccess(token)
		if err != nil {
			abort(c, http.StatusUnauthorized, "ERR_UNAUTHORIZED", "invalid token")
			return
		}
		c.Set(ctxUserIDKey, claims.UserID)
		c.Set(ctxRoleKey, claims.Role)
		c.Next()
	}
}

// RequireRole allows only the listed roles.
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		if !allowed[CurrentRole(c)] {
			abort(c, http.StatusForbidden, "ERR_FORBIDDEN", "insufficient role")
			return
		}
		c.Next()
	}
}

type bucket struct {
	tokens float64
	last   time.Time
}

// RateLimiter is an in-memory token bucket keyed by client IP.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rps     float64
	burst   float64
}

// RateLimit builds token-bucket middleware with rps refill and burst capacity.
func RateLimit(rps float64, burst int) gin.HandlerFunc {
	lim := &RateLimiter{buckets: map[string]*bucket{}, rps: rps, burst: float64(burst)}
	return func(c *gin.Context) {
		key := c.ClientIP()
		now := time.Now()
		lim.mu.Lock()
		b, ok := lim.buckets[key]
		if !ok {
			b = &bucket{tokens: lim.burst, last: now}
			lim.buckets[key] = b
		}
		elapsed := now.Sub(b.last).Seconds()
		b.tokens += elapsed * lim.rps
		if b.tokens > lim.burst {
			b.tokens = lim.burst
		}
		b.last = now
		if b.tokens < 1 {
			lim.mu.Unlock()
			abort(c, http.StatusTooManyRequests, "ERR_RATE_LIMITED", "too many requests")
			return
		}
		b.tokens--
		lim.mu.Unlock()
		c.Next()
	}
}

// RequestID assigns (or propagates) a request ID for log correlation and
// echoes it on the response.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.GetHeader(RequestIDHeader))
		if id == "" {
			id = uuid.NewString()
		}
		c.Set(ctxRequestIDKey, id)
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}

// CurrentRequestID returns the request ID assigned by the RequestID middleware.
func CurrentRequestID(c *gin.Context) string {
	if v, ok := c.Get(ctxRequestIDKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// SecurityHeaders applies baseline hardening headers to every response.
// The API serves JSON only, so the policy is deny-by-default for framing and
// MIME sniffing; HSTS is emitted outside local/dev environments.
func SecurityHeaders(cfg *config.Config) gin.HandlerFunc {
	hsts := cfg != nil && cfg.Env != "" && cfg.Env != "local" && cfg.Env != "dev" && cfg.Env != "test"
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		if hsts {
			h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		c.Next()
	}
}

// CORS enforces an exact-match origin allowlist. With an empty allowlist the
// API is same-origin only and no CORS headers are emitted.
func CORS(cfg *config.Config) gin.HandlerFunc {
	allowed := map[string]bool{}
	if cfg != nil {
		for _, o := range cfg.Security.CORSAllowedOrigins {
			allowed[o] = true
		}
	}
	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin == "" || !allowed[origin] {
			c.Next()
			return
		}
		h := c.Writer.Header()
		h.Set("Access-Control-Allow-Origin", origin)
		h.Set("Vary", "Origin")
		h.Set("Access-Control-Allow-Credentials", "true")
		h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
		h.Set("Access-Control-Max-Age", "86400")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// AuditLog records method, path, status, principal, and latency per request.
func AuditLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		slog.Info("http_request",
			"request_id", CurrentRequestID(c),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"user", CurrentUserID(c),
			"role", CurrentRole(c),
			"ip", c.ClientIP(),
			"latency_ms", time.Since(start).Milliseconds(),
		)
	}
}

func abort(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, gin.H{"data": data})
}

// errCodeOfDomain maps a domain error to its stable machine code (""
// when the error has no public code).
func errCodeOfDomain(err error) string {
	var gate *domain.GateError
	if errors.As(err, &gate) {
		return gate.Code
	}
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return "ERR_NOT_FOUND"
	case errors.Is(err, domain.ErrEmailTaken):
		return "ERR_EMAIL_TAKEN"
	case errors.Is(err, domain.ErrInvalidCredentials):
		return "ERR_INVALID_CREDENTIALS"
	case errors.Is(err, domain.ErrUserInactive):
		return "ERR_USER_INACTIVE"
	case errors.Is(err, domain.ErrForbidden):
		return "ERR_FORBIDDEN"
	case errors.Is(err, domain.ErrNotStreamable):
		return "ERR_NOT_STREAMABLE"
	case errors.Is(err, domain.ErrInvalidInput):
		return "ERR_INVALID_INPUT"
	case errors.Is(err, domain.ErrPaymentUnavailable):
		return "ERR_PAYMENT_UNAVAILABLE"
	default:
		return ""
	}
}

// AbortError maps domain and gate errors to status-coded JSON errors.
func AbortError(c *gin.Context, err error) {
	var gate *domain.GateError
	if errors.As(err, &gate) {
		abort(c, gate.StatusCode(), gate.Code, gate.Message)
		return
	}
	switch {
	case errors.Is(err, domain.ErrNotFound):
		abort(c, http.StatusNotFound, "ERR_NOT_FOUND", "not found")
	case errors.Is(err, domain.ErrEmailTaken):
		abort(c, http.StatusConflict, "ERR_EMAIL_TAKEN", "email already registered")
	case errors.Is(err, domain.ErrInvalidCredentials):
		abort(c, http.StatusUnauthorized, "ERR_INVALID_CREDENTIALS", "invalid credentials")
	case errors.Is(err, domain.ErrUserInactive):
		abort(c, http.StatusForbidden, "ERR_USER_INACTIVE", "account disabled")
	case errors.Is(err, domain.ErrForbidden):
		abort(c, http.StatusForbidden, "ERR_FORBIDDEN", "forbidden")
	case errors.Is(err, domain.ErrNotStreamable):
		abort(c, http.StatusForbidden, "ERR_NOT_STREAMABLE", "track is not streamable")
	case errors.Is(err, domain.ErrInvalidInput):
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
	case errors.Is(err, domain.ErrPaymentUnavailable):
		abort(c, http.StatusServiceUnavailable, "ERR_PAYMENT_UNAVAILABLE", "payment provider not configured")
	default:
		slog.Error("internal_error",
			"request_id", CurrentRequestID(c),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"error", err.Error(),
		)
		abort(c, http.StatusInternalServerError, "ERR_INTERNAL", "internal error")
	}
}
