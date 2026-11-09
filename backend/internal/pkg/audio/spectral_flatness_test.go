package audio

import (
	"math"
	"math/rand"
	"testing"
)

func TestSpectralFlatness_EmptyBuffer(t *testing.T) {
	sfa := NewSpectralFlatnessAnalyzer(DefaultSpectralFlatnessConfig())
	_, err := sfa.Analyze(nil)
	if err != ErrNilPCMBuffer {
		t.Fatalf("expected ErrNilPCMBuffer, got %v", err)
	}
}

func TestSpectralFlatness_PureSineWave(t *testing.T) {
	cfg := DefaultSpectralFlatnessConfig()
	cfg.FFTSize = 512
	cfg.HopSize = 256
	sfa := NewSpectralFlatnessAnalyzer(cfg)

	sampleRate := 48000
	numSamples := 4800 // 100ms
	samples := make([]float64, numSamples)

	// Pure 440Hz sine wave
	for i := 0; i < numSamples; i++ {
		samples[i] = 0.5 * math.Sin(2*math.Pi*440.0*float64(i)/float64(sampleRate))
	}

	buf := &PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	res, err := sfa.Analyze(buf)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Pure tone has low SFM (< 0.15) and high Tonality (> 85%)
	if res.GlobalSFM > 0.15 {
		t.Errorf("expected low SFM for pure sine, got %f", res.GlobalSFM)
	}
	if res.TonalityScore < 85.0 {
		t.Errorf("expected high TonalityScore for pure sine, got %f", res.TonalityScore)
	}
}

func TestSpectralFlatness_WhiteNoise(t *testing.T) {
	cfg := DefaultSpectralFlatnessConfig()
	cfg.FFTSize = 512
	cfg.HopSize = 256
	sfa := NewSpectralFlatnessAnalyzer(cfg)

	sampleRate := 48000
	numSamples := 4800
	samples := make([]float64, numSamples)

	r := rand.New(rand.NewSource(42))
	for i := 0; i < numSamples; i++ {
		samples[i] = r.Float64()*2.0 - 1.0 // White noise in [-1.0, 1.0]
	}

	buf := &PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	res, err := sfa.Analyze(buf)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// White noise has high SFM (> 0.50) and high Noisiness (> 50%)
	if res.GlobalSFM < 0.50 {
		t.Errorf("expected high SFM for white noise, got %f", res.GlobalSFM)
	}
	if res.NoisinessScore < 50.0 {
		t.Errorf("expected high NoisinessScore for white noise, got %f", res.NoisinessScore)
	}

	jsonStr, err := res.ToJSON()
	if err != nil || len(jsonStr) == 0 {
		t.Fatalf("ToJSON failed: %v", err)
	}
}
