package audio

import (
	"math"
	"testing"
)

func TestPhaseCorrelator_DualMono(t *testing.T) {
	pc := NewPhaseCorrelator(DefaultPhaseCorrelatorConfig())

	n := 48000
	left := make([]float64, n)
	right := make([]float64, n)
	for i := 0; i < n; i++ {
		val := 0.5 * math.Sin(2*math.Pi*440.0*float64(i)/48000.0)
		left[i] = val
		right[i] = val
	}

	buf := &PCMBuffer{
		Channels:   2,
		SampleRate: 48000,
		NumSamples: n,
		Samples:    [][]float64{left, right},
	}

	res, err := pc.Analyze(buf)
	if err != nil {
		t.Fatalf("Analyze dual mono failed: %v", err)
	}

	if res.GlobalCorrelation < 0.99 {
		t.Errorf("expected global correlation ~1.0, got %f", res.GlobalCorrelation)
	}
	if res.MonoEnergyLossDb > 0.01 {
		t.Errorf("expected 0dB mono loss, got %f dB", res.MonoEnergyLossDb)
	}
	if !res.MonoCompatible {
		t.Errorf("dual mono should be 100%% mono compatible")
	}
	if res.HasSevereAntiPhase {
		t.Errorf("unexpected anti-phase flag for dual mono")
	}
}

func TestPhaseCorrelator_PureAntiPhase(t *testing.T) {
	pc := NewPhaseCorrelator(DefaultPhaseCorrelatorConfig())

	n := 48000
	left := make([]float64, n)
	right := make([]float64, n)
	for i := 0; i < n; i++ {
		val := 0.5 * math.Sin(2*math.Pi*150.0*float64(i)/48000.0)
		left[i] = val
		right[i] = -val // 180 degree out of phase!
	}

	buf := &PCMBuffer{
		Channels:   2,
		SampleRate: 48000,
		NumSamples: n,
		Samples:    [][]float64{left, right},
	}

	res, err := pc.Analyze(buf)
	if err != nil {
		t.Fatalf("Analyze anti-phase failed: %v", err)
	}

	if res.GlobalCorrelation > -0.99 {
		t.Errorf("expected global correlation ~ -1.0, got %f", res.GlobalCorrelation)
	}
	if !res.HasSevereAntiPhase {
		t.Errorf("expected HasSevereAntiPhase to be true")
	}
	if res.MonoCompatible {
		t.Errorf("anti-phase should NOT be mono compatible")
	}
	if res.MonoEnergyLossDb < 50.0 {
		t.Errorf("expected severe mono cancellation loss (>50dB), got %f dB", res.MonoEnergyLossDb)
	}
}

func TestPhaseCorrelator_ChannelValidation(t *testing.T) {
	pc := NewPhaseCorrelator(DefaultPhaseCorrelatorConfig())

	bufMono := &PCMBuffer{
		Channels:   1,
		SampleRate: 48000,
		NumSamples: 100,
		Samples:    [][]float64{make([]float64, 100)},
	}
	_, err := pc.Analyze(bufMono)
	if err != ErrNotStereo {
		t.Fatalf("expected ErrNotStereo, got %v", err)
	}
}
