package audio

import (
	"math"
	"strings"
	"testing"
)

func TestGainPumpingDetector_EmptyBuffer(t *testing.T) {
	gpd := NewGainPumpingDetector(DefaultGainPumpingConfig())
	_, err := gpd.Analyze(nil)
	if err != ErrNilPCMBuffer {
		t.Fatalf("expected ErrNilPCMBuffer, got %v", err)
	}
}

func TestGainPumpingDetector_CleanAudioPass(t *testing.T) {
	gpd := NewGainPumpingDetector(DefaultGainPumpingConfig())

	sampleRate := 48000
	numSamples := sampleRate * 2 // 2 seconds
	samples := make([]float64, numSamples)

	// Smooth gentle tone with constant amplitude
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		samples[i] = 0.5 * math.Sin(2*math.Pi*440.0*tSec)
	}

	buf := &PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	res, err := gpd.Analyze(buf)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if res.HasExcessivePumping {
		t.Errorf("clean audio should NOT have excessive pumping")
	}
	if res.RecommendedAction != "PASS" {
		t.Errorf("expected recommended action 'PASS', got '%s'", res.RecommendedAction)
	}
	if res.PumpingEventsCount != 0 {
		t.Errorf("expected 0 pumping events in clean audio, got %d", res.PumpingEventsCount)
	}
}

func TestGainPumpingDetector_ExtremePumpingReject(t *testing.T) {
	gpd := NewGainPumpingDetector(DefaultGainPumpingConfig())

	sampleRate := 48000
	numSamples := sampleRate * 2 // 2 seconds = 40 50ms frames
	samples := make([]float64, numSamples)

	// Alternate amplitude every 50ms between 0.8 (-2dBFS) and 0.05 (-26dBFS) -> 24dB cyclic pumping swings!
	windowSamples := sampleRate / 20 // 50ms = 2400 samples
	for i := 0; i < numSamples; i++ {
		frameIdx := i / windowSamples
		amp := 0.8
		if frameIdx%2 == 1 {
			amp = 0.05
		}
		tSec := float64(i) / float64(sampleRate)
		samples[i] = amp * math.Sin(2*math.Pi*440.0*tSec)
	}

	buf := &PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	res, err := gpd.Analyze(buf)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if !res.HasExcessivePumping {
		t.Errorf("expected HasExcessivePumping to be true for cyclic 24dB pumping")
	}
	if !strings.HasPrefix(res.RecommendedAction, "REJECT") {
		t.Errorf("expected REJECT action, got: %s", res.RecommendedAction)
	}
	if res.PumpingEventsCount < 5 {
		t.Errorf("expected at least 5 pumping events, got %d", res.PumpingEventsCount)
	}
	if res.MaxEnvelopeSwingDb < 20.0 {
		t.Errorf("expected max swing > 20dB, got %f dB", res.MaxEnvelopeSwingDb)
	}

	jsonStr, err := res.ToJSON()
	if err != nil || len(jsonStr) == 0 {
		t.Fatalf("ToJSON failed: %v", err)
	}
}
