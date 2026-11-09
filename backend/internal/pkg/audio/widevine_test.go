package audio

import (
	"bytes"
	"crypto/rand"
	"strings"
	"testing"
)

func TestBuildWidevinePSSH(t *testing.T) {
	keyID := make([]byte, 16)
	_, _ = rand.Read(keyID)

	contentID := "avibe-stream-track-001"
	psshBox, err := BuildWidevinePSSH(keyID, contentID)
	if err != nil {
		t.Fatalf("BuildWidevinePSSH failed: %v", err)
	}

	if len(psshBox) < 32 {
		t.Fatalf("pssh box too short: %d", len(psshBox))
	}

	// Box type 'pssh' at index 4..8
	boxType := string(psshBox[4:8])
	if boxType != "pssh" {
		t.Errorf("expected box type 'pssh', got '%s'", boxType)
	}

	// Version 1 at index 8
	version := psshBox[8]
	if version != 1 {
		t.Errorf("expected version 1, got %d", version)
	}

	// Widevine SystemID at index 12..28
	sysID := psshBox[12:28]
	if !bytes.Equal(sysID, WidevineSystemID) {
		t.Errorf("system ID does not match Widevine System ID")
	}

	// KeyID present in payload
	if !bytes.Contains(psshBox, keyID) {
		t.Errorf("KeyID not found in PSSH box")
	}
}

func TestFormatDASHContentProtection(t *testing.T) {
	keyID := make([]byte, 16)
	for i := range keyID {
		keyID[i] = byte(i)
	}

	psshBox, err := BuildWidevinePSSH(keyID, "track-sample")
	if err != nil {
		t.Fatalf("BuildWidevinePSSH failed: %v", err)
	}

	xml := FormatDASHContentProtection(keyID, psshBox, SchemeCENC)

	if !strings.Contains(xml, `value="cenc"`) {
		t.Errorf("missing scheme value in xml: %s", xml)
	}
	if !strings.Contains(xml, "urn:uuid:edef8ba9-79d6-4ace-a3c8-27dcd51d21ed") {
		t.Errorf("missing Widevine UUID in xml: %s", xml)
	}
	if !strings.Contains(xml, "<cenc:pssh>") {
		t.Errorf("missing cenc:pssh element in xml: %s", xml)
	}
}

func TestEncryptAndDecryptSubsampleCTR(t *testing.T) {
	// Frame: 7 bytes clear header + 128 bytes audio payload = 135 bytes
	frame := make([]byte, 135)
	clearHeader := []byte{0xFF, 0xF1, 0x50, 0x80, 0x01, 0x1F, 0xFC} // Mock ADTS AAC header
	copy(frame[:7], clearHeader)
	_, _ = rand.Read(frame[7:])

	key := make([]byte, 16)
	iv := make([]byte, 16)
	_, _ = rand.Read(key)
	_, _ = rand.Read(iv)

	subsamples := []SubsampleEntry{
		{
			ClearBytes:     7,
			ProtectedBytes: 128,
		},
	}

	encrypted, err := EncryptSubsampleCTR(frame, subsamples, key, iv)
	if err != nil {
		t.Fatalf("EncryptSubsampleCTR failed: %v", err)
	}

	// Clear header MUST remain untouched
	if !bytes.Equal(encrypted[:7], clearHeader) {
		t.Fatalf("clear header was modified during encryption!")
	}

	// Protected audio payload MUST be encrypted
	if bytes.Equal(encrypted[7:], frame[7:]) {
		t.Fatalf("protected payload was not encrypted!")
	}

	// Decrypt
	decrypted, err := DecryptSubsampleCTR(encrypted, subsamples, key, iv)
	if err != nil {
		t.Fatalf("DecryptSubsampleCTR failed: %v", err)
	}

	if !bytes.Equal(frame, decrypted) {
		t.Fatalf("roundtrip decrypted sample does not match original frame!")
	}
}

func TestSubsampleRangeValidation(t *testing.T) {
	frame := make([]byte, 50)
	key := make([]byte, 16)
	iv := make([]byte, 16)

	// Subsample exceeds frame length (30 + 30 = 60 > 50)
	subsamples := []SubsampleEntry{
		{
			ClearBytes:     30,
			ProtectedBytes: 30,
		},
	}

	_, err := EncryptSubsampleCTR(frame, subsamples, key, iv)
	if err != ErrSubsampleRange {
		t.Fatalf("expected ErrSubsampleRange, got %v", err)
	}
}
