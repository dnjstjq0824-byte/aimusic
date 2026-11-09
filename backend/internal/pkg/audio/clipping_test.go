package audio

import (
	"math"
	"testing"
)

func TestClippingDetector_EmptyBuffer(t *testing.T) {
	detector := NewClippingDetector(DefaultClippingConfig())
	_, err := detector.Analyze(nil)
	if err != ErrNilPCMBuffer {
		t.Fatalf("expected ErrNilPCMBuffer, got %v", err)
	}
}

func TestClippingDetector_CleanAudio(t *testing.T) {
	detector := NewClippingDetector(DefaultClippingConfig())

	n := 48000 // 1 second
	samples := make([]float64, n)
	for i := 0; i < n; i++ {
		samples[i] = 0.7 * math.Sin(2*math.Pi*440.0*float64(i)/48000.0)
	}

	buf := &PCMBuffer{
		Channels:   1,
		SampleRate: 48000,
		NumSamples: n,
		Samples:    [][]float64{samples},
	}

	res, err := detector.Analyze(buf)
	if err != nil {
		t.Fatalf("Analyze clean failed: %v", err)
	}

	if res.TotalClips != 0 {
		t.Errorf("expected 0 clips in clean audio, got %d", res.TotalClips)
	}
	if res.HasWarning {
		t.Errorf("unexpected warning in clean audio")
	}
}

func TestClippingDetector_MultipleClipsWarning(t *testing.T) {
	detector := NewClippingDetector(DefaultClippingConfig())

	sampleRate := 48000
	numSamples := sampleRate * 3 // 3 seconds
	samples := make([]float64, numSamples)

	// Inject 6 separate clipping spikes (exceeding alert threshold of 5)
	clipPositions := []int{2000, 15000, 30000, 50000, 80000, 110000}
	for _, pos := range clipPositions {
		for k := 0; k < 5; k++ { // 5 consecutive samples at 1.05
			samples[pos+k] = 1.05
		}
	}

	buf := &PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	res, err := detector.Analyze(buf)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if res.TotalClips != 6 {
		t.Fatalf("expected 6 clips, got %d", res.TotalClips)
	}
	if !res.HasWarning {
		t.Errorf("expected QualityWarning for 6 clips, but has_warning was false")
	}
	if len(res.Heatmap) != 3 {
		t.Errorf("expected 3 1-second heatmap bins for 3 seconds of audio, got %d", len(res.Heatmap))
	}

	jsonStr, err := res.ToJSON()
	if err != nil || len(jsonStr) == 0 {
		t.Fatalf("failed to serialize clipping JSON: %v", err)
	}
}
