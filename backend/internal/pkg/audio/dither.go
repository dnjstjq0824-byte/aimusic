// Package audio provides professional Triangular Probability Density Function (TPDF) dithering,
// psychoacoustic noise shaping (Lipshitz/Wannamaker 2nd-order error feedback),
// and harmonic distortion decorrelation for 24-bit/float64 to 16-bit PCM word-length reduction.
package audio

import (
	"errors"
	"math"
	"math/rand"
)

// DitherType defines dither noise generation and spectral distribution models.
type DitherType string

const (
	DitherNone        DitherType = "NONE"         // Simple rounding without dither (harmonic distortion baseline)
	DitherRPDF        DitherType = "RPDF"         // Rectangular probability density function (1 uniform random variable)
	DitherTPDF        DitherType = "TPDF"         // Triangular probability density function (sum of 2 independent uniforms)
	DitherNoiseShaped DitherType = "NOISE_SHAPED" // TPDF with 2nd-order psychoacoustic error feedback noise shaping
)

// Dither validation errors.
var (
	ErrNilDitherBuffer     = errors.New("pcm buffer is nil for dithering")
	ErrEmptyDitherBuffer   = errors.New("pcm buffer contains no samples for dithering")
	ErrInvalidTargetBit    = errors.New("target bit depth must be between 8 and 24 bits")
	ErrInvalidDitherConfig = errors.New("invalid dither configuration")
)

// DithererConfig specifies word-length reduction and noise-shaping parameters.
type DithererConfig struct {
	TargetBits int        // Target bit depth (e.g. 16 for CD/standard streaming, 8 for lo-fi)
	Type       DitherType // Dither noise algorithm (DitherTPDF or DitherNoiseShaped recommended)
	Seed       int64      // Random seed for deterministic reproducibility (0 for pseudo-random)
}

// DefaultDithererConfig returns standard 16-bit mastering dither configuration.
func DefaultDithererConfig() DithererConfig {
	return DithererConfig{
		TargetBits: 16,
		Type:       DitherNoiseShaped,
		Seed:       0,
	}
}

// Ditherer manages stateful dither noise generation and error-feedback filtering.
type Ditherer struct {
	cfg DithererConfig
}

// NewDitherer constructs a configured ditherer.
func NewDitherer(cfg DithererConfig) (*Ditherer, error) {
	if cfg.TargetBits < 8 || cfg.TargetBits > 24 {
		return nil, ErrInvalidTargetBit
	}
	if cfg.Type == "" {
		cfg.Type = DitherTPDF
	}
	return &Ditherer{cfg: cfg}, nil
}

// Process applies dithering and quantization to a multi-channel PCM buffer.
func (d *Ditherer) Process(buf *PCMBuffer) (*PCMBuffer, error) {
	if buf == nil {
		return nil, ErrNilDitherBuffer
	}
	if buf.NumSamples == 0 || buf.Channels == 0 || len(buf.Samples) == 0 {
		return nil, ErrEmptyDitherBuffer
	}

	channels := buf.Channels
	numSamples := buf.NumSamples
	sr := buf.SampleRate

	// Calculate LSB step size for target bit depth: LSB = 1.0 / (2^(bits - 1))
	scale := math.Pow(2.0, float64(d.cfg.TargetBits-1))
	lsb := 1.0 / scale
	maxPositive := (scale - 1.0) / scale
	minNegative := -1.0

	// Pseudo-random number generator
	var rng *rand.Rand
	if d.cfg.Seed != 0 {
		rng = rand.New(rand.NewSource(d.cfg.Seed))
	} else {
		rng = rand.New(rand.NewSource(20261109))
	}

	outSamples := make([][]float64, channels)

	// Noise shaping filter coefficients: 2nd-order Lipshitz/Wannamaker curve
	// H(z) = 1 - 1.62*z^-1 + 0.81*z^-2 (highpass shaping shifting noise above 15kHz)
	a1 := 1.62
	a2 := -0.81

	for c := 0; c < channels; c++ {
		outSamples[c] = make([]float64, numSamples)
		inSamples := buf.Samples[c]

		// Error feedback delay registers
		e1 := 0.0
		e2 := 0.0

		for s := 0; s < numSamples; s++ {
			inVal := inSamples[s]

			// Compute dither noise sample
			dither := 0.0
			switch d.cfg.Type {
			case DitherRPDF:
				// Uniform random in [-0.5*lsb, +0.5*lsb]
				dither = (rng.Float64() - 0.5) * lsb
			case DitherTPDF, DitherNoiseShaped:
				// Sum of two independent uniform randoms -> triangular distribution in [-1.0*lsb, +1.0*lsb]
				u1 := rng.Float64()
				u2 := rng.Float64()
				dither = (u1 - u2) * lsb
			case DitherNone:
				dither = 0.0
			}

			// Add shaped error feedback if enabled
			noiseShapingTerm := 0.0
			if d.cfg.Type == DitherNoiseShaped {
				noiseShapingTerm = a1*e1 + a2*e2
			}

			// Input with dither and error feedback
			targetSample := inVal + dither + noiseShapingTerm

			// Quantize to target discrete grid: round to nearest LSB multiple
			quantizedInt := math.Round(targetSample * scale)
			quantizedSample := quantizedInt * lsb

			// Strict digital clipping protection at boundary limits
			if quantizedSample > maxPositive {
				quantizedSample = maxPositive
			}
			if quantizedSample < minNegative {
				quantizedSample = minNegative
			}

			outSamples[c][s] = quantizedSample

			// Compute quantization error: e[n] = target - quantized
			// Update delay registers for noise shaping
			if d.cfg.Type == DitherNoiseShaped {
				errVal := targetSample - quantizedSample
				e2 = e1
				e1 = errVal
			}
		}
	}

	return &PCMBuffer{
		Channels:   channels,
		SampleRate: sr,
		NumSamples: numSamples,
		Samples:    outSamples,
	}, nil
}

// DitherTo16Bit is a convenience helper applying 16-bit TPDF noise-shaped dithering to an audio buffer.
func DitherTo16Bit(buf *PCMBuffer) (*PCMBuffer, error) {
	d, err := NewDitherer(DefaultDithererConfig())
	if err != nil {
		return nil, err
	}
	return d.Process(buf)
}

// GenerateTPDF produces a single triangular probability density function random dither value in [-lsb, +lsb].
func GenerateTPDF(rng *rand.Rand, lsb float64) float64 {
	u1 := rng.Float64()
	u2 := rng.Float64()
	return (u1 - u2) * lsb
}

// GenerateRPDF produces a single rectangular probability density function random dither value in [-0.5*lsb, +0.5*lsb].
func GenerateRPDF(rng *rand.Rand, lsb float64) float64 {
	return (rng.Float64() - 0.5) * lsb
}
