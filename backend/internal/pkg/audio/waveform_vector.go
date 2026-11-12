// Package audio provides high-performance audio waveform vector extraction,
// peak-to-peak decimation, and ultra-compact serialization for frontend audio visualization.
package audio

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
)

// PeakVectorPoint represents decimation metrics for a single time slice.
type PeakVectorPoint struct {
	Min float64 `json:"min"` // Minimum negative peak in [-1.0, 0.0]
	Max float64 `json:"max"` // Maximum positive peak in [0.0, 1.0]
	RMS float64 `json:"rms"` // Root Mean Square energy in [0.0, 1.0]
}

// PeakVectorConfig configures waveform extraction parameters.
type PeakVectorConfig struct {
	Points    int  `json:"points"`    // Target number of points, default: 1000
	MonoMix   bool `json:"mono_mix"`  // Downmix multi-channel to mono before decimation, default: true
	Precision int  `json:"precision"` // Decimal precision for JSON rounding (1 to 4), default: 2
}

// DefaultPeakVectorConfig returns standard 1000-point waveform extraction config.
func DefaultPeakVectorConfig() PeakVectorConfig {
	return PeakVectorConfig{
		Points:    1000,
		MonoMix:   true,
		Precision: 2,
	}
}

// WaveformVectorResult holds extracted waveform vectors and compact representation.
type WaveformVectorResult struct {
	DurationSec float64           `json:"duration_sec"`
	SampleRate  int               `json:"sample_rate"`
	NumPoints   int               `json:"num_points"`
	GlobalPeak  float64           `json:"global_peak"`
	GlobalRMS   float64           `json:"global_rms"`
	Peaks       []float64         `json:"peaks"`            // 1000 normalized absolute peaks [0.00..1.00]
	Points      []PeakVectorPoint `json:"points,omitempty"` // Detailed min/max/rms points
	PackedB64   string            `json:"packed_b64"`       // Ultra-compact binary base64 string (< 2KB)
}

// PeakVectorExtractor decimates raw audio PCM into waveform vectors.
type PeakVectorExtractor struct {
	cfg PeakVectorConfig
}

// NewPeakVectorExtractor creates a new waveform vector extractor.
func NewPeakVectorExtractor(cfg PeakVectorConfig) *PeakVectorExtractor {
	if cfg.Points <= 0 {
		cfg.Points = 1000
	}
	if cfg.Precision <= 0 || cfg.Precision > 4 {
		cfg.Precision = 2
	}
	return &PeakVectorExtractor{cfg: cfg}
}

// Extract decimates audio buffer into 1000-point peak-to-peak waveform vector.
func (pve *PeakVectorExtractor) Extract(input *PCMBuffer) (*WaveformVectorResult, error) {
	if input == nil {
		return nil, ErrNilPCMBuffer
	}
	if len(input.Samples) == 0 || input.NumSamples == 0 {
		return nil, ErrEmptyPCMBuffer
	}

	numSamples := input.NumSamples
	sampleRate := input.SampleRate
	duration := float64(numSamples) / float64(sampleRate)

	// Step 1: Downmix to mono channel if required
	mono := make([]float64, numSamples)
	numChannels := input.Channels

	if numChannels == 1 || !pve.cfg.MonoMix {
		copy(mono, input.Samples[0])
	} else {
		scale := 1.0 / float64(numChannels)
		for i := 0; i < numSamples; i++ {
			var sum float64
			for ch := 0; ch < numChannels; ch++ {
				sum += input.Samples[ch][i]
			}
			mono[i] = sum * scale
		}
	}

	// Step 2: Global peak & RMS calculation
	var globalPeak float64
	var totalSquareSum float64
	for i := 0; i < numSamples; i++ {
		s := mono[i]
		abs := math.Abs(s)
		if abs > globalPeak {
			globalPeak = abs
		}
		totalSquareSum += s * s
	}
	globalRMS := math.Sqrt(totalSquareSum / float64(numSamples))

	// Step 3: Decimate into N points
	targetPoints := pve.cfg.Points
	if numSamples < targetPoints {
		targetPoints = numSamples
	}

	points := make([]PeakVectorPoint, targetPoints)
	peaks := make([]float64, targetPoints)
	precMult := math.Pow(10, float64(pve.cfg.Precision))

	// Binary packing buffer: 2 bytes per point (1 byte peak + 1 byte RMS)
	b64Buf := new(bytes.Buffer)
	stepSize := float64(numSamples) / float64(targetPoints)

	for p := 0; p < targetPoints; p++ {
		startIdx := int(float64(p) * stepSize)
		endIdx := int(float64(p+1) * stepSize)
		if endIdx > numSamples {
			endIdx = numSamples
		}
		if startIdx >= endIdx {
			startIdx = endIdx - 1
		}

		var minVal, maxVal, sqSum float64
		for i := startIdx; i < endIdx; i++ {
			s := mono[i]
			if s < minVal {
				minVal = s
			}
			if s > maxVal {
				maxVal = s
			}
			sqSum += s * s
		}

		sliceCount := endIdx - startIdx
		rmsVal := 0.0
		if sliceCount > 0 {
			rmsVal = math.Sqrt(sqSum / float64(sliceCount))
		}

		// Absolute peak for UI representation
		peakVal := math.Max(math.Abs(minVal), math.Abs(maxVal))
		if peakVal > 1.0 {
			peakVal = 1.0
		}

		// Round to configured precision
		roundedMin := math.Round(minVal*precMult) / precMult
		roundedMax := math.Round(maxVal*precMult) / precMult
		roundedRMS := math.Round(rmsVal*precMult) / precMult
		roundedPeak := math.Round(peakVal*precMult) / precMult

		points[p] = PeakVectorPoint{
			Min: roundedMin,
			Max: roundedMax,
			RMS: roundedRMS,
		}
		peaks[p] = roundedPeak

		// Quantize into 8-bit unsigned (0..255) for binary packing
		peakByte := uint8(math.Min(255.0, math.Round(peakVal*255.0)))
		rmsByte := uint8(math.Min(255.0, math.Round(rmsVal*255.0)))
		_ = binary.Write(b64Buf, binary.BigEndian, peakByte)
		_ = binary.Write(b64Buf, binary.BigEndian, rmsByte)
	}

	packedB64 := base64.StdEncoding.EncodeToString(b64Buf.Bytes())

	return &WaveformVectorResult{
		DurationSec: math.Round(duration*100) / 100,
		SampleRate:  sampleRate,
		NumPoints:   targetPoints,
		GlobalPeak:  math.Round(globalPeak*1000) / 1000,
		GlobalRMS:   math.Round(globalRMS*1000) / 1000,
		Peaks:       peaks,
		Points:      points,
		PackedB64:   packedB64,
	}, nil
}

// CompactPeakJSON is the high-efficiency payload formatted strictly under 10KB.
type CompactPeakJSON struct {
	DurationSec float64   `json:"duration_sec"`
	SampleRate  int       `json:"sample_rate"`
	NumPoints   int       `json:"num_points"`
	GlobalPeak  float64   `json:"global_peak"`
	GlobalRMS   float64   `json:"global_rms"`
	Peaks       []float64 `json:"peaks"`
	PackedB64   string    `json:"packed_b64"`
}

// ToJSON serializes the waveform vector to compact JSON guaranteed < 10KB for 1,000 points.
func (wv *WaveformVectorResult) ToJSON() ([]byte, error) {
	compact := CompactPeakJSON{
		DurationSec: wv.DurationSec,
		SampleRate:  wv.SampleRate,
		NumPoints:   wv.NumPoints,
		GlobalPeak:  wv.GlobalPeak,
		GlobalRMS:   wv.GlobalRMS,
		Peaks:       wv.Peaks,
		PackedB64:   wv.PackedB64,
	}

	data, err := json.Marshal(compact)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal waveform vector: %w", err)
	}

	// Verification check: ensure size is strictly under 10KB (10,240 bytes)
	if len(data) > 10240 {
		return nil, fmt.Errorf("waveform json size %d bytes exceeds 10KB limit", len(data))
	}

	return data, nil
}

// DecodePackedB64 decodes the 8-bit packed base64 string back into normalized peak & RMS arrays.
func DecodePackedB64(b64 string) ([]float64, []float64, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid base64 waveform: %w", err)
	}
	if len(raw)%2 != 0 {
		return nil, nil, fmt.Errorf("corrupt packed waveform payload length %d", len(raw))
	}

	n := len(raw) / 2
	peaks := make([]float64, n)
	rmss := make([]float64, n)

	for i := 0; i < n; i++ {
		peaks[i] = math.Round((float64(raw[i*2])/255.0)*1000) / 1000
		rmss[i] = math.Round((float64(raw[i*2+1])/255.0)*1000) / 1000
	}

	return peaks, rmss, nil
}
