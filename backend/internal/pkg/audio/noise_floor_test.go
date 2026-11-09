package audio

import (
	"math"
	"testing"
)

func TestNoiseFloor_CleanStudioAudio(t *testing.T) {
	analyzer := NewNoiseFloorAnalyzer(DefaultNoiseFloorConfig())

	sampleRate := 48000
	numSamples := sampleRate * 2 // 2 seconds

	samples := make([]float64, numSamples)
	// First 0.5s has ultra low noise (-80 dBFS -> amplitude ~ 0.0001)
	quietSamples := sampleRate / 2
	for i := 0; i < quietSamples; i++ {
		samples[i] = 0.00005 * math.Sin(2*math.Pi*1000*float64(i)/float64(sampleRate))
	}
	// Remainder has music (-12 dBFS)
	for i := quietSamples; i < numSamples; i++ {
		samples[i] = 0.25 * math.Sin(2*math.Pi*440*float64(i)/float64(sampleRate))
	}

	buf := &PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	res, err := analyzer.Analyze(buf)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if !res.PassesQualityGate {
		t.Errorf("expected passes quality gate (<= -70dBFS), got floor: %f dBFS", res.NoiseFloorDbFS)
	}
	if res.NoiseFloorDbFS > -70.0 {
		t.Errorf("noise floor %f dBFS exceeds -70dBFS", res.NoiseFloorDbFS)
	}
}

func TestNoiseFloor_NoisyAudioRejection(t *testing.T) {
	analyzer := NewNoiseFloorAnalyzer(DefaultNoiseFloorConfig())

	sampleRate := 48000
	numSamples := sampleRate * 2

	// Entire track has high hiss (-50 dBFS -> amplitude ~ 0.00316)
	samples := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		samples[i] = 0.004 * math.Sin(2*math.Pi*3000*float64(i)/float64(sampleRate))
	}

	buf := &PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	res, err := analyzer.Analyze(buf)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if res.PassesQualityGate {
		t.Errorf("noisy audio should fail quality filter (< -70dBFS), but passed: %f", res.NoiseFloorDbFS)
	}
}

func TestNoiseFloor_60HzMainsHum(t *testing.T) {
	analyzer := NewNoiseFloorAnalyzer(DefaultNoiseFloorConfig())

	sampleRate := 48000
	numSamples := sampleRate * 2

	// Quiet silence (-90 dBFS) + prominent 60Hz hum (-60 dBFS -> amp 0.001)
	samples := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		hum := 0.001 * math.Sin(2*math.Pi*60.0*float64(i)/float64(sampleRate))
		samples[i] = hum
	}

	buf := &PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	res, err := analyzer.Analyze(buf)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if !res.Has60HzHum {
		t.Errorf("expected 60Hz hum detection, but Has60HzHum was false")
	}
}
