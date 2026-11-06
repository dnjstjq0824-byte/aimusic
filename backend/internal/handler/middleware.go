// Package handler exposes the Gin REST API with JWT, RBAC, rate-limit, and audit middleware.
package handler

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"ai-music-platform/backend/internal/domain"
	"ai-music-platform/backend/internal/service"
)

// Context keys for the authenticated principal.
const (
	ctxUserIDKey = "auth.userID"
	ctxRoleKey   = "auth.role"
)

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

// AuditLog records method, path, status, principal, and latency per request.
func AuditLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Printf("audit method=%s path=%s status=%d user=%s role=%s ip=%s latency=%s",
			c.Request.Method, c.Request.URL.Path, c.Writer.Status(),
			CurrentUserID(c), CurrentRole(c), c.ClientIP(), time.Since(start))
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
	default:
		log.Printf("internal error: %v", err)
		abort(c, http.StatusInternalServerError, "ERR_INTERNAL", "internal error")
	}
}
