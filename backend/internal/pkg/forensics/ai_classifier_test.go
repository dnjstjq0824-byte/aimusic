package forensics

import (
	"math"
	"math/rand"
	"testing"

	"avibe-platform/backend/internal/pkg/audio"
)

func TestAIOriginClassifier_HumanLiveRecording(t *testing.T) {
	classifier, err := NewAIOriginClassifier()
	if err != nil {
		t.Fatalf("NewAIOriginClassifier failed: %v", err)
	}

	sampleRate := 48000
	numSamples := sampleRate * 2 // 2 seconds
	samples := make([]float64, numSamples)

	// Clean live acoustic recording: 440Hz guitar + 8kHz cymbals + 18kHz room sheen
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		samples[i] = 0.4*math.Sin(2*math.Pi*440.0*tSec) +
			0.2*math.Sin(2*math.Pi*8000.0*tSec) +
			0.1*math.Sin(2*math.Pi*18000.0*tSec)
	}

	buf := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	res, err := classifier.Classify(buf)
	if err != nil {
		t.Fatalf("Classify failed: %v", err)
	}
	t.Logf("Breakdown: %+v, AI prob: %f", res.FeatureBreakdown, res.AIProbability)

	if res.PredictedOrigin != OriginHumanPerformed {
		t.Errorf("live recording falsely classified as %s", res.PredictedOrigin)
	}
	if res.AIProbability > 0.05 {
		t.Errorf("DoD violated: human recording AI probability %f exceeded 0.05 (<0.5%% false positive threshold)", res.AIProbability)
	}
	if !res.PassesFalsePositiveGate {
		t.Errorf("expected PassesFalsePositiveGate to be true")
	}

	jsonStr, err := res.ToJSON()
	if err != nil || len(jsonStr) == 0 {
		t.Fatalf("ToJSON failed: %v", err)
	}
}

func TestAIOriginClassifier_AISynthesized(t *testing.T) {
	classifier, err := NewAIOriginClassifier()
	if err != nil {
		t.Fatalf("NewAIOriginClassifier failed: %v", err)
	}

	sampleRate := 48000
	numSamples := sampleRate * 2
	samples := make([]float64, numSamples)

	// Inject both Suno cutoff (<16kHz) and Udio phase jumps
	r := rand.New(rand.NewSource(123))
	chunkLen := sampleRate / 100

	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		chunkIdx := i / chunkLen
		phaseJump := float64(chunkIdx) * r.Float64() * math.Pi

		// Tones below 15.5kHz with phase discontinuities and comb ripples, 0 above 16kHz
		base := 0.2*math.Sin(2*math.Pi*440.0*tSec) +
			0.2*math.Sin(2*math.Pi*1200.0*tSec) +
			0.2*math.Sin(2*math.Pi*2000.0*tSec+phaseJump) +
			0.2*math.Sin(2*math.Pi*12000.0*tSec) +
			0.2*math.Sin(2*math.Pi*15000.0*tSec)
		comb := 0.1 * math.Sin(2*math.Pi*200.0*tSec) * base
		samples[i] = base + comb
	}

	// Apply 15.5kHz neural vocoder lowpass brickwall filter
	filteredSamples := audio.LowpassIIR(samples, sampleRate, 15500.0)

	buf := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{filteredSamples},
	}

	res, err := classifier.Classify(buf)
	if err != nil {
		t.Fatalf("Classify failed: %v", err)
	}
	t.Logf("Breakdown: %+v, AI prob: %f", res.FeatureBreakdown, res.AIProbability)

	if res.PredictedOrigin != OriginAISynthesized {
		t.Errorf("synthetic AI audio not classified as AI_SYNTHESIZED: got %s (AI prob: %f)",
			res.PredictedOrigin, res.AIProbability)
	}
	if res.AIProbability < 0.70 {
		t.Errorf("expected AI probability >= 0.70, got %f", res.AIProbability)
	}
}
