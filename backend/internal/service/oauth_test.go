package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"avibe-platform/backend/internal/config"
	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/repository"
)

type fakeOAuthUsers struct {
	byID     map[string]*domain.User
	byEmail  map[string]*domain.User
	created  []*domain.User
	emailErr error
}

func (f *fakeOAuthUsers) GetByID(_ context.Context, id string) (*domain.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (f *fakeOAuthUsers) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	if f.emailErr != nil {
		return nil, f.emailErr
	}
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

func (f *fakeOAuthUsers) GrantRole(_ context.Context, _, _, _ string) error { return nil }

type fakeOAuthLinks struct {
	byUID  map[string]*domain.OAuthAccount
	getErr error
}

func (f *fakeOAuthLinks) GetByProviderUID(_ context.Context, provider, uid string) (*domain.OAuthAccount, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
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

func (f *fakeOAuthLinks) ListByUserID(_ context.Context, userID string) ([]*domain.OAuthAccount, error) {
	var out []*domain.OAuthAccount
	for _, a := range f.byUID {
		if a.UserID == userID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (f *fakeOAuthLinks) DeleteByUserProvider(_ context.Context, userID, provider string) error {
	for k, a := range f.byUID {
		if a.UserID == userID && a.Provider == provider {
			delete(f.byUID, k)
			return nil
		}
	}
	return domain.ErrNotFound
}

// stubWorld emulates token+userinfo+jwks endpoints for all five providers.
// Paths: /{google|kakao|apple|naver|line}/token|userinfo|jwks.
type stubWorld struct {
	t        *testing.T
	mu       sync.Mutex
	userinfo map[string]string
	idTokens map[string]string
	forms    map[string]url.Values
	rsaKey   *rsa.PrivateKey
	kid      string
	srv      *httptest.Server
}

func newStubWorld(t *testing.T) *stubWorld {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa key: %v", err)
	}
	w := &stubWorld{
		t: t, userinfo: map[string]string{}, idTokens: map[string]string{},
		forms: map[string]url.Values{}, rsaKey: key, kid: "stub-kid-1",
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 2 {
			http.NotFound(rw, r)
			return
		}
		prov, kind := strings.ToUpper(parts[0]), parts[1]
		rw.Header().Set("Content-Type", "application/json")
		switch kind {
		case "token":
			if err := r.ParseForm(); err != nil {
				t.Errorf("parse form: %v", err)
			}
			if r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("code") == "" {
				rw.WriteHeader(http.StatusBadRequest)
				_, _ = rw.Write([]byte(`{}`))
				return
			}
			w.mu.Lock()
			w.forms[prov] = r.Form
			idTok := w.idTokens[prov]
			w.mu.Unlock()
			resp := map[string]string{"access_token": "stub-access", "token_type": "Bearer"}
			if idTok != "" {
				resp["id_token"] = idTok
			}
			_ = json.NewEncoder(rw).Encode(resp)
		case "userinfo":
			if r.Header.Get("Authorization") != "Bearer stub-access" {
				rw.WriteHeader(http.StatusUnauthorized)
				_, _ = rw.Write([]byte(`{}`))
				return
			}
			w.mu.Lock()
			body := w.userinfo[prov]
			w.mu.Unlock()
			_, _ = rw.Write([]byte(body))
		case "jwks":
			n := base64.RawURLEncoding.EncodeToString(w.rsaKey.PublicKey.N.Bytes())
			e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(w.rsaKey.PublicKey.E)).Bytes())
			_ = json.NewEncoder(rw).Encode(map[string]any{"keys": []any{map[string]string{
				"kty": "RSA", "kid": w.kid, "use": "sig", "alg": "RS256", "n": n, "e": e,
			}}})
		default:
			http.NotFound(rw, r)
		}
	})
	w.srv = httptest.NewServer(mux)
	return w
}

func (w *stubWorld) close() { w.srv.Close() }

func (w *stubWorld) setUserinfo(prov, body string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.userinfo[prov] = body
}

func (w *stubWorld) setIDToken(prov, tok string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.idTokens[prov] = tok
}

func (w *stubWorld) lastForm(prov string) url.Values {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.forms[prov]
}

// mintIDToken signs an RS256 id_token the stub JWKS verifies.
func (w *stubWorld) mintIDToken(t *testing.T, iss, aud, nonce, sub, name, email string, now time.Time) string {
	t.Helper()
	claims := jwt.MapClaims{"iss": iss, "aud": aud, "sub": sub, "nonce": nonce,
		"iat": now.Unix(), "exp": now.Add(5 * time.Minute).Unix()}
	if name != "" {
		claims["name"] = name
	}
	if email != "" {
		claims["email"] = email
		claims["email_verified"] = "true"
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = w.kid
	signed, err := tok.SignedString(w.rsaKey)
	if err != nil {
		t.Fatalf("sign id_token: %v", err)
	}
	return signed
}

// oauthFixture wires a service whose providers all point at the stub world.
type oauthFixture struct {
	s     *OAuthService
	users *fakeOAuthUsers
	links *fakeOAuthLinks
	world *stubWorld
	now   time.Time
}

func (f *oauthFixture) advance(d time.Duration) { f.now = f.now.Add(d) }

func oauthTestService(t *testing.T) *oauthFixture {
	t.Helper()
	w := newStubWorld(t)
	t.Cleanup(w.close)
	cfg := &config.Config{}
	cfg.JWT.AccessSecret = "test-access-secret-32-chars-minimum-ok"
	cfg.JWT.RefreshSecret = "test-refresh-secret-32-chars-minimum-ok"
	cfg.JWT.AccessTTLMinutes = 15
	cfg.JWT.RefreshTTLDays = 14
	cfg.OAuth.RedirectBaseURL = "https://api.example.test/api/v1/auth/oauth"
	cfg.OAuth.StateSecret = "test-state-secret"
	cfg.OAuth.GoogleClientID, cfg.OAuth.GoogleClientSecret = "gid", "gsecret"
	cfg.OAuth.GoogleTokenURL = w.srv.URL + "/google/token"
	cfg.OAuth.GoogleUserinfoURL = w.srv.URL + "/google/userinfo"
	cfg.OAuth.KakaoClientID = "kid"
	cfg.OAuth.KakaoTokenURL = w.srv.URL + "/kakao/token"
	cfg.OAuth.KakaoUserinfoURL = w.srv.URL + "/kakao/userinfo"
	cfg.OAuth.AppleServicesID, cfg.OAuth.AppleTeamID, cfg.OAuth.AppleKeyID = "com.example.signin", "TEAM123", "KEY123"
	cfg.OAuth.ApplePrivateKey = testAppleP8(t)
	cfg.OAuth.AppleTokenURL = w.srv.URL + "/apple/token"
	cfg.OAuth.AppleJWKSURL = w.srv.URL + "/apple/jwks"
	cfg.OAuth.NaverClientID, cfg.OAuth.NaverClientSecret = "nid", "nsecret"
	cfg.OAuth.NaverTokenURL = w.srv.URL + "/naver/token"
	cfg.OAuth.NaverUserinfoURL = w.srv.URL + "/naver/userinfo"
	cfg.OAuth.LineChannelID, cfg.OAuth.LineChannelSecret = "lid", "lsecret"
	cfg.OAuth.LineTokenURL = w.srv.URL + "/line/token"
	cfg.OAuth.LineJWKSURL = w.srv.URL + "/line/jwks"
	auth := NewAuthService(nil, cfg)
	users := &fakeOAuthUsers{}
	links := &fakeOAuthLinks{}
	s := NewOAuthService(&repository.Repos{}, auth, cfg, w.srv.Client())
	s.users, s.links = users, links
	fx := &oauthFixture{s: s, users: users, links: links, world: w, now: time.Now().Truncate(time.Second)}
	s.nowFn = func() time.Time { return fx.now }
	return fx
}

// testAppleP8 generates a P-256 key and returns it as p8 PEM (APPLE_PRIVATE_KEY shape).
func testAppleP8(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ec key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("pkcs8: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

func startFor(t *testing.T, fx *oauthFixture, provider, linkToken string) *OAuthStart {
	t.Helper()
	st, err := fx.s.Start(provider, linkToken)
	if err != nil {
		t.Fatalf("start %s: %v", provider, err)
	}
	return st
}

func cbInput(st *OAuthStart, provider, code string) OAuthCallbackInput {
	return OAuthCallbackInput{Provider: provider, Code: code, State: st.State, CookieNonce: st.Nonce, PKCEVerifier: st.PKCEVerifier}
}

func stateNonce(t *testing.T, state string) string {
	t.Helper()
	parts := strings.Split(state, ".")
	if len(parts) < 5 {
		t.Fatalf("bad state %q", state)
	}
	return parts[1]
}

func providerCodes(infos []OAuthProviderInfo) []string {
	out := make([]string, 0, len(infos))
	for _, p := range infos {
		out = append(out, p.Provider)
	}
	return out
}

func TestOAuthProvidersLocaleOrder(t *testing.T) {
	fx := oauthTestService(t)
	if got := providerCodes(fx.s.Providers("ko")); strings.Join(got, ",") != "KAKAO,NAVER,GOOGLE,APPLE,LINE" {
		t.Fatalf("ko order = %v", got)
	}
	if got := providerCodes(fx.s.Providers("ja")); strings.Join(got, ",") != "LINE,GOOGLE,APPLE,KAKAO,NAVER" {
		t.Fatalf("ja order = %v", got)
	}
	if got := providerCodes(fx.s.Providers("en")); strings.Join(got, ",") != "GOOGLE,APPLE,KAKAO,NAVER,LINE" {
		t.Fatalf("en order = %v", got)
	}
	if got := providerCodes(fx.s.Providers("xx")); strings.Join(got, ",") != "GOOGLE,APPLE,KAKAO,NAVER,LINE" {
		t.Fatalf("unknown locale order = %v", got)
	}
	for _, p := range fx.s.Providers("") {
		if !p.Enabled {
			t.Fatalf("provider %s should be enabled in fixture", p.Provider)
		}
	}
}

// TestOAuthEnvToggle proves enablement follows credentials only: drop one
// variable and the provider (and only it) goes dark.
func TestOAuthEnvToggle(t *testing.T) {
	fx := oauthTestService(t)
	if got := len(fx.s.EnabledProviders()); got != 5 {
		t.Fatalf("enabled = %d, want 5", got)
	}
	fx.s.cfg.AppleTeamID = ""
	if fx.s.IsEnabled("APPLE") {
		t.Fatalf("apple still enabled without team id")
	}
	if got := len(fx.s.EnabledProviders()); got != 4 {
		t.Fatalf("enabled = %d, want 4", got)
	}
	fx.s.cfg.AppleTeamID = "TEAM123"
	if !fx.s.IsEnabled("APPLE") {
		t.Fatalf("apple not re-enabled")
	}
	// Kakao secret stays optional: ID alone enables.
	fx.s.cfg.KakaoClientSecret = ""
	if !fx.s.IsEnabled("KAKAO") {
		t.Fatalf("kakao requires only the client id")
	}
	fx.s.cfg.KakaoClientID = ""
	if fx.s.IsEnabled("KAKAO") {
		t.Fatalf("kakao enabled without id")
	}
}

func TestOAuthStartURLShapes(t *testing.T) {
	fx := oauthTestService(t)
	st, err := fx.s.Start("GOOGLE", "")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if !strings.HasPrefix(st.URL, "https://accounts.google.com/") || !strings.Contains(st.URL, "state=") {
		t.Fatalf("google url = %q", st.URL)
	}
	if !strings.Contains(st.URL, "code_challenge=") || st.PKCEVerifier == "" {
		t.Fatalf("google must use PKCE")
	}
	if st.SameSiteNone {
		t.Fatalf("google must not need SameSite=None")
	}
	nav := startFor(t, fx, "NAVER", "")
	if strings.Contains(nav.URL, "code_challenge=") || nav.PKCEVerifier != "" {
		t.Fatalf("naver must not use PKCE")
	}
	apple := startFor(t, fx, "APPLE", "")
	if !strings.Contains(apple.URL, "response_mode=form_post") || !strings.Contains(apple.URL, "nonce=") {
		t.Fatalf("apple url = %q", apple.URL)
	}
	if !apple.SameSiteNone {
		t.Fatalf("apple needs SameSite=None")
	}
	line := startFor(t, fx, "LINE", "")
	if !strings.Contains(line.URL, "nonce=") || !strings.Contains(line.URL, "scope=openid") {
		t.Fatalf("line url = %q", line.URL)
	}
	if _, err := fx.s.Start("WECHAT", ""); err == nil {
		t.Fatalf("wechat accepted (deferred provider)")
	}
	fx.s.cfg.NaverClientSecret = ""
	if _, err := fx.s.Start("NAVER", ""); err == nil {
		t.Fatalf("disabled provider accepted")
	}
}

func TestOAuthGoogleNewUser(t *testing.T) {
	fx := oauthTestService(t)
	fx.world.setUserinfo("GOOGLE", `{"sub":"g-123","email":"New@Example.com","email_verified":true,"name":"New User"}`)
	st := startFor(t, fx, "GOOGLE", "")
	res, err := fx.s.Callback(context.Background(), cbInput(st, "GOOGLE", "code-1"))
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	if res.ExchangeCode == "" || res.EmailToken != "" || res.Linked {
		t.Fatalf("result = %+v", res)
	}
	tokens, u, err := fx.s.ExchangeCode(context.Background(), res.ExchangeCode)
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if u.Email != "new@example.com" || u.Role != domain.RoleUser || u.DisplayName != "New User" {
		t.Fatalf("user = %+v", u)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("tokens missing")
	}
	if _, err := fx.links.GetByProviderUID(context.Background(), "GOOGLE", "g-123"); err != nil {
		t.Fatalf("link not stored: %v", err)
	}
	if len(fx.users.created) != 1 {
		t.Fatalf("created = %d", len(fx.users.created))
	}
	// Second login reuses the link without creating another user.
	st2 := startFor(t, fx, "GOOGLE", "")
	res2, err := fx.s.Callback(context.Background(), cbInput(st2, "GOOGLE", "code-2"))
	if err != nil {
		t.Fatalf("re-login: %v", err)
	}
	tokens2, u2, err := fx.s.ExchangeCode(context.Background(), res2.ExchangeCode)
	if err != nil {
		t.Fatalf("re-exchange: %v", err)
	}
	if u2.ID != u.ID || tokens2.AccessToken == "" || len(fx.users.created) != 1 {
		t.Fatalf("re-login mismatch: %+v created=%d", u2, len(fx.users.created))
	}
}

func TestOAuthCallbackTransientStoreFailsClosed(t *testing.T) {
	fx := oauthTestService(t)
	fx.world.setUserinfo("GOOGLE", `{"sub":"g-9","email":"t@example.com","email_verified":true,"name":"T"}`)
	fx.links.getErr = errors.New("db blip")
	st := startFor(t, fx, "GOOGLE", "")
	if _, err := fx.s.Callback(context.Background(), cbInput(st, "GOOGLE", "code-1")); err == nil || errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("link-lookup blip must surface, got %v", err)
	}
	if len(fx.users.created) != 0 {
		t.Fatalf("blip created %d users", len(fx.users.created))
	}
	fx.links.getErr = nil
	fx.users.emailErr = errors.New("db blip")
	st2 := startFor(t, fx, "GOOGLE", "")
	if _, err := fx.s.Callback(context.Background(), cbInput(st2, "GOOGLE", "code-2")); err == nil || errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("email-lookup blip must surface, got %v", err)
	}
	if len(fx.users.created) != 0 {
		t.Fatalf("blip created %d users", len(fx.users.created))
	}
}

func TestOAuthKakaoVerifiedLinksExistingEmail(t *testing.T) {
	fx := oauthTestService(t)
	fx.world.setUserinfo("KAKAO", `{"id":4242,"kakao_account":{"email":"Kakao@Example.com","is_email_verified":true,"is_email_valid":true},"properties":{"nickname":"카카오닉"}}`)
	fx.users.byID = map[string]*domain.User{"u-9": {ID: "u-9", Email: "kakao@example.com", Role: "USER", IsActive: true}}
	fx.users.byEmail = map[string]*domain.User{"kakao@example.com": fx.users.byID["u-9"]}
	st := startFor(t, fx, "KAKAO", "")
	res, err := fx.s.Callback(context.Background(), cbInput(st, "KAKAO", "code-1"))
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	if res.ExchangeCode == "" {
		t.Fatalf("result = %+v", res)
	}
	_, u, err := fx.s.ExchangeCode(context.Background(), res.ExchangeCode)
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if u.ID != "u-9" || len(fx.users.created) != 0 {
		t.Fatalf("existing email not linked: %+v created=%d", u, len(fx.users.created))
	}
	// Kakao secret is optional: the token form omits it when unset.
	if v := fx.world.lastForm("KAKAO").Get("client_secret"); v != "" {
		t.Fatalf("client_secret sent while unset: %q", v)
	}
}

func TestOAuthAppleFullFlow(t *testing.T) {
	fx := oauthTestService(t)
	st := startFor(t, fx, "APPLE", "")
	nonce := fx.s.oidcNonce(stateNonce(t, st.State))
	fx.world.setIDToken("APPLE", fx.world.mintIDToken(t, appleIssuer, "com.example.signin", nonce,
		"apple-sub-1", "", "private@privaterelay.appleid.com", fx.now))
	in := cbInput(st, "APPLE", "code-1")
	in.RawUser = `{"name":{"firstName":"Ada","lastName":"Lovelace"},"email":"private@privaterelay.appleid.com"}`
	res, err := fx.s.Callback(context.Background(), in)
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	if res.ExchangeCode == "" {
		t.Fatalf("result = %+v", res)
	}
	_, u, err := fx.s.ExchangeCode(context.Background(), res.ExchangeCode)
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	// First-auth name comes from the form_post user field, not the id_token.
	if u.DisplayName != "Ada Lovelace" || u.Email != "private@privaterelay.appleid.com" {
		t.Fatalf("user = %+v", u)
	}
	// The token request carried an ES256 client_secret minted from the p8 key.
	secret := fx.world.lastForm("APPLE").Get("client_secret")
	block, _ := pem.Decode([]byte(fx.s.cfg.ApplePrivateKey))
	der := block.Bytes
	ecPriv, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		t.Fatalf("p8 parse: %v", err)
	}
	claims := jwt.MapClaims{}
	vt, err := jwt.ParseWithClaims(secret, claims, func(tok *jwt.Token) (any, error) {
		if _, ok := tok.Method.(*jwt.SigningMethodECDSA); !ok {
			t.Fatalf("client_secret alg = %v", tok.Method.Alg())
		}
		return &ecPriv.(*ecdsa.PrivateKey).PublicKey, nil
	})
	if err != nil || !vt.Valid {
		t.Fatalf("client_secret verify: %v", err)
	}
	if kid, _ := vt.Header["kid"].(string); kid != "KEY123" {
		t.Fatalf("client_secret kid = %q", kid)
	}
	if claims["iss"] != "TEAM123" || claims["sub"] != "com.example.signin" || claims["aud"] != appleIssuer {
		t.Fatalf("client_secret claims = %v", claims)
	}
	exp, _ := claims["exp"].(float64)
	iat, _ := claims["iat"].(float64)
	if int64(exp)-int64(iat) != 300 {
		t.Fatalf("client_secret ttl = %v", claims)
	}
}

func TestOAuthAppleFormPostIDToken(t *testing.T) {
	fx := oauthTestService(t)
	st := startFor(t, fx, "APPLE", "")
	nonce := fx.s.oidcNonce(stateNonce(t, st.State))
	// Token response without id_token: the form_post id_token still verifies.
	fx.world.setIDToken("APPLE", "")
	in := cbInput(st, "APPLE", "code-1")
	in.IDToken = fx.world.mintIDToken(t, appleIssuer, "com.example.signin", nonce,
		"apple-sub-9", "", "form@privaterelay.appleid.com", fx.now)
	res, err := fx.s.Callback(context.Background(), in)
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	if res.ExchangeCode == "" {
		t.Fatalf("result = %+v", res)
	}
}

func TestOAuthNaverFlow(t *testing.T) {
	fx := oauthTestService(t)
	fx.world.setUserinfo("NAVER", `{"resultcode":"00","response":{"id":"naver-1","email":"naver@example.com","name":"네이버","nickname":"nv"}}`)
	st := startFor(t, fx, "NAVER", "")
	res, err := fx.s.Callback(context.Background(), cbInput(st, "NAVER", "code-1"))
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	_, u, err := fx.s.ExchangeCode(context.Background(), res.ExchangeCode)
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if u.Email != "naver@example.com" || u.DisplayName != "네이버" {
		t.Fatalf("user = %+v", u)
	}
	// Naver echoes state at the token endpoint.
	if v := fx.world.lastForm("NAVER").Get("state"); v == "" {
		t.Fatalf("naver token form lost state")
	}
}

func TestOAuthNaverMissingEmailIssuesToken(t *testing.T) {
	fx := oauthTestService(t)
	fx.world.setUserinfo("NAVER", `{"resultcode":"00","response":{"id":"naver-2","name":"거부","nickname":"nv2"}}`)
	st := startFor(t, fx, "NAVER", "")
	res, err := fx.s.Callback(context.Background(), cbInput(st, "NAVER", "code-1"))
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	if res.EmailToken == "" || res.ExchangeCode != "" {
		t.Fatalf("result = %+v", res)
	}
	pend, err := fx.s.CompleteEmail(context.Background(), res.EmailToken, "Typed@Example.com")
	if err != nil {
		t.Fatalf("complete-email: %v", err)
	}
	if pend.Provider != "NAVER" || pend.Email != "typed@example.com" || !pend.EmailVerificationRequired {
		t.Fatalf("pending = %+v", pend)
	}
	// The token survives for WI-A2 (peek, not consume).
	if _, err := fx.s.CompleteEmail(context.Background(), res.EmailToken, "typed@example.com"); err != nil {
		t.Fatalf("token consumed early: %v", err)
	}
}

func TestOAuthLineFlow(t *testing.T) {
	fx := oauthTestService(t)
	st := startFor(t, fx, "LINE", "")
	nonce := fx.s.oidcNonce(stateNonce(t, st.State))
	fx.world.setIDToken("LINE", fx.world.mintIDToken(t, lineIssuer, "lid", nonce,
		"line-sub-1", "ライン", "line@example.com", fx.now))
	res, err := fx.s.Callback(context.Background(), cbInput(st, "LINE", "code-1"))
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	_, u, err := fx.s.ExchangeCode(context.Background(), res.ExchangeCode)
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if u.Email != "line@example.com" || u.DisplayName != "ライン" {
		t.Fatalf("user = %+v", u)
	}
}

func TestOAuthLineNonceMismatch(t *testing.T) {
	fx := oauthTestService(t)
	st := startFor(t, fx, "LINE", "")
	fx.world.setIDToken("LINE", fx.world.mintIDToken(t, lineIssuer, "lid", "wrong-nonce",
		"line-sub-2", "", "x@example.com", fx.now))
	if _, err := fx.s.Callback(context.Background(), cbInput(st, "LINE", "code-1")); err == nil {
		t.Fatalf("forged nonce accepted")
	}
}

func seedExistingUser(fx *oauthFixture, id, email string) {
	if fx.users.byID == nil {
		fx.users.byID = map[string]*domain.User{}
		fx.users.byEmail = map[string]*domain.User{}
	}
	u := &domain.User{ID: id, Email: email, Role: "USER", IsActive: true}
	fx.users.byID[id] = u
	fx.users.byEmail[strings.ToLower(email)] = u
}

// TestOAuthC11RefusesUnverifiedAutoLink: only Google/Apple/Kakao-verified
// addresses auto-link an existing account; anything else must log in first.
func TestOAuthC11RefusesUnverifiedAutoLink(t *testing.T) {
	cases := []struct {
		name     string
		provider string
		userinfo string
		wantLink bool
	}{
		{"google verified links", "GOOGLE",
			`{"sub":"g-9","email":"same@example.com","email_verified":true,"name":"G"}`, true},
		{"google unverified refused", "GOOGLE",
			`{"sub":"g-9","email":"same@example.com","email_verified":false,"name":"G"}`, false},
		{"kakao verified links", "KAKAO",
			`{"id":99,"kakao_account":{"email":"same@example.com","is_email_verified":true,"is_email_valid":true},"properties":{"nickname":"k"}}`, true},
		{"kakao unverified refused", "KAKAO",
			`{"id":99,"kakao_account":{"email":"same@example.com","is_email_verified":true,"is_email_valid":false},"properties":{"nickname":"k"}}`, false},
		{"kakao missing flags refused", "KAKAO",
			`{"id":99,"kakao_account":{"email":"same@example.com"},"properties":{"nickname":"k"}}`, false},
		{"naver always refused", "NAVER",
			`{"resultcode":"00","response":{"id":"n-9","email":"same@example.com","name":"N"}}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := oauthTestService(t)
			seedExistingUser(fx, "u-exist", "same@example.com")
			fx.world.setUserinfo(tc.provider, tc.userinfo)
			st := startFor(t, fx, tc.provider, "")
			_, err := fx.s.Callback(context.Background(), cbInput(st, tc.provider, "code-1"))
			if tc.wantLink {
				if err != nil {
					t.Fatalf("verified link rejected: %v", err)
				}
				return
			}
			if !errors.Is(err, domain.ErrOAuthLinkLoginRequired) {
				t.Fatalf("err = %v, want ErrOAuthLinkLoginRequired", err)
			}
			if len(fx.users.created) != 0 {
				t.Fatalf("refused login created %d users", len(fx.users.created))
			}
		})
	}
}

func TestOAuthC12StateBinding(t *testing.T) {
	newFlow := func(t *testing.T) (*oauthFixture, *OAuthStart) {
		fx := oauthTestService(t)
		fx.world.setUserinfo("GOOGLE", `{"sub":"g-c12","email":"c12@example.com","email_verified":true,"name":"C"}`)
		return fx, startFor(t, fx, "GOOGLE", "")
	}
	t.Run("tampered signature", func(t *testing.T) {
		fx, st := newFlow(t)
		in := cbInput(st, "GOOGLE", "code-1")
		in.State = st.State[:len(st.State)-2] + "ff"
		if _, err := fx.s.Callback(context.Background(), in); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("cookie mismatch", func(t *testing.T) {
		fx, st := newFlow(t)
		in := cbInput(st, "GOOGLE", "code-1")
		in.CookieNonce = "deadbeefdeadbeefdeadbeefdeadbeef"
		if _, err := fx.s.Callback(context.Background(), in); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("missing cookie", func(t *testing.T) {
		fx, st := newFlow(t)
		in := cbInput(st, "GOOGLE", "code-1")
		in.CookieNonce = ""
		if _, err := fx.s.Callback(context.Background(), in); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("missing pkce", func(t *testing.T) {
		fx, st := newFlow(t)
		in := cbInput(st, "GOOGLE", "code-1")
		in.PKCEVerifier = ""
		if _, err := fx.s.Callback(context.Background(), in); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("provider swap", func(t *testing.T) {
		fx, st := newFlow(t)
		fx.world.setUserinfo("KAKAO", `{"id":1,"kakao_account":{"email":"c12@example.com","is_email_verified":true,"is_email_valid":true},"properties":{"nickname":"k"}}`)
		in := cbInput(st, "KAKAO", "code-1")
		if _, err := fx.s.Callback(context.Background(), in); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("expired", func(t *testing.T) {
		fx, st := newFlow(t)
		fx.advance(11 * time.Minute)
		if _, err := fx.s.Callback(context.Background(), cbInput(st, "GOOGLE", "code-1")); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("single use", func(t *testing.T) {
		fx, st := newFlow(t)
		if _, err := fx.s.Callback(context.Background(), cbInput(st, "GOOGLE", "code-1")); err != nil {
			t.Fatalf("first: %v", err)
		}
		if _, err := fx.s.Callback(context.Background(), cbInput(st, "GOOGLE", "code-2")); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("replay err = %v", err)
		}
	})
	t.Run("empty code", func(t *testing.T) {
		fx, st := newFlow(t)
		if _, err := fx.s.Callback(context.Background(), cbInput(st, "GOOGLE", "")); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestOAuthC16ExchangeCodes(t *testing.T) {
	setup := func(t *testing.T) (*oauthFixture, string) {
		fx := oauthTestService(t)
		fx.world.setUserinfo("GOOGLE", `{"sub":"g-c16","email":"c16@example.com","email_verified":true,"name":"C"}`)
		st := startFor(t, fx, "GOOGLE", "")
		res, err := fx.s.Callback(context.Background(), cbInput(st, "GOOGLE", "code-1"))
		if err != nil {
			t.Fatalf("callback: %v", err)
		}
		return fx, res.ExchangeCode
	}
	t.Run("success", func(t *testing.T) {
		fx, code := setup(t)
		tokens, u, err := fx.s.ExchangeCode(context.Background(), code)
		if err != nil {
			t.Fatalf("exchange: %v", err)
		}
		if tokens.AccessToken == "" || u.Email != "c16@example.com" {
			t.Fatalf("tokens/user = %+v %+v", tokens, u)
		}
	})
	t.Run("single use", func(t *testing.T) {
		fx, code := setup(t)
		if _, _, err := fx.s.ExchangeCode(context.Background(), code); err != nil {
			t.Fatalf("first: %v", err)
		}
		if _, _, err := fx.s.ExchangeCode(context.Background(), code); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("replay err = %v", err)
		}
	})
	t.Run("expired", func(t *testing.T) {
		fx, code := setup(t)
		fx.advance(61 * time.Second)
		if _, _, err := fx.s.ExchangeCode(context.Background(), code); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("bogus", func(t *testing.T) {
		fx := oauthTestService(t)
		if _, _, err := fx.s.ExchangeCode(context.Background(), "nope"); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("err = %v", err)
		}
		if _, _, err := fx.s.ExchangeCode(context.Background(), ""); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("empty err = %v", err)
		}
	})
}

func TestOAuthCompleteEmailGuards(t *testing.T) {
	fx := oauthTestService(t)
	if _, err := fx.s.CompleteEmail(context.Background(), "bogus", "a@example.com"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("bogus token err = %v", err)
	}
	fx.world.setUserinfo("LINE", "")
	// LINE without email in id_token still needs a valid token for the guard below,
	// so mint through the email path with Naver instead.
	fx.world.setUserinfo("NAVER", `{"resultcode":"00","response":{"id":"nv-g","name":"G"}}`)
	st := startFor(t, fx, "NAVER", "")
	res, err := fx.s.Callback(context.Background(), cbInput(st, "NAVER", "code-1"))
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	if _, err := fx.s.CompleteEmail(context.Background(), res.EmailToken, "not-an-email"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("bad email err = %v", err)
	}
	fx.advance(11 * time.Minute)
	if _, err := fx.s.CompleteEmail(context.Background(), res.EmailToken, "a@example.com"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expired token err = %v", err)
	}
}

func TestOAuthLinkFlow(t *testing.T) {
	linkUserinfo := `{"sub":"g-link","email":"link@example.com","email_verified":true,"name":"L"}`
	t.Run("link then idempotent relink", func(t *testing.T) {
		fx := oauthTestService(t)
		seedExistingUser(fx, "u-link", "owner@example.com")
		fx.world.setUserinfo("GOOGLE", linkUserinfo)
		lt, err := fx.s.MintLinkToken("u-link")
		if err != nil {
			t.Fatalf("mint: %v", err)
		}
		st := startFor(t, fx, "GOOGLE", lt)
		if !strings.Contains(st.State, ".link.u-link.") {
			t.Fatalf("state = %q", st.State)
		}
		res, err := fx.s.Callback(context.Background(), cbInput(st, "GOOGLE", "code-1"))
		if err != nil {
			t.Fatalf("callback: %v", err)
		}
		if !res.Linked || res.ExchangeCode != "" {
			t.Fatalf("result = %+v", res)
		}
		conns, err := fx.s.ListConnections(context.Background(), "u-link")
		if err != nil || len(conns) != 1 || conns[0].Provider != "GOOGLE" {
			t.Fatalf("conns = %+v err=%v", conns, err)
		}
		// Relinking the same identity to the same user is a no-op success.
		lt2, _ := fx.s.MintLinkToken("u-link")
		st2 := startFor(t, fx, "GOOGLE", lt2)
		res2, err := fx.s.Callback(context.Background(), cbInput(st2, "GOOGLE", "code-2"))
		if err != nil || !res2.Linked {
			t.Fatalf("relink = %+v err=%v", res2, err)
		}
	})
	t.Run("link taken by another user", func(t *testing.T) {
		fx := oauthTestService(t)
		seedExistingUser(fx, "u-a", "a@example.com")
		seedExistingUser(fx, "u-b", "b@example.com")
		if _, err := fx.links.Upsert(context.Background(), "u-a", "GOOGLE", "g-link", nil); err != nil {
			t.Fatalf("upsert: %v", err)
		}
		fx.world.setUserinfo("GOOGLE", linkUserinfo)
		lt, _ := fx.s.MintLinkToken("u-b")
		st := startFor(t, fx, "GOOGLE", lt)
		if _, err := fx.s.Callback(context.Background(), cbInput(st, "GOOGLE", "code-1")); !errors.Is(err, domain.ErrOAuthLinkTaken) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("bad link token", func(t *testing.T) {
		fx := oauthTestService(t)
		if _, err := fx.s.Start("GOOGLE", "bogus"); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("err = %v", err)
		}
		lt, _ := fx.s.MintLinkToken("u-link")
		fx.advance(6 * time.Minute)
		if _, err := fx.s.Start("GOOGLE", lt); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("expired err = %v", err)
		}
	})
}

func TestOAuthUnlink(t *testing.T) {
	fx := oauthTestService(t)
	ctx := context.Background()
	if err := fx.s.Unlink(ctx, "u-1", "GOOGLE"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("absent err = %v", err)
	}
	if err := fx.s.Unlink(ctx, "u-1", "WECHAT"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("provider err = %v", err)
	}
	if _, err := fx.links.Upsert(ctx, "u-1", "GOOGLE", "g-1", nil); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := fx.s.Unlink(ctx, "u-1", "GOOGLE"); !errors.Is(err, domain.ErrOAuthLastMethod) {
		t.Fatalf("last-method err = %v", err)
	}
	if _, err := fx.links.Upsert(ctx, "u-1", "KAKAO", "k-1", nil); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := fx.s.Unlink(ctx, "u-1", "GOOGLE"); err != nil {
		t.Fatalf("unlink: %v", err)
	}
	conns, err := fx.s.ListConnections(ctx, "u-1")
	if err != nil || len(conns) != 1 || conns[0].Provider != "KAKAO" {
		t.Fatalf("conns = %+v err=%v", conns, err)
	}
	if err := fx.s.Unlink(ctx, "u-1", "KAKAO"); !errors.Is(err, domain.ErrOAuthLastMethod) {
		t.Fatalf("final err = %v", err)
	}
}

func TestOAuthStartupWarnings(t *testing.T) {
	fx := oauthTestService(t)
	// Fully configured fixture: only the "all good" path (no warnings).
	fx.s.cfg.StateSecret = "set"
	if got := fx.s.StartupWarnings(); len(got) != 0 {
		t.Fatalf("warnings = %v", got)
	}
	// Empty config: every provider warns with variable NAMES only.
	empty := NewOAuthService(&repository.Repos{}, nil, &config.Config{}, nil)
	empty.users, empty.links = fx.users, fx.links
	warnings := empty.StartupWarnings()
	if len(warnings) != 6 {
		t.Fatalf("warnings = %v", warnings)
	}
	joined := strings.Join(warnings, "\n")
	for _, name := range []string{"GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET", "KAKAO_CLIENT_ID",
		"APPLE_SERVICES_ID", "APPLE_TEAM_ID", "APPLE_KEY_ID", "APPLE_PRIVATE_KEY",
		"NAVER_CLIENT_ID", "NAVER_CLIENT_SECRET", "LINE_CHANNEL_ID", "LINE_CHANNEL_SECRET"} {
		if !strings.Contains(joined, name) {
			t.Fatalf("warnings miss %s: %v", name, warnings)
		}
	}
}

// emailTokenFor seeds the missing-email path and returns its signup token.
func emailTokenFor(t *testing.T, fx *oauthFixture) string {
	t.Helper()
	fx.world.setUserinfo("NAVER", `{"resultcode":"00","response":{"id":"naver-3","name":"신규","nickname":"nv3"}}`)
	st := startFor(t, fx, "NAVER", "")
	res, err := fx.s.Callback(context.Background(), cbInput(st, "NAVER", "code-1"))
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	if res.EmailToken == "" {
		t.Fatalf("result = %+v", res)
	}
	return res.EmailToken
}

func TestOAuthFinishVerifiedSignupCreatesAndLinks(t *testing.T) {
	fx := oauthTestService(t)
	tok := emailTokenFor(t, fx)
	u, err := fx.s.FinishVerifiedSignup(context.Background(), tok, "Typed@Example.com")
	if err != nil {
		t.Fatalf("finish: %v", err)
	}
	if u.Email != "typed@example.com" || len(fx.users.created) != 1 {
		t.Fatalf("user = %+v created=%d", u, len(fx.users.created))
	}
	link, err := fx.links.GetByProviderUID(context.Background(), "NAVER", "naver-3")
	if err != nil || link.UserID != u.ID {
		t.Fatalf("link = %+v err=%v", link, err)
	}
	// The signup token is consumed: a second finish fails.
	if _, err := fx.s.FinishVerifiedSignup(context.Background(), tok, "typed@example.com"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("replay err = %v", err)
	}
}

func TestOAuthFinishVerifiedSignupLinksExisting(t *testing.T) {
	fx := oauthTestService(t)
	seedExistingUser(fx, "u-keep", "owner@example.com")
	tok := emailTokenFor(t, fx)
	u, err := fx.s.FinishVerifiedSignup(context.Background(), tok, "owner@example.com")
	if err != nil {
		t.Fatalf("finish: %v", err)
	}
	if u.ID != "u-keep" || len(fx.users.created) != 0 {
		t.Fatalf("user = %+v created=%d", u, len(fx.users.created))
	}
	link, err := fx.links.GetByProviderUID(context.Background(), "NAVER", "naver-3")
	if err != nil || link.UserID != "u-keep" {
		t.Fatalf("link = %+v err=%v", link, err)
	}
}

func TestOAuthFinishVerifiedSignupGuards(t *testing.T) {
	fx := oauthTestService(t)
	ctx := context.Background()
	if _, err := fx.s.FinishVerifiedSignup(ctx, "bogus", "a@example.com"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("bogus token err = %v", err)
	}
	tok := emailTokenFor(t, fx)
	if _, err := fx.s.FinishVerifiedSignup(ctx, tok, "not-an-email"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("bad email err = %v", err)
	}
	fx.advance(11 * time.Minute)
	if _, err := fx.s.FinishVerifiedSignup(ctx, tok, "a@example.com"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expired token err = %v", err)
	}
}

func TestOAuthAppPlatformRoundTrip(t *testing.T) {
	fx := oauthTestService(t)
	fx.world.setUserinfo("GOOGLE", `{"sub":"g-app-1","email":"app@example.com","email_verified":true,"name":"App User"}`)
	st, err := fx.s.StartForPlatform("GOOGLE", "", "app")
	if err != nil {
		t.Fatalf("start app: %v", err)
	}
	if !strings.Contains(st.State, ".app.") {
		t.Fatalf("app marker missing: %q", st.State)
	}
	res, err := fx.s.Callback(context.Background(), cbInput(st, "GOOGLE", "code-1"))
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	if res.Platform != OAuthPlatformApp || res.ExchangeCode == "" {
		t.Fatalf("result = %+v", res)
	}
	if _, _, err := fx.s.ExchangeCode(context.Background(), res.ExchangeCode); err != nil {
		t.Fatalf("exchange: %v", err)
	}
}

func TestOAuthAppPlatformGuards(t *testing.T) {
	fx := oauthTestService(t)
	if _, err := fx.s.StartForPlatform("GOOGLE", "", "evil"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("bad platform err = %v", err)
	}
	// Forging the .app marker onto a web state breaks the HMAC.
	st := startFor(t, fx, "GOOGLE", "")
	parts := strings.Split(st.State, ".")
	forged := strings.Join(parts[:len(parts)-1], ".") + ".app." + parts[len(parts)-1]
	in := cbInput(st, "GOOGLE", "code-1")
	in.State = forged
	if _, err := fx.s.Callback(context.Background(), in); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("forged platform err = %v", err)
	}
}
