package audio

import (
	"bytes"
	"math/rand"
	"strconv"
	"strings"
	"testing"
)

func TestSynchsafe_RoundTrip(t *testing.T) {
	testValues := []uint32{
		0,
		1,
		127,
		128,
		255,
		256,
		1024,
		16383,
		16384,
		65535,
		1000000,
		0x0FFFFFFF, // Maximum 28-bit capacity
	}

	for _, v := range testValues {
		encoded, err := EncodeSynchsafe(v)
		if err != nil {
			t.Fatalf("EncodeSynchsafe(%d) failed: %v", v, err)
		}

		// Ensure MSB is zero in each byte
		for i, b := range encoded {
			if (b & 0x80) != 0 {
				t.Errorf("value %d: encoded[%d] has MSB set: 0x%02X", v, i, b)
			}
		}

		decoded, err := DecodeSynchsafe(encoded)
		if err != nil {
			t.Fatalf("DecodeSynchsafe failed for %d: %v", v, err)
		}

		if decoded != v {
			t.Errorf("roundtrip mismatch: original=%d, decoded=%d", v, decoded)
		}
	}

	// Test overflow
	_, err := EncodeSynchsafe(0x10000000)
	if err != ErrSynchsafeOverflow {
		t.Errorf("expected ErrSynchsafeOverflow, got %v", err)
	}

	// Test invalid MSB
	_, err = DecodeSynchsafe([4]byte{0x80, 0x00, 0x00, 0x00})
	if err != ErrInvalidSynchsafeByte {
		t.Errorf("expected ErrInvalidSynchsafeByte, got %v", err)
	}
}

func TestID3v24_BuildAndParseStandard(t *testing.T) {
	meta := &AudioMetadata{
		Title:   "Quantum Sonata",
		Artist:  "Cyber Maestro",
		Album:   "Neural Beats 2026",
		Genre:   "Synthwave",
		Year:    "2026",
		ISRC:    "KRA012600123",
		Comment: "Commercial sync verified for streaming",
	}

	tagBytes, err := BuildID3v24Tag(meta)
	if err != nil {
		t.Fatalf("BuildID3v24Tag failed: %v", err)
	}

	// Header inspection
	if len(tagBytes) < 10 {
		t.Fatalf("tag too short: %d bytes", len(tagBytes))
	}
	if !bytes.Equal(tagBytes[0:3], []byte("ID3")) {
		t.Errorf("missing ID3 identifier: %s", string(tagBytes[0:3]))
	}
	if tagBytes[3] != 0x04 || tagBytes[4] != 0x00 {
		t.Errorf("expected version 2.4.0, got 2.%d.%d", tagBytes[3], tagBytes[4])
	}

	// Parse back
	parsed, err := ParseID3v24(tagBytes)
	if err != nil {
		t.Fatalf("ParseID3v24 failed: %v", err)
	}

	if parsed.Title != meta.Title {
		t.Errorf("title mismatch: %s vs %s", parsed.Title, meta.Title)
	}
	if parsed.Artist != meta.Artist {
		t.Errorf("artist mismatch: %s vs %s", parsed.Artist, meta.Artist)
	}
	if parsed.Album != meta.Album {
		t.Errorf("album mismatch: %s vs %s", parsed.Album, meta.Album)
	}
	if parsed.Genre != meta.Genre {
		t.Errorf("genre mismatch: %s vs %s", parsed.Genre, meta.Genre)
	}
	if parsed.Year != meta.Year {
		t.Errorf("year mismatch: %s vs %s", parsed.Year, meta.Year)
	}
	if parsed.ISRC != meta.ISRC {
		t.Errorf("isrc mismatch: %s vs %s", parsed.ISRC, meta.ISRC)
	}
	if parsed.Comment != meta.Comment {
		t.Errorf("comment mismatch: %s vs %s", parsed.Comment, meta.Comment)
	}
}

func TestID3v24_AITransparencyAndClearance(t *testing.T) {
	meta := &AudioMetadata{
		Title:            "AI Symphony No. 9",
		Artist:           "Wonseoppei & AVIBE Engine",
		AIProvider:       "avibe-diffusion-v2",
		AIModelVersion:   "v2.5.4",
		AIGenerationType: "AI_ASSISTED_HUMAN",
		AIPrompt:         "orchestral strings with modular analog synth 120bpm c major",
		LicenseTier:      "COMMERCIAL_BROADCAST",
		ClearanceID:      "CLR-2026-998822",
		CustomFields: map[string]string{
			"BPM":           "120",
			"KEY_SIGNATURE": "C Major",
			"MOOD":          "Uplifting",
		},
	}

	tagBytes, err := BuildID3v24Tag(meta)
	if err != nil {
		t.Fatalf("BuildID3v24Tag failed: %v", err)
	}

	parsed, err := ParseID3v24(tagBytes)
	if err != nil {
		t.Fatalf("ParseID3v24 failed: %v", err)
	}

	if parsed.AIProvider != meta.AIProvider {
		t.Errorf("ai provider mismatch: %s vs %s", parsed.AIProvider, meta.AIProvider)
	}
	if parsed.AIModelVersion != meta.AIModelVersion {
		t.Errorf("ai model version mismatch: %s vs %s", parsed.AIModelVersion, meta.AIModelVersion)
	}
	if parsed.AIGenerationType != meta.AIGenerationType {
		t.Errorf("ai generation type mismatch: %s vs %s", parsed.AIGenerationType, meta.AIGenerationType)
	}
	if parsed.AIPrompt != meta.AIPrompt {
		t.Errorf("ai prompt mismatch: %s vs %s", parsed.AIPrompt, meta.AIPrompt)
	}
	if parsed.LicenseTier != meta.LicenseTier {
		t.Errorf("license tier mismatch: %s vs %s", parsed.LicenseTier, meta.LicenseTier)
	}
	if parsed.ClearanceID != meta.ClearanceID {
		t.Errorf("clearance id mismatch: %s vs %s", parsed.ClearanceID, meta.ClearanceID)
	}

	for k, expectedVal := range meta.CustomFields {
		actualVal, exists := parsed.CustomFields[k]
		if !exists || actualVal != expectedVal {
			t.Errorf("custom field %s mismatch: %s vs %s", k, actualVal, expectedVal)
		}
	}
}

func TestID3v24_CryptographicWatermarkAndCoverArt(t *testing.T) {
	watermarkOwner := "avibe.io/watermark/v1"
	watermarkPayload := []byte{
		0xDE, 0xAD, 0xBE, 0xEF, 0x01, 0x02, 0x03, 0x04,
		0xCA, 0xFE, 0xBA, 0xBE, 0x10, 0x20, 0x30, 0x40,
	}

	coverArtMime := "image/jpeg"
	coverArtData := make([]byte, 512)
	coverArtData[0] = 0xFF
	coverArtData[1] = 0xD8 // JPEG SOI marker
	coverArtData[510] = 0xFF
	coverArtData[511] = 0xD9 // JPEG EOI marker

	meta := &AudioMetadata{
		Title:          "Signed Master Track",
		WatermarkOwner: watermarkOwner,
		WatermarkData:  watermarkPayload,
		CoverArtMime:   coverArtMime,
		CoverArtData:   coverArtData,
	}

	tagBytes, err := BuildID3v24Tag(meta)
	if err != nil {
		t.Fatalf("BuildID3v24Tag failed: %v", err)
	}

	parsed, err := ParseID3v24(tagBytes)
	if err != nil {
		t.Fatalf("ParseID3v24 failed: %v", err)
	}

	if parsed.WatermarkOwner != watermarkOwner {
		t.Errorf("watermark owner mismatch: %s vs %s", parsed.WatermarkOwner, watermarkOwner)
	}
	if !bytes.Equal(parsed.WatermarkData, watermarkPayload) {
		t.Errorf("watermark data mismatch: %v vs %v", parsed.WatermarkData, watermarkPayload)
	}

	if parsed.CoverArtMime != coverArtMime {
		t.Errorf("cover art mime mismatch: %s vs %s", parsed.CoverArtMime, coverArtMime)
	}
	if !bytes.Equal(parsed.CoverArtData, coverArtData) {
		t.Errorf("cover art bytes mismatch length %d vs %d", len(parsed.CoverArtData), len(coverArtData))
	}
}

func TestID3v24_StreamInjectAndStrip(t *testing.T) {
	// Raw audio mock payload (e.g. simulated AAC / MP3 frames)
	rawAudio := make([]byte, 2048)
	for i := range rawAudio {
		rawAudio[i] = byte((i * 31) % 256)
	}

	meta := &AudioMetadata{
		Title:  "Stream Injection Test",
		Artist: "Sonic Pioneer",
	}

	// 1. Inject tag into raw audio
	injected, err := InjectID3v24(rawAudio, meta)
	if err != nil {
		t.Fatalf("InjectID3v24 failed: %v", err)
	}

	if len(injected) <= len(rawAudio) {
		t.Errorf("injected length must be greater than raw audio: %d <= %d", len(injected), len(rawAudio))
	}

	// 2. Parse from injected stream
	parsed, err := ParseID3v24(injected)
	if err != nil {
		t.Fatalf("ParseID3v24 on injected stream failed: %v", err)
	}
	if parsed.Title != meta.Title {
		t.Errorf("parsed title mismatch: %s vs %s", parsed.Title, meta.Title)
	}

	// 3. Strip tag and verify clean audio recovered
	stripped, err := StripID3v24(injected)
	if err != nil {
		t.Fatalf("StripID3v24 failed: %v", err)
	}
	if !bytes.Equal(stripped, rawAudio) {
		t.Fatalf("stripped audio does not match original raw audio")
	}

	// 4. Re-inject with updated metadata
	metaUpdated := &AudioMetadata{
		Title:  "Stream Updated Title",
		Artist: "Sonic Pioneer Enhanced",
	}
	reInjected, err := InjectID3v24(injected, metaUpdated)
	if err != nil {
		t.Fatalf("Re-inject failed: %v", err)
	}

	parsedUpdated, err := ParseID3v24(reInjected)
	if err != nil {
		t.Fatalf("Parse updated stream failed: %v", err)
	}
	if parsedUpdated.Title != "Stream Updated Title" {
		t.Errorf("updated title mismatch: %s", parsedUpdated.Title)
	}
}

func TestID3v24_EdgeCases(t *testing.T) {
	// Nil metadata
	_, err := BuildID3v24Tag(nil)
	if err != ErrNilMetadata {
		t.Errorf("expected ErrNilMetadata, got %v", err)
	}

	// Empty stream
	_, err = ParseID3v24([]byte{})
	if err != ErrCorruptID3Header {
		t.Errorf("expected ErrCorruptID3Header, got %v", err)
	}

	// No ID3 tag found
	_, err = ParseID3v24([]byte("RIFF12345678"))
	if err != ErrNoID3TagFound {
		t.Errorf("expected ErrNoID3TagFound, got %v", err)
	}

	// Unsupported ID3 version (e.g. ID3v2.3)
	tagV3 := []byte{0x49, 0x44, 0x33, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10}
	_, err = ParseID3v24(tagV3)
	if !strings.Contains(err.Error(), "unsupported id3 version") {
		t.Errorf("expected unsupported version error, got %v", err)
	}

	// Strip non-ID3 stream returns raw data with ErrNoID3TagFound
	nonID3 := []byte("SOME_RAW_AUDIO_STREAM_BYTES")
	clean, err := StripID3v24(nonID3)
	if err != ErrNoID3TagFound {
		t.Errorf("expected ErrNoID3TagFound, got %v", err)
	}
	if !bytes.Equal(clean, nonID3) {
		t.Errorf("expected unstripped data to be preserved")
	}
}

func TestID3v24_100HeterogeneousMetadata(t *testing.T) {
	rng := rand.New(rand.NewSource(20261109))
	totalTests := 100

	koreanTitles := []string{"사이버펑크 야상곡", "인공지능 교향곡 1번", "네온 시티의 밤", "가을비 왈츠", "빛의 잔상"}
	artists := []string{"AVIBE Studio", "Maestro AI", "Wonseoppei", "Cyber Symphony", "Digital Sound Lab"}
	providers := []string{"avibe-core", "suno-v4", "udio-v2", "stable-audio", "music-lm"}

	for iter := 0; iter < totalTests; iter++ {
		title := koreanTitles[rng.Intn(len(koreanTitles))] + " #" + strconv.Itoa(iter)
		artist := artists[rng.Intn(len(artists))]
		provider := providers[rng.Intn(len(providers))]

		meta := &AudioMetadata{
			Title:            title,
			Artist:           artist,
			Album:            "Collection " + strconv.Itoa(iter),
			Genre:            "Electronic",
			Year:             "2026",
			ISRC:             "KR-A01-26-" + strconv.Itoa(20000+iter),
			Comment:          "Special license test: " + strconv.Itoa(iter),
			AIProvider:       provider,
			AIModelVersion:   "v" + strconv.Itoa(rng.Intn(5)+1),
			AIGenerationType: "AI_ASSISTED_HUMAN",
			AIPrompt:         "cinematic ambient strings with beats " + strconv.Itoa(iter),
			LicenseTier:      "COMMERCIAL_BROADCAST",
			ClearanceID:      "CLEARANCE-" + strconv.Itoa(iter*12345),
			WatermarkOwner:   "avibe.io/watermark",
			WatermarkData:    []byte("WATERMARK_HASH_" + strconv.Itoa(iter)),
			CustomFields: map[string]string{
				"ITERATION": strconv.Itoa(iter),
				"CHECK_SUM": strconv.Itoa(iter * 9999),
			},
		}

		tagBytes, err := BuildID3v24Tag(meta)
		if err != nil {
			t.Fatalf("iteration %d: BuildID3v24Tag failed: %v", iter, err)
		}

		parsed, err := ParseID3v24(tagBytes)
		if err != nil {
			t.Fatalf("iteration %d: ParseID3v24 failed: %v", iter, err)
		}

		if parsed.Title != meta.Title {
			t.Fatalf("iteration %d: title mismatch %s vs %s", iter, parsed.Title, meta.Title)
		}
		if parsed.AIProvider != meta.AIProvider {
			t.Fatalf("iteration %d: ai provider mismatch %s vs %s", iter, parsed.AIProvider, meta.AIProvider)
		}
		if !bytes.Equal(parsed.WatermarkData, meta.WatermarkData) {
			t.Fatalf("iteration %d: watermark mismatch", iter)
		}
		if parsed.CustomFields["ITERATION"] != strconv.Itoa(iter) {
			t.Fatalf("iteration %d: custom field mismatch", iter)
		}
	}

	t.Logf("Successfully verified 100 heterogeneous ID3v2.4 metadata tag generations and parsings with 0 errors.")
}
