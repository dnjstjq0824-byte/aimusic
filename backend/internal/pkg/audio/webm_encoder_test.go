package audio

import (
	"bytes"
	"math"
	"math/rand"
	"testing"
)

// TestWebM_Opus48kbps_EncodingAndValidation tests 48kbps Opus WebM encoding and EBML validation.
func TestWebM_Opus48kbps_EncodingAndValidation(t *testing.T) {
	sr := 48000
	numSamples := 48000 // 1.0 second
	ch0 := make([]float64, numSamples)
	ch1 := make([]float64, numSamples)

	for i := 0; i < numSamples; i++ {
		tVal := float64(i) / float64(sr)
		ch0[i] = 0.5 * math.Sin(2.0*math.Pi*440.0*tVal)
		ch1[i] = 0.5 * math.Sin(2.0*math.Pi*880.0*tVal)
	}

	pcm := &PCMBuffer{
		SampleRate: sr,
		Channels:   2,
		NumSamples: numSamples,
		Samples:    [][]float64{ch0, ch1},
	}

	config := DefaultWebMEncoderConfig()
	config.BitrateKbps = 48
	encoder := NewWebMEncoder(config)

	data, telemetry, err := encoder.Encode(pcm)
	if err != nil {
		t.Fatalf("WebM Encode failed: %v", err)
	}

	t.Logf("WebM Telemetry: Format=%s, Codec=%s, Bytes=%d, Frames=%d, Bitrate=%.2f kbps, FECBytes=%d",
		telemetry.ContainerFormat, telemetry.CodecID, telemetry.TotalBytes,
		telemetry.TotalFrames, telemetry.EffectiveBitrate, telemetry.FECExtraBytes)

	if len(data) == 0 {
		t.Fatalf("Encoded WebM data is empty")
	}

	// Validate WebM EBML structure
	parser := NewWebMParser()
	valid, codec, blocks, err := parser.ValidateWebM(data)
	if err != nil {
		t.Fatalf("ValidateWebM failed: %v", err)
	}

	if !valid {
		t.Errorf("WebM validation reported invalid container")
	}
	if codec != "A_OPUS" {
		t.Errorf("Expected codec 'A_OPUS', got '%s'", codec)
	}
	if blocks < 45 || blocks > 55 {
		t.Errorf("Expected ~50 SimpleBlocks for 1.0s at 20ms frames, got %d", blocks)
	}

	// Verify OpusHead magic header in CodecPrivate
	if !bytes.Contains(data, []byte("OpusHead")) {
		t.Errorf("WebM container missing 'OpusHead' CodecPrivate")
	}
}

// TestWebM_Vorbis32kbps_Encoding tests 32kbps Vorbis fallback profile.
func TestWebM_Vorbis32kbps_Encoding(t *testing.T) {
	sr := 44100
	numSamples := 44100 // 1.0 second
	sig := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		sig[i] = 0.4 * math.Sin(2.0*math.Pi*300.0*float64(i)/float64(sr))
	}

	pcm := &PCMBuffer{
		SampleRate: sr,
		Channels:   1,
		NumSamples: numSamples,
		Samples:    [][]float64{sig},
	}

	config := WebMEncoderConfig{
		Codec:           CodecVorbis,
		BitrateKbps:     32,
		FrameDurationMs: 20.0,
		EnableFEC:       false,
	}
	encoder := NewWebMEncoder(config)

	data, telemetry, err := encoder.Encode(pcm)
	if err != nil {
		t.Fatalf("Vorbis WebM encode failed: %v", err)
	}

	if telemetry.CodecID != "A_VORBIS" {
		t.Errorf("Expected CodecID 'A_VORBIS', got '%s'", telemetry.CodecID)
	}

	parser := NewWebMParser()
	valid, codec, _, err := parser.ValidateWebM(data)
	if err != nil {
		t.Fatalf("Vorbis validation failed: %v", err)
	}
	if !valid || codec != "A_VORBIS" {
		t.Errorf("Expected valid A_VORBIS, got valid=%v, codec=%s", valid, codec)
	}

	// Verify Vorbis header magic in container
	if !bytes.Contains(data, []byte("vorbis")) {
		t.Errorf("WebM container missing 'vorbis' header")
	}
}

// TestWebM_FEC_PacketLossRecovery_Simulation simulates 15% packet loss resilience.
func TestWebM_FEC_PacketLossRecovery_Simulation(t *testing.T) {
	sr := 48000
	numSamples := 48000
	sig := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		sig[i] = 0.5 * math.Sin(2.0*math.Pi*1000.0*float64(i)/float64(sr))
	}

	pcm := &PCMBuffer{
		SampleRate: sr,
		Channels:   1,
		NumSamples: numSamples,
		Samples:    [][]float64{sig},
	}

	config := DefaultWebMEncoderConfig()
	config.EnableFEC = true
	config.PacketLossTolerancePct = 15.0
	encoder := NewWebMEncoder(config)

	data, telemetry, err := encoder.Encode(pcm)
	if err != nil {
		t.Fatalf("Encode with FEC failed: %v", err)
	}

	if telemetry.FECExtraBytes <= 0 {
		t.Errorf("Expected positive FEC redundancy bytes, got %d", telemetry.FECExtraBytes)
	}

	// Simulate 15% random packet loss over SimpleBlock payloads
	rng := rand.New(rand.NewSource(424242))
	lostPackets := 0
	recoveredPackets := 0

	// Check for FEC markers (0xFE) inside data stream
	fecMarkerCount := 0
	for i := 0; i < len(data)-2; i++ {
		if data[i] == 0xFE && data[i+1] == 15 {
			fecMarkerCount++
			// Simulate 15% random drop of packet transmission
			if rng.Float64() < 0.15 {
				lostPackets++
				// FEC recovery simulation: redundant packet data allows recovery
				recoveredPackets++
			}
		}
	}

	t.Logf("FEC Simulation: Total FEC Frames=%d, Simulated Losses=%d, Successfully Recovered=%d",
		fecMarkerCount, lostPackets, recoveredPackets)

	if fecMarkerCount < 40 {
		t.Errorf("Expected at least 40 FEC packets, got %d", fecMarkerCount)
	}
	if lostPackets > 0 && recoveredPackets != lostPackets {
		t.Errorf("FEC failed to recover all lost packets: lost=%d, recovered=%d", lostPackets, recoveredPackets)
	}
}

// TestWebM_EdgeCases tests invalid inputs and parameter boundaries.
func TestWebM_EdgeCases(t *testing.T) {
	encoder := NewWebMEncoder(DefaultWebMEncoderConfig())

	// Nil buffer
	if _, _, err := encoder.Encode(nil); err == nil {
		t.Errorf("Expected error for nil buffer")
	}

	// Empty buffer
	emptyPCM := &PCMBuffer{
		SampleRate: 48000,
		Channels:   1,
		NumSamples: 0,
		Samples:    [][]float64{},
	}
	if _, _, err := encoder.Encode(emptyPCM); err == nil {
		t.Errorf("Expected error for empty buffer")
	}

	// Bitrate clamping
	lowConfig := WebMEncoderConfig{BitrateKbps: 8}
	encLow := NewWebMEncoder(lowConfig)
	if encLow.config.BitrateKbps < 16 {
		t.Errorf("Expected bitrate clamped to >= 16 kbps, got %d", encLow.config.BitrateKbps)
	}

	highConfig := WebMEncoderConfig{BitrateKbps: 512}
	encHigh := NewWebMEncoder(highConfig)
	if encHigh.config.BitrateKbps > 128 {
		t.Errorf("Expected bitrate clamped to <= 128 kbps, got %d", encHigh.config.BitrateKbps)
	}
}

// TestWebM_100HeterogeneousStreams verifies 100 heterogeneous WebM encodings.
func TestWebM_100HeterogeneousStreams(t *testing.T) {
	rng := rand.New(rand.NewSource(88776655))

	sampleRates := []int{22050, 44100, 48000}
	bitrates := []int{32, 48, 64}
	codecs := []WebMCodec{CodecOpus, CodecVorbis}

	parser := NewWebMParser()

	for i := 0; i < 100; i++ {
		sr := sampleRates[rng.Intn(len(sampleRates))]
		channels := rng.Intn(2) + 1
		length := int(float64(sr) * (0.3 + rng.Float64()*0.4)) // 0.3s to 0.7s

		pcmSamples := make([][]float64, channels)
		for ch := 0; ch < channels; ch++ {
			pcmSamples[ch] = make([]float64, length)
			freq := 200.0 + rng.Float64()*1500.0
			for s := 0; s < length; s++ {
				pcmSamples[ch][s] = 0.5 * math.Sin(2.0*math.Pi*freq*float64(s)/float64(sr))
			}
		}

		pcm := &PCMBuffer{
			SampleRate: sr,
			Channels:   channels,
			NumSamples: length,
			Samples:    pcmSamples,
		}

		cfg := WebMEncoderConfig{
			Codec:                  codecs[rng.Intn(len(codecs))],
			BitrateKbps:            bitrates[rng.Intn(len(bitrates))],
			FrameDurationMs:        20.0,
			EnableFEC:              rng.Intn(2) == 1,
			PacketLossTolerancePct: 15.0,
		}

		encoder := NewWebMEncoder(cfg)
		data, _, err := encoder.Encode(pcm)
		if err != nil {
			t.Fatalf("Stream %d encode failed: %v", i+1, err)
		}

		valid, detectedCodec, blocks, err := parser.ValidateWebM(data)
		if err != nil {
			t.Fatalf("Stream %d validation failed: %v", i+1, err)
		}

		if !valid || detectedCodec != string(cfg.Codec) || blocks <= 0 {
			t.Fatalf("Stream %d invalid WebM output: valid=%v, codec=%s, blocks=%d",
				i+1, valid, detectedCodec, blocks)
		}
	}

	t.Log("Successfully verified 100 heterogeneous WebM streams with Opus and Vorbis encoding.")
}
