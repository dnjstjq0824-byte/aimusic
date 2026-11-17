package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"avibe-platform/backend/internal/config"
	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/repository"
)

// Token types carried in claims.
const (
	TokenAccess  = "access"
	TokenRefresh = "refresh"
)

// Claims is the JWT claim set for access and refresh tokens.
//
// TokenVersion must equal the account's current version; password change,
// withdraw, and logout-all bump the version and instantly revoke every
// outstanding token (checked on refresh; access tokens additionally expire
// by their short TTL).
type Claims struct {
	UserID       string   `json:"uid"`
	Role         string   `json:"role"`
	Roles        []string `json:"roles,omitempty"`
	TokenType    string   `json:"typ"`
	TokenVersion int      `json:"tver"`
	jwt.RegisteredClaims
}

// AuthService handles registration, login, and token rotation.
type AuthService struct {
	repos      *repository.Repos
	accessKey  []byte
	refreshKey []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	inviteCode string
	// getUser resolves accounts for per-request validation (override in tests).
	getUser func(ctx context.Context, userID string) (*domain.User, error)
	// listRoles resolves effective roles (override in tests; nil = [primary]).
	listRoles func(ctx context.Context, userID string) ([]string, error)
	// bumpVersion revokes every outstanding token (override in tests).
	bumpVersion func(ctx context.Context, userID string) error
	// refreshTracker enforces single-use rotation (nil disables tracking).
	refreshTracker refreshTracker
}

// refreshConsume is the outcome of presenting a refresh-token jti.
type refreshConsume int

const (
	// refreshUnknown was never tracked: a token issued before rotation
	// tracking deployed. Graced exactly once, then tracked.
	refreshUnknown refreshConsume = iota
	// refreshLive was tracked and is now consumed.
	refreshLive
	// refreshReused was already consumed: a theft signal, revoke the family.
	refreshReused
	// refreshMismatch is tracked under another user: reject without revoke.
	refreshMismatch
)

// refreshTracker records refresh-token ids at issue and consumes them at rotation.
type refreshTracker interface {
	record(ctx context.Context, jti, userID string) error
	consume(ctx context.Context, jti, userID string) (refreshConsume, error)
	markUsed(ctx context.Context, jti, userID string) error
}

// redisRefreshTracker implements refreshTracker over Redis. Consume runs as
// one Lua script so concurrent double-refresh cannot both slip through.
type redisRefreshTracker struct {
	rdb *redis.Client
	ttl time.Duration
}

// refreshConsumeLua returns 1 (consumed live), 2 (reused), 0 (unknown) or
// 3 (tracked under another user). KEYS[1]=live, KEYS[2]=used,
// ARGV[1]=userID, ARGV[2]=ttl seconds.
const refreshConsumeLua = `
local live = redis.call('GET', KEYS[1])
if live then
  if live ~= ARGV[1] then return 3 end
  redis.call('DEL', KEYS[1])
  redis.call('SET', KEYS[2], ARGV[1], 'EX', ARGV[2])
  return 1
end
local used = redis.call('GET', KEYS[2])
if used then
  if used ~= ARGV[1] then return 3 end
  return 2
end
return 0
`

func refreshLiveKey(jti string) string { return "refresh:live:" + jti }
func refreshUsedKey(jti string) string { return "refresh:used:" + jti }

func (t *redisRefreshTracker) record(ctx context.Context, jti, userID string) error {
	return t.rdb.Set(ctx, refreshLiveKey(jti), userID, t.ttl).Err()
}

func (t *redisRefreshTracker) markUsed(ctx context.Context, jti, userID string) error {
	return t.rdb.Set(ctx, refreshUsedKey(jti), userID, t.ttl).Err()
}

func (t *redisRefreshTracker) consume(ctx context.Context, jti, userID string) (refreshConsume, error) {
	n, err := t.rdb.Eval(ctx, refreshConsumeLua,
		[]string{refreshLiveKey(jti), refreshUsedKey(jti)},
		userID, int64(t.ttl/time.Second)).Int()
	if err != nil {
		return refreshUnknown, err
	}
	switch n {
	case 1:
		return refreshLive, nil
	case 2:
		return refreshReused, nil
	case 3:
		return refreshMismatch, nil
	default:
		return refreshUnknown, nil
	}
}

// NewAuthService builds an AuthService from config.
func NewAuthService(repos *repository.Repos, cfg *config.Config) *AuthService {
	s := &AuthService{
		repos:      repos,
		accessKey:  []byte(cfg.JWT.AccessSecret),
		refreshKey: []byte(cfg.JWT.RefreshSecret),
		accessTTL:  time.Duration(cfg.JWT.AccessTTLMinutes) * time.Minute,
		refreshTTL: time.Duration(cfg.JWT.RefreshTTLDays) * 24 * time.Hour,
		inviteCode: cfg.Security.RegistrationInviteCode,
	}
	if repos != nil && repos.Users != nil {
		users := repos.Users
		s.getUser = users.GetByID
		s.listRoles = users.ListRoles
		s.bumpVersion = users.BumpTokenVersion
		if repos.RDB != nil {
			s.refreshTracker = &redisRefreshTracker{rdb: repos.RDB, ttl: s.refreshTTL}
		}
	}
	return s
}

// checkInvite enforces the pre-launch invite gate (C8). Open when unset.
func (s *AuthService) checkInvite(code string) error {
	if s.inviteCode == "" {
		return nil
	}
	if strings.TrimSpace(code) != s.inviteCode {
		return domain.ErrForbidden
	}
	return nil
}

// checkAge enforces the 14+ gate (C9): callers must confirm, and an optional
// birth year must itself prove 14+.
func checkAge(birthYear *int, ageConfirmed bool) error {
	if !ageConfirmed {
		return fmt.Errorf("%w: ageConfirmed must be true (14+)", domain.ErrInvalidInput)
	}
	if birthYear != nil {
		year := *birthYear
		now := time.Now().Year()
		if year < 1900 || year > now {
			return fmt.Errorf("%w: birthYear", domain.ErrInvalidInput)
		}
		if now-year < 14 {
			return fmt.Errorf("%w: under-14 cannot register", domain.ErrInvalidInput)
		}
	}
	return nil
}

// Register creates a USER account.
func (s *AuthService) Register(ctx context.Context, email, password, displayName string, birthYear *int, ageConfirmed bool, inviteCode string) (*domain.User, error) {
	if err := s.checkInvite(inviteCode); err != nil {
		return nil, err
	}
	if err := checkAge(birthYear, ageConfirmed); err != nil {
		return nil, err
	}
	email = strings.TrimSpace(strings.ToLower(email))
	if !strings.Contains(email, "@") || displayName == "" || len(password) < 8 {
		return nil, fmt.Errorf("%w: email/displayName/password", domain.ErrInvalidInput)
	}
	if _, err := s.repos.Users.GetByEmail(ctx, email); err == nil {
		return nil, domain.ErrEmailTaken
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u, err := s.repos.Users.CreateWithProfile(ctx, email, string(hash), domain.RoleUser, displayName, birthYear, ageConfirmed)
	if err != nil {
		return nil, err
	}
	if err := s.repos.Users.GrantRole(ctx, u.ID, domain.RoleUser, "register"); err != nil {
		return nil, err
	}
	return u, nil
}

// RegisterPartner creates a PARTNER account with a pending profile.
func (s *AuthService) RegisterPartner(ctx context.Context, email, password, displayName, company, contactEmail string, birthYear *int, ageConfirmed bool, inviteCode string) (*domain.User, *domain.PartnerProfile, error) {
	if err := s.checkInvite(inviteCode); err != nil {
		return nil, nil, err
	}
	if err := checkAge(birthYear, ageConfirmed); err != nil {
		return nil, nil, err
	}
	email = strings.TrimSpace(strings.ToLower(email))
	if !strings.Contains(email, "@") || company == "" || len(password) < 8 {
		return nil, nil, fmt.Errorf("%w: email/company/password", domain.ErrInvalidInput)
	}
	if _, err := s.repos.Users.GetByEmail(ctx, email); err == nil {
		return nil, nil, domain.ErrEmailTaken
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, err
	}
	u, err := s.repos.Users.CreateWithProfile(ctx, email, string(hash), domain.RolePartner, displayName, birthYear, ageConfirmed)
	if err != nil {
		return nil, nil, err
	}
	if err := s.repos.Users.GrantRole(ctx, u.ID, domain.RolePartner, "register"); err != nil {
		return nil, nil, err
	}
	if contactEmail == "" {
		contactEmail = email
	}
	p, err := s.repos.Partners.Create(ctx, u.ID, company, contactEmail)
	if err != nil {
		return nil, nil, err
	}
	// S1: every partner is addressable at /creators/{handle} from day one.
	// Claim failures (collision) retry with a fresh suffix; a persistent
	// failure still returns the account (handle stays claimable later).
	if s.repos == nil || s.repos.SEO == nil {
		return u, p, nil
	}
	for i := 0; i < 3; i++ {
		suffix, err := domain.NewShortID()
		if err != nil {
			break
		}
		handle := "creator_" + strings.ToLower(suffix)
		if err := s.repos.SEO.ClaimHandle(ctx, u.ID, handle); err == nil {
			u.Handle = &handle
			break
		} else if !errors.Is(err, domain.ErrHandleTaken) {
			break
		}
	}
	return u, p, nil
}

// RegisterBusiness creates a BUSINESS account with a company profile (R10).
// KR accounts must present a checksum-valid business number and verify
// immediately; other countries stay PENDING for manual admin approval.
func (s *AuthService) RegisterBusiness(ctx context.Context, email, password, displayName, company, bizNo, country, industry string, birthYear *int, ageConfirmed bool, inviteCode string) (*domain.User, *domain.BusinessProfile, error) {
	if err := s.checkInvite(inviteCode); err != nil {
		return nil, nil, err
	}
	if err := checkAge(birthYear, ageConfirmed); err != nil {
		return nil, nil, err
	}
	email = strings.TrimSpace(strings.ToLower(email))
	company = strings.TrimSpace(company)
	country = strings.ToUpper(strings.TrimSpace(country))
	if country == "" {
		country = "KR"
	}
	industry = strings.TrimSpace(industry)
	if !strings.Contains(email, "@") || company == "" || len(password) < 8 {
		return nil, nil, fmt.Errorf("%w: email/company/password", domain.ErrInvalidInput)
	}
	if len(company) > 200 || len(industry) > 200 || len(country) > 16 {
		return nil, nil, fmt.Errorf("%w: company/industry/country too long", domain.ErrInvalidInput)
	}
	normalized := ""
	status := domain.PartnerPending
	if country == "KR" {
		var err error
		normalized, err = domain.NormalizeBusinessNumber(bizNo)
		if err != nil {
			return nil, nil, err
		}
		status = domain.PartnerApproved
	}
	if _, err := s.repos.Users.GetByEmail(ctx, email); err == nil {
		return nil, nil, domain.ErrEmailTaken
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, err
	}
	if displayName == "" {
		displayName = company
	}
	u, err := s.repos.Users.CreateWithProfile(ctx, email, string(hash), domain.RoleBusiness, displayName, birthYear, ageConfirmed)
	if err != nil {
		return nil, nil, err
	}
	p, err := s.repos.Business.CreateBusinessProfile(ctx, u.ID, company, normalized, country, industry, status)
	if err != nil {
		return nil, nil, err
	}
	if err := s.repos.Users.GrantRole(ctx, u.ID, domain.RoleBusiness, "register"); err != nil {
		return nil, nil, err
	}
	return u, p, nil
}

// EffectiveRoles returns the account's roles for API responses (R10).
func (s *AuthService) EffectiveRoles(ctx context.Context, userID, primary string) []string {
	return s.effectiveRoles(ctx, userID, primary)
}

// Tokens is an issued token pair.
type Tokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
}

// Login verifies credentials (+ TOTP when enrolled) and issues tokens.
// totpCode is the 6-digit code or a single-use recovery code.
func (s *AuthService) Login(ctx context.Context, email, password, totpCode string) (*Tokens, *domain.User, error) {
	mail := strings.TrimSpace(strings.ToLower(email))
	if limited, err := s.loginThrottled(ctx, mail); err != nil {
		return nil, nil, err
	} else if limited {
		return nil, nil, domain.ErrTooManyAttempts
	}
	fail := func() (*Tokens, *domain.User, error) {
		s.recordLoginFailure(ctx, mail)
		return nil, nil, domain.ErrInvalidCredentials
	}
	u, err := s.repos.Users.GetByEmail(ctx, mail)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return fail()
		}
		return nil, nil, err
	}
	if !u.IsActive {
		return nil, nil, domain.ErrUserInactive
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return fail()
	}
	if u.TOTPEnabled {
		if strings.TrimSpace(totpCode) == "" {
			return nil, nil, domain.ErrTOTPRequired
		}
		if err := s.checkTOTP(ctx, u, totpCode); err != nil {
			s.recordLoginFailure(ctx, mail)
			return nil, nil, err
		}
	}
	s.clearLoginFailures(ctx, mail)
	tokens, err := s.issue(ctx, u.ID, u.Role, u.TokenVersion)
	if err != nil {
		return nil, nil, err
	}
	return tokens, u, nil
}

// Refresh rotates a refresh token into a new pair. Rotation is single-use:
// replaying a consumed token revokes every token on the account (B-09).
// Stale token versions (after password change, withdraw, or logout-all)
// are rejected. Tokens issued before tracking deployed are graced once.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*Tokens, error) {
	claims, err := s.parse(refreshToken, s.refreshKey, TokenRefresh)
	if err != nil {
		return nil, err
	}
	grace := false
	if s.refreshTracker != nil {
		st, err := s.refreshTracker.consume(ctx, claims.ID, claims.UserID)
		if err != nil {
			return nil, err
		}
		switch st {
		case refreshReused:
			if s.bumpVersion != nil {
				if err := s.bumpVersion(ctx, claims.UserID); err != nil {
					return nil, err
				}
			}
			return nil, domain.ErrInvalidCredentials
		case refreshMismatch:
			return nil, domain.ErrInvalidCredentials
		case refreshUnknown:
			grace = true
		case refreshLive:
		}
	}
	u, err := s.getUser(ctx, claims.UserID)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}
	if !u.IsActive {
		return nil, domain.ErrUserInactive
	}
	if claims.TokenVersion != u.TokenVersion {
		return nil, domain.ErrInvalidCredentials
	}
	if grace {
		if err := s.refreshTracker.markUsed(ctx, claims.ID, claims.UserID); err != nil {
			return nil, err
		}
	}
	return s.issue(ctx, u.ID, u.Role, u.TokenVersion)
}

// LogoutAll revokes every outstanding token for the account.
func (s *AuthService) LogoutAll(ctx context.Context, userID string) error {
	if userID == "" {
		return domain.ErrInvalidCredentials
	}
	return s.repos.Users.BumpTokenVersion(ctx, userID)
}

// Profile returns the authenticated user's row (drives /auth/me).
func (s *AuthService) Profile(ctx context.Context, userID string) (*domain.User, error) {
	if userID == "" {
		return nil, domain.ErrInvalidCredentials
	}
	return s.repos.Users.GetByID(ctx, userID)
}

// UpdateDisplayName changes an account's display name (R8 profile).
func (s *AuthService) UpdateDisplayName(ctx context.Context, userID, displayName string) (*domain.User, error) {
	if userID == "" {
		return nil, domain.ErrInvalidCredentials
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" || len([]rune(displayName)) > 30 {
		return nil, fmt.Errorf("%w: displayName must be 1-30 chars", domain.ErrInvalidInput)
	}
	if err := s.repos.Users.SetDisplayName(ctx, userID, displayName); err != nil {
		return nil, err
	}
	return s.repos.Users.GetByID(ctx, userID)
}

// UpdateLocale changes the caller's display language (A2).
func (s *AuthService) UpdateLocale(ctx context.Context, userID, locale string) (*domain.User, error) {
	if userID == "" {
		return nil, domain.ErrInvalidCredentials
	}
	locale = strings.TrimSpace(locale)
	if !domain.ValidLocale(locale) {
		return nil, fmt.Errorf("%w: locale must be ko|en|ja|zh-cn|th", domain.ErrInvalidInput)
	}
	if err := s.repos.Users.SetLocale(ctx, userID, locale); err != nil {
		return nil, err
	}
	return s.repos.Users.GetByID(ctx, userID)
}

// ConfirmAge records the 14+ self-declaration for accounts created without
// one (social login). Under-14 users must not confirm; the API cannot verify
// real age, so the declaration is the legal attestation point.
func (s *AuthService) ConfirmAge(ctx context.Context, userID string, ageConfirmed bool, birthYear *int) error {
	if userID == "" {
		return domain.ErrInvalidCredentials
	}
	if !ageConfirmed {
		return fmt.Errorf("%w: ageConfirmed must be true (14+)", domain.ErrInvalidInput)
	}
	return s.repos.Users.ConfirmAge(ctx, userID, birthYear)
}

// ParseAccess validates an access token.
func (s *AuthService) ParseAccess(token string) (*Claims, error) {
	return s.parse(token, s.accessKey, TokenAccess)
}

// ValidateAccess parses an access token and revalidates it against the live
// account: deactivated accounts and stale token versions (after password
// change, withdraw, or logout-all) are rejected. The returned claims carry
// the current DB role so role changes take effect without re-login.
func (s *AuthService) ValidateAccess(ctx context.Context, token string) (*Claims, error) {
	claims, err := s.parse(token, s.accessKey, TokenAccess)
	if err != nil {
		return nil, err
	}
	if s.getUser == nil {
		return nil, domain.ErrInvalidCredentials
	}
	u, err := s.getUser(ctx, claims.UserID)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}
	if !u.IsActive {
		return nil, domain.ErrUserInactive
	}
	if claims.TokenVersion != u.TokenVersion {
		return nil, domain.ErrInvalidCredentials
	}
	claims.Role = u.Role
	claims.Roles = s.effectiveRoles(ctx, claims.UserID, u.Role)
	return claims, nil
}

// Issue mints a token pair for an already-authenticated principal
// (used by social login after provider verification).
func (s *AuthService) Issue(ctx context.Context, userID, role string) (*Tokens, error) {
	version := 0
	if s.repos != nil && s.repos.Users != nil {
		if u, err := s.repos.Users.GetByID(ctx, userID); err == nil {
			version = u.TokenVersion
		}
	}
	return s.issue(ctx, userID, role, version)
}

// Withdraw deactivates the user account immediately (in-app immediate
// withdrawal compliance), purges personal data, and revokes every
// outstanding token. Tax/legal retention rows (settlements, consents,
// streaming logs) stay pseudonymous under the anonymized alias.
func (s *AuthService) Withdraw(ctx context.Context, userID string) error {
	if userID == "" {
		return domain.ErrInvalidCredentials
	}
	return s.repos.WithdrawUserTx(ctx, userID)
}

// effectiveRoles returns the account's roles for the JWT claim, falling
// back to the primary role when the grant table is unreachable (tests,
// pre-R10 databases mid-migration).
func (s *AuthService) effectiveRoles(ctx context.Context, userID, primary string) []string {
	if s.listRoles != nil {
		if roles, err := s.listRoles(ctx, userID); err == nil && len(roles) > 0 {
			return roles
		}
	}
	return []string{primary}
}

func (s *AuthService) issue(ctx context.Context, userID, role string, version int) (*Tokens, error) {
	now := time.Now()
	roles := s.effectiveRoles(ctx, userID, role)
	access := &Claims{
		UserID: userID, Role: role, Roles: roles, TokenType: TokenAccess, TokenVersion: version,
		RegisteredClaims: jwt.RegisteredClaims{
			ID: uuid.NewString(), IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
		},
	}
	refresh := &Claims{
		UserID: userID, Role: role, Roles: roles, TokenType: TokenRefresh, TokenVersion: version,
		RegisteredClaims: jwt.RegisteredClaims{
			ID: uuid.NewString(), IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshTTL)),
		},
	}
	accessStr, err := jwt.NewWithClaims(jwt.SigningMethodHS256, access).SignedString(s.accessKey)
	if err != nil {
		return nil, err
	}
	refreshStr, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refresh).SignedString(s.refreshKey)
	if err != nil {
		return nil, err
	}
	if s.refreshTracker != nil {
		if err := s.refreshTracker.record(ctx, refresh.ID, userID); err != nil {
			return nil, err
		}
	}
	return &Tokens{AccessToken: accessStr, RefreshToken: refreshStr, ExpiresIn: int64(s.accessTTL.Seconds())}, nil
}

func (s *AuthService) parse(token string, key []byte, wantType string) (*Claims, error) {
	claims := &Claims{}
	t, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return key, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !t.Valid || claims.TokenType != wantType {
		return nil, domain.ErrInvalidCredentials
	}
	return claims, nil
}
