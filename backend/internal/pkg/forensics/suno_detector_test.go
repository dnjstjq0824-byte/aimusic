package forensics

import (
	"math"
	"testing"

	"avibe-platform/backend/internal/pkg/audio"
)

func TestSunoDetector_CleanAcousticRecording(t *testing.T) {
	sd, err := NewSunoDetector()
	if err != nil {
		t.Fatalf("NewSunoDetector failed: %v", err)
	}

	sampleRate := 48000
	numSamples := sampleRate * 2 // 2 seconds
	samples := make([]float64, numSamples)

	// Clean full-range instrument: 440Hz + 8kHz + 18kHz overtones (natural acoustic mastering)
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		samples[i] = 0.3*math.Sin(2*math.Pi*440.0*tSec) +
			0.2*math.Sin(2*math.Pi*8000.0*tSec) +
			0.1*math.Sin(2*math.Pi*18000.0*tSec)
	}

	buf := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	res, err := sd.Analyze(buf)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if res.IsSunoGenerated {
		t.Errorf("clean recording falsely flagged as Suno: confidence %f", res.ConfidenceScore)
	}
	if res.CutoffDetected {
		t.Errorf("cutoff should not be detected in full-spectrum recording")
	}
}

func TestSunoDetector_SunoArtifactSynthetic(t *testing.T) {
	sd, err := NewSunoDetector()
	if err != nil {
		t.Fatalf("NewSunoDetector failed: %v", err)
	}

	sampleRate := 48000
	numSamples := sampleRate * 2
	samples := make([]float64, numSamples)

	// Emulate Suno v3 output:
	// Strong broadband energy up to 15.5kHz, then complete 0 above 16kHz
	// Plus comb filter modulation (periodic ripple every 200 Hz)
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		// Multi-tone cluster up to 15kHz
		base := 0.2*math.Sin(2*math.Pi*440.0*tSec) +
			0.2*math.Sin(2*math.Pi*1200.0*tSec) +
			0.2*math.Sin(2*math.Pi*5000.0*tSec) +
			0.2*math.Sin(2*math.Pi*12000.0*tSec) +
			0.2*math.Sin(2*math.Pi*15000.0*tSec)

		// Comb modulation
		comb := 0.1 * math.Sin(2*math.Pi*200.0*tSec) * base
		samples[i] = base + comb
	}

	buf := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	res, err := sd.Analyze(buf)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if !res.IsSunoGenerated {
		t.Errorf("expected Suno detection, but got false (confidence: %f)", res.ConfidenceScore)
	}
	if !res.CutoffDetected {
		t.Errorf("expected 16.5kHz cutoff to be detected")
	}
	if res.HighFreqAttenuationDb < 25.0 {
		t.Errorf("expected high attenuation above 16.5kHz, got %f dB", res.HighFreqAttenuationDb)
	}

	jsonStr, err := res.ToJSON()
	if err != nil || len(jsonStr) == 0 {
		t.Fatalf("ToJSON failed: %v", err)
	}
}
