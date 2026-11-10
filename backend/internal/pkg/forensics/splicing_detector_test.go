package forensics

import (
	"math"
	"math/rand"
	"testing"

	"avibe-platform/backend/internal/pkg/audio"
)

func TestSplicingDetector_ContinuousAudio(t *testing.T) {
	sd := NewSplicingDetector()

	sampleRate := 48000
	numSamples := sampleRate * 2 // 2 seconds
	samples := make([]float64, numSamples)

	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		samples[i] = 0.5 * math.Sin(2*math.Pi*440.0*tSec)
	}

	buf := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	res, err := sd.Analyze(buf)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	t.Logf("IsTampered: %v, SpliceCount: %d, SplicePoints: %+v", res.IsTampered, res.SpliceCount, res.SplicePoints)

	if res.IsTampered {
		t.Errorf("continuous audio was falsely flagged as spliced")
	}
	if res.SpliceCount != 0 {
		t.Errorf("expected 0 splices, got %d", res.SpliceCount)
	}
}

func TestSplicingDetector_SplicedBoundary(t *testing.T) {
	sd := NewSplicingDetector()

	sampleRate := 48000
	numSamples := sampleRate * 2 // 2 seconds
	samples := make([]float64, numSamples)

	r := rand.New(rand.NewSource(42))

	// First 1 second: Low noise floor (-80 dBFS hiss)
	for i := 0; i < sampleRate; i++ {
		tSec := float64(i) / float64(sampleRate)
		samples[i] = 0.3*math.Sin(2*math.Pi*440.0*tSec) + (r.Float64()*2.0-1.0)*0.0001
	}

	// Splice at 1.0s: Second half recorded in different room with prominent hiss (-40 dBFS hiss)
	for i := sampleRate; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		samples[i] = 0.3*math.Sin(2*math.Pi*600.0*tSec) + (r.Float64()*2.0-1.0)*0.01
	}

	buf := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	res, err := sd.Analyze(buf)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	t.Logf("IsTampered: %v, SpliceCount: %d, SplicePoints: %+v", res.IsTampered, res.SpliceCount, res.SplicePoints)

	// Verification check against Definition of Done: Splice timestamp recorded
	if !res.IsTampered {
		t.Fatalf("DoD violated: audio splice was not detected")
	}
	if res.SpliceCount < 1 {
		t.Fatalf("expected at least 1 splice point, got %d", res.SpliceCount)
	}

	topSplice := res.SplicePoints[0]
	// Splice was at 1000ms (allow +-50ms frame resolution)
	if math.Abs(topSplice.TimestampMs-1000.0) > 60.0 {
		t.Errorf("expected splice near 1000ms, got %f ms", topSplice.TimestampMs)
	}
	t.Logf("Detected splice at %f ms with %f dB noise floor jump", topSplice.TimestampMs, topSplice.NoiseFloorDiffDb)

	jsonStr, err := res.ToJSON()
	if err != nil || len(jsonStr) == 0 {
		t.Fatalf("ToJSON failed: %v", err)
	}
}
