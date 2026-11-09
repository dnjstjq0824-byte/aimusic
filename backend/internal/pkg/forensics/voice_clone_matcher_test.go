package forensics

import (
	"math"
	"testing"
	"time"

	"avibe-platform/backend/internal/pkg/audio"
)

func TestVoiceCloneMatcher_DetectionSpeedUnder3Seconds(t *testing.T) {
	matcher := NewVoiceCloneMatcher()

	sampleRate := 48000
	numSamples := sampleRate * 3 // 3 seconds
	samplesIU := make([]float64, numSamples)

	// Distinctive vocal formant spectrum for Artist IU (F1=600Hz, F2=1800Hz, F3=2700Hz)
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		samplesIU[i] = 0.4*math.Sin(2*math.Pi*600.0*tSec) +
			0.3*math.Sin(2*math.Pi*1800.0*tSec) +
			0.2*math.Sin(2*math.Pi*2700.0*tSec)
	}

	bufIU := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samplesIU},
	}

	// Extract and register reference voiceprint
	iuDVector, err := ExtractVoiceprint(bufIU)
	if err != nil {
		t.Fatalf("ExtractVoiceprint failed: %v", err)
	}

	matcher.RegisterArtist(ProtectedArtistVoiceprint{
		ArtistID:   "artist-iu",
		ArtistName: "IU (Lee Ji-eun)",
		DVector:    iuDVector,
	})

	// User uploads clone track mimicking the exact same vocal formants + subtle noise
	samplesClone := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		samplesClone[i] = samplesIU[i] + 0.005*math.Sin(2*math.Pi*120.0*float64(i)/float64(sampleRate))
	}
	bufClone := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samplesClone},
	}

	start := time.Now()
	res, err := matcher.MatchVoice(bufClone)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("MatchVoice failed: %v", err)
	}

	// Verification check against Definition of Done: < 3.0 seconds
	if elapsed >= 3*time.Second {
		t.Fatalf("DoD violated: voice clone detection took %v (limit: 3.0s)", elapsed)
	}
	t.Logf("Voice clone screening completed in %v (DoD limit: < 3.0s)", elapsed)

	if !res.IsImpersonationDetected {
		t.Errorf("expected impersonation detection, got false (similarity: %f)", res.SimilarityScore)
	}
	if res.MatchedArtistID != "artist-iu" {
		t.Errorf("expected matched artist 'artist-iu', got '%s'", res.MatchedArtistID)
	}
	if res.RecommendedAction != "FLAG_UNAUTHORIZED_VOICE_CLONE" {
		t.Errorf("expected action 'FLAG_UNAUTHORIZED_VOICE_CLONE', got '%s'", res.RecommendedAction)
	}

	jsonStr, err := res.ToJSON()
	if err != nil || len(jsonStr) == 0 {
		t.Fatalf("ToJSON failed: %v", err)
	}
}

func TestVoiceCloneMatcher_DifferentVoicePass(t *testing.T) {
	matcher := NewVoiceCloneMatcher()

	sampleRate := 48000
	numSamples := sampleRate * 2

	// Register low-baritone voice (F1=350Hz, F2=1100Hz)
	samplesBaritone := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		samplesBaritone[i] = 0.5*math.Sin(2*math.Pi*350.0*tSec) +
			0.3*math.Sin(2*math.Pi*1100.0*tSec)
	}
	bufBaritone := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samplesBaritone},
	}
	vpBaritone, _ := ExtractVoiceprint(bufBaritone)
	matcher.RegisterArtist(ProtectedArtistVoiceprint{
		ArtistID:   "artist-drake",
		ArtistName: "Drake",
		DVector:    vpBaritone,
	})

	// User uploads high soprano vocal track (F1=900Hz, F2=2800Hz)
	samplesSoprano := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		samplesSoprano[i] = 0.5*math.Sin(2*math.Pi*900.0*tSec) +
			0.4*math.Sin(2*math.Pi*2800.0*tSec)
	}
	bufSoprano := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samplesSoprano},
	}

	res, err := matcher.MatchVoice(bufSoprano)
	if err != nil {
		t.Fatalf("MatchVoice failed: %v", err)
	}

	if res.IsImpersonationDetected {
		t.Errorf("different voice was falsely flagged as clone: %f similarity", res.SimilarityScore)
	}
	if res.RecommendedAction != "PASS" {
		t.Errorf("expected action 'PASS', got '%s'", res.RecommendedAction)
	}
}
