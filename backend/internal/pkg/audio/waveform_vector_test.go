package audio

import (
	"math"
	"testing"
)

func TestPeakVectorExtractor_EmptyBuffer(t *testing.T) {
	extractor := NewPeakVectorExtractor(DefaultPeakVectorConfig())

	_, err := extractor.Extract(nil)
	if err != ErrNilPCMBuffer {
		t.Fatalf("expected ErrNilPCMBuffer, got %v", err)
	}

	buf := &PCMBuffer{
		Channels:   1,
		SampleRate: 48000,
		NumSamples: 0,
		Samples:    [][]float64{},
	}
	_, err = extractor.Extract(buf)
	if err != ErrEmptyPCMBuffer {
		t.Fatalf("expected ErrEmptyPCMBuffer, got %v", err)
	}
}

func TestPeakVectorExtractor_1000Points(t *testing.T) {
	extractor := NewPeakVectorExtractor(DefaultPeakVectorConfig())

	// 5 seconds at 48kHz = 240,000 samples
	sampleRate := 48000
	duration := 5.0
	numSamples := int(duration * float64(sampleRate))

	samples := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		// 440 Hz tone modulated with slow envelope
		tSec := float64(i) / float64(sampleRate)
		env := 0.5 * (1.0 + math.Sin(2*math.Pi*0.5*tSec)) // 0.0 to 1.0 envelope
		samples[i] = 0.8 * env * math.Sin(2*math.Pi*440.0*tSec)
	}

	buf := &PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	vec, err := extractor.Extract(buf)
	if err != nil {
		t.Fatalf("waveform extraction failed: %v", err)
	}

	if vec.NumPoints != 1000 {
		t.Fatalf("expected 1000 points, got %d", vec.NumPoints)
	}
	if len(vec.Peaks) != 1000 {
		t.Fatalf("expected 1000 peaks, got %d", len(vec.Peaks))
	}
	if len(vec.Points) != 1000 {
		t.Fatalf("expected 1000 detailed points, got %d", len(vec.Points))
	}

	// Verify Peak and RMS bounds
	if vec.GlobalPeak < 0.79 || vec.GlobalPeak > 0.81 {
		t.Errorf("expected global peak ~0.80, got %f", vec.GlobalPeak)
	}

	// Verify JSON serialization size limit (< 10KB / 10240 bytes)
	jsonData, err := vec.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}
	if len(jsonData) >= 10240 {
		t.Fatalf("JSON payload size %d bytes exceeded 10KB threshold", len(jsonData))
	}
	t.Logf("1000-point Waveform JSON size: %d bytes (limit: 10,240 bytes)", len(jsonData))

	// Verify Base64 binary decoding
	decodedPeaks, decodedRMSs, err := DecodePackedB64(vec.PackedB64)
	if err != nil {
		t.Fatalf("DecodePackedB64 failed: %v", err)
	}
	if len(decodedPeaks) != 1000 || len(decodedRMSs) != 1000 {
		t.Fatalf("expected 1000 decoded items, got peaks:%d, rmss:%d", len(decodedPeaks), len(decodedRMSs))
	}

	// Verify sample point fidelity between raw and quantized
	for i := 0; i < 1000; i += 100 {
		diff := math.Abs(vec.Peaks[i] - decodedPeaks[i])
		if diff > 0.02 {
			t.Errorf("point %d quantization error too large: %f vs %f", i, vec.Peaks[i], decodedPeaks[i])
		}
	}
}

func TestPeakVectorExtractor_StereoPanned(t *testing.T) {
	extractor := NewPeakVectorExtractor(DefaultPeakVectorConfig())

	n := 48000
	left := make([]float64, n)
	right := make([]float64, n)
	for i := 0; i < n; i++ {
		left[i] = 0.6
		right[i] = 0.4
	}

	buf := &PCMBuffer{
		Channels:   2,
		SampleRate: 48000,
		NumSamples: n,
		Samples:    [][]float64{left, right},
	}

	vec, err := extractor.Extract(buf)
	if err != nil {
		t.Fatalf("Extract stereo failed: %v", err)
	}

	// Mono mix of 0.6 and 0.4 is 0.5
	if math.Abs(vec.GlobalPeak-0.5) > 0.01 {
		t.Errorf("expected global peak 0.5 for mixed stereo, got %f", vec.GlobalPeak)
	}
}
