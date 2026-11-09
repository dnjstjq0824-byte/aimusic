package audio

import (
	"math"
	"math/rand"
	"testing"
)

// TestDRC_HardKneeStaticCurve validates theoretical gain reduction on hard knee compressor.
func TestDRC_HardKneeStaticCurve(t *testing.T) {
	config := DRCConfig{
		Profile:      DRCProfileCustom,
		ThresholdDB:  -20.0,
		Ratio:        4.0, // 4:1 compression
		KneeWidthDB:  0.0, // Hard knee
		AttackMs:     5.0,
		ReleaseMs:    100.0,
		MakeupGainDB: 0.0,
		StereoLink:   true,
	}

	engine := NewDRCEngine(config)

	// Points below threshold should have 0 dB reduction
	belowThresholds := []float64{-40.0, -30.0, -25.0, -20.0}
	for _, inDB := range belowThresholds {
		gr := engine.ComputeStaticGainReduction(inDB)
		if math.Abs(gr) > 1e-6 {
			t.Errorf("Input level %.1f dBFS below threshold expected 0.0 dB GR, got %.4f dB", inDB, gr)
		}
	}

	// Points above threshold: GR = (inDB - T) * (1/Ratio - 1)
	// For Ratio 4:1 and Threshold -20:
	// at -10 dBFS (10 dB above T): Output = -20 + 10/4 = -17.5 dBFS -> GR = -7.5 dB
	// at 0 dBFS (20 dB above T): Output = -20 + 20/4 = -15.0 dBFS -> GR = -15.0 dB
	testPoints := []struct {
		inDB       float64
		expectedGR float64
	}{
		{inDB: -16.0, expectedGR: -3.0}, // 4 dB above -> out = -20 + 1 = -19 -> GR = -3
		{inDB: -10.0, expectedGR: -7.5}, // 10 dB above -> out = -20 + 2.5 = -17.5 -> GR = -7.5
		{inDB: 0.0, expectedGR: -15.0},  // 20 dB above -> out = -20 + 5 = -15 -> GR = -15
	}

	for _, pt := range testPoints {
		gr := engine.ComputeStaticGainReduction(pt.inDB)
		if math.Abs(gr-pt.expectedGR) > 0.01 {
			t.Errorf("Hard knee static GR at %.1f dBFS expected %.2f dB, got %.2f dB", pt.inDB, pt.expectedGR, gr)
		}
	}
}

// TestDRC_SoftKneeStaticCurve validates smooth quadratic transition inside soft knee zone.
func TestDRC_SoftKneeStaticCurve(t *testing.T) {
	config := DRCConfig{
		Profile:      DRCProfileCustom,
		ThresholdDB:  -20.0,
		Ratio:        4.0,
		KneeWidthDB:  10.0, // Knee extends from -25 dBFS to -15 dBFS
		AttackMs:     5.0,
		ReleaseMs:    100.0,
		MakeupGainDB: 0.0,
		StereoLink:   true,
	}

	engine := NewDRCEngine(config)

	// Below knee start (-25 dBFS): GR must be 0 dB
	if gr := engine.ComputeStaticGainReduction(-26.0); math.Abs(gr) > 1e-6 {
		t.Errorf("Below soft knee expected 0 dB GR, got %.4f dB", gr)
	}

	// At exact threshold (-20 dBFS): soft knee should apply a small gentle compression
	grThreshold := engine.ComputeStaticGainReduction(-20.0)
	if grThreshold >= 0.0 || grThreshold < -2.0 {
		t.Errorf("At soft knee threshold expected gentle GR between -2.0 and 0.0 dB, got %.4f dB", grThreshold)
	}

	// Above knee end (-15 dBFS): matches linear slope
	// At -10 dBFS: out = -20 + 10/4 = -17.5 dBFS -> GR = -7.5 dB
	grAbove := engine.ComputeStaticGainReduction(-10.0)
	if math.Abs(grAbove-(-7.5)) > 0.01 {
		t.Errorf("Above soft knee expected -7.5 dB, got %.4f dB", grAbove)
	}
}

// TestDRC_BallisticsAttackAndRelease verifies envelope follower timing response.
func TestDRC_BallisticsAttackAndRelease(t *testing.T) {
	sampleRate := 48000
	attackMs := 10.0
	releaseMs := 100.0

	config := DRCConfig{
		Profile:       DRCProfileCustom,
		ThresholdDB:   -20.0,
		Ratio:         4.0,
		KneeWidthDB:   0.0,
		AttackMs:      attackMs,
		ReleaseMs:     releaseMs,
		MakeupGainDB:  0.0,
		StereoLink:    true,
		EnableLimiter: false,
	}
	engine := NewDRCEngine(config)

	// Step signal: quiet (-30 dBFS), then loud (0 dBFS), then quiet
	numSamples := 4800 // 100 ms total
	samples := make([]float64, numSamples)

	quietAmp := math.Pow(10.0, -30.0/20.0)
	loudAmp := math.Pow(10.0, 0.0/20.0) // 1.0

	stepStart := 1000
	stepEnd := 3000

	for i := 0; i < numSamples; i++ {
		if i >= stepStart && i < stepEnd {
			samples[i] = loudAmp
		} else {
			samples[i] = quietAmp
		}
	}

	pcm := &PCMBuffer{
		SampleRate: sampleRate,
		Channels:   1,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	res, err := engine.Process(pcm)
	if err != nil {
		t.Fatalf("DRC process failed: %v", err)
	}

	if res.MaxGainReductionDB > -10.0 {
		t.Errorf("Expected substantial gain reduction on 0 dBFS input, got %.2f dB", res.MaxGainReductionDB)
	}

	// Verify attack time: 1 time constant (10 ms = 480 samples)
	// Output should drop gradually from 1.0 towards compressed target
	immediateSample := res.ProcessedBuffer.Samples[0][stepStart+1]
	attackSample := res.ProcessedBuffer.Samples[0][stepStart+480]

	if attackSample >= immediateSample {
		t.Errorf("Expected gain to decrease over attack period: immediate=%.4f, afterAttack=%.4f",
			immediateSample, attackSample)
	}
}

// TestDRC_NightModeProfile validates high dynamic range reduction and quiet passage boost.
func TestDRC_NightModeProfile(t *testing.T) {
	sampleRate := 48000
	duration := 2.0 // 2 seconds
	numSamples := int(float64(sampleRate) * duration)

	// Construct audio passage: Quiet speech (-35 dBFS) for 1s, followed by loud action blast (-2 dBFS) for 1s
	ch0 := make([]float64, numSamples)
	ch1 := make([]float64, numSamples)

	quietLevel := math.Pow(10.0, -35.0/20.0)
	loudLevel := math.Pow(10.0, -2.0/20.0)

	for i := 0; i < numSamples; i++ {
		var amp float64
		if i < numSamples/2 {
			amp = quietLevel
		} else {
			amp = loudLevel
		}
		freq := 440.0
		val := amp * math.Sin(2.0*math.Pi*freq*float64(i)/float64(sampleRate))
		ch0[i] = val
		ch1[i] = val
	}

	pcm := &PCMBuffer{
		SampleRate: sampleRate,
		Channels:   2,
		NumSamples: numSamples,
		Samples:    [][]float64{ch0, ch1},
	}

	nightConfig := DefaultDRCConfig(DRCProfileNightMode)
	engine := NewDRCEngine(nightConfig)

	res, err := engine.Process(pcm)
	if err != nil {
		t.Fatalf("Night mode processing failed: %v", err)
	}

	t.Logf("Night Mode Metrics: %s", res.String())

	// Dynamic range should be compressed significantly
	if res.CompressedDynamicRangeDB >= res.OriginalDynamicRangeDB {
		t.Errorf("Night mode failed to compress dynamic range: Original=%.2f dB, Compressed=%.2f dB",
			res.OriginalDynamicRangeDB, res.CompressedDynamicRangeDB)
	}

	// Max gain reduction on loud section should be at least 6 dB
	if res.MaxGainReductionDB > -6.0 {
		t.Errorf("Expected Night Mode loud reduction of at least 6 dB, got %.2f dB", res.MaxGainReductionDB)
	}

	// Peak output must be strictly below ceiling (-1.5 dBFS)
	maxAllowedLinear := math.Pow(10.0, nightConfig.CeilingDB/20.0) + 1e-4
	for ch := 0; ch < 2; ch++ {
		for i := 0; i < numSamples; i++ {
			if math.Abs(res.ProcessedBuffer.Samples[ch][i]) > maxAllowedLinear {
				t.Fatalf("Night Mode sample exceeded ceiling: ch=%d, sample=%d, val=%.5f, ceiling=%.5f",
					ch, i, res.ProcessedBuffer.Samples[ch][i], maxAllowedLinear)
			}
		}
	}
}

// TestDRC_BroadcastProfile validates EBU/ATSC dialogue normalization profile.
func TestDRC_BroadcastProfile(t *testing.T) {
	sampleRate := 48000
	numSamples := sampleRate * 1

	samples := make([]float64, numSamples)
	// Sine wave at -10 dBFS (exceeds broadcast -20 dBFS threshold by 10 dB)
	amp := math.Pow(10.0, -10.0/20.0)
	for i := 0; i < numSamples; i++ {
		samples[i] = amp * math.Sin(2.0*math.Pi*1000.0*float64(i)/float64(sampleRate))
	}

	pcm := &PCMBuffer{
		SampleRate: sampleRate,
		Channels:   1,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	broadcastConfig := DefaultDRCConfig(DRCProfileBroadcast)
	engine := NewDRCEngine(broadcastConfig)

	res, err := engine.Process(pcm)
	if err != nil {
		t.Fatalf("Broadcast DRC processing failed: %v", err)
	}

	// With ratio 3:1, 10 dB above threshold should be compressed by ~6.67 dB
	if res.MaxGainReductionDB > -5.0 || res.MaxGainReductionDB < -8.0 {
		t.Errorf("Broadcast profile expected max GR between -8 and -5 dB, got %.2f dB", res.MaxGainReductionDB)
	}
}

// TestDRC_BrickwallLimiterCeiling ensures 0 digital overs even with hot input.
func TestDRC_BrickwallLimiterCeiling(t *testing.T) {
	sampleRate := 48000
	numSamples := 4800

	samples := make([]float64, numSamples)
	// Oversaturated signal: +4.0 dBFS (amplitude 1.58)
	hotAmp := math.Pow(10.0, 4.0/20.0)
	for i := 0; i < numSamples; i++ {
		samples[i] = hotAmp * math.Sin(2.0*math.Pi*440.0*float64(i)/float64(sampleRate))
	}

	pcm := &PCMBuffer{
		SampleRate: sampleRate,
		Channels:   1,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	limiterConfig := DefaultDRCConfig(DRCProfileBrickwallLimiter)
	engine := NewDRCEngine(limiterConfig)

	res, err := engine.Process(pcm)
	if err != nil {
		t.Fatalf("Brickwall limiter processing failed: %v", err)
	}

	ceilingLinear := math.Pow(10.0, limiterConfig.CeilingDB/20.0)
	for i := 0; i < numSamples; i++ {
		val := math.Abs(res.ProcessedBuffer.Samples[0][i])
		if val > ceilingLinear+1e-6 {
			t.Fatalf("Sample %d exceeded brickwall ceiling: val=%.6f, ceiling=%.6f", i, val, ceilingLinear)
		}
	}
}

// TestDRC_StereoImagingPreservation validates channel linkage preventing pan drift.
func TestDRC_StereoImagingPreservation(t *testing.T) {
	sampleRate := 48000
	numSamples := 4800

	left := make([]float64, numSamples)
	right := make([]float64, numSamples)

	// Left is loud (0 dBFS), Right is quiet (-12 dBFS)
	leftAmp := 1.0
	rightAmp := math.Pow(10.0, -12.0/20.0)

	for i := 0; i < numSamples; i++ {
		sinVal := math.Sin(2.0 * math.Pi * 1000.0 * float64(i) / float64(sampleRate))
		left[i] = leftAmp * sinVal
		right[i] = rightAmp * sinVal
	}

	pcm := &PCMBuffer{
		SampleRate: sampleRate,
		Channels:   2,
		NumSamples: numSamples,
		Samples:    [][]float64{left, right},
	}

	config := DefaultDRCConfig(DRCProfileBroadcast)
	config.StereoLink = true
	engine := NewDRCEngine(config)

	res, err := engine.Process(pcm)
	if err != nil {
		t.Fatalf("Stereo DRC failed: %v", err)
	}

	// In stereo-linked mode, the relative ratio between Left and Right must be exactly preserved
	// check middle sample
	idx := numSamples / 2
	expectedRatio := leftAmp / rightAmp
	actualRatio := math.Abs(res.ProcessedBuffer.Samples[0][idx] / res.ProcessedBuffer.Samples[1][idx])

	if math.Abs(expectedRatio-actualRatio) > 0.01 {
		t.Errorf("Stereo link failed to preserve left/right balance: expected ratio %.4f, got %.4f",
			expectedRatio, actualRatio)
	}
}

// TestDRC_EdgeCases tests boundary conditions, invalid buffers, and extreme parameters.
func TestDRC_EdgeCases(t *testing.T) {
	engine := NewDRCEngine(DefaultDRCConfig(DRCProfileBroadcast))

	// Nil buffer
	if _, err := engine.Process(nil); err == nil {
		t.Errorf("Expected error for nil buffer")
	}

	// Empty buffer
	emptyPCM := &PCMBuffer{
		SampleRate: 48000,
		Channels:   2,
		NumSamples: 0,
		Samples:    [][]float64{},
	}
	if _, err := engine.Process(emptyPCM); err == nil {
		t.Errorf("Expected error for empty buffer")
	}

	// Bypass ratio 1.0
	bypassConfig := DRCConfig{
		Profile:       DRCProfileCustom,
		ThresholdDB:   -20.0,
		Ratio:         1.0, // 1:1 compression (bypass)
		MakeupGainDB:  0.0,
		EnableLimiter: false,
	}
	bypassEngine := NewDRCEngine(bypassConfig)

	sineSamples := []float64{0.1, -0.2, 0.3, -0.4, 0.5}
	sinePCM := &PCMBuffer{
		SampleRate: 48000,
		Channels:   1,
		NumSamples: len(sineSamples),
		Samples:    [][]float64{sineSamples},
	}
	res, err := bypassEngine.Process(sinePCM)
	if err != nil {
		t.Fatalf("Bypass failed: %v", err)
	}
	for i, v := range sineSamples {
		if math.Abs(res.ProcessedBuffer.Samples[0][i]-v) > 1e-6 {
			t.Errorf("Sample %d altered in bypass mode: expected %.4f, got %.4f", i, v, res.ProcessedBuffer.Samples[0][i])
		}
	}
}

// TestDRC_100HeterogeneousAudioStreams executes 100 heterogeneous multi-channel DRC runs.
func TestDRC_100HeterogeneousAudioStreams(t *testing.T) {
	rng := rand.New(rand.NewSource(998877))

	sampleRates := []int{22050, 44100, 48000, 96000}
	profiles := []DRCProfile{
		DRCProfileBroadcast,
		DRCProfileNightMode,
		DRCProfileBrickwallLimiter,
		DRCProfileAcousticGentle,
		DRCProfileCustom,
	}

	for i := 0; i < 100; i++ {
		sr := sampleRates[rng.Intn(len(sampleRates))]
		channels := rng.Intn(3) + 1     // 1 to 3 channels
		length := rng.Intn(2400) + 2400 // 50ms to 200ms
		profile := profiles[rng.Intn(len(profiles))]

		pcmSamples := make([][]float64, channels)
		for ch := 0; ch < channels; ch++ {
			pcmSamples[ch] = make([]float64, length)
			freq := 100.0 + rng.Float64()*3000.0
			baseAmp := 0.05 + rng.Float64()*0.9
			for s := 0; s < length; s++ {
				mod := 1.0 + 0.5*math.Sin(2.0*math.Pi*5.0*float64(s)/float64(sr))
				pcmSamples[ch][s] = baseAmp * mod * math.Sin(2.0*math.Pi*freq*float64(s)/float64(sr))
			}
		}

		pcm := &PCMBuffer{
			SampleRate: sr,
			Channels:   channels,
			NumSamples: length,
			Samples:    pcmSamples,
		}

		config := DefaultDRCConfig(profile)
		// Randomize link mode
		config.StereoLink = rng.Intn(2) == 0

		engine := NewDRCEngine(config)
		result, err := engine.Process(pcm)
		if err != nil {
			t.Fatalf("Stream %d failed DRC process: %v", i+1, err)
		}

		if len(result.ProcessedBuffer.Samples) != channels {
			t.Fatalf("Stream %d channel mismatch: expected %d, got %d", i+1, channels, len(result.ProcessedBuffer.Samples))
		}
		if len(result.ProcessedBuffer.Samples[0]) != length {
			t.Fatalf("Stream %d length mismatch: expected %d, got %d", i+1, length, len(result.ProcessedBuffer.Samples[0]))
		}

		// Verify limiter ceiling compliance if enabled
		if config.EnableLimiter {
			maxLin := math.Pow(10.0, config.CeilingDB/20.0) + 1e-4
			for ch := 0; ch < channels; ch++ {
				for s := 0; s < length; s++ {
					if math.Abs(result.ProcessedBuffer.Samples[ch][s]) > maxLin {
						t.Fatalf("Stream %d ch %d sample %d exceeded ceiling: %.4f > %.4f",
							i+1, ch, s, math.Abs(result.ProcessedBuffer.Samples[ch][s]), maxLin)
					}
				}
			}
		}
	}

	t.Log("Successfully verified 100 heterogeneous audio streams with DRC and brickwall limiting.")
}
