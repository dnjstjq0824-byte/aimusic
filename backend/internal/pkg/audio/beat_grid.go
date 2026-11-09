// Package audio provides BPM-synchronized beat grid generation, 1/4, 1/8, 1/16 note quantizing,
// zero-crossing loop slicing, and DJ/remixer loop marker extraction.
package audio

import (
	"encoding/json"
	"errors"
	"math"
)

var (
	ErrInvalidBPM = errors.New("BPM must be greater than 20 and less than 300")
)

// BeatDivision defines standard musical quantization subdivisions.
type BeatDivision string

const (
	DivisionQuarter   BeatDivision = "1/4"  // Quarter note (1 beat)
	DivisionEighth    BeatDivision = "1/8"  // Eighth note (0.5 beat)
	DivisionSixteenth BeatDivision = "1/16" // Sixteenth note (0.25 beat)
)

// BeatMarker represents a precise timestamped beat or subdivision boundary.
type BeatMarker struct {
	Index       int          `json:"index"`
	Bar         int          `json:"bar"`         // 1-indexed bar number (e.g. Bar 1, 2, ...)
	Beat        int          `json:"beat"`        // 1-indexed beat within bar (1..4 in 4/4)
	Subdivision int          `json:"subdivision"` // 1-indexed sixteenth or eighth step
	Division    BeatDivision `json:"division"`
	SampleIndex int          `json:"sample_index"`
	TimestampMs float64      `json:"timestamp_ms"`
}

// AudioSlice represents an exported loop segment sliced on zero-crossing boundaries.
type AudioSlice struct {
	SliceID       string      `json:"slice_id"`
	StartSample   int         `json:"start_sample"`
	EndSample     int         `json:"end_sample"`
	StartTimeMs   float64     `json:"start_time_ms"`
	EndTimeMs     float64     `json:"end_time_ms"`
	DurationMs    float64     `json:"duration_ms"`
	StartMarker   BeatMarker  `json:"start_marker"`
	EndMarker     BeatMarker  `json:"end_marker"`
	Buffer        *PCMBuffer  `json:"-"`
}

// BeatGridConfig configures beat grid and quantization parameters.
type BeatGridConfig struct {
	BPM             float64      `json:"bpm"`
	TimeSigBeats    int          `json:"time_sig_beats"` // Beats per measure (default: 4 for 4/4)
	Division        BeatDivision `json:"division"`       // Subdivision resolution (default: "1/16")
	AlignZeroCross  bool         `json:"align_zero_cross"`
}

// DefaultBeatGridConfig returns standard 4/4 1/16th beat grid parameters.
func DefaultBeatGridConfig(bpm float64) BeatGridConfig {
	return BeatGridConfig{
		BPM:            bpm,
		TimeSigBeats:   4,
		Division:       DivisionSixteenth,
		AlignZeroCross: true,
	}
}

// BeatGridResult encapsulates generated markers and loop slices.
type BeatGridResult struct {
	BPM             float64      `json:"bpm"`
	TotalBars       int          `json:"total_bars"`
	TotalBeats      int          `json:"total_beats"`
	Division        BeatDivision `json:"division"`
	Markers         []BeatMarker `json:"markers"`
	Slices          []AudioSlice `json:"slices"`
}

// BeatGridQuantizer generates musical beat grids and slices audio loops.
type BeatGridQuantizer struct {
	cfg BeatGridConfig
}

// NewBeatGridQuantizer instantiates a beat grid quantizer.
func NewBeatGridQuantizer(cfg BeatGridConfig) (*BeatGridQuantizer, error) {
	if cfg.BPM < 20.0 || cfg.BPM > 300.0 {
		return nil, ErrInvalidBPM
	}
	if cfg.TimeSigBeats <= 0 {
		cfg.TimeSigBeats = 4
	}
	if cfg.Division == "" {
		cfg.Division = DivisionSixteenth
	}
	return &BeatGridQuantizer{cfg: cfg}, nil
}

// Quantize generates beat markers and 1-bar loop slices across the audio buffer.
func (bg *BeatGridQuantizer) Quantize(input *PCMBuffer) (*BeatGridResult, error) {
	if input == nil {
		return nil, ErrNilPCMBuffer
	}
	if input.NumSamples == 0 || input.Channels == 0 || len(input.Samples) == 0 {
		return nil, ErrEmptyPCMBuffer
	}

	numSamples := input.NumSamples
	sampleRate := input.SampleRate
	bpm := bg.cfg.BPM

	// Seconds per beat
	spb := 60.0 / bpm
	samplesPerBeat := spb * float64(sampleRate)

	// Subdivision factor
	var stepsPerBeat int
	switch bg.cfg.Division {
	case DivisionQuarter:
		stepsPerBeat = 1
	case DivisionEighth:
		stepsPerBeat = 2
	case DivisionSixteenth:
		stepsPerBeat = 4
	default:
		stepsPerBeat = 4
	}

	samplesPerStep := samplesPerBeat / float64(stepsPerBeat)
	totalSteps := int(float64(numSamples) / samplesPerStep)

	markers := make([]BeatMarker, 0, totalSteps)
	timeSig := bg.cfg.TimeSigBeats

	for i := 0; i <= totalSteps; i++ {
		idealSample := int(math.Round(float64(i) * samplesPerStep))
		if idealSample > numSamples {
			idealSample = numSamples
		}

		actualSample := idealSample
		if bg.cfg.AlignZeroCross {
			actualSample = findNearestZeroCrossing(input.Samples[0], idealSample, 100)
		}

		tsMs := (float64(actualSample) / float64(sampleRate)) * 1000.0

		// Calculate Bar, Beat, Subdivision
		beatIndex := i / stepsPerBeat
		barIndex := (beatIndex / timeSig) + 1
		beatInBar := (beatIndex % timeSig) + 1
		subInBeat := (i % stepsPerBeat) + 1

		markers = append(markers, BeatMarker{
			Index:       i,
			Bar:         barIndex,
			Beat:        beatInBar,
			Subdivision: subInBeat,
			Division:    bg.cfg.Division,
			SampleIndex: actualSample,
			TimestampMs: math.Round(tsMs*10) / 10,
		})
	}

	// Generate 1-bar loop slices (each bar has timeSig beats)
	stepsPerBar := stepsPerBeat * timeSig
	numBars := totalSteps / stepsPerBar
	slices := make([]AudioSlice, 0, numBars)

	for b := 0; b < numBars; b++ {
		startIdx := b * stepsPerBar
		endIdx := (b + 1) * stepsPerBar

		if endIdx >= len(markers) {
			break
		}

		startM := markers[startIdx]
		endM := markers[endIdx]

		sliceLen := endM.SampleIndex - startM.SampleIndex
		if sliceLen <= 0 {
			continue
		}

		sliceBuf := &PCMBuffer{
			Channels:   input.Channels,
			SampleRate: sampleRate,
			NumSamples: sliceLen,
			Samples:    make([][]float64, input.Channels),
		}

		for ch := 0; ch < input.Channels; ch++ {
			chSlice := make([]float64, sliceLen)
			copy(chSlice, input.Samples[ch][startM.SampleIndex:endM.SampleIndex])
			sliceBuf.Samples[ch] = chSlice
		}

		durationMs := (float64(sliceLen) / float64(sampleRate)) * 1000.0

		slices = append(slices, AudioSlice{
			SliceID:     sliceBufDescription(b + 1),
			StartSample: startM.SampleIndex,
			EndSample:   endM.SampleIndex,
			StartTimeMs: startM.TimestampMs,
			EndTimeMs:   endM.TimestampMs,
			DurationMs:  math.Round(durationMs*10) / 10,
			StartMarker: startM,
			EndMarker:   endM,
			Buffer:      sliceBuf,
		})
	}

	totalBeats := totalSteps / stepsPerBeat

	return &BeatGridResult{
		BPM:        bpm,
		TotalBars:  numBars,
		TotalBeats: totalBeats,
		Division:   bg.cfg.Division,
		Markers:    markers,
		Slices:     slices,
	}, nil
}

func sliceBufDescription(barNum int) string {
	b, _ := json.Marshal(barNum)
	return "bar-" + string(b)
}

// findNearestZeroCrossing searches within a window for the point closest to zero voltage.
func findNearestZeroCrossing(samples []float64, targetIdx, maxWindow int) int {
	n := len(samples)
	if targetIdx >= n {
		return n - 1
	}
	if targetIdx < 0 {
		return 0
	}

	bestIdx := targetIdx
	minAbs := math.Abs(samples[targetIdx])

	start := targetIdx - maxWindow
	if start < 0 {
		start = 0
	}
	end := targetIdx + maxWindow
	if end >= n {
		end = n - 1
	}

	for i := start; i < end; i++ {
		// Detect zero crossing
		if (samples[i] <= 0 && samples[i+1] >= 0) || (samples[i] >= 0 && samples[i+1] <= 0) {
			if math.Abs(samples[i]) < minAbs {
				minAbs = math.Abs(samples[i])
				bestIdx = i
			}
		}
	}

	return bestIdx
}

// ToJSON serializes beat grid markers and slice metadata.
func (r *BeatGridResult) ToJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
