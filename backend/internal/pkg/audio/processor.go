// Package audio wraps ffprobe/ffmpeg for validation and HLS packaging.
package audio

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// ProbeResult is the normalized output of ffprobe for one audio file.
type ProbeResult struct {
	FormatName  string
	DurationSec float64
	BitrateKbps int
	CodecName   string
	SampleRate  int
	Channels    int
	Tags        map[string]string
	Ext         string
}

// RunFFprobe executes ffprobe and parses format/stream metadata.
func RunFFprobe(ctx context.Context, path string) (*ProbeResult, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format", "-show_streams",
		path,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe %s: %w", path, err)
	}
	var raw struct {
		Format struct {
			FormatName string            `json:"format_name"`
			Duration   string            `json:"duration"`
			BitRate    string            `json:"bit_rate"`
			Tags       map[string]string `json:"tags"`
		} `json:"format"`
		Streams []struct {
			CodecName  string            `json:"codec_name"`
			CodecType  string            `json:"codec_type"`
			SampleRate string            `json:"sample_rate"`
			Channels   int               `json:"channels"`
			BitRate    string            `json:"bit_rate"`
			Tags       map[string]string `json:"tags"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parse ffprobe output: %w", err)
	}
	res := &ProbeResult{
		FormatName: raw.Format.FormatName,
		Tags:       map[string]string{},
		Ext:        strings.ToLower(filepath.Ext(path)),
	}
	for k, v := range raw.Format.Tags {
		res.Tags[k] = v
	}
	streamIdx := -1
	for i := range raw.Streams {
		if raw.Streams[i].CodecType == "audio" {
			streamIdx = i
			break
		}
	}
	if streamIdx < 0 && len(raw.Streams) > 0 {
		streamIdx = 0
	}
	if streamIdx >= 0 {
		s := raw.Streams[streamIdx]
		res.CodecName = s.CodecName
		res.Channels = s.Channels
		if n, err := strconv.Atoi(s.SampleRate); err == nil {
			res.SampleRate = n
		}
		for k, v := range s.Tags {
			res.Tags[k] = v
		}
		if bps, err := strconv.Atoi(s.BitRate); err == nil && bps > 0 {
			res.BitrateKbps = bps / 1000
		}
	}
	if res.BitrateKbps <= 0 {
		if bps, err := strconv.Atoi(raw.Format.BitRate); err == nil && bps > 0 {
			res.BitrateKbps = bps / 1000
		}
	}
	if d, err := strconv.ParseFloat(raw.Format.Duration, 64); err == nil {
		res.DurationSec = d
	}
	return res, nil
}

// DefaultBlockedMetadataKeywords is the baseline non-commercial watermark/tag filter
// mandated by Spec 6.2.2.
var DefaultBlockedMetadataKeywords = []string{
	"free",
	"non-commercial",
	"trial",
	"demo",
	"evaluation",
	"personal use only",
	"watermarked",
	"free-trial",
}

// AIWatermarkComment is the stream header watermark noting AI generation and
// the commercial-terms check (KR AI Basic Act Art.31 notice; process wording,
// not a rights guarantee).
const AIWatermarkComment = "AI-Generated Audio (KR AI Basic Act Art.31 notice | commercial terms checked)"

// CheckBlockedMetadata scans all tags (keys and values, lowercased) for blocked
// keywords (combining policy keywords and DefaultBlockedMetadataKeywords).
// It returns the matched keyword when found.
func CheckBlockedMetadata(tags map[string]string, blocked []string) (string, bool) {
	lowered := make([]string, 0, len(tags)*2)
	for k, v := range tags {
		lowered = append(lowered, strings.ToLower(k), strings.ToLower(v))
	}
	all := make([]string, 0, len(blocked)+len(DefaultBlockedMetadataKeywords))
	all = append(all, blocked...)
	all = append(all, DefaultBlockedMetadataKeywords...)
	for _, b := range all {
		needle := strings.ToLower(strings.TrimSpace(b))
		if needle == "" {
			continue
		}
		for _, hay := range lowered {
			if strings.Contains(hay, needle) {
				return b, true
			}
		}
	}
	return "", false
}

// BitrateError reports a below-minimum bitrate.
type BitrateError struct {
	Got  int
	Want int
}

func (e *BitrateError) Error() string {
	return fmt.Sprintf("bitrate %dkbps below minimum %dkbps", e.Got, e.Want)
}

// CheckBitrate enforces min_bitrate_kbps for lossy audio. Lossless WAV/FLAC
// (by extension or codec) bypass the check.
func CheckBitrate(ext, codec string, bitrateKbps, minKbps int) error {
	e := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(ext), "."))
	c := strings.ToLower(strings.TrimSpace(codec))
	switch e {
	case "wav", "flac", "aif", "aiff", "caf":
		return nil
	}
	if strings.HasPrefix(c, "pcm") || c == "flac" || c == "alac" {
		return nil
	}
	if bitrateKbps <= 0 {
		return fmt.Errorf("cannot determine bitrate for lossy audio")
	}
	if minKbps > 0 && bitrateKbps < minKbps {
		return &BitrateError{Got: bitrateKbps, Want: minKbps}
	}
	return nil
}

// BuildHLSArgs returns ffmpeg arguments that loudness-normalize the input,
// stamp license metadata, and emit 128k/256k AAC HLS variants.
func BuildHLSArgs(inputPath, outDir, comment, aiProvider, aiTier string) []string {
	return []string{
		"-y",
		"-i", inputPath,
		"-filter:a", "loudnorm",
		"-metadata", "comment=" + comment,
		"-metadata", "ai_provider=" + aiProvider,
		"-metadata", "ai_tier=" + aiTier,
		"-map", "0:a",
		"-map", "0:a",
		"-c:a", "aac",
		"-b:a:0", "128k",
		"-b:a:1", "256k",
		"-ar:0", "44100",
		"-ar:1", "48000",
		"-ac:0", "2",
		"-ac:1", "2",
		"-f", "hls",
		"-hls_time", "6",
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", filepath.Join(outDir, "seg_%v_%03d.ts"),
		"-master_pl_name", "master.m3u8",
		"-var_stream_map", "a:0 a:1",
		filepath.Join(outDir, "stream_%v.m3u8"),
	}
}

// BuildHLSCommand builds the ffmpeg HLS packaging command.
func BuildHLSCommand(ctx context.Context, inputPath, outDir, comment, aiProvider, aiTier string) *exec.Cmd {
	return exec.CommandContext(ctx, "ffmpeg", BuildHLSArgs(inputPath, outDir, comment, aiProvider, aiTier)...)
}

// ExtractWaveform decodes audio to mono 8kHz float samples and returns
// normalized peak values (0..1) across buckets buckets.
func ExtractWaveform(ctx context.Context, path string, buckets int) ([]float64, error) {
	if buckets <= 0 {
		buckets = 100
	}
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-v", "quiet",
		"-i", path,
		"-ac", "1", "-ar", "8000",
		"-f", "f32le", "-",
	)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("decode for waveform: %w", err)
	}
	raw := buf.Bytes()
	n := len(raw) / 4
	if n == 0 {
		return nil, fmt.Errorf("no audio samples decoded")
	}
	samples := make([]float64, n)
	for i := 0; i < n; i++ {
		bits := binary.LittleEndian.Uint32(raw[i*4 : i*4+4])
		samples[i] = math.Abs(float64(math.Float32frombits(bits)))
	}
	peaks := make([]float64, buckets)
	per := float64(n) / float64(buckets)
	for b := 0; b < buckets; b++ {
		start := int(float64(b) * per)
		end := int(float64(b+1) * per)
		if end <= start {
			end = start + 1
		}
		if end > n {
			end = n
		}
		max := 0.0
		for _, s := range samples[start:end] {
			if s > max {
				max = s
			}
		}
		peaks[b] = max
	}
	maxAll := 0.0
	for _, p := range peaks {
		if p > maxAll {
			maxAll = p
		}
	}
	if maxAll > 0 {
		for i := range peaks {
			peaks[i] /= maxAll
		}
	}
	return peaks, nil
}

// WaveformJSON marshals peaks for storage alongside the HLS package.
func WaveformJSON(peaks []float64) (string, error) {
	b, err := json.Marshal(peaks)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ComputeAudioFingerprint produces a deterministic 64-bit FNV-1a hex signature
// over quantized waveform peaks for duplicate/derivative audio detection.
func ComputeAudioFingerprint(peaks []float64) string {
	if len(peaks) == 0 {
		return ""
	}
	var h uint64 = 14695981039346656037
	for _, p := range peaks {
		q := uint64(math.Round(math.Max(0, math.Min(1, p)) * 255))
		h ^= q
		h *= 1099511628211
	}
	return fmt.Sprintf("fp_%016x", h)
}

// FormatFFprobeReportJSON serializes the Gate 3 forensic report stored in
// track_rights_declarations.ffprobe_report_json.
func FormatFFprobeReportJSON(probe *ProbeResult, gate3Passed bool, rejectionReason string) string {
	report := map[string]any{
		"gate3Passed":     gate3Passed,
		"rejectionReason": rejectionReason,
	}
	if probe != nil {
		report["formatName"] = probe.FormatName
		report["durationSec"] = probe.DurationSec
		report["bitrateKbps"] = probe.BitrateKbps
		report["codecName"] = probe.CodecName
		report["sampleRate"] = probe.SampleRate
		report["channels"] = probe.Channels
		report["tags"] = probe.Tags
		report["ext"] = probe.Ext
	}
	b, err := json.Marshal(report)
	if err != nil {
		return "{}"
	}
	return string(b)
}
