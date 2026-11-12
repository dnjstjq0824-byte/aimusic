// Package forensics provides pitch-shift invariant acoustic fingerprint matching (+-5 semitones)
// via cyclic chromagram cross-correlation and key transposition alignment.
package forensics

import (
	"encoding/json"
	"math"

	"avibe-platform/backend/internal/pkg/audio"
)

// PitchShiftMatchResult holds the result of pitch-invariant fingerprint comparison.
type PitchShiftMatchResult struct {
	Matched           bool    `json:"matched"`
	BestShiftSemitone int     `json:"best_shift_semitone"` // -5 to +5 semitones
	SimilarityScore   float64 `json:"similarity_score"`    // 0.0 to 1.0 (or 0-100%)
	ConfidencePct     float64 `json:"confidence_pct"`
	Summary           string  `json:"summary"`
}

// PitchShiftMatcher compares audio fingerprints across pitch transposition shifts (-5 to +5 semitones).
type PitchShiftMatcher struct {
	threshold float64 // match threshold, e.g. 0.80 (80%)
	frameSize int
	hopSize   int
}

// NewPitchShiftMatcher creates a new pitch-shift invariant matcher.
func NewPitchShiftMatcher() *PitchShiftMatcher {
	return &PitchShiftMatcher{
		threshold: 0.80,
		frameSize: 2048,
		hopSize:   1024,
	}
}

// ChromaFrame represents energy distribution across 12 chromatic pitch classes (C, C#, D, ... B).
type ChromaFrame [12]float64

// ExtractChromagram computes 12-bin chroma frames across time for a PCM buffer.
func (m *PitchShiftMatcher) ExtractChromagram(buf *audio.PCMBuffer) ([]ChromaFrame, error) {
	if buf == nil {
		return nil, audio.ErrNilPCMBuffer
	}
	if buf.NumSamples == 0 || buf.Channels == 0 || len(buf.Samples) == 0 {
		return nil, audio.ErrEmptyPCMBuffer
	}

	sampleRate := buf.SampleRate
	numSamples := buf.NumSamples
	channels := buf.Channels

	// Downmix to mono
	mono := make([]float64, numSamples)
	scale := 1.0 / float64(channels)
	for i := 0; i < numSamples; i++ {
		var sum float64
		for ch := 0; ch < channels; ch++ {
			sum += buf.Samples[ch][i]
		}
		mono[i] = sum * scale
	}

	monoBuf := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{mono},
	}

	stftEngine, err := audio.NewSTFTEngine(audio.STFTConfig{
		FFTSize:    m.frameSize,
		HopSize:    m.hopSize,
		Window:     audio.WindowHann,
		NumMelBins: 16,
	})
	if err != nil {
		return nil, err
	}

	stftRes, err := stftEngine.Forward(monoBuf)
	if err != nil {
		return nil, err
	}
	if len(stftRes.Frames) == 0 || len(stftRes.Frames[0]) == 0 {
		return nil, audio.ErrEmptyPCMBuffer
	}

	frames := stftRes.Frames[0]
	numFrames := len(frames)
	chroma := make([]ChromaFrame, numFrames)

	// Frequency resolution per FFT bin
	binHz := float64(sampleRate) / float64(m.frameSize)
	numBins := m.frameSize/2 + 1

	// Pre-map FFT bin k to pitch class (0..11, 0=C, 1=C#, ..., 9=A 440Hz)
	binToPitchClass := make([]int, numBins)
	validBin := make([]bool, numBins)
	for k := 1; k < numBins; k++ {
		freq := float64(k) * binHz
		// Consider musical range: C2 (65.4 Hz) to B7 (3951 Hz)
		if freq >= 65.0 && freq <= 4000.0 {
			midi := 69.0 + 12.0*math.Log2(freq/440.0)
			pitchClass := int(math.Round(midi)) % 12
			if pitchClass < 0 {
				pitchClass += 12
			}
			binToPitchClass[k] = pitchClass
			validBin[k] = true
		}
	}

	// Map spectral magnitude to 12 chroma bins per frame
	for f := 0; f < numFrames; f++ {
		frameMag := frames[f].Magnitude
		var cf ChromaFrame
		for k := 1; k < numBins; k++ {
			if validBin[k] && k < len(frameMag) {
				pc := binToPitchClass[k]
				cf[pc] += frameMag[k]
			}
		}

		// L2 normalize chroma frame for energy invariance
		var normSq float64
		for pc := 0; pc < 12; pc++ {
			normSq += cf[pc] * cf[pc]
		}
		if normSq > 1e-9 {
			norm := math.Sqrt(normSq)
			for pc := 0; pc < 12; pc++ {
				cf[pc] /= norm
			}
		}
		chroma[f] = cf
	}

	return chroma, nil
}

// Match compares a query audio buffer against a reference audio buffer across shifts from -5 to +5 semitones.
func (m *PitchShiftMatcher) Match(query, reference *audio.PCMBuffer) (*PitchShiftMatchResult, error) {
	qChroma, err := m.ExtractChromagram(query)
	if err != nil {
		return nil, err
	}
	refChroma, err := m.ExtractChromagram(reference)
	if err != nil {
		return nil, err
	}

	return m.MatchChromagrams(qChroma, refChroma)
}

// MatchChromagrams performs circular shift cross-correlation between query and reference chromagram sequences.
func (m *PitchShiftMatcher) MatchChromagrams(qChroma, refChroma []ChromaFrame) (*PitchShiftMatchResult, error) {
	frames := len(qChroma)
	if len(refChroma) < frames {
		frames = len(refChroma)
	}
	if frames == 0 {
		return &PitchShiftMatchResult{Matched: false}, nil
	}

	bestShift := 0
	bestSim := -1.0

	// Test shifts k in [-5, +5]
	for k := -5; k <= 5; k++ {
		var totalDot float64
		for f := 0; f < frames; f++ {
			qFrame := qChroma[f]
			refFrame := refChroma[f]

			var frameDot float64
			for pc := 0; pc < 12; pc++ {
				// Query pitch class shifted by k to match reference
				shiftedPC := (pc + k) % 12
				if shiftedPC < 0 {
					shiftedPC += 12
				}
				frameDot += qFrame[shiftedPC] * refFrame[pc]
			}
			totalDot += frameDot
		}

		sim := totalDot / float64(frames)
		if sim > bestSim {
			bestSim = sim
			bestShift = k
		}
	}

	matched := bestSim >= m.threshold
	conf := bestSim * 100.0
	if conf > 100.0 {
		conf = 100.0
	}
	if conf < 0.0 {
		conf = 0.0
	}

	summary := "No pitch-shifted copyright match detected."
	if matched {
		if bestShift == 0 {
			summary = "Exact pitch match detected (original key)."
		} else {
			summary = "Pitch-shifted copyright match detected (transposed by semi-tones)."
		}
	}

	return &PitchShiftMatchResult{
		Matched:           matched,
		BestShiftSemitone: bestShift,
		SimilarityScore:   math.Round(bestSim*10000) / 10000,
		ConfidencePct:     math.Round(conf*10) / 10,
		Summary:           summary,
	}, nil
}

// ToJSON serializes pitch shift match result.
func (r *PitchShiftMatchResult) ToJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
