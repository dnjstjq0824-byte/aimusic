package audio

import (
	"errors"
	"math"
	"testing"
)

func TestBesselI0(t *testing.T) {
	cases := []struct {
		x        float64
		expected float64
	}{
		{0.0, 1.0},
		{1.0, 1.2660658777520084},
		{2.0, 2.2795853023360673},
		{5.0, 27.239871823670984},
		{-1.0, 1.2660658777520084},
		{-2.0, 2.2795853023360673},
	}

	for _, tc := range cases {
		actual := BesselI0(tc.x)
		diff := math.Abs(actual - tc.expected)
		t.Logf("BesselI0(%f): actual=%.15f expected=%.15f diff=%e", tc.x, actual, tc.expected, diff)
		if diff > 1e-10 {
			t.Errorf("BesselI0(%f) = %v; expected %v (diff %e)", tc.x, actual, tc.expected, diff)
		}
	}
}

func TestSinc(t *testing.T) {
	if s := Sinc(0.0); s != 1.0 {
		t.Errorf("Sinc(0) expected 1.0, got %f", s)
	}
	if s := Sinc(1.0); math.Abs(s) > 1e-15 {
		t.Errorf("Sinc(1) expected 0.0, got %e", s)
	}
	if s := Sinc(2.0); math.Abs(s) > 1e-15 {
		t.Errorf("Sinc(2) expected 0.0, got %e", s)
	}
	expectedHalf := 2.0 / math.Pi
	if s := Sinc(0.5); math.Abs(s-expectedHalf) > 1e-15 {
		t.Errorf("Sinc(0.5) expected %v, got %v", expectedHalf, s)
	}
	if s := Sinc(-0.5); math.Abs(s-expectedHalf) > 1e-15 {
		t.Errorf("Sinc(-0.5) expected %v, got %v", expectedHalf, s)
	}
}

func TestKaiserWindow(t *testing.T) {
	halfWidth := 10.0
	beta := 8.5

	wCenter := KaiserWindow(0.0, halfWidth, beta)
	if math.Abs(wCenter-1.0) > 1e-14 {
		t.Errorf("KaiserWindow at center expected 1.0, got %f", wCenter)
	}

	wEdge := KaiserWindow(halfWidth, halfWidth, beta)
	expectedEdge := 1.0 / BesselI0(beta)
	if math.Abs(wEdge-expectedEdge) > 1e-12 {
		t.Errorf("KaiserWindow at edge expected %e, got %e", expectedEdge, wEdge)
	}

	wOutside := KaiserWindow(halfWidth+0.1, halfWidth, beta)
	if wOutside != 0.0 {
		t.Errorf("KaiserWindow outside halfWidth expected 0.0, got %f", wOutside)
	}

	// Verify monotonic decrease
	prev := 1.0
	for d := 0.5; d <= halfWidth; d += 0.5 {
		curr := KaiserWindow(d, halfWidth, beta)
		if curr > prev {
			t.Errorf("KaiserWindow not monotonically decreasing at d=%f: curr %f > prev %f", d, curr, prev)
		}
		prev = curr
	}
}

func TestResample_44100_to_48000_SampleCount(t *testing.T) {
	inRate := 44100
	outRate := 48000
	numSamples := inRate // 1 second of audio

	resampler, err := NewResampler(inRate, outRate, WithQuality(QualityHigh))
	if err != nil {
		t.Fatalf("NewResampler failed: %v", err)
	}

	input := make([]float64, numSamples)
	for i := range input {
		input[i] = math.Sin(2.0 * math.Pi * 440.0 * float64(i) / float64(inRate))
	}

	output, err := resampler.ResampleChannel(input)
	if err != nil {
		t.Fatalf("ResampleChannel failed: %v", err)
	}

	if len(output) != outRate {
		t.Errorf("Expected output length %d, got %d", outRate, len(output))
	}
}

func TestResample_IdentityBypass(t *testing.T) {
	rate := 48000
	numSamples := 1000

	resampler, err := NewResampler(rate, rate)
	if err != nil {
		t.Fatalf("NewResampler failed: %v", err)
	}

	input := make([]float64, numSamples)
	for i := range input {
		input[i] = math.Sin(2.0 * math.Pi * 1000.0 * float64(i) / float64(rate))
	}

	output, err := resampler.ResampleChannel(input)
	if err != nil {
		t.Fatalf("ResampleChannel failed: %v", err)
	}

	if len(output) != len(input) {
		t.Fatalf("Length mismatch: expected %d, got %d", len(input), len(output))
	}

	for i := range input {
		if input[i] != output[i] {
			t.Fatalf("Identity bypass altered sample at index %d: expected %v, got %v", i, input[i], output[i])
		}
	}
}

func TestResample_SineWaveSNR(t *testing.T) {
	inRate := 44100
	outRate := 48000
	freq := 1000.0
	durationSec := 0.5
	numIn := int(float64(inRate) * durationSec)

	input := make([]float64, numIn)
	for i := 0; i < numIn; i++ {
		input[i] = math.Sin(2.0 * math.Pi * freq * float64(i) / float64(inRate))
	}

	resampler, err := NewResampler(inRate, outRate, WithQuality(QualityHigh))
	if err != nil {
		t.Fatalf("NewResampler failed: %v", err)
	}

	output, err := resampler.ResampleChannel(input)
	if err != nil {
		t.Fatalf("ResampleChannel failed: %v", err)
	}

	// Stable region excluding filter transition boundaries (lobes / ratio)
	guard := int(resampler.halfWidth * float64(outRate) / float64(inRate) * 1.5)
	if guard < 100 {
		guard = 100
	}
	startIdx := guard
	endIdx := len(output) - guard

	if startIdx >= endIdx {
		t.Fatalf("Guard margins too large: start %d, end %d", startIdx, endIdx)
	}

	var sigPower float64
	var noisePower float64

	for m := startIdx; m < endIdx; m++ {
		tTarget := float64(m) / float64(outRate)
		ideal := math.Sin(2.0 * math.Pi * freq * tTarget)
		actual := output[m]

		sigPower += ideal * ideal
		errVal := actual - ideal
		noisePower += errVal * errVal
	}

	snrDB := 10.0 * math.Log10(sigPower/noisePower)
	t.Logf("Measured 1kHz Sine Wave Resampling SNR: %.2f dB (QualityHigh)", snrDB)

	if snrDB < 100.0 {
		t.Errorf("SNR %.2f dB is below required 100 dB threshold", snrDB)
	}
}

func TestResample_Downsample96000_to_48000(t *testing.T) {
	inRate := 96000
	outRate := 48000
	durationSec := 0.2
	numIn := int(float64(inRate) * durationSec)

	resampler, err := NewResampler(inRate, outRate, WithQuality(QualityHigh))
	if err != nil {
		t.Fatalf("NewResampler failed: %v", err)
	}

	// 1. Passband signal: 1kHz tone should be preserved with SNR > 100 dB
	inputPass := make([]float64, numIn)
	for i := 0; i < numIn; i++ {
		inputPass[i] = math.Sin(2.0 * math.Pi * 1000.0 * float64(i) / float64(inRate))
	}

	outPass, err := resampler.ResampleChannel(inputPass)
	if err != nil {
		t.Fatalf("ResampleChannel passband failed: %v", err)
	}

	guard := int(resampler.halfWidth*float64(outRate)/float64(inRate)*1.5) + 50
	var sigPower, noisePower float64
	for m := guard; m < len(outPass)-guard; m++ {
		ideal := math.Sin(2.0 * math.Pi * 1000.0 * float64(m) / float64(outRate))
		sigPower += ideal * ideal
		diff := outPass[m] - ideal
		noisePower += diff * diff
	}
	snr := 10.0 * math.Log10(sigPower/noisePower)
	t.Logf("96kHz->48kHz passband SNR: %.2f dB", snr)
	if snr < 100.0 {
		t.Errorf("Passband SNR %.2f dB below 100 dB", snr)
	}

	// 2. Stopband signal: 35kHz tone (above 24kHz Nyquist) must be attenuated by > 80 dB
	inputStop := make([]float64, numIn)
	for i := 0; i < numIn; i++ {
		inputStop[i] = math.Sin(2.0 * math.Pi * 35000.0 * float64(i) / float64(inRate))
	}

	outStop, err := resampler.ResampleChannel(inputStop)
	if err != nil {
		t.Fatalf("ResampleChannel stopband failed: %v", err)
	}

	var maxStopAmp float64
	for m := guard; m < len(outStop)-guard; m++ {
		if a := math.Abs(outStop[m]); a > maxStopAmp {
			maxStopAmp = a
		}
	}
	attenuationDB := -20.0 * math.Log10(maxStopAmp)
	t.Logf("35kHz stopband anti-aliasing attenuation: %.2f dB", attenuationDB)
	if attenuationDB < 80.0 {
		t.Errorf("Stopband attenuation %.2f dB is less than required 80 dB", attenuationDB)
	}
}

func TestResample_Upsample22050_to_48000(t *testing.T) {
	inRate := 22050
	outRate := 48000
	numIn := 22050 // 1 sec

	input := make([]float64, numIn)
	for i := 0; i < numIn; i++ {
		input[i] = math.Sin(2.0 * math.Pi * 440.0 * float64(i) / float64(inRate))
	}

	resampler, err := NewResampler(inRate, outRate, WithQuality(QualityHigh))
	if err != nil {
		t.Fatalf("NewResampler failed: %v", err)
	}

	output, err := resampler.ResampleChannel(input)
	if err != nil {
		t.Fatalf("ResampleChannel failed: %v", err)
	}

	if len(output) != outRate {
		t.Errorf("Expected %d output samples, got %d", outRate, len(output))
	}

	guard := int(resampler.halfWidth) + 100
	var sigPower, noisePower float64
	for m := guard; m < len(output)-guard; m++ {
		ideal := math.Sin(2.0 * math.Pi * 440.0 * float64(m) / float64(outRate))
		sigPower += ideal * ideal
		diff := output[m] - ideal
		noisePower += diff * diff
	}
	snr := 10.0 * math.Log10(sigPower/noisePower)
	t.Logf("22050Hz -> 48000Hz 440Hz tone SNR: %.2f dB", snr)
	if snr < 95.0 {
		t.Errorf("Upsampling SNR %.2f dB below 95 dB", snr)
	}
}

func TestResample_MultiChannelStereoSeparation(t *testing.T) {
	inRate := 44100
	outRate := 48000
	numIn := 4410 // 0.1 sec

	left := make([]float64, numIn)
	right := make([]float64, numIn)
	for i := 0; i < numIn; i++ {
		left[i] = math.Sin(2.0 * math.Pi * 440.0 * float64(i) / float64(inRate))
		right[i] = math.Cos(2.0 * math.Pi * 880.0 * float64(i) / float64(inRate))
	}

	buf := &PCMBuffer{
		Channels:   2,
		SampleRate: inRate,
		NumSamples: numIn,
		Samples:    [][]float64{left, right},
	}

	resampledBuf, err := ResampleTo48kHz(buf)
	if err != nil {
		t.Fatalf("ResampleTo48kHz failed: %v", err)
	}

	if resampledBuf.Channels != 2 {
		t.Fatalf("Expected 2 channels, got %d", resampledBuf.Channels)
	}
	expectedLen := int(math.Round(float64(numIn) * float64(outRate) / float64(inRate)))
	if resampledBuf.NumSamples != expectedLen {
		t.Errorf("Expected %d samples, got %d", expectedLen, resampledBuf.NumSamples)
	}

	// Verify channels are independent (no crosstalk)
	// Left should have 440Hz phase and Right 880Hz phase
	mid := expectedLen / 2
	tTarget := float64(mid) / float64(outRate)
	idealLeft := math.Sin(2.0 * math.Pi * 440.0 * tTarget)
	idealRight := math.Cos(2.0 * math.Pi * 880.0 * tTarget)

	if math.Abs(resampledBuf.Samples[0][mid]-idealLeft) > 1e-4 {
		t.Errorf("Left channel mismatch: expected %f, got %f", idealLeft, resampledBuf.Samples[0][mid])
	}
	if math.Abs(resampledBuf.Samples[1][mid]-idealRight) > 1e-4 {
		t.Errorf("Right channel mismatch: expected %f, got %f", idealRight, resampledBuf.Samples[1][mid])
	}
}

func TestResample_EdgeCases(t *testing.T) {
	// 1. Nil buffer
	_, err := ResampleBuffer(nil, 48000)
	if !errors.Is(err, ErrNilPCMBuffer) {
		t.Errorf("Expected ErrNilPCMBuffer, got %v", err)
	}

	// 2. Empty buffer
	emptyBuf := &PCMBuffer{
		Channels:   0,
		SampleRate: 44100,
		NumSamples: 0,
		Samples:    nil,
	}
	_, err = ResampleBuffer(emptyBuf, 48000)
	if !errors.Is(err, ErrEmptyPCMBuffer) {
		t.Errorf("Expected ErrEmptyPCMBuffer, got %v", err)
	}

	// 3. Invalid sample rate
	_, err = NewResampler(-1, 48000)
	if !errors.Is(err, ErrInvalidSampleRate) {
		t.Errorf("Expected ErrInvalidSampleRate, got %v", err)
	}
	_, err = NewResampler(44100, 0)
	if !errors.Is(err, ErrInvalidSampleRate) {
		t.Errorf("Expected ErrInvalidSampleRate, got %v", err)
	}

	// 4. Rate mismatch between resampler and buffer
	resampler, _ := NewResampler(44100, 48000)
	buf48 := &PCMBuffer{
		Channels:   1,
		SampleRate: 48000,
		NumSamples: 100,
		Samples:    [][]float64{make([]float64, 100)},
	}
	_, err = resampler.ResampleBuffer(buf48)
	if !errors.Is(err, ErrRateMismatch) {
		t.Errorf("Expected ErrRateMismatch, got %v", err)
	}

	// 5. Compare table lookup mode vs direct evaluation mode
	resDirect, _ := NewResampler(44100, 48000, WithDirectEvaluation(true), WithQuality(QualityStandard))
	resTable, _ := NewResampler(44100, 48000, WithDirectEvaluation(false), WithQuality(QualityStandard))

	testSignal := make([]float64, 500)
	for i := range testSignal {
		testSignal[i] = math.Sin(2.0 * math.Pi * 500.0 * float64(i) / 44100.0)
	}

	outDirect, err1 := resDirect.ResampleChannel(testSignal)
	outTable, err2 := resTable.ResampleChannel(testSignal)
	if err1 != nil || err2 != nil {
		t.Fatalf("Resample failed: %v, %v", err1, err2)
	}

	var maxDiff float64
	for i := range outDirect {
		d := math.Abs(outDirect[i] - outTable[i])
		if d > maxDiff {
			maxDiff = d
		}
	}
	t.Logf("Max difference between direct evaluation and table lookup: %e", maxDiff)
	if maxDiff > 1e-4 {
		t.Errorf("Table lookup differs too much from direct calculation: maxDiff = %e", maxDiff)
	}
}

func TestResample100RateTransitions(t *testing.T) {
	inRates := []int{8000, 11025, 16000, 22050, 32000, 44100, 48000, 88200, 96000, 192000}
	outRates := []int{16000, 22050, 44100, 48000, 96000}

	count := 0
	for _, inR := range inRates {
		for _, outR := range outRates {
			// Subsample 100 combinations with 2 variations of channel count
			for _, ch := range []int{1, 2} {
				count++
				numIn := 500
				input := make([][]float64, ch)
				for c := 0; c < ch; c++ {
					input[c] = make([]float64, numIn)
					freq := 200.0 * float64(c+1)
					for i := 0; i < numIn; i++ {
						input[c][i] = 0.8 * math.Sin(2.0*math.Pi*freq*float64(i)/float64(inR))
					}
				}

				resampler, err := NewResampler(inR, outR, WithQuality(QualityFast))
				if err != nil {
					t.Fatalf("Transition %d: NewResampler(%d, %d) failed: %v", count, inR, outR, err)
				}

				out, err := resampler.Resample(input)
				if err != nil {
					t.Fatalf("Transition %d: Resample failed: %v", count, err)
				}

				expectedOut := int(math.Round(float64(numIn) * float64(outR) / float64(inR)))
				if expectedOut <= 0 {
					expectedOut = 1
				}

				for c := 0; c < ch; c++ {
					if len(out[c]) != expectedOut {
						t.Errorf("Transition %d: rate %d->%d ch %d expected %d samples, got %d",
							count, inR, outR, c, expectedOut, len(out[c]))
					}
					for i, v := range out[c] {
						if math.IsNaN(v) || math.IsInf(v, 0) {
							t.Fatalf("Transition %d: NaN or Inf at sample %d", count, i)
						}
						if math.Abs(v) > 1.2 {
							t.Fatalf("Transition %d: sample %d clipped/exploded: %f", count, i, v)
						}
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

	t.Logf("Successfully executed %d heterogeneous rate transition resamplings with 0 errors.", count)
}
