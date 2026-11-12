// Package audio provides stereo phase correlation analysis, sub-bass phase alignment verification,
// frequency-split correlation metrics, and mono fold-down cancellation risk detection.
package audio

import (
	"encoding/json"
	"errors"
	"math"
)

var (
	ErrNotStereo = errors.New("phase correlation analysis requires exactly 2 channels (stereo)")
)

// PhaseCorrelatorConfig defines thresholds for phase correlation and mono compatibility.
type PhaseCorrelatorConfig struct {
	MinAcceptableCorrelation float64 `json:"min_acceptable_correlation"` // Recommended >= 0.20
	SubBassThresholdHz       float64 `json:"sub_bass_threshold_hz"`      // Default 200Hz
	HighThresholdHz          float64 `json:"high_threshold_hz"`          // Default 4000Hz
	MaxMonoEnergyLossDb      float64 `json:"max_mono_energy_loss_db"`    // Default 3.0 dB
}

// DefaultPhaseCorrelatorConfig returns standard broadcast and vinyl mastering phase criteria.
func DefaultPhaseCorrelatorConfig() PhaseCorrelatorConfig {
	return PhaseCorrelatorConfig{
		MinAcceptableCorrelation: 0.20,
		SubBassThresholdHz:       200.0,
		HighThresholdHz:          4000.0,
		MaxMonoEnergyLossDb:      3.0,
	}
}

// PhaseCorrelationResult encapsulates global, band-split correlation, and mono fold-down risk.
type PhaseCorrelationResult struct {
	GlobalCorrelation  float64 `json:"global_correlation"`    // -1.0 (anti-phase) to +1.0 (dual mono)
	BassCorrelation    float64 `json:"bass_correlation"`      // Sub-bass phase (<200Hz)
	MidCorrelation     float64 `json:"mid_correlation"`       // Vocal & instrument body (200Hz-4kHz)
	HighCorrelation    float64 `json:"high_correlation"`      // High air & cymbals (>4kHz)
	MonoEnergyLossDb   float64 `json:"mono_energy_loss_db"`   // dB drop when summed to mono (L+R)/2
	MonoCompatible     bool    `json:"mono_compatible"`       // True if passes mono fold-down without severe loss
	HasSevereAntiPhase bool    `json:"has_severe_anti_phase"` // True if correlation < 0 (high cancellation danger)
	WarningSummary     string  `json:"warning_summary,omitempty"`
}

// PhaseCorrelator measures stereo phase relationships and predicts mono compatibility.
type PhaseCorrelator struct {
	cfg PhaseCorrelatorConfig
}

// NewPhaseCorrelator instantiates a phase correlator.
func NewPhaseCorrelator(cfg PhaseCorrelatorConfig) *PhaseCorrelator {
	if cfg.MinAcceptableCorrelation <= -1.0 || cfg.MinAcceptableCorrelation > 1.0 {
		cfg.MinAcceptableCorrelation = 0.20
	}
	if cfg.SubBassThresholdHz <= 0 {
		cfg.SubBassThresholdHz = 200.0
	}
	if cfg.HighThresholdHz <= cfg.SubBassThresholdHz {
		cfg.HighThresholdHz = 4000.0
	}
	if cfg.MaxMonoEnergyLossDb <= 0 {
		cfg.MaxMonoEnergyLossDb = 3.0
	}
	return &PhaseCorrelator{cfg: cfg}
}

// Analyze evaluates phase correlation across time and frequency bands.
func (pc *PhaseCorrelator) Analyze(input *PCMBuffer) (*PhaseCorrelationResult, error) {
	if input == nil {
		return nil, ErrNilPCMBuffer
	}
	if input.NumSamples == 0 || len(input.Samples) == 0 {
		return nil, ErrEmptyPCMBuffer
	}
	if input.Channels != 2 {
		return nil, ErrNotStereo
	}

	left := input.Samples[0]
	right := input.Samples[1]
	n := input.NumSamples
	sampleRate := input.SampleRate

	// Global Pearson Correlation
	globalCorr := computeCorrelation(left, right)

	// Mono Fold-down Loss calculation: (L+R)/2 vs (RMS(L)+RMS(R))/2
	var sumSqL, sumSqR, sumSqMono float64
	for i := 0; i < n; i++ {
		l := left[i]
		r := right[i]
		m := 0.5 * (l + r)

		sumSqL += l * l
		sumSqR += r * r
		sumSqMono += m * m
	}

	fn := float64(n)
	rmsL := math.Sqrt(sumSqL / fn)
	rmsR := math.Sqrt(sumSqR / fn)
	rmsMono := math.Sqrt(sumSqMono / fn)
	avgStereoRMS := 0.5 * (rmsL + rmsR)

	monoLossDb := 0.0
	if avgStereoRMS > 1e-6 {
		if rmsMono < 1e-6 {
			monoLossDb = 120.0 // Near total cancellation
		} else {
			monoLossDb = 20.0 * math.Log10(avgStereoRMS/rmsMono)
			if monoLossDb < 0 {
				monoLossDb = 0
			}
		}
	}

	// Band-split filtering using single-pole IIR lowpass and highpass
	// Sub-bass filter: lowpass at SubBassThresholdHz
	lowL := lowpassIIR(left, sampleRate, pc.cfg.SubBassThresholdHz)
	lowR := lowpassIIR(right, sampleRate, pc.cfg.SubBassThresholdHz)
	bassCorr := computeCorrelation(lowL, lowR)

	// High air filter: highpass at HighThresholdHz
	highL := highpassIIR(left, sampleRate, pc.cfg.HighThresholdHz)
	highR := highpassIIR(right, sampleRate, pc.cfg.HighThresholdHz)
	highCorr := computeCorrelation(highL, highR)

	// Mid band: original - (low + high)
	midL := make([]float64, n)
	midR := make([]float64, n)
	for i := 0; i < n; i++ {
		midL[i] = left[i] - lowL[i] - highL[i]
		midR[i] = right[i] - lowR[i] - highR[i]
	}
	midCorr := computeCorrelation(midL, midR)

	hasSevereAntiPhase := globalCorr < 0.0 || bassCorr < 0.0
	monoCompatible := globalCorr >= pc.cfg.MinAcceptableCorrelation && monoLossDb <= pc.cfg.MaxMonoEnergyLossDb && bassCorr >= 0.5

	warningSummary := ""
	if hasSevereAntiPhase {
		warningSummary = "Critical Warning: Negative phase correlation detected. Significant cancellation will occur on mono speakers."
	} else if !monoCompatible {
		warningSummary = "Warning: Stereo mix has poor mono compatibility (sub-bass out of phase or mono fold-down energy drop > 3dB)."
	}

	return &PhaseCorrelationResult{
		GlobalCorrelation:  math.Round(globalCorr*1000) / 1000,
		BassCorrelation:    math.Round(bassCorr*1000) / 1000,
		MidCorrelation:     math.Round(midCorr*1000) / 1000,
		HighCorrelation:    math.Round(highCorr*1000) / 1000,
		MonoEnergyLossDb:   math.Round(monoLossDb*100) / 100,
		MonoCompatible:     monoCompatible,
		HasSevereAntiPhase: hasSevereAntiPhase,
		WarningSummary:     warningSummary,
	}, nil
}

// LowpassIIR implements a single-pole lowpass filter.
func LowpassIIR(in []float64, sampleRate int, cutoff float64) []float64 {
	out := make([]float64, len(in))
	dt := 1.0 / float64(sampleRate)
	rc := 1.0 / (2.0 * math.Pi * cutoff)
	alpha := dt / (rc + dt)

	prev := 0.0
	for i, x := range in {
		val := prev + alpha*(x-prev)
		out[i] = val
		prev = val
	}
	return out
}

func lowpassIIR(in []float64, sampleRate int, cutoff float64) []float64 {
	return LowpassIIR(in, sampleRate, cutoff)
}

// highpassIIR implements a single-pole highpass filter.
func highpassIIR(in []float64, sampleRate int, cutoff float64) []float64 {
	out := make([]float64, len(in))
	dt := 1.0 / float64(sampleRate)
	rc := 1.0 / (2.0 * math.Pi * cutoff)
	alpha := rc / (rc + dt)

	prevIn := 0.0
	prevOut := 0.0
	for i, x := range in {
		val := alpha * (prevOut + x - prevIn)
		out[i] = val
		prevIn = x
		prevOut = val
	}
	return out
}

// ToJSON serializes correlation telemetry.
func (r *PhaseCorrelationResult) ToJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
