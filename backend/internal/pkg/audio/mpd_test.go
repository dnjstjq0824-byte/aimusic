package audio

import (
	"math"
	"math/rand"
	"strconv"
	"strings"
	"testing"
)

func TestISODuration_RoundTrip(t *testing.T) {
	testCases := []struct {
		seconds  float64
		expected string
	}{
		{0.0, "PT0S"},
		{5.5, "PT5.500S"},
		{60.0, "PT1M"},
		{125.0, "PT2M5S"},
		{185.5, "PT3M5.500S"},
		{3600.0, "PT1H"},
		{3661.25, "PT1H1M1.250S"},
	}

	for _, tc := range testCases {
		iso := FormatISODuration(tc.seconds)
		if iso != tc.expected {
			t.Errorf("FormatISODuration(%f) = %s, expected %s", tc.seconds, iso, tc.expected)
		}

		parsed, err := ParseISODuration(iso)
		if err != nil {
			t.Errorf("ParseISODuration(%s) failed: %v", iso, err)
		}
		if math.Abs(parsed-tc.seconds) > 0.001 {
			t.Errorf("ParseISODuration(%s) = %f, expected %f", iso, parsed, tc.seconds)
		}
	}
}

func TestDASHManifest_StandardVOD(t *testing.T) {
	spec := &DASHStreamSpec{
		TrackID:            "track_test_vod_001",
		Title:              "Cyberpunk Sunset",
		Artist:             "AI Maestro",
		DurationSec:        180.0,
		SegmentDurationSec: 6.0,
		BaseURL:            "https://cdn.avibe.io/hls/tracks/track_test_vod_001/",
		Profiles:           DefaultStreamingAudioProfiles(),
	}

	xmlData, err := BuildDASHManifest(spec)
	if err != nil {
		t.Fatalf("BuildDASHManifest failed: %v", err)
	}

	xmlStr := string(xmlData)
	if !strings.HasPrefix(xmlStr, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>") {
		t.Fatalf("missing standard XML declaration: %s", xmlStr[:50])
	}

	mpd, err := ValidateDASHManifest(xmlData)
	if err != nil {
		t.Fatalf("ValidateDASHManifest failed: %v", err)
	}

	if mpd.Type != "static" {
		t.Errorf("expected static type, got %s", mpd.Type)
	}
	if mpd.MediaPresentationDuration != "PT3M" {
		t.Errorf("expected PT3M duration, got %s", mpd.MediaPresentationDuration)
	}
	if len(mpd.Period) != 1 {
		t.Fatalf("expected 1 period, got %d", len(mpd.Period))
	}

	period := mpd.Period[0]
	if len(period.AdaptationSets) != 1 {
		t.Fatalf("expected 1 adaptation set, got %d", len(period.AdaptationSets))
	}

	as := period.AdaptationSets[0]
	if as.ContentType != "audio" || as.MimeType != "audio/mp4" {
		t.Errorf("invalid adaptation set MIME/Content: %s / %s", as.MimeType, as.ContentType)
	}
	if len(as.Representations) != 5 {
		t.Fatalf("expected 5 representations, got %d", len(as.Representations))
	}

	expectedBandwidths := []int{64000, 128000, 192000, 256000, 320000}
	for i, rep := range as.Representations {
		if rep.Bandwidth != expectedBandwidths[i] {
			t.Errorf("representation[%d] expected bandwidth %d, got %d", i, expectedBandwidths[i], rep.Bandwidth)
		}
	}
}

func TestDASHManifest_WithTimeline(t *testing.T) {
	spec := &DASHStreamSpec{
		TrackID:            "track_timeline_002",
		DurationSec:        45.5,
		SegmentDurationSec: 6.0,
		UseTimeline:        true,
	}

	xmlData, err := BuildDASHManifest(spec)
	if err != nil {
		t.Fatalf("BuildDASHManifest with timeline failed: %v", err)
	}

	mpd, err := ValidateDASHManifest(xmlData)
	if err != nil {
		t.Fatalf("manifest validation failed: %v", err)
	}

	as := mpd.Period[0].AdaptationSets[0]
	if as.SegmentTemplate == nil || as.SegmentTemplate.SegmentTimeline == nil {
		t.Fatalf("expected SegmentTimeline in SegmentTemplate")
	}

	timeline := as.SegmentTemplate.SegmentTimeline
	if len(timeline.Segments) == 0 {
		t.Fatalf("timeline has no segments")
	}

	// 45.5s at 48000 timescale = 2,184,000 ticks
	// 6s segment = 288,000 ticks. 45.5s = 7 * 6s (42s) + 3.5s remainder (168,000 ticks)
	firstSeg := timeline.Segments[0]
	if firstSeg.D != 288000 {
		t.Errorf("expected 288000 tick duration for full segment, got %d", firstSeg.D)
	}
	if firstSeg.R == nil || *firstSeg.R != 6 {
		t.Errorf("expected repeat count 6, got %v", firstSeg.R)
	}
	if len(timeline.Segments) > 1 {
		lastSeg := timeline.Segments[1]
		if lastSeg.D != 168000 {
			t.Errorf("expected remainder segment 168000 ticks (3.5s), got %d", lastSeg.D)
		}
	}
}

func TestDASHManifest_AIDisclosureAndDRM(t *testing.T) {
	spec := &DASHStreamSpec{
		TrackID:            "track_ai_drm_003",
		DurationSec:        120.0,
		SegmentDurationSec: 6.0,
		AIProvider:         "avibe-diffusion",
		AIModelVersion:     "v3.1",
		AIGenerationType:   "AI_ASSISTED_HUMAN",
		ISRC:               "KRA012600999",
		EnableDRM:          true,
		DRMKeyID:           "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
	}

	xmlData, err := BuildDASHManifest(spec)
	if err != nil {
		t.Fatalf("BuildDASHManifest failed: %v", err)
	}

	xmlStr := string(xmlData)
	if !strings.Contains(xmlStr, DASHAIDisclosureScheme) {
		t.Errorf("missing AI disclosure scheme in manifest")
	}
	if !strings.Contains(xmlStr, "AI_ASSISTED_HUMAN") || !strings.Contains(xmlStr, "KRA012600999") {
		t.Errorf("missing AI disclosure attributes in manifest")
	}
	if !strings.Contains(xmlStr, DASHProtectionCENC) {
		t.Errorf("missing CENC protection in manifest")
	}
	if !strings.Contains(xmlStr, DASHProtectionWidevine) {
		t.Errorf("missing Widevine protection in manifest")
	}
	if !strings.Contains(xmlStr, DASHProtectionPlayReady) {
		t.Errorf("missing PlayReady protection in manifest")
	}

	_, err = ValidateDASHManifest(xmlData)
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}
}

func TestDASHManifest_EdgeCases(t *testing.T) {
	// Nil spec
	_, err := BuildDASHManifest(nil)
	if err != ErrNilStreamSpec {
		t.Errorf("expected ErrNilStreamSpec, got %v", err)
	}

	// Empty track ID
	_, err = BuildDASHManifest(&DASHStreamSpec{TrackID: "", DurationSec: 100})
	if err != ErrInvalidTrackID {
		t.Errorf("expected ErrInvalidTrackID, got %v", err)
	}

	// Non-positive duration
	_, err = BuildDASHManifest(&DASHStreamSpec{TrackID: "id1", DurationSec: 0})
	if err != ErrInvalidDuration {
		t.Errorf("expected ErrInvalidDuration, got %v", err)
	}

	// Invalid ISO duration
	_, err = ParseISODuration("INVALID_ISO")
	if err != ErrInvalidISODuration {
		t.Errorf("expected ErrInvalidISODuration, got %v", err)
	}

	// Corrupt manifest XML
	_, err = ValidateDASHManifest([]byte("<MPD>broken xml"))
	if err == nil {
		t.Errorf("expected error on corrupt XML")
	}
}

func TestDASHManifest_100HeterogeneousStreams(t *testing.T) {
	rng := rand.New(rand.NewSource(20261109))
	totalTests := 100

	providers := []string{"avibe-core", "suno-v4", "udio-v2", "stable-audio", "riffusion"}
	genTypes := []string{"AI_ASSISTED_HUMAN", "FULL_AI", "HUMAN_COMPOSED"}

	for iter := 0; iter < totalTests; iter++ {
		durationSec := 10.0 + rng.Float64()*1200.0 // 10s to 20 minutes
		segDuration := 2.0 + rng.Float64()*8.0     // 2s to 10s
		useTimeline := rng.Intn(2) == 1
		enableDRM := rng.Intn(2) == 1

		numProfiles := 1 + rng.Intn(5)
		allProfiles := DefaultStreamingAudioProfiles()
		profiles := allProfiles[:numProfiles]

		spec := &DASHStreamSpec{
			TrackID:            "track_rand_" + strconv.Itoa(iter),
			Title:              "Dynamic Track " + strconv.Itoa(iter),
			Artist:             "Artist " + strconv.Itoa(iter),
			DurationSec:        durationSec,
			SegmentDurationSec: segDuration,
			BaseURL:            "https://cdn.avibe.io/stream/",
			Profiles:           profiles,
			AIProvider:         providers[rng.Intn(len(providers))],
			AIGenerationType:   genTypes[rng.Intn(len(genTypes))],
			AIModelVersion:     "v" + strconv.Itoa(rng.Intn(5)+1),
			ISRC:               "KR-A01-26-" + strconv.Itoa(10000+iter),
			EnableDRM:          enableDRM,
			DRMKeyID:           "key-id-" + strconv.Itoa(iter),
			UseTimeline:        useTimeline,
		}

		xmlBytes, err := BuildDASHManifest(spec)
		if err != nil {
			t.Fatalf("iteration %d: BuildDASHManifest failed: %v", iter, err)
		}

		mpd, err := ValidateDASHManifest(xmlBytes)
		if err != nil {
			t.Fatalf("iteration %d: ValidateDASHManifest failed: %v", iter, err)
		}

		if len(mpd.Period) == 0 {
			t.Fatalf("iteration %d: zero periods", iter)
		}
		if len(mpd.Period[0].AdaptationSets[0].Representations) != numProfiles {
			t.Fatalf("iteration %d: representations mismatch", iter)
		}
	}

	t.Logf("Successfully verified 100 heterogeneous MPEG-DASH MPD dynamic manifests with 0 errors.")
}
