// Package audio provides lossless, high-performance audio waveform peak extraction,
// multi-resolution bucket calculation, dual min/max envelopes, RMS energy tracking,
// binary quantization, and SVG vector rendering without subprocess overhead.
package audio

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
)

// Waveform error definitions.
var (
	ErrInvalidBucketCount = errors.New("bucket count must be greater than zero")
	ErrCorruptBinaryData  = errors.New("corrupt waveform binary data")
)

// WaveformBucket holds bidirectional peak and energy metrics for a single time bucket.
type WaveformBucket struct {
	Min    float64 `json:"min"`     // Negative peak in range [-1.0, 0.0]
	Max    float64 `json:"max"`     // Positive peak in range [0.0, 1.0]
	AbsMax float64 `json:"abs_max"` // Normalized absolute peak in range [0.0, 1.0]
	RMS    float64 `json:"rms"`     // Root mean square energy in range [0.0, 1.0]
}

// WaveformData encapsulates comprehensive multi-bucket waveform analysis results.
type WaveformData struct {
	Buckets     []WaveformBucket `json:"buckets"`
	Peaks       []float64        `json:"peaks"` // Canonical 0.0-1.0 absolute peaks for visualizer
	NumBuckets  int              `json:"num_buckets"`
	DurationSec float64          `json:"duration_sec"`
	SampleRate  int              `json:"sample_rate"`
	Channels    int              `json:"channels"`
	MaxPeak     float64          `json:"max_peak"`
	AverageRMS  float64          `json:"average_rms"`
}

// WaveformExtractor configures waveform generation options.
type WaveformExtractor struct {
	Buckets      int
	DecibelScale bool
	MinDecibels  float64
}

// NewWaveformExtractor creates an extractor configured with target bucket count.
// Default buckets is 100 when non-positive value is passed.
func NewWaveformExtractor(buckets int) *WaveformExtractor {
	if buckets <= 0 {
		buckets = 100
	}
	return &WaveformExtractor{
		Buckets:      buckets,
		DecibelScale: false,
		MinDecibels:  -60.0,
	}
}

// ExtractWaveformFromPCM computes normalized waveform peaks and dual envelopes
// directly from unpacked PCM audio data in memory with zero subprocess overhead.
func ExtractWaveformFromPCM(buf *PCMBuffer, targetBuckets int) (*WaveformData, error) {
	if buf == nil {
		return nil, ErrNilPCMBuffer
	}
	if buf.NumSamples == 0 || buf.Channels == 0 || len(buf.Samples) == 0 {
		return nil, ErrEmptyPCMBuffer
	}
	if targetBuckets <= 0 {
		targetBuckets = 100
	}

	totalSamples := buf.NumSamples
	channels := buf.Channels
	duration := float64(totalSamples) / float64(buf.SampleRate)

	buckets := make([]WaveformBucket, targetBuckets)
	peaks := make([]float64, targetBuckets)

	// Step 1: Compute raw peaks and RMS per bucket
	rawMaxAll := 0.0
	sumRMSAll := 0.0

	samplesPerBucket := float64(totalSamples) / float64(targetBuckets)

	for b := 0; b < targetBuckets; b++ {
		startIdx := int(float64(b) * samplesPerBucket)
		endIdx := int(float64(b+1) * samplesPerBucket)

		if endIdx <= startIdx {
			endIdx = startIdx + 1
		}
		if endIdx > totalSamples {
			endIdx = totalSamples
		}
		if startIdx >= totalSamples {
			startIdx = totalSamples - 1
			endIdx = totalSamples
		}

		bucketMin := 0.0
		bucketMax := 0.0
		sumSquares := 0.0
		count := 0

		for s := startIdx; s < endIdx; s++ {
			for c := 0; c < channels; c++ {
				val := buf.Samples[c][s]
				if val > bucketMax {
					bucketMax = val
				}
				if val < bucketMin {
					bucketMin = val
				}
				sumSquares += val * val
				count++
			}
		}

		rmsVal := 0.0
		if count > 0 {
			rmsVal = math.Sqrt(sumSquares / float64(count))
		}

		absMax := math.Max(math.Abs(bucketMin), math.Abs(bucketMax))
		if absMax > rawMaxAll {
			rawMaxAll = absMax
		}

		buckets[b] = WaveformBucket{
			Min:    bucketMin,
			Max:    bucketMax,
			AbsMax: absMax,
			RMS:    rmsVal,
		}
		sumRMSAll += rmsVal
	}

	// Step 2: Normalize values to [0.0, 1.0] range
	avgRMS := 0.0
	if targetBuckets > 0 {
		avgRMS = sumRMSAll / float64(targetBuckets)
	}

	normFactor := 1.0
	if rawMaxAll > 0.0 {
		normFactor = 1.0 / rawMaxAll
	}

	for b := 0; b < targetBuckets; b++ {
		if rawMaxAll > 0.0 {
			buckets[b].Min = math.Max(-1.0, buckets[b].Min*normFactor)
			buckets[b].Max = math.Min(1.0, buckets[b].Max*normFactor)
			buckets[b].AbsMax = math.Min(1.0, buckets[b].AbsMax*normFactor)
			buckets[b].RMS = math.Min(1.0, buckets[b].RMS*normFactor)
		} else {
			buckets[b].Min = 0.0
			buckets[b].Max = 0.0
			buckets[b].AbsMax = 0.0
			buckets[b].RMS = 0.0
		}
		peaks[b] = buckets[b].AbsMax
	}

	return &WaveformData{
		Buckets:     buckets,
		Peaks:       peaks,
		NumBuckets:  targetBuckets,
		DurationSec: duration,
		SampleRate:  buf.SampleRate,
		Channels:    channels,
		MaxPeak:     math.Min(1.0, rawMaxAll),
		AverageRMS:  math.Min(1.0, avgRMS*normFactor),
	}, nil
}

// ExtractWaveformFromSamples calculates normalized peaks from a flat mono sample slice.
func ExtractWaveformFromSamples(samples []float64, targetBuckets int) ([]float64, error) {
	if len(samples) == 0 {
		return nil, ErrEmptyPCMBuffer
	}
	if targetBuckets <= 0 {
		targetBuckets = 100
	}

	buf := &PCMBuffer{
		Channels:   1,
		SampleRate: 48000,
		NumSamples: len(samples),
		Samples:    [][]float64{samples},
	}

	wd, err := ExtractWaveformFromPCM(buf, targetBuckets)
	if err != nil {
		return nil, err
	}
	return wd.Peaks, nil
}

// ExtractWaveformFromReader decodes an audio stream from an io.Reader and extracts waveform data.
func ExtractWaveformFromReader(r io.Reader, targetBuckets int) (*WaveformData, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading stream: %w", err)
	}
	if len(data) == 0 {
		return nil, ErrEmptyAudioStream
	}

	streamInfo, err := DecodeHeader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decoding header: %w", err)
	}

	var pcmReader io.Reader
	if streamInfo.DataSize > 0 && int64(len(data)) >= streamInfo.DataSize {
		offset := len(data) - int(streamInfo.DataSize)
		pcmReader = bytes.NewReader(data[offset:])
	} else {
		pcmReader = bytes.NewReader(data)
	}

	pcm, err := UnpackPCM(pcmReader, streamInfo)
	if err != nil {
		return nil, fmt.Errorf("unpacking pcm: %w", err)
	}

	return ExtractWaveformFromPCM(pcm, targetBuckets)
}

// ExtractWaveformFromFile opens an audio file on disk and extracts waveform data.
func ExtractWaveformFromFile(filePath string, targetBuckets int) (*WaveformData, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open audio file: %w", err)
	}
	defer f.Close()

	return ExtractWaveformFromReader(f, targetBuckets)
}

// WaveformDataJSON marshals the complete WaveformData structure into a formatted JSON string.
func WaveformDataJSON(wd *WaveformData) (string, error) {
	if wd == nil {
		return "", errors.New("waveform data is nil")
	}
	b, err := json.Marshal(wd)
	if err != nil {
		return "", fmt.Errorf("marshal waveform data: %w", err)
	}
	return string(b), nil
}

// WaveformPeaksJSON marshals a slice of float64 peak values into compact JSON array.
func WaveformPeaksJSON(peaks []float64) (string, error) {
	if peaks == nil {
		return "[]", nil
	}
	b, err := json.Marshal(peaks)
	if err != nil {
		return "", fmt.Errorf("marshal peaks: %w", err)
	}
	return string(b), nil
}

// PackWaveformBinary quantizes peaks into a compact byte slice where each peak
// in [0.0, 1.0] is scaled to an 8-bit unsigned integer [0, 255].
func PackWaveformBinary(peaks []float64) []byte {
	out := make([]byte, len(peaks))
	for i, p := range peaks {
		clamped := math.Max(0.0, math.Min(1.0, p))
		out[i] = uint8(math.Round(clamped * 255.0))
	}
	return out
}

// UnpackWaveformBinary reconstructs normalized float64 peaks from an 8-bit quantized byte slice.
func UnpackWaveformBinary(data []byte) ([]float64, error) {
	if len(data) == 0 {
		return []float64{}, nil
	}
	out := make([]float64, len(data))
	for i, b := range data {
		out[i] = float64(b) / 255.0
	}
	return out, nil
}

// PackWaveformBucketsBinary packs dual envelopes (Min, Max, RMS) into a binary byte slice.
func PackWaveformBucketsBinary(buckets []WaveformBucket) []byte {
	buf := new(bytes.Buffer)
	// Header: 4-byte bucket count
	_ = binary.Write(buf, binary.LittleEndian, uint32(len(buckets)))
	for _, b := range buckets {
		// Min is mapped from [-1.0, 0.0] to [0, 127]
		minNorm := math.Max(0.0, math.Min(1.0, -b.Min))
		minByte := uint8(math.Round(minNorm * 127.0))

		// Max is mapped from [0.0, 1.0] to [0, 127]
		maxNorm := math.Max(0.0, math.Min(1.0, b.Max))
		maxByte := uint8(math.Round(maxNorm * 127.0))

		// RMS is mapped from [0.0, 1.0] to [0, 255]
		rmsNorm := math.Max(0.0, math.Min(1.0, b.RMS))
		rmsByte := uint8(math.Round(rmsNorm * 255.0))

		buf.WriteByte(minByte)
		buf.WriteByte(maxByte)
		buf.WriteByte(rmsByte)
	}
	return buf.Bytes()
}

// UnpackWaveformBucketsBinary unpacks binary encoded buckets into WaveformBucket slice.
func UnpackWaveformBucketsBinary(data []byte) ([]WaveformBucket, error) {
	if len(data) < 4 {
		return nil, ErrCorruptBinaryData
	}
	count := binary.LittleEndian.Uint32(data[:4])
	expectedLen := 4 + int(count)*3
	if len(data) != expectedLen {
		return nil, ErrCorruptBinaryData
	}

	buckets := make([]WaveformBucket, count)
	offset := 4
	for i := 0; i < int(count); i++ {
		minByte := data[offset]
		maxByte := data[offset+1]
		rmsByte := data[offset+2]
		offset += 3

		minVal := -(float64(minByte) / 127.0)
		maxVal := float64(maxByte) / 127.0
		rmsVal := float64(rmsByte) / 255.0
		absMax := math.Max(math.Abs(minVal), math.Abs(maxVal))

		buckets[i] = WaveformBucket{
			Min:    minVal,
			Max:    maxVal,
			AbsMax: absMax,
			RMS:    rmsVal,
		}
	}
	return buckets, nil
}

// ResampleWaveform resamples a peak slice to a different number of target buckets
// using linear interpolation while preserving peak transients.
func ResampleWaveform(peaks []float64, targetBuckets int) []float64 {
	srcLen := len(peaks)
	if srcLen == 0 || targetBuckets <= 0 {
		return []float64{}
	}
	if srcLen == targetBuckets {
		result := make([]float64, srcLen)
		copy(result, peaks)
		return result
	}

	out := make([]float64, targetBuckets)

	if targetBuckets < srcLen {
		// Downsampling: peak retention in window
		ratio := float64(srcLen) / float64(targetBuckets)
		for b := 0; b < targetBuckets; b++ {
			start := int(float64(b) * ratio)
			end := int(float64(b+1) * ratio)
			if end <= start {
				end = start + 1
			}
			if end > srcLen {
				end = srcLen
			}
			maxVal := 0.0
			for _, p := range peaks[start:end] {
				if p > maxVal {
					maxVal = p
				}
			}
			out[b] = maxVal
		}
	} else {
		// Upsampling: linear interpolation
		ratio := float64(srcLen-1) / float64(targetBuckets-1)
		for b := 0; b < targetBuckets; b++ {
			srcPos := float64(b) * ratio
			idx0 := int(math.Floor(srcPos))
			idx1 := idx0 + 1
			if idx1 >= srcLen {
				idx1 = srcLen - 1
			}
			alpha := srcPos - float64(idx0)
			out[b] = (1.0-alpha)*peaks[idx0] + alpha*peaks[idx1]
		}
	}

	return out
}

// ScaleWaveformToDecibels compresses peaks from linear amplitude to decibel scale
// with a specified noise floor (e.g. -60 dB), mapping resulting dB values to [0.0, 1.0].
func ScaleWaveformToDecibels(peaks []float64, minDB float64) []float64 {
	if minDB >= 0.0 {
		minDB = -60.0
	}
	out := make([]float64, len(peaks))
	for i, p := range peaks {
		if p <= 1e-6 {
			out[i] = 0.0
			continue
		}
		db := 20.0 * math.Log10(p)
		if db < minDB {
			db = minDB
		}
		// Scale [minDB, 0.0] -> [0.0, 1.0]
		norm := (db - minDB) / (-minDB)
		out[i] = math.Max(0.0, math.Min(1.0, norm))
	}
	return out
}

// GenerateWaveformSVGPath constructs a path string ("M... L... Z") representing a
// mirrored waveform centered at vertical height/2 across the specified pixel width.
func GenerateWaveformSVGPath(peaks []float64, width, height float64) string {
	n := len(peaks)
	if n == 0 || width <= 0 || height <= 0 {
		return "M0,0 Z"
	}

	midY := height / 2.0
	stepX := width / float64(n)

	var upperPath bytes.Buffer
	var lowerPath bytes.Buffer

	upperPath.WriteString(fmt.Sprintf("M0,%.2f", midY))

	// Forward top half
	for i, p := range peaks {
		x := float64(i)*stepX + stepX*0.5
		halfAmp := (p * midY) * 0.95
		yTop := midY - halfAmp
		upperPath.WriteString(fmt.Sprintf(" L%.2f,%.2f", x, yTop))
	}
	upperPath.WriteString(fmt.Sprintf(" L%.2f,%.2f", width, midY))

	// Backward bottom half
	for i := n - 1; i >= 0; i-- {
		x := float64(i)*stepX + stepX*0.5
		halfAmp := (peaks[i] * midY) * 0.95
		yBottom := midY + halfAmp
		lowerPath.WriteString(fmt.Sprintf(" L%.2f,%.2f", x, yBottom))
	}
	lowerPath.WriteString(" Z")

	return upperPath.String() + lowerPath.String()
}

// GenerateWaveformDualSVGPath builds a mirrored SVG path using explicit positive and negative envelopes.
func GenerateWaveformDualSVGPath(buckets []WaveformBucket, width, height float64) string {
	n := len(buckets)
	if n == 0 || width <= 0 || height <= 0 {
		return "M0,0 Z"
	}

	midY := height / 2.0
	stepX := width / float64(n)

	var topBuf bytes.Buffer
	var botBuf bytes.Buffer

	topBuf.WriteString(fmt.Sprintf("M0,%.2f", midY))

	for i, b := range buckets {
		x := float64(i)*stepX + stepX*0.5
		topY := midY - (b.Max * midY * 0.95)
		topBuf.WriteString(fmt.Sprintf(" L%.2f,%.2f", x, topY))
	}
	topBuf.WriteString(fmt.Sprintf(" L%.2f,%.2f", width, midY))

	for i := n - 1; i >= 0; i-- {
		x := float64(i)*stepX + stepX*0.5
		// b.Min is negative, so subtracting b.Min adds positive offset downwards
		botY := midY - (buckets[i].Min * midY * 0.95)
		botBuf.WriteString(fmt.Sprintf(" L%.2f,%.2f", x, botY))
	}
	botBuf.WriteString(" Z")

	return topBuf.String() + botBuf.String()
}

// GenerateWaveformBarsSVG generates a complete standalone SVG document containing
// individual rounded vertical bars matching modern streaming UI visualizer styling.
func GenerateWaveformBarsSVG(peaks []float64, width, height, barWidth, barGap float64) string {
	n := len(peaks)
	if n == 0 || width <= 0 || height <= 0 {
		return `<svg xmlns="http://www.w3.org/2000/svg" width="0" height="0"></svg>`
	}

	totalBarPitch := barWidth + barGap
	maxBarsPossible := int(width / totalBarPitch)
	resampled := peaks
	if maxBarsPossible > 0 && maxBarsPossible != n {
		resampled = ResampleWaveform(peaks, maxBarsPossible)
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.1f %.1f" width="100%%" height="100%%">`, width, height))
	buf.WriteString(`<defs><linearGradient id="wfGrad" x1="0" y1="1" x2="0" y2="0"><stop offset="0%" stop-color="#4f46e5"/><stop offset="100%" stop-color="#fbbf24"/></linearGradient></defs>`)

	for i, p := range resampled {
		x := float64(i) * totalBarPitch
		barH := math.Max(2.0, p*height*0.95)
		y := height - barH
		rx := barWidth * 0.5
		buf.WriteString(fmt.Sprintf(`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="%.2f" fill="url(#wfGrad)"/>`, x, y, barWidth, barH, rx))
	}

	buf.WriteString(`</svg>`)
	return buf.String()
}
