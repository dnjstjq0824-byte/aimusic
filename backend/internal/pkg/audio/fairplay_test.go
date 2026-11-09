package audio

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"strings"
	"testing"
)

func TestParseSKDURI(t *testing.T) {
	cases := []struct {
		input    string
		expected string
		isErr    bool
	}{
		{"skd://aimusic.dnjstjq0824.workers.dev/fps/track-uuid-999", "track-uuid-999", false},
		{"skd://track-uuid-888", "track-uuid-888", false},
		{"https://aimusic.dnjstjq0824.workers.dev/fps/track", "", true},
		{"invalid-uri", "", true},
	}

	for _, c := range cases {
		assetID, err := ParseSKDURI(c.input)
		if c.isErr && err == nil {
			t.Errorf("expected error for input %s, got nil", c.input)
		}
		if !c.isErr && err != nil {
			t.Errorf("unexpected error for input %s: %v", c.input, err)
		}
		if !c.isErr && assetID != c.expected {
			t.Errorf("expected assetID %s, got %s", c.expected, assetID)
		}
	}
}

func TestFormatSKDURI_And_EXTXKey(t *testing.T) {
	skdURI := FormatSKDURI("aimusic.dnjstjq0824.workers.dev", "audio-track-001")
	if skdURI != "skd://aimusic.dnjstjq0824.workers.dev/fps/audio-track-001" {
		t.Fatalf("unexpected skdURI: %s", skdURI)
	}

	tag := FormatEXTXKeyFairPlay(skdURI)
	if !strings.Contains(tag, `KEYFORMAT="com.apple.streamingkeydelivery"`) {
		t.Errorf("missing keyformat in tag: %s", tag)
	}
	if !strings.Contains(tag, skdURI) {
		t.Errorf("missing skdURI in tag: %s", tag)
	}
}

func TestEncodeTLLV(t *testing.T) {
	tag := uint64(0x1234567890ABCDEF)
	val := []byte("FairPlayTLLVTestValue") // 21 bytes

	pkt := EncodeTLLV(tag, val)

	// Header is 16 bytes. Total content is 16 + 21 = 37 bytes.
	// 37 bytes padded to multiple of 16 is 48 bytes.
	if len(pkt)%16 != 0 {
		t.Fatalf("TLLV packet len %d is not a multiple of 16", len(pkt))
	}
	if len(pkt) != 48 {
		t.Fatalf("expected 48 bytes, got %d", len(pkt))
	}

	// Verify Header fields
	parsedTag := binary.BigEndian.Uint64(pkt[0:8])
	totalLen := binary.BigEndian.Uint32(pkt[8:12])
	valLen := binary.BigEndian.Uint32(pkt[12:16])

	if parsedTag != tag {
		t.Errorf("tag mismatch: %x vs %x", parsedTag, tag)
	}
	if totalLen != 48 {
		t.Errorf("totalLen mismatch: expected 48, got %d", totalLen)
	}
	if valLen != 21 {
		t.Errorf("valLen mismatch: expected 21, got %d", valLen)
	}
	if !bytes.Equal(pkt[16:16+valLen], val) {
		t.Errorf("value mismatch")
	}
}

func TestFairPlayKSM_ProcessSPC(t *testing.T) {
	askHex := "0123456789abcdef0123456789abcdef"
	ksm, err := NewFairPlayKSM(FairPlayConfig{
		ASKHex:     askHex,
		RentalSecs: 86400,
		LeaseSecs:  3600,
	})
	if err != nil {
		t.Fatalf("NewFairPlayKSM failed: %v", err)
	}

	// Register test content key
	assetID := "test-asset-123"
	contentKey := make([]byte, 16)
	_, _ = rand.Read(contentKey)
	_ = ksm.RegisterContentKey(assetID, contentKey)

	// Mock SPC payload (at least 32 bytes)
	mockSPC := make([]byte, 64)
	_, _ = rand.Read(mockSPC)

	ckc, err := ksm.ProcessSPC(mockSPC, assetID)
	if err != nil {
		t.Fatalf("ProcessSPC failed: %v", err)
	}

	if len(ckc) == 0 {
		t.Fatalf("CKC payload is empty")
	}
	if len(ckc)%16 != 0 {
		t.Fatalf("CKC payload length %d is not aligned to 16 bytes", len(ckc))
	}

	// Verify asset not found error
	_, err = ksm.ProcessSPC(mockSPC, "unknown-asset")
	if err != ErrAssetNotFound {
		t.Fatalf("expected ErrAssetNotFound, got %v", err)
	}

	// Invalid short SPC
	_, err = ksm.ProcessSPC([]byte{1, 2, 3}, assetID)
	if err != ErrInvalidSPC {
		t.Fatalf("expected ErrInvalidSPC, got %v", err)
	}
}

func TestFairPlayKSM_InvalidASK(t *testing.T) {
	_, err := NewFairPlayKSM(FairPlayConfig{
		ASKHex: "tooshort",
	})
	if err != ErrInvalidASK {
		t.Fatalf("expected ErrInvalidASK, got %v", err)
	}
}
