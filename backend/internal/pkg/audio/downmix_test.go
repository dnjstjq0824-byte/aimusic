package audio

import (
	"math"
	"testing"
)

func TestDownmix_EmptyBuffer(t *testing.T) {
	dm := NewMultichannelDownmixer(DefaultDownmixConfig())

	_, err := dm.Downmix(nil)
	if err != ErrNilPCMBuffer {
		t.Fatalf("expected ErrNilPCMBuffer, got %v", err)
	}

	buf := &PCMBuffer{
		Channels:   6,
		SampleRate: 48000,
		NumSamples: 0,
		Samples:    [][]float64{},
	}
	_, err = dm.Downmix(buf)
	if err != ErrEmptyPCMBuffer {
		t.Fatalf("expected ErrEmptyPCMBuffer, got %v", err)
	}
}

func TestDownmix_StereoPassThrough(t *testing.T) {
	dm := NewMultichannelDownmixer(DefaultDownmixConfig())

	n := 480
	left := make([]float64, n)
	right := make([]float64, n)
	for i := 0; i < n; i++ {
		left[i] = 0.5 * math.Sin(2*math.Pi*440*float64(i)/48000.0)
		right[i] = 0.5 * math.Sin(2*math.Pi*440*float64(i)/48000.0)
	}

	buf := &PCMBuffer{
		Channels:   2,
		SampleRate: 48000,
		NumSamples: n,
		Samples:    [][]float64{left, right},
	}

	res, err := dm.Downmix(buf)
	if err != nil {
		t.Fatalf("Downmix failed: %v", err)
	}

	if res.SourceLayout != LayoutStereo || res.TargetLayout != LayoutStereo {
		t.Fatalf("expected stereo->stereo layout, got %s -> %s", res.SourceLayout, res.TargetLayout)
	}

	if res.PhaseCorrelation < 0.99 {
		t.Errorf("expected phase correlation ~1.0 for dual mono, got %f", res.PhaseCorrelation)
	}
}

func TestDownmix_5Point1ToLoRo(t *testing.T) {
	cfg := DefaultDownmixConfig()
	cfg.IncludeLFE = false
	dm := NewMultichannelDownmixer(cfg)

	n := 480
	// 5.1: 0:L, 1:R, 2:C, 3:LFE, 4:Ls, 5:Rs
	samples := make([][]float64, 6)
	for ch := 0; ch < 6; ch++ {
		samples[ch] = make([]float64, n)
	}

	// Only Center channel active at 0.5 amplitude
	for i := 0; i < n; i++ {
		samples[2][i] = 0.5
	}

	buf := &PCMBuffer{
		Channels:   6,
		SampleRate: 48000,
		NumSamples: n,
		Samples:    samples,
	}

	res, err := dm.Downmix(buf)
	if err != nil {
		t.Fatalf("Downmix failed: %v", err)
	}

	if res.SourceLayout != Layout5Point1 || res.TargetLayout != LayoutStereo {
		t.Fatalf("expected 5.1 -> stereo, got %s -> %s", res.SourceLayout, res.TargetLayout)
	}

	// In Lo/Ro, center channel should be attenuated by ~0.7071 and split equally to Left and Right
	expectedAmp := 0.5 * math.Sqrt(0.5)
	if math.Abs(res.StereoSamples[0][100]-expectedAmp) > 0.001 {
		t.Errorf("expected left amp %f, got %f", expectedAmp, res.StereoSamples[0][100])
	}
	if math.Abs(res.StereoSamples[1][100]-expectedAmp) > 0.001 {
		t.Errorf("expected right amp %f, got %f", expectedAmp, res.StereoSamples[1][100])
	}
}

func TestDownmix_7Point1ToStereo(t *testing.T) {
	dm := NewMultichannelDownmixer(DefaultDownmixConfig())

	n := 480
	// 7.1: 8 channels
	samples := make([][]float64, 8)
	for ch := 0; ch < 8; ch++ {
		samples[ch] = make([]float64, n)
	}

	// L, R at 0.2
	for i := 0; i < n; i++ {
		samples[0][i] = 0.2
		samples[1][i] = 0.2
		samples[6][i] = 0.1 // Rear left surround
		samples[7][i] = 0.1 // Rear right surround
	}

	buf := &PCMBuffer{
		Channels:   8,
		SampleRate: 48000,
		NumSamples: n,
		Samples:    samples,
	}

	res, err := dm.Downmix(buf)
	if err != nil {
		t.Fatalf("Downmix 7.1 failed: %v", err)
	}

	if res.SourceLayout != Layout7Point1 {
		t.Fatalf("expected Layout7Point1, got %s", res.SourceLayout)
	}
	if len(res.StereoSamples) != 2 {
		t.Fatalf("expected 2 output channels, got %d", len(res.StereoSamples))
	}
}

func TestDownmix_HeadroomNormalization(t *testing.T) {
	cfg := DefaultDownmixConfig()
	cfg.NormalizeHeadroom = true
	dm := NewMultichannelDownmixer(cfg)

	n := 480
	// All channels at 0.9 amplitude -> Sum will easily exceed 1.0 (clipping)
	samples := make([][]float64, 6)
	for ch := 0; ch < 6; ch++ {
		samples[ch] = make([]float64, n)
		for i := 0; i < n; i++ {
			samples[ch][i] = 0.9
		}
	}

	buf := &PCMBuffer{
		Channels:   6,
		SampleRate: 48000,
		NumSamples: n,
		Samples:    samples,
	}

	res, err := dm.Downmix(buf)
	if err != nil {
		t.Fatalf("Downmix failed: %v", err)
	}

	// Peak after downmix should not exceed 1.0 (target ceiling is 0.98)
	peakL := calcPeak(res.StereoSamples[0])
	peakR := calcPeak(res.StereoSamples[1])
	maxPeak := math.Max(peakL, peakR)

	if maxPeak > 0.981 {
		t.Errorf("peak after normalization exceeds ceiling: %f", maxPeak)
	}
	if res.AppliedGainDb >= 0.0 {
		t.Errorf("expected negative attenuation gain dB, got %f", res.AppliedGainDb)
	}
}

func TestDownmix_MonoMode(t *testing.T) {
	cfg := DefaultDownmixConfig()
	cfg.Mode = DownmixMono
	dm := NewMultichannelDownmixer(cfg)

	n := 480
	samples := make([][]float64, 6)
	for ch := 0; ch < 6; ch++ {
		samples[ch] = make([]float64, n)
	}
	for i := 0; i < n; i++ {
		samples[0][i] = 0.4 // Left
		samples[1][i] = 0.2 // Right
	}

	buf := &PCMBuffer{
		Channels:   6,
		SampleRate: 48000,
		NumSamples: n,
		Samples:    samples,
	}

	res, err := dm.Downmix(buf)
	if err != nil {
		t.Fatalf("Downmix mono failed: %v", err)
	}

	if res.TargetLayout != LayoutMono {
		t.Fatalf("expected LayoutMono, got %s", res.TargetLayout)
	}

	// Left and Right channels in mono mode should be identical
	for i := 0; i < n; i++ {
		if math.Abs(res.StereoSamples[0][i]-res.StereoSamples[1][i]) > 1e-9 {
			t.Fatalf("sample mismatch at %d: %f vs %f", i, res.StereoSamples[0][i], res.StereoSamples[1][i])
		}
	}
}

func TestDownmix_JSONTelemetry(t *testing.T) {
	dm := NewMultichannelDownmixer(DefaultDownmixConfig())
	buf := &PCMBuffer{
		Channels:   2,
		SampleRate: 48000,
		NumSamples: 100,
		Samples: [][]float64{
			make([]float64, 100),
			make([]float64, 100),
		},
	}
	res, err := dm.Downmix(buf)
	if err != nil {
		t.Fatalf("Downmix failed: %v", err)
	}
	jsonStr, err := res.ToJSON()
	if err != nil || len(jsonStr) == 0 {
		t.Fatalf("failed to serialize telemetry: %v", err)
	}
}
