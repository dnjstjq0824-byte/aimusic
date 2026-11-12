package audio

import (
	"math"
	"math/rand"
	"testing"
)

// generateTestSignal synthesizes a rich multi-frequency signal with transients.
func generateTestSignal(sampleRate, numSamples int) []float64 {
	sig := make([]float64, numSamples)
	sr := float64(sampleRate)

	freqs := []float64{60.0, 120.0, 240.0, 480.0, 1000.0, 2500.0}
	weights := []float64{0.3, 0.25, 0.2, 0.15, 0.1, 0.05}

	for i := 0; i < numSamples; i++ {
		t := float64(i) / sr
		var val float64
		for idx, f := range freqs {
			val += weights[idx] * math.Sin(2.0*math.Pi*f*t)
		}
		// Transient kick-like envelope every 0.5s
		envelope := math.Exp(-math.Mod(t, 0.5) * 15.0)
		sig[i] = val * (0.4 + 0.6*envelope)
	}
	return sig
}

// TestPhaseCorrelation_IdenticalAndInverted tests correlation metric extremes.
func TestPhaseCorrelation_IdenticalAndInverted(t *testing.T) {
	sr := 48000
	numSamples := 4800
	sigA := generateTestSignal(sr, numSamples)

	// Identical signals
	corrIdentical := ComputePhaseCorrelation(sigA, sigA)
	if math.Abs(corrIdentical-1.0) > 1e-6 {
		t.Errorf("Expected identical correlation 1.0, got %.6f", corrIdentical)
	}

	// 180-degree inverted signal
	sigInverted := make([]float64, numSamples)
	for i, v := range sigA {
		sigInverted[i] = -v
	}
	corrInverted := ComputePhaseCorrelation(sigA, sigInverted)
	if math.Abs(corrInverted-(-1.0)) > 1e-6 {
		t.Errorf("Expected inverted correlation -1.0, got %.6f", corrInverted)
	}

	// Orthogonal signals (sine vs cosine)
	sine := make([]float64, numSamples)
	cosine := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		phase := 2.0 * math.Pi * 100.0 * float64(i) / float64(sr)
		sine[i] = math.Sin(phase)
		cosine[i] = math.Cos(phase)
	}
	corrOrthogonal := ComputePhaseCorrelation(sine, cosine)
	if math.Abs(corrOrthogonal) > 0.02 {
		t.Errorf("Expected orthogonal correlation ~0.0, got %.4f", corrOrthogonal)
	}
}

// TestPhaseAligner_IntegerDelayAlignment verifies estimation and correction of integer sample delays.
func TestPhaseAligner_IntegerDelayAlignment(t *testing.T) {
	sr := 48000
	numSamples := 4800
	refSig := generateTestSignal(sr, numSamples)

	// Target lags by exactly 32 samples
	targetDelay := 32
	targetSig := make([]float64, numSamples)
	for i := targetDelay; i < numSamples; i++ {
		targetSig[i] = refSig[i-targetDelay]
	}

	refPCM := &PCMBuffer{
		SampleRate: sr,
		Channels:   1,
		NumSamples: numSamples,
		Samples:    [][]float64{refSig},
	}
	tgtPCM := &PCMBuffer{
		SampleRate: sr,
		Channels:   1,
		NumSamples: numSamples,
		Samples:    [][]float64{targetSig},
	}

	aligner := NewPhaseAligner(DefaultPhaseAlignerConfig())
	res, err := aligner.AlignTracks(refPCM, tgtPCM)
	if err != nil {
		t.Fatalf("AlignTracks failed: %v", err)
	}

	t.Logf("Integer Alignment: %s", res.String())

	if math.Abs(res.EstimatedDelaySamples-float64(targetDelay)) > 0.2 {
		t.Errorf("Expected delay ~%d samples, got %.3f", targetDelay, res.EstimatedDelaySamples)
	}

	if res.AlignedCorrelation < 0.95 {
		t.Errorf("Expected aligned correlation > 0.95, got %.4f", res.AlignedCorrelation)
	}

	if res.PolarityInverted {
		t.Errorf("Expected polarity not inverted, got true")
	}
}

// TestPhaseAligner_SubSampleFractionalDelay verifies sub-sample estimation precision.
func TestPhaseAligner_SubSampleFractionalDelay(t *testing.T) {
	sr := 48000
	numSamples := 4800
	refSig := generateTestSignal(sr, numSamples)

	// Create fractional delay (e.g. 14.4 samples) using high-precision Sinc filter
	aligner := NewPhaseAligner(DefaultPhaseAlignerConfig())
	fractionalDelay := 14.4
	delayedSig := aligner.applyFractionalDelay(refSig, fractionalDelay)

	refPCM := &PCMBuffer{
		SampleRate: sr,
		Channels:   1,
		NumSamples: numSamples,
		Samples:    [][]float64{refSig},
	}
	tgtPCM := &PCMBuffer{
		SampleRate: sr,
		Channels:   1,
		NumSamples: numSamples,
		Samples:    [][]float64{delayedSig},
	}

	res, err := aligner.AlignTracks(refPCM, tgtPCM)
	if err != nil {
		t.Fatalf("Fractional AlignTracks failed: %v", err)
	}

	t.Logf("Sub-sample Alignment: %s", res.String())

	// Estimation should be accurate within 0.2 samples
	if math.Abs(res.EstimatedDelaySamples-fractionalDelay) > 0.2 {
		t.Errorf("Expected sub-sample delay ~%.2f, got %.3f", fractionalDelay, res.EstimatedDelaySamples)
	}

	if res.AlignedCorrelation < 0.90 {
		t.Errorf("Expected aligned correlation > 0.90, got %.4f", res.AlignedCorrelation)
	}
}

// TestPhaseAligner_PolarityInversionDetection tests 180-degree phase flip detection and correction.
func TestPhaseAligner_PolarityInversionDetection(t *testing.T) {
	sr := 48000
	numSamples := 4800
	refSig := generateTestSignal(sr, numSamples)

	// Target is 180-degree inverted AND delayed by 18 samples
	targetSig := make([]float64, numSamples)
	delay := 18
	for i := delay; i < numSamples; i++ {
		targetSig[i] = -refSig[i-delay] // Inverted polarity
	}

	refPCM := &PCMBuffer{
		SampleRate: sr,
		Channels:   1,
		NumSamples: numSamples,
		Samples:    [][]float64{refSig},
	}
	tgtPCM := &PCMBuffer{
		SampleRate: sr,
		Channels:   1,
		NumSamples: numSamples,
		Samples:    [][]float64{targetSig},
	}

	aligner := NewPhaseAligner(DefaultPhaseAlignerConfig())
	res, err := aligner.AlignTracks(refPCM, tgtPCM)
	if err != nil {
		t.Fatalf("AlignTracks failed: %v", err)
	}

	t.Logf("Polarity Inversion Alignment: %s", res.String())

	if !res.PolarityInverted {
		t.Errorf("Expected PolarityInverted=true, got false")
	}

	if res.AlignedCorrelation < 0.90 {
		t.Errorf("Expected aligned correlation > 0.90 after polarity flip, got %.4f", res.AlignedCorrelation)
	}
}

// TestPhaseAligner_CombFilteringRestoration tests cancellation prevention and constructive gain.
func TestPhaseAligner_CombFilteringRestoration(t *testing.T) {
	sr := 48000
	numSamples := 9600

	// 1kHz tone: period is 48 samples. Half period is 24 samples (complete cancellation).
	freq := 1000.0
	refSig := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		refSig[i] = math.Sin(2.0 * math.Pi * freq * float64(i) / float64(sr))
	}

	// Target delayed by half period (24 samples) -> comb notch causes cancellation
	targetSig := make([]float64, numSamples)
	delay := 24
	for i := delay; i < numSamples; i++ {
		targetSig[i] = refSig[i-delay]
	}

	refPCM := &PCMBuffer{
		SampleRate: sr,
		Channels:   1,
		NumSamples: numSamples,
		Samples:    [][]float64{refSig},
	}
	tgtPCM := &PCMBuffer{
		SampleRate: sr,
		Channels:   1,
		NumSamples: numSamples,
		Samples:    [][]float64{targetSig},
	}

	aligner := NewPhaseAligner(DefaultPhaseAlignerConfig())
	res, err := aligner.AlignTracks(refPCM, tgtPCM)
	if err != nil {
		t.Fatalf("AlignTracks failed: %v", err)
	}

	t.Logf("Comb Filtering Restoration: %s", res.String())

	// Coherence gain must be significant (summing out-of-phase signals cancels energy, aligned doubles amplitude)
	if res.CoherenceGainDB < 3.0 {
		t.Errorf("Expected CoherenceGainDB > 3.0 dB, got %.2f dB", res.CoherenceGainDB)
	}

	if res.AlignedCorrelation < 0.85 {
		t.Errorf("Expected AlignedCorrelation > 0.85, got %.4f", res.AlignedCorrelation)
	}
}

// TestPhaseAligner_MultiTrackStemAlignment tests multi-track stem alignment against a reference.
func TestPhaseAligner_MultiTrackStemAlignment(t *testing.T) {
	sr := 48000
	numSamples := 9600

	baseSig := generateTestSignal(sr, numSamples)

	// Simulate 4 drum/percussion tracks with different acoustic mic distances and 1 inverted mic
	// Track 0 (Ref): Kick In (0 ms)
	// Track 1: Kick Out (+1.5 ms delay = 72 samples)
	// Track 2: Sub Kick (-0.8 ms lead = -38 samples)
	// Track 3: Overhead Mic (+2.2 ms delay = 106 samples, inverted polarity)

	delays := []int{0, 72, -38, 106}
	inverts := []bool{false, false, false, true}

	tracks := make([]*PCMBuffer, 4)
	for trkIdx := 0; trkIdx < 4; trkIdx++ {
		samples := make([]float64, numSamples)
		d := delays[trkIdx]
		inv := inverts[trkIdx]

		for i := 0; i < numSamples; i++ {
			srcIdx := i - d
			if srcIdx >= 0 && srcIdx < numSamples {
				v := baseSig[srcIdx]
				if inv {
					v = -v
				}
				samples[i] = v
			}
		}

		tracks[trkIdx] = &PCMBuffer{
			SampleRate: sr,
			Channels:   1,
			NumSamples: numSamples,
			Samples:    [][]float64{samples},
		}
	}

	aligner := NewPhaseAligner(DefaultPhaseAlignerConfig())
	multiRes, err := aligner.AlignMultiTrack(tracks, 0)
	if err != nil {
		t.Fatalf("AlignMultiTrack failed: %v", err)
	}

	t.Logf("MultiTrack Total Coherence Gain: +%.2f dB", multiRes.TotalCoherenceGainDB)

	for i, trkRes := range multiRes.TrackResults {
		t.Logf("  Track %d: %s", i, trkRes.String())
		if trkRes.AlignedCorrelation < 0.85 {
			t.Errorf("Track %d aligned correlation < 0.85: got %.4f", i, trkRes.AlignedCorrelation)
		}
	}

	if multiRes.TotalCoherenceGainDB < 2.0 {
		t.Errorf("Expected MultiTrack total coherence gain > 2.0 dB, got %.2f dB", multiRes.TotalCoherenceGainDB)
	}
}

// TestPhaseAligner_EdgeCases tests invalid configurations and corner conditions.
func TestPhaseAligner_EdgeCases(t *testing.T) {
	aligner := NewPhaseAligner(DefaultPhaseAlignerConfig())

	// Nil tracks
	if _, err := aligner.AlignTracks(nil, nil); err == nil {
		t.Errorf("Expected error for nil tracks")
	}

	// Empty buffers
	emptyBuf := &PCMBuffer{
		SampleRate: 48000,
		Channels:   1,
		NumSamples: 0,
		Samples:    [][]float64{},
	}
	if _, err := aligner.AlignTracks(emptyBuf, emptyBuf); err == nil {
		t.Errorf("Expected error for empty buffers")
	}

	// Mismatched sample rate
	buf441 := &PCMBuffer{
		SampleRate: 44100,
		Channels:   1,
		NumSamples: 1000,
		Samples:    [][]float64{make([]float64, 1000)},
	}
	buf480 := &PCMBuffer{
		SampleRate: 48000,
		Channels:   1,
		NumSamples: 1000,
		Samples:    [][]float64{make([]float64, 1000)},
	}
	if _, err := aligner.AlignTracks(buf441, buf480); err == nil {
		t.Errorf("Expected error for mismatched sample rates")
	}
}

// TestPhaseAligner_100HeterogeneousMultiTracks runs 100 heterogeneous multi-track alignment tests.
func TestPhaseAligner_100HeterogeneousMultiTracks(t *testing.T) {
	rng := rand.New(rand.NewSource(11223344))
	sampleRates := []int{22050, 44100, 48000, 96000}

	for i := 0; i < 100; i++ {
		sr := sampleRates[rng.Intn(len(sampleRates))]
		channels := rng.Intn(2) + 1     // 1 or 2 channels
		length := rng.Intn(2400) + 2400 // 50ms to 200ms

		refBase := generateTestSignal(sr, length)
		refSamples := make([][]float64, channels)
		for ch := 0; ch < channels; ch++ {
			refSamples[ch] = make([]float64, length)
			copy(refSamples[ch], refBase)
		}

		refPCM := &PCMBuffer{
			SampleRate: sr,
			Channels:   channels,
			NumSamples: length,
			Samples:    refSamples,
		}

		// Random delay in [-10ms, +10ms]
		maxLagSamples := int(float64(sr) * 0.010)
		randomLag := rng.Intn(2*maxLagSamples) - maxLagSamples
		randomInvert := rng.Intn(2) == 1

		targetSamples := make([][]float64, channels)
		for ch := 0; ch < channels; ch++ {
			targetSamples[ch] = make([]float64, length)
			for s := 0; s < length; s++ {
				srcIdx := s - randomLag
				if srcIdx >= 0 && srcIdx < length {
					val := refBase[srcIdx]
					if randomInvert {
						val = -val
					}
					targetSamples[ch][s] = val
				}
			}
		}

		targetPCM := &PCMBuffer{
			SampleRate: sr,
			Channels:   channels,
			NumSamples: length,
			Samples:    targetSamples,
		}

		aligner := NewPhaseAligner(PhaseAlignerConfig{
			MaxDelayMs:         15.0,
			SubSampleAccuracy:  true,
			AutoPolarityInvert: true,
			FilterOrder:        15,
		})

		res, err := aligner.AlignTracks(refPCM, targetPCM)
		if err != nil {
			t.Fatalf("Stream %d alignment failed: %v", i+1, err)
		}

		// Requirement: Phase correlation > 0.85
		if res.AlignedCorrelation < 0.85 {
			t.Fatalf("Stream %d aligned correlation < 0.85: got %.4f (Initial=%.4f, Lag=%d, Invert=%v)",
				i+1, res.AlignedCorrelation, res.InitialCorrelation, randomLag, randomInvert)
		}
	}

	t.Log("Successfully verified 100 heterogeneous multi-track streams with Phase Correlation > 0.85.")
}
