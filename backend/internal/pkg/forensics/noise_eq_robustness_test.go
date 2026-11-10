package forensics

import (
	"math"
	"math/rand"
	"testing"

	"avibe-platform/backend/internal/pkg/audio"
)

// Helper to synthesize a complex acoustic track with transients and tonal components
func synthesizeAcousticSong(sampleRate int, durationSec float64) *audio.PCMBuffer {
	numSamples := int(durationSec * float64(sampleRate))
	samples := make([]float64, numSamples)

	r := rand.New(rand.NewSource(12345))

	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)

		// Percussive transients at 4 Hz
		transient := 0.0
		tBeat := math.Mod(tSec, 0.25)
		if tBeat < 0.02 {
			transient = math.Exp(-tBeat*150.0) * (r.Float64()*2.0 - 1.0) * 0.4
		}

		// Evolving harmonic melodic content (rising pitch over time)
		pitchRamp := 1.0 + (tSec / durationSec)
		lead := 0.25*math.Sin(2*math.Pi*(300.0*pitchRamp)*tSec) +
			0.15*math.Sin(2*math.Pi*(600.0*pitchRamp)*tSec) +
			0.10*math.Sin(2*math.Pi*(900.0*pitchRamp)*tSec)

		// Bass line with distinct low progression
		bass := 0.3 * math.Sin(2*math.Pi*(110.0+40.0*math.Floor(tSec))*tSec)

		samples[i] = lead + bass + transient
	}

	return &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}
}

func TestNoiseEQRobustness_CleanMatch(t *testing.T) {
	verifier := NewNoiseEQRobustnessVerifier()
	sampleRate := 48000

	ref := synthesizeAcousticSong(sampleRate, 2.5)
	query := synthesizeAcousticSong(sampleRate, 2.5)

	res, err := verifier.Match(query, ref)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	if !res.Matched {
		t.Errorf("expected clean track to match, got false")
	}
	if res.RobustSimilarity < 0.95 {
		t.Errorf("expected similarity >= 0.95, got %f", res.RobustSimilarity)
	}
	t.Logf("Clean track match: Similarity=%.2f%%", res.RobustSimilarity*100)
}

func TestNoiseEQRobustness_15dBSNRWhiteNoise(t *testing.T) {
	verifier := NewNoiseEQRobustnessVerifier()
	sampleRate := 48000

	ref := synthesizeAcousticSong(sampleRate, 2.5)

	// Create query with additive noise at ~15 dB SNR
	// Signal RMS ~ 0.3, 15 dB down is RMS ~ 0.3 / 10^(15/20) = 0.3 / 5.62 = ~0.053
	numSamples := ref.NumSamples
	noisySamples := make([]float64, numSamples)
	r := rand.New(rand.NewSource(999))

	noiseAmp := 0.055
	for i := 0; i < numSamples; i++ {
		white := (r.Float64()*2.0 - 1.0) * noiseAmp
		noisySamples[i] = ref.Samples[0][i] + white
	}

	query := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{noisySamples},
	}

	res, err := verifier.Match(query, ref)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	// Verification against DoD: SNR 15dB noise environment matching maintained
	if !res.Matched {
		t.Fatalf("DoD violated: failed to match audio under SNR 15dB noise: similarity=%f", res.RobustSimilarity)
	}
	if res.RobustSimilarity < 0.55 {
		t.Errorf("expected robust similarity >= 0.55, got %f", res.RobustSimilarity)
	}
	t.Logf("SNR 15dB noise match: Similarity=%.2f%% (threshold=%.2f%%)", res.RobustSimilarity*100, verifier.threshold*100)
}

func TestNoiseEQRobustness_SevereLowpassCutoff(t *testing.T) {
	verifier := NewNoiseEQRobustnessVerifier()
	sampleRate := 48000

	ref := synthesizeAcousticSong(sampleRate, 2.5)

	// Lowpass filter query at 2500Hz (eliminating high frequencies)
	filtered := audio.LowpassIIR(ref.Samples[0], sampleRate, 2500.0)

	query := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: ref.NumSamples,
		Samples:    [][]float64{filtered},
	}

	res, err := verifier.Match(query, ref)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	// Verification against DoD: EQ filtering tampering matching maintained
	if !res.Matched {
		t.Fatalf("DoD violated: failed to match audio under severe lowpass filtering: similarity=%f", res.RobustSimilarity)
	}
	t.Logf("Severe 2.5kHz lowpass match: Similarity=%.2f%% (threshold=%.2f%%)", res.RobustSimilarity*100, verifier.threshold*100)
}

func TestNoiseEQRobustness_UnrelatedAudioRejection(t *testing.T) {
	verifier := NewNoiseEQRobustnessVerifier()
	sampleRate := 48000

	ref := synthesizeAcousticSong(sampleRate, 2.5)

	// Unrelated ambient sound (500Hz and 2000Hz sines with no rhythm)
	numSamples := ref.NumSamples
	unrelatedSamples := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		unrelatedSamples[i] = 0.3*math.Sin(2*math.Pi*500.0*tSec) + 0.2*math.Sin(2*math.Pi*2000.0*tSec)
	}

	unrelated := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{unrelatedSamples},
	}

	res, err := verifier.Match(unrelated, ref)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	if res.Matched {
		t.Errorf("unrelated audio was falsely matched: similarity=%f", res.RobustSimilarity)
	}
	t.Logf("Unrelated audio rejected: Similarity=%.2f%% (threshold=%.2f%%)", res.RobustSimilarity*100, verifier.threshold*100)
}
