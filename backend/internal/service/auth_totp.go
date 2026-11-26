package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
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
	// Never silent: a failing increment quietly disables the brute-force
	// budget while reads keep succeeding.
	if _, err := pipe.Exec(ctx); err != nil {
		slog.Warn("login_throttle_record_failed", "err", err)
	}
}

func (s *AuthService) clearLoginFailures(ctx context.Context, email string) {
	if s.repos.RDB == nil {
		return
	}
	_ = s.repos.RDB.Del(ctx, loginFailKey(email)).Err()
}

// totpClaimLua claims a TOTP counter step for a user: 1 when the step is
// fresh, 0 when it (or a later step) already verified. KEYS[1]=user key,
// ARGV[1]=step, ARGV[2]=ttl seconds.
const totpClaimLua = `
local last = redis.call('GET', KEYS[1])
if last and tonumber(last) >= tonumber(ARGV[1]) then return 0 end
redis.call('SET', KEYS[1], ARGV[1], 'EX', ARGV[2])
return 1
`

func totpUsedKey(userID string) string { return "totp_used:" + userID }

// claimTOTPStep binds a verified code to its counter step (single-use).
// Redis failures fail closed: an unclaimable code is unusable, exactly like
// the login throttle. A nil store (unit tests) skips the claim.
func (s *AuthService) claimTOTPStep(ctx context.Context, userID string, step int64) error {
	if s.repos == nil || s.repos.RDB == nil {
		return nil
	}
	n, err := s.repos.RDB.Eval(ctx, totpClaimLua, []string{totpUsedKey(userID)}, step, 180).Int()
	if err != nil {
		return err
	}
	if n != 1 {
		return domain.ErrTOTPInvalid
	}
	return nil
}

// checkTOTP verifies a 6-digit code or consumes a single-use recovery code.
// TOTP codes are single-use per step: without the claim below, an
// intercepted code replays for its whole ±1-step window.
func (s *AuthService) checkTOTP(ctx context.Context, u *domain.User, code string) error {
	code = strings.TrimSpace(code)
	if step, ok := apcrypto.VerifyTOTPStep(u.TOTPSecret, code, time.Now()); ok {
		return s.claimTOTPStep(ctx, u.ID, step)
	}
	hashes, err := s.repos.Users.TOTPRecoveryHashes(ctx, u.ID)
	if err != nil {
		return err
	}
	for _, h := range hashes {
		if bcrypt.CompareHashAndPassword([]byte(h), []byte(code)) == nil {
			if err := s.repos.Users.ConsumeTOTPRecoveryHash(ctx, u.ID, h); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return domain.ErrTOTPInvalid
				}
				return err
			}
			return nil
		}
	}
	return domain.ErrTOTPInvalid
}

// TOTPEnrollment is the staged secret shown once during enrollment.
type TOTPEnrollment struct {
	Secret          string `json:"secret"`
	ProvisioningURI string `json:"provisioningUri"`
}

// EnrollTOTP stages a fresh secret (a previous *staged* secret is replaced).
// Refuses accounts with 2FA already enabled: staging overwrites the live
// secret, so re-enrolling and abandoning the flow would silently strip 2FA.
// Disable first, then enroll.
func (s *AuthService) EnrollTOTP(ctx context.Context, userID string) (*TOTPEnrollment, error) {
	u, err := s.getUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u.TOTPEnabled {
		return nil, fmt.Errorf("%w: 2FA already enabled (disable first to re-enroll)", domain.ErrInvalidInput)
	}
	secret, err := apcrypto.GenerateTOTPSecret()
	if err != nil {
		return nil, err
	}
	if err := s.repos.Users.StageTOTPSecret(ctx, u.ID, secret); err != nil {
		return nil, err
	}
	return &TOTPEnrollment{
		Secret:          secret,
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
