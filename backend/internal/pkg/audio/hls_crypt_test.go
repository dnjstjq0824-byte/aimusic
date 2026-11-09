package audio

import (
	"bytes"
	"crypto/rand"
	"strings"
	"testing"
)

func TestPKCS7PadAndUnpad(t *testing.T) {
	data := []byte("Hello, HLS AES-128 Encryption!")
	padded := PKCS7Pad(data, 16)

	if len(padded)%16 != 0 {
		t.Fatalf("padded length %d is not a multiple of 16", len(padded))
	}

	unpadded, err := PKCS7Unpad(padded, 16)
	if err != nil {
		t.Fatalf("PKCS7Unpad failed: %v", err)
	}

	if !bytes.Equal(data, unpadded) {
		t.Fatalf("unpadded data does not match original: %s vs %s", string(unpadded), string(data))
	}

	// Corrupt padding byte
	corrupt := make([]byte, len(padded))
	copy(corrupt, padded)
	corrupt[len(corrupt)-1] = 99 // Invalid pad length > 16
	_, err = PKCS7Unpad(corrupt, 16)
	if err == nil {
		t.Fatalf("expected error on corrupt padding, got nil")
	}
}

func TestDeriveSequenceIV(t *testing.T) {
	seq := uint64(42)
	iv := DeriveSequenceIV(seq)

	if len(iv) != 16 {
		t.Fatalf("IV length must be 16, got %d", len(iv))
	}
	if iv[15] != 42 {
		t.Errorf("expected last byte to be 42, got %d", iv[15])
	}
	for i := 0; i < 8; i++ {
		if iv[i] != 0 {
			t.Errorf("expected high 8 bytes to be 0, got %d at idx %d", iv[i], i)
		}
	}
}

func TestEncryptAndDecryptHLSSegment_RoundTrip(t *testing.T) {
	// Mock 188-byte MPEG-TS packet stream (10 packets = 1880 bytes)
	mockSegment := make([]byte, 1880)
	for i := 0; i < 1880; i += 188 {
		mockSegment[i] = 0x47 // MPEG-TS Sync byte
	}
	_, _ = rand.Read(mockSegment[1:])

	key := make([]byte, 16)
	_, _ = rand.Read(key)

	iv := DeriveSequenceIV(105)

	ciphertext, err := EncryptHLSSegment(mockSegment, key, iv)
	if err != nil {
		t.Fatalf("EncryptHLSSegment failed: %v", err)
	}

	// Ciphertext should be padded and strictly longer than plaintext
	if len(ciphertext) <= len(mockSegment) {
		t.Fatalf("ciphertext length %d should be greater than plaintext %d", len(ciphertext), len(mockSegment))
	}
	if len(ciphertext)%16 != 0 {
		t.Fatalf("ciphertext length %d is not a multiple of 16", len(ciphertext))
	}

	decrypted, err := DecryptHLSSegment(ciphertext, key, iv)
	if err != nil {
		t.Fatalf("DecryptHLSSegment failed: %v", err)
	}

	if !bytes.Equal(mockSegment, decrypted) {
		t.Fatalf("roundtrip decrypted segment does not match original plaintext!")
	}
}

func TestHLSKeyRotationManager(t *testing.T) {
	mgr := NewHLSKeyRotationManager("https://aimusic.dnjstjq0824.workers.dev/api/v1/drm", 3600)

	keyID := "key-track-001-seq-0"
	explicitIV := DeriveSequenceIV(0)

	meta, err := mgr.GenerateNewKey(keyID, explicitIV)
	if err != nil {
		t.Fatalf("GenerateNewKey failed: %v", err)
	}

	if len(meta.KeyBytes) != 16 {
		t.Fatalf("generated key bytes length %d != 16", len(meta.KeyBytes))
	}

	tagWithIV := meta.FormatEXTXKey()
	if !strings.Contains(tagWithIV, `#EXT-X-KEY:METHOD=AES-128,URI="https://aimusic.dnjstjq0824.workers.dev/api/v1/drm/keys/key-track-001-seq-0"`) {
		t.Errorf("unexpected tag: %s", tagWithIV)
	}
	if !strings.Contains(tagWithIV, "IV=0x") {
		t.Errorf("tag should contain IV: %s", tagWithIV)
	}

	// Fetch key back
	fetchedKey, err := mgr.GetKey(keyID)
	if err != nil {
		t.Fatalf("GetKey failed: %v", err)
	}
	if !bytes.Equal(meta.KeyBytes, fetchedKey) {
		t.Fatalf("retrieved key does not match generated key")
	}

	// Fetch nonexistent key
	_, err = mgr.GetKey("nonexistent-key")
	if err != ErrKeyNotFound {
		t.Fatalf("expected ErrKeyNotFound, got %v", err)
	}
}
