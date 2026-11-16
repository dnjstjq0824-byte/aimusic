package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/service"
)

// AuthHandler serves registration, login, refresh, password recovery,
// and social login.
type AuthHandler struct {
	Auth               *service.AuthService
	Users              userGetter
	Resets             *service.PasswordResetService
	EmailVerify        *service.EmailVerifyService
	OAuth              *service.OAuthService
	FrontendBase       string
	OAuthSecureCookies bool
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

type registerBusinessRequest struct {
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required,min=8"`
	DisplayName  string `json:"displayName"`
	Company      string `json:"company" binding:"required"`
	BizNo        string `json:"bizNo"`
	Country      string `json:"country"`
	Industry     string `json:"industry"`
	BirthYear    *int   `json:"birthYear"`
	AgeConfirmed bool   `json:"ageConfirmed"`
	InviteCode   string `json:"inviteCode"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	TOTPCode string `json:"totpCode"`
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

// RegisterBusiness creates a business account with a company profile.
func (h *AuthHandler) RegisterBusiness(c *gin.Context) {
	var req registerBusinessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	u, p, err := h.Auth.RegisterBusiness(c.Request.Context(), req.Email, req.Password, req.DisplayName,
		req.Company, req.BizNo, req.Country, req.Industry, req.BirthYear, req.AgeConfirmed, req.InviteCode)
	if err != nil {
		AbortError(c, err)
		return
	}
	created(c, gin.H{"user": u, "business": p})
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
	ok(c, gin.H{"userId": u.ID, "role": u.Role, "roles": h.Auth.EffectiveRoles(c.Request.Context(), u.ID, u.Role), "ageConfirmedAt": u.AgeConfirmedAt, "displayName": u.DisplayName, "email": u.Email, "totpEnabled": u.TOTPEnabled, "locale": u.Locale, "emailVerified": u.EmailVerifiedAt != nil})
}

// UpdateLocale changes the caller's display language (A2).
func (h *AuthHandler) UpdateLocale(c *gin.Context) {
	var req struct {
		Locale string `json:"locale"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortError(c, fmt.Errorf("%w: locale required", domain.ErrInvalidInput))
		return
	}
	u, err := h.Auth.UpdateLocale(c.Request.Context(), CurrentUserID(c), req.Locale)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"userId": u.ID, "locale": u.Locale})
}

// UpdateProfile changes the caller's display name (R8).
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	var req struct {
		DisplayName string `json:"displayName"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortError(c, fmt.Errorf("%w: displayName required", domain.ErrInvalidInput))
		return
	}
	u, err := h.Auth.UpdateDisplayName(c.Request.Context(), CurrentUserID(c), req.DisplayName)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"userId": u.ID, "displayName": u.DisplayName})
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

// RequestEmailVerify mails a fresh verification link to the caller (A2).
// Always 202 so resend buttons stay idempotent.
func (h *AuthHandler) RequestEmailVerify(c *gin.Context) {
	if err := h.EmailVerify.Request(c.Request.Context(), CurrentUserID(c)); err != nil {
		AbortError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"data": gin.H{"requested": true}})
}

type emailVerifyConfirmRequest struct {
	Token string `json:"token" binding:"required"`
}

// ConfirmEmailVerify consumes a verification link and logs the user in (A2):
// clicking the mailed link continues in a logged-in state.
func (h *AuthHandler) ConfirmEmailVerify(c *gin.Context) {
	var req emailVerifyConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	res, err := h.EmailVerify.Confirm(c.Request.Context(), req.Token)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"verified": true, "tokens": res.Tokens, "user": gin.H{"id": res.User.ID, "email": res.User.Email, "displayName": res.User.DisplayName}})
}

// OAuthProviders lists social login availability in locale display order
// (drives frontend buttons). ?locale=ko|en|ja|zh-cn|th, default en.
func (h *AuthHandler) OAuthProviders(c *gin.Context) {
	okList(c, h.OAuth.Providers(c.Query("locale")))
}

// MobileOAuthCallbackBase is the fixed app-link return for platform=app
// flows (WI-R12). It is a constant, never request-derived: no open redirect.
const MobileOAuthCallbackBase = "avibe://oauth/callback"

// OAuthStart redirects to the provider authorization page, setting the C-12
// browser-bound state cookies. ?intent=link&linkToken= binds an account-link,
// ?platform=app signs a mobile return (WI-R12).
func (h *AuthHandler) OAuthStart(c *gin.Context) {
	linkToken := ""
	if c.Query("intent") == "link" {
		linkToken = c.Query("linkToken")
	}
	start, err := h.OAuth.StartForPlatform(c.Param("provider"), linkToken, c.Query("platform"))
	if err != nil {
		AbortError(c, err)
		return
	}
	if start.SameSiteNone {
		c.SetSameSite(http.SameSiteNoneMode)
		c.SetCookie(service.OAuthStateCookie, start.Nonce, 600, "/api/v1/auth/oauth", "", true, true)
		if start.PKCEVerifier != "" {
			c.SetCookie(service.OAuthPKCECookie, start.PKCEVerifier, 600, "/api/v1/auth/oauth", "", true, true)
		}
	} else {
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(service.OAuthStateCookie, start.Nonce, 600, "/api/v1/auth/oauth", "", h.OAuthSecureCookies, true)
		if start.PKCEVerifier != "" {
			c.SetCookie(service.OAuthPKCECookie, start.PKCEVerifier, 600, "/api/v1/auth/oauth", "", h.OAuthSecureCookies, true)
		}
	}
	c.Redirect(http.StatusFound, start.URL)
}

// OAuthCallback completes social login and redirects to the SPA with a
// one-time code (C-16: no tokens in the URL). Apple posts form_post fields;
// every other provider redirects with a query string. Failures redirect to
// login with a machine-readable oauthError code. platform=app flows (WI-R12)
// redirect to the fixed avibe:// app link instead.
func (h *AuthHandler) OAuthCallback(c *gin.Context) {
	provider := c.Param("provider")
	nonceCookie, _ := c.Cookie(service.OAuthStateCookie)
	pkceCookie, _ := c.Cookie(service.OAuthPKCECookie)
	res, err := h.OAuth.Callback(c.Request.Context(), service.OAuthCallbackInput{
		Provider:     provider,
		Code:         firstNonEmpty(c.Query("code"), c.PostForm("code")),
		State:        firstNonEmpty(c.Query("state"), c.PostForm("state")),
		CookieNonce:  nonceCookie,
		PKCEVerifier: pkceCookie,
		IDToken:      c.PostForm("id_token"),
		RawUser:      c.PostForm("user"),
	})
	if err != nil {
		// The state is unverified here, so the platform hint below only
		// selects the error redirect target; it carries no secret.
		if appReturnHint(firstNonEmpty(c.Query("state"), c.PostForm("state"))) {
			c.Redirect(http.StatusFound, MobileOAuthCallbackBase+"?oauthError="+errCodeForOAuth(err))
			return
		}
		c.Redirect(http.StatusFound, h.FrontendBase+"/login?oauthError="+errCodeForOAuth(err))
		return
	}
	if res.Platform == service.OAuthPlatformApp {
		switch {
		case res.Linked:
			c.Redirect(http.StatusFound, MobileOAuthCallbackBase+"?linked="+strings.ToLower(res.Provider))
		case res.EmailToken != "":
			// Mobile has no complete-email screen yet (R12 deferred):
			// surface a typed error instead of a dead web URL.
			c.Redirect(http.StatusFound, MobileOAuthCallbackBase+"?oauthError=email_required")
		default:
			c.Redirect(http.StatusFound, MobileOAuthCallbackBase+"?code="+res.ExchangeCode)
		}
		return
	}
	switch {
	case res.Linked:
		c.Redirect(http.StatusFound, h.FrontendBase+"/account/connections?linked="+strings.ToLower(res.Provider))
	case res.EmailToken != "":
		c.Redirect(http.StatusFound, h.FrontendBase+"/signup/complete-email?token="+res.EmailToken)
	default:
		c.Redirect(http.StatusFound, h.FrontendBase+"/oauth/callback?code="+res.ExchangeCode)
	}
}

// appReturnHint peeks at the unsigned state shape for the .app marker.
// It only routes error redirects (no secret), never authorizes anything.
func appReturnHint(state string) bool {
	parts := strings.Split(state, ".")
	if len(parts) < 2 {
		return false
	}
	return parts[len(parts)-2] == service.OAuthPlatformApp
}

type oauthExchangeRequest struct {
	Code string `json:"code" binding:"required"`
}

// OAuthExchange swaps a one-time callback code for a token pair (C-16).
func (h *AuthHandler) OAuthExchange(c *gin.Context) {
	var req oauthExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	tokens, _, err := h.OAuth.ExchangeCode(c.Request.Context(), req.Code)
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"tokens": tokens})
}

type oauthCompleteEmailRequest struct {
	Token  string `json:"token" binding:"required"`
	Email  string `json:"email" binding:"required,email"`
	Locale string `json:"locale"`
}

// OAuthCompleteEmail files the typed email for a provider identity that gave
// none, then mails it a verification link (WI-A2): the account is created
// and linked when the link is clicked.
func (h *AuthHandler) OAuthCompleteEmail(c *gin.Context) {
	var req oauthCompleteEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "ERR_INVALID_INPUT", err.Error())
		return
	}
	pending, err := h.OAuth.CompleteEmail(c.Request.Context(), req.Token, req.Email)
	if err != nil {
		AbortError(c, err)
		return
	}
	if err := h.EmailVerify.RequestOAuthSignup(c.Request.Context(), req.Email, req.Token, req.Locale); err != nil {
		AbortError(c, err)
		return
	}
	ok(c, pending)
}

// OAuthLinkToken mints a short-lived token binding a link flow to the caller.
func (h *AuthHandler) OAuthLinkToken(c *gin.Context) {
	tok, err := h.OAuth.MintLinkToken(CurrentUserID(c))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"linkToken": tok})
}

// OAuthConnections lists the caller's linked providers.
func (h *AuthHandler) OAuthConnections(c *gin.Context) {
	conns, err := h.OAuth.ListConnections(c.Request.Context(), CurrentUserID(c))
	if err != nil {
		AbortError(c, err)
		return
	}
	okList(c, conns)
}

// OAuthUnlink removes one provider link (last login method is refused).
func (h *AuthHandler) OAuthUnlink(c *gin.Context) {
	if err := h.OAuth.Unlink(c.Request.Context(), CurrentUserID(c), c.Param("provider")); err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"unlinked": true})
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
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

func queryInt64(c *gin.Context, key string) int64 {
	if raw := c.Query(key); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return 0
}

func queryFloat(c *gin.Context, key string) float64 {
	if raw := c.Query(key); raw != "" {
		if n, err := strconv.ParseFloat(raw, 64); err == nil && n > 0 {
			return n
		}
	}
	return 0
}

func queryBool(c *gin.Context, key string) bool {
	switch strings.ToLower(strings.TrimSpace(c.Query(key))) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}
