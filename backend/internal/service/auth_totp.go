package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"avibe-platform/backend/internal/domain"
	apcrypto "avibe-platform/backend/internal/pkg/crypto"
)

// Per-account login brute-force budget: 10 failures lock the email for
// 15 minutes (Redis-backed so it survives restarts and replicas).
const (
	loginFailLimit  = 10
	loginFailWindow = 15 * time.Minute
)

func loginFailKey(email string) string { return "login_fail:" + email }

func (s *AuthService) loginThrottled(ctx context.Context, email string) (bool, error) {
	if s.repos.RDB == nil {
		return false, nil
	}
	n, err := s.repos.RDB.Get(ctx, loginFailKey(email)).Int()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}
	return n >= loginFailLimit, nil
}

func (s *AuthService) recordLoginFailure(ctx context.Context, email string) {
	if s.repos.RDB == nil {
		return
	}
	key := loginFailKey(email)
	pipe := s.repos.RDB.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, loginFailWindow)
	_, _ = pipe.Exec(ctx)
}

func (s *AuthService) clearLoginFailures(ctx context.Context, email string) {
	if s.repos.RDB == nil {
		return
	}
	_ = s.repos.RDB.Del(ctx, loginFailKey(email)).Err()
}

// checkTOTP verifies a 6-digit code or consumes a single-use recovery code.
func (s *AuthService) checkTOTP(ctx context.Context, u *domain.User, code string) error {
	code = strings.TrimSpace(code)
	if apcrypto.VerifyTOTP(u.TOTPSecret, code, time.Now()) {
		return nil
	}
	hashes, err := s.repos.Users.TOTPRecoveryHashes(ctx, u.ID)
	if err != nil {
		return err
	}
	for _, h := range hashes {
		if bcrypt.CompareHashAndPassword([]byte(h), []byte(code)) == nil {
			return s.repos.Users.ConsumeTOTPRecoveryHash(ctx, u.ID, h)
		}
	}
	return domain.ErrTOTPInvalid
}

// TOTPEnrollment is the staged secret shown once during enrollment.
type TOTPEnrollment struct {
	Secret         string `json:"secret"`
	ProvisioningURI string `json:"provisioningUri"`
}

// EnrollTOTP stages a fresh secret (previous staged secret is replaced).
func (s *AuthService) EnrollTOTP(ctx context.Context, userID string) (*TOTPEnrollment, error) {
	u, err := s.repos.Users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	secret, err := apcrypto.GenerateTOTPSecret()
	if err != nil {
		return nil, err
	}
	if err := s.repos.Users.StageTOTPSecret(ctx, u.ID, secret); err != nil {
		return nil, err
	}
	return &TOTPEnrollment{
		Secret:         secret,
		ProvisioningURI: apcrypto.TOTPProvisioningURI("AVIBE", u.Email, secret),
	}, nil
}

// ConfirmTOTPEnrollment verifies a code against the staged secret and
// activates 2FA, returning single-use recovery codes (shown once).
func (s *AuthService) ConfirmTOTPEnrollment(ctx context.Context, userID, code string) ([]string, error) {
	u, err := s.repos.Users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u.TOTPSecret == "" {
		return nil, errors.New("no staged enrollment")
	}
	if !apcrypto.VerifyTOTP(u.TOTPSecret, strings.TrimSpace(code), time.Now()) {
		return nil, domain.ErrTOTPInvalid
	}
	plain, err := apcrypto.GenerateRecoveryCodes(10)
	if err != nil {
		return nil, err
	}
	hashes := make([]string, 0, len(plain))
	for _, c := range plain {
		h, err := bcrypt.GenerateFromPassword([]byte(c), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		hashes = append(hashes, string(h))
	}
	if err := s.repos.Users.EnableTOTP(ctx, u.ID, u.TOTPSecret, hashes); err != nil {
		return nil, err
	}
	return plain, nil
}

// DisableTOTP removes 2FA after re-verifying the password, and revokes every
// session since the account's auth strength changed.
func (s *AuthService) DisableTOTP(ctx context.Context, userID, password string) error {
	u, err := s.repos.Users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return domain.ErrInvalidCredentials
	}
	if err := s.repos.Users.DisableTOTP(ctx, u.ID); err != nil {
		return err
	}
	return s.repos.Users.BumpTokenVersion(ctx, u.ID)
}
