package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"avibe-platform/backend/internal/config"
	"avibe-platform/backend/internal/domain"
)

type fakeOAuthUsers struct {
	byID    map[string]*domain.User
	byEmail map[string]*domain.User
	created []*domain.User
}

func (f *fakeOAuthUsers) GetByID(_ context.Context, id string) (*domain.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (f *fakeOAuthUsers) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := f.byEmail[strings.ToLower(email)]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (f *fakeOAuthUsers) Create(_ context.Context, email, hash, role, displayName string) (*domain.User, error) {
	if f.byID == nil {
		f.byID = map[string]*domain.User{}
		f.byEmail = map[string]*domain.User{}
	}
	u := &domain.User{ID: "u-new", Email: email, PasswordHash: hash, Role: role, DisplayName: displayName, IsActive: true}
	f.byID[u.ID] = u
	f.byEmail[strings.ToLower(email)] = u
	f.created = append(f.created, u)
	return u, nil
}

type fakeOAuthLinks struct {
	byUID map[string]*domain.OAuthAccount
}

func (f *fakeOAuthLinks) GetByProviderUID(_ context.Context, provider, uid string) (*domain.OAuthAccount, error) {
	a, ok := f.byUID[provider+"|"+uid]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return a, nil
}

func (f *fakeOAuthLinks) Upsert(_ context.Context, userID, provider, uid string, email *string) (*domain.OAuthAccount, error) {
	if f.byUID == nil {
		f.byUID = map[string]*domain.OAuthAccount{}
	}
	a := &domain.OAuthAccount{ID: "link-1", UserID: userID, Provider: provider, ProviderUserID: uid, Email: email}
	f.byUID[provider+"|"+uid] = a
	return a, nil
}

// stubProvider emulates token+userinfo endpoints for one provider.
func stubProvider(t *testing.T, userinfo string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/token"):
			if err := r.ParseForm(); err != nil {
				t.Errorf("parse form: %v", err)
			}
			if r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("code") == "" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{}`))
				return
			}
			_, _ = w.Write([]byte(`{"access_token":"stub-access","token_type":"Bearer"}`))
		case strings.HasSuffix(r.URL.Path, "/userinfo"):
			if r.Header.Get("Authorization") != "Bearer stub-access" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{}`))
				return
			}
			_, _ = w.Write([]byte(userinfo))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func oauthTestService(t *testing.T, googleUserinfo, kakaoUserinfo string) (*OAuthService, *fakeOAuthUsers, *fakeOAuthLinks, func()) {
	t.Helper()
	gsrv := stubProvider(t, googleUserinfo)
	ksrv := stubProvider(t, kakaoUserinfo)
	cfg := &config.Config{}
	cfg.JWT.AccessSecret = "test-access-secret-32-chars-minimum-ok"
	cfg.JWT.RefreshSecret = "test-refresh-secret-32-chars-minimum-ok"
	cfg.JWT.AccessTTLMinutes = 15
	cfg.JWT.RefreshTTLDays = 14
	cfg.OAuth.RedirectBaseURL = "https://api.example.test/api/v1/auth/oauth"
	cfg.OAuth.GoogleClientID, cfg.OAuth.GoogleClientSecret = "gid", "gsecret"
	cfg.OAuth.GoogleTokenURL = gsrv.URL + "/token"
	cfg.OAuth.GoogleUserinfoURL = gsrv.URL + "/userinfo"
	cfg.OAuth.KakaoClientID, cfg.OAuth.KakaoClientSecret = "kid", "ksecret"
	cfg.OAuth.KakaoTokenURL = ksrv.URL + "/token"
	cfg.OAuth.KakaoUserinfoURL = ksrv.URL + "/userinfo"
	cfg.OAuth.StateSecret = "test-state-secret"
	auth := NewAuthService(nil, cfg)
	users := &fakeOAuthUsers{}
	links := &fakeOAuthLinks{}
	s := &OAuthService{users: users, links: links, auth: auth, cfg: cfg.OAuth, client: gsrv.Client()}
	return s, users, links, func() { gsrv.Close(); ksrv.Close() }
}

func signedStateFor(t *testing.T, s *OAuthService, provider string) string {
	t.Helper()
	startURL, err := s.StartURL(provider)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	idx := strings.Index(startURL, "state=")
	if idx < 0 {
		t.Fatalf("no state in %q", startURL)
	}
	state := startURL[idx+len("state="):]
	if amp := strings.Index(state, "&"); amp >= 0 {
		state = state[:amp]
	}
	return state
}

func TestOAuthProvidersAndStartURL(t *testing.T) {
	s, _, _, done := oauthTestService(t, `{"sub":"g1","email":"g@example.com","name":"G"}`, `{}`)
	defer done()
	provs := s.Providers()
	if len(provs) != 2 || !provs[0].Enabled || !provs[1].Enabled {
		t.Fatalf("providers = %+v", provs)
	}
	startURL, err := s.StartURL("GOOGLE")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if !strings.HasPrefix(startURL, "https://accounts.google.com/") || !strings.Contains(startURL, "state=") {
		t.Fatalf("start url = %q", startURL)
	}
	if _, err := s.StartURL("APPLE"); err == nil {
		t.Fatalf("unknown provider accepted")
	}
}

func TestOAuthGoogleNewUser(t *testing.T) {
	s, users, links, done := oauthTestService(t,
		`{"sub":"g-123","email":"New@Example.com","name":"New User"}`, `{}`)
	defer done()
	state := signedStateFor(t, s, "GOOGLE")
	tokens, u, err := s.Callback(context.Background(), "GOOGLE", "code-1", state)
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	if u.Email != "new@example.com" || u.Role != domain.RoleUser || u.DisplayName != "New User" {
		t.Fatalf("user = %+v", u)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("tokens missing")
	}
	if _, err := links.GetByProviderUID(context.Background(), "GOOGLE", "g-123"); err != nil {
		t.Fatalf("link not stored: %v", err)
	}
	if len(users.created) != 1 {
		t.Fatalf("created = %d", len(users.created))
	}
	// Second login reuses the link without creating another user.
	tokens2, u2, err := s.Callback(context.Background(), "GOOGLE", "code-2", signedStateFor(t, s, "GOOGLE"))
	if err != nil {
		t.Fatalf("re-login: %v", err)
	}
	if u2.ID != u.ID || tokens2.AccessToken == "" || len(users.created) != 1 {
		t.Fatalf("re-login mismatch: %+v created=%d", u2, len(users.created))
	}
}

func TestOAuthKakaoLinksExistingEmail(t *testing.T) {
	s, users, _, done := oauthTestService(t, `{}`,
		`{"id":4242,"kakao_account":{"email":"Kakao@Example.com"},"properties":{"nickname":"카카오닉"}}`)
	defer done()
	users.byID = map[string]*domain.User{"u-9": {ID: "u-9", Email: "kakao@example.com", Role: "USER", IsActive: true}}
	users.byEmail = map[string]*domain.User{"kakao@example.com": users.byID["u-9"]}
	_, u, err := s.Callback(context.Background(), "KAKAO", "code-1", signedStateFor(t, s, "KAKAO"))
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	if u.ID != "u-9" || len(users.created) != 0 {
		t.Fatalf("existing email not linked: %+v created=%d", u, len(users.created))
	}
}

func TestOAuthRejectsTamperedState(t *testing.T) {
	s, _, _, done := oauthTestService(t, `{"sub":"g1","email":"g@example.com"}`, `{}`)
	defer done()
	if _, _, err := s.Callback(context.Background(), "GOOGLE", "code-1", "nonce.deadbeef"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("tampered state err = %v", err)
	}
	if _, _, err := s.Callback(context.Background(), "GOOGLE", "", signedStateFor(t, s, "GOOGLE")); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("empty code err = %v", err)
	}
}

func TestOAuthRequiresEmail(t *testing.T) {
	s, _, _, done := oauthTestService(t, `{}`,
		`{"id":77,"kakao_account":{},"properties":{"nickname":"noname"}}`)
	defer done()
	_, _, err := s.Callback(context.Background(), "KAKAO", "code-1", signedStateFor(t, s, "KAKAO"))
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("missing email err = %v", err)
	}
}
