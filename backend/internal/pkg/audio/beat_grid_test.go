package audio

import (
	"math"
	"testing"
)

func TestBeatGridQuantizer_120BPM_16th(t *testing.T) {
	cfg := DefaultBeatGridConfig(120.0) // 120 BPM
	cfg.Division = DivisionSixteenth
	cfg.AlignZeroCross = false // Exact mathematical grid for formula check

	bg, err := NewBeatGridQuantizer(cfg)
	if err != nil {
		t.Fatalf("NewBeatGridQuantizer failed: %v", err)
	}

	sampleRate := 48000
	numSamples := sampleRate * 4 // 4 seconds = 2 bars at 120 BPM

	samples := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		samples[i] = 0.5 * math.Sin(2*math.Pi*440.0*float64(i)/float64(sampleRate))
	}

	buf := &PCMBuffer{
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{samples},
	}

	res, err := bg.Quantize(buf)
	if err != nil {
		t.Fatalf("Quantize failed: %v", err)
	}

	if res.BPM != 120.0 {
		t.Errorf("expected 120 BPM, got %f", res.BPM)
	}
	if res.TotalBars != 2 {
		t.Errorf("expected 2 bars for 4 seconds at 120 BPM, got %d", res.TotalBars)
	}
	if res.TotalBeats != 8 {
		t.Errorf("expected 8 beats, got %d", res.TotalBeats)
	}

	// 2 bars of 16th notes = 32 markers
	if len(res.Markers) < 32 {
		t.Errorf("expected at least 32 markers, got %d", len(res.Markers))
	}

	// 1st beat marker should be Bar 1, Beat 1, Subdivision 1 at sample 0
	m0 := res.Markers[0]
	if m0.Bar != 1 || m0.Beat != 1 || m0.Subdivision != 1 || m0.SampleIndex != 0 {
		t.Errorf("unexpected marker 0: %+v", m0)
	}

	// 5th marker should be Bar 1, Beat 2, Subdivision 1 (4 steps per beat)
	m4 := res.Markers[4]
	if m4.Bar != 1 || m4.Beat != 2 || m4.Subdivision != 1 {
		t.Errorf("unexpected marker 4: %+v", m4)
	}

	// Check Slices
	if len(res.Slices) != 2 {
		t.Fatalf("expected 2 1-bar slices, got %d", len(res.Slices))
	}
	for i, s := range res.Slices {
		if math.Abs(s.DurationMs-2000.0) > 10.0 {
			t.Errorf("slice %d duration %f ms not close to 2000ms", i, s.DurationMs)
		}
	}

	jsonStr, err := res.ToJSON()
	if err != nil || len(jsonStr) == 0 {
		t.Fatalf("ToJSON failed: %v", err)
	}
}

func TestBeatGridQuantizer_InvalidBPM(t *testing.T) {
	_, err := NewBeatGridQuantizer(DefaultBeatGridConfig(10.0))
	if err != ErrInvalidBPM {
		t.Errorf("expected ErrInvalidBPM for 10 BPM, got %v", err)
	}

	_, err = NewBeatGridQuantizer(DefaultBeatGridConfig(450.0))
	if err != ErrInvalidBPM {
		t.Errorf("expected ErrInvalidBPM for 450 BPM, got %v", err)
	}
}
