// Package audio provides noise floor estimation, 50Hz/60Hz AC mains hum detection,
// and automated quality gate filtering (< -70 dBFS noise floor specification).
package audio

import (
	"encoding/json"
	"errors"
	"math"
	"sort"
)

var (
	ErrTooShortForNoiseAnalysis = errors.New("audio duration too short for noise floor analysis")
)

// NoiseFloorConfig configures noise floor and mains hum measurement.
type NoiseFloorConfig struct {
	WindowMs           float64 `json:"window_ms"`             // Sliding window size in ms, default: 100ms
	QuietPercentile    float64 `json:"quiet_percentile"`      // Bottom percentile to designate as silence floor (e.g. 0.10)
	QualityLimitDbFS   float64 `json:"quality_limit_dbfs"`    // Max acceptable noise floor, default: -70.0 dBFS
	HumThresholdDiffDb float64 `json:"hum_threshold_diff_db"` // dB above noise floor to flag distinct hum (default: 6.0 dB)
}

// DefaultNoiseFloorConfig returns standard studio acoustic compliance parameters.
func DefaultNoiseFloorConfig() NoiseFloorConfig {
	return NoiseFloorConfig{
		WindowMs:           100.0,
		QuietPercentile:    0.10,
		QualityLimitDbFS:   -70.0,
		HumThresholdDiffDb: 6.0,
	}
}

// NoiseFloorResult contains estimated floor levels, hum metrics, and pass/fail gate status.
type NoiseFloorResult struct {
	NoiseFloorDbFS     float64 `json:"noise_floor_dbfs"`
	PassesQualityGate  bool    `json:"passes_quality_gate"` // True if NoiseFloorDbFS <= QualityLimitDbFS (-70dBFS)
	Hum50HzDbFS        float64 `json:"hum_50hz_dbfs"`
	Hum60HzDbFS        float64 `json:"hum_60hz_dbfs"`
	Has50HzHum         bool    `json:"has_50hz_hum"`
	Has60HzHum         bool    `json:"has_60hz_hum"`
	QuietDurationMs    float64 `json:"quiet_duration_ms"`
	AnalyzedDurationMs float64 `json:"analyzed_duration_ms"`
}

// NoiseFloorAnalyzer measures residual room noise, ADC hiss, and electrical hum.
type NoiseFloorAnalyzer struct {
	cfg NoiseFloorConfig
}

// NewNoiseFloorAnalyzer creates a new noise floor analyzer.
func NewNoiseFloorAnalyzer(cfg NoiseFloorConfig) *NoiseFloorAnalyzer {
	if cfg.WindowMs <= 0 {
		cfg.WindowMs = 100.0
	}
	if cfg.QuietPercentile <= 0 || cfg.QuietPercentile > 0.5 {
		cfg.QuietPercentile = 0.10
	}
	if cfg.QualityLimitDbFS >= 0 {
		cfg.QualityLimitDbFS = -70.0
	}
	if cfg.HumThresholdDiffDb <= 0 {
		cfg.HumThresholdDiffDb = 6.0
	}
	return &NoiseFloorAnalyzer{cfg: cfg}
}

type windowMetric struct {
	rms        float64
	startIndex int
	endIndex   int
}

// Analyze estimates background noise floor and calculates 50Hz/60Hz hum prominence.
func (na *NoiseFloorAnalyzer) Analyze(input *PCMBuffer) (*NoiseFloorResult, error) {
	if input == nil {
		return nil, ErrNilPCMBuffer
	}
	if input.NumSamples == 0 || input.Channels == 0 || len(input.Samples) == 0 {
		return nil, ErrEmptyPCMBuffer
	}

	numSamples := input.NumSamples
	sampleRate := input.SampleRate
	durationMs := (float64(numSamples) / float64(sampleRate)) * 1000.0

	samplesPerWindow := int((na.cfg.WindowMs / 1000.0) * float64(sampleRate))
	if samplesPerWindow <= 0 {
		samplesPerWindow = 1024
	}
	if numSamples < samplesPerWindow {
		return nil, ErrTooShortForNoiseAnalysis
	}

	// Mix channels to mono for floor estimation
	mono := make([]float64, numSamples)
	channels := input.Channels
	scale := 1.0 / float64(channels)
	for i := 0; i < numSamples; i++ {
		var sum float64
		for ch := 0; ch < channels; ch++ {
			sum += input.Samples[ch][i]
		}
		mono[i] = sum * scale
	}

	// Compute RMS for each sliding window
	numWindows := numSamples / samplesPerWindow
	windows := make([]windowMetric, numWindows)

	for w := 0; w < numWindows; w++ {
		start := w * samplesPerWindow
		end := start + samplesPerWindow
		var sumSq float64
		for i := start; i < end; i++ {
			s := mono[i]
			sumSq += s * s
		}
		rms := math.Sqrt(sumSq / float64(samplesPerWindow))
		windows[w] = windowMetric{
			rms:        rms,
			startIndex: start,
			endIndex:   end,
		}
	}

	// Sort windows by RMS energy ascending
	sort.Slice(windows, func(i, j int) bool {
		return windows[i].rms < windows[j].rms
	})

	// Select quietest percentile
	kCount := int(math.Ceil(float64(numWindows) * na.cfg.QuietPercentile))
	if kCount < 1 {
		kCount = 1
	}

	// Calculate noise floor RMS across quietest windows
	var quietSumSq float64
	var quietTotalSamples int
	for i := 0; i < kCount; i++ {
		w := windows[i]
		for s := w.startIndex; s < w.endIndex; s++ {
			v := mono[s]
			quietSumSq += v * v
			quietTotalSamples++
		}
	}

	noiseRMS := 1e-6
	if quietTotalSamples > 0 {
		noiseRMS = math.Sqrt(quietSumSq / float64(quietTotalSamples))
	}
	noiseFloorDbFS := toDbFS(noiseRMS)

	// Measure 50Hz and 60Hz hum using Goertzel algorithm over quiet windows
	hum50Energy := 0.0
	hum60Energy := 0.0
	for i := 0; i < kCount; i++ {
		w := windows[i]
		slice := mono[w.startIndex:w.endIndex]
		hum50Energy += goertzelMagnitude(slice, sampleRate, 50.0)
		hum60Energy += goertzelMagnitude(slice, sampleRate, 60.0)
	}
	hum50Energy /= float64(kCount)
	hum60Energy /= float64(kCount)

	hum50DbFS := toDbFS(hum50Energy)
	hum60DbFS := toDbFS(hum60Energy)

	// A distinct hum tone is detected when the single 50Hz/60Hz frequency bin
	// constitutes >= 35% of total noise floor RMS energy and is audible (>= -85 dBFS),
	// or when its level exceeds broadband floor by the configured threshold.
	has50HzHum := (hum50Energy >= 0.35*noiseRMS || (hum50DbFS-noiseFloorDbFS) >= na.cfg.HumThresholdDiffDb) && hum50DbFS >= -85.0
	has60HzHum := (hum60Energy >= 0.35*noiseRMS || (hum60DbFS-noiseFloorDbFS) >= na.cfg.HumThresholdDiffDb) && hum60DbFS >= -85.0

	passesGate := noiseFloorDbFS <= na.cfg.QualityLimitDbFS

	return &NoiseFloorResult{
		NoiseFloorDbFS:     math.Round(noiseFloorDbFS*10) / 10,
		PassesQualityGate:  passesGate,
		Hum50HzDbFS:        math.Round(hum50DbFS*10) / 10,
		Hum60HzDbFS:        math.Round(hum60DbFS*10) / 10,
		Has50HzHum:         has50HzHum,
		Has60HzHum:         has60HzHum,
		QuietDurationMs:    math.Round(float64(quietTotalSamples)/float64(sampleRate)*1000.0*10) / 10,
		AnalyzedDurationMs: math.Round(durationMs*10) / 10,
	}, nil
}

// goertzelMagnitude computes the spectral magnitude of a target frequency component.
func goertzelMagnitude(samples []float64, sampleRate int, targetFreq float64) float64 {
	n := len(samples)
	if n == 0 {
		return 0.0
	}
	k := int(0.5 + (float64(n)*targetFreq)/float64(sampleRate))
	omega := (2.0 * math.Pi * float64(k)) / float64(n)
	cosine := math.Cos(omega)
	coeff := 2.0 * cosine

	q0, q1, q2 := 0.0, 0.0, 0.0
	for i := 0; i < n; i++ {
		q0 = coeff*q1 - q2 + samples[i]
		q2 = q1
		q1 = q0
	}

	power := q1*q1 + q2*q2 - q1*q2*coeff
	if power < 0 {
		power = 0
	}
	return (2.0 * math.Sqrt(power)) / float64(n)
}

// ToJSON serializes noise analysis result.
func (r *NoiseFloorResult) ToJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
