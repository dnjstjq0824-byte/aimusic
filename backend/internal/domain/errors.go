// Package domain holds shared entities, enums, and error codes.
package domain

import (
	"errors"
	"net/http"
)

// License-gate error codes returned with HTTP 403 (duplicate maps to 409).
const (
	CodeNonCommercialTier         = "ERR_NON_COMMERCIAL_TIER"
	CodeRetroactiveGenerationDate = "ERR_RETROACTIVE_GENERATION_DATE"
	CodeSyncOnlyLicense           = "ERR_SYNC_ONLY_LICENSE"
	CodeDuplicateAudioFile        = "ERR_DUPLICATE_AUDIO_FILE"
	CodeProviderNotAllowlisted    = "ERR_PROVIDER_NOT_ALLOWED"
)

// Settlement forfeiture reason for policy violations.
const ForfeitViolation = "FORFEITED_VIOLATION"

// GateError is a license-gate rejection with a stable machine code.
type GateError struct {
	Code    string
	Message string
}

func (e *GateError) Error() string { return e.Code + ": " + e.Message }

// StatusCode maps the gate failure to an HTTP status.
func (e *GateError) StatusCode() int {
	if e.Code == CodeDuplicateAudioFile {
		return http.StatusConflict
	}
	return http.StatusForbidden
}

var (
	// ErrDuplicateAudioFile is returned when the SHA-256 already exists.
	ErrDuplicateAudioFile = &GateError{Code: CodeDuplicateAudioFile, Message: "duplicate audio file"}
	// ErrNotFound marks missing rows.
	ErrNotFound = errors.New("not found")
	// ErrEmailTaken marks duplicate registration email.
	ErrEmailTaken = errors.New("email taken")
	// ErrHandleTaken marks a creator handle owned by someone else (S1).
	ErrHandleTaken = errors.New("handle taken")
	// ErrInvalidCredentials marks login failures.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrUserInactive marks disabled accounts.
	ErrUserInactive = errors.New("user inactive")
	// ErrForbidden marks authorization failures.
	ErrForbidden = errors.New("forbidden")
	// ErrNotStreamable marks tracks that are not READY for playback.
	ErrNotStreamable = errors.New("track not streamable")
	// ErrInvalidInput marks validation failures.
	ErrInvalidInput = errors.New("invalid input")
	// ErrPaymentUnavailable marks billing calls when no payment provider is configured.
	ErrPaymentUnavailable = errors.New("payment unavailable")
	// ErrTOTPRequired marks logins that need a second-factor code.
	ErrTOTPRequired = errors.New("totp required")
	// ErrTOTPInvalid marks wrong second-factor codes.
	ErrTOTPInvalid = errors.New("totp invalid")
	// ErrTooManyAttempts marks rate-limited credential attempts.
	ErrTooManyAttempts = errors.New("too many attempts")
	// ErrAgeConfirmRequired marks accounts missing the 14+ declaration.
	ErrAgeConfirmRequired = errors.New("age confirmation required")
	// ErrConsentRequired marks accounts missing required terms consent.
	ErrConsentRequired = errors.New("terms consent required")
	// ErrTOTPEnrollRequired blocks admins without a second factor (S-03).
	ErrTOTPEnrollRequired = errors.New("totp enrollment required")
	// ErrActivePassExists blocks buying a pass while one is active (B-08).
	ErrActivePassExists = errors.New("active pass exists")
	// ErrOAuthLinkLoginRequired marks a social login whose email matches an
	// existing account without provider-verified email (C-11): the user must
	// sign in with the existing account first, then link at /account/connections.
	ErrOAuthLinkLoginRequired = errors.New("oauth link requires existing-account login")
	// ErrOAuthLinkTaken marks a provider identity already linked to another user.
	ErrOAuthLinkTaken = errors.New("oauth identity already linked")
	// ErrOAuthLastMethod blocks unlinking the final login method (WI-A1).
	ErrOAuthLastMethod = errors.New("cannot unlink last login method")
	// ErrEmailNotVerified gates commerce surfaces (billing orders/payments,
	// upload pre-check/finalize, store registration, generations) until the
	// account email is verified (WI-A2). Streaming stays open.
	ErrEmailNotVerified = errors.New("email not verified")
	// ErrConflict marks a create that raced an existing row (unique
	// violation): the caller holds stale state and should re-read.
	ErrConflict = errors.New("conflict")
	// ErrQuotaExceeded marks an atomically enforced usage quota (download
	// log): the window filled between the advisory check and the write.
	ErrQuotaExceeded = errors.New("quota exceeded")
)
