// Package forensics provides time-stretch invariant acoustic fingerprint matching (+-20% tempo)
// via multi-scale tempo grid search and timeline resampling.
package forensics

import (
	"encoding/json"
	"math"

	"avibe-platform/backend/internal/pkg/audio"
)

// TempoMatchResult encapsulates time-stretch invariant matching metrics.
type TempoMatchResult struct {
	Matched         bool    `json:"matched"`
	BestSpeedFactor float64 `json:"best_speed_factor"` // e.g. 0.80 to 1.20
	SimilarityScore float64 `json:"similarity_score"`  // 0.0 to 1.0 (or 0-100%)
	ConfidencePct   float64 `json:"confidence_pct"`
	Summary         string  `json:"summary"`
}

// TempoInvariantMatcher performs scale-invariant fingerprint matching across speed variations (-20% to +20%).
type TempoInvariantMatcher struct {
	threshold  float64
	minSpeed   float64
	maxSpeed   float64
	speedSteps int
	frameSize  int
	hopSize    int
}

// NewTempoInvariantMatcher creates a new tempo-invariant matcher.
func NewTempoInvariantMatcher() *TempoInvariantMatcher {
	return &TempoInvariantMatcher{
		threshold:  0.80,
		minSpeed:   0.80, // -20% tempo (0.8x)
		maxSpeed:   1.20, // +20% tempo (1.2x)
		speedSteps: 21,   // steps of 0.02 (0.80, 0.82, ..., 1.20)
		frameSize:  2048,
		hopSize:    1024,
	}
}

// ExtractEnergyVectors computes multi-band energy profile across time.
func (m *TempoInvariantMatcher) ExtractEnergyVectors(buf *audio.PCMBuffer) ([][]float64, error) {
	if buf == nil {
		return nil, audio.ErrNilPCMBuffer
	}
	if buf.NumSamples == 0 || buf.Channels == 0 || len(buf.Samples) == 0 {
		return nil, audio.ErrEmptyPCMBuffer
	}

	sampleRate := buf.SampleRate
	numSamples := buf.NumSamples
	channels := buf.Channels

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

	numMel := 16
	stftEngine, err := audio.NewSTFTEngine(audio.STFTConfig{
		FFTSize:    m.frameSize,
		HopSize:    m.hopSize,
		Window:     audio.WindowHann,
		NumMelBins: numMel,
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

	rawFrames := stftRes.Frames[0]
	numFrames := len(rawFrames)
	numBins := m.frameSize/2 + 1
	binHz := float64(sampleRate) / float64(m.frameSize)

	// Group FFT bins into 8 logarithmic sub-bands
	subBands := 8
	bandBoundaries := []float64{100, 250, 500, 1000, 2000, 4000, 8000, 16000}

	vectors := make([][]float64, numFrames)
	for f := 0; f < numFrames; f++ {
		frame := rawFrames[f]
		vec := make([]float64, subBands)

		for k := 1; k < numBins && k < len(frame.Magnitude); k++ {
			fHz := float64(k) * binHz
			bandIdx := -1
			for b := 0; b < subBands-1; b++ {
				if fHz >= bandBoundaries[b] && fHz < bandBoundaries[b+1] {
					bandIdx = b
					break
				}
			}
			if bandIdx >= 0 {
				vec[bandIdx] += frame.Magnitude[k]
			}
		}

		// Normalize vector
		var sumSq float64
		for b := 0; b < subBands; b++ {
			sumSq += vec[b] * vec[b]
		}
		if sumSq > 1e-9 {
			norm := math.Sqrt(sumSq)
			for b := 0; b < subBands; b++ {
				vec[b] /= norm
			}
		}
		vectors[f] = vec
	}

	return vectors, nil
}

// Match compares a query audio buffer against a reference audio across speed factors (0.8x to 1.2x).
func (m *TempoInvariantMatcher) Match(query, reference *audio.PCMBuffer) (*TempoMatchResult, error) {
	qVectors, err := m.ExtractEnergyVectors(query)
	if err != nil {
		return nil, err
	}
	refVectors, err := m.ExtractEnergyVectors(reference)
	if err != nil {
		return nil, err
	}

	return m.MatchVectors(qVectors, refVectors)
}

// MatchVectors resamples the query vectors along the time axis by each scale factor and evaluates similarity.
func (m *TempoInvariantMatcher) MatchVectors(qVectors, refVectors [][]float64) (*TempoMatchResult, error) {
	qLen := len(qVectors)
	refLen := len(refVectors)
	if qLen == 0 || refLen == 0 {
		return &TempoMatchResult{Matched: false}, nil
	}

	bestSpeed := 1.0
	bestSim := -1.0

	stepSize := (m.maxSpeed - m.minSpeed) / float64(m.speedSteps-1)

	for step := 0; step < m.speedSteps; step++ {
		speed := m.minSpeed + float64(step)*stepSize

		// Query speed factor: query_tempo / ref_tempo = speed
		// An event at reference frame rf corresponds to query frame qPos = rf / speed
		evalFrames := refLen
		maxRefFrames := int(float64(qLen) * speed)
		if evalFrames > maxRefFrames {
			evalFrames = maxRefFrames
		}
		if evalFrames < 5 {
			continue
		}

		var totalSim float64
		var countFrames int

		for rf := 0; rf < evalFrames; rf++ {
			qPos := float64(rf) / speed
			qIdx0 := int(math.Floor(qPos))
			qIdx1 := qIdx0 + 1
			alpha := qPos - float64(qIdx0)

			if qIdx0 >= qLen {
				break
			}

			// Linear interpolation of query feature vector
			refVec := refVectors[rf]
			vecDim := len(refVec)

			var qNormSq, refNormSq, dot float64
			for d := 0; d < vecDim; d++ {
				v0 := qVectors[qIdx0][d]
				v1 := v0
				if qIdx1 < qLen {
					v1 = qVectors[qIdx1][d]
				}
				interpolated := (1.0-alpha)*v0 + alpha*v1
				dot += interpolated * refVec[d]
				qNormSq += interpolated * interpolated
				refNormSq += refVec[d] * refVec[d]
			}

			qNorm := math.Sqrt(qNormSq)
			refNorm := math.Sqrt(refNormSq)

			var frameSim float64
			if qNorm < 1e-6 && refNorm < 1e-6 {
				frameSim = 1.0 // both quiet
			} else if qNorm < 1e-6 || refNorm < 1e-6 {
				frameSim = 0.0 // mismatch
			} else {
				frameSim = dot / (qNorm * refNorm)
			}

			totalSim += frameSim
			countFrames++
		}

		if countFrames > 0 {
			sim := totalSim / float64(countFrames)
			if sim > bestSim {
				bestSim = sim
				bestSpeed = speed
			}
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

	summary := "No tempo-stretched copyright match detected."
	if matched {
		if math.Abs(bestSpeed-1.0) < 0.02 {
			summary = "Exact tempo match detected (original playback speed)."
		} else {
			summary = "Time-stretched copyright match detected (tempo altered to bypass Content ID)."
		}
	}

	return &TempoMatchResult{
		Matched:         matched,
		BestSpeedFactor: math.Round(bestSpeed*100) / 100,
		SimilarityScore: math.Round(bestSim*10000) / 10000,
		ConfidencePct:   math.Round(conf*10) / 10,
		Summary:         summary,
	}, nil
}

// ToJSON serializes tempo match result.
func (r *TempoMatchResult) ToJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
