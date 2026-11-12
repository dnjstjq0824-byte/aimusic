package audio

import (
	"math"
	"math/rand"
	"testing"
)

func TestSilenceTrimmer_LeadInAndLeadOutTrimming(t *testing.T) {
	sr := 48000
	channels := 2

	leadInSilenceSec := 2.0
	activeAudioSec := 5.0
	leadOutSilenceSec := 3.0

	leadInSamples := int(leadInSilenceSec * float64(sr))
	activeSamples := int(activeAudioSec * float64(sr))
	leadOutSamples := int(leadOutSilenceSec * float64(sr))
	totalSamples := leadInSamples + activeSamples + leadOutSamples

	pcm := createSyntheticPCMBuffer(sr, channels, totalSamples, func(s, c int) float64 {
		if s < leadInSamples || s >= (leadInSamples+activeSamples) {
			return 0.0 // Digital silence
		}
		tSec := float64(s-leadInSamples) / float64(sr)
		return 0.8 * math.Sin(2.0*math.Pi*440.0*tSec)
	})

	cfg := SilenceTrimmerConfig{
		ThresholdDB:       -60.0,
		LeadInMarginMs:    50.0,  // 0.05s
		LeadOutMarginMs:   100.0, // 0.10s
		FadeInMs:          20.0,
		FadeOutMs:         50.0,
		Curve:             FadeCurveSCurve,
		AlignZeroCrossing: true,
		WindowMs:          10.0,
	}

	trimmer := NewSilenceTrimmer(cfg)
	res, err := trimmer.Trim(pcm)
	if err != nil {
		t.Fatalf("Trim failed: %v", err)
	}

	// Original duration = 10.0s
	if math.Abs(res.OriginalDurationSec-10.0) > 0.01 {
		t.Errorf("original duration expected 10.0s, got %f", res.OriginalDurationSec)
	}

	// Expected lead-in trimmed: ~ 2.0 - 0.05 = 1.95s
	if math.Abs(res.LeadInTrimmedSec-1.95) > 0.05 {
		t.Errorf("lead-in trimmed expected ~1.95s, got %f", res.LeadInTrimmedSec)
	}

	// Expected lead-out trimmed: ~ 3.0 - 0.10 = 2.90s
	if math.Abs(res.LeadOutTrimmedSec-2.90) > 0.05 {
		t.Errorf("lead-out trimmed expected ~2.90s, got %f", res.LeadOutTrimmedSec)
	}

	// Total trimmed duration should be ~ 5.15s (5.0s active + 0.05s + 0.10s margins)
	if math.Abs(res.TrimmedDurationSec-5.15) > 0.08 {
		t.Errorf("trimmed duration expected ~5.15s, got %f", res.TrimmedDurationSec)
	}

	// Verify silence removed > 4.7s
	if res.SilenceRemovedSec < 4.7 {
		t.Errorf("expected >4.7s silence removed, got %f", res.SilenceRemovedSec)
	}

	// Check that fade-in ramps from zero
	trimmedBuf := res.TrimmedBuffer
	for c := 0; c < channels; c++ {
		firstSample := trimmedBuf.Samples[c][0]
		if math.Abs(firstSample) > 0.001 {
			t.Errorf("channel %d first sample after fade-in expected ~0.0, got %f", c, firstSample)
		}
		lastSample := trimmedBuf.Samples[c][trimmedBuf.NumSamples-1]
		if math.Abs(lastSample) > 0.001 {
			t.Errorf("channel %d last sample after fade-out expected ~0.0, got %f", c, lastSample)
		}
	}
}

func TestFadeCurves_MathematicalWeights(t *testing.T) {
	curves := []FadeCurve{
		FadeCurveLinear,
		FadeCurveEqualPower,
		FadeCurveSCurve,
		FadeCurveExponential,
	}

	for _, c := range curves {
		// Fade in boundaries
		w0 := CalculateCurveWeight(0.0, c, true)
		if math.Abs(w0-0.0) > 1e-6 {
			t.Errorf("curve %s: w(0) fade-in expected 0.0, got %f", c, w0)
		}
		w1 := CalculateCurveWeight(1.0, c, true)
		if math.Abs(w1-1.0) > 1e-6 {
			t.Errorf("curve %s: w(1) fade-in expected 1.0, got %f", c, w1)
		}

		// Fade out boundaries
		wOut0 := CalculateCurveWeight(0.0, c, false)
		if math.Abs(wOut0-1.0) > 1e-6 {
			t.Errorf("curve %s: w(0) fade-out expected 1.0, got %f", c, wOut0)
		}
		wOut1 := CalculateCurveWeight(1.0, c, false)
		if math.Abs(wOut1-0.0) > 1e-6 {
			t.Errorf("curve %s: w(1) fade-out expected 0.0, got %f", c, wOut1)
		}
	}

	// Verify Equal-Power constant acoustic power preservation:
	// w_in(p)^2 + w_out(p)^2 == 1.0 for all p in [0.0, 1.0]
	for step := 0; step <= 10; step++ {
		p := float64(step) / 10.0
		inGain := CalculateCurveWeight(p, FadeCurveEqualPower, true)
		outGain := CalculateCurveWeight(p, FadeCurveEqualPower, false)
		totalPower := inGain*inGain + outGain*outGain
		if math.Abs(totalPower-1.0) > 1e-5 {
			t.Errorf("equal power violated at p=%.1f: power=%f", p, totalPower)
		}
	}

	// S-curve midpoint Hann property: w(0.5) == 0.5
	sMid := CalculateCurveWeight(0.5, FadeCurveSCurve, true)
	if math.Abs(sMid-0.5) > 1e-6 {
		t.Errorf("s-curve midpoint expected 0.5, got %f", sMid)
	}
}

func TestZeroCrossing_Alignment(t *testing.T) {
	sr := 48000
	// 400 Hz tone -> period is 120 samples, zero crossings occur at 0, 60, 120, 180, ...
	samples := make([]float64, 480)
	for i := range samples {
		samples[i] = math.Sin(2.0 * math.Pi * 400.0 * float64(i) / float64(sr))
	}

	// Pick a non-zero crossing point at i=80 (peak is at i=30, 90 is negative peak)
	targetIdx := 80
	zeroIdx := FindZeroCrossing(samples, targetIdx, 30) // Search radius 30 covers [50, 110], containing zero crossing at 60

	if zeroIdx < 1 || zeroIdx >= len(samples) {
		t.Fatalf("zero crossing index out of bounds: %d", zeroIdx)
	}
	sPrev := samples[zeroIdx-1]
	sCurr := samples[zeroIdx]
	isCross := (sPrev <= 0 && sCurr >= 0) || (sPrev >= 0 && sCurr <= 0) || math.Abs(sCurr) < 0.05
	if !isCross {
		t.Errorf("found index %d is not a valid zero crossing: s[i-1]=%f, s[i]=%f", zeroIdx, sPrev, sCurr)
	}
}

func TestApplyFadeInAndOut(t *testing.T) {
	sr := 48000
	numSamples := 4800 // 100ms
	pcm := createSyntheticPCMBuffer(sr, 1, numSamples, func(s, c int) float64 {
		return 1.0 // Constant unity amplitude
	})

	// 20ms fade-in, 30ms fade-out
	err := ApplyFadeIn(pcm, 20.0, FadeCurveSCurve)
	if err != nil {
		t.Fatalf("ApplyFadeIn failed: %v", err)
	}
	err = ApplyFadeOut(pcm, 30.0, FadeCurveSCurve)
	if err != nil {
		t.Fatalf("ApplyFadeOut failed: %v", err)
	}

	// Sample 0 should be 0.0
	if math.Abs(pcm.Samples[0][0]-0.0) > 1e-6 {
		t.Errorf("sample 0 expected 0.0, got %f", pcm.Samples[0][0])
	}

	// Sample at 50ms (index 2400) should be untouched 1.0
	midIdx := 2400
	if math.Abs(pcm.Samples[0][midIdx]-1.0) > 1e-6 {
		t.Errorf("midpoint sample expected 1.0, got %f", pcm.Samples[0][midIdx])
	}

	// Last sample should be 0.0
	lastIdx := numSamples - 1
	if math.Abs(pcm.Samples[0][lastIdx]-0.0) > 1e-6 {
		t.Errorf("last sample expected 0.0, got %f", pcm.Samples[0][lastIdx])
	}
}

func TestSilenceTrimmer_EdgeCases(t *testing.T) {
	trimmer := NewSilenceTrimmer(DefaultTrimmerConfig())

	// Nil buffer
	_, err := trimmer.Trim(nil)
	if err != ErrNilTrimmerBuffer {
		t.Errorf("expected ErrNilTrimmerBuffer, got %v", err)
	}

	// Empty buffer
	emptyBuf := &PCMBuffer{Channels: 2, SampleRate: 48000, NumSamples: 0, Samples: [][]float64{{}, {}}}
	_, err = trimmer.Trim(emptyBuf)
	if err != ErrEmptyAudioBuffer {
		t.Errorf("expected ErrEmptyAudioBuffer, got %v", err)
	}

	// Pure silence (all zeros)
	silentBuf := createSyntheticPCMBuffer(48000, 1, 4800, func(s, c int) float64 { return 0.0 })
	_, err = trimmer.Trim(silentBuf)
	if err != ErrAllSilentAudio {
		t.Errorf("expected ErrAllSilentAudio, got %v", err)
	}

	// Negative fade param
	pcm := createSyntheticPCMBuffer(48000, 1, 100, func(s, c int) float64 { return 0.5 })
	err = ApplyFadeIn(pcm, -10.0, FadeCurveLinear)
	if err != ErrInvalidFadeParam {
		t.Errorf("expected ErrInvalidFadeParam, got %v", err)
	}
	err = ApplyFadeOut(pcm, -10.0, FadeCurveLinear)
	if err != ErrInvalidFadeParam {
		t.Errorf("expected ErrInvalidFadeParam, got %v", err)
	}
}

func TestSilenceTrimmer_100HeterogeneousAudioStreams(t *testing.T) {
	rng := rand.New(rand.NewSource(20261109))
	totalTests := 100

	sampleRates := []int{8000, 16000, 22050, 32000, 44100, 48000, 96000}
	channelsList := []int{1, 2, 4}
	curves := []FadeCurve{FadeCurveSCurve, FadeCurveEqualPower, FadeCurveLinear, FadeCurveExponential}

	for iter := 0; iter < totalTests; iter++ {
		sr := sampleRates[rng.Intn(len(sampleRates))]
		ch := channelsList[rng.Intn(len(channelsList))]
		curve := curves[rng.Intn(len(curves))]

		leadInSec := 0.2 + rng.Float64()*1.5  // 0.2s to 1.7s silence
		activeSec := 0.5 + rng.Float64()*2.0  // 0.5s to 2.5s active
		leadOutSec := 0.2 + rng.Float64()*1.5 // 0.2s to 1.7s silence

		leadInSamples := int(leadInSec * float64(sr))
		activeSamples := int(activeSec * float64(sr))
		leadOutSamples := int(leadOutSec * float64(sr))
		totalSamples := leadInSamples + activeSamples + leadOutSamples

		freq := 200.0 + rng.Float64()*1000.0
		amp := 0.1 + rng.Float64()*0.85

		pcm := createSyntheticPCMBuffer(sr, ch, totalSamples, func(s, c int) float64 {
			if s < leadInSamples || s >= (leadInSamples+activeSamples) {
				return 0.0
			}
			tSec := float64(s-leadInSamples) / float64(sr)
			return amp * math.Sin(2.0*math.Pi*freq*tSec)
		})

		cfg := SilenceTrimmerConfig{
			ThresholdDB:       -54.0,
			LeadInMarginMs:    20.0,
			LeadOutMarginMs:   30.0,
			FadeInMs:          10.0,
			FadeOutMs:         15.0,
			Curve:             curve,
			AlignZeroCrossing: true,
			WindowMs:          5.0,
		}

		trimmer := NewSilenceTrimmer(cfg)
		res, err := trimmer.Trim(pcm)
		if err != nil {
			t.Fatalf("iteration %d: Trim failed: %v", iter, err)
		}

		if res.TrimmedBuffer.NumSamples <= 0 {
			t.Fatalf("iteration %d: trimmed buffer is empty", iter)
		}
		if res.TrimmedDurationSec >= res.OriginalDurationSec {
			t.Fatalf("iteration %d: expected trimmed duration < original duration", iter)
		}
		if res.SilenceRemovedSec <= 0.0 {
			t.Fatalf("iteration %d: expected positive silence removed", iter)
		}
	}

	t.Logf("Successfully verified 100 heterogeneous audio streams with silence trimming and smart fading.")
}
