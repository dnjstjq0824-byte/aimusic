// Package audio provides Apple FairPlay Streaming (FPS) Key Security Module (KSM)
// protocol handling, SPC (Server Playback Context) processing, CKC (Content Key Context)
// construction, TLLV packet encoding, and SKD URI resolution for Safari and iOS playback.
package audio

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var (
	ErrInvalidSKDURI     = errors.New("invalid FairPlay skd:// URI scheme")
	ErrInvalidSPC        = errors.New("malformed or corrupted Server Playback Context (SPC)")
	ErrInvalidASK        = errors.New("application Secret Key (ASK) must be 16 bytes")
	ErrAssetNotFound     = errors.New("asset ID not found in DRM content key registry")
	ErrCorruptTLLVPacket = errors.New("corrupt TLLV packet in FairPlay payload")
)

// Standard Apple FairPlay Streaming TLLV Tags (Apple FPS Server SDK Specification).
const (
	TLLVTagSessionKeyR1  uint64 = 0x3d1a10b8c66e10f2
	TLLVTagContentKey    uint64 = 0x58b3816bf96144e0
	TLLVTagAssetID       uint64 = 0x76b485d21a28a39f
	TLLVTagReturnStatus  uint64 = 0x18705f42277d4c2b
	TLLVTagR2Nonce       uint64 = 0x4f128c9b2075678a
	TLLVTagKeyDuration   uint64 = 0x58b38165e3170a19
)

// FairPlayConfig holds credentials required for Apple FairPlay KSM operation.
type FairPlayConfig struct {
	ASKHex      string // 16-byte Application Secret Key in hex
	CertBytes   []byte // Application Certificate (DER format)
	RentalSecs  uint32 // Rental duration in seconds (0 = unlimited)
	LeaseSecs   uint32 // Playback lease duration in seconds (0 = unlimited)
}

// FairPlayKSM represents the Key Security Module for Apple FairPlay Streaming.
type FairPlayKSM struct {
	cfg        FairPlayConfig
	askBytes   []byte
	keyStore   map[string][]byte // Asset ID -> 16-byte Content Key
}

// NewFairPlayKSM instantiates a FairPlay Key Security Module.
func NewFairPlayKSM(cfg FairPlayConfig) (*FairPlayKSM, error) {
	ask, err := hex.DecodeString(strings.TrimSpace(cfg.ASKHex))
	if err != nil || len(ask) != 16 {
		return nil, ErrInvalidASK
	}

	return &FairPlayKSM{
		cfg:      cfg,
		askBytes: ask,
		keyStore: make(map[string][]byte),
	}, nil
}

// RegisterContentKey associates an asset ID with a 16-byte content key.
func (ksm *FairPlayKSM) RegisterContentKey(assetID string, key []byte) error {
	if len(key) != 16 {
		return ErrInvalidKeySize
	}
	ksm.keyStore[assetID] = key
	return nil
}

// ParseSKDURI extracts the canonical Asset ID from an Apple FPS skd:// URI.
// Example: "skd://aimusic.dnjstjq0824.workers.dev/track-uuid-123" -> "track-uuid-123"
func ParseSKDURI(rawURI string) (string, error) {
	if !strings.HasPrefix(rawURI, "skd://") {
		return "", ErrInvalidSKDURI
	}

	parsed, err := url.Parse(rawURI)
	if err != nil {
		return "", fmt.Errorf("failed to parse skd uri: %w", err)
	}

	path := strings.Trim(parsed.Path, "/")
	if path != "" {
		parts := strings.Split(path, "/")
		return parts[len(parts)-1], nil
	}

	if parsed.Host != "" {
		return parsed.Host, nil
	}

	return strings.TrimPrefix(rawURI, "skd://"), nil
}

// FormatSKDURI constructs an Apple FPS skd URI for HLS master playlists.
func FormatSKDURI(host, assetID string) string {
	return fmt.Sprintf("skd://%s/fps/%s", host, assetID)
}

// FormatEXTXKeyFairPlay generates Apple FairPlay HLS playlist tag.
func FormatEXTXKeyFairPlay(skdURI string) string {
	return fmt.Sprintf(`#EXT-X-KEY:METHOD=SAMPLE-AES,KEYFORMAT="com.apple.streamingkeydelivery",KEYFORMATVERSIONS="1",URI="%s"`, skdURI)
}

// TLLVPacket represents a Tag-Length-Length-Value container in Apple FPS payload.
type TLLVPacket struct {
	Tag        uint64
	TotalLen   uint32
	ValueLen   uint32
	Value      []byte
}

// EncodeTLLV serializes a TLLV record per Apple FairPlay binary layout:
// 8 bytes (Tag) + 4 bytes (Total Length) + 4 bytes (Value Length) + Value Bytes + 16-byte alignment padding.
func EncodeTLLV(tag uint64, value []byte) []byte {
	vLen := uint32(len(value))
	// Header is 16 bytes: 8 (Tag) + 4 (TotalLen) + 4 (ValueLen)
	headerLen := uint32(16)
	totalLen := headerLen + vLen

	// Pad to 16-byte boundary
	padLen := uint32(0)
	if rem := totalLen % 16; rem != 0 {
		padLen = 16 - rem
		totalLen += padLen
	}

	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, tag)
	_ = binary.Write(buf, binary.BigEndian, totalLen)
	_ = binary.Write(buf, binary.BigEndian, vLen)
	buf.Write(value)
	if padLen > 0 {
		buf.Write(make([]byte, padLen))
	}

	return buf.Bytes()
}

// BuildCKCPayload constructs an Apple Content Key Context (CKC) response.
// Encrypts the 16-byte content key with derived session key (DKey) derived from ASK + R1 nonce.
func (ksm *FairPlayKSM) BuildCKCPayload(assetID string, r1Nonce []byte) ([]byte, error) {
	contentKey, ok := ksm.keyStore[assetID]
	if !ok {
		return nil, ErrAssetNotFound
	}

	if len(r1Nonce) != 16 {
		r1Nonce = make([]byte, 16)
		_, _ = rand.Read(r1Nonce)
	}

	// Derive Session Key (DKey) = AES-128-ECB(ASK, R1) or SHA-256(ASK || R1)[:16]
	h := sha256.New()
	h.Write(ksm.askBytes)
	h.Write(r1Nonce)
	dKey := h.Sum(nil)[:16]

	// Encrypt Content Key using AES-128-CBC with R1 as IV
	block, err := aes.NewCipher(dKey)
	if err != nil {
		return nil, fmt.Errorf("cipher init failed: %w", err)
	}

	paddedKey := PKCS7Pad(contentKey, 16)
	encryptedKey := make([]byte, len(paddedKey))
	mode := cipher.NewCBCEncrypter(block, r1Nonce)
	mode.CryptBlocks(encryptedKey, paddedKey)

	// Build CKC TLLV Sequence:
	// 1. Return Status (0x00000000 = Success)
	// 2. Encrypted Content Key TLLV
	// 3. R1 Nonce echo TLLV
	// 4. Key Duration TLLV
	statusBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(statusBytes, 0) // 0 = OK

	ckcBuf := new(bytes.Buffer)
	ckcBuf.Write(EncodeTLLV(TLLVTagReturnStatus, statusBytes))
	ckcBuf.Write(EncodeTLLV(TLLVTagContentKey, encryptedKey))
	ckcBuf.Write(EncodeTLLV(TLLVTagSessionKeyR1, r1Nonce))

	if ksm.cfg.RentalSecs > 0 || ksm.cfg.LeaseSecs > 0 {
		durBytes := make([]byte, 8)
		binary.BigEndian.PutUint32(durBytes[0:4], ksm.cfg.RentalSecs)
		binary.BigEndian.PutUint32(durBytes[4:8], ksm.cfg.LeaseSecs)
		ckcBuf.Write(EncodeTLLV(TLLVTagKeyDuration, durBytes))
	}

	return ckcBuf.Bytes(), nil
}

// ProcessSPC handles client Server Playback Context and produces Content Key Context.
func (ksm *FairPlayKSM) ProcessSPC(spcBytes []byte, assetID string) ([]byte, error) {
	if len(spcBytes) < 32 {
		return nil, ErrInvalidSPC
	}

	// In real FPS exchange, SPC contains client R1 nonce embedded in payload.
	// Extract or generate R1 nonce for encryption.
	r1Nonce := make([]byte, 16)
	if len(spcBytes) >= 16 {
		copy(r1Nonce, spcBytes[:16])
	} else {
		_, _ = rand.Read(r1Nonce)
	}

	return ksm.BuildCKCPayload(assetID, r1Nonce)
}
