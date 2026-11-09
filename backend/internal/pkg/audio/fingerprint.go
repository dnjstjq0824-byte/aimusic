// Package audio provides professional cryptographic and acoustic audio fingerprinting,
// container-agnostic canonical PCM SHA-256 hashing, perceptual spectral subband bit-vectors,
// similarity matching, and HLS chunk Merkle tree validation.
package audio

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"math/bits"
)

// Constants for perceptual subband analysis.
const (
	FingerprintFFTSize  = 2048
	FingerprintHopSize  = 1024
	FingerprintNumBands = 32
	MinFrequencyHz      = 300.0
	MaxFrequencyHz      = 5000.0
)

// Fingerprint validation errors.
var (
	ErrNilFingerprintBuffer = errors.New("pcm buffer is nil for fingerprinting")
	ErrShortAudioBuffer     = errors.New("pcm buffer is too short for acoustic fingerprinting")
	ErrEmptyChunkSlice      = errors.New("chunk slice is empty for merkle tree computation")
)

// AudioFingerprint encapsulates cryptographic and perceptual audio fingerprints.
type AudioFingerprint struct {
	CanonicalPCMHashSHA256 string   `json:"canonical_pcm_hash_sha256"`
	AcousticHashSHA256     string   `json:"acoustic_hash_sha256"`
	SubbandBitVectors      []uint32 `json:"subband_bit_vectors"`
	DurationSec            float64  `json:"duration_sec"`
	TotalFrames            int      `json:"total_frames"`
	SampleRate             int      `json:"sample_rate"`
	Channels               int      `json:"channels"`
}

// FingerprintMatch describes the similarity between two audio fingerprints.
type FingerprintMatch struct {
	ExactPCM        bool    `json:"exact_pcm"`
	Similarity      float64 `json:"similarity"`
	HammingDistance int     `json:"hamming_distance"`
	TotalBits       int     `json:"total_bits"`
	IsDuplicate     bool    `json:"is_duplicate"`
	IsDerivative    bool    `json:"is_derivative"`
}

// MerkleTree stores leaf hashes and root hash for stream chunk integrity.
type MerkleTree struct {
	LeafHashes []string `json:"leaf_hashes"`
	RootHash   string   `json:"root_hash"`
}

// FingerprintEngine generates canonical and perceptual audio fingerprints.
type FingerprintEngine struct {
	sampleRate int
	fftSize    int
	hopSize    int
	numBands   int
	bandEdges  []int
	window     []float64
}

// NewFingerprintEngine initializes a fingerprint generator.
func NewFingerprintEngine() *FingerprintEngine {
	fftSize := FingerprintFFTSize
	hopSize := FingerprintHopSize
	sampleRate := DefaultTargetSampleRate
	numBands := FingerprintNumBands

	binWidth := float64(sampleRate) / float64(fftSize)
	bandEdges := make([]int, numBands+1)
	ratio := math.Pow(MaxFrequencyHz/MinFrequencyHz, 1.0/float64(numBands))

	for k := 0; k <= numBands; k++ {
		freq := MinFrequencyHz * math.Pow(ratio, float64(k))
		bin := int(math.Round(freq / binWidth))
		if bin < 1 {
			bin = 1
		}
		if bin >= fftSize/2 {
			bin = fftSize/2 - 1
		}
		bandEdges[k] = bin
	}

	for k := 1; k <= numBands; k++ {
		if bandEdges[k] <= bandEdges[k-1] {
			bandEdges[k] = bandEdges[k-1] + 1
		}
	}

	window := make([]float64, fftSize)
	for n := 0; n < fftSize; n++ {
		window[n] = 0.5 * (1.0 - math.Cos(2.0*math.Pi*float64(n)/float64(fftSize-1)))
	}

	return &FingerprintEngine{
		sampleRate: sampleRate,
		fftSize:    fftSize,
		hopSize:    hopSize,
		numBands:   numBands,
		bandEdges:  bandEdges,
		window:     window,
	}
}

// GenerateFingerprint extracts canonical PCM SHA-256 and perceptual spectral fingerprints.
func (fe *FingerprintEngine) GenerateFingerprint(buf *PCMBuffer) (*AudioFingerprint, error) {
	if buf == nil {
		return nil, ErrNilFingerprintBuffer
	}
	if buf.Channels <= 0 || buf.NumSamples <= 0 || len(buf.Samples) == 0 {
		return nil, ErrEmptyAudioBuffer
	}

	workBuf := buf
	if buf.SampleRate != fe.sampleRate {
		resampled, err := ResampleTo48kHz(buf)
		if err != nil {
			return nil, fmt.Errorf("resample to 48kHz for fingerprinting: %w", err)
		}
		workBuf = resampled
	}

	pcmHash, err := ComputeCanonicalPCMHash(workBuf)
	if err != nil {
		return nil, fmt.Errorf("compute canonical pcm hash: %w", err)
	}

	numSamples := workBuf.NumSamples
	durationSec := float64(numSamples) / float64(fe.sampleRate)

	mono := make([]float64, numSamples)
	invCh := 1.0 / float64(workBuf.Channels)
	for ch := 0; ch < workBuf.Channels; ch++ {
		for n := 0; n < numSamples; n++ {
			mono[n] += workBuf.Samples[ch][n] * invCh
		}
	}

	if numSamples < fe.fftSize {
		return &AudioFingerprint{
			CanonicalPCMHashSHA256: pcmHash,
			AcousticHashSHA256:     pcmHash,
			SubbandBitVectors:      []uint32{0},
			DurationSec:            durationSec,
			TotalFrames:            1,
			SampleRate:             fe.sampleRate,
			Channels:               workBuf.Channels,
		}, nil
	}

	numFrames := (numSamples-fe.fftSize)/fe.hopSize + 1
	subbandEnergies := make([][]float64, numFrames)

	real := make([]float64, fe.fftSize)
	imag := make([]float64, fe.fftSize)

	for m := 0; m < numFrames; m++ {
		offset := m * fe.hopSize
		for n := 0; n < fe.fftSize; n++ {
			real[n] = mono[offset+n] * fe.window[n]
			imag[n] = 0.0
		}

		fft(real, imag)

		energies := make([]float64, fe.numBands)
		for k := 0; k < fe.numBands; k++ {
			startBin := fe.bandEdges[k]
			endBin := fe.bandEdges[k+1]
			var sumSq float64
			for b := startBin; b < endBin; b++ {
				sumSq += real[b]*real[b] + imag[b]*imag[b]
			}
			energies[k] = sumSq
		}
		subbandEnergies[m] = energies
	}

	bitVectors := make([]uint32, numFrames)
	numBands := 16
	for m := 1; m < numFrames; m++ {
		var word uint32
		// Lower 16 bits: Spectral subband contrast (pitch/timbre invariant to volume)
		for k := 0; k < numBands-1; k++ {
			if subbandEnergies[m][k] > subbandEnergies[m][k+1] {
				word |= (1 << uint(k))
			}
		}
		// Upper 16 bits: Temporal flux (rhythm and energy dynamics)
		for k := 0; k < numBands; k++ {
			if subbandEnergies[m][k] > subbandEnergies[m-1][k] {
				word |= (1 << uint(16+k))
			}
		}
		bitVectors[m] = word
	}
	if numFrames > 1 {
		bitVectors[0] = bitVectors[1]
	}

	hAcoustic := sha256.New()
	for _, w := range bitVectors {
		hAcoustic.Write([]byte{
			byte(w),
			byte(w >> 8),
			byte(w >> 16),
			byte(w >> 24),
		})
	}
	acousticHash := fmt.Sprintf("%x", hAcoustic.Sum(nil))

	return &AudioFingerprint{
		CanonicalPCMHashSHA256: pcmHash,
		AcousticHashSHA256:     acousticHash,
		SubbandBitVectors:      bitVectors,
		DurationSec:            durationSec,
		TotalFrames:            numFrames,
		SampleRate:             fe.sampleRate,
		Channels:               workBuf.Channels,
	}, nil
}

// ComputeCanonicalPCMHash computes a 24-bit Little Endian interleaved SHA-256 hash.
func ComputeCanonicalPCMHash(buf *PCMBuffer) (string, error) {
	if buf == nil {
		return "", ErrNilFingerprintBuffer
	}
	if buf.Channels <= 0 || buf.NumSamples <= 0 || len(buf.Samples) == 0 {
		return "", ErrEmptyAudioBuffer
	}

	workBuf := buf
	if buf.SampleRate != DefaultTargetSampleRate {
		resampled, err := ResampleTo48kHz(buf)
		if err != nil {
			return "", fmt.Errorf("resample to 48kHz for canonical hash: %w", err)
		}
		workBuf = resampled
	}

	channels := workBuf.Channels
	numSamples := workBuf.NumSamples
	rawBytes := make([]byte, numSamples*channels*3)
	idx := 0

	for n := 0; n < numSamples; n++ {
		for ch := 0; ch < channels; ch++ {
			s := workBuf.Samples[ch][n]
			if s < -1.0 {
				s = -1.0
			} else if s > 1.0 {
				s = 1.0
			}
			val := int32(math.Round(s * 8388607.0))
			if val < -8388608 {
				val = -8388608
			} else if val > 8388607 {
				val = 8388607
			}
			uVal := uint32(val) & 0x00FFFFFF
			rawBytes[idx] = byte(uVal)
			rawBytes[idx+1] = byte(uVal >> 8)
			rawBytes[idx+2] = byte(uVal >> 16)
			idx += 3
		}
	}

	h := sha256.Sum256(rawBytes)
	return fmt.Sprintf("%x", h), nil
}

// CompareFingerprints calculates acoustic alignment and similarity between two fingerprints.
func CompareFingerprints(fp1, fp2 *AudioFingerprint) *FingerprintMatch {
	if fp1 == nil || fp2 == nil {
		return &FingerprintMatch{}
	}

	exactPCM := (fp1.CanonicalPCMHashSHA256 != "" && fp1.CanonicalPCMHashSHA256 == fp2.CanonicalPCMHashSHA256)
	if exactPCM {
		return &FingerprintMatch{
			ExactPCM:     true,
			Similarity:   1.0,
			IsDuplicate:  true,
			IsDerivative: true,
		}
	}

	n1 := len(fp1.SubbandBitVectors)
	n2 := len(fp2.SubbandBitVectors)
	if n1 == 0 || n2 == 0 {
		return &FingerprintMatch{}
	}

	maxOffset := 50
	if maxOffset > n1/2 {
		maxOffset = n1 / 2
	}
	if maxOffset > n2/2 {
		maxOffset = n2 / 2
	}

	bestSimilarity := 0.0
	bestDist := 0
	bestBits := 0

	for offset := -maxOffset; offset <= maxOffset; offset++ {
		var totalBits int
		var bitErrors int

		for i := 0; i < n1; i++ {
			j := i + offset
			if j < 0 || j >= n2 {
				continue
			}
			xor := fp1.SubbandBitVectors[i] ^ fp2.SubbandBitVectors[j]
			bitErrors += bits.OnesCount32(xor)
			totalBits += 32
		}

		if totalBits > 0 {
			sim := 1.0 - float64(bitErrors)/float64(totalBits)
			if sim > bestSimilarity {
				bestSimilarity = sim
				bestDist = bitErrors
				bestBits = totalBits
			}
		}
	}

	return &FingerprintMatch{
		ExactPCM:        false,
		Similarity:      bestSimilarity,
		HammingDistance: bestDist,
		TotalBits:       bestBits,
		IsDuplicate:     bestSimilarity >= 0.95,
		IsDerivative:    bestSimilarity >= 0.80,
	}
}

// ComputeChunkMerkleTree constructs a Merkle tree from raw stream chunks.
func ComputeChunkMerkleTree(chunks [][]byte) (*MerkleTree, error) {
	if len(chunks) == 0 {
		return nil, ErrEmptyChunkSlice
	}

	leaves := make([]string, len(chunks))
	currentLevel := make([][]byte, len(chunks))
	for i, c := range chunks {
		h := sha256.Sum256(c)
		hexStr := fmt.Sprintf("%x", h)
		leaves[i] = hexStr
		currentLevel[i] = []byte(hexStr)
	}

	for len(currentLevel) > 1 {
		var nextLevel [][]byte
		for i := 0; i < len(currentLevel); i += 2 {
			if i+1 < len(currentLevel) {
				combined := append(currentLevel[i], currentLevel[i+1]...)
				h := sha256.Sum256(combined)
				nextLevel = append(nextLevel, []byte(fmt.Sprintf("%x", h)))
			} else {
				combined := append(currentLevel[i], currentLevel[i]...)
				h := sha256.Sum256(combined)
				nextLevel = append(nextLevel, []byte(fmt.Sprintf("%x", h)))
			}
		}
		currentLevel = nextLevel
	}

	return &MerkleTree{
		LeafHashes: leaves,
		RootHash:   string(currentLevel[0]),
	}, nil
}

// fft performs an in-place Radix-2 Cooley-Tukey decimation-in-time Fast Fourier Transform.
func fft(real, imag []float64) {
	n := len(real)
	j := 0
	for i := 0; i < n-1; i++ {
		if i < j {
			real[i], real[j] = real[j], real[i]
			imag[i], imag[j] = imag[j], imag[i]
		}
		k := n / 2
		for k <= j {
			j -= k
			k /= 2
		}
		j += k
	}

	for step := 2; step <= n; step <<= 1 {
		halfStep := step / 2
		angle := -2.0 * math.Pi / float64(step)
		wRealStep := math.Cos(angle)
		wImagStep := math.Sin(angle)

		for group := 0; group < n; group += step {
			wReal := 1.0
			wImag := 0.0
			for k := 0; k < halfStep; k++ {
				idxU := group + k
				idxV := group + k + halfStep

				vR := real[idxV]*wReal - imag[idxV]*wImag
				vI := real[idxV]*wImag + imag[idxV]*wReal

				uR := real[idxU]
				uI := imag[idxU]

				real[idxU] = uR + vR
				imag[idxU] = uI + vI
				real[idxV] = uR - vR
				imag[idxV] = uI - vI

				nextWR := wReal*wRealStep - wImag*wImagStep
				nextWI := wReal*wImagStep + wImag*wRealStep
				wReal = nextWR
				wImag = nextWI
			}
		}
	}
}
