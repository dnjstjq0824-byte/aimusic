package audio

import (
	"math"
	"testing"
)

func TestHarmonicExciter_EmptyBuffer(t *testing.T) {
	exciter := NewHarmonicExciter(DefaultExciterConfig())
	_, err := exciter.Process(nil)
	if err != ErrNilPCMBuffer {
		t.Fatalf("expected ErrNilPCMBuffer, got %v", err)
	}
}

func TestHarmonicExciter_TransparencyImprovement(t *testing.T) {
	cfg := DefaultExciterConfig()
	cfg.CutoffHz = 7500.0 // Upper range crossover
	cfg.AirMixDb = -6.0   // Blend air harmonics at -6dB
	exciter := NewHarmonicExciter(cfg)

	sampleRate := 48000
	numSamples := 48000 // 1 second
	samples := make([]float64, numSamples)

	// 8 kHz tone: 2nd harmonic (16kHz) and 3rd harmonic (24kHz) will be synthesized into the air band (>15kHz)
	for i := 0; i < numSamples; i++ {
		samples[i] = 0.5 * math.Sin(2*math.Pi*8000.0*float64(i)/float64(sampleRate))
	}

	buf := &PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	res, err := exciter.Process(buf)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if res.TransparencyGainPct < 15.0 {
		t.Errorf("expected transparency improvement >= 15%%, got %f%%", res.TransparencyGainPct)
	}
	if res.OutputPeak > 1.0 {
		t.Errorf("output peak exceeded 1.0 (clipping): %f", res.OutputPeak)
	}
	if res.EnhancedAirEnergy <= res.OriginalAirEnergy {
		t.Errorf("enhanced air energy should be strictly greater than original: %f vs %f", res.EnhancedAirEnergy, res.OriginalAirEnergy)
	}

	jsonStr, err := res.ToJSON()
	if err != nil || len(jsonStr) == 0 {
		t.Fatalf("ToJSON failed: %v", err)
	}
}

func TestHarmonicExciter_StereoPreservation(t *testing.T) {
	exciter := NewHarmonicExciter(DefaultExciterConfig())

	n := 24000
	left := make([]float64, n)
	right := make([]float64, n)
	for i := 0; i < n; i++ {
		left[i] = 0.4 * math.Sin(2*math.Pi*13000.0*float64(i)/48000.0)
		right[i] = 0.2 * math.Sin(2*math.Pi*13000.0*float64(i)/48000.0)
	}

	buf := &PCMBuffer{
		Channels:   2,
		SampleRate: 48000,
		NumSamples: n,
		Samples:    [][]float64{left, right},
	}

	res, err := exciter.Process(buf)
	if err != nil {
		t.Fatalf("Process stereo failed: %v", err)
	}

	outBuf := res.ProcessedBuffer
	if outBuf.Channels != 2 {
		t.Fatalf("expected 2 channels, got %d", outBuf.Channels)
	}

	// Verify Left and Right maintain proportional difference
	peakL := calcPeak(outBuf.Samples[0])
	peakR := calcPeak(outBuf.Samples[1])
	if peakL <= peakR {
		t.Errorf("channel separation violated: peakL=%f, peakR=%f", peakL, peakR)
	}
}
