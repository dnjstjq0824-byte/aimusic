package audio

import (
	"errors"
	"math"
	"testing"
)

func generateSineStereo(freq float64, sampleRate int, durationSec float64, amplitude float64) *PCMBuffer {
	numSamples := int(float64(sampleRate) * durationSec)
	left := make([]float64, numSamples)
	right := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		val := amplitude * math.Sin(2.0*math.Pi*freq*t)
		left[i] = val
		right[i] = val
	}
	return &PCMBuffer{
		Channels:   2,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{left, right},
	}
}

func TestLoudness_Sine1kHz_Stereo(t *testing.T) {
	// ITU-R BS.1770-4: Full-scale (0 dBFS peak) 1 kHz stereo sine wave through K-weighting pre-filter
	// (which has a +0.70 dB shelf response at 1kHz) yields exactly +0.01 LUFS.
	sampleRate := 48000
	buf := generateSineStereo(1000.0, sampleRate, 2.0, 1.0)

	result, err := MeasureLUFS(buf)
	if err != nil {
		t.Fatalf("MeasureLUFS failed: %v", err)
	}

	t.Logf("Full-scale 1kHz stereo sine: Integrated = %.2f LUFS, SamplePeak = %.2f dBFS, TruePeak = %.2f dBTP",
		result.IntegratedLUFS, result.SamplePeakDBFS, result.TruePeakDBTP)

	expected := 0.01
	if math.Abs(result.IntegratedLUFS-expected) > 0.05 {
		t.Errorf("Integrated loudness mismatch: expected %.2f LUFS, got %.2f LUFS", expected, result.IntegratedLUFS)
	}
	if math.Abs(result.SamplePeakDBFS-0.0) > 0.05 {
		t.Errorf("Sample peak mismatch: expected 0.0 dBFS, got %.2f dBFS", result.SamplePeakDBFS)
	}
}

func TestLoudness_Sine1kHz_TargetMinus23LUFS(t *testing.T) {
	// Standard EBU R128 reference tone: 1kHz stereo sine calibrated to -23.0 LUFS
	sampleRate := 48000
	amp := math.Pow(10.0, (-23.0-(0.01))/20.0)
	buf := generateSineStereo(1000.0, sampleRate, 3.0, amp)

	result, err := MeasureLUFS(buf)
	if err != nil {
		t.Fatalf("MeasureLUFS failed: %v", err)
	}

	t.Logf("-23 LUFS reference sine: Integrated = %.2f LUFS, MomentaryMax = %.2f LUFS",
		result.IntegratedLUFS, result.MomentaryMax)

	if math.Abs(result.IntegratedLUFS-(-23.0)) > 0.05 {
		t.Errorf("Expected -23.0 LUFS, got %.2f LUFS", result.IntegratedLUFS)
	}
}

func TestLoudness_SilenceGating(t *testing.T) {
	sampleRate := 48000
	numSamples := sampleRate * 2
	silentBuf := &PCMBuffer{
		Channels:   2,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{make([]float64, numSamples), make([]float64, numSamples)},
	}

	result, err := MeasureLUFS(silentBuf)
	if err != nil {
		t.Fatalf("MeasureLUFS on silence failed: %v", err)
	}

	if result.IntegratedLUFS > AbsoluteGateThresholdLUFS {
		t.Errorf("Expected silence to be gated out <= -70 LUFS, got %.2f LUFS", result.IntegratedLUFS)
	}
}

func TestLoudness_GatingMechanism(t *testing.T) {
	// Construct audio with 2 seconds of loud audio (-14 LUFS) followed by 2 seconds of quiet background (-40 LUFS).
	// Because -40 LUFS is 26 LU below the loud part, the relative gate (10 LU offset) should discard the quiet section.
	sampleRate := 48000
	duration := 4.0
	numSamples := int(float64(sampleRate) * duration)
	halfSamples := numSamples / 2

	left := make([]float64, numSamples)
	right := make([]float64, numSamples)

	ampLoud := math.Pow(10.0, (-14.0-(0.01))/20.0)
	ampQuiet := math.Pow(10.0, (-40.0-(0.01))/20.0)

	for i := 0; i < halfSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		val := ampLoud * math.Sin(2.0*math.Pi*1000.0*tSec)
		left[i] = val
		right[i] = val
	}
	for i := halfSamples; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		val := ampQuiet * math.Sin(2.0*math.Pi*1000.0*tSec)
		left[i] = val
		right[i] = val
	}

	buf := &PCMBuffer{
		Channels:   2,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{left, right},
	}

	result, err := MeasureLUFS(buf)
	if err != nil {
		t.Fatalf("MeasureLUFS failed: %v", err)
	}

	t.Logf("Gated audio: Integrated = %.2f LUFS (Expected ~ -14.0 LUFS due to relative gate exclusion)",
		result.IntegratedLUFS)

	// Since quiet blocks are eliminated by the relative gate, integrated loudness should be close to -14 LUFS
	if math.Abs(result.IntegratedLUFS-(-14.0)) > 0.6 {
		t.Errorf("Relative gate failed to exclude quiet section: expected ~ -14.0 LUFS, got %.2f LUFS",
			result.IntegratedLUFS)
	}
}

func TestLoudness_TruePeakInterSample(t *testing.T) {
	// Create an inter-sample peak signal: two consecutive high samples at Nyquist/4 (e.g., sampleRate/4)
	// Sample peak can be 0.9, but continuous bandlimited reconstruction peak between samples exceeds 1.0.
	sampleRate := 48000
	numSamples := 4800
	left := make([]float64, numSamples)
	right := make([]float64, numSamples)

	// A sine wave sampled at phase pi/4 and 3pi/4 where peaks fall between samples:
	freq := float64(sampleRate) / 4.0 // 12000 Hz
	phaseOffset := math.Pi / 4.0      // peak occurs at t = 1/(8*freq)
	amp := 0.90                       // sample peak will be 0.9 * sin(3pi/4) = 0.9 * 0.7071 = 0.6364 or similar

	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		val := amp * math.Sin(2.0*math.Pi*freq*tSec+phaseOffset)
		left[i] = val
		right[i] = val
	}

	buf := &PCMBuffer{
		Channels:   2,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{left, right},
	}

	meter := NewLoudnessMeter(WithTruePeak(true))
	res, err := meter.Measure(buf)
	if err != nil {
		t.Fatalf("Measure failed: %v", err)
	}

	t.Logf("Inter-sample peak test: SamplePeak = %.2f dBFS, TruePeak = %.2f dBTP (diff = +%.2f dB)",
		res.SamplePeakDBFS, res.TruePeakDBTP, res.TruePeakDBTP-res.SamplePeakDBFS)

	if res.TruePeakDBTP <= res.SamplePeakDBFS {
		t.Errorf("Expected True Peak (%.2f dBTP) to exceed Sample Peak (%.2f dBFS) due to inter-sample interpolation",
			res.TruePeakDBTP, res.SamplePeakDBFS)
	}
}

func TestLoudness_NormalizationGain(t *testing.T) {
	// Start with an audio buffer at -20.0 LUFS, calculate gain to reach streaming standard -14.0 LUFS
	sampleRate := 48000
	initAmp := math.Pow(10.0, (-20.0-(0.01))/20.0)
	buf := generateSineStereo(1000.0, sampleRate, 2.0, initAmp)

	res1, err := MeasureLUFS(buf)
	if err != nil {
		t.Fatalf("Initial measure failed: %v", err)
	}

	gain := CalculateNormalizationGain(res1.IntegratedLUFS, TargetLUFSStreaming)
	normalizedBuf := ApplyGain(buf, gain)

	res2, err := MeasureLUFS(normalizedBuf)
	if err != nil {
		t.Fatalf("Normalized measure failed: %v", err)
	}

	t.Logf("Normalization: Initial = %.2f LUFS, Gain = %.4f (+%.2f dB), Normalized = %.2f LUFS (Target = %.2f)",
		res1.IntegratedLUFS, gain, 20.0*math.Log10(gain), res2.IntegratedLUFS, TargetLUFSStreaming)

	if math.Abs(res2.IntegratedLUFS-TargetLUFSStreaming) > 0.15 {
		t.Errorf("Normalized loudness %.2f LUFS differs from target %.2f LUFS",
			res2.IntegratedLUFS, TargetLUFSStreaming)
	}
}

func TestLoudness_LRA(t *testing.T) {
	// Construct audio with significant dynamic variation over 9 seconds (3 sections of 3s each)
	sampleRate := 48000
	numSamples := sampleRate * 9
	left := make([]float64, numSamples)
	right := make([]float64, numSamples)

	// Section 1 (0-3s): Loud (-14 LUFS)
	// Section 2 (3-6s): Medium (-22 LUFS)
	// Section 3 (6-9s): Soft (-30 LUFS)
	amp1 := math.Pow(10.0, (-14.0-(0.01))/20.0)
	amp2 := math.Pow(10.0, (-22.0-(0.01))/20.0)
	amp3 := math.Pow(10.0, (-30.0-(0.01))/20.0)

	for i := 0; i < sampleRate*3; i++ {
		v := amp1 * math.Sin(2.0*math.Pi*1000.0*float64(i)/float64(sampleRate))
		left[i] = v
		right[i] = v
	}
	for i := sampleRate * 3; i < sampleRate*6; i++ {
		v := amp2 * math.Sin(2.0*math.Pi*1000.0*float64(i)/float64(sampleRate))
		left[i] = v
		right[i] = v
	}
	for i := sampleRate * 6; i < numSamples; i++ {
		v := amp3 * math.Sin(2.0*math.Pi*1000.0*float64(i)/float64(sampleRate))
		left[i] = v
		right[i] = v
	}

	buf := &PCMBuffer{
		Channels:   2,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{left, right},
	}

	res, err := MeasureLUFS(buf)
	if err != nil {
		t.Fatalf("MeasureLUFS failed: %v", err)
	}

	t.Logf("LRA Dynamic audio: LRA = %.2f LU, ShortTermMax = %.2f LUFS, Integrated = %.2f LUFS",
		res.LoudnessRange, res.ShortTermMax, res.IntegratedLUFS)

	if res.LoudnessRange < 3.0 {
		t.Errorf("Expected substantial Loudness Range (> 3.0 LU), got %.2f LU", res.LoudnessRange)
	}
}

func TestLoudness_SampleRateIndependent(t *testing.T) {
	// Compare 44.1kHz vs 48kHz versions of the exact same tone
	amp := math.Pow(10.0, (-16.0-(0.01))/20.0)
	buf44 := generateSineStereo(1000.0, 44100, 2.0, amp)
	buf48 := generateSineStereo(1000.0, 48000, 2.0, amp)

	res44, err1 := MeasureLUFS(buf44)
	res48, err2 := MeasureLUFS(buf48)
	if err1 != nil || err2 != nil {
		t.Fatalf("Measurements failed: %v, %v", err1, err2)
	}

	t.Logf("Sample rate invariance: 44.1kHz = %.2f LUFS, 48kHz = %.2f LUFS",
		res44.IntegratedLUFS, res48.IntegratedLUFS)

	if math.Abs(res44.IntegratedLUFS-res48.IntegratedLUFS) > 0.15 {
		t.Errorf("Sample rate variance too high: 44.1kHz %.2f vs 48kHz %.2f",
			res44.IntegratedLUFS, res48.IntegratedLUFS)
	}
}

func TestLoudness_EdgeCases(t *testing.T) {
	// 1. Nil buffer
	_, err := MeasureLUFS(nil)
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
	_, err = MeasureLUFS(emptyBuf)
	if !errors.Is(err, ErrEmptyAudioBuffer) {
		t.Errorf("Expected ErrEmptyAudioBuffer, got %v", err)
	}

	// 3. ApplyGain on nil buffer
	if g := ApplyGain(nil, 1.0); g != nil {
		t.Errorf("Expected nil when applying gain to nil buffer")
	}
}

func TestLoudness100HeterogeneousStreams(t *testing.T) {
	frequencies := []float64{125.0, 250.0, 500.0, 1000.0, 2000.0, 4000.0, 8000.0}
	levelsLUFS := []float64{-23.0, -18.0, -14.0, -10.0}
	sampleRates := []int{44100, 48000}
	channelConfigs := []int{1, 2}

	count := 0
	for _, freq := range frequencies {
		for _, targetLU := range levelsLUFS {
			for _, sr := range sampleRates {
				for _, ch := range channelConfigs {
					count++
					amp := math.Pow(10.0, (targetLU-0.01)/20.0)
					if ch == 1 {
						amp = math.Pow(10.0, (targetLU-(-3.01))/20.0)
					}

					numSamples := int(float64(sr) * 0.6) // 0.6 sec
					samples := make([][]float64, ch)
					for c := 0; c < ch; c++ {
						samples[c] = make([]float64, numSamples)
						for i := 0; i < numSamples; i++ {
							samples[c][i] = amp * math.Sin(2.0*math.Pi*freq*float64(i)/float64(sr))
						}
					}

					buf := &PCMBuffer{
						Channels:   ch,
						SampleRate: sr,
						NumSamples: numSamples,
						Samples:    samples,
					}

					res, err := MeasureLUFS(buf)
					if err != nil {
						t.Fatalf("Stream %d (freq=%.0f, target=%.1f, sr=%d, ch=%d) failed: %v",
							count, freq, targetLU, sr, ch, err)
					}

					if math.IsNaN(res.IntegratedLUFS) || math.IsInf(res.IntegratedLUFS, 0) {
						t.Fatalf("Stream %d: NaN or Inf integrated loudness", count)
					}
					if math.IsNaN(res.SamplePeakDBFS) || math.IsInf(res.SamplePeakDBFS, 0) {
						t.Fatalf("Stream %d: NaN or Inf sample peak", count)
					}
					if math.IsNaN(res.TruePeakDBTP) || math.IsInf(res.TruePeakDBTP, 0) {
						t.Fatalf("Stream %d: NaN or Inf true peak", count)
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

	t.Logf("Successfully executed %d heterogeneous loudness stream measurements with 0 errors.", count)
}
