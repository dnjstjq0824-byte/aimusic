package forensics

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestInvertedSubFPIndex_SearchSpeedUnder50ms(t *testing.T) {
	index := NewInvertedSubFPIndex()
	r := rand.New(rand.NewSource(12345))

	// Index 500 catalog tracks, each with 100 sub-fingerprints = 50,000 indexed records
	numTracks := 500
	framesPerTrack := 100

	var targetTrackID string
	var targetTrackFPs []uint32

	for i := 0; i < numTracks; i++ {
		trackID := fmt.Sprintf("track-catalog-%04d", i)
		fps := make([]uint32, framesPerTrack)
		for f := 0; f < framesPerTrack; f++ {
			fps[f] = r.Uint32()
		}
		index.IndexTrack(trackID, fps)

		if i == 250 {
			targetTrackID = trackID
			targetTrackFPs = fps
		}
	}

	// Create query with slight bit perturbations on target track
	queryFPs := make([]uint32, framesPerTrack)
	for f := 0; f < framesPerTrack; f++ {
		// Flip 1 bit occasionally
		orig := targetTrackFPs[f]
		if f%2 == 0 {
			orig ^= 0x01 // flip bit 0
		}
		queryFPs[f] = orig
	}

	start := time.Now()
	res, err := index.SearchCandidates(queryFPs, 4)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("SearchCandidates failed: %v", err)
	}

	// Verification check against Definition of Done: < 50ms
	if elapsed >= 50*time.Millisecond {
		t.Fatalf("DoD violated: search took %v (limit: 50ms)", elapsed)
	}
	t.Logf("Hamming index search completed in %v across %d records (DoD limit: < 50ms)", elapsed, res.TotalIndexed)

	if len(res.Matches) == 0 {
		t.Fatalf("expected matches, got 0")
	}

	// Rank 1 match MUST be the target track
	topMatch := res.Matches[0]
	if topMatch.TrackID != targetTrackID {
		t.Errorf("expected Rank 1 match to be %s, got %s", targetTrackID, topMatch.TrackID)
	}
	if topMatch.SimilarityPct < 90.0 {
		t.Errorf("expected top match similarity >= 90%%, got %f%%", topMatch.SimilarityPct)
	}

	jsonStr, err := res.ToJSON()
	if err != nil || len(jsonStr) == 0 {
		t.Fatalf("ToJSON failed: %v", err)
	}
}
