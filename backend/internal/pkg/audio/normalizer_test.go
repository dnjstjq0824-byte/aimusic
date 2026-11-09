package audio

import (
	"errors"
	"math"
	"testing"
)

func TestNormalize_StreamingStandard_Linear(t *testing.T) {
	// Source audio: 1kHz stereo sine at -20.0 LUFS (peak ~ -6 dBFS)
	sampleRate := 48000
	amp := math.Pow(10.0, (-20.0-0.01)/20.0) // ~0.09988
	buf := generateSineStereo(1000.0, sampleRate, 2.0, amp)

	norm, err := NewNormalizer(WithProfile(ProfileStreamingStandard))
	if err != nil {
		t.Fatalf("NewNormalizer failed: %v", err)
	}

	outBuf, report, err := norm.Normalize(buf)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	t.Logf("Streaming Normalization: Original = %.2f LUFS, Final = %.2f LUFS, Gain = +%.2f dB, FinalTP = %.2f dBTP",
		report.OriginalLUFS, report.FinalLUFS, report.AppliedGainDB, report.FinalTruePeak)

	if math.Abs(report.FinalLUFS-TargetLUFSStreaming) > 0.15 {
		t.Errorf("Final LUFS %.2f differs from target %.2f LUFS", report.FinalLUFS, TargetLUFSStreaming)
	}
	if report.ClippingPrevented {
		t.Errorf("Expected linear gain without clipping prevention, got clippingPrevented=true")
	}
	if report.FinalTruePeak > norm.Config().CeilingDBTP+0.05 {
		t.Errorf("Final True Peak %.2f exceeds ceiling %.2f dBTP", report.FinalTruePeak, norm.Config().CeilingDBTP)
	}
	if len(outBuf.Samples[0]) != buf.NumSamples {
		t.Errorf("Sample count altered: expected %d, got %d", buf.NumSamples, len(outBuf.Samples[0]))
	}
}

func TestNormalize_PeakBackoff_PreventClipping(t *testing.T) {
	// Create audio with a transient peak that would clip if boosted by full 6dB:
	// Ambient tone is -20 LUFS, but has a burst with peak at -2.0 dBFS
	sampleRate := 48000
	numSamples := sampleRate * 2
	left := make([]float64, numSamples)
	right := make([]float64, numSamples)

	baseAmp := math.Pow(10.0, (-20.0-0.01)/20.0)
	for i := 0; i < numSamples; i++ {
		v := baseAmp * math.Sin(2.0*math.Pi*1000.0*float64(i)/float64(sampleRate))
		left[i] = v
		right[i] = v
	}
	// Insert short transient spike with peak at -2.0 dBFS (~0.794)
	spikeAmp := math.Pow(10.0, -2.0/20.0)
	spikeIdx := sampleRate / 2
	for k := 0; k < 48; k++ {
		v := spikeAmp * math.Sin(2.0*math.Pi*500.0*float64(k)/float64(sampleRate))
		left[spikeIdx+k] = v
		right[spikeIdx+k] = v
	}

	buf := &PCMBuffer{
		Channels:   2,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{left, right},
	}

	norm, err := NewNormalizer(
		WithTargetLUFS(-14.0),
		WithCeilingDBTP(-1.0),
		WithPreventClipping(true),
		WithLookaheadLimiter(false), // Backoff mode
	)
	if err != nil {
		t.Fatalf("NewNormalizer failed: %v", err)
	}

	_, report, err := norm.Normalize(buf)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	t.Logf("Peak backoff: OriginalLUFS = %.2f, Target = -14.0, AppliedGain = %.2f dB, FinalTruePeak = %.2f dBTP (Ceiling = -1.0)",
		report.OriginalLUFS, report.AppliedGainDB, report.FinalTruePeak)

	if !report.ClippingPrevented {
		t.Errorf("Expected clipping prevention backoff to engage")
	}
	if report.FinalTruePeak > -1.0+0.05 {
		t.Errorf("Final true peak %.2f dBTP exceeded ceiling -1.0 dBTP", report.FinalTruePeak)
	}
}

func TestNormalize_LookaheadLimiter_Engaged(t *testing.T) {
	// Same audio as above, but with lookahead limiter enabled
	sampleRate := 48000
	numSamples := sampleRate * 2
	left := make([]float64, numSamples)
	right := make([]float64, numSamples)

	baseAmp := math.Pow(10.0, (-20.0-0.01)/20.0)
	for i := 0; i < numSamples; i++ {
		v := baseAmp * math.Sin(2.0*math.Pi*1000.0*float64(i)/float64(sampleRate))
		left[i] = v
		right[i] = v
	}
	spikeAmp := math.Pow(10.0, -2.0/20.0)
	spikeIdx := sampleRate / 2
	for k := 0; k < 96; k++ {
		v := spikeAmp * math.Sin(2.0*math.Pi*500.0*float64(k)/float64(sampleRate))
		left[spikeIdx+k] = v
		right[spikeIdx+k] = v
	}

	buf := &PCMBuffer{
		Channels:   2,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{left, right},
	}

	norm, err := NewNormalizer(
		WithTargetLUFS(-14.0),
		WithCeilingDBTP(-1.0),
		WithPreventClipping(true),
		WithLookaheadLimiter(true),
	)
	if err != nil {
		t.Fatalf("NewNormalizer failed: %v", err)
	}

	_, report, err := norm.Normalize(buf)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	t.Logf("Lookahead Limiter: FinalLUFS = %.2f (Target = -14.0), FinalTruePeak = %.2f dBTP (Ceiling = -1.0), LimiterEngaged = %v",
		report.FinalLUFS, report.FinalTruePeak, report.LimiterEngaged)

	if !report.LimiterEngaged {
		t.Errorf("Expected lookahead limiter to be engaged")
	}
	if report.FinalTruePeak > -1.0+0.05 {
		t.Errorf("Final true peak %.2f dBTP exceeded ceiling -1.0 dBTP with limiter", report.FinalTruePeak)
	}
	if math.Abs(report.FinalLUFS-(-14.0)) > 0.40 {
		t.Errorf("Final LUFS %.2f not close to target -14.0 LUFS with limiter", report.FinalLUFS)
	}
}

func TestNormalize_EBUR128_Attenuation(t *testing.T) {
	// Loud source audio at -12.0 LUFS, target EBU R128 (-23.0 LUFS) -> attenuation of ~11.0 dB
	sampleRate := 48000
	amp := math.Pow(10.0, (-12.0-0.01)/20.0)
	buf := generateSineStereo(1000.0, sampleRate, 2.0, amp)

	outBuf, report, err := NormalizeToTarget(buf, TargetLUFSEBUR128, -1.0)
	if err != nil {
		t.Fatalf("NormalizeToTarget failed: %v", err)
	}

	t.Logf("EBU R128 Attenuation: Original = %.2f LUFS, Final = %.2f LUFS, AppliedGain = %.2f dB",
		report.OriginalLUFS, report.FinalLUFS, report.AppliedGainDB)

	if math.Abs(report.FinalLUFS-TargetLUFSEBUR128) > 0.15 {
		t.Errorf("Expected -23.0 LUFS, got %.2f LUFS", report.FinalLUFS)
	}
	if report.AppliedGainDB >= 0.0 {
		t.Errorf("Expected negative gain for downward normalization, got %.2f dB", report.AppliedGainDB)
	}
	if report.FinalTruePeak > report.OriginalTruePeak {
		t.Errorf("True peak should decrease under attenuation: orig %.2f, final %.2f",
			report.OriginalTruePeak, report.FinalTruePeak)
	}
	_ = outBuf
}

func TestNormalize_AppleMusic_Profile(t *testing.T) {
	sampleRate := 48000
	amp := math.Pow(10.0, (-22.0-0.01)/20.0)
	buf := generateSineStereo(1000.0, sampleRate, 2.0, amp)

	norm, err := NewNormalizer(WithProfile(ProfileAppleMusic))
	if err != nil {
		t.Fatalf("NewNormalizer failed: %v", err)
	}

	_, report, err := norm.Normalize(buf)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	t.Logf("Apple Music Profile: FinalLUFS = %.2f (Target = %.2f)", report.FinalLUFS, TargetLUFSAppleMusic)
	if math.Abs(report.FinalLUFS-TargetLUFSAppleMusic) > 0.15 {
		t.Errorf("Expected Apple Music target %.2f LUFS, got %.2f LUFS", TargetLUFSAppleMusic, report.FinalLUFS)
	}
}

func TestNormalize_StereoPreservation(t *testing.T) {
	sampleRate := 48000
	numSamples := sampleRate * 2
	left := make([]float64, numSamples)
	right := make([]float64, numSamples)

	amp := math.Pow(10.0, (-20.0-0.01)/20.0)
	for i := 0; i < numSamples; i++ {
		left[i] = amp * math.Sin(2.0*math.Pi*440.0*float64(i)/float64(sampleRate))
		right[i] = amp * math.Sin(2.0*math.Pi*880.0*float64(i)/float64(sampleRate))
	}

	buf := &PCMBuffer{
		Channels:   2,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{left, right},
	}

	outBuf, report, err := NormalizeForStreaming(buf)
	if err != nil {
		t.Fatalf("NormalizeForStreaming failed: %v", err)
	}

	// Verify both channels scaled by the exact same linear gain
	scaleFactor := report.AppliedGainLinear
	for i := 0; i < 100; i++ {
		expectedL := left[i] * scaleFactor
		expectedR := right[i] * scaleFactor
		if math.Abs(outBuf.Samples[0][i]-expectedL) > 1e-10 {
			t.Fatalf("Left channel scaling discrepancy at %d", i)
		}
		if math.Abs(outBuf.Samples[1][i]-expectedR) > 1e-10 {
			t.Fatalf("Right channel scaling discrepancy at %d", i)
		}
	}
}

func TestNormalize_EdgeCases(t *testing.T) {
	// 1. Nil buffer
	norm, _ := NewNormalizer()
	_, _, err := norm.Normalize(nil)
	if !errors.Is(err, ErrNilLoudnessBuffer) {
		t.Errorf("Expected ErrNilLoudnessBuffer, got %v", err)
	}

	// 2. Empty buffer
	emptyBuf := &PCMBuffer{
		Channels:   0,
		SampleRate: 48000,
		NumSamples: 0,
		Samples:    nil,
	}
	_, _, err = norm.Normalize(emptyBuf)
	if !errors.Is(err, ErrEmptyAudioBuffer) {
		t.Errorf("Expected ErrEmptyAudioBuffer, got %v", err)
	}

	// 3. Invalid ceiling (> 0.0 dBTP)
	_, err = NewNormalizer(WithCeilingDBTP(1.5))
	if !errors.Is(err, ErrInvalidCeiling) {
		t.Errorf("Expected ErrInvalidCeiling, got %v", err)
	}

	// 4. Silence below threshold
	silentBuf := &PCMBuffer{
		Channels:   1,
		SampleRate: 48000,
		NumSamples: 48000,
		Samples:    [][]float64{make([]float64, 48000)},
	}
	_, _, err = norm.Normalize(silentBuf)
	if !errors.Is(err, ErrUnmeasurableAudio) {
		t.Errorf("Expected ErrUnmeasurableAudio for silent input, got %v", err)
	}
}

func TestNormalize100HeterogeneousTracks(t *testing.T) {
	frequencies := []float64{220.0, 440.0, 880.0, 1760.0}
	initialLevels := []float64{-26.0, -22.0, -18.0, -15.0, -10.0}
	targets := []float64{-23.0, -16.0, -14.0}
	channels := []int{1, 2}

	count := 0
	for _, freq := range frequencies {
		for _, initLU := range initialLevels {
			for _, targetLU := range targets {
				for _, ch := range channels {
					count++
					sampleRate := 48000
					amp := math.Pow(10.0, (initLU-0.01)/20.0)
					if ch == 1 {
						amp = math.Pow(10.0, (initLU-(-3.01))/20.0)
					}

					numSamples := int(float64(sampleRate) * 0.5) // 0.5 sec
					samples := make([][]float64, ch)
					for c := 0; c < ch; c++ {
						samples[c] = make([]float64, numSamples)
						for i := 0; i < numSamples; i++ {
							samples[c][i] = amp * math.Sin(2.0*math.Pi*freq*float64(i)/float64(sampleRate))
						}
					}

					buf := &PCMBuffer{
						Channels:   ch,
						SampleRate: sampleRate,
						NumSamples: numSamples,
						Samples:    samples,
					}

					norm, err := NewNormalizer(WithTargetLUFS(targetLU), WithCeilingDBTP(-1.0))
					if err != nil {
						t.Fatalf("Track %d: NewNormalizer failed: %v", count, err)
					}

					_, report, err := norm.Normalize(buf)
					if err != nil {
						t.Fatalf("Track %d (freq=%.0f, init=%.1f, target=%.1f, ch=%d) failed: %v",
							count, freq, initLU, targetLU, ch, err)
					}

					if math.IsNaN(report.FinalLUFS) || math.IsInf(report.FinalLUFS, 0) {
						t.Fatalf("Track %d: NaN/Inf final LUFS", count)
					}
					if report.FinalTruePeak > -1.0+0.1 {
						t.Fatalf("Track %d: True peak %.2f dBTP exceeded ceiling -1.0 dBTP", count, report.FinalTruePeak)
					}

					if count >= 100 {
						break
					}
				}
				if count >= 100 {
					break
				}
			}
			if count >= 100 {
				break
			}
		}
		if count >= 100 {
			break
		}
	}

	t.Logf("Successfully executed %d heterogeneous track normalizations with 0 errors.", count)
}
