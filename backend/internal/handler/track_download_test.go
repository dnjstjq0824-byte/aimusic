package handler

import (
	"testing"

	"avibe-platform/backend/internal/domain"
)

func TestDetectDownloadFormat(t *testing.T) {
	wav := []byte("RIFF\x00\x00\x00\x00WAVEfmt ")
	if got := detectDownloadFormat(wav); got != "wav" {
		t.Fatalf("wav = %q", got)
	}
	if got := detectDownloadFormat([]byte("ID3\x04\x00\x00\x00")); got != "mp3" {
		t.Fatalf("id3 = %q", got)
	}
	if got := detectDownloadFormat([]byte{0xFF, 0xFB, 0x90, 0x00}); got != "mp3" {
		t.Fatalf("frame sync = %q", got)
	}
	for _, raw := range [][]byte{nil, {}, []byte("RIFF"), []byte("OggS........")} {
		if got := detectDownloadFormat(raw); got != "" {
			t.Fatalf("%q = %q, want empty", raw, got)
		}
	}
}

func TestDownloadFilename(t *testing.T) {
	got := downloadFilename("abcdef12-3456-7890-abcd-ef1234567890", "Neon Booster (2026 Remaster)!", "wav")
	want := "AVIBE-abcdef12-neon-booster-2026-remaster.wav"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	got = downloadFilename("short", "!!!", "mp3")
	if got != "AVIBE-short-track.mp3" {
		t.Fatalf("fallback = %q", got)
	}
}

func TestDownloadDisclosureText(t *testing.T) {
	track := &domain.Track{ID: "tid-1", AIProvider: "SUNO"}
	got := downloadDisclosureText(track, "https://avibemusic.com/")
	if got != "AI-generated music via SUNO. Verify: https://avibemusic.com/tracks/tid-1" {
		t.Fatalf("got %q", got)
	}
	got = downloadDisclosureText(track, "")
	if got != "AI-generated music via SUNO. Verify: https://avibemusic.com/tracks/tid-1" {
		t.Fatalf("default base = %q", got)
	}
}
