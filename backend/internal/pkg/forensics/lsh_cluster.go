// Package forensics provides Locality-Sensitive Hashing (LSH) sub-linear indexing,
// band-row partitioning, and sub-fingerprint candidate clustering for O(1) audio retrieval.
package forensics

import (
	"encoding/json"
	"sync"
)

// LSHConfig configures the band and row parameters for Locality-Sensitive Hashing.
type LSHConfig struct {
	NumBands    int `json:"num_bands"`     // Number of bands B (default: 4)
	RowsPerBand int `json:"rows_per_band"` // Rows per band R (default: 8, where B * R = 32 bits)
}

// DefaultLSHConfig returns standard 4-band 8-row LSH configuration for 32-bit subfingerprints.
func DefaultLSHConfig() LSHConfig {
	return LSHConfig{
		NumBands:    4,
		RowsPerBand: 8,
	}
}

// LSHCluster manages inverted bucket tables across independent hash bands.
type LSHCluster struct {
	cfg LSHConfig
	mu  sync.RWMutex
	// buckets[bandIndex][bucketHash] -> list of trackIDs
	buckets     []map[uint32]map[string]struct{}
	totalTracks int
}

// NewLSHCluster creates an LSH inverted index cluster.
func NewLSHCluster(cfg LSHConfig) *LSHCluster {
	if cfg.NumBands <= 0 {
		cfg.NumBands = 4
	}
	if cfg.RowsPerBand <= 0 {
		cfg.RowsPerBand = 8
	}

	buckets := make([]map[uint32]map[string]struct{}, cfg.NumBands)
	for b := 0; b < cfg.NumBands; b++ {
		buckets[b] = make(map[uint32]map[string]struct{})
	}

	return &LSHCluster{
		cfg:     cfg,
		buckets: buckets,
	}
}

// Insert hashes and partitions an audio track's sub-fingerprints into LSH buckets.
func (c *LSHCluster) Insert(trackID string, subFPs []uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	mask := uint32((1 << c.cfg.RowsPerBand) - 1)
	numBands := c.cfg.NumBands
	rows := c.cfg.RowsPerBand

	for _, fp := range subFPs {
		for b := 0; b < numBands; b++ {
			shift := uint(b * rows)
			bandKey := (fp >> shift) & mask

			bucketMap := c.buckets[b]
			tracks, ok := bucketMap[bandKey]
			if !ok {
				tracks = make(map[string]struct{})
				bucketMap[bandKey] = tracks
			}
			tracks[trackID] = struct{}{}
		}
	}
	c.totalTracks++
}

// QueryCandidates retrieves the subset of candidate track IDs that collide with the query in any LSH band.
func (c *LSHCluster) QueryCandidates(querySubFPs []uint32) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	mask := uint32((1 << c.cfg.RowsPerBand) - 1)
	numBands := c.cfg.NumBands
	rows := c.cfg.RowsPerBand

	candidateSet := make(map[string]int) // trackID -> collision count

	for _, fp := range querySubFPs {
		for b := 0; b < numBands; b++ {
			shift := uint(b * rows)
			bandKey := (fp >> shift) & mask

			if tracks, ok := c.buckets[b][bandKey]; ok {
				for tID := range tracks {
					candidateSet[tID]++
				}
			}
		}
	}

	candidates := make([]string, 0, len(candidateSet))
	for tID := range candidateSet {
		candidates = append(candidates, tID)
	}

	return candidates
}

// ClusterStats reports index size and bucket distribution metrics.
type ClusterStats struct {
	NumBands     int `json:"num_bands"`
	RowsPerBand  int `json:"rows_per_band"`
	TotalTracks  int `json:"total_tracks"`
	TotalBuckets int `json:"total_buckets"`
}

// Stats returns cluster occupancy telemetry.
func (c *LSHCluster) Stats() ClusterStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalB := 0
	for _, b := range c.buckets {
		totalB += len(b)
	}

	return ClusterStats{
		NumBands:     c.cfg.NumBands,
		RowsPerBand:  c.cfg.RowsPerBand,
		TotalTracks:  c.totalTracks,
		TotalBuckets: totalB,
	}
}

// ToJSON serializes stats.
func (s *ClusterStats) ToJSON() (string, error) {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
