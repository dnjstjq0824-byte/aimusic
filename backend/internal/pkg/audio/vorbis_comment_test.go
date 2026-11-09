package audio

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestVorbisComment_OggOpus_InjectAndExtract(t *testing.T) {
	canonicalOgg, err := BuildCanonicalTestOgg(OggCodecOpus, 1.0, 48000, 2)
	if err != nil {
		t.Fatalf("BuildCanonicalTestOgg Opus failed: %v", err)
	}

	vc := NewVorbisComment()
	vc.Title = "Neural Horizon"
	vc.Artist = "AVibe Cyber Quartet"
	vc.Album = "Acoustic Singularity 2026"
	vc.TrackNumber = "4"
	vc.Date = "2026"
	vc.Genre = "Cyber Ambient"
	vc.Copyright = "© 2026 AVibe AI Records"
	vc.Description = "Procedurally generated ambient soundscape"
	vc.AIArticle31Disclosure = DefaultVorbisArticle31Disclosure()
	vc.AIModelName = "AVibe-Opus-v4.1"
	vc.AIPrompt = "ethereal ambient synth pad with binaural rain textures, 60 bpm"
	vc.AISeed = "5544332211"
	vc.AIGenerator = "AVibe Ogg Engine"
	vc.AIIsSynthetic = true
	vc.AICreationTimestamp = time.Now().UTC().Format(time.RFC3339)
	vc.Set("CUSTOM_AI_LICENSE", "AVIBE-OPEN-MODEL-V1")

	injectedOgg, err := InjectOggVorbisComment(canonicalOgg, vc)
	if err != nil {
		t.Fatalf("InjectOggVorbisComment failed: %v", err)
	}

	// Verify all Ogg pages in the injected stream have valid CRCs
	pages, err := ParseOggPages(injectedOgg)
	if err != nil {
		t.Fatalf("ParseOggPages on injected stream failed (invalid CRC or structure): %v", err)
	}
	if len(pages) < 3 {
		t.Fatalf("expected at least 3 pages in injected Ogg stream, got %d", len(pages))
	}

	extractedVC, err := ExtractOggVorbisComment(injectedOgg)
	if err != nil {
		t.Fatalf("ExtractOggVorbisComment failed: %v", err)
	}

	if extractedVC.Title != vc.Title {
		t.Errorf("Title mismatch: expected %q, got %q", vc.Title, extractedVC.Title)
	}
	if extractedVC.Artist != vc.Artist {
		t.Errorf("Artist mismatch: expected %q, got %q", vc.Artist, extractedVC.Artist)
	}
	if extractedVC.Album != vc.Album {
		t.Errorf("Album mismatch: expected %q, got %q", vc.Album, extractedVC.Album)
	}
	if extractedVC.TrackNumber != vc.TrackNumber {
		t.Errorf("TrackNumber mismatch: expected %q, got %q", vc.TrackNumber, extractedVC.TrackNumber)
	}
	if extractedVC.AIArticle31Disclosure != vc.AIArticle31Disclosure {
		t.Errorf("AIArticle31Disclosure mismatch:\nexpected %q\ngot %q", vc.AIArticle31Disclosure, extractedVC.AIArticle31Disclosure)
	}
	if extractedVC.AIModelName != vc.AIModelName {
		t.Errorf("AIModelName mismatch: expected %q, got %q", vc.AIModelName, extractedVC.AIModelName)
	}
	if extractedVC.AIPrompt != vc.AIPrompt {
		t.Errorf("AIPrompt mismatch: expected %q, got %q", vc.AIPrompt, extractedVC.AIPrompt)
	}
	if !extractedVC.AIIsSynthetic {
		t.Errorf("AIIsSynthetic mismatch: expected true, got %v", extractedVC.AIIsSynthetic)
	}
	if extractedVC.Get("CUSTOM_AI_LICENSE") != "AVIBE-OPEN-MODEL-V1" {
		t.Errorf("custom tag mismatch: got %q", extractedVC.Get("CUSTOM_AI_LICENSE"))
	}
}

func TestVorbisComment_OggVorbis_InjectAndExtract(t *testing.T) {
	canonicalOgg, err := BuildCanonicalTestOgg(OggCodecVorbis, 1.0, 48000, 2)
	if err != nil {
		t.Fatalf("BuildCanonicalTestOgg Vorbis failed: %v", err)
	}

	vc := NewVorbisComment()
	vc.Title = "Vorbis Spectrum Harmony"
	vc.Artist = "Synth Collective"
	vc.Album = "Open Source Sonics"
	vc.AIArticle31Disclosure = DefaultVorbisArticle31Disclosure()
	vc.AIModelName = "AVibe-Vorbis-Core"
	vc.AIIsSynthetic = true

	injectedOgg, err := InjectOggVorbisComment(canonicalOgg, vc)
	if err != nil {
		t.Fatalf("InjectOggVorbisComment on Vorbis failed: %v", err)
	}

	pages, err := ParseOggPages(injectedOgg)
	if err != nil {
		t.Fatalf("ParseOggPages on Vorbis failed: %v", err)
	}
	if len(pages) < 4 {
		t.Fatalf("expected at least 4 pages in Vorbis stream, got %d", len(pages))
	}

	extractedVC, err := ExtractOggVorbisComment(injectedOgg)
	if err != nil {
		t.Fatalf("ExtractOggVorbisComment failed: %v", err)
	}

	if extractedVC.Title != vc.Title {
		t.Errorf("Title mismatch: expected %q, got %q", vc.Title, extractedVC.Title)
	}
	if extractedVC.AIArticle31Disclosure != vc.AIArticle31Disclosure {
		t.Errorf("Article 31 disclosure mismatch")
	}
}

func TestVorbisComment_AudioPayload_BytePerfectPreservation(t *testing.T) {
	canonicalOgg, err := BuildCanonicalTestOgg(OggCodecOpus, 2.0, 48000, 2)
	if err != nil {
		t.Fatalf("BuildCanonicalTestOgg failed: %v", err)
	}

	origPages, err := ParseOggPages(canonicalOgg)
	if err != nil {
		t.Fatalf("ParseOggPages orig failed: %v", err)
	}

	// Audio pages start after Page 1
	var origAudioPayloads [][]byte
	var origGranules []uint64
	for idx := 2; idx < len(origPages); idx++ {
		origAudioPayloads = append(origAudioPayloads, origPages[idx].Payload)
		origGranules = append(origGranules, origPages[idx].Granule)
	}

	vc := NewVorbisComment()
	vc.Title = "Byte-Perfect Preservation Test"
	vc.AIArticle31Disclosure = DefaultVorbisArticle31Disclosure()
	vc.AIIsSynthetic = true

	injectedOgg, err := InjectOggVorbisComment(canonicalOgg, vc)
	if err != nil {
		t.Fatalf("InjectOggVorbisComment failed: %v", err)
	}

	newPages, err := ParseOggPages(injectedOgg)
	if err != nil {
		t.Fatalf("ParseOggPages injected failed: %v", err)
	}

	var newAudioPayloads [][]byte
	var newGranules []uint64
	for idx := 2; idx < len(newPages); idx++ {
		newAudioPayloads = append(newAudioPayloads, newPages[idx].Payload)
		newGranules = append(newGranules, newPages[idx].Granule)
	}

	if len(origAudioPayloads) != len(newAudioPayloads) {
		t.Fatalf("audio page count mismatch: orig %d, new %d", len(origAudioPayloads), len(newAudioPayloads))
	}

	for i := range origAudioPayloads {
		if !bytes.Equal(origAudioPayloads[i], newAudioPayloads[i]) {
			t.Fatalf("audio payload byte mismatch on page %d!", i+2)
		}
		if origGranules[i] != newGranules[i] {
			t.Fatalf("audio granule position mismatch on page %d: orig %d, new %d", i+2, origGranules[i], newGranules[i])
		}
	}

	t.Logf("Successfully verified %d audio pages preserved with 100%% byte-perfect payload and granule integrity.", len(origAudioPayloads))
}

func TestVorbisComment_MultiPageSpanning(t *testing.T) {
	canonicalOgg, err := BuildCanonicalTestOgg(OggCodecOpus, 0.5, 48000, 2)
	if err != nil {
		t.Fatalf("BuildCanonicalTestOgg failed: %v", err)
	}

	// Create a comment packet exceeding 65,025 bytes to force multi-page Ogg segmentation
	largePrompt := strings.Repeat("Epic procedural prompt embedding text with deep timbre and harmonic frequencies. ", 1200) // ~75 KB
	vc := NewVorbisComment()
	vc.Title = "Large Multi-Page Metadata Stream"
	vc.AIPrompt = largePrompt
	vc.AIArticle31Disclosure = DefaultVorbisArticle31Disclosure()
	vc.AIIsSynthetic = true

	injectedOgg, err := InjectOggVorbisComment(canonicalOgg, vc)
	if err != nil {
		t.Fatalf("InjectOggVorbisComment with large comment failed: %v", err)
	}

	pages, err := ParseOggPages(injectedOgg)
	if err != nil {
		t.Fatalf("ParseOggPages failed on multi-page comment: %v", err)
	}

	// Verify that page 2 has continuation flag (0x01)
	if (pages[2].Flags & 0x01) == 0 {
		t.Errorf("expected page 2 to have continuation flag 0x01, got flags=0x%02x", pages[2].Flags)
	}

	extractedVC, err := ExtractOggVorbisComment(injectedOgg)
	if err != nil {
		t.Fatalf("ExtractOggVorbisComment on multi-page stream failed: %v", err)
	}

	if extractedVC.AIPrompt != largePrompt {
		t.Fatalf("large AIPrompt reassembly mismatch: expected len %d, got len %d", len(largePrompt), len(extractedVC.AIPrompt))
	}
	t.Logf("Successfully verified multi-page VorbisComment packet (%d bytes) spanned across Ogg pages and reassembled flawlessly.", len(largePrompt))
}

func TestVorbisComment_CaseInsensitiveKeyHandling(t *testing.T) {
	vc := NewVorbisComment()
	vc.Set("title", "Case Invariance Song")
	vc.Set("Artist", "Case Invariance Singer")
	vc.Add("GENRE", "Rock")
	vc.Add("genre", "Electronic") // multi-value tag

	if vc.Get("TITLE") != "Case Invariance Song" {
		t.Errorf("expected 'Case Invariance Song', got %q", vc.Get("TITLE"))
	}
	if vc.Get("artist") != "Case Invariance Singer" {
		t.Errorf("expected 'Case Invariance Singer', got %q", vc.Get("artist"))
	}

	genres := vc.GetAll("Genre")
	if len(genres) != 2 || genres[0] != "Rock" || genres[1] != "Electronic" {
		t.Errorf("expected 2 genres [Rock, Electronic], got %v", genres)
	}
}

func TestVorbisComment_TagPreservation_IncrementalUpdate(t *testing.T) {
	canonicalOgg, err := BuildCanonicalTestOgg(OggCodecOpus, 1.0, 48000, 2)
	if err != nil {
		t.Fatalf("BuildCanonicalTestOgg failed: %v", err)
	}

	// Pass 1: inject title and artist
	vc1 := NewVorbisComment()
	vc1.Title = "Initial Title"
	vc1.Artist = "Initial Artist"
	pass1Ogg, err := InjectOggVorbisComment(canonicalOgg, vc1)
	if err != nil {
		t.Fatalf("Pass 1 injection failed: %v", err)
	}

	// Pass 2: inject album and AI disclosure without title or artist
	vc2 := NewVorbisComment()
	vc2.Album = "New Album AI Edition"
	vc2.AIArticle31Disclosure = DefaultVorbisArticle31Disclosure()
	vc2.AIIsSynthetic = true
	pass2Ogg, err := InjectOggVorbisComment(pass1Ogg, vc2)
	if err != nil {
		t.Fatalf("Pass 2 injection failed: %v", err)
	}

	extracted, err := ExtractOggVorbisComment(pass2Ogg)
	if err != nil {
		t.Fatalf("ExtractOggVorbisComment on pass 2 failed: %v", err)
	}

	if extracted.Title != "Initial Title" {
		t.Errorf("expected Title to be preserved, got %q", extracted.Title)
	}
	if extracted.Artist != "Initial Artist" {
		t.Errorf("expected Artist to be preserved, got %q", extracted.Artist)
	}
	if extracted.Album != "New Album AI Edition" {
		t.Errorf("expected Album to be added, got %q", extracted.Album)
	}
}

func TestVorbisComment_EdgeCases(t *testing.T) {
	_, err := InjectOggVorbisComment(nil, NewVorbisComment())
	if err == nil {
		t.Error("expected error on nil ogg data")
	}

	canonical, _ := BuildCanonicalTestOgg(OggCodecOpus, 1.0, 48000, 2)
	_, err = InjectOggVorbisComment(canonical, nil)
	if err != ErrNilVorbisComment {
		t.Errorf("expected ErrNilVorbisComment, got %v", err)
	}

	_, err = ExtractOggVorbisComment([]byte("OggS" + strings.Repeat("\x00", 20)))
	if err == nil {
		t.Error("expected error on truncated ogg")
	}

	// Corrupted CRC test
	corruptOgg := make([]byte, len(canonical))
	copy(corruptOgg, canonical)
	corruptOgg[22] ^= 0xFF // flip CRC byte
	_, err = ExtractOggVorbisComment(corruptOgg)
	if err == nil {
		t.Error("expected CRC error on corrupted Ogg page")
	}
}

func TestVorbisComment_100HeterogeneousStreams(t *testing.T) {
	codecs := []OggCodecType{OggCodecOpus, OggCodecVorbis}
	sampleRates := []int{44100, 48000, 96000}
	channelConfigs := []int{1, 2, 6, 8}

	for i := 1; i <= 100; i++ {
		codec := codecs[i%len(codecs)]
		sr := sampleRates[i%len(sampleRates)]
		ch := channelConfigs[i%len(channelConfigs)]
		dur := 0.1 + float64(i%15)*0.1 // 0.1s to 1.5s

		canonical, err := BuildCanonicalTestOgg(codec, dur, sr, ch)
		if err != nil {
			t.Fatalf("[%d/100] BuildCanonicalTestOgg failed (codec=%s, dur=%.2f, sr=%d, ch=%d): %v", i, codec, dur, sr, ch, err)
		}

		origPages, err := ParseOggPages(canonical)
		if err != nil {
			t.Fatalf("[%d/100] ParseOggPages on canonical failed: %v", i, err)
		}

		// Record audio payloads
		startAudioIdx := 2
		if codec == OggCodecVorbis {
			startAudioIdx = 3 // Page 0: ID, Page 1: Comment, Page 2: Setup, Page 3: Audio
		}
		var origAudioPayloads [][]byte
		for a := startAudioIdx; a < len(origPages); a++ {
			origAudioPayloads = append(origAudioPayloads, origPages[a].Payload)
		}

		vc := NewVorbisComment()
		vc.Title = fmt.Sprintf("Heterogeneous Stream #%d", i)
		vc.Artist = fmt.Sprintf("Neural Ensemble #%d", i%12)
		vc.Album = fmt.Sprintf("Batch 100 Ogg Edition Vol %d", i%6+1)
		vc.TrackNumber = fmt.Sprintf("%d", i)
		vc.AIArticle31Disclosure = DefaultVorbisArticle31Disclosure()
		vc.AIModelName = fmt.Sprintf("AVibe-Gen-Ogg-%d", (i%5)+1)
		vc.AIPrompt = fmt.Sprintf("Heterogeneous variation #%d with codec %s", i, codec)
		vc.AISeed = fmt.Sprintf("%d", 20000000+i*8191)
		vc.AIGenerator = "AVibe Batch Runner"
		vc.AIIsSynthetic = true
		vc.AICreationTimestamp = time.Now().UTC().Format(time.RFC3339)

		injected, err := InjectOggVorbisComment(canonical, vc)
		if err != nil {
			t.Fatalf("[%d/100] InjectOggVorbisComment failed: %v", i, err)
		}

		// Verify CRC on all pages
		newPages, err := ParseOggPages(injected)
		if err != nil {
			t.Fatalf("[%d/100] ParseOggPages on injected failed (CRC or formatting): %v", i, err)
		}

		extracted, err := ExtractOggVorbisComment(injected)
		if err != nil {
			t.Fatalf("[%d/100] ExtractOggVorbisComment failed: %v", i, err)
		}

		if extracted.Title != vc.Title || extracted.Artist != vc.Artist || extracted.TrackNumber != vc.TrackNumber {
			t.Fatalf("[%d/100] metadata roundtrip mismatch!", i)
		}
		if extracted.AIArticle31Disclosure != vc.AIArticle31Disclosure || !extracted.AIIsSynthetic {
			t.Fatalf("[%d/100] AI Article 31 disclosure mismatch!", i)
		}

		// Verify byte-perfect audio payload preservation
		var newAudioPayloads [][]byte
		for a := startAudioIdx; a < len(newPages); a++ {
			newAudioPayloads = append(newAudioPayloads, newPages[a].Payload)
		}

		if len(origAudioPayloads) != len(newAudioPayloads) {
			t.Fatalf("[%d/100] audio page count mismatch: orig %d, new %d", i, len(origAudioPayloads), len(newAudioPayloads))
		}
		for a := range origAudioPayloads {
			if !bytes.Equal(origAudioPayloads[a], newAudioPayloads[a]) {
				t.Fatalf("[%d/100] audio payload altered on page %d!", i, a+startAudioIdx)
			}
		}
	}

	t.Log("Successfully verified 100 heterogeneous Ogg Vorbis/Opus streams with Article 31 VorbisComment metadata and byte-perfect audio preservation.")
}
