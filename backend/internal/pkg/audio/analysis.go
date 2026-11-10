package audio

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os/exec"
)

// AnalysisSampleRate is the mono rate used for tempo/key detection.
const AnalysisSampleRate = 22050

// AnalysisMaxSeconds caps decode for detection (full-track analysis is
// O(n log n) in samples; leading minutes carry tempo and key).
const AnalysisMaxSeconds = 360

// TrackAnalysis is the worker-persisted tempo/key result (WI-05).
type TrackAnalysis struct {
	BPM           float64
	BPMConfidence float64
	MusicalKey    string // e.g. "C Major"
	KeyConfidence float64
}

// AnalyzeTrack runs BPM and key detection over decoded PCM.
func AnalyzeTrack(pcm *PCMBuffer) (*TrackAnalysis, error) {
	if pcm == nil || pcm.NumSamples == 0 || len(pcm.Samples) == 0 {
		return nil, fmt.Errorf("analysis requires decoded audio")
	}
	bpmDetector, err := NewBPMDetector(DefaultBPMConfig())
	if err != nil {
		return nil, fmt.Errorf("bpm detector: %w", err)
	}
	tempo, err := bpmDetector.Analyze(pcm)
	if err != nil {
		return nil, fmt.Errorf("bpm analyze: %w", err)
	}
	keyDetector, err := NewKeyDetector(DefaultKeyDetectorConfig())
	if err != nil {
		return nil, fmt.Errorf("key detector: %w", err)
	}
	key, err := keyDetector.Analyze(pcm)
	if err != nil {
		return nil, fmt.Errorf("key analyze: %w", err)
	}
	return &TrackAnalysis{
		BPM:           math.Round(tempo.BPM*10) / 10,
		BPMConfidence: tempo.Confidence,
		MusicalKey:    key.Key,
		KeyConfidence: key.Confidence,
	}, nil
}

// BuildAnalysisArgs decodes input to mono f32le for detection, capped at
// maxSeconds (<=0 means AnalysisMaxSeconds).
func BuildAnalysisArgs(inputPath string, maxSeconds int) []string {
	if maxSeconds <= 0 {
		maxSeconds = AnalysisMaxSeconds
	}
	return []string{
		"-v", "quiet",
		"-i", inputPath,
		"-t", fmt.Sprintf("%d", maxSeconds),
		"-ac", "1", "-ar", fmt.Sprintf("%d", AnalysisSampleRate),
		"-f", "f32le", "-",
	}
}

// DecodeMonoForAnalysis decodes audio to a mono PCM buffer via ffmpeg.
func DecodeMonoForAnalysis(ctx context.Context, inputPath string, maxSeconds int) (*PCMBuffer, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg", BuildAnalysisArgs(inputPath, maxSeconds)...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("decode for analysis: %w", err)
	}
	raw := buf.Bytes()
	n := len(raw) / 4
	if n == 0 {
		return nil, fmt.Errorf("no audio samples decoded")
	}
	samples := make([]float64, n)
	for i := 0; i < n; i++ {
		samples[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4 : i*4+4])))
	}
	return &PCMBuffer{Channels: 1, SampleRate: AnalysisSampleRate, NumSamples: n, Samples: [][]float64{samples}}, nil
}
