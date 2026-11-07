package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"ai-music-platform/backend/internal/config"
	"ai-music-platform/backend/internal/domain"
	"ai-music-platform/backend/internal/repository"
)

// oauthUserStore is the user surface social login needs.
type oauthUserStore interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Create(ctx context.Context, email, passwordHash, role, displayName string) (*domain.User, error)
}

// oauthLinkStore is the provider-link surface social login needs.
type oauthLinkStore interface {
	GetByProviderUID(ctx context.Context, provider, providerUID string) (*domain.OAuthAccount, error)
	Upsert(ctx context.Context, userID, provider, providerUID string, email *string) (*domain.OAuthAccount, error)
}

// OAuthProviderInfo describes one social provider's availability.
type OAuthProviderInfo struct {
	Provider string `json:"provider"`
	Enabled  bool   `json:"enabled"`
}

// OAuthService runs the Google/Kakao authorization-code flow. Social accounts
// always land on the USER role; partners onboard through the email flow with
// business verification.
type OAuthService struct {
	users  oauthUserStore
	links  oauthLinkStore
	auth   *AuthService
	cfg    config.OAuthConfig
	client *http.Client
}

// NewOAuthService builds the service; httpClient may be nil for the default.
func NewOAuthService(repos *repository.Repos, auth *AuthService, cfg *config.Config, httpClient *http.Client) *OAuthService {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	oauthCfg := cfg.OAuth
	if strings.TrimSpace(oauthCfg.StateSecret) == "" {
		oauthCfg.StateSecret = cfg.JWT.AccessSecret
	}
	return &OAuthService{users: repos.Users, links: repos.OAuth, auth: auth, cfg: oauthCfg, client: httpClient}
}

// Providers reports social provider availability (enabled = ID + secret set).
func (s *OAuthService) Providers() []OAuthProviderInfo {
	return []OAuthProviderInfo{
		{Provider: domain.OAuthGoogle, Enabled: s.cfg.GoogleClientID != "" && s.cfg.GoogleClientSecret != ""},
		{Provider: domain.OAuthKakao, Enabled: s.cfg.KakaoClientID != "" && s.cfg.KakaoClientSecret != ""},
	}
}

// IsEnabled reports whether a provider is configured.
func (s *OAuthService) IsEnabled(provider string) bool {
	switch provider {
	case domain.OAuthGoogle:
		return s.cfg.GoogleClientID != "" && s.cfg.GoogleClientSecret != ""
	case domain.OAuthKakao:
		return s.cfg.KakaoClientID != "" && s.cfg.KakaoClientSecret != ""
	default:
		return false
	}
}

func (s *OAuthService) redirectURI(provider string) string {
	return s.cfg.RedirectBaseURL + "/" + strings.ToLower(provider) + "/callback"
}

// StartURL builds the provider authorization URL with a signed state.
func (s *OAuthService) StartURL(provider string) (string, error) {
	provider = strings.ToUpper(strings.TrimSpace(provider))
	if !s.IsEnabled(provider) {
		return "", fmt.Errorf("%w: provider", domain.ErrInvalidInput)
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	state := s.signState(hex.EncodeToString(nonce))
	switch provider {
	case domain.OAuthGoogle:
		q := url.Values{
			"client_id": {s.cfg.GoogleClientID}, "redirect_uri": {s.redirectURI(provider)},
			"response_type": {"code"}, "scope": {"openid email profile"}, "state": {state},
		}
		return "https://accounts.google.com/o/oauth2/v2/auth?" + q.Encode(), nil
	case domain.OAuthKakao:
		q := url.Values{
			"client_id": {s.cfg.KakaoClientID}, "redirect_uri": {s.redirectURI(provider)},
			"response_type": {"code"}, "scope": {"profile_nickname account_email"}, "state": {state},
		}
		return "https://kauth.kakao.com/oauth/authorize?" + q.Encode(), nil
	default:
		return "", fmt.Errorf("%w: provider", domain.ErrInvalidInput)
	}
}

func (s *OAuthService) signState(nonce string) string {
	mac := hmac.New(sha256.New, []byte(s.cfg.StateSecret))
	mac.Write([]byte(nonce))
	return nonce + "." + hex.EncodeToString(mac.Sum(nil))
}

// verifyState checks the signed round-trip state.
func (s *OAuthService) verifyState(state string) bool {
	nonce, sig, ok := strings.Cut(state, ".")
	if !ok || nonce == "" || sig == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(s.cfg.StateSecret))
	mac.Write([]byte(nonce))
	want := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(want))
}

type oauthTokenResponse struct {
	AccessToken string `json:"access_token"`
}

// exchange swaps an authorization code for a provider access token.
func (s *OAuthService) exchange(ctx context.Context, provider, code string) (string, error) {
	var tokenURL, clientID, clientSecret string
	switch provider {
	case domain.OAuthGoogle:
		tokenURL, clientID, clientSecret = s.cfg.GoogleTokenURL, s.cfg.GoogleClientID, s.cfg.GoogleClientSecret
	case domain.OAuthKakao:
		tokenURL, clientID, clientSecret = s.cfg.KakaoTokenURL, s.cfg.KakaoClientID, s.cfg.KakaoClientSecret
	default:
		return "", fmt.Errorf("%w: provider", domain.ErrInvalidInput)
	}
	form := url.Values{
		"grant_type": {"authorization_code"}, "code": {code},
		"redirect_uri": {s.redirectURI(provider)},
		"client_id":    {clientID}, "client_secret": {clientSecret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("token exchange failed (http %d)", res.StatusCode)
	}
	var tok oauthTokenResponse
	if err := json.Unmarshal(raw, &tok); err != nil || tok.AccessToken == "" {
		return "", fmt.Errorf("token exchange failed: bad payload")
	}
	return tok.AccessToken, nil
}

// providerIdentity is the normalized provider user.
type providerIdentity struct {
	UID         string
	Email       string
	DisplayName string
}

// fetchIdentity loads the provider user profile.
func (s *OAuthService) fetchIdentity(ctx context.Context, provider, accessToken string) (*providerIdentity, error) {
	var userinfoURL string
	switch provider {
	case domain.OAuthGoogle:
		userinfoURL = s.cfg.GoogleUserinfoURL
	case domain.OAuthKakao:
		userinfoURL = s.cfg.KakaoUserinfoURL
	default:
		return nil, fmt.Errorf("%w: provider", domain.ErrInvalidInput)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userinfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := s.client.Do(req)
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
	if provider == domain.OAuthGoogle {
		var g struct {
			Sub   string `json:"sub"`
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		if err := json.Unmarshal(raw, &g); err != nil || g.Sub == "" {
			return nil, fmt.Errorf("userinfo failed: bad payload")
		}
		name := strings.TrimSpace(g.Name)
		if name == "" {
			name = g.Email
		}
		return &providerIdentity{UID: g.Sub, Email: strings.TrimSpace(g.Email), DisplayName: name}, nil
	}
	var k struct {
		ID           int64 `json:"id"`
		KakaoAccount struct {
			Email string `json:"email"`
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
		UID: fmt.Sprintf("%d", k.ID), Email: strings.TrimSpace(k.KakaoAccount.Email), DisplayName: name,
	}, nil
}

// Callback verifies state, exchanges the code, links or creates the user,
// and issues our token pair.
func (s *OAuthService) Callback(ctx context.Context, provider, code, state string) (*Tokens, *domain.User, error) {
	provider = strings.ToUpper(strings.TrimSpace(provider))
	if !s.IsEnabled(provider) {
		return nil, nil, fmt.Errorf("%w: provider", domain.ErrInvalidInput)
	}
	if strings.TrimSpace(code) == "" || !s.verifyState(state) {
		return nil, nil, domain.ErrInvalidCredentials
	}
	accessToken, err := s.exchange(ctx, provider, code)
	if err != nil {
		return nil, nil, err
	}
	ident, err := s.fetchIdentity(ctx, provider, accessToken)
	if err != nil {
		return nil, nil, err
	}
	if ident.Email == "" {
		return nil, nil, fmt.Errorf("%w: provider email consent required", domain.ErrInvalidInput)
	}
	if link, err := s.links.GetByProviderUID(ctx, provider, ident.UID); err == nil {
		u, err := s.users.GetByID(ctx, link.UserID)
		if err != nil {
			return nil, nil, domain.ErrInvalidCredentials
		}
		if !u.IsActive {
			return nil, nil, domain.ErrUserInactive
		}
		tokens, err := s.auth.Issue(u.ID, u.Role)
		if err != nil {
			return nil, nil, err
		}
		return tokens, u, nil
	}
	email := strings.ToLower(ident.Email)
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		randomPW := make([]byte, 32)
		if _, err := rand.Read(randomPW); err != nil {
			return nil, nil, err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(randomPW)), bcrypt.DefaultCost)
		if err != nil {
			return nil, nil, err
		}
		u, err = s.users.Create(ctx, email, string(hash), domain.RoleUser, ident.DisplayName)
		if err != nil {
			return nil, nil, err
		}
	} else if !u.IsActive {
		return nil, nil, domain.ErrUserInactive
	}
	if _, err := s.links.Upsert(ctx, u.ID, provider, ident.UID, &email); err != nil {
		return nil, nil, err
	}
	tokens, err := s.auth.Issue(u.ID, u.Role)
	if err != nil {
		return nil, nil, err
	}
	return tokens, u, nil
}
