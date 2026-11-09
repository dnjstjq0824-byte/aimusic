// Package audio provides professional silence trimming, smart zero-crossing alignment,
// smooth envelope fade-in/fade-out, and gapless playback DSP filtering without subprocess overhead.
package audio

import (
	"errors"
	"math"
)

// FadeCurve defines mathematical envelope transition curves.
type FadeCurve string

const (
	FadeCurveLinear      FadeCurve = "LINEAR"
	FadeCurveEqualPower  FadeCurve = "EQUAL_POWER"
	FadeCurveSCurve      FadeCurve = "S_CURVE"
	FadeCurveExponential FadeCurve = "EXPONENTIAL"
)

// Trimmer error definitions.
var (
	ErrNilTrimmerBuffer = errors.New("pcm buffer is nil for silence trimming")
	ErrAllSilentAudio   = errors.New("audio track is entirely silent; trimming would result in empty audio")
	ErrInvalidFadeParam = errors.New("fade duration cannot be negative")
)

// SilenceTrimmerConfig controls silence detection and fade curve parameters.
type SilenceTrimmerConfig struct {
	ThresholdDB       float64   // Threshold in dBFS below which audio is considered silence (e.g. -60.0)
	LeadInMarginMs    float64   // Safety acoustic margin kept before first detected sound (e.g. 50ms)
	LeadOutMarginMs   float64   // Safety acoustic margin kept after last detected sound (e.g. 100ms)
	FadeInMs          float64   // Duration of smooth micro-fade-in to eliminate initial DC click
	FadeOutMs         float64   // Duration of smooth micro-fade-out to eliminate tail click
	Curve             FadeCurve // Mathematical shape of the fade envelope
	AlignZeroCrossing bool      // Snap trimming points to adjacent zero-crossing samples
	WindowMs          float64   // RMS analysis sliding window size (e.g. 10ms)
}

// DefaultTrimmerConfig returns standard broadcast and streaming trimming settings.
func DefaultTrimmerConfig() SilenceTrimmerConfig {
	return SilenceTrimmerConfig{
		ThresholdDB:       -60.0,
		LeadInMarginMs:    50.0,
		LeadOutMarginMs:   100.0,
		FadeInMs:          20.0,
		FadeOutMs:         50.0,
		Curve:             FadeCurveSCurve,
		AlignZeroCrossing: true,
		WindowMs:          10.0,
	}
}

// TrimResult contains metrics and the cleaned PCM buffer.
type TrimResult struct {
	TrimmedBuffer       *PCMBuffer `json:"-"`
	OriginalDurationSec float64    `json:"original_duration_sec"`
	TrimmedDurationSec  float64    `json:"trimmed_duration_sec"`
	LeadInTrimmedSec    float64    `json:"lead_in_trimmed_sec"`
	LeadOutTrimmedSec   float64    `json:"lead_out_trimmed_sec"`
	TrimmedSamplesStart int        `json:"trimmed_samples_start"`
	TrimmedSamplesEnd   int        `json:"trimmed_samples_end"`
	SilenceRemovedSec   float64    `json:"silence_removed_sec"`
}

// SilenceTrimmer executes silence detection, zero-crossing boundary snapping, and smart fading.
type SilenceTrimmer struct {
	cfg SilenceTrimmerConfig
}

// NewSilenceTrimmer constructs a configured silence trimmer instance.
func NewSilenceTrimmer(cfg SilenceTrimmerConfig) *SilenceTrimmer {
	if cfg.ThresholdDB >= 0.0 {
		cfg.ThresholdDB = -60.0
	}
	if cfg.WindowMs <= 0.0 {
		cfg.WindowMs = 10.0
	}
	if cfg.Curve == "" {
		cfg.Curve = FadeCurveSCurve
	}
	return &SilenceTrimmer{cfg: cfg}
}

// Trim analyzes the audio buffer, cuts leading/trailing dead air, and applies smooth micro-fades.
func (st *SilenceTrimmer) Trim(buf *PCMBuffer) (*TrimResult, error) {
	if buf == nil {
		return nil, ErrNilTrimmerBuffer
	}
	if buf.NumSamples == 0 || buf.Channels == 0 || len(buf.Samples) == 0 {
		return nil, ErrEmptyAudioBuffer
	}

	sr := buf.SampleRate
	totalSamples := buf.NumSamples
	channels := buf.Channels
	origDuration := float64(totalSamples) / float64(sr)

	// Convert threshold from dB to linear amplitude
	thresholdLinear := math.Pow(10.0, st.cfg.ThresholdDB/20.0)

	windowSamples := int(math.Round(st.cfg.WindowMs * float64(sr) / 1000.0))
	if windowSamples < 1 {
		windowSamples = 1
	}

	// Step 1: Detect start index (first window with RMS >= threshold)
	firstActiveSample := -1
	for start := 0; start < totalSamples; start += windowSamples {
		end := start + windowSamples
		if end > totalSamples {
			end = totalSamples
		}

		sumSq := 0.0
		cnt := 0
		for s := start; s < end; s++ {
			for c := 0; c < channels; c++ {
				v := buf.Samples[c][s]
				sumSq += v * v
				cnt++
			}
		}

		rms := 0.0
		if cnt > 0 {
			rms = math.Sqrt(sumSq / float64(cnt))
		}

		if rms >= thresholdLinear {
			firstActiveSample = start
			break
		}
	}

	if firstActiveSample == -1 {
		return nil, ErrAllSilentAudio
	}

	// Step 2: Detect end index (last window with RMS >= threshold)
	lastActiveSample := -1
	for start := totalSamples - windowSamples; start >= 0; start -= windowSamples {
		end := start + windowSamples
		if end > totalSamples {
			end = totalSamples
		}
		if start < 0 {
			start = 0
		}

		sumSq := 0.0
		cnt := 0
		for s := start; s < end; s++ {
			for c := 0; c < channels; c++ {
				v := buf.Samples[c][s]
				sumSq += v * v
				cnt++
			}
		}

		rms := 0.0
		if cnt > 0 {
			rms = math.Sqrt(sumSq / float64(cnt))
		}

		if rms >= thresholdLinear {
			lastActiveSample = end
			break
		}
		if start == 0 {
			break
		}
	}

	if lastActiveSample == -1 {
		lastActiveSample = totalSamples
	}
	if lastActiveSample < firstActiveSample {
		lastActiveSample = firstActiveSample + windowSamples
	}
	if lastActiveSample > totalSamples {
		lastActiveSample = totalSamples
	}

	// Step 3: Apply safety margins
	leadInMarginSamples := int(math.Round(st.cfg.LeadInMarginMs * float64(sr) / 1000.0))
	leadOutMarginSamples := int(math.Round(st.cfg.LeadOutMarginMs * float64(sr) / 1000.0))

	cutStart := firstActiveSample - leadInMarginSamples
	if cutStart < 0 {
		cutStart = 0
	}

	cutEnd := lastActiveSample + leadOutMarginSamples
	if cutEnd > totalSamples {
		cutEnd = totalSamples
	}

	// Step 4: Snap to nearest zero-crossing sample
	if st.cfg.AlignZeroCrossing && cutStart > 0 {
		cutStart = FindZeroCrossing(buf.Samples[0], cutStart, int(0.005*float64(sr))) // 5ms search radius
	}
	if st.cfg.AlignZeroCrossing && cutEnd < totalSamples {
		cutEnd = FindZeroCrossing(buf.Samples[0], cutEnd, int(0.005*float64(sr)))
	}

	if cutEnd <= cutStart {
		cutEnd = cutStart + 1
	}

	trimmedSampleCount := cutEnd - cutStart
	trimmedSamples := make([][]float64, channels)
	for c := 0; c < channels; c++ {
		trimmedSamples[c] = make([]float64, trimmedSampleCount)
		copy(trimmedSamples[c], buf.Samples[c][cutStart:cutEnd])
	}

	trimmedBuf := &PCMBuffer{
		Channels:   channels,
		SampleRate: sr,
		NumSamples: trimmedSampleCount,
		Samples:    trimmedSamples,
	}

	// Step 5: Apply smooth micro-fade-in and micro-fade-out
	if st.cfg.FadeInMs > 0 {
		_ = ApplyFadeIn(trimmedBuf, st.cfg.FadeInMs, st.cfg.Curve)
	}
	if st.cfg.FadeOutMs > 0 {
		_ = ApplyFadeOut(trimmedBuf, st.cfg.FadeOutMs, st.cfg.Curve)
	}

	trimmedDuration := float64(trimmedSampleCount) / float64(sr)
	leadInSec := float64(cutStart) / float64(sr)
	leadOutSec := float64(totalSamples-cutEnd) / float64(sr)

	return &TrimResult{
		TrimmedBuffer:       trimmedBuf,
		OriginalDurationSec: origDuration,
		TrimmedDurationSec:  trimmedDuration,
		LeadInTrimmedSec:    leadInSec,
		LeadOutTrimmedSec:   leadOutSec,
		TrimmedSamplesStart: cutStart,
		TrimmedSamplesEnd:   totalSamples - cutEnd,
		SilenceRemovedSec:   origDuration - trimmedDuration,
	}, nil
}

// FindZeroCrossing searches for the closest sample index with sign change or near-zero amplitude.
func FindZeroCrossing(samples []float64, targetIdx int, searchRadius int) int {
	n := len(samples)
	if n == 0 || targetIdx < 0 || targetIdx >= n {
		return targetIdx
	}

	minIdx := targetIdx - searchRadius
	if minIdx < 1 {
		minIdx = 1
	}
	maxIdx := targetIdx + searchRadius
	if maxIdx >= n {
		maxIdx = n - 1
	}

	bestIdx := targetIdx
	minAbs := math.Abs(samples[targetIdx])

	for i := minIdx; i <= maxIdx; i++ {
		// Strict zero crossing detection: consecutive samples with opposite signs
		if (samples[i-1] <= 0 && samples[i] >= 0) || (samples[i-1] >= 0 && samples[i] <= 0) {
			return i
		}
		currAbs := math.Abs(samples[i])
		if currAbs < minAbs {
			minAbs = currAbs
			bestIdx = i
		}
	}
	return bestIdx
}

// CalculateCurveWeight evaluates the mathematical fade gain in range [0.0, 1.0].
func CalculateCurveWeight(progress float64, curve FadeCurve, isFadeIn bool) float64 {
	p := math.Max(0.0, math.Min(1.0, progress))

	if curve == FadeCurveEqualPower {
		// Quarter sine/cosine cycle preserves constant acoustic power: sin^2(x) + cos^2(x) = 1
		if isFadeIn {
			return math.Sin(p * (math.Pi / 2.0))
		}
		return math.Cos(p * (math.Pi / 2.0))
	}

	var gain float64
	switch curve {
	case FadeCurveLinear:
		gain = p
	case FadeCurveSCurve:
		// Raised cosine / Hann window S-curve (zero derivative at edges)
		gain = 0.5 * (1.0 - math.Cos(p*math.Pi))
	case FadeCurveExponential:
		// Smooth exponential curve: (exp(3*p) - 1) / (exp(3) - 1)
		gain = (math.Exp(3.0*p) - 1.0) / (math.Exp(3.0) - 1.0)
	default:
		gain = p
	}

	if !isFadeIn {
		gain = 1.0 - gain
	}
	return math.Max(0.0, math.Min(1.0, gain))
}

// ApplyFadeIn ramps audio amplitude from 0.0 to 1.0 over the specified duration.
func ApplyFadeIn(buf *PCMBuffer, durationMs float64, curve FadeCurve) error {
	if buf == nil {
		return ErrNilTrimmerBuffer
	}
	if durationMs < 0 {
		return ErrInvalidFadeParam
	}
	if durationMs == 0 || buf.NumSamples == 0 {
		return nil
	}

	fadeSamples := int(math.Round(durationMs * float64(buf.SampleRate) / 1000.0))
	if fadeSamples > buf.NumSamples {
		fadeSamples = buf.NumSamples
	}
	if fadeSamples == 0 {
		return nil
	}

	denom := float64(fadeSamples)
	if fadeSamples > 1 {
		denom = float64(fadeSamples - 1)
	}

	for s := 0; s < fadeSamples; s++ {
		progress := float64(s) / denom
		weight := CalculateCurveWeight(progress, curve, true)

		for c := 0; c < buf.Channels; c++ {
			buf.Samples[c][s] *= weight
		}
	}
	return nil
}

// ApplyFadeOut ramps audio amplitude from 1.0 to 0.0 at the end of the buffer.
func ApplyFadeOut(buf *PCMBuffer, durationMs float64, curve FadeCurve) error {
	if buf == nil {
		return ErrNilTrimmerBuffer
	}
	if durationMs < 0 {
		return ErrInvalidFadeParam
	}
	if durationMs == 0 || buf.NumSamples == 0 {
		return nil
	}

	fadeSamples := int(math.Round(durationMs * float64(buf.SampleRate) / 1000.0))
	if fadeSamples > buf.NumSamples {
		fadeSamples = buf.NumSamples
	}
	if fadeSamples == 0 {
		return nil
	}

	startSample := buf.NumSamples - fadeSamples
	denom := float64(fadeSamples)
	if fadeSamples > 1 {
		denom = float64(fadeSamples - 1)
	}

	for s := 0; s < fadeSamples; s++ {
		progress := float64(s) / denom
		weight := CalculateCurveWeight(progress, curve, false)
		targetIdx := startSample + s

		for c := 0; c < buf.Channels; c++ {
			buf.Samples[c][targetIdx] *= weight
		}
	}
	return nil
}

// QuickTrim trims an audio buffer using standard broadcast defaults.
func QuickTrim(buf *PCMBuffer) (*TrimResult, error) {
	trimmer := NewSilenceTrimmer(DefaultTrimmerConfig())
	return trimmer.Trim(buf)
}
