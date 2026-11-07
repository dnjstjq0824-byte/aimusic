package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"ai-music-platform/backend/internal/config"
	"ai-music-platform/backend/internal/domain"
	"ai-music-platform/backend/internal/pkg/mail"
	"ai-music-platform/backend/internal/repository"
)

// resetUserStore is the user surface password reset needs.
type resetUserStore interface {
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	UpdatePasswordHash(ctx context.Context, id, hash string) error
}

// resetTokenStore is the reset-token surface password reset needs.
type resetTokenStore interface {
	Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*domain.PasswordResetToken, error)
	GetByHash(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error)
	MarkUsed(ctx context.Context, id string) error
	DeleteForUser(ctx context.Context, userID string) error
}

// PasswordResetService issues single-use reset links and consumes them.
// Request never reveals whether an email exists (no account enumeration).
type PasswordResetService struct {
	users    resetUserStore
	tokens   resetTokenStore
	mailer   mail.Mailer
	ttl      time.Duration
	linkBase string
	now      func() time.Time
}

// NewPasswordResetService builds the service from repositories and config.
func NewPasswordResetService(repos *repository.Repos, mailer mail.Mailer, cfg *config.Config) *PasswordResetService {
	ttl := time.Duration(cfg.Reset.TokenTTLMinutes) * time.Minute
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	return &PasswordResetService{
		users: repos.Users, tokens: repos.Resets, mailer: mailer,
		ttl: ttl, linkBase: cfg.Reset.FrontendBaseURL, now: time.Now,
	}
}

// GenerateResetToken returns a random 32-byte hex token for reset links.
func GenerateResetToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// HashResetToken hashes a raw token for storage (tokens are secrets at rest).
func HashResetToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// BuildResetLink renders the frontend reset URL carrying the raw token.
func BuildResetLink(frontendBase, token string) string {
	return strings.TrimRight(frontendBase, "/") + "/reset-password?token=" + token
}

// Request issues a reset link for email. Unknown or inactive accounts yield
// success without side effects so callers cannot enumerate accounts.
func (s *PasswordResetService) Request(ctx context.Context, email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if !strings.Contains(email, "@") {
		return fmt.Errorf("%w: email", domain.ErrInvalidInput)
	}
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil || !u.IsActive {
		return nil
	}
	token, err := GenerateResetToken()
	if err != nil {
		return err
	}
	if err := s.tokens.DeleteForUser(ctx, u.ID); err != nil {
		return err
	}
	if _, err := s.tokens.Create(ctx, u.ID, HashResetToken(token), s.now().Add(s.ttl)); err != nil {
		return err
	}
	link := BuildResetLink(s.linkBase, token)
	body := "AIVIBE 비밀번호 재설정 링크입니다 (" + ttlMinutes(s.ttl) + "분 유효, 1회용):\n" + link +
		"\n\n본인이 요청하지 않았다면 이 메일을 무시하세요."
	return s.mailer.Send(ctx, u.Email, "[AIVIBE] 비밀번호 재설정", body)
}

func ttlMinutes(d time.Duration) string {
	return strings.TrimSpace(fmt.Sprintf("%d", int(d.Minutes())))
}

// Confirm consumes a reset token and sets a new password.
func (s *PasswordResetService) Confirm(ctx context.Context, token, newPassword string) error {
	if strings.TrimSpace(token) == "" || len(newPassword) < 8 {
		return fmt.Errorf("%w: token/newPassword", domain.ErrInvalidInput)
	}
	t, err := s.tokens.GetByHash(ctx, HashResetToken(strings.TrimSpace(token)))
	if err != nil {
		return domain.ErrInvalidCredentials
	}
	if t.UsedAt != nil || !s.now().Before(t.ExpiresAt) {
		return domain.ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.users.UpdatePasswordHash(ctx, t.UserID, string(hash)); err != nil {
		return err
	}
	if err := s.tokens.MarkUsed(ctx, t.ID); err != nil {
		return domain.ErrInvalidCredentials
	}
	return nil
}
