package forensics

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestLSHCluster_InsertAndCandidateRetrieval(t *testing.T) {
	cluster := NewLSHCluster(DefaultLSHConfig())
	r := rand.New(rand.NewSource(999))

	numTracks := 100
	framesPerTrack := 50

	var targetTrackID string
	var targetTrackFPs []uint32

	for i := 0; i < numTracks; i++ {
		tID := fmt.Sprintf("track-lsh-%03d", i)
		fps := make([]uint32, framesPerTrack)
		for f := 0; f < framesPerTrack; f++ {
			fps[f] = r.Uint32()
		}
		cluster.Insert(tID, fps)

		if i == 42 {
			targetTrackID = tID
			targetTrackFPs = fps
		}
	}

	// Query with target track sub-fingerprints
	candidates := cluster.QueryCandidates(targetTrackFPs[:10])

	found := false
	for _, cID := range candidates {
		if cID == targetTrackID {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("target track %s was not returned in LSH candidate set", targetTrackID)
	}

	// Verify stats
	stats := cluster.Stats()
	if stats.TotalTracks != 100 {
		t.Errorf("expected 100 total tracks, got %d", stats.TotalTracks)
	}
	if stats.TotalBuckets == 0 {
		t.Errorf("expected non-zero total buckets")
	}

	jsonStr, err := stats.ToJSON()
	if err != nil || len(jsonStr) == 0 {
		t.Fatalf("ToJSON failed: %v", err)
	}
}
