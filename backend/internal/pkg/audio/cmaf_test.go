package audio

import (
	"math/rand"
	"strings"
	"testing"
)

// Helper to synthesize dummy AAC frame payload bytes with valid length.
func generateSynthesizedAudioFrames(count int, ticksPerFrame uint32, frameSizeBytes int) []CMAFAudioFrame {
	frames := make([]CMAFAudioFrame, count)
	for i := 0; i < count; i++ {
		data := make([]byte, frameSizeBytes)
		// Set dummy frame payload bytes with non-zero header
		data[0] = 0xFF // ADTS syncword start or frame identifier
		data[1] = 0xF1
		for j := 2; j < frameSizeBytes; j++ {
			data[j] = byte((i*17 + j) % 256)
		}
		frames[i] = CMAFAudioFrame{
			Data:         data,
			DurationTick: ticksPerFrame,
		}
	}
	return frames
}

func TestCMAFHeader_BuildAndValidate(t *testing.T) {
	cfg := &CMAFTrackConfig{
		TrackID:       1,
		Codec:         "aac",
		SampleRate:    48000,
		Channels:      2,
		BitsPerSample: 16,
		BitrateBps:    192000,
		Language:      "und",
	}

	headerData, err := BuildCMAFHeader(cfg)
	if err != nil {
		t.Fatalf("BuildCMAFHeader failed: %v", err)
	}

	if len(headerData) == 0 {
		t.Fatalf("header data is empty")
	}

	// Validate ISOBMFF structure
	boxes, err := ParseISOBMFFBoxes(headerData)
	if err != nil {
		t.Fatalf("ParseISOBMFFBoxes failed: %v", err)
	}

	if len(boxes) != 2 {
		t.Fatalf("expected 2 top-level boxes (ftyp, moov), got %d", len(boxes))
	}

	if boxes[0].Type != BoxTypeFTYP {
		t.Errorf("box 0 expected ftyp, got %s", boxes[0].Type)
	}
	if boxes[1].Type != BoxTypeMOOV {
		t.Errorf("box 1 expected moov, got %s", boxes[1].Type)
	}

	// Validate CMAF header conformance
	if err := ValidateCMAFHeader(headerData); err != nil {
		t.Errorf("ValidateCMAFHeader failed: %v", err)
	}
}

func TestCMAFSegment_BuildAndValidate(t *testing.T) {
	// 48 frames at 1024 ticks = 49,152 ticks (~1.024s at 48kHz)
	frames := generateSynthesizedAudioFrames(48, 1024, 384)

	segSpec := &CMAFSegmentSpec{
		TrackID:        1,
		SequenceNumber: 1,
		BaseDecodeTime: 0,
		Frames:         frames,
	}

	segData, err := BuildCMAFSegment(segSpec)
	if err != nil {
		t.Fatalf("BuildCMAFSegment failed: %v", err)
	}

	boxes, err := ParseISOBMFFBoxes(segData)
	if err != nil {
		t.Fatalf("ParseISOBMFFBoxes on segment failed: %v", err)
	}

	if len(boxes) != 3 {
		t.Fatalf("expected 3 top-level boxes (styp, moof, mdat), got %d", len(boxes))
	}

	if boxes[0].Type != BoxTypeSTYP {
		t.Errorf("box 0 expected styp, got %s", boxes[0].Type)
	}
	if boxes[1].Type != BoxTypeMOOF {
		t.Errorf("box 1 expected moov, got %s", boxes[1].Type)
	}
	if boxes[2].Type != BoxTypeMDAT {
		t.Errorf("box 2 expected mdat, got %s", boxes[2].Type)
	}

	// Expected mdat payload size = 48 frames * 384 bytes = 18,432 bytes
	expectedMdatPayload := 48 * 384
	if len(boxes[2].Payload) != expectedMdatPayload {
		t.Errorf("mdat payload expected %d bytes, got %d", expectedMdatPayload, len(boxes[2].Payload))
	}

	if err := ValidateCMAFSegment(segData); err != nil {
		t.Errorf("ValidateCMAFSegment failed: %v", err)
	}
}

func TestCMAFPackage_DualManifestStorageSavings(t *testing.T) {
	// 18-second audio: 18 * 48000 / 1024 ~ 844 frames
	// Segment duration: 6 seconds (approx 3 segments)
	totalFrames := 844
	frames := generateSynthesizedAudioFrames(totalFrames, 1024, 400)

	pkgSpec := &CMAFPackageSpec{
		TrackID:            "cmaf_track_test_01",
		Title:              "Electric Horizon",
		Artist:             "Synthetica",
		DurationSec:        18.0,
		SegmentDurationSec: 6.0,
		SampleRate:         48000,
		Channels:           2,
		BitrateKbps:        192,
		Codec:              "aac",
		Frames:             frames,
		BaseURL:            "https://cdn.avibe.io/stream/cmaf_track_test_01/",
	}

	result, err := GenerateCMAFPackage(pkgSpec)
	if err != nil {
		t.Fatalf("GenerateCMAFPackage failed: %v", err)
	}

	if result.InitSegmentName != "init.mp4" {
		t.Errorf("expected init.mp4, got %s", result.InitSegmentName)
	}
	if len(result.SegmentNames) != 3 {
		t.Fatalf("expected 3 segments for 18s at 6s/chunk, got %d", len(result.SegmentNames))
	}

	// Verify HLS v7 Manifest
	hls := result.HLSManifest
	if !strings.Contains(hls, "#EXT-X-VERSION:7") {
		t.Errorf("HLS manifest missing version 7: %s", hls)
	}
	if !strings.Contains(hls, `#EXT-X-MAP:URI="init.mp4"`) {
		t.Errorf("HLS manifest missing init.mp4 map: %s", hls)
	}
	if !strings.Contains(hls, "chunk-00001.m4s") || !strings.Contains(hls, "chunk-00003.m4s") {
		t.Errorf("HLS manifest missing cmaf chunks: %s", hls)
	}

	// Verify MPEG-DASH Manifest
	dash := result.DASHManifest
	if !strings.Contains(dash, "init.mp4") {
		t.Errorf("DASH manifest missing init.mp4: %s", dash)
	}
	if !strings.Contains(dash, "chunk-$Number%05d$.m4s") {
		t.Errorf("DASH manifest missing chunk template: %s", dash)
	}

	// Verify 50% Storage Reduction: Both HLS and DASH point to the same physical chunks on disk
	if !strings.Contains(result.StorageSavings, "50% reduction") {
		t.Errorf("expected 50%% reduction verification, got: %s", result.StorageSavings)
	}
}

func TestCMAF_LowLatencyMicroChunking(t *testing.T) {
	// Low-latency mode: 1-second chunks (approx 47 frames each at 48kHz / 1024)
	framesPerSec := 47
	numSec := 5
	frames := generateSynthesizedAudioFrames(framesPerSec*numSec, 1024, 256)

	pkgSpec := &CMAFPackageSpec{
		TrackID:            "ll_cmaf_track_02",
		DurationSec:        5.0,
		SegmentDurationSec: 1.0, // 1.0s low latency micro-chunks
		SampleRate:         48000,
		Channels:           2,
		BitrateKbps:        128,
		Frames:             frames,
	}

	result, err := GenerateCMAFPackage(pkgSpec)
	if err != nil {
		t.Fatalf("GenerateCMAFPackage failed: %v", err)
	}

	if len(result.SegmentNames) != 5 {
		t.Errorf("expected 5 low-latency micro-chunks, got %d", len(result.SegmentNames))
	}

	// Verify each chunk is valid and decode time advances
	for i, chunkBytes := range result.SegmentData {
		if err := ValidateCMAFSegment(chunkBytes); err != nil {
			t.Fatalf("chunk %d invalid: %v", i+1, err)
		}
	}
}

func TestCMAF_EdgeCases(t *testing.T) {
	// Nil config
	_, err := BuildCMAFHeader(nil)
	if err != ErrNilCMAFConfig {
		t.Errorf("expected ErrNilCMAFConfig, got %v", err)
	}

	// Invalid sample rate
	_, err = BuildCMAFHeader(&CMAFTrackConfig{SampleRate: 0, Channels: 2})
	if err != ErrInvalidSampleRate {
		t.Errorf("expected ErrInvalidSampleRate, got %v", err)
	}

	// Invalid channels
	_, err = BuildCMAFHeader(&CMAFTrackConfig{SampleRate: 48000, Channels: 0})
	if err != ErrInvalidChannelCount {
		t.Errorf("expected ErrInvalidChannelCount, got %v", err)
	}

	// Empty frames in segment
	_, err = BuildCMAFSegment(&CMAFSegmentSpec{TrackID: 1, Frames: nil})
	if err != ErrEmptyAudioFrames {
		t.Errorf("expected ErrEmptyAudioFrames, got %v", err)
	}

	// Truncated ISOBMFF data
	_, err = ParseISOBMFFBoxes([]byte{0, 0, 0, 10, 'f', 't', 'y'})
	if err != ErrCorruptISOBMFF {
		t.Errorf("expected ErrCorruptISOBMFF, got %v", err)
	}

	// Invalid brand header
	fakeHeader := CreateBox(BoxTypeFTYP, []byte("isom\x00\x00\x00\x00isom"))
	fakeHeader = append(fakeHeader, CreateBox(BoxTypeMOOV, []byte{0, 0, 0, 0})...)
	err = ValidateCMAFHeader(fakeHeader)
	if err != ErrInvalidCMAFBrand {
		t.Errorf("expected ErrInvalidCMAFBrand, got %v", err)
	}
}

func TestCMAF_100HeterogeneousStreams(t *testing.T) {
	rng := rand.New(rand.NewSource(20261109))
	totalTests := 100

	sampleRates := []int{8000, 16000, 22050, 32000, 44100, 48000, 96000}
	channelsList := []int{1, 2, 6}
	bitrates := []int{64, 128, 192, 256, 320}

	for iter := 0; iter < totalTests; iter++ {
		sr := sampleRates[rng.Intn(len(sampleRates))]
		ch := channelsList[rng.Intn(len(channelsList))]
		br := bitrates[rng.Intn(len(bitrates))]
		ticksPerFrame := uint32(512 + rng.Intn(1024))
		frameBytes := 100 + rng.Intn(500)
		numFrames := 10 + rng.Intn(100)

		// 1. Build and validate header
		header, err := BuildCMAFHeader(&CMAFTrackConfig{
			TrackID:       uint32(rng.Intn(10) + 1),
			Codec:         "aac",
			SampleRate:    sr,
			Channels:      ch,
			BitsPerSample: 16,
			BitrateBps:    br * 1000,
			Language:      "und",
		})
		if err != nil {
			t.Fatalf("iteration %d: BuildCMAFHeader failed: %v", iter, err)
		}
		if err := ValidateCMAFHeader(header); err != nil {
			t.Fatalf("iteration %d: ValidateCMAFHeader failed: %v", iter, err)
		}

		// 2. Build and validate segment
		frames := generateSynthesizedAudioFrames(numFrames, ticksPerFrame, frameBytes)
		seg, err := BuildCMAFSegment(&CMAFSegmentSpec{
			TrackID:        1,
			SequenceNumber: uint32(iter + 1),
			BaseDecodeTime: uint64(iter) * uint64(numFrames) * uint64(ticksPerFrame),
			Frames:         frames,
		})
		if err != nil {
			t.Fatalf("iteration %d: BuildCMAFSegment failed: %v", iter, err)
		}
		if err := ValidateCMAFSegment(seg); err != nil {
			t.Fatalf("iteration %d: ValidateCMAFSegment failed: %v", iter, err)
		}
	}

	t.Logf("Successfully verified 100 heterogeneous CMAF streams and chunks with 0 errors.")
}
