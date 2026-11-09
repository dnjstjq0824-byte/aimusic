// Package forensics provides diagonal correlation acoustic alignment,
// unauthorized sampling detection, and automated flagging of contiguous plagiarism segments (>= 3.0 seconds).
package forensics

import (
	"encoding/json"
	"math"
	"math/bits"
	"sort"
	"time"
)

// SampleMatchSegment records a specific time range where identical or cloned audio was detected.
type SampleMatchSegment struct {
	ReferenceTrackID string  `json:"reference_track_id"`
	QueryStartSec    float64 `json:"query_start_sec"`
	QueryEndSec      float64 `json:"query_end_sec"`
	DurationSec      float64 `json:"duration_sec"`
	RefStartSec      float64 `json:"ref_start_sec"`
	RefEndSec        float64 `json:"ref_end_sec"`
	AvgHammingDist   float64 `json:"avg_hamming_dist"`
}

// PlagiarismReport encapsulates detection results and Gate 4 compliance flags.
type PlagiarismReport struct {
	PlagiarismDetected    bool                 `json:"plagiarism_detected"`     // True if contiguous match >= 3.0s
	MaxContiguousMatchSec float64              `json:"max_contiguous_match_sec"`
	InfringingSegments    []SampleMatchSegment `json:"infringing_segments"`
	RecommendedAction     string               `json:"recommended_action"`      // "FLAG_REVIEW_GATE_4" or "PASS"
	DiagnosticSummary     string               `json:"diagnostic_summary"`
	ScoringDuration       time.Duration        `json:"scoring_duration"`
}

// PlagiarismScorer evaluates diagonal alignment and contiguous sample overlap.
type PlagiarismScorer struct {
	maxBitErrors     int
	fpsRate          float64 // Frames per second (approx 21.5 for Chromaprint 512 hop at 11025Hz)
	flagThresholdSec float64 // DoD threshold: >= 3.0 seconds
}

// NewPlagiarismScorer creates a plagiarism scorer.
func NewPlagiarismScorer() *PlagiarismScorer {
	return &PlagiarismScorer{
		maxBitErrors:     4,     // Max bit difference per sub-fingerprint
		fpsRate:          21.5,  // Standard Chromaprint subfingerprints per second
		flagThresholdSec: 3.0,   // 3.0 seconds contiguous match threshold
	}
}

// ScoreTrack evaluates a query track's subfingerprints against reference catalog fingerprints.
func (ps *PlagiarismScorer) ScoreTrack(queryFPs []uint32, refCatalog map[string][]uint32) (*PlagiarismReport, error) {
	start := time.Now()

	var allSegments []SampleMatchSegment
	var maxContiguousSec float64

	for refTrackID, refFPs := range refCatalog {
		segments := ps.findContiguousMatches(queryFPs, refFPs, refTrackID)
		for _, seg := range segments {
			allSegments = append(allSegments, seg)
			if seg.DurationSec > maxContiguousSec {
				maxContiguousSec = seg.DurationSec
			}
		}
	}

	sort.Slice(allSegments, func(i, j int) bool {
		return allSegments[i].DurationSec > allSegments[j].DurationSec
	})

	plagiarismDetected := maxContiguousSec >= ps.flagThresholdSec

	action := "PASS"
	summary := "No unauthorized sampling or contiguous copyright infringement detected."
	if plagiarismDetected {
		action = "FLAG_REVIEW_GATE_4"
		summary = "Infringement detected: audio contains contiguous matching segments exceeding 3.0 seconds."
	}

	elapsed := time.Since(start)

	return &PlagiarismReport{
		PlagiarismDetected:    plagiarismDetected,
		MaxContiguousMatchSec: math.Round(maxContiguousSec*10) / 10,
		InfringingSegments:    allSegments,
		RecommendedAction:     action,
		DiagnosticSummary:     summary,
		ScoringDuration:       elapsed,
	}, nil
}

// findContiguousMatches searches along diagonal time offsets (offset = qFrame - rFrame).
func (ps *PlagiarismScorer) findContiguousMatches(queryFPs, refFPs []uint32, refTrackID string) []SampleMatchSegment {
	qLen := len(queryFPs)
	rLen := len(refFPs)
	if qLen == 0 || rLen == 0 {
		return nil
	}

	// Diagonal offsets map: offset -> list of matching query frame indices
	offsets := make(map[int][]int)

	for q := 0; q < qLen; q++ {
		qFP := queryFPs[q]
		for r := 0; r < rLen; r++ {
			rFP := refFPs[r]
			dist := bits.OnesCount32(qFP ^ rFP)
			if dist <= ps.maxBitErrors {
				offset := q - r
				offsets[offset] = append(offsets[offset], q)
			}
		}
	}

	var segments []SampleMatchSegment

	// In each diagonal offset, find contiguous sequences of matching frames
	for offset, frames := range offsets {
		if len(frames) < 10 {
			continue
		}
		sort.Ints(frames)

		runStart := frames[0]
		runLen := 1
		var sumDist int

		for i := 1; i < len(frames); i++ {
			// If adjacent frame in diagonal sequence (allowing 1 frame jitter)
			if frames[i]-frames[i-1] <= 2 {
				runLen++
			} else {
				// Evaluate run
				durSec := float64(runLen) / ps.fpsRate
				if durSec >= 1.0 { // Minimum 1 second run to register
					qStartSec := float64(runStart) / ps.fpsRate
					qEndSec := qStartSec + durSec
					rStartSec := float64(runStart-offset) / ps.fpsRate
					rEndSec := rStartSec + durSec

					segments = append(segments, SampleMatchSegment{
						ReferenceTrackID: refTrackID,
						QueryStartSec:    math.Round(qStartSec*10) / 10,
						QueryEndSec:      math.Round(qEndSec*10) / 10,
						DurationSec:      math.Round(durSec*10) / 10,
						RefStartSec:      math.Round(rStartSec*10) / 10,
						RefEndSec:        math.Round(rEndSec*10) / 10,
						AvgHammingDist:   math.Round(float64(sumDist)/float64(runLen)*10) / 10,
					})
				}
				runStart = frames[i]
				runLen = 1
			}
		}

		// Trailing run
		durSec := float64(runLen) / ps.fpsRate
		if durSec >= 1.0 {
			qStartSec := float64(runStart) / ps.fpsRate
			qEndSec := qStartSec + durSec
			rStartSec := float64(runStart-offset) / ps.fpsRate
			rEndSec := rStartSec + durSec

			segments = append(segments, SampleMatchSegment{
				ReferenceTrackID: refTrackID,
				QueryStartSec:    math.Round(qStartSec*10) / 10,
				QueryEndSec:      math.Round(qEndSec*10) / 10,
				DurationSec:      math.Round(durSec*10) / 10,
				RefStartSec:      math.Round(rStartSec*10) / 10,
				RefEndSec:        math.Round(rEndSec*10) / 10,
				AvgHammingDist:   0.0,
			})
		}
	}

	return segments
}

// ToJSON serializes plagiarism report.
func (r *PlagiarismReport) ToJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
