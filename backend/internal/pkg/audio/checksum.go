// Package audio provides simultaneous streaming multi-checksum calculation
// (SHA-256, SHA-512, MD5, CRC32, Adler-32), bit-identical audio provenance verification,
// and tamper detection for audio assets and transcoded outputs.
package audio

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"hash/adler32"
	"hash/crc32"
	"io"
	"math"
	"time"
)

var (
	ErrNilReader        = errors.New("input stream reader is nil")
	ErrMismatchChecksum = errors.New("audio checksum mismatch")
)

// AudioChecksums encapsulates multiple cryptographic and non-cryptographic checksums.
type AudioChecksums struct {
	SHA256   string `json:"sha256"`
	SHA512   string `json:"sha512"`
	MD5      string `json:"md5"`
	CRC32    string `json:"crc32"`
	Adler32  string `json:"adler32"`
	ByteSize int64  `json:"byte_size"`
}

// AudioIntegrityManifest encapsulates container file checksums and underlying PCM audio checksums.
type AudioIntegrityManifest struct {
	TrackID       string         `json:"track_id,omitempty"`
	FileChecksums AudioChecksums `json:"file_checksums"`
	PCMChecksums  AudioChecksums `json:"pcm_checksums"`
	SampleRate    int            `json:"sample_rate,omitempty"`
	Channels      int            `json:"channels,omitempty"`
	DurationSec   float64        `json:"duration_sec,omitempty"`
	GeneratedAt   time.Time      `json:"generated_at"`
}

// MultiChecksumBuilder calculates 5 concurrent hashes in a single streaming pass.
type MultiChecksumBuilder struct {
	sha256Hash  hash.Hash
	sha512Hash  hash.Hash
	md5Hash     hash.Hash
	crc32Hash   hash.Hash32
	adler32Hash hash.Hash32
	multiWriter io.Writer
	bytesCount  int64
}

// NewMultiChecksumBuilder instantiates a multi-hash streaming builder.
func NewMultiChecksumBuilder() *MultiChecksumBuilder {
	h256 := sha256.New()
	h512 := sha512.New()
	hMD5 := md5.New()
	hCRC := crc32.NewIEEE()
	hAdler := adler32.New()

	mw := io.MultiWriter(h256, h512, hMD5, hCRC, hAdler)

	return &MultiChecksumBuilder{
		sha256Hash:  h256,
		sha512Hash:  h512,
		md5Hash:     hMD5,
		crc32Hash:   hCRC,
		adler32Hash: hAdler,
		multiWriter: mw,
	}
}

// Write processes a byte chunk through all 5 hashes simultaneously.
func (b *MultiChecksumBuilder) Write(p []byte) (int, error) {
	n, err := b.multiWriter.Write(p)
	b.bytesCount += int64(n)
	return n, err
}

// Sum finalizes and returns the AudioChecksums structure.
func (b *MultiChecksumBuilder) Sum() AudioChecksums {
	return AudioChecksums{
		SHA256:   hex.EncodeToString(b.sha256Hash.Sum(nil)),
		SHA512:   hex.EncodeToString(b.sha512Hash.Sum(nil)),
		MD5:      hex.EncodeToString(b.md5Hash.Sum(nil)),
		CRC32:    fmt.Sprintf("%08x", b.crc32Hash.Sum32()),
		Adler32:  fmt.Sprintf("%08x", b.adler32Hash.Sum32()),
		ByteSize: b.bytesCount,
	}
}

// ComputeStreamChecksums reads an entire stream and computes all checksums in one pass.
func ComputeStreamChecksums(r io.Reader) (AudioChecksums, error) {
	if r == nil {
		return AudioChecksums{}, ErrNilReader
	}

	builder := NewMultiChecksumBuilder()
	buf := make([]byte, 64*1024) // 64KB buffer
	if _, err := io.CopyBuffer(builder, r, buf); err != nil {
		return AudioChecksums{}, fmt.Errorf("failed to compute stream checksums: %w", err)
	}

	return builder.Sum(), nil
}

// ComputeBytesChecksums computes checksums for an in-memory byte slice.
func ComputeBytesChecksums(data []byte) AudioChecksums {
	builder := NewMultiChecksumBuilder()
	_, _ = builder.Write(data)
	return builder.Sum()
}

// ComputePCMChecksums calculates container-independent audio content checksums
// by streaming normalized interleaved 16-bit signed PCM samples.
func ComputePCMChecksums(buf *PCMBuffer) (AudioChecksums, error) {
	if buf == nil {
		return AudioChecksums{}, ErrNilPCMBuffer
	}
	if buf.NumSamples == 0 || buf.Channels == 0 || len(buf.Samples) == 0 {
		return AudioChecksums{}, ErrEmptyPCMBuffer
	}

	builder := NewMultiChecksumBuilder()
	channels := buf.Channels
	numSamples := buf.NumSamples

	// Write 16-bit interleaved PCM samples
	sampleBuf := make([]byte, 2)
	for i := 0; i < numSamples; i++ {
		for ch := 0; ch < channels; ch++ {
			v := buf.Samples[ch][i]
			if v > 1.0 {
				v = 1.0
			} else if v < -1.0 {
				v = -1.0
			}
			s16 := int16(math.Round(v * 32767.0))
			binary.LittleEndian.PutUint16(sampleBuf, uint16(s16))
			_, _ = builder.Write(sampleBuf)
		}
	}

	return builder.Sum(), nil
}

// BuildManifest creates a complete audio integrity and provenance manifest.
func BuildManifest(trackID string, fileBytes []byte, pcmBuf *PCMBuffer) (*AudioIntegrityManifest, error) {
	fileSums := ComputeBytesChecksums(fileBytes)
	pcmSums, err := ComputePCMChecksums(pcmBuf)
	if err != nil {
		return nil, fmt.Errorf("failed computing pcm checksums: %w", err)
	}

	duration := 0.0
	sr := 0
	ch := 0
	if pcmBuf != nil && pcmBuf.SampleRate > 0 {
		duration = float64(pcmBuf.NumSamples) / float64(pcmBuf.SampleRate)
		sr = pcmBuf.SampleRate
		ch = pcmBuf.Channels
	}

	return &AudioIntegrityManifest{
		TrackID:       trackID,
		FileChecksums: fileSums,
		PCMChecksums:  pcmSums,
		SampleRate:    sr,
		Channels:      ch,
		DurationSec:   math.Round(duration*100) / 100,
		GeneratedAt:   time.Now().UTC(),
	}, nil
}

// VerifyFileChecksums checks if input matches expected checksums.
func (c AudioChecksums) Matches(other AudioChecksums) bool {
	if c.SHA256 != "" && other.SHA256 != "" && c.SHA256 != other.SHA256 {
		return false
	}
	if c.MD5 != "" && other.MD5 != "" && c.MD5 != other.MD5 {
		return false
	}
	if c.CRC32 != "" && other.CRC32 != "" && c.CRC32 != other.CRC32 {
		return false
	}
	return true
}

// ToJSON serializes manifest to JSON.
func (m *AudioIntegrityManifest) ToJSON() (string, error) {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
