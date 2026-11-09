package forensics

import (
	"math/rand"
	"testing"
)

func TestPlagiarismScorer_CleanTrackPass(t *testing.T) {
	scorer := NewPlagiarismScorer()
	r := rand.New(rand.NewSource(111))

	queryFPs := make([]uint32, 100)
	for i := range queryFPs {
		queryFPs[i] = r.Uint32()
	}

	refFPs := make([]uint32, 100)
	for i := range refFPs {
		refFPs[i] = r.Uint32()
	}

	catalog := map[string][]uint32{
		"catalog-ref-001": refFPs,
	}

	res, err := scorer.ScoreTrack(queryFPs, catalog)
	if err != nil {
		t.Fatalf("ScoreTrack failed: %v", err)
	}

	if res.PlagiarismDetected {
		t.Errorf("clean track was falsely flagged for plagiarism: max contiguous %f s", res.MaxContiguousMatchSec)
	}
	if res.RecommendedAction != "PASS" {
		t.Errorf("expected action 'PASS', got '%s'", res.RecommendedAction)
	}
}

func TestPlagiarismScorer_InfringingSample3SecondsFlag(t *testing.T) {
	scorer := NewPlagiarismScorer()
	r := rand.New(rand.NewSource(222))

	// Reference track: 200 frames (~9.3 seconds)
	refFPs := make([]uint32, 200)
	for i := range refFPs {
		refFPs[i] = r.Uint32()
	}

	// Query track: 200 frames
	queryFPs := make([]uint32, 200)
	for i := range queryFPs {
		queryFPs[i] = r.Uint32()
	}

	// Splice 4.0 seconds (4.0 * 21.5 = 86 frames) of reference track into query starting at frame 50
	spliceLen := 86
	for i := 0; i < spliceLen; i++ {
		queryFPs[50+i] = refFPs[20+i]
	}

	catalog := map[string][]uint32{
		"hit-song-001": refFPs,
	}

	res, err := scorer.ScoreTrack(queryFPs, catalog)
	if err != nil {
		t.Fatalf("ScoreTrack failed: %v", err)
	}

	// Verification check against Definition of Done: >= 3.0s contiguous flag
	if !res.PlagiarismDetected {
		t.Fatalf("DoD violated: 4.0s spliced sample was NOT flagged for plagiarism (max contiguous: %f s)", res.MaxContiguousMatchSec)
	}
	if res.MaxContiguousMatchSec < 3.0 {
		t.Errorf("expected max contiguous match >= 3.0s, got %f s", res.MaxContiguousMatchSec)
	}
	if res.RecommendedAction != "FLAG_REVIEW_GATE_4" {
		t.Errorf("expected action 'FLAG_REVIEW_GATE_4', got '%s'", res.RecommendedAction)
	}
	t.Logf("Plagiarism flagged with %f seconds contiguous matching segment", res.MaxContiguousMatchSec)

	jsonStr, err := res.ToJSON()
	if err != nil || len(jsonStr) == 0 {
		t.Fatalf("ToJSON failed: %v", err)
	}
}
