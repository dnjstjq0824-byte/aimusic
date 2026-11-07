package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ai-music-platform/backend/internal/service"
)

// AuthHandler serves registration, login, refresh, password recovery,
// and social login.
type AuthHandler struct {
	Auth         *service.AuthService
	Users        userGetter
	Resets       *service.PasswordResetService
	OAuth        *service.OAuthService
	FrontendBase string
}

type userGetter interface {
	Me(userID string) (any, error)
}

type registerRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	DisplayName string `json:"displayName" binding:"required"`
}

type registerPartnerRequest struct {
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required,min=8"`
	DisplayName  string `json:"displayName" binding:"required"`
	Company      string `json:"company" binding:"required"`
	ContactEmail string `json:"contactEmail"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// Register creates a listener account.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	u, err := h.Auth.Register(c.Request.Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		AbortError(c, err)
		return
	}
	created(c, u)
}

// RegisterPartner creates a partner account with a pending profile.
func (h *AuthHandler) RegisterPartner(c *gin.Context) {
	var req registerPartnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	u, p, err := h.Auth.RegisterPartner(c.Request.Context(), req.Email, req.Password, req.DisplayName, req.Company, req.ContactEmail)
	if err != nil {
		AbortError(c, err)
		return
	}
	created(c, gin.H{"user": u, "partner": p})
}

// Login issues a token pair.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	tokens, user, err := h.Auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"tokens": tokens, "user": user})
}

// Refresh rotates tokens.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	tokens, err := h.Auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"tokens": tokens})
}

// Me returns the authenticated principal.
func (h *AuthHandler) Me(c *gin.Context) {
	ok(c, gin.H{"userId": CurrentUserID(c), "role": CurrentRole(c)})
}

// Withdraw deactivates the authenticated user's account immediately.
func (h *AuthHandler) Withdraw(c *gin.Context) {
	if err := h.Auth.Withdraw(c.Request.Context(), CurrentUserID(c)); err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"withdrawn": true})
}

type passwordResetRequestRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// RequestPasswordReset issues a reset link. It always succeeds so callers
// cannot enumerate accounts.
func (h *AuthHandler) RequestPasswordReset(c *gin.Context) {
	var req passwordResetRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	if err := h.Resets.Request(c.Request.Context(), req.Email); err != nil {
		AbortError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"data": gin.H{"requested": true}})
}

type passwordResetConfirmRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}

// ConfirmPasswordReset consumes a reset token and sets a new password.
func (h *AuthHandler) ConfirmPasswordReset(c *gin.Context) {
	var req passwordResetConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	if err := h.Resets.Confirm(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"reset": true})
}

// OAuthProviders lists social login availability (drives frontend buttons).
func (h *AuthHandler) OAuthProviders(c *gin.Context) {
	ok(c, h.OAuth.Providers())
}

// OAuthStart redirects to the provider authorization page.
func (h *AuthHandler) OAuthStart(c *gin.Context) {
	startURL, err := h.OAuth.StartURL(c.Param("provider"))
	if err != nil {
		AbortError(c, err)
		return
	}
	c.Redirect(http.StatusFound, startURL)
}

// OAuthCallback completes social login and redirects to the SPA with tokens.
// Failures redirect to login with a machine-readable oauthError code.
func (h *AuthHandler) OAuthCallback(c *gin.Context) {
	provider := c.Param("provider")
	tokens, user, err := h.OAuth.Callback(c.Request.Context(), provider, c.Query("code"), c.Query("state"))
	if err != nil {
		c.Redirect(http.StatusFound, h.FrontendBase+"/login?oauthError="+errCodeForOAuth(err))
		return
	}
	c.Redirect(http.StatusFound, h.FrontendBase+"/oauth/callback?accessToken="+tokens.AccessToken+
		"&refreshToken="+tokens.RefreshToken+"&role="+user.Role)
}

func errCodeForOAuth(err error) string {
	code := errCodeOfDomain(err)
	if code == "" {
		return "ERR_INTERNAL"
	}
	return code
}

func queryInt(c *gin.Context, key string, def int) int {
	if raw := c.Query(key); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			return n
		}
	}
	return def
}
