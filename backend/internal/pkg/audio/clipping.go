// Package audio provides audio clipping event detection, inter-sample clip clustering,
// time-bucketed heatmap aggregation, and quality control threshold alerts (>5 incidents).
package audio

import (
	"encoding/json"
	"math"
)

// ClippingEvent represents a localized clipping incident.
type ClippingEvent struct {
	Channel      int     `json:"channel"`
	StartSample  int     `json:"start_sample"`
	EndSample    int     `json:"end_sample"`
	StartTimeMs  float64 `json:"start_time_ms"`
	EndTimeMs    float64 `json:"end_time_ms"`
	DurationMs   float64 `json:"duration_ms"`
	PeakValue    float64 `json:"peak_value"`
	ConsecutiveN int     `json:"consecutive_samples"`
}

// ClippingHeatmapBin represents a localized time slice with clipping frequency.
type ClippingHeatmapBin struct {
	BinIndex    int     `json:"bin_index"`
	StartTimeMs float64 `json:"start_time_ms"`
	EndTimeMs   float64 `json:"end_time_ms"`
	ClipCount   int     `json:"clip_count"`
	MaxPeak     float64 `json:"max_peak"`
}

// ClippingConfig configures clipping detection parameters.
type ClippingConfig struct {
	ThresholdLinear float64 `json:"threshold_linear"` // Default 0.999 (-0.01 dBFS)
	MinConsecutive  int     `json:"min_consecutive"`  // Minimum consecutive clipped samples to register event (default: 1)
	HeatmapBinMs    float64 `json:"heatmap_bin_ms"`   // Bin resolution in ms (default: 1000ms)
	AlertThreshold  int     `json:"alert_threshold"`  // Threshold for QualityWarning (default: 5)
}

// DefaultClippingConfig provides standard broadcast-grade clipping parameters.
func DefaultClippingConfig() ClippingConfig {
	return ClippingConfig{
		ThresholdLinear: 0.999, // -0.01 dBFS
		MinConsecutive:  1,
		HeatmapBinMs:    1000.0,
		AlertThreshold:  5,
	}
}

// ClippingAnalysisResult holds detected clipping incidents and heatmap data.
type ClippingAnalysisResult struct {
	TotalClips      int                  `json:"total_clips"`
	HasWarning      bool                 `json:"has_warning"`
	WarningMessage  string               `json:"warning_message,omitempty"`
	HighestPeak     float64              `json:"highest_peak"`
	HighestPeakDbFS float64              `json:"highest_peak_dbfs"`
	Events          []ClippingEvent      `json:"events"`
	Heatmap         []ClippingHeatmapBin `json:"heatmap"`
}

// ClippingDetector analyzes audio buffers for digital overs and clipping events.
type ClippingDetector struct {
	cfg ClippingConfig
}

// NewClippingDetector instantiates a clipping analyzer.
func NewClippingDetector(cfg ClippingConfig) *ClippingDetector {
	if cfg.ThresholdLinear <= 0.0 || cfg.ThresholdLinear > 1.0 {
		cfg.ThresholdLinear = 0.999
	}
	if cfg.MinConsecutive <= 0 {
		cfg.MinConsecutive = 1
	}
	if cfg.HeatmapBinMs <= 0 {
		cfg.HeatmapBinMs = 1000.0
	}
	if cfg.AlertThreshold <= 0 {
		cfg.AlertThreshold = 5
	}
	return &ClippingDetector{cfg: cfg}
}

// Analyze scans the buffer and extracts clipping events and heatmap bins.
func (cd *ClippingDetector) Analyze(input *PCMBuffer) (*ClippingAnalysisResult, error) {
	if input == nil {
		return nil, ErrNilPCMBuffer
	}
	if input.NumSamples == 0 || input.Channels == 0 || len(input.Samples) == 0 {
		return nil, ErrEmptyPCMBuffer
	}

	sampleRate := input.SampleRate
	numSamples := input.NumSamples
	channels := input.Channels
	durationMs := (float64(numSamples) / float64(sampleRate)) * 1000.0

	var events []ClippingEvent
	var highestPeak float64

	threshold := cd.cfg.ThresholdLinear
	minConsec := cd.cfg.MinConsecutive

	for ch := 0; ch < channels; ch++ {
		samples := input.Samples[ch]
		inClip := false
		startSample := 0
		peakInCluster := 0.0
		consecCount := 0

		for i := 0; i < numSamples; i++ {
			abs := math.Abs(samples[i])
			if abs > highestPeak {
				highestPeak = abs
			}

			if abs >= threshold {
				if !inClip {
					inClip = true
					startSample = i
					peakInCluster = abs
					consecCount = 1
				} else {
					consecCount++
					if abs > peakInCluster {
						peakInCluster = abs
					}
				}
			} else {
				if inClip {
					if consecCount >= minConsec {
						startMs := (float64(startSample) / float64(sampleRate)) * 1000.0
						endMs := (float64(i-1) / float64(sampleRate)) * 1000.0
						events = append(events, ClippingEvent{
							Channel:      ch,
							StartSample:  startSample,
							EndSample:    i - 1,
							StartTimeMs:  math.Round(startMs*10) / 10,
							EndTimeMs:    math.Round(endMs*10) / 10,
							DurationMs:   math.Round((endMs-startMs)*10) / 10,
							PeakValue:    math.Round(peakInCluster*10000) / 10000,
							ConsecutiveN: consecCount,
						})
					}
					inClip = false
					consecCount = 0
				}
			}
		}

		// Handle trailing clip at buffer end
		if inClip && consecCount >= minConsec {
			startMs := (float64(startSample) / float64(sampleRate)) * 1000.0
			endMs := (float64(numSamples-1) / float64(sampleRate)) * 1000.0
			events = append(events, ClippingEvent{
				Channel:      ch,
				StartSample:  startSample,
				EndSample:    numSamples - 1,
				StartTimeMs:  math.Round(startMs*10) / 10,
				EndTimeMs:    math.Round(endMs*10) / 10,
				DurationMs:   math.Round((endMs-startMs)*10) / 10,
				PeakValue:    math.Round(peakInCluster*10000) / 10000,
				ConsecutiveN: consecCount,
			})
		}
	}

	// Build Heatmap bins
	numBins := int(math.Ceil(durationMs / cd.cfg.HeatmapBinMs))
	if numBins == 0 {
		numBins = 1
	}
	heatmap := make([]ClippingHeatmapBin, numBins)
	for b := 0; b < numBins; b++ {
		startMs := float64(b) * cd.cfg.HeatmapBinMs
		endMs := math.Min(startMs+cd.cfg.HeatmapBinMs, durationMs)
		heatmap[b] = ClippingHeatmapBin{
			BinIndex:    b,
			StartTimeMs: math.Round(startMs*10) / 10,
			EndTimeMs:   math.Round(endMs*10) / 10,
			ClipCount:   0,
			MaxPeak:     0.0,
		}
	}

	for _, ev := range events {
		binIdx := int(ev.StartTimeMs / cd.cfg.HeatmapBinMs)
		if binIdx >= numBins {
			binIdx = numBins - 1
		}
		heatmap[binIdx].ClipCount++
		if ev.PeakValue > heatmap[binIdx].MaxPeak {
			heatmap[binIdx].MaxPeak = ev.PeakValue
		}
	}

	totalClips := len(events)
	hasWarning := totalClips >= cd.cfg.AlertThreshold
	warningMsg := ""
	if hasWarning {
		warningMsg = "Quality Alert: audio exceeds recommended clipping threshold (>5 incidents detected)"
	}

	return &ClippingAnalysisResult{
		TotalClips:      totalClips,
		HasWarning:      hasWarning,
		WarningMessage:  warningMsg,
		HighestPeak:     math.Round(highestPeak*10000) / 10000,
		HighestPeakDbFS: math.Round(toDbFS(highestPeak)*100) / 100,
		Events:          events,
		Heatmap:         heatmap,
	}, nil
}

// ToJSON serializes clipping analysis.
func (r *ClippingAnalysisResult) ToJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
