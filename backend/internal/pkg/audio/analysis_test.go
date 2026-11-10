package audio

import (
	"math"
	"testing"
)

func TestAnalyzeTrackDetectsTempoAndKey(t *testing.T) {
	keyed := GenerateMusicalKeyTestSignal("C", ScaleMajor, 6.0, 22050, 1)
	mixed := &PCMBuffer{Channels: 1, SampleRate: 22050, NumSamples: keyed.NumSamples, Samples: [][]float64{make([]float64, keyed.NumSamples)}}
	// Pluck-like 120 BPM amplitude gate: sharp onsets for tempo, harmonic
	// energy preserved for key.
	for i := range mixed.Samples[0] {
		phase := math.Mod(float64(i)/22050.0, 0.5)
		env := 0.15 + 0.85*math.Exp(-phase/0.05)
		mixed.Samples[0][i] = keyed.Samples[0][i] * env
	}
	got, err := AnalyzeTrack(mixed)
	if err != nil {
		t.Fatalf("AnalyzeTrack: %v", err)
	}
	if math.Abs(got.BPM-120.0) > 3.0 {
		t.Errorf("BPM = %.2f, want ~120", got.BPM)
	}
	if got.MusicalKey != "C Major" {
		t.Errorf("key = %q, want C Major", got.MusicalKey)
	}
}

func TestAnalyzeTrackRejectsSilence(t *testing.T) {
	pcm := &PCMBuffer{Channels: 1, SampleRate: 22050, NumSamples: 22050, Samples: [][]float64{make([]float64, 22050)}}
	if _, err := AnalyzeTrack(pcm); err == nil {
		t.Error("silence must fail BPM analysis")
	}
	if _, err := AnalyzeTrack(nil); err == nil {
		t.Error("nil PCM must fail")
	}
}
