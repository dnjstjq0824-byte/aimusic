package forensics

import (
	"math"
	"math/rand"
	"testing"

	"avibe-platform/backend/internal/pkg/audio"
)

func TestUdioDetector_CoherentAcousticTone(t *testing.T) {
	ud, err := NewUdioDetector()
	if err != nil {
		t.Fatalf("NewUdioDetector failed: %v", err)
	}

	sampleRate := 48000
	numSamples := sampleRate * 2 // 2 seconds
	samples := make([]float64, numSamples)

	// Clean, physically coherent vocal tone at 1200 Hz with harmonic at 2400 Hz
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		samples[i] = 0.4*math.Sin(2*math.Pi*1200.0*tSec) +
			0.3*math.Sin(2*math.Pi*2400.0*tSec)
	}

	buf := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	res, err := ud.Analyze(buf)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if res.IsUdioGenerated {
		t.Errorf("coherent tone falsely flagged as Udio: confidence %f", res.ConfidenceScore)
	}
	if res.ConfidenceScore > 0.30 {
		t.Errorf("expected low confidence score for coherent tone, got %f", res.ConfidenceScore)
	}
}

func TestUdioDetector_DiscontinuousPhaseArtifact(t *testing.T) {
	ud, err := NewUdioDetector()
	if err != nil {
		t.Fatalf("NewUdioDetector failed: %v", err)
	}

	sampleRate := 48000
	numSamples := sampleRate * 2
	samples := make([]float64, numSamples)

	// Synthesize audio with artificial random phase discontinuities every 10ms in the 1.5kHz-3kHz region
	r := rand.New(rand.NewSource(777))
	chunkLen := sampleRate / 100 // 10ms chunks

	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		chunkIdx := i / chunkLen
		randomPhase := float64(chunkIdx) * r.Float64() * math.Pi

		// Random phase shifts in 1.8kHz and 2.5kHz
		samples[i] = 0.4*math.Sin(2*math.Pi*1800.0*tSec+randomPhase) +
			0.4*math.Sin(2*math.Pi*2500.0*tSec+randomPhase)
	}

	buf := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	res, err := ud.Analyze(buf)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if !res.IsUdioGenerated {
		t.Errorf("expected Udio detection for phase discontinuous audio, but got false (confidence: %f, variance: %f)",
			res.ConfidenceScore, res.PhaseDiscontinuityScore)
	}
	if res.ConfidenceScore < 0.70 {
		t.Errorf("expected confidence >= 0.70, got %f", res.ConfidenceScore)
	}

	jsonStr, err := res.ToJSON()
	if err != nil || len(jsonStr) == 0 {
		t.Fatalf("ToJSON failed: %v", err)
	}
}
