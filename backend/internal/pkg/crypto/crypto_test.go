package crypto

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func testKey() []byte {
	return []byte("0123456789abcdef0123456789abcdef")
}

func TestNewCipherKeyLength(t *testing.T) {
	if _, err := NewCipher(testKey()); err != nil {
		t.Fatalf("32-byte key rejected: %v", err)
	}
	if _, err := NewCipher([]byte("short")); err == nil {
		t.Fatal("short key accepted")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	c, err := NewCipher(testKey())
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte("123-456-789012")
	enc, err := c.Encrypt(plain)
	if err != nil {
		t.Fatal(err)
	}
	if enc == "" || strings.Contains(enc, string(plain)) {
		t.Fatalf("ciphertext leaks plaintext: %q", enc)
	}
	opened, err := c.Decrypt(enc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(opened, plain) {
		t.Fatalf("round trip mismatch: got %q want %q", opened, plain)
	}
	// Fresh nonce per encryption: same input must not repeat output.
	enc2, err := c.Encrypt(plain)
	if err != nil {
		t.Fatal(err)
	}
	if enc == enc2 {
		t.Fatal("deterministic ciphertext: nonce reuse suspected")
	}
}

func TestDecryptRejectsTampered(t *testing.T) {
	c, err := NewCipher(testKey())
	if err != nil {
		t.Fatal(err)
	}
	enc, err := c.Encrypt([]byte("account-1"))
	if err != nil {
		t.Fatal(err)
	}
	tampered := enc[:len(enc)-2] + "AA"
	if _, err := c.Decrypt(tampered); err == nil {
		t.Fatal("tampered ciphertext accepted")
	}
	if _, err := c.Decrypt("not-base64!!"); err == nil {
		t.Fatal("invalid base64 accepted")
	}
	other, err := NewCipher([]byte("abcdef0123456789abcdef0123456789"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.Decrypt(enc); err == nil {
		t.Fatal("wrong-key ciphertext accepted")
	}
}

func TestPlaybackTokenSignVerify(t *testing.T) {
	exp := time.Now().Add(time.Minute).Unix()
	sig := SignPlaybackToken("track-1", "jti-1", exp, PlaybackSQ, "secret")
	if !VerifyPlaybackToken("track-1", "jti-1", exp, PlaybackSQ, sig, "secret") {
		t.Fatal("valid token rejected")
	}
	if VerifyPlaybackToken("track-1", "jti-1", exp, PlaybackSQ, sig, "wrong") {
		t.Fatal("wrong-key token accepted")
	}
	past := time.Now().Add(-time.Minute).Unix()
	old := SignPlaybackToken("track-1", "jti-1", past, PlaybackSQ, "secret")
	if VerifyPlaybackToken("track-1", "jti-1", past, PlaybackSQ, old, "secret") {
		t.Fatal("expired token accepted")
	}
}

func TestPlaybackTokenBindsQuality(t *testing.T) {
	exp := time.Now().Add(time.Minute).Unix()
	sq := SignPlaybackToken("track-1", "jti-1", exp, PlaybackSQ, "secret")
	if VerifyPlaybackToken("track-1", "jti-1", exp, PlaybackHQ, sq, "secret") {
		t.Fatal("an SQ signature must not verify as HQ")
	}
	if VerifyPlaybackToken("track-1", "jti-1", exp, "", sq, "secret") {
		t.Fatal("missing quality must not verify")
	}
	u := BuildSignedPlaybackURL("https://cdn.example/hls/t/stream_0.m3u8", "track-1", "jti-1", time.Unix(exp, 0), PlaybackSQ, "secret")
	if !strings.Contains(u, "q=SQ") || !strings.Contains(u, "sig="+sq) {
		t.Fatalf("url = %s", u)
	}
}
