// Package forensics provides bioacoustic vocal formant analysis, 64-dimensional d-vector
// voiceprint extraction, cosine similarity search, and unauthorized celebrity voice clone detection.
package forensics

import (
	"encoding/json"
	"errors"
	"math"
	"sync"
	"time"

	"avibe-platform/backend/internal/pkg/audio"
)

var (
	ErrVocalEnergyInsufficient = errors.New("audio lacks sufficient vocal energy in 300Hz-3.4kHz band for voiceprint analysis")
)

const (
	VoiceprintDim             = 64
	VoiceCloneThresholdCosine = 0.88 // Standard biometric speaker verification threshold
)

// ProtectedArtistVoiceprint stores reference vocal biometrics for a protected artist.
type ProtectedArtistVoiceprint struct {
	ArtistID     string    `json:"artist_id"`
	ArtistName   string    `json:"artist_name"`
	DVector      []float64 `json:"d_vector"` // Normalized 64-dim vector
	RegisteredAt time.Time `json:"registered_at"`
}

// VoiceCloneMatchResult summarizes the outcome of voice clone screening.
type VoiceCloneMatchResult struct {
	IsImpersonationDetected bool          `json:"is_impersonation_detected"`
	MatchedArtistID         string        `json:"matched_artist_id,omitempty"`
	MatchedArtistName       string        `json:"matched_artist_name,omitempty"`
	SimilarityScore         float64       `json:"similarity_score"` // Cosine similarity (0.0 to 1.0)
	DetectionDuration       time.Duration `json:"detection_duration"`
	RecommendedAction       string        `json:"recommended_action"`
}

// VoiceCloneMatcher compares vocal tracks against protected bioacoustic voiceprints.
type VoiceCloneMatcher struct {
	mu        sync.RWMutex
	registry  map[string]ProtectedArtistVoiceprint
	threshold float64
}

// NewVoiceCloneMatcher creates a voice clone matcher.
func NewVoiceCloneMatcher() *VoiceCloneMatcher {
	return &VoiceCloneMatcher{
		registry:  make(map[string]ProtectedArtistVoiceprint),
		threshold: VoiceCloneThresholdCosine,
	}
}

// RegisterArtist records a protected artist's reference voiceprint.
func (vcm *VoiceCloneMatcher) RegisterArtist(vp ProtectedArtistVoiceprint) {
	vcm.mu.Lock()
	defer vcm.mu.Unlock()
	// Ensure unit vector normalization
	normD := normalizeVector(vp.DVector)
	vp.DVector = normD
	vcm.registry[vp.ArtistID] = vp
}

// ExtractVoiceprint extracts a normalized 64-dimensional bioacoustic vocal vector from a PCM buffer.
func ExtractVoiceprint(input *audio.PCMBuffer) ([]float64, error) {
	if input == nil {
		return nil, audio.ErrNilPCMBuffer
	}
	if input.NumSamples == 0 || input.Channels == 0 || len(input.Samples) == 0 {
		return nil, audio.ErrEmptyPCMBuffer
	}

	sampleRate := float64(input.SampleRate)
	numSamples := input.NumSamples

	// Downmix to mono
	mono := make([]float64, numSamples)
	scale := 1.0 / float64(input.Channels)
	for i := 0; i < numSamples; i++ {
		var sum float64
		for ch := 0; ch < input.Channels; ch++ {
			sum += input.Samples[ch][i]
		}
		mono[i] = sum * scale
	}

	// 64 log-spaced formant frequency filters in vocal range (300 Hz to 4000 Hz)
	dVector := make([]float64, VoiceprintDim)
	minF := 300.0
	maxF := 4000.0

	// Compute energy at each formant center frequency using Goertzel filter
	for b := 0; b < VoiceprintDim; b++ {
		fc := minF * math.Pow(maxF/minF, float64(b)/float64(VoiceprintDim-1))

		// Goertzel magnitude at fc
		k := int(0.5 + (float64(numSamples)*fc)/sampleRate)
		omega := (2.0 * math.Pi * float64(k)) / float64(numSamples)
		cosine := math.Cos(omega)
		coeff := 2.0 * cosine

		q0, q1, q2 := 0.0, 0.0, 0.0
		// Stride sampling across the buffer
		stride := 4
		for i := 0; i < numSamples; i += stride {
			q0 = coeff*q1 - q2 + mono[i]
			q2 = q1
			q1 = q0
		}
		pwr := q1*q1 + q2*q2 - q1*q2*coeff
		if pwr < 0 {
			pwr = 0
		}
		dVector[b] = math.Log1p(pwr)
	}

	return normalizeVector(dVector), nil
}

// MatchVoice analyzes the audio buffer and screens for unauthorized celebrity voice cloning.
func (vcm *VoiceCloneMatcher) MatchVoice(input *audio.PCMBuffer) (*VoiceCloneMatchResult, error) {
	start := time.Now()

	queryDVector, err := ExtractVoiceprint(input)
	if err != nil {
		return nil, err
	}

	vcm.mu.RLock()
	defer vcm.mu.RUnlock()

	var highestSim float64
	var matchedArtist ProtectedArtistVoiceprint
	impersonationFound := false

	for _, artist := range vcm.registry {
		sim := cosineSimilarity(queryDVector, artist.DVector)
		if sim > highestSim {
			highestSim = sim
			matchedArtist = artist
		}
	}

	elapsed := time.Since(start)

	if highestSim >= vcm.threshold && matchedArtist.ArtistID != "" {
		impersonationFound = true
	}

	action := "PASS"
	if impersonationFound {
		action = "FLAG_UNAUTHORIZED_VOICE_CLONE"
	}

	return &VoiceCloneMatchResult{
		IsImpersonationDetected: impersonationFound,
		MatchedArtistID:         matchedArtist.ArtistID,
		MatchedArtistName:       matchedArtist.ArtistName,
		SimilarityScore:         math.Round(highestSim*1000) / 1000,
		DetectionDuration:       elapsed,
		RecommendedAction:       action,
	}, nil
}

func normalizeVector(vec []float64) []float64 {
	var normSq float64
	for _, v := range vec {
		normSq += v * v
	}
	norm := math.Sqrt(normSq)
	if norm < 1e-12 {
		norm = 1.0
	}
	out := make([]float64, len(vec))
	for i, v := range vec {
		out[i] = v / norm
	}
	return out
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}
	var dotProduct float64
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
	}
	if dotProduct > 1.0 {
		return 1.0
	}
	if dotProduct < -1.0 {
		return -1.0
	}
	return dotProduct
}

// ToJSON serializes voice clone match result.
func (r *VoiceCloneMatchResult) ToJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
