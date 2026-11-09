package audio

import (
	"math"
	"math/rand"
	"testing"
)

// TestTransition_03s_DiscontinuityRemoval verifies 0.3s micro-fade suppresses edge clicks.
func TestTransition_03s_DiscontinuityRemoval(t *testing.T) {
	sr := 48000
	numSamples := 48000 // 1.0 second

	// Non-zero truncated sine wave causing loud clicks at boundaries:
	// x[0] = 0.8, x[N-1] = -0.75
	sig := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		// Non-zero phase offset
		sig[i] = 0.8 * math.Cos(2.0*math.Pi*440.0*float64(i)/float64(sr))
	}

	pcm := &PCMBuffer{
		SampleRate: sr,
		Channels:   1,
		NumSamples: numSamples,
		Samples:    [][]float64{sig},
	}

	processor := NewTransitionProcessor(DefaultTransitionConfig())
	res, err := processor.ProcessTrackBoundaries(pcm)
	if err != nil {
		t.Fatalf("ProcessTrackBoundaries failed: %v", err)
	}

	t.Logf("Transition 0.3s: InitialHead=%.2f dBFS, InitialTail=%.2f dBFS, FinalHead=%.2e, FinalTail=%.2e",
		res.InitialHeadDiscontinuityDBFS, res.InitialTailDiscontinuityDBFS,
		res.FinalHeadEdgeAmplitude, res.FinalTailEdgeAmplitude)

	if !res.ClickTransientSuppressed {
		t.Errorf("Click transients were not suppressed at boundary")
	}

	if res.FinalHeadEdgeAmplitude > 1e-5 {
		t.Errorf("Final head edge amplitude not zero: %e", res.FinalHeadEdgeAmplitude)
	}
	if res.FinalTailEdgeAmplitude > 1e-5 {
		t.Errorf("Final tail edge amplitude not zero: %e", res.FinalTailEdgeAmplitude)
	}

	// Mid-track content (outside 0.3s head/tail) must be 100% preserved
	midIdx := numSamples / 2
	if math.Abs(res.ProcessedBuffer.Samples[0][midIdx]-sig[midIdx]) > 1e-12 {
		t.Errorf("Mid-track sample altered during boundary transition: %f != %f",
			res.ProcessedBuffer.Samples[0][midIdx], sig[midIdx])
	}
}

// TestTransition_Crossfade_EqualPowerEnergyConservation tests constant acoustic energy across crossfade.
func TestTransition_Crossfade_EqualPowerEnergyConservation(t *testing.T) {
	sr := 48000
	numSamples := 48000

	sigA := make([]float64, numSamples)
	sigB := make([]float64, numSamples)

	for i := 0; i < numSamples; i++ {
		tVal := float64(i) / float64(sr)
		sigA[i] = 0.5 * math.Sin(2.0*math.Pi*440.0*tVal)
		sigB[i] = 0.5 * math.Sin(2.0*math.Pi*880.0*tVal)
	}

	pcmA := &PCMBuffer{SampleRate: sr, Channels: 1, NumSamples: numSamples, Samples: [][]float64{sigA}}
	pcmB := &PCMBuffer{SampleRate: sr, Channels: 1, NumSamples: numSamples, Samples: [][]float64{sigB}}

	cfg := DefaultTransitionConfig()
	cfg.DurationSeconds = 0.3
	cfg.Curve = FadeCurveEqualPower

	processor := NewTransitionProcessor(cfg)
	res, err := processor.Crossfade(pcmA, pcmB)
	if err != nil {
		t.Fatalf("Crossfade failed: %v", err)
	}

	t.Logf("Crossfade Energy Variance: %.6f dB", res.MidpointEnergyVarianceDB)

	if !res.EnergyConservationCompliant {
		t.Errorf("Energy conservation non-compliant: variance = %.4f dB (expected < 0.1 dB)",
			res.MidpointEnergyVarianceDB)
	}
	if res.MidpointEnergyVarianceDB > 0.05 {
		t.Errorf("Equal power variance exceeds 0.05 dB: got %.6f dB", res.MidpointEnergyVarianceDB)
	}

	// Verify total output length
	expectedLen := numSamples + numSamples - res.CrossfadeSamples
	if res.BlendedBuffer.NumSamples != expectedLen {
		t.Errorf("Blended length mismatch: expected %d, got %d", expectedLen, res.BlendedBuffer.NumSamples)
	}
}

// TestTransition_MultipleCurves tests fade curves compatibility.
func TestTransition_MultipleCurves(t *testing.T) {
	sr := 48000
	numSamples := 48000
	sig := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		sig[i] = 0.5
	}

	curves := []FadeCurve{
		FadeCurveEqualPower,
		FadeCurveSCurve,
		FadeCurveLinear,
		FadeCurveExponential,
	}

	for _, c := range curves {
		pcm := &PCMBuffer{SampleRate: sr, Channels: 1, NumSamples: numSamples, Samples: [][]float64{sig}}
		cfg := DefaultTransitionConfig()
		cfg.Curve = c
		proc := NewTransitionProcessor(cfg)

		res, err := proc.ProcessTrackBoundaries(pcm)
		if err != nil {
			t.Fatalf("Curve %s failed: %v", c, err)
		}
		if !res.ClickTransientSuppressed {
			t.Errorf("Curve %s failed to suppress click transients", c)
		}
	}
}

// TestTransition_EdgeCases tests invalid inputs and short buffers.
func TestTransition_EdgeCases(t *testing.T) {
	proc := NewTransitionProcessor(DefaultTransitionConfig())

	// Nil buffer
	if _, err := proc.ProcessTrackBoundaries(nil); err == nil {
		t.Errorf("Expected error for nil buffer")
	}

	// Empty buffer
	emptyPCM := &PCMBuffer{SampleRate: 48000, Channels: 1, NumSamples: 0, Samples: [][]float64{}}
	if _, err := proc.ProcessTrackBoundaries(emptyPCM); err == nil {
		t.Errorf("Expected error for empty buffer")
	}

	// Short audio buffer (0.1s < 0.3s transition duration)
	shortSamples := make([]float64, 4800)
	for i := range shortSamples {
		shortSamples[i] = 0.5
	}
	shortPCM := &PCMBuffer{SampleRate: 48000, Channels: 1, NumSamples: 4800, Samples: [][]float64{shortSamples}}
	res, err := proc.ProcessTrackBoundaries(shortPCM)
	if err != nil {
		t.Fatalf("Short buffer failed: %v", err)
	}
	if !res.ClickTransientSuppressed {
		t.Errorf("Short buffer clicks not suppressed")
	}
}

// TestTransition_100HeterogeneousStreams verifies 100 heterogeneous transitions and crossfades.
func TestTransition_100HeterogeneousStreams(t *testing.T) {
	rng := rand.New(rand.NewSource(11224466))

	sampleRates := []int{22050, 44100, 48000, 96000}
	curves := []FadeCurve{FadeCurveEqualPower, FadeCurveSCurve, FadeCurveLinear, FadeCurveExponential}

	for i := 0; i < 100; i++ {
		sr := sampleRates[rng.Intn(len(sampleRates))]
		channels := rng.Intn(2) + 1
		lengthA := int(float64(sr) * (0.5 + rng.Float64()*0.5)) // 0.5s to 1.0s
		lengthB := int(float64(sr) * (0.5 + rng.Float64()*0.5))

		samplesA := make([][]float64, channels)
		samplesB := make([][]float64, channels)
		for ch := 0; ch < channels; ch++ {
			samplesA[ch] = make([]float64, lengthA)
			samplesB[ch] = make([]float64, lengthB)
			for s := 0; s < lengthA; s++ {
				samplesA[ch][s] = 0.6 * math.Sin(2.0*math.Pi*440.0*float64(s)/float64(sr))
			}
			for s := 0; s < lengthB; s++ {
				samplesB[ch][s] = 0.6 * math.Sin(2.0*math.Pi*880.0*float64(s)/float64(sr))
			}
		}

		pcmA := &PCMBuffer{SampleRate: sr, Channels: channels, NumSamples: lengthA, Samples: samplesA}
		pcmB := &PCMBuffer{SampleRate: sr, Channels: channels, NumSamples: lengthB, Samples: samplesB}

		curve := curves[rng.Intn(len(curves))]
		cfg := TransitionConfig{
			DurationSeconds:          0.15 + rng.Float64()*0.2, // 0.15s to 0.35s
			Curve:                    curve,
			AlignZeroCrossing:        true,
			ZeroCrossingSearchRadius: 64,
		}

		proc := NewTransitionProcessor(cfg)

		// 1. Process track boundaries
		resA, err := proc.ProcessTrackBoundaries(pcmA)
		if err != nil {
			t.Fatalf("Stream %d boundary transition failed: %v", i+1, err)
		}
		if !resA.ClickTransientSuppressed {
			t.Fatalf("Stream %d click transient not suppressed", i+1)
		}

		// 2. Crossfade A and B
		resCross, err := proc.Crossfade(pcmA, pcmB)
		if err != nil {
			t.Fatalf("Stream %d crossfade failed: %v", i+1, err)
		}
		if resCross.BlendedBuffer.Channels != channels {
			t.Fatalf("Stream %d channel mismatch", i+1)
		}
	}

	t.Log("Successfully verified 100 heterogeneous audio streams with 0.3s transitions and crossfades.")
}
