package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"avibe-platform/backend/internal/config"
	"avibe-platform/backend/internal/domain"
)

// oauthTokens is the provider token response we care about.
type oauthTokens struct {
	AccessToken string
	IDToken     string
}

// providerIdentity is the normalized provider user. EmailVerified is true
// only when the provider cryptographically asserts the address — it gates
// the C-11 auto-link of an existing same-email account.
type providerIdentity struct {
	UID           string
	Email         string
	EmailVerified bool
	DisplayName   string
}

// oauthAdapter is one social provider (WI-A1). WeChat has no adapter yet:
// interface only, implementation deferred (needs a China entity).
type oauthAdapter interface {
	Provider() string
	Enabled(cfg config.OAuthConfig) bool
	// MissingVars names the unset required credentials (names only).
	MissingVars(cfg config.OAuthConfig) []string
	AuthURL(cfg config.OAuthConfig, redirectURI, state, challenge, oidcNonce string) string
	UsesPKCE() bool
	// SameSiteNone marks providers whose callback crosses sites (Apple
	// form_post): their state cookie needs SameSite=None; Secure.
	SameSiteNone() bool
	Exchange(ctx context.Context, s *OAuthService, redirectURI, code, verifier, state string) (*oauthTokens, error)
	Identity(ctx context.Context, s *OAuthService, toks *oauthTokens, oidcNonce, rawUser string) (*providerIdentity, error)
}

func missingEnv(names ...[2]string) []string {
	var out []string
	for _, nv := range names {
		if strings.TrimSpace(nv[1]) == "" {
			out = append(out, nv[0])
		}
	}
	return out
}

func oauthPostForm(ctx context.Context, client *http.Client, endpoint string, form url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("token exchange failed (http %d)", res.StatusCode)
	}
	return raw, nil
}

func parseTokenResponse(raw []byte) (*oauthTokens, error) {
	var tok struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if err := json.Unmarshal(raw, &tok); err != nil || tok.AccessToken == "" {
		return nil, fmt.Errorf("token exchange failed: bad payload")
	}
	return &oauthTokens{AccessToken: tok.AccessToken, IDToken: tok.IDToken}, nil
}

func oauthGetBearer(ctx context.Context, client *http.Client, endpoint, token string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("userinfo failed (http %d)", res.StatusCode)
	}
	return raw, nil
}

// --- Google ---

type googleAdapter struct{}

func (googleAdapter) Provider() string { return domain.OAuthGoogle }

func (googleAdapter) Enabled(cfg config.OAuthConfig) bool {
	return cfg.GoogleClientID != "" && cfg.GoogleClientSecret != ""
}

func (googleAdapter) MissingVars(cfg config.OAuthConfig) []string {
	return missingEnv([2]string{"GOOGLE_CLIENT_ID", cfg.GoogleClientID}, [2]string{"GOOGLE_CLIENT_SECRET", cfg.GoogleClientSecret})
}

func (googleAdapter) UsesPKCE() bool     { return true }
func (googleAdapter) SameSiteNone() bool { return false }

func (googleAdapter) AuthURL(cfg config.OAuthConfig, redirectURI, state, challenge, _ string) string {
	q := url.Values{
		"client_id": {cfg.GoogleClientID}, "redirect_uri": {redirectURI},
		"response_type": {"code"}, "scope": {"openid email profile"}, "state": {state},
		"code_challenge": {challenge}, "code_challenge_method": {"S256"},
	}
	return "https://accounts.google.com/o/oauth2/v2/auth?" + q.Encode()
}

func (googleAdapter) Exchange(ctx context.Context, s *OAuthService, redirectURI, code, verifier, _ string) (*oauthTokens, error) {
	raw, err := oauthPostForm(ctx, s.client, s.cfg.GoogleTokenURL, url.Values{
		"grant_type": {"authorization_code"}, "code": {code},
		"redirect_uri": {redirectURI},
		"client_id":    {s.cfg.GoogleClientID}, "client_secret": {s.cfg.GoogleClientSecret},
		"code_verifier": {verifier},
	})
	if err != nil {
		return nil, err
	}
	return parseTokenResponse(raw)
}

func (googleAdapter) Identity(ctx context.Context, s *OAuthService, toks *oauthTokens, _ string, _ string) (*providerIdentity, error) {
	raw, err := oauthGetBearer(ctx, s.client, s.cfg.GoogleUserinfoURL, toks.AccessToken)
	if err != nil {
		return nil, err
	}
	var g struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := json.Unmarshal(raw, &g); err != nil || g.Sub == "" {
		return nil, fmt.Errorf("userinfo failed: bad payload")
	}
	name := strings.TrimSpace(g.Name)
	if name == "" {
		name = strings.TrimSpace(g.Email)
	}
	return &providerIdentity{UID: g.Sub, Email: strings.TrimSpace(g.Email), EmailVerified: g.EmailVerified, DisplayName: name}, nil
}

// --- Kakao ---

type kakaoAdapter struct{}

func (kakaoAdapter) Provider() string { return domain.OAuthKakao }

// Enabled needs the REST API key only; the client secret stays optional
// (console "client secret" switch off by default).
func (kakaoAdapter) Enabled(cfg config.OAuthConfig) bool { return cfg.KakaoClientID != "" }

func (kakaoAdapter) MissingVars(cfg config.OAuthConfig) []string {
	return missingEnv([2]string{"KAKAO_CLIENT_ID", cfg.KakaoClientID})
}

func (kakaoAdapter) UsesPKCE() bool     { return true }
func (kakaoAdapter) SameSiteNone() bool { return false }

func (kakaoAdapter) AuthURL(cfg config.OAuthConfig, redirectURI, state, challenge, _ string) string {
	q := url.Values{
		"client_id": {cfg.KakaoClientID}, "redirect_uri": {redirectURI},
		"response_type": {"code"}, "scope": {"profile_nickname account_email"}, "state": {state},
		"code_challenge": {challenge}, "code_challenge_method": {"S256"},
	}
	return "https://kauth.kakao.com/oauth/authorize?" + q.Encode()
}

func (kakaoAdapter) Exchange(ctx context.Context, s *OAuthService, redirectURI, code, verifier, _ string) (*oauthTokens, error) {
	form := url.Values{
		"grant_type": {"authorization_code"}, "code": {code},
		"redirect_uri": {redirectURI}, "client_id": {s.cfg.KakaoClientID},
		"code_verifier": {verifier},
	}
	if strings.TrimSpace(s.cfg.KakaoClientSecret) != "" {
		form.Set("client_secret", s.cfg.KakaoClientSecret)
	}
	raw, err := oauthPostForm(ctx, s.client, s.cfg.KakaoTokenURL, form)
	if err != nil {
		return nil, err
	}
	return parseTokenResponse(raw)
}

func (kakaoAdapter) Identity(ctx context.Context, s *OAuthService, toks *oauthTokens, _ string, _ string) (*providerIdentity, error) {
	raw, err := oauthGetBearer(ctx, s.client, s.cfg.KakaoUserinfoURL, toks.AccessToken)
	if err != nil {
		return nil, err
	}
	var k struct {
		ID           int64 `json:"id"`
		KakaoAccount struct {
			Email           string `json:"email"`
			IsEmailVerified bool   `json:"is_email_verified"`
			IsEmailValid    bool   `json:"is_email_valid"`
		} `json:"kakao_account"`
		Properties struct {
			Nickname string `json:"nickname"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &k); err != nil || k.ID == 0 {
		return nil, fmt.Errorf("userinfo failed: bad payload")
	}
	name := strings.TrimSpace(k.Properties.Nickname)
	if name == "" {
		name = fmt.Sprintf("kakao-%d", k.ID)
	}
	return &providerIdentity{
		UID: fmt.Sprintf("%d", k.ID), Email: strings.TrimSpace(k.KakaoAccount.Email),
		EmailVerified: k.KakaoAccount.IsEmailVerified && k.KakaoAccount.IsEmailValid,
		DisplayName:   name,
	}, nil
}

// --- Apple ---

type appleAdapter struct{}

func (appleAdapter) Provider() string { return domain.OAuthApple }

func (appleAdapter) Enabled(cfg config.OAuthConfig) bool {
	return cfg.AppleServicesID != "" && cfg.AppleTeamID != "" && cfg.AppleKeyID != "" && cfg.ApplePrivateKey != ""
}

func (appleAdapter) MissingVars(cfg config.OAuthConfig) []string {
	return missingEnv(
		[2]string{"APPLE_SERVICES_ID", cfg.AppleServicesID},
		[2]string{"APPLE_TEAM_ID", cfg.AppleTeamID},
		[2]string{"APPLE_KEY_ID", cfg.AppleKeyID},
		[2]string{"APPLE_PRIVATE_KEY", cfg.ApplePrivateKey},
	)
}

func (appleAdapter) UsesPKCE() bool     { return true }
func (appleAdapter) SameSiteNone() bool { return true }

func (appleAdapter) AuthURL(cfg config.OAuthConfig, redirectURI, state, challenge, oidcNonce string) string {
	q := url.Values{
		"client_id": {cfg.AppleServicesID}, "redirect_uri": {redirectURI},
		"response_type": {"code id_token"}, "response_mode": {"form_post"},
		"scope": {"name email"}, "state": {state}, "nonce": {oidcNonce},
		"code_challenge": {challenge}, "code_challenge_method": {"S256"},
	}
	return "https://appleid.apple.com/auth/authorize?" + q.Encode()
}

func (appleAdapter) Exchange(ctx context.Context, s *OAuthService, redirectURI, code, verifier, _ string) (*oauthTokens, error) {
	secret, err := appleClientSecret(s.cfg.AppleTeamID, s.cfg.AppleKeyID, s.cfg.AppleServicesID, s.cfg.ApplePrivateKey, s.clock())
	if err != nil {
		return nil, err
	}
	raw, err := oauthPostForm(ctx, s.client, s.cfg.AppleTokenURL, url.Values{
		"grant_type": {"authorization_code"}, "code": {code},
		"redirect_uri": {redirectURI}, "client_id": {s.cfg.AppleServicesID},
		"client_secret": {secret}, "code_verifier": {verifier},
	})
	if err != nil {
		return nil, err
	}
	return parseTokenResponse(raw)
}

func (appleAdapter) Identity(ctx context.Context, s *OAuthService, toks *oauthTokens, oidcNonce, rawUser string) (*providerIdentity, error) {
	claims, err := s.verifyIDToken(ctx, s.cfg.AppleJWKSURL, toks.IDToken, appleIssuer, s.cfg.AppleServicesID, oidcNonce)
	if err != nil {
		return nil, err
	}
	sub := claimString(claims, "sub")
	if sub == "" {
		return nil, fmt.Errorf("userinfo failed: bad payload")
	}
	email := strings.TrimSpace(claimString(claims, "email"))
	name := appleDisplayName(rawUser)
	if name == "" {
		name = email
	}
	return &providerIdentity{UID: sub, Email: email, EmailVerified: claimBoolish(claims, "email_verified"), DisplayName: name}, nil
}

// appleDisplayName reads the first-auth-only `user` form field
// ({"name":{"firstName":"..","lastName":".."},"email":".."}).
func appleDisplayName(rawUser string) string {
	if strings.TrimSpace(rawUser) == "" {
		return ""
	}
	var u struct {
		Name struct {
			First string `json:"firstName"`
			Last  string `json:"lastName"`
		} `json:"name"`
	}
	if err := json.Unmarshal([]byte(rawUser), &u); err != nil {
		return ""
	}
	return strings.TrimSpace(strings.TrimSpace(u.Name.First) + " " + strings.TrimSpace(u.Name.Last))
}

// --- Naver ---

type naverAdapter struct{}

func (naverAdapter) Provider() string { return domain.OAuthNaver }

func (naverAdapter) Enabled(cfg config.OAuthConfig) bool {
	return cfg.NaverClientID != "" && cfg.NaverClientSecret != ""
}

func (naverAdapter) MissingVars(cfg config.OAuthConfig) []string {
	return missingEnv([2]string{"NAVER_CLIENT_ID", cfg.NaverClientID}, [2]string{"NAVER_CLIENT_SECRET", cfg.NaverClientSecret})
}

// Naver's authorize/token endpoints speak plain code flow (no PKCE).
func (naverAdapter) UsesPKCE() bool     { return false }
func (naverAdapter) SameSiteNone() bool { return false }

func (naverAdapter) AuthURL(cfg config.OAuthConfig, redirectURI, state, _, _ string) string {
	q := url.Values{
		"response_type": {"code"}, "client_id": {cfg.NaverClientID},
		"redirect_uri": {redirectURI}, "state": {state},
	}
	return "https://nid.naver.com/oauth2.0/authorize?" + q.Encode()
}

func (naverAdapter) Exchange(ctx context.Context, s *OAuthService, _ string, code, _ string, state string) (*oauthTokens, error) {
	raw, err := oauthPostForm(ctx, s.client, s.cfg.NaverTokenURL, url.Values{
		"grant_type": {"authorization_code"}, "client_id": {s.cfg.NaverClientID},
		"client_secret": {s.cfg.NaverClientSecret}, "code": {code}, "state": {state},
	})
	if err != nil {
		return nil, err
	}
	return parseTokenResponse(raw)
}

func (naverAdapter) Identity(ctx context.Context, s *OAuthService, toks *oauthTokens, _ string, _ string) (*providerIdentity, error) {
	raw, err := oauthGetBearer(ctx, s.client, s.cfg.NaverUserinfoURL, toks.AccessToken)
	if err != nil {
		return nil, err
	}
	var n struct {
		ResultCode string `json:"resultcode"`
		Response   struct {
			ID       string `json:"id"`
			Email    string `json:"email"`
			Name     string `json:"name"`
			Nickname string `json:"nickname"`
		} `json:"response"`
	}
	if err := json.Unmarshal(raw, &n); err != nil || n.ResultCode != "00" || n.Response.ID == "" {
		return nil, fmt.Errorf("userinfo failed: bad payload")
	}
	name := strings.TrimSpace(n.Response.Name)
	if name == "" {
		name = strings.TrimSpace(n.Response.Nickname)
	}
	if name == "" {
		name = strings.TrimSpace(n.Response.Email)
	}
	// Naver never asserts address ownership here: email consent is optional,
	// so C-11 auto-link stays off and consent refusers take the email flow.
	return &providerIdentity{UID: n.Response.ID, Email: strings.TrimSpace(n.Response.Email), EmailVerified: false, DisplayName: name}, nil
}

// --- LINE ---

type lineAdapter struct{}

func (lineAdapter) Provider() string { return domain.OAuthLine }

func (lineAdapter) Enabled(cfg config.OAuthConfig) bool {
	return cfg.LineChannelID != "" && cfg.LineChannelSecret != ""
}

func (lineAdapter) MissingVars(cfg config.OAuthConfig) []string {
	return missingEnv([2]string{"LINE_CHANNEL_ID", cfg.LineChannelID}, [2]string{"LINE_CHANNEL_SECRET", cfg.LineChannelSecret})
}

func (lineAdapter) UsesPKCE() bool     { return true }
func (lineAdapter) SameSiteNone() bool { return false }

func (lineAdapter) AuthURL(cfg config.OAuthConfig, redirectURI, state, challenge, oidcNonce string) string {
	q := url.Values{
		"response_type": {"code"}, "client_id": {cfg.LineChannelID},
		"redirect_uri": {redirectURI}, "state": {state},
		"scope": {"openid profile email"}, "nonce": {oidcNonce},
		"code_challenge": {challenge}, "code_challenge_method": {"S256"},
	}
	return "https://access.line.me/oauth2/v2.1/authorize?" + q.Encode()
}

func (lineAdapter) Exchange(ctx context.Context, s *OAuthService, redirectURI, code, verifier, _ string) (*oauthTokens, error) {
	raw, err := oauthPostForm(ctx, s.client, s.cfg.LineTokenURL, url.Values{
		"grant_type": {"authorization_code"}, "code": {code},
		"redirect_uri": {redirectURI}, "client_id": {s.cfg.LineChannelID},
		"client_secret": {s.cfg.LineChannelSecret}, "code_verifier": {verifier},
	})
	if err != nil {
		return nil, err
	}
	return parseTokenResponse(raw)
}

func (lineAdapter) Identity(ctx context.Context, s *OAuthService, toks *oauthTokens, oidcNonce, _ string) (*providerIdentity, error) {
	claims, err := s.verifyIDToken(ctx, s.cfg.LineJWKSURL, toks.IDToken, lineIssuer, s.cfg.LineChannelID, oidcNonce)
	if err != nil {
		return nil, err
	}
	sub := claimString(claims, "sub")
	if sub == "" {
		return nil, fmt.Errorf("userinfo failed: bad payload")
	}
	email := strings.TrimSpace(claimString(claims, "email"))
	name := strings.TrimSpace(claimString(claims, "name"))
	if name == "" {
		name = email
	}
	// LINE's id_token carries no email_verified assertion (and the email
	// scope needs a console permission grant), so C-11 auto-link stays off.
	return &providerIdentity{UID: sub, Email: email, EmailVerified: false, DisplayName: name}, nil
}
