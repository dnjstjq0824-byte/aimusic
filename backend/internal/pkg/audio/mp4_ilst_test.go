package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
	"time"
)

func TestMP4Ilst_BuildAndInject_Roundtrip(t *testing.T) {
	canonicalMP4, err := BuildCanonicalTestMP4(1.5, 48000, 2)
	if err != nil {
		t.Fatalf("BuildCanonicalTestMP4 failed: %v", err)
	}

	meta := &MP4Metadata{
		Title:                 "Cybernetic Symphony No. 9",
		Artist:                "AVibe Neural Ensemble",
		Album:                 "Future Harmonics 2026",
		Composer:              "Maestro AI v4",
		Year:                  "2026",
		Copyright:             "© 2026 AVibe AI Music Inc. All rights reserved.",
		Comment:               "High-fidelity AI generated orchestral piece",
		Genre:                 "Cyber Classical",
		Description:           "Generated with AVibe real-time acoustic neural model",
		TrackNumber:           3,
		TotalTracks:           12,
		DiscNumber:            1,
		TotalDiscs:            2,
		AIArticle31Disclosure: DefaultArticle31Disclosure(),
		AIModelName:           "AVibe-Orchestral-v4.2",
		AIPrompt:              "epic symphonic crescendo with oboe, cello, and synth brass, 120 bpm, D minor",
		AISeed:                "9876543210",
		AIGenerator:           "AVibe Engine Core DSP",
		AIIsSynthetic:         true,
		AICreationTimestamp:   time.Now().UTC().Format(time.RFC3339),
		CustomFields: map[string]string{
			"AI_TRANSPARENCY_STANDARD": "KR-AI-ACT-ART31-COMPLIANT",
			"AI_PROMPT_VERSION":        "v2.1",
		},
	}

	injectedMP4, err := InjectMP4IlstMetadata(canonicalMP4, meta)
	if err != nil {
		t.Fatalf("InjectMP4IlstMetadata failed: %v", err)
	}

	if len(injectedMP4) <= len(canonicalMP4) {
		t.Fatalf("expected injected MP4 to expand with metadata, got %d <= %d", len(injectedMP4), len(canonicalMP4))
	}

	extracted, err := ExtractMP4IlstMetadata(injectedMP4)
	if err != nil {
		t.Fatalf("ExtractMP4IlstMetadata failed: %v", err)
	}

	if extracted.Title != meta.Title {
		t.Errorf("Title mismatch: expected %q, got %q", meta.Title, extracted.Title)
	}
	if extracted.Artist != meta.Artist {
		t.Errorf("Artist mismatch: expected %q, got %q", meta.Artist, extracted.Artist)
	}
	if extracted.Album != meta.Album {
		t.Errorf("Album mismatch: expected %q, got %q", meta.Album, extracted.Album)
	}
	if extracted.Composer != meta.Composer {
		t.Errorf("Composer mismatch: expected %q, got %q", meta.Composer, extracted.Composer)
	}
	if extracted.Year != meta.Year {
		t.Errorf("Year mismatch: expected %q, got %q", meta.Year, extracted.Year)
	}
	if extracted.Copyright != meta.Copyright {
		t.Errorf("Copyright mismatch: expected %q, got %q", meta.Copyright, extracted.Copyright)
	}
	if extracted.Comment != meta.Comment {
		t.Errorf("Comment mismatch: expected %q, got %q", meta.Comment, extracted.Comment)
	}
	if extracted.Genre != meta.Genre {
		t.Errorf("Genre mismatch: expected %q, got %q", meta.Genre, extracted.Genre)
	}
	if extracted.Description != meta.Description {
		t.Errorf("Description mismatch: expected %q, got %q", meta.Description, extracted.Description)
	}
	if extracted.TrackNumber != meta.TrackNumber || extracted.TotalTracks != meta.TotalTracks {
		t.Errorf("TrackNumber mismatch: expected %d/%d, got %d/%d", meta.TrackNumber, meta.TotalTracks, extracted.TrackNumber, extracted.TotalTracks)
	}
	if extracted.DiscNumber != meta.DiscNumber || extracted.TotalDiscs != meta.TotalDiscs {
		t.Errorf("DiscNumber mismatch: expected %d/%d, got %d/%d", meta.DiscNumber, meta.TotalDiscs, extracted.DiscNumber, extracted.TotalDiscs)
	}

	// Verify Article 31 AI transparency fields
	if extracted.AIArticle31Disclosure != meta.AIArticle31Disclosure {
		t.Errorf("AIArticle31Disclosure mismatch:\nexpected %q\ngot %q", meta.AIArticle31Disclosure, extracted.AIArticle31Disclosure)
	}
	if extracted.AIModelName != meta.AIModelName {
		t.Errorf("AIModelName mismatch: expected %q, got %q", meta.AIModelName, extracted.AIModelName)
	}
	if extracted.AIPrompt != meta.AIPrompt {
		t.Errorf("AIPrompt mismatch: expected %q, got %q", meta.AIPrompt, extracted.AIPrompt)
	}
	if extracted.AISeed != meta.AISeed {
		t.Errorf("AISeed mismatch: expected %q, got %q", meta.AISeed, extracted.AISeed)
	}
	if extracted.AIGenerator != meta.AIGenerator {
		t.Errorf("AIGenerator mismatch: expected %q, got %q", meta.AIGenerator, extracted.AIGenerator)
	}
	if !extracted.AIIsSynthetic {
		t.Errorf("AIIsSynthetic mismatch: expected true, got %v", extracted.AIIsSynthetic)
	}
	if extracted.CustomFields["AI_TRANSPARENCY_STANDARD"] != "KR-AI-ACT-ART31-COMPLIANT" {
		t.Errorf("Custom field AI_TRANSPARENCY_STANDARD mismatch: got %q", extracted.CustomFields["AI_TRANSPARENCY_STANDARD"])
	}
}

func TestMP4Ilst_ChunkOffsetIntegrity_AudioSeek(t *testing.T) {
	canonicalMP4, err := BuildCanonicalTestMP4(2.0, 48000, 2)
	if err != nil {
		t.Fatalf("BuildCanonicalTestMP4 failed: %v", err)
	}

	// Extract original stco chunk offsets
	origOffsets, origPayloads, err := extractStcoOffsetsAndPayloads(canonicalMP4)
	if err != nil {
		t.Fatalf("extractStcoOffsetsAndPayloads on canonical failed: %v", err)
	}
	if len(origOffsets) == 0 {
		t.Fatal("expected at least one stco chunk offset in canonical MP4")
	}

	meta := &MP4Metadata{
		Title:                 "Chunk Offset Continuity Check",
		Artist:                "Precision DSP Test Runner",
		AIArticle31Disclosure: DefaultArticle31Disclosure(),
		AIIsSynthetic:         true,
	}

	injectedMP4, err := InjectMP4IlstMetadata(canonicalMP4, meta)
	if err != nil {
		t.Fatalf("InjectMP4IlstMetadata failed: %v", err)
	}

	delta := int64(len(injectedMP4)) - int64(len(canonicalMP4))
	if delta <= 0 {
		t.Fatalf("expected positive delta from injecting metadata, got delta=%d", delta)
	}

	newOffsets, newPayloads, err := extractStcoOffsetsAndPayloads(injectedMP4)
	if err != nil {
		t.Fatalf("extractStcoOffsetsAndPayloads on injected MP4 failed: %v", err)
	}

	if len(newOffsets) != len(origOffsets) {
		t.Fatalf("chunk count mismatch: expected %d, got %d", len(origOffsets), len(newOffsets))
	}

	// Verify exact offset arithmetic and audio payload byte-for-byte equality
	for i := range origOffsets {
		expectedOffset := origOffsets[i] + uint32(delta)
		actualOffset := newOffsets[i]
		if actualOffset != expectedOffset {
			t.Errorf("chunk %d offset mismatch: expected %d, got %d (delta was %d)", i, expectedOffset, actualOffset, delta)
		}

		if !bytes.Equal(origPayloads[i], newPayloads[i]) {
			t.Errorf("chunk %d audio payload corrupted after metadata injection!", i)
		}
	}

	t.Logf("Verified %d audio chunks shifted by exactly +%d bytes with 100%% byte-perfect audio payload integrity.", len(origOffsets), delta)
}

func TestMP4Ilst_CoverArt_JPEG_PNG(t *testing.T) {
	canonicalMP4, err := BuildCanonicalTestMP4(1.0, 48000, 2)
	if err != nil {
		t.Fatalf("BuildCanonicalTestMP4 failed: %v", err)
	}

	// Test JPEG cover
	jpegBytes := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01, 0x01, 0x01, 0x00, 0x48, 0x00, 0x48, 0x00, 0x00, 0xFF, 0xD9}
	metaJPEG := &MP4Metadata{
		Title:         "JPEG Cover Test",
		CoverArt:      jpegBytes,
		CoverMimeType: "image/jpeg",
		AIIsSynthetic: true,
	}

	injectedJPEG, err := InjectMP4IlstMetadata(canonicalMP4, metaJPEG)
	if err != nil {
		t.Fatalf("InjectMP4IlstMetadata with JPEG failed: %v", err)
	}
	extractedJPEG, err := ExtractMP4IlstMetadata(injectedJPEG)
	if err != nil {
		t.Fatalf("ExtractMP4IlstMetadata JPEG failed: %v", err)
	}
	if extractedJPEG.CoverMimeType != "image/jpeg" {
		t.Errorf("expected CoverMimeType image/jpeg, got %q", extractedJPEG.CoverMimeType)
	}
	if !bytes.Equal(extractedJPEG.CoverArt, jpegBytes) {
		t.Errorf("JPEG CoverArt payload mismatch")
	}

	// Test PNG cover
	pngBytes := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 'I', 'H', 'D', 'R', 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, 0x89}
	metaPNG := &MP4Metadata{
		Title:         "PNG Cover Test",
		CoverArt:      pngBytes,
		CoverMimeType: "image/png",
		AIIsSynthetic: true,
	}

	injectedPNG, err := InjectMP4IlstMetadata(canonicalMP4, metaPNG)
	if err != nil {
		t.Fatalf("InjectMP4IlstMetadata with PNG failed: %v", err)
	}
	extractedPNG, err := ExtractMP4IlstMetadata(injectedPNG)
	if err != nil {
		t.Fatalf("ExtractMP4IlstMetadata PNG failed: %v", err)
	}
	if extractedPNG.CoverMimeType != "image/png" {
		t.Errorf("expected CoverMimeType image/png, got %q", extractedPNG.CoverMimeType)
	}
	if !bytes.Equal(extractedPNG.CoverArt, pngBytes) {
		t.Errorf("PNG CoverArt payload mismatch")
	}
}

func TestMP4Ilst_TagPreservation_IncrementalUpdate(t *testing.T) {
	canonicalMP4, err := BuildCanonicalTestMP4(1.0, 48000, 2)
	if err != nil {
		t.Fatalf("BuildCanonicalTestMP4 failed: %v", err)
	}

	// First pass: inject initial metadata
	meta1 := &MP4Metadata{
		Title:  "Original Title",
		Artist: "Original Artist",
		Year:   "2025",
	}
	pass1MP4, err := InjectMP4IlstMetadata(canonicalMP4, meta1)
	if err != nil {
		t.Fatalf("Pass 1 injection failed: %v", err)
	}

	// Second pass: inject Article 31 disclosure and album without providing Title or Artist
	meta2 := &MP4Metadata{
		Album:                 "Extended AI Edition",
		AIArticle31Disclosure: DefaultArticle31Disclosure(),
		AIModelName:           "AVibe-2026-v2",
		AIIsSynthetic:         true,
	}
	pass2MP4, err := InjectMP4IlstMetadata(pass1MP4, meta2)
	if err != nil {
		t.Fatalf("Pass 2 injection failed: %v", err)
	}

	extracted, err := ExtractMP4IlstMetadata(pass2MP4)
	if err != nil {
		t.Fatalf("ExtractMP4IlstMetadata on pass 2 failed: %v", err)
	}

	// Verify original fields were preserved
	if extracted.Title != "Original Title" {
		t.Errorf("expected Title %q to be preserved, got %q", "Original Title", extracted.Title)
	}
	if extracted.Artist != "Original Artist" {
		t.Errorf("expected Artist %q to be preserved, got %q", "Original Artist", extracted.Artist)
	}
	if extracted.Year != "2025" {
		t.Errorf("expected Year %q to be preserved, got %q", "2025", extracted.Year)
	}
	// Verify new fields were added
	if extracted.Album != "Extended AI Edition" {
		t.Errorf("expected Album %q, got %q", "Extended AI Edition", extracted.Album)
	}
	if extracted.AIModelName != "AVibe-2026-v2" {
		t.Errorf("expected AIModelName %q, got %q", "AVibe-2026-v2", extracted.AIModelName)
	}
	if extracted.AIArticle31Disclosure != DefaultArticle31Disclosure() {
		t.Errorf("expected Article 31 disclosure to match default statutory text")
	}
}

func TestMP4Ilst_EdgeCases(t *testing.T) {
	_, err := InjectMP4IlstMetadata(nil, &MP4Metadata{Title: "Test"})
	if err != ErrInvalidMP4Data {
		t.Errorf("expected ErrInvalidMP4Data on nil mp4 data, got %v", err)
	}

	canonicalMP4, _ := BuildCanonicalTestMP4(1.0, 48000, 2)
	_, err = InjectMP4IlstMetadata(canonicalMP4, nil)
	if err != ErrNilMP4Metadata {
		t.Errorf("expected ErrNilMP4Metadata on nil meta, got %v", err)
	}

	_, err = ExtractMP4IlstMetadata([]byte{0, 0, 0, 8, 'f', 't', 'y', 'p'})
	if err != ErrInvalidMP4Data {
		t.Errorf("expected ErrInvalidMP4Data on truncated MP4, got %v", err)
	}

	// MP4 without ilst box
	_, err = ExtractMP4IlstMetadata(canonicalMP4)
	if err != ErrMissingIlstBox {
		t.Errorf("expected ErrMissingIlstBox on unmodified canonical MP4, got %v", err)
	}
}

func TestMP4Ilst_100HeterogeneousStreams(t *testing.T) {
	sampleRates := []int{44100, 48000, 96000}
	channelConfigs := []int{1, 2, 6, 8}

	for i := 1; i <= 100; i++ {
		sr := sampleRates[i%len(sampleRates)]
		ch := channelConfigs[i%len(channelConfigs)]
		dur := 0.2 + float64(i%20)*0.1 // 0.2s to 2.1s

		canonical, err := BuildCanonicalTestMP4(dur, sr, ch)
		if err != nil {
			t.Fatalf("[%d/100] BuildCanonicalTestMP4 failed (dur=%.2f, sr=%d, ch=%d): %v", i, dur, sr, ch, err)
		}

		origOffsets, origPayloads, err := extractStcoOffsetsAndPayloads(canonical)
		if err != nil {
			t.Fatalf("[%d/100] extractStcoOffsetsAndPayloads failed: %v", i, err)
		}

		meta := &MP4Metadata{
			Title:                 fmt.Sprintf("Heterogeneous Stream #%d", i),
			Artist:                fmt.Sprintf("Neural Artist #%d", i%10),
			Album:                 fmt.Sprintf("Batch 100 Edition Vol %d", i%5+1),
			Year:                  "2026",
			TrackNumber:           uint16(i),
			TotalTracks:           100,
			AIArticle31Disclosure: DefaultArticle31Disclosure(),
			AIModelName:           fmt.Sprintf("AVibe-Gen-%d", (i%7)+1),
			AIPrompt:              fmt.Sprintf("Procedural stream prompt variation #%d with acoustic layers", i),
			AISeed:                fmt.Sprintf("%d", 10000000+i*7919),
			AIGenerator:           "AVibe Batch Engine",
			AIIsSynthetic:         true,
			AICreationTimestamp:   time.Now().UTC().Format(time.RFC3339),
		}

		injected, err := InjectMP4IlstMetadata(canonical, meta)
		if err != nil {
			t.Fatalf("[%d/100] InjectMP4IlstMetadata failed: %v", i, err)
		}

		delta := int64(len(injected)) - int64(len(canonical))
		if delta <= 0 {
			t.Fatalf("[%d/100] expected positive delta from injecting metadata, got %d", i, delta)
		}

		extracted, err := ExtractMP4IlstMetadata(injected)
		if err != nil {
			t.Fatalf("[%d/100] ExtractMP4IlstMetadata failed: %v", i, err)
		}

		if extracted.Title != meta.Title || extracted.Artist != meta.Artist || extracted.TrackNumber != meta.TrackNumber {
			t.Fatalf("[%d/100] roundtrip metadata mismatch!", i)
		}
		if extracted.AIArticle31Disclosure != meta.AIArticle31Disclosure || !extracted.AIIsSynthetic {
			t.Fatalf("[%d/100] AI Article 31 disclosure mismatch!", i)
		}

		// Verify chunk offset arithmetic and audio payload byte integrity
		newOffsets, newPayloads, err := extractStcoOffsetsAndPayloads(injected)
		if err != nil {
			t.Fatalf("[%d/100] extractStcoOffsetsAndPayloads on injected failed: %v", i, err)
		}

		for c := range origOffsets {
			expectedOffset := origOffsets[c] + uint32(delta)
			if newOffsets[c] != expectedOffset {
				t.Fatalf("[%d/100] chunk %d offset mismatch: expected %d, got %d", i, c, expectedOffset, newOffsets[c])
			}
			if !bytes.Equal(origPayloads[c], newPayloads[c]) {
				t.Fatalf("[%d/100] chunk %d audio payload data altered!", i, c)
			}
		}
	}

	t.Log("Successfully verified 100 heterogeneous MP4 audio streams with Article 31 ilst metadata and byte-perfect chunk offset shift.")
}

// extractStcoOffsetsAndPayloads parses an MP4 file, finds the stco table,
// and extracts the chunk offsets and corresponding raw frame payloads from mdat.
func extractStcoOffsetsAndPayloads(data []byte) ([]uint32, [][]byte, error) {
	nodes, err := parseBoxTree(data)
	if err != nil {
		return nil, nil, err
	}

	var stcoNode *MP4BoxNode
	var findStco func(n *MP4BoxNode)
	findStco = func(n *MP4BoxNode) {
		if n.Type == "stco" {
			stcoNode = n
			return
		}
		for _, ch := range n.Children {
			findStco(ch)
			if stcoNode != nil {
				return
			}
		}
	}

	for _, n := range nodes {
		if n.Type == BoxTypeMOOV {
			findStco(n)
			break
		}
	}

	if stcoNode == nil || len(stcoNode.Payload) < 8 {
		return nil, nil, fmt.Errorf("stco box not found or empty")
	}

	entryCount := binary.BigEndian.Uint32(stcoNode.Payload[4:8])
	var offsets []uint32
	var payloads [][]byte

	for i := uint32(0); i < entryCount; i++ {
		idx := 8 + i*4
		if idx+4 > uint32(len(stcoNode.Payload)) {
			break
		}
		offset := binary.BigEndian.Uint32(stcoNode.Payload[idx : idx+4])
		offsets = append(offsets, offset)

		chunkSize := 256
		if int(offset)+chunkSize <= len(data) {
			chunkBytes := make([]byte, chunkSize)
			copy(chunkBytes, data[offset:offset+uint32(chunkSize)])
			payloads = append(payloads, chunkBytes)
		} else {
			payloads = append(payloads, nil)
		}
	}

	return offsets, payloads, nil
}
