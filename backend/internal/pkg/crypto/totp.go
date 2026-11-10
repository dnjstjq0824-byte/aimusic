package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" // #nosec G505 -- TOTP (RFC 6238) mandates SHA-1.
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// TOTP parameters per RFC 6238 (SHA-1, 30s step, 6 digits).
const (
	totpStepSeconds = 30
	totpDigits      = 6
	totpSkewSteps   = 1
)

// GenerateTOTPSecret returns a base32 secret (no padding) for enrollment.
func GenerateTOTPSecret() (string, error) {
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return strings.TrimRight(base32.StdEncoding.EncodeToString(raw), "="), nil
}

// TOTPProvisioningURI builds the otpauth:// URL authenticator apps consume.
func TOTPProvisioningURI(issuer, account, secret string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&digits=6&period=30",
		issuer, account, strings.ToUpper(secret), issuer)
}

// VerifyTOTP accepts the code for the current step plus/minus one step of skew.
// Recovery codes (avibe-xxxx-xxxx) never verify here; they are checked
// separately against bcrypt hashes in the users row.
func VerifyTOTP(secret, code string, now time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != totpDigits {
		return false
	}
	if _, err := strconv.Atoi(code); err != nil {
		return false
	}
	key, err := base32.StdEncoding.DecodeString(padBase32(secret))
	if err != nil || len(key) == 0 {
		return false
	}
	step := now.Unix() / totpStepSeconds
	for _, s := range []int64{step - totpSkewSteps, step, step + totpSkewSteps} {
		if s < 0 {
			continue
		}
		if hotp(key, uint64(s)) == code {
			return true
		}
	}
	return false
}

func padBase32(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if m := len(s) % 8; m != 0 {
		s += strings.Repeat("=", 8-m)
	}
	return s
}

func hotp(key []byte, counter uint64) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)
	mac := hmac.New(sha1.New, key) // #nosec G505 -- RFC 6238 TOTP.
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	bin := (uint32(sum[offset])&0x7f)<<24 |
		uint32(sum[offset+1])<<16 |
		uint32(sum[offset+2])<<8 |
		uint32(sum[offset+3])
	return fmt.Sprintf("%06d", bin%1_000_000)
}

// GenerateRecoveryCodes returns n printable single-use codes
// (avibe-XXXX-XXXX); only bcrypt hashes are stored.
func GenerateRecoveryCodes(n int) ([]string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	out := make([]string, 0, n)
	for len(out) < n {
		b := make([]byte, 8)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		var sb strings.Builder
		sb.WriteString("avibe-")
		for i, v := range b {
			if i == 4 {
				sb.WriteByte('-')
			}
			sb.WriteByte(alphabet[int(v)%len(alphabet)])
		}
		out = append(out, sb.String())
	}
	return out, nil
}
