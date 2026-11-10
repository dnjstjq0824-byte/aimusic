package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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
	UserID       string `json:"uid"`
	Role         string `json:"role"`
	TokenType    string `json:"typ"`
	TokenVersion int    `json:"tver"`
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
		return fmt.Errorf("%w: 만 14세 이상 확인이 필요합니다", domain.ErrInvalidInput)
	}
	if birthYear != nil {
		year := *birthYear
		now := time.Now().Year()
		if year < 1900 || year > now {
			return fmt.Errorf("%w: birthYear", domain.ErrInvalidInput)
		}
		if now-year < 14 {
			return fmt.Errorf("%w: 만 14세 미만은 가입할 수 없습니다", domain.ErrInvalidInput)
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
	return s.repos.Users.CreateWithProfile(ctx, email, string(hash), domain.RoleUser, displayName, birthYear, ageConfirmed)
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
	if contactEmail == "" {
		contactEmail = email
	}
	p, err := s.repos.Partners.Create(ctx, u.ID, company, contactEmail)
	if err != nil {
		return nil, nil, err
	}
	return u, p, nil
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
	tokens, err := s.issue(u.ID, u.Role, u.TokenVersion)
	if err != nil {
		return nil, nil, err
	}
	return tokens, u, nil
}

// Refresh rotates a refresh token into a new pair. Stale token versions
// (after password change, withdraw, or logout-all) are rejected.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*Tokens, error) {
	claims, err := s.parse(refreshToken, s.refreshKey, TokenRefresh)
	if err != nil {
		return nil, err
	}
	u, err := s.repos.Users.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}
	if !u.IsActive {
		return nil, domain.ErrUserInactive
	}
	if claims.TokenVersion != u.TokenVersion {
		return nil, domain.ErrInvalidCredentials
	}
	return s.issue(u.ID, u.Role, u.TokenVersion)
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

// ConfirmAge records the 14+ self-declaration for accounts created without
// one (social login). Under-14 users must not confirm; the API cannot verify
// real age, so the declaration is the legal attestation point.
func (s *AuthService) ConfirmAge(ctx context.Context, userID string, ageConfirmed bool, birthYear *int) error {
	if userID == "" {
		return domain.ErrInvalidCredentials
	}
	if !ageConfirmed {
		return fmt.Errorf("%w: 만 14세 이상 확인이 필요합니다", domain.ErrInvalidInput)
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
	return s.issue(userID, role, version)
}

// Withdraw deactivates the user account immediately (in-app immediate
// withdrawal compliance), purges personal data, and revokes every
// outstanding token. Tax/legal retention rows (settlements, consents,
// streaming logs) stay pseudonymous under the anonymized alias.
func (s *AuthService) Withdraw(ctx context.Context, userID string) error {
	if userID == "" {
		return domain.ErrInvalidCredentials
	}
	if err := s.repos.Users.SetActive(ctx, userID, false); err != nil {
		return err
	}
	if err := s.repos.Users.PurgePII(ctx, userID); err != nil {
		return err
	}
	if err := s.repos.Partners.PurgePIIByUser(ctx, userID); err != nil {
		return err
	}
	_ = s.repos.Billing.CancelActiveSubscription(ctx, userID)
	_ = s.repos.Resets.DeleteForUser(ctx, userID)
	_ = s.repos.OAuth.DeleteForUser(ctx, userID)
	_ = s.repos.Notify.DeleteForUser(ctx, userID)
	return s.repos.Users.BumpTokenVersion(ctx, userID)
}

func (s *AuthService) issue(userID, role string, version int) (*Tokens, error) {
	now := time.Now()
	access := &Claims{
		UserID: userID, Role: role, TokenType: TokenAccess, TokenVersion: version,
		RegisteredClaims: jwt.RegisteredClaims{
			ID: uuid.NewString(), IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
		},
	}
	refresh := &Claims{
		UserID: userID, Role: role, TokenType: TokenRefresh, TokenVersion: version,
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
