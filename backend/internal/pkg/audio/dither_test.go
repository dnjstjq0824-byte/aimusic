package audio

import (
	"math"
	"math/rand"
	"testing"
)

func TestTPDF_StatisticalDistribution(t *testing.T) {
	rng := rand.New(rand.NewSource(20261109))
	lsb := 1.0
	n := 100000

	sum := 0.0
	sumSq := 0.0
	minVal := math.MaxFloat64
	maxVal := -math.MaxFloat64

	for i := 0; i < n; i++ {
		v := GenerateTPDF(rng, lsb)
		sum += v
		sumSq += v * v
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	mean := sum / float64(n)
	variance := (sumSq / float64(n)) - (mean * mean)

	// Theoretical mean for TPDF is 0.0
	if math.Abs(mean) > 0.01 {
		t.Errorf("TPDF mean expected ~0.0, got %f", mean)
	}

	// Theoretical variance for TPDF with LSB=1.0 is 1/6 ~ 0.16667
	expectedVar := 1.0 / 6.0
	if math.Abs(variance-expectedVar) > 0.005 {
		t.Errorf("TPDF variance expected %f, got %f", expectedVar, variance)
	}

	// Range must be strictly within [-1.0, +1.0]
	if minVal < -1.0 || maxVal > 1.0 {
		t.Errorf("TPDF values out of bounds [-1.0, 1.0]: min=%f, max=%f", minVal, maxVal)
	}
}

func TestRPDF_StatisticalDistribution(t *testing.T) {
	rng := rand.New(rand.NewSource(20261109))
	lsb := 1.0
	n := 100000

	sum := 0.0
	sumSq := 0.0
	minVal := math.MaxFloat64
	maxVal := -math.MaxFloat64

	for i := 0; i < n; i++ {
		v := GenerateRPDF(rng, lsb)
		sum += v
		sumSq += v * v
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	mean := sum / float64(n)
	variance := (sumSq / float64(n)) - (mean * mean)

	// Theoretical mean for RPDF is 0.0
	if math.Abs(mean) > 0.01 {
		t.Errorf("RPDF mean expected ~0.0, got %f", mean)
	}

	// Theoretical variance for RPDF with LSB=1.0 is 1/12 ~ 0.08333
	expectedVar := 1.0 / 12.0
	if math.Abs(variance-expectedVar) > 0.005 {
		t.Errorf("RPDF variance expected %f, got %f", expectedVar, variance)
	}

	// Range must be within [-0.5, +0.5]
	if minVal < -0.5 || maxVal > 0.5 {
		t.Errorf("RPDF values out of bounds [-0.5, 0.5]: min=%f, max=%f", minVal, maxVal)
	}
}

func TestDither_HarmonicDistortionElimination(t *testing.T) {
	// Synthesize a very low-level 1kHz sine wave near 8-bit quantization threshold
	sr := 48000
	numSamples := 48000
	bits := 8
	scale := math.Pow(2.0, float64(bits-1))
	lsb := 1.0 / scale // LSB ~ 1/128 = 0.0078125

	amp := 2.5 * lsb // Low amplitude: 2.5 LSBs
	freq := 1000.0

	pcmNoDither := createSyntheticPCMBuffer(sr, 1, numSamples, func(s, c int) float64 {
		tSec := float64(s) / float64(sr)
		return amp * math.Sin(2.0*math.Pi*freq*tSec)
	})

	pcmTPDF := createSyntheticPCMBuffer(sr, 1, numSamples, func(s, c int) float64 {
		tSec := float64(s) / float64(sr)
		return amp * math.Sin(2.0*math.Pi*freq*tSec)
	})

	// 1. Process without dither (truncation / direct round)
	ditherNone, _ := NewDitherer(DithererConfig{TargetBits: bits, Type: DitherNone, Seed: 1234})
	outNone, err := ditherNone.Process(pcmNoDither)
	if err != nil {
		t.Fatalf("dither none failed: %v", err)
	}

	// 2. Process with TPDF dither
	ditherTPDF, _ := NewDitherer(DithererConfig{TargetBits: bits, Type: DitherTPDF, Seed: 1234})
	outTPDF, err := ditherTPDF.Process(pcmTPDF)
	if err != nil {
		t.Fatalf("dither tpdf failed: %v", err)
	}

	// Calculate cross-correlation between quantization error and input signal:
	// rho = E[(error - mean(error)) * (input - mean(input))] / (std(error) * std(input))
	corrNone := computeSignalErrorCorrelation(pcmNoDither.Samples[0], outNone.Samples[0])
	corrTPDF := computeSignalErrorCorrelation(pcmTPDF.Samples[0], outTPDF.Samples[0])

	// For truncation without dither: error is strongly correlated to signal (correlated harmonic distortion)
	if math.Abs(corrNone) < 0.20 {
		t.Errorf("expected high correlation for undithered quantization error, got %f", corrNone)
	}

	// For TPDF dither: error is completely decorrelated (converted to uncorrelated white noise)
	if math.Abs(corrTPDF) > 0.05 {
		t.Errorf("expected near-zero correlation for TPDF dithered error, got %f", corrTPDF)
	}
}

// Helper to compute Pearson correlation between input signal and quantization error.
func computeSignalErrorCorrelation(in, out []float64) float64 {
	n := len(in)
	err := make([]float64, n)
	for i := 0; i < n; i++ {
		err[i] = out[i] - in[i]
	}

	sumIn, sumErr := 0.0, 0.0
	for i := 0; i < n; i++ {
		sumIn += in[i]
		sumErr += err[i]
	}
	meanIn := sumIn / float64(n)
	meanErr := sumErr / float64(n)

	num := 0.0
	denIn := 0.0
	denErr := 0.0
	for i := 0; i < n; i++ {
		diffIn := in[i] - meanIn
		diffErr := err[i] - meanErr
		num += diffIn * diffErr
		denIn += diffIn * diffIn
		denErr += diffErr * diffErr
	}

	if denIn == 0 || denErr == 0 {
		return 0.0
	}
	return num / math.Sqrt(denIn*denErr)
}

func TestDither_NoiseShapingHighFrequencyShift(t *testing.T) {
	sr := 48000
	numSamples := 48000
	bits := 16

	// Zero input signal: analyze pure dither noise spectra
	pcmFlat := createSyntheticPCMBuffer(sr, 1, numSamples, func(s, c int) float64 { return 0.0 })

	ditherShaped, _ := NewDitherer(DithererConfig{TargetBits: bits, Type: DitherNoiseShaped, Seed: 42})
	outShaped, err := ditherShaped.Process(pcmFlat)
	if err != nil {
		t.Fatalf("Process noise shaped failed: %v", err)
	}

	// Measure high-frequency energy via first-difference derivative:
	// Noise shaping pushes noise to high frequencies, so adjacent sample differences (hf energy) are much larger
	diffSumSq := 0.0
	samples := outShaped.Samples[0]
	for i := 1; i < len(samples); i++ {
		d := samples[i] - samples[i-1]
		diffSumSq += d * d
	}
	hfEnergyShaped := diffSumSq / float64(len(samples))

	// Compare with flat unshaped TPDF
	ditherFlat, _ := NewDitherer(DithererConfig{TargetBits: bits, Type: DitherTPDF, Seed: 42})
	outFlat, _ := ditherFlat.Process(pcmFlat)
	diffSumSqFlat := 0.0
	samplesFlat := outFlat.Samples[0]
	for i := 1; i < len(samplesFlat); i++ {
		d := samplesFlat[i] - samplesFlat[i-1]
		diffSumSqFlat += d * d
	}
	hfEnergyFlat := diffSumSqFlat / float64(len(samplesFlat))

	// Shaped noise must have significantly higher high-frequency difference energy (shifted above audible band)
	if hfEnergyShaped <= hfEnergyFlat*1.5 {
		t.Errorf("noise shaping failed to shift noise to high frequencies: shaped=%f, flat=%f", hfEnergyShaped, hfEnergyFlat)
	}
}

func TestDither_16Bit_ClippingProtection(t *testing.T) {
	sr := 48000
	numSamples := 4800
	// Signal reaching exact extremes +1.0 and -1.0
	pcm := createSyntheticPCMBuffer(sr, 2, numSamples, func(s, c int) float64 {
		if s%2 == 0 {
			return 1.0
		}
		return -1.0
	})

	ditherer, _ := NewDitherer(DithererConfig{TargetBits: 16, Type: DitherNoiseShaped, Seed: 999})
	out, err := ditherer.Process(pcm)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	scale := math.Pow(2.0, 15.0)
	maxAllowed := (scale - 1.0) / scale
	minAllowed := -1.0

	for c := 0; c < out.Channels; c++ {
		for s := 0; s < out.NumSamples; s++ {
			val := out.Samples[c][s]
			if val > maxAllowed || val < minAllowed {
				t.Fatalf("sample at ch %d, idx %d clipped out of bounds: %f (allowed [%f, %f])", c, s, val, minAllowed, maxAllowed)
			}
		}
	}
}

func TestDither_EdgeCases(t *testing.T) {
	// Nil buffer
	ditherer, _ := NewDitherer(DefaultDithererConfig())
	_, err := ditherer.Process(nil)
	if err != ErrNilDitherBuffer {
		t.Errorf("expected ErrNilDitherBuffer, got %v", err)
	}

	// Empty buffer
	emptyBuf := &PCMBuffer{Channels: 1, SampleRate: 48000, NumSamples: 0, Samples: [][]float64{{}}}
	_, err = ditherer.Process(emptyBuf)
	if err != ErrEmptyDitherBuffer {
		t.Errorf("expected ErrEmptyDitherBuffer, got %v", err)
	}

	// Invalid target bits
	_, err = NewDitherer(DithererConfig{TargetBits: 4})
	if err != ErrInvalidTargetBit {
		t.Errorf("expected ErrInvalidTargetBit for 4 bits, got %v", err)
	}
	_, err = NewDitherer(DithererConfig{TargetBits: 32})
	if err != ErrInvalidTargetBit {
		t.Errorf("expected ErrInvalidTargetBit for 32 bits, got %v", err)
	}

	// DitherTo16Bit helper
	pcm := createSyntheticPCMBuffer(48000, 1, 100, func(s, c int) float64 { return 0.5 })
	out16, err := DitherTo16Bit(pcm)
	if err != nil {
		t.Fatalf("DitherTo16Bit failed: %v", err)
	}
	if out16.NumSamples != 100 {
		t.Errorf("expected 100 samples, got %d", out16.NumSamples)
	}
}

func TestDither_100HeterogeneousStreams(t *testing.T) {
	rng := rand.New(rand.NewSource(20261109))
	totalTests := 100

	sampleRates := []int{8000, 16000, 22050, 32000, 44100, 48000, 96000}
	channelsList := []int{1, 2, 4}
	bitDepths := []int{8, 12, 16, 20, 24}
	ditherTypes := []DitherType{DitherTPDF, DitherNoiseShaped, DitherRPDF, DitherNone}

	for iter := 0; iter < totalTests; iter++ {
		sr := sampleRates[rng.Intn(len(sampleRates))]
		ch := channelsList[rng.Intn(len(channelsList))]
		bits := bitDepths[rng.Intn(len(bitDepths))]
		dType := ditherTypes[rng.Intn(len(ditherTypes))]
		numSamples := sr/4 + rng.Intn(sr/2) // 0.25s to 0.75s

		pcm := createSyntheticPCMBuffer(sr, ch, numSamples, func(s, c int) float64 {
			tSec := float64(s) / float64(sr)
			return 0.8 * math.Sin(2.0*math.Pi*350.0*tSec)
		})

		cfg := DithererConfig{
			TargetBits: bits,
			Type:       dType,
			Seed:       int64(iter + 100),
		}

		d, err := NewDitherer(cfg)
		if err != nil {
			t.Fatalf("iteration %d: NewDitherer failed: %v", iter, err)
		}

		out, err := d.Process(pcm)
		if err != nil {
			t.Fatalf("iteration %d: Process failed: %v", iter, err)
		}

		if out.NumSamples != numSamples || out.Channels != ch {
			t.Fatalf("iteration %d: buffer dimensions mismatch", iter)
		}

		// Verify that all output samples lie strictly on the quantized discrete grid:
		// val / LSB must be an exact integer
		scale := math.Pow(2.0, float64(bits-1))
		for c := 0; c < ch; c++ {
			for s := 0; s < numSamples; s++ {
				val := out.Samples[c][s]
				intVal := math.Round(val * scale)
				gridVal := intVal / scale
				if math.Abs(val-gridVal) > 1e-9 {
					t.Fatalf("iteration %d: sample not on quantized grid: val=%f, gridVal=%f", iter, val, gridVal)
				}
			}
		}
	}

	t.Logf("Successfully verified 100 heterogeneous audio streams with TPDF and noise-shaped dithering.")
}
