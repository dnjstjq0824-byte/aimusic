// Package audio provides high-frequency air band (>16kHz) spectral excitation,
// non-linear harmonic generation (even/odd overtones), and transparency enhancement
// to restore vitality to band-limited or muffled AI audio generations.
package audio

import (
	"encoding/json"
	"math"
)

// ExciterConfig configures spectral exciter parameters.
type ExciterConfig struct {
	CutoffHz     float64 `json:"cutoff_hz"`      // Crossover frequency to extract highs, default: 12000 Hz
	Drive        float64 `json:"drive"`          // Harmonic drive intensity (1.0 to 5.0), default: 2.0
	AirMixDb     float64 `json:"air_mix_db"`     // Blend level of generated harmonics in dB, default: -12.0 dB
	EvenOddRatio float64 `json:"even_odd_ratio"` // 0.0 = pure odd, 1.0 = pure even, default: 0.5 (balanced)
}

// DefaultExciterConfig returns standard mastering-grade air band exciter parameters.
func DefaultExciterConfig() ExciterConfig {
	return ExciterConfig{
		CutoffHz:     12000.0,
		Drive:        2.0,
		AirMixDb:     -12.0,
		EvenOddRatio: 0.5,
	}
}

// ExciterResult contains processed PCM buffer and spectral energy telemetry.
type ExciterResult struct {
	ProcessedBuffer     *PCMBuffer `json:"-"`
	OriginalAirEnergy   float64    `json:"original_air_energy"`
	EnhancedAirEnergy   float64    `json:"enhanced_air_energy"`
	TransparencyGainPct float64    `json:"transparency_gain_pct"` // Percentage improvement in high-end air band (>15%)
	OutputPeak          float64    `json:"output_peak"`
}

// HarmonicExciter synthesizes subtle harmonic overtones above 16kHz.
type HarmonicExciter struct {
	cfg ExciterConfig
}

// NewHarmonicExciter creates a new harmonic exciter.
func NewHarmonicExciter(cfg ExciterConfig) *HarmonicExciter {
	if cfg.CutoffHz <= 1000.0 {
		cfg.CutoffHz = 12000.0
	}
	if cfg.Drive <= 0.1 {
		cfg.Drive = 2.0
	}
	if cfg.AirMixDb > 0.0 {
		cfg.AirMixDb = -12.0
	}
	if cfg.EvenOddRatio < 0.0 || cfg.EvenOddRatio > 1.0 {
		cfg.EvenOddRatio = 0.5
	}
	return &HarmonicExciter{cfg: cfg}
}

// Process applies spectral excitement to the audio buffer.
func (he *HarmonicExciter) Process(input *PCMBuffer) (*ExciterResult, error) {
	if input == nil {
		return nil, ErrNilPCMBuffer
	}
	if input.NumSamples == 0 || input.Channels == 0 || len(input.Samples) == 0 {
		return nil, ErrEmptyPCMBuffer
	}

	numSamples := input.NumSamples
	sampleRate := input.SampleRate
	channels := input.Channels

	wetGain := math.Pow(10.0, he.cfg.AirMixDb/20.0)
	drive := he.cfg.Drive
	ratioEven := he.cfg.EvenOddRatio
	ratioOdd := 1.0 - ratioEven

	// Prepare output buffer
	outSamples := make([][]float64, channels)
	var origAirEnergySum, enhAirEnergySum float64
	var maxOutPeak float64

	airBandCutoff := 15000.0
	if float64(sampleRate)/2.0 <= airBandCutoff {
		// If Nyquist is at or below 15kHz, adjust cutoff to 70% of Nyquist
		airBandCutoff = float64(sampleRate) * 0.35
	}

	for ch := 0; ch < channels; ch++ {
		src := input.Samples[ch]
		dst := make([]float64, numSamples)

		// Step 1: Extract high-frequency components using highpass filter
		highs := biquadHighpass(src, sampleRate, he.cfg.CutoffHz)

		// Step 2: Generate harmonics through non-linear waveshaping
		harmonics := make([]float64, numSamples)
		for i := 0; i < numSamples; i++ {
			x := highs[i] * drive

			// Odd harmonics: soft cubic saturation (3rd harmonic)
			odd := x - (x*x*x)/3.0
			if odd > 1.0 {
				odd = 1.0
			} else if odd < -1.0 {
				odd = -1.0
			}

			// Even harmonics: asymmetrical quadratic curve (2nd harmonic)
			even := x * math.Abs(x)

			harm := ratioOdd*odd + ratioEven*even
			harmonics[i] = harm
		}

		// Step 3: 4th-order steep highpass on generated harmonics to isolate pure upper air band
		airHarmonics := biquadHighpass(biquadHighpass(harmonics, sampleRate, airBandCutoff), sampleRate, airBandCutoff)

		// Step 4: Blend air harmonics back with dry signal and soft-clip headroom
		for i := 0; i < numSamples; i++ {
			blended := src[i] + airHarmonics[i]*wetGain

			// Transparent soft limiter to avoid 0dBFS clipping
			if blended > 0.98 {
				blended = 0.98 + 0.02*math.Tanh((blended-0.98)/0.02)
			} else if blended < -0.98 {
				blended = -0.98 + 0.02*math.Tanh((blended+0.98)/0.02)
			}

			dst[i] = blended

			abs := math.Abs(blended)
			if abs > maxOutPeak {
				maxOutPeak = abs
			}
		}

		outSamples[ch] = dst

		// Measure air band energy before and after using steep 4th-order analysis
		origAir := biquadHighpass(biquadHighpass(src, sampleRate, airBandCutoff), sampleRate, airBandCutoff)
		enhAir := biquadHighpass(biquadHighpass(dst, sampleRate, airBandCutoff), sampleRate, airBandCutoff)

		for i := 0; i < numSamples; i++ {
			origAirEnergySum += origAir[i] * origAir[i]
			enhAirEnergySum += enhAir[i] * enhAir[i]
		}
	}

	totalSampleCount := float64(channels * numSamples)
	origAirRMS := math.Sqrt(origAirEnergySum / totalSampleCount)
	enhAirRMS := math.Sqrt(enhAirEnergySum / totalSampleCount)

	transparencyGainPct := 0.0
	if origAirRMS > 1e-6 {
		transparencyGainPct = ((enhAirRMS - origAirRMS) / origAirRMS) * 100.0
	} else if enhAirRMS > 1e-6 {
		transparencyGainPct = 100.0
	}

	outBuf := &PCMBuffer{
		Channels:   channels,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    outSamples,
	}

	return &ExciterResult{
		ProcessedBuffer:     outBuf,
		OriginalAirEnergy:   math.Round(origAirRMS*10000) / 10000,
		EnhancedAirEnergy:   math.Round(enhAirRMS*10000) / 10000,
		TransparencyGainPct: math.Round(transparencyGainPct*10) / 10,
		OutputPeak:          math.Round(maxOutPeak*10000) / 10000,
	}, nil
}

// biquadHighpass implements a 2nd-order Butterworth highpass filter (12dB/octave).
func biquadHighpass(in []float64, sampleRate int, cutoff float64) []float64 {
	out := make([]float64, len(in))
	if len(in) == 0 {
		return out
	}

	nyquist := float64(sampleRate) * 0.5
	if cutoff >= nyquist {
		cutoff = nyquist * 0.95
	}
	if cutoff <= 10.0 {
		cutoff = 10.0
	}

	omega := 2.0 * math.Pi * cutoff / float64(sampleRate)
	cosOmega := math.Cos(omega)
	sinOmega := math.Sin(omega)
	alpha := sinOmega / (2.0 * math.Sqrt(0.5)) // Q = 0.7071

	b0 := (1.0 + cosOmega) / 2.0
	b1 := -(1.0 + cosOmega)
	b2 := (1.0 + cosOmega) / 2.0
	a0 := 1.0 + alpha
	a1 := -2.0 * cosOmega
	a2 := 1.0 - alpha

	b0Norm := b0 / a0
	b1Norm := b1 / a0
	b2Norm := b2 / a0
	a1Norm := a1 / a0
	a2Norm := a2 / a0

	var x1, x2, y1, y2 float64
	for i, x0 := range in {
		y0 := b0Norm*x0 + b1Norm*x1 + b2Norm*x2 - a1Norm*y1 - a2Norm*y2
		out[i] = y0
		x2 = x1
		x1 = x0
		y2 = y1
		y1 = y0
	}

	return out
}

// ToJSON serializes exciter telemetry.
func (r *ExciterResult) ToJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
