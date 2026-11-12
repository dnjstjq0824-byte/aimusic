// Package crypto provides AES-256-GCM encryption and HLS signed-URL tokens.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"time"
)

// Cipher is an AES-256-GCM encryptor for settlement account numbers.
type Cipher struct {
	gcm cipher.AEAD
}

// NewCipher builds a Cipher from a 32-byte key.
func NewCipher(key []byte) (*Cipher, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("aes-256 key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{gcm: gcm}, nil
}

// Encrypt seals plaintext and returns base64(nonce || ciphertext).
func (c *Cipher) Encrypt(plaintext []byte) (string, error) {
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := c.gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt opens a value produced by Encrypt.
func (c *Cipher) Decrypt(enc string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}
	nonceSize := c.gcm.NonceSize()
	if len(raw) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := raw[:nonceSize], raw[nonceSize:]
	plain, err := c.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plain, nil
}

// Playback grant quality tiers. The tier is part of the signed message, so
// a standard-quality grant cannot be replayed against the 256k variant
// (B-04). The edge Worker (worker.js) enforces which files each tier may
// fetch: SQ = stream_0 playlist + seg_0_* segments + peaks.json, HQ = all.
const (
	PlaybackSQ = "SQ"
	PlaybackHQ = "HQ"
)

// SignPlaybackToken returns the HMAC-SHA256 hex signature over
// "trackId|jti|exp|quality".
func SignPlaybackToken(trackID, jti string, expUnix int64, quality, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	msg := trackID + "|" + jti + "|" + strconv.FormatInt(expUnix, 10) + "|" + quality
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyPlaybackToken checks signature, tier and expiry of a playback token.
func VerifyPlaybackToken(trackID, jti string, expUnix int64, quality, sig, key string) bool {
	if key == "" || sig == "" || (quality != PlaybackSQ && quality != PlaybackHQ) {
		return false
	}
	if time.Now().Unix() > expUnix {
		return false
	}
	expected := SignPlaybackToken(trackID, jti, expUnix, quality, key)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(sig)) == 1
}

// BuildSignedPlaybackURL appends exp, jti, q (quality tier) and sig query
// parameters to a base HLS URL.
func BuildSignedPlaybackURL(baseURL, trackID, jti string, exp time.Time, quality, key string) string {
	expUnix := exp.Unix()
	sig := SignPlaybackToken(trackID, jti, expUnix, quality, key)
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}
	q := u.Query()
	q.Set("exp", strconv.FormatInt(expUnix, 10))
	q.Set("jti", jti)
	q.Set("q", quality)
	q.Set("sig", sig)
	u.RawQuery = q.Encode()
	return u.String()
}

// SHA256Hex returns the lowercase hex SHA-256 of data.
func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
