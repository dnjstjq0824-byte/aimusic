// Package audio provides professional loudness normalization, gain matching,
// ceiling enforcement, and lookahead peak limiting conforming to EBU R128,
// AES TD1004, and modern streaming platform standards (Spotify, Apple Music, YouTube).
package audio

import (
	"errors"
	"fmt"
	"math"
)

// NormalizationProfile represents a standardized target loudness profile.
type NormalizationProfile string

const (
	// ProfileStreamingStandard targets -14.0 LUFS with a -1.0 dBTP ceiling (Spotify, YouTube, Tidal, Amazon Music).
	ProfileStreamingStandard NormalizationProfile = "STREAMING_STANDARD"
	// ProfileAppleMusic targets -16.0 LUFS with a -1.0 dBTP ceiling (Apple Music SoundCheck).
	ProfileAppleMusic NormalizationProfile = "APPLE_MUSIC"
	// ProfileEBUR128 targets -23.0 LUFS with a -1.0 dBTP ceiling (European Broadcast Standard).
	ProfileEBUR128 NormalizationProfile = "EBU_R128"
	// ProfileCustom indicates user-specified TargetLUFS and CeilingDBTP.
	ProfileCustom NormalizationProfile = "CUSTOM"
)

// Default limits and thresholds.
const (
	DefaultCeilingDBTP = -1.0
	MaxAllowedCeiling  = 0.0
)

// Normalizer validation errors.
var (
	ErrInvalidCeiling   = errors.New("invalid ceiling: ceiling must be <= 0.0 dBTP")
	ErrNilNormalizer    = errors.New("normalizer instance is nil")
	ErrUnmeasurableAudio = errors.New("cannot normalize audio: loudness is below measurable threshold")
)

// NormalizerConfig holds settings for loudness gain matching and peak protection.
type NormalizerConfig struct {
	TargetLUFS          float64
	CeilingDBTP         float64
	PreventClipping     bool
	UseLookaheadLimiter bool
	Profile             NormalizationProfile
}

// NormalizerOption configures a Normalizer.
type NormalizerOption func(*NormalizerConfig)

// WithTargetLUFS sets the desired integrated loudness in LUFS.
func WithTargetLUFS(lufs float64) NormalizerOption {
	return func(c *NormalizerConfig) {
		c.TargetLUFS = lufs
		c.Profile = ProfileCustom
	}
}

// WithCeilingDBTP sets the maximum allowable true peak ceiling in dBTP.
func WithCeilingDBTP(dbtp float64) NormalizerOption {
	return func(c *NormalizerConfig) {
		c.CeilingDBTP = dbtp
	}
}

// WithPreventClipping enables gain backoff or peak limiting to prevent exceeding CeilingDBTP.
func WithPreventClipping(prevent bool) NormalizerOption {
	return func(c *NormalizerConfig) {
		c.PreventClipping = prevent
	}
}

// WithLookaheadLimiter enables transparent brickwall lookahead limiting when boost exceeds ceiling.
func WithLookaheadLimiter(enable bool) NormalizerOption {
	return func(c *NormalizerConfig) {
		c.UseLookaheadLimiter = enable
	}
}

// WithProfile applies preset parameters for a recognized industry standard.
func WithProfile(p NormalizationProfile) NormalizerOption {
	return func(c *NormalizerConfig) {
		c.Profile = p
		switch p {
		case ProfileAppleMusic:
			c.TargetLUFS = TargetLUFSAppleMusic
			c.CeilingDBTP = DefaultCeilingDBTP
		case ProfileEBUR128:
			c.TargetLUFS = TargetLUFSEBUR128
			c.CeilingDBTP = DefaultCeilingDBTP
		case ProfileStreamingStandard:
			fallthrough
		default:
			c.TargetLUFS = TargetLUFSStreaming
			c.CeilingDBTP = DefaultCeilingDBTP
		}
	}
}

// NormalizationReport summarizes the analysis and adjustments applied to the buffer.
type NormalizationReport struct {
	Profile           NormalizationProfile
	OriginalLUFS      float64
	TargetLUFS        float64
	AppliedGainDB     float64
	AppliedGainLinear float64
	FinalLUFS         float64
	OriginalPeakDBFS  float64
	FinalPeakDBFS     float64
	OriginalTruePeak  float64
	FinalTruePeak     float64
	ClippingPrevented bool
	LimiterEngaged    bool
}

// Normalizer manages loudness normalization and peak headroom enforcement.
type Normalizer struct {
	cfg   NormalizerConfig
	meter *LoudnessMeter
}

// NewNormalizer constructs an initialized Normalizer.
func NewNormalizer(opts ...NormalizerOption) (*Normalizer, error) {
	cfg := NormalizerConfig{
		TargetLUFS:          TargetLUFSStreaming,
		CeilingDBTP:         DefaultCeilingDBTP,
		PreventClipping:     true,
		UseLookaheadLimiter: false,
		Profile:             ProfileStreamingStandard,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.CeilingDBTP > MaxAllowedCeiling {
		return nil, fmt.Errorf("%w: got %.2f dBTP, max allowed is %.2f dBTP",
			ErrInvalidCeiling, cfg.CeilingDBTP, MaxAllowedCeiling)
	}

	return &Normalizer{
		cfg:   cfg,
		meter: NewLoudnessMeter(WithTruePeak(true)),
	}, nil
}

// Config returns a copy of the active normalizer configuration.
func (n *Normalizer) Config() NormalizerConfig {
	return n.cfg
}

// Normalize analyzes, adjusts gain, and protects peaks on a PCMBuffer.
func (n *Normalizer) Normalize(buf *PCMBuffer) (*PCMBuffer, *NormalizationReport, error) {
	if n == nil {
		return nil, nil, ErrNilNormalizer
	}
	if buf == nil {
		return nil, nil, ErrNilLoudnessBuffer
	}
	if buf.Channels <= 0 || buf.NumSamples <= 0 || len(buf.Samples) == 0 {
		return nil, nil, ErrEmptyAudioBuffer
	}

	// 1. Initial measurement of source buffer
	initialMetrics, err := n.meter.Measure(buf)
	if err != nil {
		return nil, nil, fmt.Errorf("initial loudness measure: %w", err)
	}

	if initialMetrics.IntegratedLUFS <= AbsoluteGateThresholdLUFS {
		return nil, nil, ErrUnmeasurableAudio
	}

	targetLUFS := n.cfg.TargetLUFS
	diffDB := targetLUFS - initialMetrics.IntegratedLUFS
	idealGainLinear := math.Pow(10.0, diffDB/20.0)

	appliedGainLinear := idealGainLinear
	appliedGainDB := diffDB
	clippingPrevented := false
	limiterEngaged := false

	// Projected peak after ideal linear gain
	projectedTruePeak := initialMetrics.TruePeakDBTP + diffDB

	var processedBuf *PCMBuffer

	// 2. Headroom & Peak protection logic
	if n.cfg.PreventClipping && projectedTruePeak > n.cfg.CeilingDBTP {
		clippingPrevented = true

		if n.cfg.UseLookaheadLimiter {
			// Apply ideal linear gain, then pass through brickwall lookahead limiter
			boostedBuf := ApplyGain(buf, idealGainLinear)
			limiter := newLookaheadLimiter(boostedBuf.SampleRate, n.cfg.CeilingDBTP, 5.0, 50.0)
			limitedSamples := limiter.process(boostedBuf.Samples)
			processedBuf = &PCMBuffer{
				Channels:   boostedBuf.Channels,
				SampleRate: boostedBuf.SampleRate,
				NumSamples: boostedBuf.NumSamples,
				Samples:    limitedSamples,
			}
			limiterEngaged = true
		} else {
			// Back off gain so True Peak precisely matches the ceiling
			maxAllowedGainDB := n.cfg.CeilingDBTP - initialMetrics.TruePeakDBTP
			appliedGainDB = maxAllowedGainDB
			appliedGainLinear = math.Pow(10.0, appliedGainDB/20.0)
			processedBuf = ApplyGain(buf, appliedGainLinear)
		}
	} else {
		processedBuf = ApplyGain(buf, appliedGainLinear)
	}

	// 3. Final verification measurement
	finalMetrics, err := n.meter.Measure(processedBuf)
	if err != nil {
		return nil, nil, fmt.Errorf("final loudness measure: %w", err)
	}

	report := &NormalizationReport{
		Profile:           n.cfg.Profile,
		OriginalLUFS:      initialMetrics.IntegratedLUFS,
		TargetLUFS:        targetLUFS,
		AppliedGainDB:     appliedGainDB,
		AppliedGainLinear: appliedGainLinear,
		FinalLUFS:         finalMetrics.IntegratedLUFS,
		OriginalPeakDBFS:  initialMetrics.SamplePeakDBFS,
		FinalPeakDBFS:     finalMetrics.SamplePeakDBFS,
		OriginalTruePeak:  initialMetrics.TruePeakDBTP,
		FinalTruePeak:     finalMetrics.TruePeakDBTP,
		ClippingPrevented: clippingPrevented,
		LimiterEngaged:    limiterEngaged,
	}

	return processedBuf, report, nil
}

// lookaheadLimiter performs transparent inter-sample peak limiting.
type lookaheadLimiter struct {
	lookaheadSamples int
	threshold        float64
	releaseCoeff     float64
	sampleRate       int
}

func newLookaheadLimiter(sampleRate int, ceilingDBTP float64, lookaheadMs, releaseMs float64) *lookaheadLimiter {
	lookahead := int(math.Round(lookaheadMs * 0.001 * float64(sampleRate)))
	if lookahead < 1 {
		lookahead = 1
	}
	// Reserve 0.6 dB true-peak headroom margin against inter-sample reconstruction overshoot
	threshold := math.Pow(10.0, (ceilingDBTP-0.6)/20.0)
	releaseCoeff := math.Exp(-1.0 / (float64(sampleRate) * releaseMs * 0.001))
	return &lookaheadLimiter{
		lookaheadSamples: lookahead,
		threshold:        threshold,
		releaseCoeff:     releaseCoeff,
		sampleRate:       sampleRate,
	}
}

func (lim *lookaheadLimiter) process(samples [][]float64) [][]float64 {
	channels := len(samples)
	if channels == 0 {
		return [][]float64{}
	}
	numSamples := len(samples[0])
	out := make([][]float64, channels)
	for ch := 0; ch < channels; ch++ {
		out[ch] = make([]float64, numSamples)
	}

	lookahead := lim.lookaheadSamples
	threshold := lim.threshold
	releaseCoeff := lim.releaseCoeff

	// 1. Linked peak envelope across all channels
	peakEnv := make([]float64, numSamples)
	for n := 0; n < numSamples; n++ {
		var maxAbs float64
		for ch := 0; ch < channels; ch++ {
			if a := math.Abs(samples[ch][n]); a > maxAbs {
				maxAbs = a
			}
		}
		peakEnv[n] = maxAbs
	}

	// 2. Instant target gain
	targetGain := make([]float64, numSamples)
	for n := 0; n < numSamples; n++ {
		if peakEnv[n] > threshold {
			targetGain[n] = threshold / peakEnv[n]
		} else {
			targetGain[n] = 1.0
		}
	}

	// 3. Lookahead minimum envelope
	minGain := make([]float64, numSamples)
	for n := 0; n < numSamples; n++ {
		g := targetGain[n]
		end := n + lookahead
		if end > numSamples {
			end = numSamples
		}
		for k := n; k < end; k++ {
			if targetGain[k] < g {
				g = targetGain[k]
			}
		}
		minGain[n] = g
	}

	// 4. Smooth gain curve
	smoothedGain := make([]float64, numSamples)
	currGain := 1.0
	for n := 0; n < numSamples; n++ {
		target := minGain[n]
		if target < currGain {
			currGain = target
		} else {
			currGain = target + (currGain-target)*releaseCoeff
		}
		smoothedGain[n] = currGain
	}

	// 5. Apply to delayed audio
	for ch := 0; ch < channels; ch++ {
		for n := 0; n < numSamples; n++ {
			inIdx := n - lookahead
			var inVal float64
			if inIdx >= 0 {
				inVal = samples[ch][inIdx]
			} else {
				inVal = samples[ch][0]
			}
			out[ch][n] = inVal * smoothedGain[n]
		}
	}

	return out
}

// NormalizeForStreaming normalizes a PCMBuffer to the industry standard -14.0 LUFS / -1.0 dBTP.
func NormalizeForStreaming(buf *PCMBuffer) (*PCMBuffer, *NormalizationReport, error) {
	norm, err := NewNormalizer(WithProfile(ProfileStreamingStandard))
	if err != nil {
		return nil, nil, err
	}
	return norm.Normalize(buf)
}

// NormalizeToTarget normalizes a PCMBuffer to specified TargetLUFS and CeilingDBTP.
func NormalizeToTarget(buf *PCMBuffer, targetLUFS, ceilingDBTP float64) (*PCMBuffer, *NormalizationReport, error) {
	norm, err := NewNormalizer(WithTargetLUFS(targetLUFS), WithCeilingDBTP(ceilingDBTP))
	if err != nil {
		return nil, nil, err
	}
	return norm.Normalize(buf)
}
