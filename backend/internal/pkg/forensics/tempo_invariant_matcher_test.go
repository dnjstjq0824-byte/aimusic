package forensics

import (
	"math"
	"testing"

	"avibe-platform/backend/internal/pkg/audio"
)

// Helper to synthesize rhythmically pulsed audio with distinct frequency evolution
func synthesizeRhythmicPattern(sampleRate int, speedFactor float64, durationSec float64) *audio.PCMBuffer {
	numSamples := int(durationSec * float64(sampleRate))
	samples := make([]float64, numSamples)

	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		// Effective time scaled by speedFactor
		tEff := tSec * speedFactor

		// Beats at 2 Hz (120 BPM)
		beatEnv := 0.2 + 0.8*math.Abs(math.Sin(2*math.Pi*2.0*tEff))

		// Sweeping frequencies across sub-bands
		f1 := 220.0 + 80.0*math.Sin(2*math.Pi*0.5*tEff)
		f2 := 880.0 + 150.0*math.Cos(2*math.Pi*0.3*tEff)
		f3 := 2500.0 + 400.0*math.Sin(2*math.Pi*0.7*tEff)

		val := (0.3*math.Sin(2*math.Pi*f1*tEff) +
			0.2*math.Sin(2*math.Pi*f2*tEff) +
			0.15*math.Sin(2*math.Pi*f3*tEff)) * beatEnv

		samples[i] = val
	}

	return &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}
}

func TestTempoInvariantMatcher_ExactTempoMatch(t *testing.T) {
	matcher := NewTempoInvariantMatcher()
	sampleRate := 48000

	ref := synthesizeRhythmicPattern(sampleRate, 1.0, 3.0)
	query := synthesizeRhythmicPattern(sampleRate, 1.0, 3.0)

	res, err := matcher.Match(query, ref)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	if !res.Matched {
		t.Errorf("expected 1.0x tempo match, got false")
	}
	if math.Abs(res.BestSpeedFactor-1.0) > 0.03 {
		t.Errorf("expected speed factor ~1.00, got %f", res.BestSpeedFactor)
	}
	if res.SimilarityScore < 0.95 {
		t.Errorf("expected similarity >= 0.95, got %f", res.SimilarityScore)
	}
	t.Logf("Exact tempo match: Speed=%.2fx, Similarity=%.2f%%", res.BestSpeedFactor, res.SimilarityScore*100)
}

func TestTempoInvariantMatcher_AcceleratedPlayback(t *testing.T) {
	matcher := NewTempoInvariantMatcher()
	sampleRate := 48000

	ref := synthesizeRhythmicPattern(sampleRate, 1.0, 3.5)
	// Query is played +10% faster (1.10x tempo)
	query := synthesizeRhythmicPattern(sampleRate, 1.10, 3.5)

	res, err := matcher.Match(query, ref)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	if !res.Matched {
		t.Fatalf("DoD violated: time-stretched query failed to match")
	}
	if math.Abs(res.BestSpeedFactor-1.10) > 0.04 {
		t.Errorf("expected speed factor ~1.10, got %f", res.BestSpeedFactor)
	}
	if res.SimilarityScore < 0.80 {
		t.Errorf("expected similarity >= 0.80, got %f", res.SimilarityScore)
	}
	t.Logf("+10%% Tempo match: Speed=%.2fx, Similarity=%.2f%%", res.BestSpeedFactor, res.SimilarityScore*100)
}

func TestTempoInvariantMatcher_DeceleratedPlayback(t *testing.T) {
	matcher := NewTempoInvariantMatcher()
	sampleRate := 48000

	ref := synthesizeRhythmicPattern(sampleRate, 1.0, 3.5)
	// Query is played -14% slower (0.86x tempo)
	query := synthesizeRhythmicPattern(sampleRate, 0.86, 3.5)

	res, err := matcher.Match(query, ref)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	if !res.Matched {
		t.Fatalf("DoD violated: time-stretched query failed to match")
	}
	if math.Abs(res.BestSpeedFactor-0.86) > 0.04 {
		t.Errorf("expected speed factor ~0.86, got %f", res.BestSpeedFactor)
	}
	if res.SimilarityScore < 0.80 {
		t.Errorf("expected similarity >= 0.80, got %f", res.SimilarityScore)
	}
	t.Logf("-14%% Tempo match: Speed=%.2fx, Similarity=%.2f%%", res.BestSpeedFactor, res.SimilarityScore*100)
}

func TestTempoInvariantMatcher_UnrelatedAudioRejection(t *testing.T) {
	matcher := NewTempoInvariantMatcher()
	sampleRate := 48000

	ref := synthesizeRhythmicPattern(sampleRate, 1.0, 3.0)

	// Unrelated random white noise burst
	numSamples := 3 * sampleRate
	noiseSamples := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		noiseSamples[i] = 0.2 * math.Sin(2*math.Pi*6000.0*tSec)
	}
	unrelated := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{noiseSamples},
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
