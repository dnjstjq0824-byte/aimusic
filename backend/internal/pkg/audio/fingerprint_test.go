package audio

import (
	"errors"
	"math"
	"math/rand"
	"testing"
)

func generateMelody(freqs []float64, sampleRate int, noteDurationSec float64, amp float64) *PCMBuffer {
	noteSamples := int(float64(sampleRate) * noteDurationSec)
	totalSamples := noteSamples * len(freqs)
	left := make([]float64, totalSamples)
	right := make([]float64, totalSamples)

	for noteIdx, f := range freqs {
		start := noteIdx * noteSamples
		for i := 0; i < noteSamples; i++ {
			t := float64(i) / float64(sampleRate)
			v := amp * math.Sin(2.0*math.Pi*f*t)
			left[start+i] = v
			right[start+i] = v
		}
	}

	return &PCMBuffer{
		Channels:   2,
		SampleRate: sampleRate,
		NumSamples: totalSamples,
		Samples:    [][]float64{left, right},
	}
}

func TestFingerprint_ExactCanonicalPCMMatch(t *testing.T) {
	engine := NewFingerprintEngine()
	notes := []float64{440.0, 554.37, 659.25, 880.0}
	buf1 := generateMelody(notes, 48000, 0.5, 0.7)
	buf2 := generateMelody(notes, 48000, 0.5, 0.7)

	fp1, err := engine.GenerateFingerprint(buf1)
	if err != nil {
		t.Fatalf("GenerateFingerprint 1 failed: %v", err)
	}
	fp2, err := engine.GenerateFingerprint(buf2)
	if err != nil {
		t.Fatalf("GenerateFingerprint 2 failed: %v", err)
	}

	if fp1.CanonicalPCMHashSHA256 != fp2.CanonicalPCMHashSHA256 {
		t.Errorf("Canonical PCM hash mismatch for identical audio: %s vs %s",
			fp1.CanonicalPCMHashSHA256, fp2.CanonicalPCMHashSHA256)
	}
	if fp1.AcousticHashSHA256 != fp2.AcousticHashSHA256 {
		t.Errorf("Acoustic hash mismatch for identical audio: %s vs %s",
			fp1.AcousticHashSHA256, fp2.AcousticHashSHA256)
	}

	match := CompareFingerprints(fp1, fp2)
	t.Logf("Exact match test: ExactPCM = %v, Similarity = %.4f", match.ExactPCM, match.Similarity)
	if !match.ExactPCM || match.Similarity != 1.0 || !match.IsDuplicate {
		t.Errorf("Expected exact PCM match and 1.0 similarity, got %+v", match)
	}
}

func TestFingerprint_VolumeInvariance(t *testing.T) {
	// Compare full amplitude melody (0.8) vs attenuated melody (0.2)
	engine := NewFingerprintEngine()
	notes := []float64{330.0, 392.0, 493.88, 587.33, 659.25}
	bufLoud := generateMelody(notes, 48000, 0.4, 0.8)
	bufQuiet := generateMelody(notes, 48000, 0.4, 0.2)

	fpLoud, err := engine.GenerateFingerprint(bufLoud)
	if err != nil {
		t.Fatalf("Loud fp failed: %v", err)
	}
	fpQuiet, err := engine.GenerateFingerprint(bufQuiet)
	if err != nil {
		t.Fatalf("Quiet fp failed: %v", err)
	}

	// Canonical PCM hash should differ because sample values are scaled
	if fpLoud.CanonicalPCMHashSHA256 == fpQuiet.CanonicalPCMHashSHA256 {
		t.Errorf("Canonical PCM hash should differ for different volume levels")
	}

	// But acoustic perceptual similarity should be very high (differential subband energy removes scaling)
	match := CompareFingerprints(fpLoud, fpQuiet)
	t.Logf("Volume invariance test: Similarity = %.4f, IsDuplicate = %v, IsDerivative = %v",
		match.Similarity, match.IsDuplicate, match.IsDerivative)

	if match.Similarity < 0.95 {
		t.Errorf("Expected acoustic similarity >= 0.95 under volume scaling, got %.4f", match.Similarity)
	}
}

func TestFingerprint_TimeShiftInvariance(t *testing.T) {
	// Original melody vs melody with 150ms of silence prepended
	engine := NewFingerprintEngine()
	notes := []float64{440.0, 493.88, 523.25, 587.33, 659.25}
	origBuf := generateMelody(notes, 48000, 0.4, 0.7)

	leadSilenceSamples := int(48000 * 0.15) // 150ms
	shiftedSamples := origBuf.NumSamples + leadSilenceSamples
	leftShifted := make([]float64, shiftedSamples)
	rightShifted := make([]float64, shiftedSamples)
	copy(leftShifted[leadSilenceSamples:], origBuf.Samples[0])
	copy(rightShifted[leadSilenceSamples:], origBuf.Samples[1])

	shiftedBuf := &PCMBuffer{
		Channels:   2,
		SampleRate: 48000,
		NumSamples: shiftedSamples,
		Samples:    [][]float64{leftShifted, rightShifted},
	}

	fpOrig, err := engine.GenerateFingerprint(origBuf)
	if err != nil {
		t.Fatalf("Orig fp failed: %v", err)
	}
	fpShifted, err := engine.GenerateFingerprint(shiftedBuf)
	if err != nil {
		t.Fatalf("Shifted fp failed: %v", err)
	}

	match := CompareFingerprints(fpOrig, fpShifted)
	t.Logf("Time-shift test (150ms offset): Similarity = %.4f, IsDerivative = %v, Dist = %d, TotalBits = %d",
		match.Similarity, match.IsDerivative, match.HammingDistance, match.TotalBits)

	if match.Similarity < 0.80 {
		t.Errorf("Expected high similarity (>= 0.80) with offset compensation, got %.4f", match.Similarity)
	}
}

func TestFingerprint_DifferentAudioUncorrelated(t *testing.T) {
	engine := NewFingerprintEngine()
	// Track A: Low frequency arpeggio
	notesA := []float64{220.0, 277.18, 329.63, 440.0}
	bufA := generateMelody(notesA, 48000, 0.5, 0.7)

	// Track B: High frequency tone cluster
	notesB := []float64{1760.0, 2093.0, 2637.0, 3136.0}
	bufB := generateMelody(notesB, 48000, 0.5, 0.7)

	fpA, _ := engine.GenerateFingerprint(bufA)
	fpB, _ := engine.GenerateFingerprint(bufB)

	match := CompareFingerprints(fpA, fpB)
	t.Logf("Uncorrelated tracks comparison: Similarity = %.4f, IsDerivative = %v, IsDuplicate = %v",
		match.Similarity, match.IsDerivative, match.IsDuplicate)

	if match.Similarity > 0.65 {
		t.Errorf("Expected low correlation (< 0.65) for different audio, got %.4f", match.Similarity)
	}
	if match.IsDerivative || match.IsDuplicate {
		t.Errorf("False positive match detected between completely different tracks")
	}
}

func TestFingerprint_MerkleTree(t *testing.T) {
	chunks := [][]byte{
		[]byte("HLS_AUDIO_CHUNK_0001_DATA_PAYLOAD"),
		[]byte("HLS_AUDIO_CHUNK_0002_DATA_PAYLOAD"),
		[]byte("HLS_AUDIO_CHUNK_0003_DATA_PAYLOAD"),
		[]byte("HLS_AUDIO_CHUNK_0004_DATA_PAYLOAD"),
		[]byte("HLS_AUDIO_CHUNK_0005_DATA_PAYLOAD"),
	}

	tree1, err := ComputeChunkMerkleTree(chunks)
	if err != nil {
		t.Fatalf("ComputeChunkMerkleTree failed: %v", err)
	}
	tree2, err := ComputeChunkMerkleTree(chunks)
	if err != nil {
		t.Fatalf("ComputeChunkMerkleTree 2 failed: %v", err)
	}

	if tree1.RootHash != tree2.RootHash {
		t.Errorf("Merkle root hash not deterministic: %s vs %s", tree1.RootHash, tree2.RootHash)
	}
	if len(tree1.LeafHashes) != len(chunks) {
		t.Errorf("Leaf hash count mismatch: expected %d, got %d", len(chunks), len(tree1.LeafHashes))
	}

	// Tamper test: alter 1 byte in chunk 2
	tamperedChunks := make([][]byte, len(chunks))
	for i := range chunks {
		tamperedChunks[i] = make([]byte, len(chunks[i]))
		copy(tamperedChunks[i], chunks[i])
	}
	tamperedChunks[1][0] ^= 0xFF

	tamperedTree, err := ComputeChunkMerkleTree(tamperedChunks)
	if err != nil {
		t.Fatalf("Tampered tree computation failed: %v", err)
	}

	if tamperedTree.RootHash == tree1.RootHash {
		t.Errorf("Merkle tree failed to detect tampered chunk payload")
	}
	t.Logf("Merkle Tree: Valid Root = %s, Tampered Root = %s", tree1.RootHash, tamperedTree.RootHash)
}

func TestFingerprint_EdgeCases(t *testing.T) {
	engine := NewFingerprintEngine()

	// 1. Nil buffer
	_, err := engine.GenerateFingerprint(nil)
	if !errors.Is(err, ErrNilFingerprintBuffer) {
		t.Errorf("Expected ErrNilFingerprintBuffer, got %v", err)
	}

	// 2. Empty buffer
	emptyBuf := &PCMBuffer{Channels: 0, SampleRate: 48000, NumSamples: 0, Samples: nil}
	_, err = engine.GenerateFingerprint(emptyBuf)
	if !errors.Is(err, ErrEmptyAudioBuffer) {
		t.Errorf("Expected ErrEmptyAudioBuffer, got %v", err)
	}

	// 3. Short buffer (< 2048 samples) should generate valid canonical hash and not panic
	shortBuf := generateMelody([]float64{440.0}, 48000, 0.02, 0.5) // ~960 samples
	fpShort, err := engine.GenerateFingerprint(shortBuf)
	if err != nil {
		t.Fatalf("Short buffer fingerprint failed: %v", err)
	}
	if len(fpShort.CanonicalPCMHashSHA256) != 64 {
		t.Errorf("Expected 64-char hex SHA256, got len %d", len(fpShort.CanonicalPCMHashSHA256))
	}

	// 4. Empty chunks for Merkle tree
	_, err = ComputeChunkMerkleTree(nil)
	if !errors.Is(err, ErrEmptyChunkSlice) {
		t.Errorf("Expected ErrEmptyChunkSlice, got %v", err)
	}
}

func TestFingerprint100HeterogeneousTracks(t *testing.T) {
	engine := NewFingerprintEngine()
	r := rand.New(rand.NewSource(42))

	count := 0
	for i := 0; i < 100; i++ {
		count++
		numNotes := r.Intn(4) + 2
		freqs := make([]float64, numNotes)
		for n := 0; n < numNotes; n++ {
			freqs[n] = 200.0 + r.Float64()*2000.0
		}
		duration := 0.2 + r.Float64()*0.4
		amp := 0.1 + r.Float64()*0.8
		channels := (i % 2) + 1

		sampleRate := 48000
		if i%3 == 0 {
			sampleRate = 44100
		}

		numSamples := int(float64(sampleRate) * duration * float64(numNotes))
		samples := make([][]float64, channels)
		for ch := 0; ch < channels; ch++ {
			samples[ch] = make([]float64, numSamples)
			noteLen := numSamples / numNotes
			for nIdx, f := range freqs {
				start := nIdx * noteLen
				end := start + noteLen
				if end > numSamples {
					end = numSamples
				}
				for s := start; s < end; s++ {
					tSec := float64(s) / float64(sampleRate)
					samples[ch][s] = amp * math.Sin(2.0*math.Pi*f*tSec)
				}
			}
		}

		buf := &PCMBuffer{
			Channels:   channels,
			SampleRate: sampleRate,
			NumSamples: numSamples,
			Samples:    samples,
		}

		fp, err := engine.GenerateFingerprint(buf)
		if err != nil {
			t.Fatalf("Track %d failed to generate fingerprint: %v", count, err)
		}

		if len(fp.CanonicalPCMHashSHA256) != 64 {
			t.Fatalf("Track %d: invalid canonical PCM hash len %d", count, len(fp.CanonicalPCMHashSHA256))
		}
		if len(fp.AcousticHashSHA256) != 64 {
			t.Fatalf("Track %d: invalid acoustic hash len %d", count, len(fp.AcousticHashSHA256))
		}
		if fp.TotalFrames <= 0 {
			t.Fatalf("Track %d: non-positive frame count %d", count, fp.TotalFrames)
		}
	}

	t.Logf("Successfully verified %d heterogeneous track fingerprints with 0 errors.", count)
}
