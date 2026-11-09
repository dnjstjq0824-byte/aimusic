// Package audio provides automated dynamic gain pumping and limiter breathing
// artifact detection, cyclic envelope modulation analysis, and quality gate rejection.
package audio

import (
	"encoding/json"
	"math"
)

// GainPumpingConfig configures limiter breathing and pumping detection.
type GainPumpingConfig struct {
	WindowMs            float64 `json:"window_ms"`             // Short RMS window, default: 50.0 ms
	PumpingThresholdDb  float64 `json:"pumping_threshold_db"`  // Envelope swing threshold in dB, default: 6.0 dB
	MaxAllowedIncidents int     `json:"max_allowed_incidents"` // Maximum allowed pumping swings before rejection, default: 5
}

// DefaultGainPumpingConfig provides standard quality control thresholds for gain pumping.
func DefaultGainPumpingConfig() GainPumpingConfig {
	return GainPumpingConfig{
		WindowMs:            50.0,
		PumpingThresholdDb:  6.0,
		MaxAllowedIncidents: 5,
	}
}

// GainPumpingResult contains detected pumping incidents, severity score, and quality gate decision.
type GainPumpingResult struct {
	PumpingSeverityScore float64 `json:"pumping_severity_score"` // 0.0 (smooth) to 100.0 (extreme breathing)
	PumpingEventsCount   int     `json:"pumping_events_count"`   // Count of rapid >6dB envelope swings
	MaxEnvelopeSwingDb   float64 `json:"max_envelope_swing_db"`  // Peak observed dB swing
	HasExcessivePumping  bool    `json:"has_excessive_pumping"`  // True if exceeds quality threshold
	RecommendedAction    string  `json:"recommended_action"`     // "PASS" or "REJECT: Excessive gain pumping artifact detected"
	FramesAnalyzed       int     `json:"frames_analyzed"`
}

// GainPumpingDetector scans audio buffers for unnatural volume modulation and pumping artifacts.
type GainPumpingDetector struct {
	cfg GainPumpingConfig
}

// NewGainPumpingDetector creates a new pumping detector.
func NewGainPumpingDetector(cfg GainPumpingConfig) *GainPumpingDetector {
	if cfg.WindowMs <= 5.0 {
		cfg.WindowMs = 50.0
	}
	if cfg.PumpingThresholdDb <= 1.0 {
		cfg.PumpingThresholdDb = 6.0
	}
	if cfg.MaxAllowedIncidents <= 0 {
		cfg.MaxAllowedIncidents = 5
	}
	return &GainPumpingDetector{cfg: cfg}
}

// Analyze evaluates envelope fluctuations and flags severe limiter breathing artifacts.
func (gpd *GainPumpingDetector) Analyze(input *PCMBuffer) (*GainPumpingResult, error) {
	if input == nil {
		return nil, ErrNilPCMBuffer
	}
	if input.NumSamples == 0 || input.Channels == 0 || len(input.Samples) == 0 {
		return nil, ErrEmptyPCMBuffer
	}

	numSamples := input.NumSamples
	sampleRate := input.SampleRate
	channels := input.Channels

	// Downmix to mono
	mono := make([]float64, numSamples)
	scale := 1.0 / float64(channels)
	for i := 0; i < numSamples; i++ {
		var sum float64
		for ch := 0; ch < channels; ch++ {
			sum += input.Samples[ch][i]
		}
		mono[i] = sum * scale
	}

	samplesPerWindow := int((gpd.cfg.WindowMs / 1000.0) * float64(sampleRate))
	if samplesPerWindow <= 0 {
		samplesPerWindow = 512
	}

	numFrames := numSamples / samplesPerWindow
	if numFrames < 4 {
		return &GainPumpingResult{
			PumpingSeverityScore: 0.0,
			PumpingEventsCount:   0,
			MaxEnvelopeSwingDb:   0.0,
			HasExcessivePumping:  false,
			RecommendedAction:    "PASS",
			FramesAnalyzed:       numFrames,
		}, nil
	}

	// Calculate RMS dBFS per frame
	frameDb := make([]float64, numFrames)
	for f := 0; f < numFrames; f++ {
		start := f * samplesPerWindow
		end := start + samplesPerWindow
		var sumSq float64
		for i := start; i < end; i++ {
			s := mono[i]
			sumSq += s * s
		}
		rms := math.Sqrt(sumSq / float64(samplesPerWindow))
		frameDb[f] = toDbFS(rms)
	}

	// Detect rapid envelope swings
	var pumpingEvents int
	var maxSwing float64
	var totalSwing float64

	thresholdDb := gpd.cfg.PumpingThresholdDb

	for f := 1; f < numFrames; f++ {
		// Ignore dead silence frames (< -70 dBFS)
		if frameDb[f] < -70.0 && frameDb[f-1] < -70.0 {
			continue
		}

		diff := math.Abs(frameDb[f] - frameDb[f-1])
		if diff > maxSwing {
			maxSwing = diff
		}

		if diff >= thresholdDb {
			pumpingEvents++
			totalSwing += diff
		}
	}

	// Severity calculation: combination of event density and swing magnitude
	eventDensity := float64(pumpingEvents) / float64(numFrames)
	severity := (eventDensity * 200.0) + (maxSwing * 2.0)
	if severity > 100.0 {
		severity = 100.0
	}

	hasExcessive := pumpingEvents >= gpd.cfg.MaxAllowedIncidents || severity >= 50.0
	action := "PASS"
	if hasExcessive {
		action = "REJECT: Excessive gain pumping artifact detected"
	}

	return &GainPumpingResult{
		PumpingSeverityScore: math.Round(severity*10) / 10,
		PumpingEventsCount:   pumpingEvents,
		MaxEnvelopeSwingDb:   math.Round(maxSwing*10) / 10,
		HasExcessivePumping:  hasExcessive,
		RecommendedAction:    action,
		FramesAnalyzed:       numFrames,
	}, nil
}

// ToJSON serializes pumping analysis telemetry.
func (r *GainPumpingResult) ToJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
