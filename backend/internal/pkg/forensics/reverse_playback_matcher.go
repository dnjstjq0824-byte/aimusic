// Package forensics provides reverse playback detection and inverted forward-match auditing
// to intercept evasion attempts where audio is reversed in time to bypass content fingerprinting.
package forensics

import (
	"encoding/json"
	"math"

	"avibe-platform/backend/internal/pkg/audio"
)

// ReversePlaybackMatchResult encapsulates reverse evasion detection telemetry.
type ReversePlaybackMatchResult struct {
	Matched                bool    `json:"matched"`
	IsReversedPlayback     bool    `json:"is_reversed_playback"`
	ForwardSimilarity      float64 `json:"forward_similarity"`  // Direct similarity (0.0 to 1.0)
	InvertedSimilarity     float64 `json:"inverted_similarity"` // Time-reversed query similarity
	EvasionAttemptDetected bool    `json:"evasion_attempt_detected"`
	ConfidencePct          float64 `json:"confidence_pct"`
	Summary                string  `json:"summary"`
}

// ReversePlaybackMatcher tests both forward and time-inverted audio to defeat reverse evasion.
type ReversePlaybackMatcher struct {
	threshold float64
	verifier  *NoiseEQRobustnessVerifier
}

// NewReversePlaybackMatcher creates a new reverse playback matcher.
func NewReversePlaybackMatcher() *ReversePlaybackMatcher {
	return &ReversePlaybackMatcher{
		threshold: 0.65,
		verifier:  NewNoiseEQRobustnessVerifier(),
	}
}

// InvertBuffer returns a new PCMBuffer with all sample channels flipped horizontally in time.
func InvertBuffer(buf *audio.PCMBuffer) (*audio.PCMBuffer, error) {
	if buf == nil {
		return nil, audio.ErrNilPCMBuffer
	}
	if buf.NumSamples == 0 || buf.Channels == 0 || len(buf.Samples) == 0 {
		return nil, audio.ErrEmptyPCMBuffer
	}

	channels := buf.Channels
	numSamples := buf.NumSamples
	inverted := make([][]float64, channels)

	for ch := 0; ch < channels; ch++ {
		orig := buf.Samples[ch]
		rev := make([]float64, numSamples)
		for i := 0; i < numSamples; i++ {
			rev[i] = orig[numSamples-1-i]
		}
		inverted[ch] = rev
	}

	return &audio.PCMBuffer{
		Channels:   channels,
		SampleRate: buf.SampleRate,
		NumSamples: numSamples,
		Samples:    inverted,
	}, nil
}

// Match tests the query against reference in both forward direction and time-inverted direction.
func (m *ReversePlaybackMatcher) Match(query, reference *audio.PCMBuffer) (*ReversePlaybackMatchResult, error) {
	if query == nil || reference == nil {
		return nil, audio.ErrNilPCMBuffer
	}

	// 1. Forward match
	forwardRes, err := m.verifier.Match(query, reference)
	if err != nil {
		return nil, err
	}
	fwdSim := forwardRes.RobustSimilarity

	// 2. Invert query buffer (time reversal)
	invQuery, err := InvertBuffer(query)
	if err != nil {
		return nil, err
	}

	// 3. Inverted match
	invRes, err := m.verifier.Match(invQuery, reference)
	if err != nil {
		return nil, err
	}
	invSim := invRes.RobustSimilarity

	// Determine if reversed playback evasion was attempted
	isReversed := invSim >= m.threshold && invSim > (fwdSim+0.15)
	matched := forwardRes.Matched || isReversed
	bestSim := fwdSim
	if invSim > bestSim {
		bestSim = invSim
	}

	summary := "Standard playback verified; no reverse audio evasion detected."
	if isReversed {
		summary = "Reverse playback evasion detected: audio was reversed to bypass content ID; matches original catalog when time-inverted."
	} else if matched {
		summary = "Direct forward copyright match verified."
	}

	conf := bestSim * 100.0
	if conf > 100.0 {
		conf = 100.0
	}

	return &ReversePlaybackMatchResult{
		Matched:                matched,
		IsReversedPlayback:     isReversed,
		ForwardSimilarity:      math.Round(fwdSim*10000) / 10000,
		InvertedSimilarity:     math.Round(invSim*10000) / 10000,
		EvasionAttemptDetected: isReversed,
		ConfidencePct:          math.Round(conf*10) / 10,
		Summary:                summary,
	}, nil
}

// ToJSON serializes reverse playback match result.
func (r *ReversePlaybackMatchResult) ToJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
