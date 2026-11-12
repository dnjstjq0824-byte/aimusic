// Package forensics provides high-performance Hamming distance indexing,
// bitwise similarity matching, and sub-50ms candidate search for millions of acoustic fingerprints.
package forensics

import (
	"encoding/json"
	"math"
	"math/bits"
	"sort"
	"sync"
	"time"
)

// FingerprintMatch represents a detected similarity match for a track.
type FingerprintMatch struct {
	TrackID        string  `json:"track_id"`
	MatchingFrames int     `json:"matching_frames"`
	TotalFrames    int     `json:"total_frames"`
	SimilarityPct  float64 `json:"similarity_pct"` // 0.0 to 100.0%
	AvgHammingDist float64 `json:"avg_hamming_dist"`
}

// HammingSearchResult encapsulates the ranked matching results and search performance.
type HammingSearchResult struct {
	QueryFrames    int                `json:"query_frames"`
	TotalIndexed   int                `json:"total_indexed_subfingerprints"`
	SearchDuration time.Duration      `json:"search_duration"`
	Matches        []FingerprintMatch `json:"matches"`
}

// InvertedSubFPIndex maps 32-bit sub-fingerprints (or their upper/lower 16-bit keys)
// to track IDs and time offsets for $O(1)$ candidate retrieval.
type InvertedSubFPIndex struct {
	mu sync.RWMutex
	// Direct bucket lookup using 16-bit segment hashing (Pigeonhole principle for Hamming search)
	// Any 32-bit integer with Hamming distance <= 1 from a query must match exactly in either the top 16 bits or bottom 16 bits!
	topBuckets    map[uint16][]trackOffset
	bottomBuckets map[uint16][]trackOffset
	trackLengths  map[string]int
	totalSubFPs   int
}

type trackOffset struct {
	trackID string
	frame   int
	subFP   uint32
}

// NewInvertedSubFPIndex instantiates the high-speed index.
func NewInvertedSubFPIndex() *InvertedSubFPIndex {
	return &InvertedSubFPIndex{
		topBuckets:    make(map[uint16][]trackOffset),
		bottomBuckets: make(map[uint16][]trackOffset),
		trackLengths:  make(map[string]int),
	}
}

// IndexTrack inserts all sub-fingerprints of an audio track into the inverted index.
func (idx *InvertedSubFPIndex) IndexTrack(trackID string, subFPs []uint32) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.trackLengths[trackID] = len(subFPs)

	for frame, fp := range subFPs {
		top := uint16(fp >> 16)
		bottom := uint16(fp & 0xFFFF)

		to := trackOffset{
			trackID: trackID,
			frame:   frame,
			subFP:   fp,
		}

		idx.topBuckets[top] = append(idx.topBuckets[top], to)
		idx.bottomBuckets[bottom] = append(idx.bottomBuckets[bottom], to)
		idx.totalSubFPs++
	}
}

// SearchCandidates queries the index for matching tracks within maxHammingDist (default: 4 bits).
func (idx *InvertedSubFPIndex) SearchCandidates(querySubFPs []uint32, maxHammingDist int) (*HammingSearchResult, error) {
	start := time.Now()
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if maxHammingDist <= 0 {
		maxHammingDist = 4 // Default max 4 bit errors per 32-bit frame (~87.5% bit similarity)
	}

	type matchStats struct {
		matchingFrames int
		sumHammingDist int
	}

	trackHits := make(map[string]*matchStats)

	for _, qFP := range querySubFPs {
		top := uint16(qFP >> 16)
		bottom := uint16(qFP & 0xFFFF)

		// Set of already inspected candidates for this query frame to avoid duplicate checks
		inspected := make(map[string]struct{})

		checkCandidate := func(cand trackOffset) {
			if _, exists := inspected[cand.trackID]; exists {
				return
			}
			dist := bits.OnesCount32(qFP ^ cand.subFP)
			if dist <= maxHammingDist {
				inspected[cand.trackID] = struct{}{}
				stats, ok := trackHits[cand.trackID]
				if !ok {
					stats = &matchStats{}
					trackHits[cand.trackID] = stats
				}
				stats.matchingFrames++
				stats.sumHammingDist += dist
			}
		}

		// Inspect matching buckets
		if cands, ok := idx.topBuckets[top]; ok {
			for _, c := range cands {
				checkCandidate(c)
			}
		}
		if cands, ok := idx.bottomBuckets[bottom]; ok {
			for _, c := range cands {
				checkCandidate(c)
			}
		}
	}

	// Aggregate and rank matches
	matches := make([]FingerprintMatch, 0, len(trackHits))
	queryLen := len(querySubFPs)

	for trackID, stats := range trackHits {
		trackLen := idx.trackLengths[trackID]
		refLen := queryLen
		if trackLen > 0 && trackLen < refLen {
			refLen = trackLen
		}
		if refLen == 0 {
			refLen = 1
		}

		simPct := (float64(stats.matchingFrames) / float64(refLen)) * 100.0
		if simPct > 100.0 {
			simPct = 100.0
		}

		avgDist := 0.0
		if stats.matchingFrames > 0 {
			avgDist = float64(stats.sumHammingDist) / float64(stats.matchingFrames)
		}

		matches = append(matches, FingerprintMatch{
			TrackID:        trackID,
			MatchingFrames: stats.matchingFrames,
			TotalFrames:    trackLen,
			SimilarityPct:  math.Round(simPct*10) / 10,
			AvgHammingDist: math.Round(avgDist*10) / 10,
		})
	}

	// Sort descending by similarity percentage
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].SimilarityPct > matches[j].SimilarityPct
	})

	elapsed := time.Since(start)

	return &HammingSearchResult{
		QueryFrames:    queryLen,
		TotalIndexed:   idx.totalSubFPs,
		SearchDuration: elapsed,
		Matches:        matches,
	}, nil
}

// ToJSON serializes search result.
func (r *HammingSearchResult) ToJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
