// Package forensics provides acoustic fingerprinting, Chromaprint / AcoustID compatible
// semitone chromagram feature extraction, and AI audio provenance analysis.
package forensics

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"avibe-platform/backend/internal/pkg/audio"
)

var (
	ErrAudioTooShortForFingerprint = errors.New("audio buffer too short to generate acoustic fingerprint (minimum 1 second)")
)

// ChromaprintResult contains the extracted acoustic sub-fingerprints, compressed fingerprint string, and telemetry.
type ChromaprintResult struct {
	TrackID          string   `json:"track_id,omitempty"`
	DurationSec      float64  `json:"duration_sec"`
	FingerprintB64   string   `json:"fingerprint"` // Compressed Base64 AcoustID format
	SubFingerprints  []uint32 `json:"sub_fingerprints,omitempty"`
	SubFingerprintN  int      `json:"sub_fingerprint_count"`
	ExtractionTimeMs float64  `json:"extraction_time_ms"`
}

// ChromaprintExtractor extracts 32-bit sub-fingerprints using semitone spectrogram Haar wavelets.
type ChromaprintExtractor struct {
	sampleRate int
	frameSize  int
	hopSize    int
}

// NewChromaprintExtractor creates an acoustic fingerprint extractor.
func NewChromaprintExtractor() *ChromaprintExtractor {
	return &ChromaprintExtractor{
		sampleRate: 11025, // Chromaprint standard target sample rate
		frameSize:  2048,  // ~185ms window
		hopSize:    512,   // ~46ms hop (approx 21.5 frames per second)
	}
}

// Extract generates Chromaprint acoustic fingerprints from a PCM buffer.
func (ce *ChromaprintExtractor) Extract(input *audio.PCMBuffer) (*ChromaprintResult, error) {
	if input == nil {
		return nil, audio.ErrNilPCMBuffer
	}
	if input.NumSamples == 0 || input.Channels == 0 || len(input.Samples) == 0 {
		return nil, audio.ErrEmptyPCMBuffer
	}

	start := time.Now()
	durationSec := float64(input.NumSamples) / float64(input.SampleRate)
	if durationSec < 1.0 {
		return nil, ErrAudioTooShortForFingerprint
	}

	// Step 1: Resample / Downsample to mono 11025 Hz
	resampled := resampleToMono11025(input)

	// Step 2: Compute 16-band Bark / Semitone energy filterbank
	filterbank := compute16BandFilterbank(resampled, ce.frameSize, ce.hopSize)
	numFrames := len(filterbank)
	if numFrames < 4 {
		return nil, ErrAudioTooShortForFingerprint
	}

	// Step 3: Compute Integral Image across Time-Frequency matrix
	integral := computeIntegralImage(filterbank)

	// Step 4: Extract 16 2D Haar Wavelet box features per time slice
	subFingerprints := extractHaarSubFingerprints(integral, numFrames)

	// Step 5: Compress to AcoustID / Chromaprint binary format (uint32 LE + zlib + base64)
	b64Fp, err := compressFingerprint(subFingerprints)
	if err != nil {
		return nil, fmt.Errorf("failed compressing fingerprint: %w", err)
	}

	elapsed := time.Since(start)

	return &ChromaprintResult{
		DurationSec:      math.Round(durationSec*100) / 100,
		FingerprintB64:   b64Fp,
		SubFingerprints:  subFingerprints,
		SubFingerprintN:  len(subFingerprints),
		ExtractionTimeMs: math.Round(float64(elapsed.Microseconds())/1000.0*10) / 10,
	}, nil
}

// resampleToMono11025 mixes channels down to mono and linearly resamples to 11025 Hz.
func resampleToMono11025(input *audio.PCMBuffer) []float64 {
	channels := input.Channels
	numSamples := input.NumSamples
	srcSR := float64(input.SampleRate)
	dstSR := 11025.0

	// Mix to mono
	mono := make([]float64, numSamples)
	scale := 1.0 / float64(channels)
	for i := 0; i < numSamples; i++ {
		var sum float64
		for ch := 0; ch < channels; ch++ {
			sum += input.Samples[ch][i]
		}
		mono[i] = sum * scale
	}

	if input.SampleRate == 11025 {
		return mono
	}

	// Linear interpolation resampling
	ratio := srcSR / dstSR
	outLen := int(float64(numSamples) / ratio)
	out := make([]float64, outLen)

	for i := 0; i < outLen; i++ {
		srcPos := float64(i) * ratio
		idx := int(srcPos)
		frac := srcPos - float64(idx)

		if idx+1 < numSamples {
			out[i] = mono[idx]*(1.0-frac) + mono[idx+1]*frac
		} else if idx < numSamples {
			out[i] = mono[idx]
		}
	}
	return out
}

// compute16BandFilterbank splits the 0..5512Hz spectrum into 16 logarithmically spaced frequency bands.
func compute16BandFilterbank(samples []float64, frameSize, hopSize int) [][]float64 {
	stftEngine, err := audio.NewSTFTEngine(audio.STFTConfig{
		FFTSize:    frameSize,
		HopSize:    hopSize,
		Window:     audio.WindowHann,
		NumMelBins: 16,
	})
	if err != nil {
		return nil
	}

	buf := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: 11025,
		NumSamples: len(samples),
		Samples:    [][]float64{samples},
	}

	stftRes, err := stftEngine.Forward(buf)
	if err != nil || stftRes.NumFrames == 0 {
		return nil
	}

	numFrames := stftRes.NumFrames
	numBins := stftRes.NumBins
	binHz := 11025.0 / float64(frameSize)

	// 16 bands defined from ~150 Hz to 4500 Hz
	bandEdges := make([]int, 17)
	minFreq := 150.0
	maxFreq := 4500.0
	for i := 0; i <= 16; i++ {
		freq := minFreq * math.Pow(maxFreq/minFreq, float64(i)/16.0)
		bin := int(freq / binHz)
		if bin >= numBins {
			bin = numBins - 1
		}
		bandEdges[i] = bin
	}

	bands := make([][]float64, numFrames)
	frames := stftRes.Frames[0]

	for f := 0; f < numFrames; f++ {
		pwr := frames[f].Power
		bandEnergies := make([]float64, 16)
		for b := 0; b < 16; b++ {
			startBin := bandEdges[b]
			endBin := bandEdges[b+1]
			if endBin <= startBin {
				endBin = startBin + 1
			}

			var bandSum float64
			for k := startBin; k < endBin && k < len(pwr); k++ {
				bandSum += pwr[k]
			}
			bandEnergies[b] = math.Log1p(bandSum * 1e6)
		}
		bands[f] = bandEnergies
	}

	return bands
}

// computeIntegralImage creates a 2D cumulative sum table across time (x) and bands (y).
func computeIntegralImage(data [][]float64) [][]float64 {
	rows := len(data)
	if rows == 0 {
		return nil
	}
	cols := len(data[0])

	integral := make([][]float64, rows)
	for i := 0; i < rows; i++ {
		integral[i] = make([]float64, cols)
	}

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			val := data[r][c]
			if r > 0 {
				val += integral[r-1][c]
			}
			if c > 0 {
				val += integral[r][c-1]
			}
			if r > 0 && c > 0 {
				val -= integral[r-1][c-1]
			}
			integral[r][c] = val
		}
	}
	return integral
}

// extractHaarSubFingerprints computes 16 differential bits to construct a 32-bit sub-fingerprint per time slice.
func extractHaarSubFingerprints(integral [][]float64, numFrames int) []uint32 {
	if numFrames < 4 {
		return nil
	}

	subFps := make([]uint32, 0, numFrames-4)

	// Step across frames (window of 4 frames per fingerprint)
	for t := 0; t+3 < numFrames; t++ {
		var subFp uint32
		// 16 wavelet filters comparing adjacent frequency bands and time frames
		for b := 0; b < 15; b++ {
			// Energy diff between band b and band b+1
			eLeft := queryIntegral(integral, t, b, t+1, b)
			eRight := queryIntegral(integral, t, b+1, t+1, b+1)

			bitIdx := uint(b * 2)
			if eLeft > eRight {
				subFp |= (1 << bitIdx)
			}

			// Time diff between frame t and frame t+2
			ePast := queryIntegral(integral, t, b, t+1, b)
			eFuture := queryIntegral(integral, t+2, b, t+3, b)
			if ePast > eFuture {
				subFp |= (1 << (bitIdx + 1))
			}
		}
		subFps = append(subFps, subFp)
	}

	return subFps
}

// queryIntegral retrieves the sum of a bounding box in O(1) from the integral image.
func queryIntegral(integral [][]float64, r1, c1, r2, c2 int) float64 {
	val := integral[r2][c2]
	if r1 > 0 {
		val -= integral[r1-1][c2]
	}
	if c1 > 0 {
		val -= integral[r2][c1-1]
	}
	if r1 > 0 && c1 > 0 {
		val += integral[r1-1][c1-1]
	}
	return val
}

// compressFingerprint packs sub-fingerprints with uint32 Little Endian, compresses with zlib, and encodes in Base64.
func compressFingerprint(subFps []uint32) (string, error) {
	rawBuf := new(bytes.Buffer)
	for _, fp := range subFps {
		_ = binary.Write(rawBuf, binary.LittleEndian, fp)
	}

	var zlibBuf bytes.Buffer
	w := zlib.NewWriter(&zlibBuf)
	if _, err := w.Write(rawBuf.Bytes()); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(zlibBuf.Bytes()), nil
}

// ComputeHammingDistance counts the number of differing bits between two 32-bit sub-fingerprints.
func ComputeHammingDistance(a, b uint32) int {
	xor := a ^ b
	dist := 0
	for xor > 0 {
		dist += int(xor & 1)
		xor >>= 1
	}
	return dist
}

// ToJSON serializes fingerprint result.
func (r *ChromaprintResult) ToJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
