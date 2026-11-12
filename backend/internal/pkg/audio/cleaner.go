// Package audio provides high-precision DC offset removal, 5Hz sub-bass preserving IIR DC blocking,
// and statistical outlier impulse de-clicking / de-popping DSP filters for AI-generated music.
package audio

import (
	"errors"
	"math"
)

// Cleaner validation errors.
var (
	ErrNilCleanBuffer        = errors.New("pcm buffer is nil for audio cleaner")
	ErrEmptyAudioCleanBuffer = errors.New("pcm buffer contains no samples to clean")
	ErrInvalidCutoffHz       = errors.New("cutoff frequency must be positive and below nyquist")
)

// DCBlockerConfig specifies parameters for removing stationary and slowly drifting direct current bias.
type DCBlockerConfig struct {
	CutoffHz           float64 // Sub-bass highpass cutoff (typically 2.0 to 10.0 Hz, default 5.0 Hz)
	UseMeanSubtraction bool    // If true, applies global mean subtraction prior to IIR filter
}

// DefaultDCBlockerConfig returns standard mastering DC blocking configuration.
func DefaultDCBlockerConfig() DCBlockerConfig {
	return DCBlockerConfig{
		CutoffHz:           5.0,
		UseMeanSubtraction: true,
	}
}

// DeclickerConfig configures statistical impulse and pop detection and reconstruction.
type DeclickerConfig struct {
	ThresholdSigma float64 // Outlier threshold in standard deviations of second difference (default 5.0)
	MaxClickLength int     // Maximum contiguous samples to interpolate (default 4)
	WindowSize     int     // Local statistics calculation window (default 64)
}

// DefaultDeclickerConfig returns standard de-clicking parameters.
func DefaultDeclickerConfig() DeclickerConfig {
	return DeclickerConfig{
		ThresholdSigma: 5.0,
		MaxClickLength: 4,
		WindowSize:     64,
	}
}

// CleanReport summarizes defects detected and repaired during audio cleaning.
type CleanReport struct {
	CleanedBuffer     *PCMBuffer `json:"-"`
	OriginalDC        []float64  `json:"original_dc"`
	ResidualDC        []float64  `json:"residual_dc"`
	ClicksDetected    int        `json:"clicks_detected"`
	ClicksRepaired    int        `json:"clicks_repaired"`
	MaxResidualDCBias float64    `json:"max_residual_dc_bias"` // Maximum channel residual DC in percentage
	DCCorrected       bool       `json:"dc_corrected"`
}

// AudioCleaner coordinates DC offset elimination and click/pop removal.
type AudioCleaner struct {
	dcCfg    DCBlockerConfig
	clickCfg DeclickerConfig
}

// NewAudioCleaner constructs an audio cleaner with specified configurations.
func NewAudioCleaner(dcCfg DCBlockerConfig, clickCfg DeclickerConfig) *AudioCleaner {
	if dcCfg.CutoffHz <= 0.0 {
		dcCfg.CutoffHz = 5.0
	}
	if clickCfg.ThresholdSigma <= 0.0 {
		clickCfg.ThresholdSigma = 5.0
	}
	if clickCfg.MaxClickLength <= 0 {
		clickCfg.MaxClickLength = 4
	}
	if clickCfg.WindowSize <= 0 {
		clickCfg.WindowSize = 64
	}
	return &AudioCleaner{
		dcCfg:    dcCfg,
		clickCfg: clickCfg,
	}
}

// Clean removes DC bias and repairs transient click/pop defects on the input audio buffer.
func (ac *AudioCleaner) Clean(buf *PCMBuffer) (*CleanReport, error) {
	if buf == nil {
		return nil, ErrNilCleanBuffer
	}
	if buf.NumSamples == 0 || buf.Channels == 0 || len(buf.Samples) == 0 {
		return nil, ErrEmptyAudioCleanBuffer
	}

	channels := buf.Channels
	numSamples := buf.NumSamples
	sr := buf.SampleRate

	// Clone buffer for non-destructive processing
	cleanedSamples := make([][]float64, channels)
	origDC := make([]float64, channels)

	for c := 0; c < channels; c++ {
		cleanedSamples[c] = make([]float64, numSamples)
		copy(cleanedSamples[c], buf.Samples[c])

		// Calculate initial DC offset
		sum := 0.0
		for s := 0; s < numSamples; s++ {
			sum += buf.Samples[c][s]
		}
		origDC[c] = sum / float64(numSamples)
	}

	cleanedBuf := &PCMBuffer{
		Channels:   channels,
		SampleRate: sr,
		NumSamples: numSamples,
		Samples:    cleanedSamples,
	}

	// 1. Apply Click and Pop Repair first so impulse spikes do not pollute DC calculation
	clicksDetected, clicksRepaired := RepairClicksAndPops(cleanedBuf, ac.clickCfg)

	// 2. Apply DC Blocking and Zero-Mean Centering
	if err := RemoveDCOffsetAdvanced(cleanedBuf, ac.dcCfg); err != nil {
		return nil, err
	}

	// Measure residual DC bias
	residualDC := make([]float64, channels)
	maxResidual := 0.0
	for c := 0; c < channels; c++ {
		sum := 0.0
		for s := 0; s < numSamples; s++ {
			sum += cleanedBuf.Samples[c][s]
		}
		res := sum / float64(numSamples)
		residualDC[c] = res
		if math.Abs(res) > maxResidual {
			maxResidual = math.Abs(res)
		}
	}

	// DC bias in percentage (e.g. 0.0001 = 0.01%)
	biasPct := maxResidual * 100.0

	return &CleanReport{
		CleanedBuffer:     cleanedBuf,
		OriginalDC:        origDC,
		ResidualDC:        residualDC,
		ClicksDetected:    clicksDetected,
		ClicksRepaired:    clicksRepaired,
		MaxResidualDCBias: biasPct,
		DCCorrected:       biasPct <= 0.001, // 0.001% compliance check
	}, nil
}

// RemoveDCOffsetAdvanced applies global zero-mean centering and a 1st-order IIR DC blocker filter:
// y[n] = x[n] - x[n-1] + R * y[n-1], where R = 1 - (2 * pi * fc / fs).
func RemoveDCOffsetAdvanced(buf *PCMBuffer, cfg DCBlockerConfig) error {
	if buf == nil {
		return ErrNilCleanBuffer
	}
	if buf.NumSamples == 0 || buf.Channels == 0 {
		return ErrEmptyAudioCleanBuffer
	}

	sr := float64(buf.SampleRate)
	nyquist := sr / 2.0
	if cfg.CutoffHz <= 0 || cfg.CutoffHz >= nyquist {
		return ErrInvalidCutoffHz
	}

	// R coefficient for DC blocker pole
	r := 1.0 - (2.0 * math.Pi * cfg.CutoffHz / sr)
	if r > 0.9999 {
		r = 0.9999
	}
	if r < 0.9000 {
		r = 0.9000
	}

	channels := buf.Channels
	numSamples := buf.NumSamples

	for c := 0; c < channels; c++ {
		samples := buf.Samples[c]

		// Step A: Subtract arithmetic mean
		if cfg.UseMeanSubtraction {
			sum := 0.0
			for s := 0; s < numSamples; s++ {
				sum += samples[s]
			}
			mean := sum / float64(numSamples)
			for s := 0; s < numSamples; s++ {
				samples[s] -= mean
			}
		}

		// Step B: IIR DC-blocking filter
		xPrev := 0.0
		yPrev := 0.0
		for s := 0; s < numSamples; s++ {
			xCurr := samples[s]
			yCurr := xCurr - xPrev + r*yPrev
			samples[s] = yCurr
			xPrev = xCurr
			yPrev = yCurr
		}

		// Second pass (reverse) to eliminate phase distortion (zero-phase filtering)
		xPrev = 0.0
		yPrev = 0.0
		for s := numSamples - 1; s >= 0; s-- {
			xCurr := samples[s]
			yCurr := xCurr - xPrev + r*yPrev
			samples[s] = yCurr
			xPrev = xCurr
			yPrev = yCurr
		}

		// Step C: Exact zero-bias recalibration post-filter settling (< 10^-12)
		if cfg.UseMeanSubtraction {
			finalSum := 0.0
			for s := 0; s < numSamples; s++ {
				finalSum += samples[s]
			}
			finalMean := finalSum / float64(numSamples)
			for s := 0; s < numSamples; s++ {
				samples[s] -= finalMean
			}
		}
	}
	return nil
}

// RepairClicksAndPops scans audio channels for isolated impulse outliers and replaces them via cubic interpolation.
func RepairClicksAndPops(buf *PCMBuffer, cfg DeclickerConfig) (int, int) {
	if buf == nil || buf.NumSamples < 5 {
		return 0, 0
	}

	totalDetected := 0
	totalRepaired := 0

	channels := buf.Channels
	numSamples := buf.NumSamples
	win := cfg.WindowSize
	if win < 16 {
		win = 16
	}

	for c := 0; c < channels; c++ {
		samples := buf.Samples[c]

		// Second difference: d2[i] = s[i+1] - 2*s[i] + s[i-1]
		d2 := make([]float64, numSamples)
		for i := 1; i < numSamples-1; i++ {
			d2[i] = samples[i+1] - 2.0*samples[i] + samples[i-1]
		}

		// Sliding window local standard deviation of d2
		for i := 2; i < numSamples-2; i++ {
			start := i - win/2
			if start < 1 {
				start = 1
			}
			end := i + win/2
			if end > numSamples-1 {
				end = numSamples - 1
			}

			// Local mean of absolute d2
			sumAbs := 0.0
			cnt := 0
			for j := start; j < end; j++ {
				sumAbs += math.Abs(d2[j])
				cnt++
			}
			if cnt == 0 {
				continue
			}
			localThreshold := (sumAbs / float64(cnt)) * cfg.ThresholdSigma
			if localThreshold < 0.05 {
				localThreshold = 0.05 // Noise floor safeguard
			}

			// Outlier impulse spike detection
			if math.Abs(d2[i]) > localThreshold {
				totalDetected++

				// Check click width
				clickLen := 1
				for k := 1; k < cfg.MaxClickLength && (i+k) < numSamples-2; k++ {
					if math.Abs(d2[i+k]) > localThreshold*0.8 {
						clickLen++
					} else {
						break
					}
				}

				// Hermite cubic spline interpolation across the corrupted interval [i, i+clickLen-1]
				idx0 := i - 2
				idx1 := i - 1
				idx2 := i + clickLen
				idx3 := i + clickLen + 1

				if idx0 >= 0 && idx3 < numSamples {
					y0 := samples[idx0]
					y1 := samples[idx1]
					y2 := samples[idx2]
					y3 := samples[idx3]

					for k := 0; k < clickLen; k++ {
						tVal := float64(k+1) / float64(clickLen+1)
						interpVal := cubicHermiteInterpolate(y0, y1, y2, y3, tVal)
						samples[i+k] = interpVal
						totalRepaired++
					}
				}
				i += clickLen // Skip past repaired interval
			}
		}
	}

	return totalDetected, totalRepaired
}

// cubicHermiteInterpolate evaluates a 4-point cubic Hermite spline at normalized position t in [0.0, 1.0].
func cubicHermiteInterpolate(y0, y1, y2, y3, t float64) float64 {
	a := -0.5*y0 + 1.5*y1 - 1.5*y2 + 0.5*y3
	b := y0 - 2.5*y1 + 2.0*y2 - 0.5*y3
	c := -0.5*y0 + 0.5*y2
	d := y1

	return a*t*t*t + b*t*t + c*t + d
}
