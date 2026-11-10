package forensics

import (
	"testing"
)

func TestReversePlaybackMatcher_NormalForwardPlayback(t *testing.T) {
	matcher := NewReversePlaybackMatcher()
	sampleRate := 48000

	ref := synthesizeAcousticSong(sampleRate, 2.5)
	query := synthesizeAcousticSong(sampleRate, 2.5)

	res, err := matcher.Match(query, ref)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	if !res.Matched {
		t.Errorf("expected normal playback to match, got false")
	}
	if res.IsReversedPlayback {
		t.Errorf("expected isReversed=false for normal playback, got true")
	}
	if res.ForwardSimilarity < 0.95 {
		t.Errorf("expected forward similarity >= 0.95, got %f", res.ForwardSimilarity)
	}
	t.Logf("Forward match: FwdSim=%.2f%%, InvSim=%.2f%%, IsReversed=%v",
		res.ForwardSimilarity*100, res.InvertedSimilarity*100, res.IsReversedPlayback)
}

func TestReversePlaybackMatcher_EvasionAttemptReversedAudio(t *testing.T) {
	matcher := NewReversePlaybackMatcher()
	sampleRate := 48000

	ref := synthesizeAcousticSong(sampleRate, 2.5)

	// Infringer reverses audio in time to evade copyright filter
	reversedQuery, err := InvertBuffer(ref)
	if err != nil {
		t.Fatalf("InvertBuffer failed: %v", err)
	}

	res, err := matcher.Match(reversedQuery, ref)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	// Verification against DoD: Reverse playback evasion intercepted and matched
	if !res.Matched {
		t.Fatalf("DoD violated: reverse playback query failed to match")
	}
	if !res.IsReversedPlayback {
		t.Fatalf("DoD violated: failed to flag is_reversed_playback=true")
	}
	if !res.EvasionAttemptDetected {
		t.Fatalf("DoD violated: failed to flag evasion_attempt_detected=true")
	}
	if res.InvertedSimilarity < 0.95 {
		t.Errorf("expected inverted similarity >= 0.95, got %f", res.InvertedSimilarity)
	}
	t.Logf("Reverse evasion intercepted: FwdSim=%.2f%%, InvSim=%.2f%%, EvasionFlag=%v",
		res.ForwardSimilarity*100, res.InvertedSimilarity*100, res.EvasionAttemptDetected)
}

func TestReversePlaybackMatcher_UnrelatedAudioPass(t *testing.T) {
	matcher := NewReversePlaybackMatcher()
	sampleRate := 48000

	ref := synthesizeAcousticSong(sampleRate, 2.5)
	unrelated := synthesizeMotif(sampleRate, 0, 2.5)

	res, err := matcher.Match(unrelated, ref)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	if res.Matched {
		t.Errorf("unrelated audio was falsely matched: fwd=%f, inv=%f", res.ForwardSimilarity, res.InvertedSimilarity)
	}
	if res.EvasionAttemptDetected {
		t.Errorf("falsely flagged evasion attempt for unrelated audio")
	}
	t.Logf("Unrelated audio clean: FwdSim=%.2f%%, InvSim=%.2f%%", res.ForwardSimilarity*100, res.InvertedSimilarity*100)
}
