package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"avibe-platform/backend/internal/domain"
)

// withPrincipal stubs an authenticated principal with primary + grants.
func withPrincipal(primary string, grants ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(ctxUserIDKey, "user-1")
		c.Set(ctxRoleKey, primary)
		roles := append([]string{primary}, grants...)
		c.Set(ctxRolesKey, roles)
		c.Next()
	}
}

func TestRequireRoleHonorsGrants(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/console", withPrincipal(domain.RolePartner, domain.RoleBusiness), RequireRole(domain.RoleBusiness), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	r.GET("/studio", withPrincipal(domain.RoleBusiness), RequireRole(domain.RolePartner, domain.RoleAdmin), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	r.GET("/legacy", func(c *gin.Context) {
		c.Set(ctxUserIDKey, "user-1")
		c.Set(ctxRoleKey, domain.RoleAdmin)
		c.Next()
	}, RequireRole(domain.RoleAdmin), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/console", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("granted BUSINESS role rejected: %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/studio", nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("business-only studio access = %d, want 403", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/legacy", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("primary-only principal rejected: %d", w.Code)
	}
}

func TestCurrentRolesFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set(ctxRoleKey, domain.RoleUser)
	roles := CurrentRoles(c)
	if len(roles) != 1 || roles[0] != domain.RoleUser {
		t.Fatalf("roles = %v, want [USER]", roles)
	}
}
