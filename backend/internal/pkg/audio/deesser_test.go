package audio

import (
	"math"
	"testing"
)

func TestDynamicDeEsser_EmptyBuffer(t *testing.T) {
	de := NewDynamicDeEsser(DefaultDeEsserConfig())
	_, err := de.Process(nil)
	if err != ErrNilPCMBuffer {
		t.Fatalf("expected ErrNilPCMBuffer, got %v", err)
	}
}

func TestDynamicDeEsser_HarshSibilanceReduction(t *testing.T) {
	de := NewDynamicDeEsser(DefaultDeEsserConfig())

	sampleRate := 48000
	numSamples := sampleRate * 1 // 1 second
	samples := make([]float64, numSamples)

	// Harsh 7000 Hz vocal sibilance at 0.5 amplitude (-6.0 dBFS)
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		samples[i] = 0.5 * math.Sin(2*math.Pi*7000.0*tSec)
	}

	buf := &PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	res, err := de.Process(buf)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if !res.TargetSuppressionMet {
		t.Errorf("expected TargetSuppressionMet (>= 4dB suppression), got max GR: %f dB", res.MaxReductionAppliedDb)
	}
	if res.MaxReductionAppliedDb < 4.0 {
		t.Errorf("expected at least 4.0 dB reduction on harsh sibilance, got %f dB", res.MaxReductionAppliedDb)
	}
	if res.SibilantEventsCount < 1 {
		t.Errorf("expected at least 1 sibilant event detected, got %d", res.SibilantEventsCount)
	}

	jsonStr, err := res.ToJSON()
	if err != nil || len(jsonStr) == 0 {
		t.Fatalf("ToJSON failed: %v", err)
	}
}

func TestDynamicDeEsser_NonSibilantPassThrough(t *testing.T) {
	de := NewDynamicDeEsser(DefaultDeEsserConfig())

	sampleRate := 48000
	numSamples := sampleRate * 1
	samples := make([]float64, numSamples)

	// 440 Hz vocal fundamental (outside 6k..8k sibilance band)
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		samples[i] = 0.5 * math.Sin(2*math.Pi*440.0*tSec)
	}

	buf := &PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	res, err := de.Process(buf)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Should not trigger sibilance compression
	if res.MaxReductionAppliedDb > 0.1 {
		t.Errorf("non-sibilant 440Hz tone was erroneously compressed: %f dB reduction", res.MaxReductionAppliedDb)
	}
}
