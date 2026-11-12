// Package forensics provides audio splicing detection, room noise floor discontinuity tracking,
// phase trajectory disruption analysis, and tamper timestamp logging.
package forensics

import (
	"encoding/json"
	"math"

	"avibe-platform/backend/internal/pkg/audio"
)

// SplicePoint represents a detected editing or splicing boundary.
type SplicePoint struct {
	TimestampMs      float64 `json:"timestamp_ms"`
	SampleIndex      int     `json:"sample_index"`
	NoiseFloorDiffDb float64 `json:"noise_floor_diff_db"`
	PhaseDisruption  float64 `json:"phase_disruption"`
	ConfidencePct    float64 `json:"confidence_pct"` // 0.0 to 100.0%
}

// SplicingAnalysisResult encapsulates detected splicing boundaries and authenticity score.
type SplicingAnalysisResult struct {
	IsTampered          bool          `json:"is_tampered"`
	TamperConfidencePct float64       `json:"tamper_confidence_pct"`
	SpliceCount         int           `json:"splice_count"`
	SplicePoints        []SplicePoint `json:"splice_points"`
	DiagnosticSummary   string        `json:"diagnostic_summary"`
}

// SplicingDetector detects non-continuous edits, cut-and-paste splices, and room acoustic jumps.
type SplicingDetector struct {
	windowMs float64
}

// NewSplicingDetector creates a splicing detector.
func NewSplicingDetector() *SplicingDetector {
	return &SplicingDetector{
		windowMs: 50.0, // 50ms inspection frames
	}
}

// Analyze scans the buffer for abrupt background acoustic steps and splice points.
func (sd *SplicingDetector) Analyze(input *audio.PCMBuffer) (*SplicingAnalysisResult, error) {
	if input == nil {
		return nil, audio.ErrNilPCMBuffer
	}
	if input.NumSamples == 0 || input.Channels == 0 || len(input.Samples) == 0 {
		return nil, audio.ErrEmptyPCMBuffer
	}

	sampleRate := input.SampleRate
	numSamples := input.NumSamples
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

	frameSize := int((sd.windowMs / 1000.0) * float64(sampleRate))
	if frameSize <= 0 {
		frameSize = 512
	}
	numFrames := numSamples / frameSize
	if numFrames < 6 {
		return &SplicingAnalysisResult{
			IsTampered:        false,
			DiagnosticSummary: "Audio duration too short for splicing analysis",
		}, nil
	}

	// 4th-order Butterworth Highpass filter at 6kHz to isolate high-frequency room tone & hiss
	highpass := cascadeBiquadHPF(mono, sampleRate, 6000.0)
	frameNoiseDb := make([]float64, numFrames)
	for f := 0; f < numFrames; f++ {
		start := f * frameSize
		end := start + frameSize
		var sumSq float64
		for i := start; i < end; i++ {
			sumSq += highpass[i] * highpass[i]
		}
		rms := math.Sqrt(sumSq / float64(frameSize))
		if rms < 1e-6 {
			frameNoiseDb[f] = -120.0
		} else {
			frameNoiseDb[f] = 20.0 * math.Log10(rms)
		}
	}

	var splicePoints []SplicePoint

	// Detect step discontinuities between frame f and frame f+1
	// Skip the first 2 frames (100ms) to allow IIR highpass filter state to settle
	for f := 2; f < numFrames-1; f++ {
		// Ignore dead silence frames
		if frameNoiseDb[f] < -85.0 && frameNoiseDb[f-1] < -85.0 {
			continue
		}

		stepDiff := math.Abs(frameNoiseDb[f] - frameNoiseDb[f-1])

		// Check zero-crossing phase continuity at the exact boundary sample
		boundarySample := f * frameSize
		phaseDisruption := 0.0
		if boundarySample+1 < numSamples && boundarySample-1 >= 0 {
			// First derivative jump
			d1 := mono[boundarySample] - mono[boundarySample-1]
			d2 := mono[boundarySample+1] - mono[boundarySample]
			phaseDisruption = math.Abs(d2 - d1)
		}

		// Splicing indicator: >8dB sudden step in room hiss AND phase disruption
		if stepDiff >= 8.0 || (stepDiff >= 6.0 && phaseDisruption > 0.4) {
			conf := (stepDiff / 15.0) * 100.0
			if conf > 100.0 {
				conf = 100.0
			}

			tsMs := (float64(boundarySample) / float64(sampleRate)) * 1000.0
			splicePoints = append(splicePoints, SplicePoint{
				TimestampMs:      math.Round(tsMs*10) / 10,
				SampleIndex:      boundarySample,
				NoiseFloorDiffDb: math.Round(stepDiff*10) / 10,
				PhaseDisruption:  math.Round(phaseDisruption*100) / 100,
				ConfidencePct:    math.Round(conf*10) / 10,
			})
		}
	}

	isTampered := len(splicePoints) > 0
	topConf := 0.0
	for _, sp := range splicePoints {
		if sp.ConfidencePct > topConf {
			topConf = sp.ConfidencePct
		}
	}

	summary := "Continuous audio recording verified; no splicing boundaries detected."
	if isTampered {
		summary = "Audio splicing detected: abrupt acoustic room noise and phase discontinuities found at boundary timestamps."
	}

	return &SplicingAnalysisResult{
		IsTampered:          isTampered,
		TamperConfidencePct: topConf,
		SpliceCount:         len(splicePoints),
		SplicePoints:        splicePoints,
		DiagnosticSummary:   summary,
	}, nil
}

// ToJSON serializes splicing analysis.
func (r *SplicingAnalysisResult) ToJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// cascadeBiquadHPF runs a 4th-order (24 dB/oct) highpass Butterworth filter.
func cascadeBiquadHPF(in []float64, sampleRate int, cutoff float64) []float64 {
	stage1 := biquadHPF(in, sampleRate, cutoff, 0.70710678)
	return biquadHPF(stage1, sampleRate, cutoff, 0.70710678)
}

func biquadHPF(in []float64, sampleRate int, cutoff float64, q float64) []float64 {
	n := len(in)
	out := make([]float64, n)
	if n == 0 || sampleRate <= 0 || cutoff <= 0 {
		return out
	}

	w0 := 2.0 * math.Pi * (cutoff / float64(sampleRate))
	cosw0 := math.Cos(w0)
	sinw0 := math.Sin(w0)
	alpha := sinw0 / (2.0 * q)

	b0 := (1.0 + cosw0) / 2.0
	b1 := -(1.0 + cosw0)
	b2 := (1.0 + cosw0) / 2.0
	a0 := 1.0 + alpha
	a1 := -2.0 * cosw0
	a2 := 1.0 - alpha

	b0 /= a0
	b1 /= a0
	b2 /= a0
	a1 /= a0
	a2 /= a0

	var x1, x2, y1, y2 float64
	for i := 0; i < n; i++ {
		x0 := in[i]
		y0 := b0*x0 + b1*x1 + b2*x2 - a1*y1 - a2*y2
		out[i] = y0
		x2 = x1
		x1 = x0
		y2 = y1
		y1 = y0
	}
	return out
}
