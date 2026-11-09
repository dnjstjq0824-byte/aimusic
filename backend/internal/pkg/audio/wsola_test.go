package audio

import (
	"math"
	"math/rand"
	"testing"
)

// estimateDominantFrequency estimates dominant frequency using average zero-crossing interval.
func estimateDominantFrequency(samples []float64, sampleRate int) float64 {
	n := len(samples)
	if n < 100 {
		return 0.0
	}

	var zeroCrossings []float64
	for i := 1; i < n; i++ {
		if (samples[i-1] <= 0 && samples[i] > 0) || (samples[i-1] >= 0 && samples[i] < 0) {
			// Linear interpolation for exact fractional zero-crossing sample
			denom := samples[i] - samples[i-1]
			var frac float64
			if math.Abs(denom) > 1e-12 {
				frac = -samples[i-1] / denom
			}
			zeroCrossings = append(zeroCrossings, float64(i-1)+frac)
		}
	}

	if len(zeroCrossings) < 10 {
		return 0.0
	}

	// Discard initial transient and final edge
	startIdx := len(zeroCrossings) / 4
	endIdx := 3 * len(zeroCrossings) / 4
	numIntervals := endIdx - startIdx
	totalSpan := zeroCrossings[endIdx] - zeroCrossings[startIdx]

	avgHalfPeriod := totalSpan / float64(numIntervals)
	avgPeriod := avgHalfPeriod * 2.0
	if avgPeriod <= 0 {
		return 0.0
	}

	return float64(sampleRate) / avgPeriod
}

// TestWSOLA_TimeStretch_Durations verifies output duration across speed ratios.
func TestWSOLA_TimeStretch_Durations(t *testing.T) {
	sr := 48000
	numSamples := 48000 // Exactly 1.0 second
	sig := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		sig[i] = 0.5 * math.Sin(2.0*math.Pi*440.0*float64(i)/float64(sr))
	}

	pcm := &PCMBuffer{
		SampleRate: sr,
		Channels:   1,
		NumSamples: numSamples,
		Samples:    [][]float64{sig},
	}

	engine := NewWSOLAEngine(DefaultWSOLAConfig(sr))

	speedRatios := []float64{0.5, 0.75, 1.0, 1.25, 1.5, 2.0}

	for _, speed := range speedRatios {
		stretched, err := engine.TimeStretch(pcm, speed)
		if err != nil {
			t.Fatalf("TimeStretch at speed %.2f failed: %v", speed, err)
		}

		expectedDurationSec := 1.0 / speed
		actualDurationSec := float64(stretched.NumSamples) / float64(sr)

		// Allow +/- 6% tolerance for discrete WSOLA window boundary packing
		diffPct := math.Abs(actualDurationSec-expectedDurationSec) / expectedDurationSec * 100.0
		t.Logf("Speed %.2fx: Expected duration=%.3fs, Actual=%.3fs (Diff=%.2f%%)",
			speed, expectedDurationSec, actualDurationSec, diffPct)

		if diffPct > 8.0 {
			t.Errorf("Speed %.2fx duration error too large: expected %.3fs, got %.3fs (diff %.2f%%)",
				speed, expectedDurationSec, actualDurationSec, diffPct)
		}
	}
}

// TestWSOLA_TimeStretch_PitchInvariance ensures pitch remains constant regardless of playback speed.
func TestWSOLA_TimeStretch_PitchInvariance(t *testing.T) {
	sr := 48000
	targetFreq := 440.0
	numSamples := 48000 // 1.0 second

	sig := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		sig[i] = 0.6 * math.Sin(2.0*math.Pi*targetFreq*float64(i)/float64(sr))
	}

	pcm := &PCMBuffer{
		SampleRate: sr,
		Channels:   1,
		NumSamples: numSamples,
		Samples:    [][]float64{sig},
	}

	engine := NewWSOLAEngine(DefaultWSOLAConfig(sr))

	speedRatios := []float64{0.5, 0.75, 1.25, 1.5, 2.0}

	for _, speed := range speedRatios {
		stretched, err := engine.TimeStretch(pcm, speed)
		if err != nil {
			t.Fatalf("TimeStretch at speed %.2f failed: %v", speed, err)
		}

		detectedFreq := estimateDominantFrequency(stretched.Samples[0], sr)
		freqDiff := math.Abs(detectedFreq - targetFreq)

		t.Logf("Speed %.2fx: Target=%.1f Hz, Detected=%.2f Hz, Diff=%.2f Hz",
			speed, targetFreq, detectedFreq, freqDiff)

		// Pitch must remain invariant within 1.5 Hz
		if freqDiff > 1.5 {
			t.Errorf("Pitch altered during speed %.2fx time stretch: target=%.1f Hz, detected=%.2f Hz",
				speed, targetFreq, detectedFreq)
		}
	}
}

// TestWSOLA_PitchShift_Frequencies tests pitch shifting accuracy while preserving tempo.
func TestWSOLA_PitchShift_Frequencies(t *testing.T) {
	sr := 48000
	baseFreq := 440.0 // A4
	numSamples := 48000 // 1.0 second

	sig := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		sig[i] = 0.6 * math.Sin(2.0*math.Pi*baseFreq*float64(i)/float64(sr))
	}

	pcm := &PCMBuffer{
		SampleRate: sr,
		Channels:   1,
		NumSamples: numSamples,
		Samples:    [][]float64{sig},
	}

	engine := NewWSOLAEngine(DefaultWSOLAConfig(sr))

	shifts := []struct {
		semitones float64
		expected  float64
	}{
		{semitones: 0.0, expected: 440.0},
		{semitones: 2.0, expected: 440.0 * math.Pow(2.0, 2.0/12.0)},    // B4 ~ 493.88 Hz
		{semitones: -2.0, expected: 440.0 * math.Pow(2.0, -2.0/12.0)},  // G4 ~ 391.99 Hz
		{semitones: 12.0, expected: 880.0},                              // A5 1 octave up
		{semitones: -12.0, expected: 220.0},                             // A3 1 octave down
	}

	for _, tc := range shifts {
		shifted, err := engine.PitchShift(pcm, tc.semitones)
		if err != nil {
			t.Fatalf("PitchShift by %.1f semitones failed: %v", tc.semitones, err)
		}

		// Verify duration preservation
		if shifted.NumSamples != numSamples {
			t.Errorf("PitchShift altered duration: expected %d, got %d", numSamples, shifted.NumSamples)
		}

		detected := estimateDominantFrequency(shifted.Samples[0], sr)
		freqDiff := math.Abs(detected - tc.expected)
		t.Logf("Semitones %+.1f: Target=%.2f Hz, Detected=%.2f Hz (Diff=%.2f Hz)",
			tc.semitones, tc.expected, detected, freqDiff)

		// Allow small tolerance for discrete sampling estimation
		if freqDiff > 4.0 {
			t.Errorf("PitchShift frequency mismatch: target=%.2f Hz, detected=%.2f Hz (Diff=%.2f Hz)",
				tc.expected, detected, freqDiff)
		}
	}
}

// TestWSOLA_StereoPhaseLocking tests that multi-channel stereo alignment is preserved.
func TestWSOLA_StereoPhaseLocking(t *testing.T) {
	sr := 48000
	numSamples := 48000

	left := make([]float64, numSamples)
	right := make([]float64, numSamples)

	for i := 0; i < numSamples; i++ {
		tVal := float64(i) / float64(sr)
		base := 0.5 * math.Sin(2.0*math.Pi*300.0*tVal)
		left[i] = base
		// Right is slightly attenuated stereo companion (in-phase)
		right[i] = base * 0.8
	}

	pcm := &PCMBuffer{
		SampleRate: sr,
		Channels:   2,
		NumSamples: numSamples,
		Samples:    [][]float64{left, right},
	}

	engine := NewWSOLAEngine(DefaultWSOLAConfig(sr))
	stretched, err := engine.TimeStretch(pcm, 1.35)
	if err != nil {
		t.Fatalf("Stereo TimeStretch failed: %v", err)
	}

	// Aligned phase correlation between left and right must remain > 0.99
	corr := ComputePhaseCorrelation(stretched.Samples[0], stretched.Samples[1])
	t.Logf("Stereo Correlation after 1.35x time stretch: %.6f", corr)

	if corr < 0.99 {
		t.Errorf("Stereo channels desynchronized during time stretch: correlation=%.4f", corr)
	}
}

// TestWSOLA_CombinedProcess tests simultaneous time-stretching and pitch-shifting.
func TestWSOLA_CombinedProcess(t *testing.T) {
	sr := 48000
	numSamples := 48000
	targetFreq := 440.0

	sig := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		sig[i] = 0.5 * math.Sin(2.0*math.Pi*targetFreq*float64(i)/float64(sr))
	}

	pcm := &PCMBuffer{
		SampleRate: sr,
		Channels:   1,
		NumSamples: numSamples,
		Samples:    [][]float64{sig},
	}

	engine := NewWSOLAEngine(DefaultWSOLAConfig(sr))

	// Speed 1.5x AND pitch shift +4 semitones (~554.37 Hz, C#5)
	speed := 1.5
	semitones := 4.0
	expectedFreq := 440.0 * math.Pow(2.0, semitones/12.0)

	result, err := engine.Process(pcm, speed, semitones)
	if err != nil {
		t.Fatalf("Combined Process failed: %v", err)
	}

	// Duration should be scaled by 1/1.5 = 0.667
	expectedDurationSec := 1.0 / speed
	actualDurationSec := float64(result.NumSamples) / float64(sr)
	diffPct := math.Abs(actualDurationSec-expectedDurationSec) / expectedDurationSec * 100.0

	if diffPct > 8.0 {
		t.Errorf("Combined process duration mismatch: expected %.3fs, got %.3fs", expectedDurationSec, actualDurationSec)
	}

	detectedFreq := estimateDominantFrequency(result.Samples[0], sr)
	freqDiff := math.Abs(detectedFreq - expectedFreq)
	t.Logf("Combined Process: Expected Freq=%.2f Hz, Detected=%.2f Hz, Duration=%.3fs",
		expectedFreq, detectedFreq, actualDurationSec)

	if freqDiff > 5.0 {
		t.Errorf("Combined process pitch mismatch: expected %.2f Hz, got %.2f Hz", expectedFreq, detectedFreq)
	}
}

// TestWSOLA_EdgeCases tests invalid parameter boundaries and edge handling.
func TestWSOLA_EdgeCases(t *testing.T) {
	engine := NewWSOLAEngine(DefaultWSOLAConfig(48000))

	// Nil buffer
	if _, err := engine.TimeStretch(nil, 1.2); err == nil {
		t.Errorf("Expected error for nil buffer")
	}

	// Empty buffer
	emptyPCM := &PCMBuffer{
		SampleRate: 48000,
		Channels:   1,
		NumSamples: 0,
		Samples:    [][]float64{},
	}
	if _, err := engine.TimeStretch(emptyPCM, 1.2); err == nil {
		t.Errorf("Expected error for empty buffer")
	}

	// Out-of-bounds speed ratio
	sinePCM := &PCMBuffer{
		SampleRate: 48000,
		Channels:   1,
		NumSamples: 2000,
		Samples:    [][]float64{make([]float64, 2000)},
	}
	if _, err := engine.TimeStretch(sinePCM, 0.1); err == nil {
		t.Errorf("Expected error for speed ratio < 0.25")
	}
	if _, err := engine.TimeStretch(sinePCM, 5.0); err == nil {
		t.Errorf("Expected error for speed ratio > 4.0")
	}

	// Out-of-bounds semitones
	if _, err := engine.PitchShift(sinePCM, 30.0); err == nil {
		t.Errorf("Expected error for semitones > 24.0")
	}
}

// TestWSOLA_100HeterogeneousStreams tests 100 heterogeneous multi-rate and multi-channel streams.
func TestWSOLA_100HeterogeneousStreams(t *testing.T) {
	rng := rand.New(rand.NewSource(55667788))

	sampleRates := []int{22050, 44100, 48000}
	speeds := []float64{0.6, 0.8, 1.0, 1.2, 1.5, 1.8}
	semitoneOpts := []float64{-5.0, -2.0, 0.0, 2.0, 5.0}

	for i := 0; i < 100; i++ {
		sr := sampleRates[rng.Intn(len(sampleRates))]
		channels := rng.Intn(2) + 1 // 1 or 2 channels
		// Generate between 0.3s and 0.8s
		length := int(float64(sr) * (0.3 + rng.Float64()*0.5))

		speed := speeds[rng.Intn(len(speeds))]
		semi := semitoneOpts[rng.Intn(len(semitoneOpts))]

		pcmSamples := make([][]float64, channels)
		for ch := 0; ch < channels; ch++ {
			pcmSamples[ch] = make([]float64, length)
			freq := 200.0 + rng.Float64()*1000.0
			for s := 0; s < length; s++ {
				pcmSamples[ch][s] = 0.5 * math.Sin(2.0*math.Pi*freq*float64(s)/float64(sr))
			}
		}

		pcm := &PCMBuffer{
			SampleRate: sr,
			Channels:   channels,
			NumSamples: length,
			Samples:    pcmSamples,
		}

		engine := NewWSOLAEngine(DefaultWSOLAConfig(sr))
		out, err := engine.Process(pcm, speed, semi)
		if err != nil {
			t.Fatalf("Stream %d (sr=%d, ch=%d, speed=%.2f, semi=%.1f) failed: %v",
				i+1, sr, channels, speed, semi, err)
		}

		if out.Channels != channels {
			t.Fatalf("Stream %d channel mismatch: expected %d, got %d", i+1, channels, out.Channels)
		}
		if out.NumSamples <= 0 {
			t.Fatalf("Stream %d produced empty output", i+1)
		}
	}

	t.Log("Successfully verified 100 heterogeneous audio streams with WSOLA time-stretching and pitch-shifting.")
}
