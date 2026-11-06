package audio

import (
	"strings"
	"testing"
)

func TestCheckBlockedMetadata(t *testing.T) {
	tags := map[string]string{
		"title":   "My AI Song",
		"comment": "Generated with Free Trial Tier",
	}
	kw, blocked := CheckBlockedMetadata(tags, nil)
	if !blocked {
		t.Fatal("expected Free Trial tag to be blocked by DefaultBlockedMetadataKeywords")
	}
	if kw == "" {
		t.Fatal("expected matched keyword")
	}

	cleanTags := map[string]string{
		"title":   "Commercial Track",
		"comment": AIWatermarkComment,
	}
	if kw2, blocked2 := CheckBlockedMetadata(cleanTags, []string{"cover", "remix"}); blocked2 {
		t.Fatalf("unexpected block on clean tags: %q", kw2)
	}
}

func TestCheckBitrate(t *testing.T) {
	if err := CheckBitrate(".wav", "pcm_s16le", 0, 192); err != nil {
		t.Fatalf("lossless wav should bypass bitrate check: %v", err)
	}
	if err := CheckBitrate(".mp3", "mp3", 128, 192); err == nil {
		t.Fatal("128kbps mp3 should fail 192kbps min check")
	}
	if err := CheckBitrate(".mp3", "mp3", 256, 192); err != nil {
		t.Fatalf("256kbps mp3 should pass: %v", err)
	}
}

func TestBuildHLSArgsContainsWatermark(t *testing.T) {
	args := BuildHLSArgs("in.wav", "out", AIWatermarkComment, "SUNO", "Pro")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "comment="+AIWatermarkComment) {
		t.Fatalf("missing watermark comment in args: %s", joined)
	}
	if !strings.Contains(joined, "ai_provider=SUNO") || !strings.Contains(joined, "ai_tier=Pro") {
		t.Fatalf("missing ai_provider/ai_tier metadata in args: %s", joined)
	}
}

func TestComputeAudioFingerprintAndReport(t *testing.T) {
	fp := ComputeAudioFingerprint([]float64{0.1, 0.5, 0.9, 0.3})
	if !strings.HasPrefix(fp, "fp_") {
		t.Fatalf("unexpected fingerprint: %q", fp)
	}
	rep := FormatFFprobeReportJSON(&ProbeResult{
		FormatName:  "wav",
		DurationSec: 42.5,
		BitrateKbps: 1411,
		CodecName:   "pcm_s16le",
		SampleRate:  44100,
		Channels:    2,
	}, true, "")
	if !strings.Contains(rep, `"gate3Passed":true`) {
		t.Fatalf("unexpected report JSON: %s", rep)
	}
}
