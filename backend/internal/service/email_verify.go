package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"avibe-platform/backend/internal/config"
	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/pkg/mail"
	"avibe-platform/backend/internal/repository"
)

// verifyUserStore is the user surface email verification needs.
type verifyUserStore interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
	MarkEmailVerified(ctx context.Context, id string) error
}

// verifyTokenStore is the verification-token surface email verification needs.
type verifyTokenStore interface {
	Create(ctx context.Context, userID, email, purpose, tokenHash, refToken string, expiresAt time.Time) (*domain.EmailVerification, error)
	GetByHash(ctx context.Context, tokenHash string) (*domain.EmailVerification, error)
	MarkUsed(ctx context.Context, id string) error
	DeletePendingForUser(ctx context.Context, userID string) error
	DeletePendingForEmail(ctx context.Context, email, purpose string) error
}

// verifyTokenIssuer mints login sessions for verified users.
type verifyTokenIssuer interface {
	Issue(ctx context.Context, userID, role string) (*Tokens, error)
}

// verifyOAuthSignup finishes pending social signups after verification.
type verifyOAuthSignup interface {
	FinishVerifiedSignup(ctx context.Context, emailToken, email string) (*domain.User, error)
}

// EmailVerifyService issues single-use verification links and consumes them.
// Confirm also logs the user in (the 24h single-use link is a bearer secret
// delivered to the account email), so clicking the mailed link continues in a
// logged-in state.
type EmailVerifyService struct {
	users    verifyUserStore
	tokens   verifyTokenStore
	mailer   mail.Mailer
	issuer   verifyTokenIssuer
	oauth    verifyOAuthSignup
	ttl      time.Duration
	linkBase string
	now      func() time.Time
}

// NewEmailVerifyService builds the service from repositories and config.
func NewEmailVerifyService(repos *repository.Repos, auth *AuthService, mailer mail.Mailer, oauth *OAuthService, cfg *config.Config) *EmailVerifyService {
	ttl := time.Duration(cfg.Verify.TokenTTLHours) * time.Hour
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &EmailVerifyService{
		users: repos.Users, tokens: repos.EmailVerify, mailer: mailer,
		issuer: auth, oauth: oauth,
		ttl: ttl, linkBase: cfg.Reset.FrontendBaseURL, now: time.Now,
	}
}

// GenerateVerifyToken returns a random 32-byte hex token for verify links.
func GenerateVerifyToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// HashVerifyToken hashes a raw token for storage (tokens are secrets at rest).
func HashVerifyToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// BuildVerifyLink renders the frontend verification URL carrying the raw token.
func BuildVerifyLink(frontendBase, token string) string {
	return strings.TrimRight(frontendBase, "/") + "/verify-email?token=" + token
}

// Request mails a fresh verification link to the caller's own address in
// their locale. Verified accounts are a silent no-op (idempotent resend).
func (s *EmailVerifyService) Request(ctx context.Context, userID string) error {
	if strings.TrimSpace(userID) == "" {
		return domain.ErrInvalidCredentials
	}
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if u.EmailVerifiedAt != nil {
		return nil
	}
	token, err := GenerateVerifyToken()
	if err != nil {
		return err
	}
	if err := s.tokens.DeletePendingForUser(ctx, u.ID); err != nil {
		return err
	}
	if _, err := s.tokens.Create(ctx, u.ID, u.Email, domain.VerifyPurposeAccount,
		HashVerifyToken(token), "", s.now().Add(s.ttl)); err != nil {
		return err
	}
	link := BuildVerifyLink(s.linkBase, token)
	subject, body := mail.RenderVerify(u.Locale, link, int(s.ttl.Hours()))
	return s.mailer.Send(ctx, u.Email, subject, body)
}

// RequestOAuthSignup mails a verification link for a pending social signup
// (A1 complete-email): the account does not exist yet, so the A1 signup
// token rides along as the row's ref and is consumed on confirm.
func (s *EmailVerifyService) RequestOAuthSignup(ctx context.Context, email, emailToken, locale string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	emailToken = strings.TrimSpace(emailToken)
	if !strings.Contains(email, "@") || emailToken == "" {
		return fmt.Errorf("%w: email/emailToken", domain.ErrInvalidInput)
	}
	token, err := GenerateVerifyToken()
	if err != nil {
		return err
	}
	if err := s.tokens.DeletePendingForEmail(ctx, email, domain.VerifyPurposeOAuthSignup); err != nil {
		return err
	}
	if _, err := s.tokens.Create(ctx, "", email, domain.VerifyPurposeOAuthSignup,
		HashVerifyToken(token), emailToken, s.now().Add(s.ttl)); err != nil {
		return err
	}
	link := BuildVerifyLink(s.linkBase, token)
	subject, body := mail.RenderVerify(locale, link, int(s.ttl.Hours()))
	return s.mailer.Send(ctx, email, subject, body)
}

// EmailVerifyResult is a consumed verification: the verified user plus a
// fresh login session.
type EmailVerifyResult struct {
	User   *domain.User
	Tokens *Tokens
}

// Confirm consumes a verification link, marks the email verified, and logs
// the user in. Unknown, expired, or replayed tokens all fail identically.
func (s *EmailVerifyService) Confirm(ctx context.Context, token string) (*EmailVerifyResult, error) {
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("%w: token", domain.ErrInvalidInput)
	}
	v, err := s.tokens.GetByHash(ctx, HashVerifyToken(strings.TrimSpace(token)))
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}
	if v.UsedAt != nil || !s.now().Before(v.ExpiresAt) {
		return nil, domain.ErrInvalidCredentials
	}
	// Single-use first: even if the signup finish below fails (stale OAuth
	// token), the clicked link dies and the user restarts from login.
	if err := s.tokens.MarkUsed(ctx, v.ID); err != nil {
		return nil, domain.ErrInvalidCredentials
	}
	switch v.Purpose {
	case domain.VerifyPurposeAccount:
		return s.confirmAccount(ctx, v)
	case domain.VerifyPurposeOAuthSignup:
		return s.confirmOAuth(ctx, v)
	default:
		return nil, domain.ErrInvalidCredentials
	}
}

func (s *EmailVerifyService) confirmAccount(ctx context.Context, v *domain.EmailVerification) (*EmailVerifyResult, error) {
	if v.UserID == nil || strings.TrimSpace(*v.UserID) == "" {
		return nil, domain.ErrInvalidCredentials
	}
	u, err := s.users.GetByID(ctx, *v.UserID)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}
	if !u.IsActive {
		return nil, domain.ErrUserInactive
	}
	if err := s.users.MarkEmailVerified(ctx, u.ID); err != nil {
		return nil, err
	}
	now := s.now()
	u.EmailVerifiedAt = &now
	tokens, err := s.issuer.Issue(ctx, u.ID, u.Role)
	if err != nil {
		return nil, err
	}
	return &EmailVerifyResult{User: u, Tokens: tokens}, nil
}

func (s *EmailVerifyService) confirmOAuth(ctx context.Context, v *domain.EmailVerification) (*EmailVerifyResult, error) {
	if s.oauth == nil || v.RefToken == nil || strings.TrimSpace(*v.RefToken) == "" {
		return nil, domain.ErrInvalidCredentials
	}
	u, err := s.oauth.FinishVerifiedSignup(ctx, strings.TrimSpace(*v.RefToken), v.Email)
	if err != nil {
		return nil, err
	}
	if err := s.users.MarkEmailVerified(ctx, u.ID); err != nil {
		return nil, err
	}
	now := s.now()
	u.EmailVerifiedAt = &now
	tokens, err := s.issuer.Issue(ctx, u.ID, u.Role)
	if err != nil {
		return nil, err
	}
	return &EmailVerifyResult{User: u, Tokens: tokens}, nil
}
