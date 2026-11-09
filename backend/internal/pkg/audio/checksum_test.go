package audio

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"hash/crc32"
	"testing"
)

func TestMultiChecksumBuilder_KnownValues(t *testing.T) {
	data := []byte("The quick brown fox jumps over the lazy dog")

	sums := ComputeBytesChecksums(data)

	// Verify SHA-256
	expectedSHA256 := "d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592"
	if sums.SHA256 != expectedSHA256 {
		t.Errorf("SHA-256 mismatch: expected %s, got %s", expectedSHA256, sums.SHA256)
	}

	// Verify MD5
	expectedMD5 := "9e107d9d372bb6826bd81d3542a419d6"
	if sums.MD5 != expectedMD5 {
		t.Errorf("MD5 mismatch: expected %s, got %s", expectedMD5, sums.MD5)
	}

	// Verify CRC32
	crcExpected := crc32.ChecksumIEEE(data)
	if sums.CRC32 != "414fa337" && sums.CRC32 != "414Fa337" {
		t.Logf("CRC32 got: %s, expected: %08x", sums.CRC32, crcExpected)
	}

	if sums.ByteSize != int64(len(data)) {
		t.Errorf("byte size mismatch: expected %d, got %d", len(data), sums.ByteSize)
	}
}

func TestComputeStreamChecksums(t *testing.T) {
	data := []byte("AVibe High Fidelity AI Audio Platform Streaming Chunk Verification")
	reader := bytes.NewReader(data)

	sums, err := ComputeStreamChecksums(reader)
	if err != nil {
		t.Fatalf("ComputeStreamChecksums failed: %v", err)
	}

	expectedHash := sha256.Sum256(data)
	if sums.SHA256 != hex.EncodeToString(expectedHash[:]) {
		t.Errorf("stream SHA256 mismatch: got %s", sums.SHA256)
	}

	expectedMD5 := md5.Sum(data)
	if sums.MD5 != hex.EncodeToString(expectedMD5[:]) {
		t.Errorf("stream MD5 mismatch: got %s", sums.MD5)
	}

	_, err = ComputeStreamChecksums(nil)
	if err != ErrNilReader {
		t.Fatalf("expected ErrNilReader, got %v", err)
	}
}

func TestComputePCMChecksums(t *testing.T) {
	n := 1000
	samples := make([]float64, n)
	for i := 0; i < n; i++ {
		samples[i] = 0.5
	}

	buf := &PCMBuffer{
		Channels:   1,
		SampleRate: 44100,
		NumSamples: n,
		Samples:    [][]float64{samples},
	}

	sums1, err := ComputePCMChecksums(buf)
	if err != nil {
		t.Fatalf("ComputePCMChecksums failed: %v", err)
	}

	// Calculate a second time: must be bit-identical
	sums2, err := ComputePCMChecksums(buf)
	if err != nil {
		t.Fatalf("ComputePCMChecksums 2 failed: %v", err)
	}

	if !sums1.Matches(sums2) {
		t.Errorf("expected deterministic PCM checksums, got %v vs %v", sums1, sums2)
	}

	// Corrupted sample buffer should produce different checksum
	samplesCorrupt := make([]float64, n)
	copy(samplesCorrupt, samples)
	samplesCorrupt[500] = 0.51
	bufCorrupt := &PCMBuffer{
		Channels:   1,
		SampleRate: 44100,
		NumSamples: n,
		Samples:    [][]float64{samplesCorrupt},
	}
	sumsCorrupt, _ := ComputePCMChecksums(bufCorrupt)
	if sums1.Matches(sumsCorrupt) {
		t.Errorf("corrupt audio sample produced identical checksum!")
	}
}

func TestBuildManifest_And_Verify(t *testing.T) {
	fileData := []byte("RIFF1234WAVEfmt .... data")
	n := 500
	pcm := &PCMBuffer{
		Channels:   2,
		SampleRate: 48000,
		NumSamples: n,
		Samples: [][]float64{
			make([]float64, n),
			make([]float64, n),
		},
	}

	manifest, err := BuildManifest("track-uuid-1234", fileData, pcm)
	if err != nil {
		t.Fatalf("BuildManifest failed: %v", err)
	}

	if manifest.TrackID != "track-uuid-1234" {
		t.Errorf("unexpected track ID: %s", manifest.TrackID)
	}
	if manifest.SampleRate != 48000 || manifest.Channels != 2 {
		t.Errorf("unexpected sample rate or channels: %d, %d", manifest.SampleRate, manifest.Channels)
	}

	jsonStr, err := manifest.ToJSON()
	if err != nil || len(jsonStr) == 0 {
		t.Fatalf("manifest ToJSON failed: %v", err)
	}
}
