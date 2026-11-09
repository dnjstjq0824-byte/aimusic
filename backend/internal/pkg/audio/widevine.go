// Package audio provides ISO/IEC 23001-7 Common Encryption (CENC) and Widevine Modular DRM
// implementations including PSSH (Protection System Specific Header) box generation,
// DASH ContentProtection descriptor XML construction, and AES-128-CTR / AES-128-CBC subsample encryption.
package audio

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidSystemID = errors.New("invalid DRM System ID")
	ErrInvalidKeyID    = errors.New("DRM Key ID must be 16 bytes")
	ErrInvalidPSSHBox  = errors.New("malformed or invalid PSSH box")
	ErrSubsampleRange  = errors.New("subsample clear + protected length exceeds sample size")
)

// WidevineSystemID is the canonical 16-byte UUID for Google Widevine DRM:
// edef8ba9-79d6-4ace-a3c8-27dcd51d21ed
var WidevineSystemID = []byte{
	0xed, 0xef, 0x8b, 0xa9, 0x79, 0xd6, 0x4a, 0xce,
	0xa3, 0xc8, 0x27, 0xdc, 0xd5, 0x1d, 0x21, 0xed,
}

// CENCProtectionScheme defines standard ISO/IEC 23001-7 four-character encryption schemes.
type CENCProtectionScheme string

const (
	SchemeCENC CENCProtectionScheme = "cenc" // AES-128 in Counter (CTR) mode
	SchemeCBCS CENCProtectionScheme = "cbcs" // AES-128 in CBC mode with pattern protection
)

// SubsampleEntry defines clear vs protected byte splits within an audio sample.
type SubsampleEntry struct {
	ClearBytes     uint16 `json:"clear_bytes"`
	ProtectedBytes uint32 `json:"protected_bytes"`
}

// WidevineConfig configures Widevine encryption parameters.
type WidevineConfig struct {
	Scheme   CENCProtectionScheme
	KeyID    []byte // 16-byte Key ID (KID)
	Key      []byte // 16-byte AES Content Key
	IV       []byte // 16-byte IV (or 8-byte for standard CENC)
	Provider string // Content provider identifier (e.g. "avibe")
}

// BuildPSSHBox generates an ISOBMFF 'pssh' box (ISO/IEC 23001-7 Section 8.1).
func BuildPSSHBox(systemID []byte, keyIDs [][]byte, privateData []byte) ([]byte, error) {
	if len(systemID) != 16 {
		return nil, ErrInvalidSystemID
	}

	version := byte(0)
	if len(keyIDs) > 0 {
		version = byte(1) // Version 1 includes explicit KIDs
	}

	payload := new(bytes.Buffer)
	// Version (1 byte) + Flags (3 bytes)
	payload.WriteByte(version)
	payload.Write([]byte{0x00, 0x00, 0x00})

	// SystemID (16 bytes)
	payload.Write(systemID)

	if version == 1 {
		// KID_count (4 bytes) + KIDs
		_ = binary.Write(payload, binary.BigEndian, uint32(len(keyIDs)))
		for _, kid := range keyIDs {
			if len(kid) != 16 {
				return nil, ErrInvalidKeyID
			}
			payload.Write(kid)
		}
	}

	// DataSize (4 bytes) + Data
	_ = binary.Write(payload, binary.BigEndian, uint32(len(privateData)))
	payload.Write(privateData)

	// Box Header: Size (4 bytes) + Type ('pssh' = 0x70737368)
	boxSize := uint32(payload.Len() + 8)
	boxBuf := new(bytes.Buffer)
	_ = binary.Write(boxBuf, binary.BigEndian, boxSize)
	boxBuf.WriteString("pssh")
	boxBuf.Write(payload.Bytes())

	return boxBuf.Bytes(), nil
}

// BuildWidevinePSSH builds a canonical Widevine version 1 PSSH box.
func BuildWidevinePSSH(keyID []byte, contentID string) ([]byte, error) {
	if len(keyID) != 16 {
		return nil, ErrInvalidKeyID
	}

	// Widevine private payload (simple TLV: 0x08 [tag] len [content_id bytes])
	var priv bytes.Buffer
	if contentID != "" {
		priv.WriteByte(0x12) // Field 2: content_id
		priv.WriteByte(byte(len(contentID)))
		priv.WriteString(contentID)
	}

	return BuildPSSHBox(WidevineSystemID, [][]byte{keyID}, priv.Bytes())
}

// FormatDASHContentProtection generates MPEG-DASH XML tags for MPD manifest embedding.
func FormatDASHContentProtection(keyID []byte, psshBox []byte, scheme CENCProtectionScheme) string {
	kidHex := hex.EncodeToString(keyID)
	// Format KID as standard UUID string: 8-4-4-4-12
	kidUUID := kidHex
	if len(kidHex) == 32 {
		kidUUID = fmt.Sprintf("%s-%s-%s-%s-%s",
			kidHex[0:8], kidHex[8:12], kidHex[12:16], kidHex[16:20], kidHex[20:32])
	}

	psshB64 := base64.StdEncoding.EncodeToString(psshBox)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`    <ContentProtection schemeIdUri="urn:mpeg:dash:mpd:cenc:2014" value="%s" cenc:default_KID="%s"/>`+"\n", scheme, kidUUID))
	sb.WriteString(`    <ContentProtection schemeIdUri="urn:uuid:edef8ba9-79d6-4ace-a3c8-27dcd51d21ed">` + "\n")
	sb.WriteString(fmt.Sprintf(`      <cenc:pssh>%s</cenc:pssh>`+"\n", psshB64))
	sb.WriteString(`    </ContentProtection>`)

	return sb.String()
}

// EncryptSubsampleCTR performs ISO/IEC 23001-7 AES-128-CTR subsample encryption.
// Clear header bytes remain unencrypted; payload bytes are encrypted with AES-CTR.
func EncryptSubsampleCTR(sampleData []byte, subsamples []SubsampleEntry, key, iv []byte) ([]byte, error) {
	if len(key) != 16 {
		return nil, ErrInvalidKeySize
	}
	if len(iv) != 16 {
		return nil, ErrInvalidIVSize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes init failed: %w", err)
	}

	// Make copy of sample data to avoid in-place mutation of caller's buffer
	out := make([]byte, len(sampleData))
	copy(out, sampleData)

	ctrIV := make([]byte, 16)
	copy(ctrIV, iv)
	stream := cipher.NewCTR(block, ctrIV)

	offset := 0
	for _, sub := range subsamples {
		clearEnd := offset + int(sub.ClearBytes)
		protEnd := clearEnd + int(sub.ProtectedBytes)

		if protEnd > len(sampleData) {
			return nil, ErrSubsampleRange
		}

		// Encrypt protected region
		if sub.ProtectedBytes > 0 {
			stream.XORKeyStream(out[clearEnd:protEnd], sampleData[clearEnd:protEnd])
		}

		offset = protEnd
	}

	// If unallocated tail remains without explicit subsample rule, encrypt remainder
	if offset < len(sampleData) {
		stream.XORKeyStream(out[offset:], sampleData[offset:])
	}

	return out, nil
}

// DecryptSubsampleCTR reverses AES-128-CTR subsample encryption (XOR operation with same stream).
func DecryptSubsampleCTR(sampleData []byte, subsamples []SubsampleEntry, key, iv []byte) ([]byte, error) {
	return EncryptSubsampleCTR(sampleData, subsamples, key, iv)
}
