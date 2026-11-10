package forensics

import (
	"math"
	"testing"

	"avibe-platform/backend/internal/pkg/audio"
)

// Helper to synthesize a chord progression / melodic motif
func synthesizeMotif(sampleRate int, semitoneOffset int, durationSec float64) *audio.PCMBuffer {
	numSamples := int(durationSec * float64(sampleRate))
	samples := make([]float64, numSamples)

	// Base frequencies: C4 (261.63), E4 (329.63), G4 (392.00)
	baseFreqs := []float64{261.63, 329.63, 392.00}
	transpositionFactor := math.Pow(2.0, float64(semitoneOffset)/12.0)

	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		var val float64
		for _, f := range baseFreqs {
			freq := f * transpositionFactor
			val += 0.25 * math.Sin(2*math.Pi*freq*tSec)
			// Add 2nd and 3rd harmonics
			val += 0.12 * math.Sin(2*math.Pi*freq*2.0*tSec)
			val += 0.06 * math.Sin(2*math.Pi*freq*3.0*tSec)
		}
		samples[i] = val
	}

	return &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}
}

func TestPitchShiftMatcher_OriginalKeyMatch(t *testing.T) {
	matcher := NewPitchShiftMatcher()
	sampleRate := 48000

	ref := synthesizeMotif(sampleRate, 0, 2.0)
	query := synthesizeMotif(sampleRate, 0, 2.0)

	res, err := matcher.Match(query, ref)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	if !res.Matched {
		t.Errorf("expected original key to match, got false")
	}
	if res.BestShiftSemitone != 0 {
		t.Errorf("expected shift 0, got %d", res.BestShiftSemitone)
	}
	if res.SimilarityScore < 0.95 {
		t.Errorf("expected similarity >= 0.95, got %f", res.SimilarityScore)
	}
	t.Logf("Original key match: Shift=%d, Similarity=%.2f%%", res.BestShiftSemitone, res.SimilarityScore*100)
}

func TestPitchShiftMatcher_TransposedPlus3Semitones(t *testing.T) {
	matcher := NewPitchShiftMatcher()
	sampleRate := 48000

	ref := synthesizeMotif(sampleRate, 0, 2.0)
	query := synthesizeMotif(sampleRate, +3, 2.0) // +3 semitones pitch shifted

	res, err := matcher.Match(query, ref)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	if !res.Matched {
		t.Fatalf("DoD violated: pitch-shifted query failed to match")
	}
	if res.BestShiftSemitone != 3 {
		t.Errorf("expected best shift +3, got %d", res.BestShiftSemitone)
	}
	if res.SimilarityScore < 0.85 {
		t.Errorf("expected similarity >= 0.85, got %f", res.SimilarityScore)
	}
	t.Logf("+3 Semitone transposition match: Shift=%d, Similarity=%.2f%%", res.BestShiftSemitone, res.SimilarityScore*100)
}

func TestPitchShiftMatcher_TransposedMinus4Semitones(t *testing.T) {
	matcher := NewPitchShiftMatcher()
	sampleRate := 48000

	ref := synthesizeMotif(sampleRate, 0, 2.0)
	query := synthesizeMotif(sampleRate, -4, 2.0) // -4 semitones pitch shifted

	res, err := matcher.Match(query, ref)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	if !res.Matched {
		t.Fatalf("DoD violated: pitch-shifted query failed to match")
	}
	if res.BestShiftSemitone != -4 {
		t.Errorf("expected best shift -4, got %d", res.BestShiftSemitone)
	}
	if res.SimilarityScore < 0.85 {
		t.Errorf("expected similarity >= 0.85, got %f", res.SimilarityScore)
	}
	t.Logf("-4 Semitone transposition match: Shift=%d, Similarity=%.2f%%", res.BestShiftSemitone, res.SimilarityScore*100)
}

func TestPitchShiftMatcher_UnrelatedAudioRejection(t *testing.T) {
	matcher := NewPitchShiftMatcher()
	sampleRate := 48000

	ref := synthesizeMotif(sampleRate, 0, 2.0)

	// Unrelated melody: F# (369.99), A# (466.16), C# (554.37)
	numSamples := 2 * sampleRate
	unrelatedSamples := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		unrelatedSamples[i] = 0.3*math.Sin(2*math.Pi*369.99*tSec) +
			0.2*math.Sin(2*math.Pi*1100.0*tSec)
	}
	unrelated := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{unrelatedSamples},
	}

	res, err := matcher.Match(unrelated, ref)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	if res.Matched {
		t.Errorf("unrelated audio was falsely matched: similarity %f", res.SimilarityScore)
	}
	t.Logf("Unrelated audio rejected: Similarity=%.2f%% (threshold=%.2f%%)", res.SimilarityScore*100, matcher.threshold*100)
}
