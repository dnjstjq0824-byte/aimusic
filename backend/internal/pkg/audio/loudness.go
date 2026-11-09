// Package audio provides professional ITU-R BS.1770-4 and EBU R128
// compliant audio loudness measurement (LUFS / LKFS), Loudness Range (LRA),
// Momentary/Short-term loudness, and True Peak detection.
package audio

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

// Standard loudness targets and gating thresholds.
const (
	AbsoluteGateThresholdLUFS = -70.0
	RelativeGateOffsetLU      = -10.0
	LRARelativeGateOffsetLU   = -20.0

	TargetLUFSEBUR128    = -23.0
	TargetLUFSStreaming  = -14.0
	TargetLUFSAppleMusic = -16.0

	LoudnessMinDB = -144.0
)

// Loudness validation errors.
var (
	ErrNilLoudnessBuffer = errors.New("pcm buffer is nil for loudness measurement")
	ErrEmptyAudioBuffer  = errors.New("pcm buffer contains no channels or samples")
)

// LoudnessResult contains comprehensive EBU R128 / ITU-R BS.1770-4 metrics.
type LoudnessResult struct {
	IntegratedLUFS float64
	MomentaryMax   float64
	ShortTermMax   float64
	LoudnessRange  float64
	SamplePeakDBFS float64
	TruePeakDBTP   float64
	DurationSec    float64
}

// biquad represents a Direct Form II Transposed 2nd-order IIR digital filter.
type biquad struct {
	b0, b1, b2 float64
	a1, a2     float64
	s1, s2     float64
}

func (f *biquad) step(x float64) float64 {
	y := f.b0*x + f.s1
	f.s1 = f.b1*x - f.a1*y + f.s2
	f.s2 = f.b2*x - f.a2*y
	return y
}

// newPreFilter creates the ITU-R BS.1770 Stage 1 head-diffraction pre-filter at 48kHz.
func newPreFilter() *biquad {
	return &biquad{
		b0: 1.53512485958697,
		b1: -2.69169618940638,
		b2: 1.19839281085285,
		a1: -1.69065929318241,
		a2: 0.73248077421585,
	}
}

// newRLBFilter creates the ITU-R BS.1770 Stage 2 Revised Low-frequency B-weighting filter at 48kHz.
func newRLBFilter() *biquad {
	return &biquad{
		b0: 1.0,
		b1: -2.0,
		b2: 1.0,
		a1: -1.99004745483398,
		a2: 0.99007225036621,
	}
}

// channelWeight returns the ITU-R BS.1770 energy weighting factor for channel index.
// 0: Left (1.0), 1: Right (1.0), 2: Center (1.0), 3: Left Surround (1.41), 4: Right Surround (1.41).
func channelWeight(channelIndex int) float64 {
	switch channelIndex {
	case 0, 1, 2:
		return 1.0
	case 3, 4:
		return 1.41
	default:
		return 1.0
	}
}

// LoudnessMeter performs loudness metering according to ITU-R BS.1770-4 and EBU R128.
type LoudnessMeter struct {
	enableTruePeak bool
}

// LoudnessOption configures measurement behavior.
type LoudnessOption func(*LoudnessMeter)

// WithTruePeak enables 4x oversampled True Peak detection.
func WithTruePeak(enable bool) LoudnessOption {
	return func(m *LoudnessMeter) {
		m.enableTruePeak = enable
	}
}

// NewLoudnessMeter constructs a configured loudness measurement meter.
func NewLoudnessMeter(opts ...LoudnessOption) *LoudnessMeter {
	m := &LoudnessMeter{
		enableTruePeak: true,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Measure analyzes a PCMBuffer and returns full EBU R128 loudness metrics.
func (m *LoudnessMeter) Measure(buf *PCMBuffer) (*LoudnessResult, error) {
	if buf == nil {
		return nil, ErrNilLoudnessBuffer
	}
	if buf.Channels <= 0 || buf.NumSamples <= 0 || len(buf.Samples) == 0 {
		return nil, ErrEmptyAudioBuffer
	}

	workBuf := buf
	if buf.SampleRate != DefaultTargetSampleRate {
		resampled, err := ResampleTo48kHz(buf)
		if err != nil {
			return nil, fmt.Errorf("resample to 48kHz for loudness measurement: %w", err)
		}
		workBuf = resampled
	}

	channels := workBuf.Channels
	numSamples := workBuf.NumSamples
	sampleRate := workBuf.SampleRate
	durationSec := float64(numSamples) / float64(sampleRate)

	filtered := make([][]float64, channels)
	for ch := 0; ch < channels; ch++ {
		filtered[ch] = make([]float64, numSamples)
		pre := newPreFilter()
		rlb := newRLBFilter()
		for n := 0; n < numSamples; n++ {
			s := pre.step(workBuf.Samples[ch][n])
			filtered[ch][n] = rlb.step(s)
		}
	}

	blockSamples := int(math.Round(0.400 * float64(sampleRate))) // 400ms = 19200
	hopSamples := int(math.Round(0.100 * float64(sampleRate)))   // 100ms = 4800

	if blockSamples <= 0 {
		blockSamples = 1
	}
	if hopSamples <= 0 {
		hopSamples = 1
	}

	var blockPowers []float64
	var blockLoudness []float64
	momentaryMax := LoudnessMinDB

	if numSamples < blockSamples {
		var totalPower float64
		for ch := 0; ch < channels; ch++ {
			var chSum float64
			for n := 0; n < numSamples; n++ {
				v := filtered[ch][n]
				chSum += v * v
			}
			totalPower += channelWeight(ch) * (chSum / float64(numSamples))
		}
		lk := LoudnessMinDB
		if totalPower > 1e-14 {
			lk = -0.691 + 10.0*math.Log10(totalPower)
		}
		blockPowers = append(blockPowers, totalPower)
		blockLoudness = append(blockLoudness, lk)
		momentaryMax = lk
	} else {
		numBlocks := (numSamples-blockSamples)/hopSamples + 1
		blockPowers = make([]float64, numBlocks)
		blockLoudness = make([]float64, numBlocks)

		for j := 0; j < numBlocks; j++ {
			start := j * hopSamples
			var powerSum float64
			for ch := 0; ch < channels; ch++ {
				w := channelWeight(ch)
				var chSum float64
				for k := 0; k < blockSamples; k++ {
					v := filtered[ch][start+k]
					chSum += v * v
				}
				powerSum += w * (chSum / float64(blockSamples))
			}
			blockPowers[j] = powerSum
			lk := LoudnessMinDB
			if powerSum > 1e-14 {
				lk = -0.691 + 10.0*math.Log10(powerSum)
			}
			blockLoudness[j] = lk
			if lk > momentaryMax {
				momentaryMax = lk
			}
		}
	}

	integratedLUFS := calculateIntegratedLoudness(blockPowers, blockLoudness)

	shortTermMax := LoudnessMinDB
	shortTermWindowSamples := int(math.Round(3.000 * float64(sampleRate))) // 3s = 144000
	var shortTermLoudnesses []float64

	if numSamples < shortTermWindowSamples {
		shortTermMax = integratedLUFS
		if integratedLUFS > AbsoluteGateThresholdLUFS {
			shortTermLoudnesses = append(shortTermLoudnesses, integratedLUFS)
		}
	} else {
		numSTBlocks := (numSamples-shortTermWindowSamples)/hopSamples + 1
		for j := 0; j < numSTBlocks; j++ {
			start := j * hopSamples
			var stPower float64
			for ch := 0; ch < channels; ch++ {
				w := channelWeight(ch)
				var chSum float64
				for k := 0; k < shortTermWindowSamples; k++ {
					v := filtered[ch][start+k]
					chSum += v * v
				}
				stPower += w * (chSum / float64(shortTermWindowSamples))
			}
			stLoudness := LoudnessMinDB
			if stPower > 1e-14 {
				stLoudness = -0.691 + 10.0*math.Log10(stPower)
			}
			if stLoudness > shortTermMax {
				shortTermMax = stLoudness
			}
			if stLoudness > AbsoluteGateThresholdLUFS {
				shortTermLoudnesses = append(shortTermLoudnesses, stLoudness)
			}
		}
	}

	lra := calculateLRA(shortTermLoudnesses)

	var maxSamplePeak float64
	for ch := 0; ch < channels; ch++ {
		for n := 0; n < numSamples; n++ {
			val := math.Abs(workBuf.Samples[ch][n])
			if val > maxSamplePeak {
				maxSamplePeak = val
			}
		}
	}
	samplePeakDBFS := LoudnessMinDB
	if maxSamplePeak > 1e-12 {
		samplePeakDBFS = 20.0 * math.Log10(maxSamplePeak)
	}

	truePeakDBTP := samplePeakDBFS
	if m.enableTruePeak {
		tp, err := measureTruePeak4x(workBuf)
		if err == nil {
			truePeakDBTP = tp
		}
	}

	return &LoudnessResult{
		IntegratedLUFS: integratedLUFS,
		MomentaryMax:   momentaryMax,
		ShortTermMax:   shortTermMax,
		LoudnessRange:  lra,
		SamplePeakDBFS: samplePeakDBFS,
		TruePeakDBTP:   truePeakDBTP,
		DurationSec:    durationSec,
	}, nil
}

// calculateIntegratedLoudness applies the two-stage (absolute + relative) gating algorithm.
func calculateIntegratedLoudness(powers, loudness []float64) float64 {
	var absGateIndices []int
	var absSumPower float64

	for idx, l := range loudness {
		if l > AbsoluteGateThresholdLUFS {
			absGateIndices = append(absGateIndices, idx)
			absSumPower += powers[idx]
		}
	}

	if len(absGateIndices) == 0 {
		return LoudnessMinDB
	}

	meanAbsPower := absSumPower / float64(len(absGateIndices))
	relativeThreshold := -0.691 + 10.0*math.Log10(meanAbsPower) + RelativeGateOffsetLU

	var relSumPower float64
	var relCount int
	for _, idx := range absGateIndices {
		if loudness[idx] > relativeThreshold {
			relSumPower += powers[idx]
			relCount++
		}
	}

	if relCount == 0 {
		return LoudnessMinDB
	}

	meanRelPower := relSumPower / float64(relCount)
	if meanRelPower <= 1e-14 {
		return LoudnessMinDB
	}

	return -0.691 + 10.0*math.Log10(meanRelPower)
}

// calculateLRA computes the Loudness Range per EBU Tech 3342.
func calculateLRA(stLoudness []float64) float64 {
	if len(stLoudness) < 2 {
		return 0.0
	}

	var sumLin float64
	for _, v := range stLoudness {
		sumLin += math.Pow(10.0, v/10.0)
	}
	meanLin := sumLin / float64(len(stLoudness))
	relativeThreshold := 10.0*math.Log10(meanLin) + LRARelativeGateOffsetLU

	var gated []float64
	for _, v := range stLoudness {
		if v > relativeThreshold {
			gated = append(gated, v)
		}
	}

	if len(gated) < 2 {
		return 0.0
	}

	sort.Float64s(gated)

	p10Idx := int(math.Round(float64(len(gated)-1) * 0.10))
	p95Idx := int(math.Round(float64(len(gated)-1) * 0.95))

	if p10Idx < 0 {
		p10Idx = 0
	}
	if p95Idx >= len(gated) {
		p95Idx = len(gated) - 1
	}

	lra := gated[p95Idx] - gated[p10Idx]
	if lra < 0.0 {
		return 0.0
	}
	return lra
}

// measureTruePeak4x calculates inter-sample peak via 4x bandlimited sinc interpolation.
func measureTruePeak4x(buf *PCMBuffer) (float64, error) {
	targetRate := buf.SampleRate * 4
	resampler, err := NewResampler(buf.SampleRate, targetRate, WithQuality(QualityStandard))
	if err != nil {
		return 0.0, err
	}
	oversampled, err := resampler.ResampleBuffer(buf)
	if err != nil {
		return 0.0, err
	}

	var maxPeak float64
	for ch := 0; ch < oversampled.Channels; ch++ {
		for n := 0; n < oversampled.NumSamples; n++ {
			val := math.Abs(oversampled.Samples[ch][n])
			if val > maxPeak {
				maxPeak = val
			}
		}
	}

	if maxPeak <= 1e-12 {
		return LoudnessMinDB, nil
	}
	return 20.0 * math.Log10(maxPeak), nil
}

// MeasureLUFS is a top-level helper that performs full EBU R128 loudness analysis on a PCMBuffer.
func MeasureLUFS(buf *PCMBuffer) (*LoudnessResult, error) {
	meter := NewLoudnessMeter()
	return meter.Measure(buf)
}

// CalculateNormalizationGain returns the linear gain multiplier to achieve targetLUFS from currentLUFS.
func CalculateNormalizationGain(currentLUFS, targetLUFS float64) float64 {
	diffDB := targetLUFS - currentLUFS
	return math.Pow(10.0, diffDB/20.0)
}

// ApplyGain scales all samples in a PCMBuffer by the linear multiplier, returning a new buffer.
func ApplyGain(buf *PCMBuffer, gain float64) *PCMBuffer {
	if buf == nil {
		return nil
	}
	outSamples := make([][]float64, buf.Channels)
	for ch := 0; ch < buf.Channels; ch++ {
		outSamples[ch] = make([]float64, buf.NumSamples)
		for n := 0; n < buf.NumSamples; n++ {
			outSamples[ch][n] = buf.Samples[ch][n] * gain
		}
	}
	return &PCMBuffer{
		Channels:   buf.Channels,
		SampleRate: buf.SampleRate,
		NumSamples: buf.NumSamples,
		Samples:    outSamples,
	}
}
