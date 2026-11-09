package forensics

import (
	"math"
	"testing"
	"time"

	"avibe-platform/backend/internal/pkg/audio"
)

func TestChromaprint_EmptyBuffer(t *testing.T) {
	ce := NewChromaprintExtractor()
	_, err := ce.Extract(nil)
	if err != audio.ErrNilPCMBuffer {
		t.Fatalf("expected ErrNilPCMBuffer, got %v", err)
	}
}

func TestChromaprint_ExtractionUnder150ms(t *testing.T) {
	ce := NewChromaprintExtractor()

	sampleRate := 44100
	numSamples := sampleRate * 3 // 3 seconds of audio
	samples := make([]float64, numSamples)

	// Rich chord progression: 440Hz + 554.37Hz (C#) + 659.25Hz (E)
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		chord := 0.3*math.Sin(2*math.Pi*440.0*tSec) +
			0.3*math.Sin(2*math.Pi*554.37*tSec) +
			0.3*math.Sin(2*math.Pi*659.25*tSec)
		samples[i] = chord
	}

	buf := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	start := time.Now()
	res, err := ce.Extract(buf)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Verification check against Definition of Done: < 150ms
	if elapsed >= 150*time.Millisecond {
		t.Fatalf("DoD violated: fingerprint extraction took %v (limit: 150ms)", elapsed)
	}
	t.Logf("Fingerprint extracted in %v (DoD limit: < 150ms)", elapsed)

	if len(res.FingerprintB64) == 0 {
		t.Errorf("FingerprintB64 is empty")
	}
	if res.SubFingerprintN == 0 {
		t.Errorf("expected >0 subfingerprints, got 0")
	}

	jsonStr, err := res.ToJSON()
	if err != nil || len(jsonStr) == 0 {
		t.Fatalf("ToJSON failed: %v", err)
	}
}

func TestChromaprint_DeterministicAndRobust(t *testing.T) {
	ce := NewChromaprintExtractor()

	sampleRate := 48000
	numSamples := sampleRate * 2
	samples1 := make([]float64, numSamples)
	samples2 := make([]float64, numSamples)

	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		sig := 0.5 * math.Sin(2*math.Pi*500.0*tSec)
		samples1[i] = sig
		samples2[i] = sig + 0.001*math.Sin(2*math.Pi*1200.0*tSec) // Tiny perturbation
	}

	buf1 := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples1},
	}
	buf2 := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples2},
	}

	res1, _ := ce.Extract(buf1)
	res2, _ := ce.Extract(buf2)

	// Compare subfingerprint similarity
	n := len(res1.SubFingerprints)
	if n == 0 || n != len(res2.SubFingerprints) {
		t.Fatalf("subfingerprint length mismatch: %d vs %d", n, len(res2.SubFingerprints))
	}

	var totalBits, matchingBits int
	for i := 0; i < n; i++ {
		dist := ComputeHammingDistance(res1.SubFingerprints[i], res2.SubFingerprints[i])
		totalBits += 32
		matchingBits += (32 - dist)
	}

	similarityPct := (float64(matchingBits) / float64(totalBits)) * 100.0
	if similarityPct < 80.0 {
		t.Errorf("expected high similarity (>80%%) for slight perturbation, got %f%%", similarityPct)
	}
	t.Logf("Acoustic fingerprint match similarity: %f%%", similarityPct)
}
