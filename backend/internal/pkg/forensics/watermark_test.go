package forensics

import (
	"bytes"
	"math"
	"math/rand"
	"testing"

	"avibe-platform/backend/internal/pkg/audio"
)

func TestWatermark_HammingFEC_BitCorrection(t *testing.T) {
	// Test all possible 4-bit nibbles (0..15)
	for originalNibble := byte(0); originalNibble < 16; originalNibble++ {
		encoded := encodeHamming7_4(originalNibble)

		// Test corruption on each of the 7 positions (1 bit error per block)
		for errorBit := 0; errorBit < 7; errorBit++ {
			corrupted := make([]int, len(encoded))
			copy(corrupted, encoded)
			corrupted[errorBit] ^= 1 // Invert bit

			decodedNibble, errCount := decodeHamming7_4(corrupted)
			if decodedNibble != originalNibble {
				t.Fatalf("Hamming FEC failed to correct bit %d for nibble %d: got %d", errorBit, originalNibble, decodedNibble)
			}
			if errCount != 1 {
				t.Fatalf("expected errCount == 1, got %d", errCount)
			}
		}
	}
}

func TestWatermark_InjectAndExtractRoundTrip(t *testing.T) {
	sampleRate := 48000
	// 208 bits * 550 samples per bit = 114,400 samples (~2.4s) -> use 3 seconds (144,000 samples)
	numSamples := sampleRate * 3
	samples := make([]float64, numSamples)

	// Background musical audio: 440 Hz + 1000 Hz tone
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		samples[i] = 0.4*math.Sin(2*math.Pi*440.0*tSec) + 0.2*math.Sin(2*math.Pi*1000.0*tSec)
	}

	buf := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	trackID := "track-uuid-9999"
	licenseID := uint32(123456)
	payload := EncodePayload(trackID, licenseID)

	injector := NewWatermarkInjector()
	watermarkedBuf, err := injector.Inject(buf, payload)
	if err != nil {
		t.Fatalf("Inject failed: %v", err)
	}

	// Verify audio waveform was not distorted (remains close to original)
	for i := 0; i < 1000; i++ {
		diff := math.Abs(watermarkedBuf.Samples[0][i] - buf.Samples[0][i])
		if diff > 0.01 { // Max watermark injection amplitude is 0.003
			t.Fatalf("audio distortion exceeded tolerance: diff=%f", diff)
		}
	}

	// Extract
	extractor := NewWatermarkExtractor()
	extracted, err := extractor.Extract(watermarkedBuf)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if extracted.PlatformMagic != "AVIBE" {
		t.Errorf("expected platform magic 'AVIBE', got '%s'", extracted.PlatformMagic)
	}
	if extracted.LicenseID != licenseID {
		t.Errorf("expected license ID %d, got %d", licenseID, extracted.LicenseID)
	}
	if !bytes.Equal(extracted.RawBytes, payload) {
		t.Errorf("extracted raw bytes mismatch")
	}
}

func TestWatermark_RobustnessToNoise(t *testing.T) {
	sampleRate := 48000
	numSamples := sampleRate * 3
	samples := make([]float64, numSamples)

	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		samples[i] = 0.3 * math.Sin(2*math.Pi*800.0*tSec)
	}

	buf := &audio.PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	trackID := "track-noise-resilience"
	licenseID := uint32(777777)
	payload := EncodePayload(trackID, licenseID)

	injector := NewWatermarkInjector()
	watermarkedBuf, _ := injector.Inject(buf, payload)

	// Add pseudo-random compression noise (simulating MP3 / AAC lossy compression)
	r := rand.New(rand.NewSource(101))
	for i := 0; i < numSamples; i++ {
		// Low-level white noise (-60dBFS)
		noise := (r.Float64()*2.0 - 1.0) * 0.0005
		watermarkedBuf.Samples[0][i] += noise
	}

	extractor := NewWatermarkExtractor()
	extracted, err := extractor.Extract(watermarkedBuf)
	if err != nil {
		t.Fatalf("Extract failed after lossy noise injection: %v", err)
	}

	if extracted.PlatformMagic != "AVIBE" {
		t.Errorf("magic mismatch under noise: %s", extracted.PlatformMagic)
	}
	if extracted.LicenseID != licenseID {
		t.Errorf("license ID mismatch under noise: expected %d, got %d", licenseID, extracted.LicenseID)
	}
	t.Logf("Watermark survived simulated compression noise with %d bit errors corrected!", extracted.BitErrorCount)
}
