package audio

import (
	"math"
	"testing"
)

func TestSubBassRumbleFilter_EmptyBuffer(t *testing.T) {
	rf := NewSubBassRumbleFilter(DefaultRumbleFilterConfig())
	_, err := rf.Process(nil)
	if err != ErrNilPCMBuffer {
		t.Fatalf("expected ErrNilPCMBuffer, got %v", err)
	}
}

func TestSubBassRumbleFilter_10HzRumbleAttenuation(t *testing.T) {
	rf := NewSubBassRumbleFilter(DefaultRumbleFilterConfig())

	sampleRate := 48000
	numSamples := sampleRate * 2 // 2 seconds
	samples := make([]float64, numSamples)

	// 10 Hz subsonic rumble (amplitude 0.5)
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		samples[i] = 0.5 * math.Sin(2*math.Pi*10.0*tSec)
	}

	buf := &PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	res, err := rf.Process(buf)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if !res.SubsonicEnergyRemoved {
		t.Errorf("expected SubsonicEnergyRemoved to be true, got attenuation: %f dB", res.RumbleAttenuationDb)
	}
	if res.RumbleAttenuationDb < 12.0 {
		t.Errorf("expected at least 12dB attenuation at 10Hz, got %f dB", res.RumbleAttenuationDb)
	}

	jsonStr, err := res.ToJSON()
	if err != nil || len(jsonStr) == 0 {
		t.Fatalf("ToJSON failed: %v", err)
	}
}

func TestSubBassRumbleFilter_MusicalBassPreservation(t *testing.T) {
	rf := NewSubBassRumbleFilter(DefaultRumbleFilterConfig())

	sampleRate := 48000
	numSamples := sampleRate * 2
	samples := make([]float64, numSamples)

	// 80 Hz musical bass tone (amplitude 0.5)
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		samples[i] = 0.5 * math.Sin(2*math.Pi*80.0*tSec)
	}

	buf := &PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	res, err := rf.Process(buf)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Calculate peak of filtered buffer
	outPeak := calcPeak(res.ProcessedBuffer.Samples[0])
	// Peak should be very close to 0.5 (within 5% / < 0.5dB drop)
	if outPeak < 0.45 {
		t.Errorf("80Hz musical bass was overly attenuated: peak %f (expected ~0.50)", outPeak)
	}
}
