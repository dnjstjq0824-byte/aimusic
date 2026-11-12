// Package forensics provides inaudible ultrasonic spread-spectrum watermarking (18.5kHz-20.5kHz),
// direct sequence spread spectrum (DSSS) modulation, and robust copyright provenance tagging.
package forensics

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math"

	"avibe-platform/backend/internal/pkg/audio"
)

var (
	ErrPayloadTooLarge = errors.New("watermark payload exceeds maximum size (12 bytes)")
	ErrSampleRateLow   = errors.New("sample rate must be at least 44100 Hz for 19kHz ultrasonic watermarking")
)

const (
	WatermarkCarrierHz   = 19500.0 // Center frequency (18.5 kHz .. 20.5 kHz band)
	WatermarkBandwidthHz = 2000.0
	WatermarkChipRate    = 50           // Samples per chip
	WatermarkAmplitude   = 0.003        // -50.4 dBFS (inaudible beneath audio content)
	PlatformMagic        = 0x4156494245 // "AVIBE" in hex (5 bytes)
)

// WatermarkPayload holds structured copyright and licensing identifiers.
type WatermarkPayload struct {
	PlatformID uint64 `json:"platform_id"` // "AVIBE" magic
	TrackHash  uint32 `json:"track_hash"`  // 4-byte truncated SHA-256 of track UUID
	LicenseID  uint32 `json:"license_id"`  // 4-byte license ID
}

// EncodePayload formats the 13-byte raw watermark bit sequence with parity check.
func EncodePayload(trackID string, licenseID uint32) []byte {
	h := sha256.Sum256([]byte(trackID))
	trackHash := binary.BigEndian.Uint32(h[:4])

	buf := make([]byte, 13)
	// 5 bytes magic
	buf[0] = 0x41 // 'A'
	buf[1] = 0x56 // 'V'
	buf[2] = 0x49 // 'I'
	buf[3] = 0x42 // 'B'
	buf[4] = 0x45 // 'E'

	// 4 bytes track hash
	binary.BigEndian.PutUint32(buf[5:9], trackHash)

	// 4 bytes license ID
	binary.BigEndian.PutUint32(buf[9:13], licenseID)

	return buf
}

// 11-bit Barker spreading sequence: [+1, +1, +1, -1, -1, -1, +1, -1, -1, +1, -1]
var barker11 = []float64{1, 1, 1, -1, -1, -1, 1, -1, -1, 1, -1}

// WatermarkInjector embeds inaudible ultrasonic spread-spectrum watermarks into audio buffers.
type WatermarkInjector struct {
	carrierFreq float64
	amplitude   float64
}

// NewWatermarkInjector creates a watermark injector.
func NewWatermarkInjector() *WatermarkInjector {
	return &WatermarkInjector{
		carrierFreq: WatermarkCarrierHz,
		amplitude:   WatermarkAmplitude,
	}
}

// Inject embeds a payload into the PCM buffer across the 18.5kHz - 20.5kHz band.
func (wi *WatermarkInjector) Inject(input *audio.PCMBuffer, payload []byte) (*audio.PCMBuffer, error) {
	if input == nil {
		return nil, audio.ErrNilPCMBuffer
	}
	if input.NumSamples == 0 || input.Channels == 0 || len(input.Samples) == 0 {
		return nil, audio.ErrEmptyPCMBuffer
	}
	if input.SampleRate < 44100 {
		return nil, ErrSampleRateLow
	}

	numSamples := input.NumSamples
	sampleRate := input.SampleRate
	channels := input.Channels

	// Step 1: Add (7, 4) Hamming error correction parity to each nibble (4 bits -> 7 bits)
	encodedBits := addHammingFEC(payload)

	// Step 2: DSSS Modulation onto ultrasonic carrier
	numBits := len(encodedBits)
	chipsPerBit := len(barker11)
	samplesPerBit := chipsPerBit * WatermarkChipRate
	totalWatermarkSamples := numBits * samplesPerBit

	// Generate watermark signal
	wmSignal := make([]float64, numSamples)
	omega := 2.0 * math.Pi * wi.carrierFreq / float64(sampleRate)

	for i := 0; i < numSamples; i++ {
		// Wrap watermark repeatedly across the entire track
		wmIdx := i % totalWatermarkSamples
		bitIdx := wmIdx / samplesPerBit
		chipIdx := (wmIdx % samplesPerBit) / WatermarkChipRate

		bitVal := encodedBits[bitIdx]
		// Bit 1 -> +1, Bit 0 -> -1
		bipolar := 1.0
		if bitVal == 0 {
			bipolar = -1.0
		}

		chipBipolar := barker11[chipIdx]
		dsssSample := bipolar * chipBipolar * wi.amplitude

		// Modulate with ultrasonic carrier
		carrier := math.Sin(omega * float64(i))
		wmSignal[i] = dsssSample * carrier
	}

	// Step 3: Mix watermark into all channels with headroom limit
	outSamples := make([][]float64, channels)
	for ch := 0; ch < channels; ch++ {
		dst := make([]float64, numSamples)
		src := input.Samples[ch]
		for i := 0; i < numSamples; i++ {
			sum := src[i] + wmSignal[i]
			// Limit to [-1.0, 1.0]
			if sum > 0.999 {
				sum = 0.999
			} else if sum < -0.999 {
				sum = -0.999
			}
			dst[i] = sum
		}
		outSamples[ch] = dst
	}

	return &audio.PCMBuffer{
		Channels:   channels,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    outSamples,
	}, nil
}

// addHammingFEC encodes each byte into two 7-bit Hamming(7,4) code blocks + 1 parity bit = 16 bits per byte.
func addHammingFEC(payload []byte) []int {
	bits := make([]int, 0, len(payload)*16)
	for _, b := range payload {
		nibbleHigh := (b >> 4) & 0x0F
		nibbleLow := b & 0x0F

		bits = append(bits, encodeHamming7_4(nibbleHigh)...)
		bits = append(bits, encodeHamming7_4(nibbleLow)...)
	}
	return bits
}

// encodeHamming7_4 encodes 4 data bits (d1,d2,d3,d4) into 8 bits: (p1, p2, d1, p3, d2, d3, d4, overallParity).
func encodeHamming7_4(d byte) []int {
	d1 := int((d >> 3) & 1)
	d2 := int((d >> 2) & 1)
	d3 := int((d >> 1) & 1)
	d4 := int(d & 1)

	p1 := d1 ^ d2 ^ d4
	p2 := d1 ^ d3 ^ d4
	p3 := d2 ^ d3 ^ d4

	overallParity := p1 ^ p2 ^ d1 ^ p3 ^ d2 ^ d3 ^ d4

	return []int{p1, p2, d1, p3, d2, d3, d4, overallParity}
}
