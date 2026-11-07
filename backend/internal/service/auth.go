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
type Claims struct {
	UserID    string `json:"uid"`
	Role      string `json:"role"`
	TokenType string `json:"typ"`
	jwt.RegisteredClaims
}

// AuthService handles registration, login, and token rotation.
type AuthService struct {
	repos      *repository.Repos
	accessKey  []byte
	refreshKey []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewAuthService builds an AuthService from config.
func NewAuthService(repos *repository.Repos, cfg *config.Config) *AuthService {
	return &AuthService{
		repos:      repos,
		accessKey:  []byte(cfg.JWT.AccessSecret),
		refreshKey: []byte(cfg.JWT.RefreshSecret),
		accessTTL:  time.Duration(cfg.JWT.AccessTTLMinutes) * time.Minute,
		refreshTTL: time.Duration(cfg.JWT.RefreshTTLDays) * 24 * time.Hour,
	}
}

// Register creates a USER account.
func (s *AuthService) Register(ctx context.Context, email, password, displayName string) (*domain.User, error) {
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
	return s.repos.Users.Create(ctx, email, string(hash), domain.RoleUser, displayName)
}

// RegisterPartner creates a PARTNER account with a pending profile.
func (s *AuthService) RegisterPartner(ctx context.Context, email, password, displayName, company, contactEmail string) (*domain.User, *domain.PartnerProfile, error) {
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
	u, err := s.repos.Users.Create(ctx, email, string(hash), domain.RolePartner, displayName)
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

// Login verifies credentials and issues tokens.
func (s *AuthService) Login(ctx context.Context, email, password string) (*Tokens, *domain.User, error) {
	u, err := s.repos.Users.GetByEmail(ctx, strings.TrimSpace(strings.ToLower(email)))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, domain.ErrInvalidCredentials
		}
		return nil, nil, err
	}
	if !u.IsActive {
		return nil, nil, domain.ErrUserInactive
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, nil, domain.ErrInvalidCredentials
	}
	tokens, err := s.issue(u.ID, u.Role)
	if err != nil {
		return nil, nil, err
	}
	return tokens, u, nil
}

// Refresh rotates a refresh token into a new pair.
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
	return s.issue(u.ID, u.Role)
}

// ParseAccess validates an access token.
func (s *AuthService) ParseAccess(token string) (*Claims, error) {
	return s.parse(token, s.accessKey, TokenAccess)
}

// Issue mints a token pair for an already-authenticated principal
// (used by social login after provider verification).
func (s *AuthService) Issue(userID, role string) (*Tokens, error) {
	return s.issue(userID, role)
}

// Withdraw deactivates the user account immediately (in-app immediate withdrawal compliance).
func (s *AuthService) Withdraw(ctx context.Context, userID string) error {
	if userID == "" {
		return domain.ErrInvalidCredentials
	}
	return s.repos.Users.SetActive(ctx, userID, false)
}

func (s *AuthService) issue(userID, role string) (*Tokens, error) {
	now := time.Now()
	access := &Claims{
		UserID: userID, Role: role, TokenType: TokenAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ID: uuid.NewString(), IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
		},
	}
	refresh := &Claims{
		UserID: userID, Role: role, TokenType: TokenRefresh,
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
