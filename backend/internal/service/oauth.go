package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"avibe-platform/backend/internal/config"
	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/repository"
)

const (
	oauthStateTTL      = 10 * time.Minute
	oauthExchangeTTL   = 60 * time.Second
	oauthEmailTokenTTL = 10 * time.Minute
	oauthLinkTokenTTL  = 5 * time.Minute
)

// OAuth cookie names for the C-12 browser-bound state.
const (
	OAuthStateCookie = "avibe_oauth_state"
	OAuthPKCECookie  = "avibe_oauth_pkce"
)

// oauthUserStore is the user surface social login needs.
type oauthUserStore interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Create(ctx context.Context, email, passwordHash, role, displayName string) (*domain.User, error)
	GrantRole(ctx context.Context, id, role, source string) error
}

// oauthLinkStore is the provider-link surface social login needs.
type oauthLinkStore interface {
	GetByProviderUID(ctx context.Context, provider, providerUID string) (*domain.OAuthAccount, error)
	Upsert(ctx context.Context, userID, provider, providerUID string, email *string) (*domain.OAuthAccount, error)
	ListByUserID(ctx context.Context, userID string) ([]*domain.OAuthAccount, error)
	DeleteByUserProvider(ctx context.Context, userID, provider string) error
}

// OAuthProviderInfo describes one social provider's availability.
type OAuthProviderInfo struct {
	Provider string `json:"provider"`
	Enabled  bool   `json:"enabled"`
}

// OAuthService runs the 5-provider authorization-code flow (WI-A1). Social
// accounts always land on the USER role; partners onboard through the email
// flow with business verification.
type OAuthService struct {
	users         oauthUserStore
	links         oauthLinkStore
	auth          *AuthService
	cfg           config.OAuthConfig
	client        *http.Client
	nonces        *oauthTTLStore
	exchangeCodes *oauthTTLStore
	emailTokens   *oauthTTLStore
	jwksMu        sync.Mutex
	jwks          map[string]*jwksEntry
	nowFn         func() time.Time
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
	return &OAuthService{
		users: repos.Users, links: repos.OAuth, auth: auth, cfg: oauthCfg, client: httpClient,
		nonces: newOAuthTTLStore(), exchangeCodes: newOAuthTTLStore(), emailTokens: newOAuthTTLStore(),
		jwks:  map[string]*jwksEntry{},
		nowFn: time.Now,
	}
}

func (s *OAuthService) clock() time.Time {
	if s.nowFn != nil {
		return s.nowFn()
	}
	return time.Now()
}

// oauthTTLStore is a tiny in-memory TTL map for nonces and one-time codes.
// Single-instance only; entries are short-lived (<=10min) and fail closed
// (a restart just forces the user to restart the login).
type oauthTTLStore struct {
	mu    sync.Mutex
	items map[string]oauthTTLItem
}

type oauthTTLItem struct {
	val string
	exp time.Time
}

func newOAuthTTLStore() *oauthTTLStore { return &oauthTTLStore{items: map[string]oauthTTLItem{}} }

func (st *oauthTTLStore) sweep(now time.Time) {
	for k, v := range st.items {
		if !now.Before(v.exp) {
			delete(st.items, k)
		}
	}
}

func (st *oauthTTLStore) set(now time.Time, key, val string, ttl time.Duration) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.sweep(now)
	st.items[key] = oauthTTLItem{val: val, exp: now.Add(ttl)}
}

// take returns and deletes a live entry (single-use codes).
func (st *oauthTTLStore) take(now time.Time, key string) (string, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.sweep(now)
	it, ok := st.items[key]
	if !ok {
		return "", false
	}
	delete(st.items, key)
	return it.val, true
}

// peek returns a live entry without consuming it (email tokens: WI-A2
// consumes them after verification).
func (st *oauthTTLStore) peek(now time.Time, key string) (string, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.sweep(now)
	it, ok := st.items[key]
	if !ok {
		return "", false
	}
	return it.val, true
}

// claim records key once; false when a live entry already exists (nonces).
func (st *oauthTTLStore) claim(now time.Time, key string, ttl time.Duration) bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.sweep(now)
	if _, ok := st.items[key]; ok {
		return false
	}
	st.items[key] = oauthTTLItem{exp: now.Add(ttl)}
	return true
}

// oauthLocaleOrder mirrors web/app/utils/social.ts ORDER (UI §3.11).
var oauthLocaleOrder = map[string][]string{
	"ko":    {domain.OAuthKakao, domain.OAuthNaver, domain.OAuthGoogle, domain.OAuthApple},
	"ja":    {domain.OAuthLine, domain.OAuthGoogle, domain.OAuthApple},
	"th":    {domain.OAuthLine, domain.OAuthGoogle, domain.OAuthApple},
	"zh-cn": {domain.OAuthGoogle, domain.OAuthApple},
	"en":    {domain.OAuthGoogle, domain.OAuthApple},
}

// Providers reports social provider availability in locale display order
// (unknown locales fall back to en; off-list providers keep canonical order).
func (s *OAuthService) Providers(locale string) []OAuthProviderInfo {
	order := oauthLocaleOrder[strings.ToLower(strings.TrimSpace(locale))]
	if order == nil {
		order = oauthLocaleOrder["en"]
	}
	rank := map[string]int{}
	for i, c := range order {
		rank[c] = i
	}
	out := make([]OAuthProviderInfo, 0, len(domain.OAuthProviders))
	for _, c := range domain.OAuthProviders {
		out = append(out, OAuthProviderInfo{Provider: c, Enabled: s.IsEnabled(c)})
	}
	sort.SliceStable(out, func(i, j int) bool {
		ri, oki := rank[out[i].Provider]
		rj, okj := rank[out[j].Provider]
		switch {
		case oki && okj:
			return ri < rj
		case oki:
			return true
		case okj:
			return false
		default:
			return false
		}
	})
	return out
}

// IsEnabled reports whether a provider is configured.
func (s *OAuthService) IsEnabled(provider string) bool {
	a, err := s.adapterFor(provider)
	if err != nil {
		return false
	}
	return a.Enabled(s.cfg)
}

// EnabledProviders lists the configured providers in canonical order.
func (s *OAuthService) EnabledProviders() []string {
	var out []string
	for _, c := range domain.OAuthProviders {
		if s.IsEnabled(c) {
			out = append(out, c)
		}
	}
	return out
}

// StartupWarnings names every disabled provider and its missing variables
// (names only, never values) for the boot log.
func (s *OAuthService) StartupWarnings() []string {
	var out []string
	if strings.TrimSpace(s.cfg.StateSecret) == "" {
		out = append(out, "oauth: OAUTH_STATE_SECRET unset (JWT_ACCESS_SECRET fallback in use)")
	}
	for _, c := range domain.OAuthProviders {
		a, err := s.adapterFor(c)
		if err != nil {
			continue
		}
		if a.Enabled(s.cfg) {
			continue
		}
		out = append(out, fmt.Sprintf("oauth provider %s disabled (missing %s)",
			c, strings.Join(a.MissingVars(s.cfg), ", ")))
	}
	return out
}

func (s *OAuthService) adapterFor(provider string) (oauthAdapter, error) {
	switch strings.ToUpper(strings.TrimSpace(provider)) {
	case domain.OAuthGoogle:
		return googleAdapter{}, nil
	case domain.OAuthKakao:
		return kakaoAdapter{}, nil
	case domain.OAuthApple:
		return appleAdapter{}, nil
	case domain.OAuthNaver:
		return naverAdapter{}, nil
	case domain.OAuthLine:
		return lineAdapter{}, nil
	default:
		return nil, fmt.Errorf("%w: provider", domain.ErrInvalidInput)
	}
}

func (s *OAuthService) redirectURI(provider string) string {
	return s.cfg.RedirectBaseURL + "/" + strings.ToLower(provider) + "/callback"
}

// OAuthStart is one authorization redirect plus the cookies the handler must
// set (C-12 browser-bound state).
type OAuthStart struct {
	URL          string
	State        string
	Nonce        string
	PKCEVerifier string
	SameSiteNone bool
}

// OAuthPlatformApp marks a mobile system-browser flow (WI-R12): the
// handler redirects the callback to the avibe:// app link instead of
// the web frontend. Only this exact value is honored.
const OAuthPlatformApp = "app"

// Start builds the provider authorization URL. linkToken (from MintLinkToken)
// turns the flow into an account-link instead of a login.
func (s *OAuthService) Start(provider, linkToken string) (*OAuthStart, error) {
	return s.StartForPlatform(provider, linkToken, "")
}

// StartForPlatform is Start with a signed platform marker. platform is
// either "" (web) or OAuthPlatformApp (mobile); anything else is rejected.
func (s *OAuthService) StartForPlatform(provider, linkToken, platform string) (*OAuthStart, error) {
	a, err := s.adapterFor(provider)
	if err != nil {
		return nil, err
	}
	if !a.Enabled(s.cfg) {
		return nil, fmt.Errorf("%w: provider disabled", domain.ErrInvalidInput)
	}
	provider = a.Provider()
	linkUser := ""
	if strings.TrimSpace(linkToken) != "" {
		uid, err := s.parseLinkToken(linkToken)
		if err != nil {
			return nil, err
		}
		linkUser = uid
	}
	nonce, err := randHex(16)
	if err != nil {
		return nil, err
	}
	platform = strings.ToLower(strings.TrimSpace(platform))
	if platform != "" && platform != OAuthPlatformApp {
		return nil, fmt.Errorf("%w: platform", domain.ErrInvalidInput)
	}
	exp := s.clock().Add(oauthStateTTL).Unix()
	payload := "v1." + nonce + "." + strconv.FormatInt(exp, 10) + "." + provider
	if linkUser != "" {
		payload += ".link." + linkUser
	}
	if platform == OAuthPlatformApp {
		payload += ".app"
	}
	state := payload + "." + hexHMAC(s.cfg.StateSecret, payload)
	var verifier, challenge string
	if a.UsesPKCE() {
		verifier, err = randBase64URL(32)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256([]byte(verifier))
		challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	}
	return &OAuthStart{
		URL:          a.AuthURL(s.cfg, s.redirectURI(provider), state, challenge, s.oidcNonce(nonce)),
		State:        state,
		Nonce:        nonce,
		PKCEVerifier: verifier,
		SameSiteNone: a.SameSiteNone(),
	}, nil
}

type oauthStateParts struct {
	nonce    string
	linkUser string
	isApp    bool
}

// verifyState checks the signed round-trip state against the browser cookie
// (C-12): signature + provider + 10-minute expiry + single use.
func (s *OAuthService) verifyState(state, provider, cookieNonce string) (*oauthStateParts, error) {
	parts := strings.Split(state, ".")
	if len(parts) < 5 || parts[0] != "v1" {
		return nil, domain.ErrInvalidCredentials
	}
	payload := strings.Join(parts[:len(parts)-1], ".")
	if !hmacOK(s.cfg.StateSecret, payload, parts[len(parts)-1]) {
		return nil, domain.ErrInvalidCredentials
	}
	exp, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || parts[3] != provider {
		return nil, domain.ErrInvalidCredentials
	}
	now := s.clock()
	if expTime := time.Unix(exp, 0); now.After(expTime) || expTime.After(now.Add(oauthStateTTL+time.Minute)) {
		return nil, domain.ErrInvalidCredentials
	}
	var linkUser string
	var isApp bool
	switch len(parts) {
	case 5:
		// login flow
	case 6:
		// login flow, mobile app platform (WI-R12)
		if parts[4] != OAuthPlatformApp {
			return nil, domain.ErrInvalidCredentials
		}
		isApp = true
	case 7:
		if parts[4] != "link" || parts[5] == "" {
			return nil, domain.ErrInvalidCredentials
		}
		linkUser = parts[5]
	case 8:
		// link flow, mobile app platform (WI-R12)
		if parts[4] != "link" || parts[5] == "" || parts[6] != OAuthPlatformApp {
			return nil, domain.ErrInvalidCredentials
		}
		linkUser = parts[5]
		isApp = true
	default:
		return nil, domain.ErrInvalidCredentials
	}
	if cookieNonce == "" || subtle.ConstantTimeCompare([]byte(cookieNonce), []byte(parts[1])) != 1 {
		return nil, domain.ErrInvalidCredentials
	}
	if !s.nonces.claim(now, "used:"+parts[1], oauthStateTTL) {
		return nil, domain.ErrInvalidCredentials
	}
	return &oauthStateParts{nonce: parts[1], linkUser: linkUser, isApp: isApp}, nil
}

// OAuthCallbackInput carries the provider redirect plus the C-12 cookies.
// IDToken/RawUser come from Apple's form_post body.
type OAuthCallbackInput struct {
	Provider     string
	Code         string
	State        string
	CookieNonce  string
	PKCEVerifier string
	IDToken      string
	RawUser      string
}

// OAuthCallbackResult is exactly one of: an exchange code (login), an email
// token (provider gave no email), or a completed link. Platform echoes the
// signed state marker ("" web, "app" mobile) for the redirect target.
type OAuthCallbackResult struct {
	Provider     string
	ExchangeCode string
	EmailToken   string
	Linked       bool
	Platform     string
}

// Callback verifies state, exchanges the code, links or creates the user,
// and answers with a one-time code (C-16: tokens never touch the URL).
func (s *OAuthService) Callback(ctx context.Context, in OAuthCallbackInput) (*OAuthCallbackResult, error) {
	a, err := s.adapterFor(in.Provider)
	if err != nil {
		return nil, err
	}
	provider := a.Provider()
	if !a.Enabled(s.cfg) {
		return nil, fmt.Errorf("%w: provider disabled", domain.ErrInvalidInput)
	}
	st, err := s.verifyState(in.State, provider, in.CookieNonce)
	if err != nil {
		return nil, err
	}
	platform := ""
	if st.isApp {
		platform = OAuthPlatformApp
	}
	if strings.TrimSpace(in.Code) == "" {
		return nil, domain.ErrInvalidCredentials
	}
	if a.UsesPKCE() && strings.TrimSpace(in.PKCEVerifier) == "" {
		return nil, domain.ErrInvalidCredentials
	}
	toks, err := a.Exchange(ctx, s, s.redirectURI(provider), in.Code, in.PKCEVerifier, in.State)
	if err != nil {
		return nil, err
	}
	if toks.IDToken == "" && strings.TrimSpace(in.IDToken) != "" {
		toks.IDToken = strings.TrimSpace(in.IDToken)
	}
	ident, err := a.Identity(ctx, s, toks, s.oidcNonce(st.nonce), in.RawUser)
	if err != nil {
		return nil, err
	}
	link, err := s.links.GetByProviderUID(ctx, provider, ident.UID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	if err == nil {
		u, err := s.users.GetByID(ctx, link.UserID)
		if err != nil {
			return nil, domain.ErrInvalidCredentials
		}
		if !u.IsActive {
			return nil, domain.ErrUserInactive
		}
		if st.linkUser != "" {
			if link.UserID != st.linkUser {
				return nil, domain.ErrOAuthLinkTaken
			}
			return &OAuthCallbackResult{Provider: provider, Linked: true, Platform: platform}, nil
		}
		code, err := s.issueExchangeCode(u.ID)
		if err != nil {
			return nil, err
		}
		return &OAuthCallbackResult{Provider: provider, ExchangeCode: code, Platform: platform}, nil
	}
	if st.linkUser != "" {
		u, err := s.users.GetByID(ctx, st.linkUser)
		if err != nil {
			return nil, domain.ErrInvalidCredentials
		}
		if !u.IsActive {
			return nil, domain.ErrUserInactive
		}
		if _, err := s.links.Upsert(ctx, u.ID, provider, ident.UID, strPtrOrNil(ident.Email)); err != nil {
			return nil, err
		}
		return &OAuthCallbackResult{Provider: provider, Linked: true, Platform: platform}, nil
	}
	if strings.TrimSpace(ident.Email) == "" {
		tok, err := s.issueEmailToken(provider, ident)
		if err != nil {
			return nil, err
		}
		return &OAuthCallbackResult{Provider: provider, EmailToken: tok, Platform: platform}, nil
	}
	email := strings.ToLower(strings.TrimSpace(ident.Email))
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	if err != nil {
		randomPW := make([]byte, 32)
		if _, err := rand.Read(randomPW); err != nil {
			return nil, err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(randomPW)), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		u, err = s.users.Create(ctx, email, string(hash), domain.RoleUser, ident.DisplayName)
		if err != nil {
			return nil, err
		}
		if err := s.users.GrantRole(ctx, u.ID, domain.RoleUser, "register"); err != nil {
			return nil, err
		}
	} else if !u.IsActive {
		return nil, domain.ErrUserInactive
	} else if !ident.EmailVerified {
		// C-11: unverified provider email never auto-links an existing account.
		return nil, domain.ErrOAuthLinkLoginRequired
	}
	if _, err := s.links.Upsert(ctx, u.ID, provider, ident.UID, &email); err != nil {
		return nil, err
	}
	code, err := s.issueExchangeCode(u.ID)
	if err != nil {
		return nil, err
	}
	return &OAuthCallbackResult{Provider: provider, ExchangeCode: code, Platform: platform}, nil
}

func (s *OAuthService) issueExchangeCode(userID string) (string, error) {
	code, err := randHex(32)
	if err != nil {
		return "", err
	}
	s.exchangeCodes.set(s.clock(), "ex:"+code, userID, oauthExchangeTTL)
	return code, nil
}

// ExchangeCode swaps a single-use 60-second code for a token pair (C-16).
func (s *OAuthService) ExchangeCode(ctx context.Context, code string) (*Tokens, *domain.User, error) {
	if strings.TrimSpace(code) == "" {
		return nil, nil, domain.ErrInvalidCredentials
	}
	userID, ok := s.exchangeCodes.take(s.clock(), "ex:"+strings.TrimSpace(code))
	if !ok {
		return nil, nil, domain.ErrInvalidCredentials
	}
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, domain.ErrInvalidCredentials
	}
	if !u.IsActive {
		return nil, nil, domain.ErrUserInactive
	}
	tokens, err := s.auth.Issue(ctx, u.ID, u.Role)
	if err != nil {
		return nil, nil, err
	}
	return tokens, u, nil
}

type oauthEmailPayload struct {
	Provider string `json:"provider"`
	UID      string `json:"uid"`
	Name     string `json:"name"`
}

func (s *OAuthService) issueEmailToken(provider string, ident *providerIdentity) (string, error) {
	tok, err := randHex(32)
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(oauthEmailPayload{Provider: provider, UID: ident.UID, Name: ident.DisplayName})
	if err != nil {
		return "", err
	}
	s.emailTokens.set(s.clock(), "em:"+tok, string(raw), oauthEmailTokenTTL)
	return tok, nil
}

// OAuthPendingEmail answers the complete-email submit. The token stays valid:
// WI-A2 consumes it after email verification and creates/links the account.
type OAuthPendingEmail struct {
	Provider                  string `json:"provider"`
	Email                     string `json:"email"`
	EmailVerificationRequired bool   `json:"emailVerificationRequired"`
}

// CompleteEmail validates the temporary signup token plus the typed email.
func (s *OAuthService) CompleteEmail(_ context.Context, token, email string) (*OAuthPendingEmail, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || !strings.Contains(email, "@") {
		return nil, fmt.Errorf("%w: email", domain.ErrInvalidInput)
	}
	raw, ok := s.emailTokens.peek(s.clock(), "em:"+strings.TrimSpace(token))
	if !ok {
		return nil, domain.ErrInvalidCredentials
	}
	var pend oauthEmailPayload
	if err := json.Unmarshal([]byte(raw), &pend); err != nil {
		return nil, domain.ErrInvalidCredentials
	}
	return &OAuthPendingEmail{Provider: pend.Provider, Email: email, EmailVerificationRequired: true}, nil
}

// FinishVerifiedSignup consumes a complete-email signup token after the user
// clicked our verification link (WI-A2) and creates or links the account.
// The clicked link proves email ownership, so an existing same-email account
// is linked directly (this is our own verification, not the provider's C-11).
func (s *OAuthService) FinishVerifiedSignup(ctx context.Context, emailToken, email string) (*domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || !strings.Contains(email, "@") {
		return nil, fmt.Errorf("%w: email", domain.ErrInvalidInput)
	}
	raw, ok := s.emailTokens.take(s.clock(), "em:"+strings.TrimSpace(emailToken))
	if !ok {
		return nil, domain.ErrInvalidCredentials
	}
	var pend oauthEmailPayload
	if err := json.Unmarshal([]byte(raw), &pend); err != nil {
		return nil, domain.ErrInvalidCredentials
	}
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		randomPW := make([]byte, 32)
		if _, err := rand.Read(randomPW); err != nil {
			return nil, err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(randomPW)), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		u, err = s.users.Create(ctx, email, string(hash), domain.RoleUser, pend.Name)
		if err != nil {
			return nil, err
		}
		if err := s.users.GrantRole(ctx, u.ID, domain.RoleUser, "register"); err != nil {
			return nil, err
		}
	} else if !u.IsActive {
		return nil, domain.ErrUserInactive
	}
	if _, err := s.links.Upsert(ctx, u.ID, pend.Provider, pend.UID, &email); err != nil {
		return nil, err
	}
	return u, nil
}

// MintLinkToken issues a 5-minute token that binds a link flow to a user.
func (s *OAuthService) MintLinkToken(userID string) (string, error) {
	if strings.TrimSpace(userID) == "" {
		return "", fmt.Errorf("%w: user", domain.ErrInvalidInput)
	}
	payload := "linktok." + strings.TrimSpace(userID) + "." + strconv.FormatInt(s.clock().Add(oauthLinkTokenTTL).Unix(), 10)
	return payload + "." + hexHMAC(s.cfg.StateSecret, payload), nil
}

func (s *OAuthService) parseLinkToken(tok string) (string, error) {
	parts := strings.Split(strings.TrimSpace(tok), ".")
	if len(parts) != 4 || parts[0] != "linktok" || parts[1] == "" {
		return "", domain.ErrInvalidCredentials
	}
	if !hmacOK(s.cfg.StateSecret, strings.Join(parts[:3], "."), parts[3]) {
		return "", domain.ErrInvalidCredentials
	}
	exp, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || s.clock().After(time.Unix(exp, 0)) {
		return "", domain.ErrInvalidCredentials
	}
	return parts[1], nil
}

// OAuthConnection is one linked provider identity.
type OAuthConnection struct {
	Provider  string    `json:"provider"`
	Email     string    `json:"email,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// ListConnections returns the user's linked providers.
func (s *OAuthService) ListConnections(ctx context.Context, userID string) ([]OAuthConnection, error) {
	links, err := s.links.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]OAuthConnection, 0, len(links))
	for _, l := range links {
		c := OAuthConnection{Provider: l.Provider, CreatedAt: l.CreatedAt}
		if l.Email != nil {
			c.Email = *l.Email
		}
		out = append(out, c)
	}
	return out, nil
}

// Unlink removes one provider link. The last remaining link is refused: the
// service cannot tell a usable password from a social-minted random one, so
// it fails closed (the connections page says as much).
func (s *OAuthService) Unlink(ctx context.Context, userID, provider string) error {
	provider = strings.ToUpper(strings.TrimSpace(provider))
	if !domain.ValidOAuthProvider(provider) {
		return fmt.Errorf("%w: provider", domain.ErrInvalidInput)
	}
	links, err := s.links.ListByUserID(ctx, userID)
	if err != nil {
		return err
	}
	found := false
	for _, l := range links {
		if l.Provider == provider {
			found = true
			break
		}
	}
	if !found {
		return domain.ErrNotFound
	}
	if len(links) <= 1 {
		return domain.ErrOAuthLastMethod
	}
	return s.links.DeleteByUserProvider(ctx, userID, provider)
}

// oidcNonce derives the OIDC nonce from the state nonce (stateless: no extra
// cookie or store entry needed).
func (s *OAuthService) oidcNonce(nonce string) string {
	return hexHMAC(s.cfg.StateSecret, "avibe-oidc-v1:"+nonce)
}

func randHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func randBase64URL(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hexHMAC(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func hmacOK(secret, payload, sig string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	want := hex.EncodeToString(mac.Sum(nil))
	return subtle.ConstantTimeCompare([]byte(sig), []byte(want)) == 1
}

func strPtrOrNil(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	v := strings.TrimSpace(s)
	return &v
}
