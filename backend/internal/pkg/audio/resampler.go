// Package audio provides professional audio decoding, stream header parsing,
// buffer unpacking, and high-fidelity bandlimited sinc audio resampling.
package audio

import (
	"errors"
	"fmt"
	"math"
)

// QualityProfile defines the resampling filter fidelity and tap density.
type QualityProfile int

const (
	// QualityFast provides low-latency conversion with 12 sinc lobes (~75dB SNR).
	QualityFast QualityProfile = 1
	// QualityStandard provides balanced studio-grade conversion with 20 sinc lobes (~100dB SNR).
	QualityStandard QualityProfile = 2
	// QualityHigh provides mastering-grade conversion with 32 sinc lobes (~115dB SNR).
	QualityHigh QualityProfile = 3
	// QualityUltra provides archival-grade conversion with 48 sinc lobes (~135dB SNR).
	QualityUltra QualityProfile = 4
)

// DefaultTargetSampleRate is the standard professional broadcast audio rate (48kHz).
const DefaultTargetSampleRate = 48000

// Resampler validation errors.
var (
	ErrInvalidSampleRate = errors.New("invalid sample rate: must be positive integer")
	ErrNilPCMBuffer      = errors.New("pcm buffer is nil")
	ErrEmptyPCMBuffer    = errors.New("pcm buffer contains no samples or channels")
	ErrNilResampler      = errors.New("resampler instance is nil")
	ErrRateMismatch      = errors.New("input buffer sample rate does not match resampler configuration")
)

// ResamplerConfig contains settings for the bandlimited sinc resampler.
type ResamplerConfig struct {
	InSampleRate        int
	OutSampleRate       int
	Quality             QualityProfile
	FilterLobes         int
	KaiserBeta          float64
	Rolloff             float64
	TableDensity        int
	UseDirectEvaluation bool
}

// ResamplerOption configures individual parameters of a Resampler.
type ResamplerOption func(*ResamplerConfig)

// WithQuality selects a predefined fidelity profile.
func WithQuality(q QualityProfile) ResamplerOption {
	return func(c *ResamplerConfig) {
		c.Quality = q
		applyQualityProfile(c, q)
	}
}

// WithFilterLobes configures the number of zero-crossings (half-width) for the sinc filter.
func WithFilterLobes(lobes int) ResamplerOption {
	return func(c *ResamplerConfig) {
		if lobes > 0 {
			c.FilterLobes = lobes
		}
	}
}

// WithKaiserBeta sets the Kaiser window shape parameter beta (controls stopband attenuation).
func WithKaiserBeta(beta float64) ResamplerOption {
	return func(c *ResamplerConfig) {
		if beta > 0 {
			c.KaiserBeta = beta
		}
	}
}

// WithRolloff sets the anti-aliasing cutoff frequency ratio relative to Nyquist (0 < rolloff <= 1.0).
func WithRolloff(rolloff float64) ResamplerOption {
	return func(c *ResamplerConfig) {
		if rolloff > 0.0 && rolloff <= 1.0 {
			c.Rolloff = rolloff
		}
	}
}

// WithDirectEvaluation forces direct analytic sinc and Bessel evaluation instead of table lookup.
func WithDirectEvaluation(direct bool) ResamplerOption {
	return func(c *ResamplerConfig) {
		c.UseDirectEvaluation = direct
	}
}

// WithTableDensity configures the number of lookup subdivisions per sample interval.
func WithTableDensity(density int) ResamplerOption {
	return func(c *ResamplerConfig) {
		if density >= 64 {
			c.TableDensity = density
		}
	}
}

func applyQualityProfile(c *ResamplerConfig, q QualityProfile) {
	switch q {
	case QualityFast:
		c.FilterLobes = 12
		c.KaiserBeta = 6.0
		c.Rolloff = 0.92
		c.TableDensity = 1024
	case QualityHigh:
		c.FilterLobes = 32
		c.KaiserBeta = 10.5
		c.Rolloff = 0.97
		c.TableDensity = 4096
	case QualityUltra:
		c.FilterLobes = 48
		c.KaiserBeta = 12.0
		c.Rolloff = 0.985
		c.TableDensity = 8192
	case QualityStandard:
		fallthrough
	default:
		c.FilterLobes = 20
		c.KaiserBeta = 8.5
		c.Rolloff = 0.95
		c.TableDensity = 2048
	}
}

// Resampler provides thread-safe, high-precision audio resampling between arbitrary sample rates.
type Resampler struct {
	cfg        ResamplerConfig
	ratio      float64
	cutoff     float64
	halfWidth  float64
	besselBeta float64
	table      []float64
	tableStep  float64
	invStep    float64
}

// NewResampler constructs an initialized resampler for conversion from inRate to outRate.
func NewResampler(inRate, outRate int, opts ...ResamplerOption) (*Resampler, error) {
	if inRate <= 0 || outRate <= 0 {
		return nil, ErrInvalidSampleRate
	}

	cfg := ResamplerConfig{
		InSampleRate:  inRate,
		OutSampleRate: outRate,
		Quality:       QualityStandard,
	}
	applyQualityProfile(&cfg, QualityStandard)

	for _, opt := range opts {
		opt(&cfg)
	}

	ratio := float64(outRate) / float64(inRate)

	var cutoff float64
	if ratio < 1.0 {
		cutoff = ratio * cfg.Rolloff
	} else {
		cutoff = cfg.Rolloff
	}

	halfWidth := float64(cfg.FilterLobes) / cutoff
	bBeta := BesselI0(cfg.KaiserBeta)

	r := &Resampler{
		cfg:        cfg,
		ratio:      ratio,
		cutoff:     cutoff,
		halfWidth:  halfWidth,
		besselBeta: bBeta,
	}

	if !cfg.UseDirectEvaluation && inRate != outRate {
		numPoints := int(math.Ceil(halfWidth*float64(cfg.TableDensity))) + 2
		step := halfWidth / float64(numPoints-1)
		table := make([]float64, numPoints)
		for i := 0; i < numPoints; i++ {
			d := float64(i) * step
			table[i] = r.directKernel(d)
		}
		r.table = table
		r.tableStep = step
		r.invStep = 1.0 / step
	}

	return r, nil
}

// Config returns a copy of the active resampler configuration.
func (r *Resampler) Config() ResamplerConfig {
	return r.cfg
}

// directKernel calculates the exact Kaiser-windowed bandlimited sinc filter tap for distance d.
func (r *Resampler) directKernel(d float64) float64 {
	if d > r.halfWidth {
		return 0.0
	}
	s := Sinc(r.cutoff * d)
	tau := d / r.halfWidth
	win := BesselI0(r.cfg.KaiserBeta*math.Sqrt(1.0-tau*tau)) / r.besselBeta
	return r.cutoff * s * win
}

// kernel returns the filter tap value at distance d using lookup table or direct calculation.
func (r *Resampler) kernel(d float64) float64 {
	if d > r.halfWidth {
		return 0.0
	}
	if !r.cfg.UseDirectEvaluation && len(r.table) > 0 {
		pos := d * r.invStep
		idx := int(pos)
		if idx >= len(r.table)-1 {
			return 0.0
		}
		frac := pos - float64(idx)
		return r.table[idx]*(1.0-frac) + r.table[idx+1]*frac
	}
	return r.directKernel(d)
}

// ResampleChannel resamples a single slice of normalized 64-bit float audio samples.
func (r *Resampler) ResampleChannel(samples []float64) ([]float64, error) {
	if r == nil {
		return nil, ErrNilResampler
	}
	numIn := len(samples)
	if numIn == 0 {
		return []float64{}, nil
	}

	if r.cfg.InSampleRate == r.cfg.OutSampleRate {
		cloned := make([]float64, numIn)
		copy(cloned, samples)
		return cloned, nil
	}

	numOut := int(math.Round(float64(numIn) * r.ratio))
	if numOut <= 0 {
		numOut = 1
	}

	out := make([]float64, numOut)

	for m := 0; m < numOut; m++ {
		tm := float64(m) / r.ratio
		kMin := int(math.Floor(tm - r.halfWidth))
		kMax := int(math.Ceil(tm + r.halfWidth))

		var sum float64
		var weightSum float64

		for k := kMin; k <= kMax; k++ {
			d := math.Abs(float64(k) - tm)
			if d > r.halfWidth {
				continue
			}
			w := r.kernel(d)
			weightSum += w
			refIdx := reflectIndex(k, numIn)
			sum += samples[refIdx] * w
		}

		if weightSum != 0.0 {
			out[m] = sum / weightSum
		} else {
			out[m] = 0.0
		}
	}

	return out, nil
}

// Resample resamples an arbitrary multi-channel slice of audio samples.
func (r *Resampler) Resample(input [][]float64) ([][]float64, error) {
	if r == nil {
		return nil, ErrNilResampler
	}
	numChannels := len(input)
	if numChannels == 0 {
		return [][]float64{}, nil
	}

	out := make([][]float64, numChannels)
	for ch := 0; ch < numChannels; ch++ {
		resampled, err := r.ResampleChannel(input[ch])
		if err != nil {
			return nil, fmt.Errorf("channel %d: %w", ch, err)
		}
		out[ch] = resampled
	}

	return out, nil
}

// ResampleBuffer resamples a complete PCMBuffer structure to the resampler's target rate.
func (r *Resampler) ResampleBuffer(buf *PCMBuffer) (*PCMBuffer, error) {
	if r == nil {
		return nil, ErrNilResampler
	}
	if buf == nil {
		return nil, ErrNilPCMBuffer
	}
	if buf.Channels <= 0 || buf.NumSamples <= 0 || len(buf.Samples) == 0 {
		return nil, ErrEmptyPCMBuffer
	}
	if buf.SampleRate != r.cfg.InSampleRate {
		return nil, fmt.Errorf("%w: buffer has %d Hz, resampler configured for %d Hz",
			ErrRateMismatch, buf.SampleRate, r.cfg.InSampleRate)
	}

	resampledData, err := r.Resample(buf.Samples)
	if err != nil {
		return nil, err
	}

	numOutSamples := 0
	if len(resampledData) > 0 {
		numOutSamples = len(resampledData[0])
	}

	return &PCMBuffer{
		Channels:   buf.Channels,
		SampleRate: r.cfg.OutSampleRate,
		NumSamples: numOutSamples,
		Samples:    resampledData,
	}, nil
}

// ResampleBuffer resamples a PCMBuffer to targetSampleRate using standard quality profile.
func ResampleBuffer(buf *PCMBuffer, targetSampleRate int) (*PCMBuffer, error) {
	return ResampleBufferWithQuality(buf, targetSampleRate, QualityStandard)
}

// ResampleBufferWithQuality resamples a PCMBuffer to targetSampleRate using a specified quality profile.
func ResampleBufferWithQuality(buf *PCMBuffer, targetSampleRate int, quality QualityProfile) (*PCMBuffer, error) {
	if buf == nil {
		return nil, ErrNilPCMBuffer
	}
	resampler, err := NewResampler(buf.SampleRate, targetSampleRate, WithQuality(quality))
	if err != nil {
		return nil, err
	}
	return resampler.ResampleBuffer(buf)
}

// ResampleTo48kHz resamples any PCMBuffer to the industry standard 48,000 Hz.
func ResampleTo48kHz(buf *PCMBuffer) (*PCMBuffer, error) {
	return ResampleBuffer(buf, DefaultTargetSampleRate)
}

// reflectIndex mirrors indices out of range symmetrically across signal boundaries.
func reflectIndex(idx, length int) int {
	if length <= 1 {
		return 0
	}
	for idx < 0 || idx >= length {
		if idx < 0 {
			idx = -idx
		} else if idx >= length {
			idx = 2*length - 2 - idx
		}
	}
	return idx
}

// BesselI0 computes the zeroth-order modified Bessel function of the first kind.
func BesselI0(x float64) float64 {
	x = math.Abs(x)
	sum := 1.0
	term := 1.0
	halfX := x * 0.5
	for k := 1; k < 60; k++ {
		term *= halfX / float64(k)
		sq := term * term
		sum += sq
		if sq < sum*1e-16 {
			break
		}
	}
	return sum
}

// Sinc computes the normalized sinc function: sinc(x) = sin(pi*x) / (pi*x).
func Sinc(x float64) float64 {
	if math.Abs(x) < 1e-12 {
		return 1.0
	}
	px := math.Pi * x
	return math.Sin(px) / px
}

// KaiserWindow computes the value of the Kaiser window at distance d from filter center.
func KaiserWindow(d, halfWidth, beta float64) float64 {
	if halfWidth <= 0.0 || math.Abs(d) > halfWidth {
		return 0.0
	}
	tau := d / halfWidth
	return BesselI0(beta*math.Sqrt(1.0-tau*tau)) / BesselI0(beta)
}
