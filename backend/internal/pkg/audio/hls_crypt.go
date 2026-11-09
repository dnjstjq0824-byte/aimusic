// Package audio provides RFC 8216 Section 5.2 compliant HLS segment encryption
// using AES-128-CBC with PKCS#7 padding, initialization vector derivation,
// and key rotation lifecycle management to protect audio streams from unauthorized extraction.
package audio

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrInvalidKeySize    = errors.New("AES-128 key must be exactly 16 bytes")
	ErrInvalidIVSize     = errors.New("AES-128 IV must be exactly 16 bytes")
	ErrCiphertextNotBlock = errors.New("ciphertext length must be a multiple of the AES block size (16 bytes)")
	ErrInvalidPKCS7Pad   = errors.New("invalid PKCS#7 padding in decrypted block")
	ErrKeyNotFound       = errors.New("cryptographic key not found in keystore")
)

const (
	AES128KeySize   = 16
	AES128BlockSize = 16
)

// DeriveSequenceIV converts an HLS media sequence number to a 16-byte big-endian IV (RFC 8216 Section 5.2).
func DeriveSequenceIV(seqNumber uint64) []byte {
	iv := make([]byte, AES128BlockSize)
	binary.BigEndian.PutUint64(iv[8:], seqNumber)
	return iv
}

// PKCS7Pad appends PKCS#7 padding up to the specified block size.
func PKCS7Pad(data []byte, blockSize int) []byte {
	padLen := blockSize - (len(data) % blockSize)
	if padLen == 0 {
		padLen = blockSize
	}
	padding := bytes.Repeat([]byte{byte(padLen)}, padLen)
	return append(data, padding...)
}

// PKCS7Unpad removes and validates PKCS#7 padding from plaintext.
func PKCS7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, ErrInvalidPKCS7Pad
	}
	padLen := int(data[len(data)-1])
	if padLen == 0 || padLen > blockSize || padLen > len(data) {
		return nil, ErrInvalidPKCS7Pad
	}
	for i := len(data) - padLen; i < len(data); i++ {
		if data[i] != byte(padLen) {
			return nil, ErrInvalidPKCS7Pad
		}
	}
	return data[:len(data)-padLen], nil
}

// EncryptHLSSegment encrypts raw segment data (e.g. .ts or fMP4 chunk) using AES-128-CBC with PKCS#7 padding.
func EncryptHLSSegment(plaintext, key, iv []byte) ([]byte, error) {
	if len(key) != AES128KeySize {
		return nil, ErrInvalidKeySize
	}
	if len(iv) != AES128BlockSize {
		return nil, ErrInvalidIVSize
	}

	padded := PKCS7Pad(plaintext, AES128BlockSize)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	ciphertext := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, padded)

	return ciphertext, nil
}

// DecryptHLSSegment decrypts AES-128-CBC encrypted segment bytes and strips PKCS#7 padding.
func DecryptHLSSegment(ciphertext, key, iv []byte) ([]byte, error) {
	if len(key) != AES128KeySize {
		return nil, ErrInvalidKeySize
	}
	if len(iv) != AES128BlockSize {
		return nil, ErrInvalidIVSize
	}
	if len(ciphertext) == 0 || len(ciphertext)%AES128BlockSize != 0 {
		return nil, ErrCiphertextNotBlock
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	decrypted := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(decrypted, ciphertext)

	plaintext, err := PKCS7Unpad(decrypted, AES128BlockSize)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// HLSKeyMetadata describes an active HLS encryption key in the rotation sequence.
type HLSKeyMetadata struct {
	KeyID      string    `json:"key_id"`
	KeyURI     string    `json:"key_uri"`
	KeyBytes   []byte    `json:"-"`
	IVHex      string    `json:"iv_hex,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// FormatEXTXKey generates standard RFC 8216 #EXT-X-KEY playlist directive.
func (km *HLSKeyMetadata) FormatEXTXKey() string {
	if km.IVHex != "" {
		return fmt.Sprintf(`#EXT-X-KEY:METHOD=AES-128,URI="%s",IV=0x%s`, km.KeyURI, km.IVHex)
	}
	return fmt.Sprintf(`#EXT-X-KEY:METHOD=AES-128,URI="%s"`, km.KeyURI)
}

// HLSKeyRotationManager manages dynamic cryptographic keys and rotation policies.
type HLSKeyRotationManager struct {
	mu           sync.RWMutex
	keys         map[string]*HLSKeyMetadata
	baseURI      string
	rotationSecs int
}

// NewHLSKeyRotationManager creates a rotation manager with a base key serving endpoint.
func NewHLSKeyRotationManager(baseURI string, rotationSecs int) *HLSKeyRotationManager {
	if rotationSecs <= 0 {
		rotationSecs = 3600 // 1 hour default rotation
	}
	return &HLSKeyRotationManager{
		keys:         make(map[string]*HLSKeyMetadata),
		baseURI:      baseURI,
		rotationSecs: rotationSecs,
	}
}

// GenerateNewKey creates and stores a new 128-bit key in the rotation keystore.
func (km *HLSKeyRotationManager) GenerateNewKey(keyID string, explicitIV []byte) (*HLSKeyMetadata, error) {
	keyBytes := make([]byte, AES128KeySize)
	if _, err := ioReadFullRand(keyBytes); err != nil {
		return nil, fmt.Errorf("crypto rand failed: %w", err)
	}

	var ivHex string
	if len(explicitIV) == AES128BlockSize {
		ivHex = hex.EncodeToString(explicitIV)
	}

	keyURI := fmt.Sprintf("%s/keys/%s", km.baseURI, keyID)
	now := time.Now().UTC()

	meta := &HLSKeyMetadata{
		KeyID:     keyID,
		KeyURI:    keyURI,
		KeyBytes:  keyBytes,
		IVHex:     ivHex,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Duration(km.rotationSecs) * time.Second),
	}

	km.mu.Lock()
	km.keys[keyID] = meta
	km.mu.Unlock()

	return meta, nil
}

// GetKey retrieves key bytes by ID.
func (km *HLSKeyRotationManager) GetKey(keyID string) ([]byte, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	meta, ok := km.keys[keyID]
	if !ok {
		return nil, ErrKeyNotFound
	}
	return meta.KeyBytes, nil
}

// ioReadFullRand wraps crypto/rand.Read.
func ioReadFullRand(b []byte) (int, error) {
	return rand.Read(b)
}
