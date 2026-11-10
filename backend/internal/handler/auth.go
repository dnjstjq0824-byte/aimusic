package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"avibe-platform/backend/internal/service"
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
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required,min=8"`
	DisplayName  string `json:"displayName" binding:"required"`
	BirthYear    *int   `json:"birthYear"`
	AgeConfirmed bool   `json:"ageConfirmed"`
	InviteCode   string `json:"inviteCode"`
}

type registerPartnerRequest struct {
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required,min=8"`
	DisplayName  string `json:"displayName" binding:"required"`
	Company      string `json:"company" binding:"required"`
	ContactEmail string `json:"contactEmail"`
	BirthYear    *int   `json:"birthYear"`
	AgeConfirmed bool   `json:"ageConfirmed"`
	InviteCode   string `json:"inviteCode"`
}

type loginRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required"`
	TOTPCode  string `json:"totpCode"`
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
	u, err := h.Auth.Register(c.Request.Context(), req.Email, req.Password, req.DisplayName, req.BirthYear, req.AgeConfirmed, req.InviteCode)
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
	u, p, err := h.Auth.RegisterPartner(c.Request.Context(), req.Email, req.Password, req.DisplayName, req.Company, req.ContactEmail, req.BirthYear, req.AgeConfirmed, req.InviteCode)
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
	tokens, user, err := h.Auth.Login(c.Request.Context(), req.Email, req.Password, req.TOTPCode)
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
	u, err := h.Auth.Profile(c.Request.Context(), CurrentUserID(c))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"userId": u.ID, "role": u.Role, "ageConfirmedAt": u.AgeConfirmedAt})
}

// ExportData returns the caller's personal-data bundle as a download.
func (h *AuthHandler) ExportData(c *gin.Context) {
	bundle, err := h.Auth.ExportData(c.Request.Context(), CurrentUserID(c))
	if err != nil {
		AbortError(c, err)
		return
	}
	c.Header("Content-Disposition", "attachment; filename=avibe-my-data.json")
	ok(c, bundle)
}

type ageConfirmRequest struct {
	AgeConfirmed bool `json:"ageConfirmed"`
	BirthYear    *int `json:"birthYear"`
}

// ConfirmAge records the 14+ self-declaration (required after social signup).
func (h *AuthHandler) ConfirmAge(c *gin.Context) {
	var req ageConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	if err := h.Auth.ConfirmAge(c.Request.Context(), CurrentUserID(c), req.AgeConfirmed, req.BirthYear); err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"confirmed": true})
}

// Withdraw deactivates the authenticated user's account immediately.
func (h *AuthHandler) Withdraw(c *gin.Context) {
	if err := h.Auth.Withdraw(c.Request.Context(), CurrentUserID(c)); err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"withdrawn": true})
}

// LogoutAll revokes every outstanding token for the authenticated account.
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	if err := h.Auth.LogoutAll(c.Request.Context(), CurrentUserID(c)); err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"loggedOutAll": true})
}

// EnrollTOTP stages a fresh second-factor secret (shown once).
func (h *AuthHandler) EnrollTOTP(c *gin.Context) {
	enrollment, err := h.Auth.EnrollTOTP(c.Request.Context(), CurrentUserID(c))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, enrollment)
}

type totpCodeRequest struct {
	Code string `json:"code" binding:"required"`
}

// ConfirmTOTPEnrollment activates 2FA and returns single-use recovery codes.
func (h *AuthHandler) ConfirmTOTPEnrollment(c *gin.Context) {
	var req totpCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	codes, err := h.Auth.ConfirmTOTPEnrollment(c.Request.Context(), CurrentUserID(c), req.Code)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"recoveryCodes": codes})
}

type totpDisableRequest struct {
	Password string `json:"password" binding:"required"`
}

// DisableTOTP removes 2FA after password re-verification.
func (h *AuthHandler) DisableTOTP(c *gin.Context) {
	var req totpDisableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	if err := h.Auth.DisableTOTP(c.Request.Context(), CurrentUserID(c), req.Password); err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"disabled": true})
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
