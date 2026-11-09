package audio

import (
	"math"
	"math/rand"
	"testing"
)

func TestDCBlocker_StationaryBiasRemoval(t *testing.T) {
	sr := 48000
	numSamples := 48000
	bias := 0.10 // +10% DC offset

	pcm := createSyntheticPCMBuffer(sr, 2, numSamples, func(s, c int) float64 {
		tSec := float64(s) / float64(sr)
		return bias + 0.5*math.Sin(2.0*math.Pi*440.0*tSec)
	})

	cfg := DefaultDCBlockerConfig()
	err := RemoveDCOffsetAdvanced(pcm, cfg)
	if err != nil {
		t.Fatalf("RemoveDCOffsetAdvanced failed: %v", err)
	}

	for c := 0; c < pcm.Channels; c++ {
		sum := 0.0
		for s := 0; s < pcm.NumSamples; s++ {
			sum += pcm.Samples[c][s]
		}
		residualDC := sum / float64(pcm.NumSamples)
		residualPercent := math.Abs(residualDC) * 100.0

		// Requirement: DC bias <= 0.001%
		if residualPercent > 0.001 {
			t.Errorf("channel %d residual DC %f%% exceeds 0.001%% requirement", c, residualPercent)
		}
	}
}

func TestDCBlocker_DriftingSubHertzBias(t *testing.T) {
	sr := 48000
	numSamples := 48000 * 2 // 2 seconds

	// Superimpose a slow 0.5 Hz drifting DC wave onto a 1000 Hz tone
	pcm := createSyntheticPCMBuffer(sr, 1, numSamples, func(s, c int) float64 {
		tSec := float64(s) / float64(sr)
		slowDrift := 0.08 * math.Sin(2.0*math.Pi*0.5*tSec)
		audioTone := 0.50 * math.Sin(2.0*math.Pi*1000.0*tSec)
		return slowDrift + audioTone
	})

	cfg := DCBlockerConfig{
		CutoffHz:           5.0,
		UseMeanSubtraction: true,
	}
	err := RemoveDCOffsetAdvanced(pcm, cfg)
	if err != nil {
		t.Fatalf("RemoveDCOffsetAdvanced failed: %v", err)
	}

	// Residual DC over whole buffer must be < 0.001%
	sum := 0.0
	for s := 0; s < pcm.NumSamples; s++ {
		sum += pcm.Samples[0][s]
	}
	residualDC := sum / float64(pcm.NumSamples)
	if math.Abs(residualDC)*100.0 > 0.001 {
		t.Errorf("drifting sub-hertz residual DC too high: %f%%", math.Abs(residualDC)*100.0)
	}

	// Audio tone (1kHz) peak amplitude must be preserved within 1% (passband preservation)
	maxAmp := 0.0
	for s := 24000; s < pcm.NumSamples-24000; s++ { // check middle second
		if math.Abs(pcm.Samples[0][s]) > maxAmp {
			maxAmp = math.Abs(pcm.Samples[0][s])
		}
	}
	if math.Abs(maxAmp-0.50) > 0.02 {
		t.Errorf("1kHz audio tone passband amplitude attenuated: expected ~0.50, got %f", maxAmp)
	}
}

func TestDeclicker_ImpulseDetectionAndInterpolation(t *testing.T) {
	sr := 48000
	numSamples := 4800 // 100ms
	pcm := createSyntheticPCMBuffer(sr, 1, numSamples, func(s, c int) float64 {
		tSec := float64(s) / float64(sr)
		return 0.3 * math.Sin(2.0*math.Pi*200.0*tSec)
	})

	// Inject 4 distinct Dirac click impulses
	clickIndices := []int{500, 1200, 2500, 3800}
	for _, idx := range clickIndices {
		pcm.Samples[0][idx] = 0.95 // Sharp pop spike
	}

	cfg := DefaultDeclickerConfig()
	detected, repaired := RepairClicksAndPops(pcm, cfg)

	if detected < 4 {
		t.Errorf("expected at least 4 clicks detected, got %d", detected)
	}
	if repaired < 4 {
		t.Errorf("expected at least 4 clicks repaired, got %d", repaired)
	}

	// Verify all corrupted indices were smoothly interpolated back close to sine wave
	for _, idx := range clickIndices {
		tSec := float64(idx) / float64(sr)
		expectedVal := 0.3 * math.Sin(2.0*math.Pi*200.0*tSec)
		actualVal := pcm.Samples[0][idx]

		diff := math.Abs(actualVal - expectedVal)
		if diff > 0.05 {
			t.Errorf("click at index %d not properly reconstructed: expected %f, got %f (diff %f)", idx, expectedVal, actualVal, diff)
		}
	}
}

func TestAudioCleaner_FullCleanPass(t *testing.T) {
	sr := 48000
	numSamples := 24000 // 0.5s
	bias := 0.06        // +6% DC offset

	pcm := createSyntheticPCMBuffer(sr, 2, numSamples, func(s, c int) float64 {
		tSec := float64(s) / float64(sr)
		return bias + 0.4*math.Sin(2.0*math.Pi*300.0*tSec)
	})

	// Inject 6 clicks in channel 0 and 6 clicks in channel 1
	injectedClicks := []int{1000, 3000, 6000, 10000, 15000, 20000}
	for _, idx := range injectedClicks {
		pcm.Samples[0][idx] = 0.99
		pcm.Samples[1][idx] = -0.99
	}

	cleaner := NewAudioCleaner(DefaultDCBlockerConfig(), DefaultDeclickerConfig())
	report, err := cleaner.Clean(pcm)
	if err != nil {
		t.Fatalf("Clean failed: %v", err)
	}

	if report.ClicksDetected < 12 {
		t.Errorf("expected at least 12 clicks detected across channels, got %d", report.ClicksDetected)
	}
	if report.ClicksRepaired < 12 {
		t.Errorf("expected at least 12 clicks repaired, got %d", report.ClicksRepaired)
	}

	// Verify DC bias corrected to <= 0.001%
	if report.MaxResidualDCBias > 0.001 {
		t.Errorf("residual DC bias %f%% exceeds 0.001%% requirement", report.MaxResidualDCBias)
	}
	if !report.DCCorrected {
		t.Errorf("expected DCCorrected = true")
	}
}

func TestCleaner_EdgeCases(t *testing.T) {
	cleaner := NewAudioCleaner(DefaultDCBlockerConfig(), DefaultDeclickerConfig())

	// Nil buffer
	_, err := cleaner.Clean(nil)
	if err != ErrNilCleanBuffer {
		t.Errorf("expected ErrNilCleanBuffer, got %v", err)
	}

	// Empty buffer
	emptyBuf := &PCMBuffer{Channels: 1, SampleRate: 48000, NumSamples: 0, Samples: [][]float64{{}}}
	_, err = cleaner.Clean(emptyBuf)
	if err != ErrEmptyAudioCleanBuffer {
		t.Errorf("expected ErrEmptyAudioCleanBuffer, got %v", err)
	}

	// Invalid cutoff frequency
	pcm := createSyntheticPCMBuffer(48000, 1, 1000, func(s, c int) float64 { return 0.1 })
	err = RemoveDCOffsetAdvanced(pcm, DCBlockerConfig{CutoffHz: -5.0})
	if err != ErrInvalidCutoffHz {
		t.Errorf("expected ErrInvalidCutoffHz for negative cutoff, got %v", err)
	}
	err = RemoveDCOffsetAdvanced(pcm, DCBlockerConfig{CutoffHz: 25000.0}) // above Nyquist (24000)
	if err != ErrInvalidCutoffHz {
		t.Errorf("expected ErrInvalidCutoffHz for above Nyquist cutoff, got %v", err)
	}

	// Clean audio without DC or clicks
	cleanPCM := createSyntheticPCMBuffer(48000, 1, 4800, func(s, c int) float64 {
		return 0.2 * math.Sin(float64(s)*0.1)
	})
	rep, err := cleaner.Clean(cleanPCM)
	if err != nil {
		t.Fatalf("Clean failed on clean audio: %v", err)
	}
	if rep.ClicksDetected != 0 {
		t.Errorf("expected 0 clicks detected on clean audio, got %d", rep.ClicksDetected)
	}
}

func TestCleaner_100HeterogeneousAudioStreams(t *testing.T) {
	rng := rand.New(rand.NewSource(20261109))
	totalTests := 100

	sampleRates := []int{8000, 16000, 22050, 32000, 44100, 48000, 96000}
	channelsList := []int{1, 2, 4}

	for iter := 0; iter < totalTests; iter++ {
		sr := sampleRates[rng.Intn(len(sampleRates))]
		ch := channelsList[rng.Intn(len(channelsList))]
		numSamples := sr/2 + rng.Intn(sr) // 0.5s to 1.5s
		dcBias := (rng.Float64()*0.20 - 0.10) // -10% to +10% DC bias

		pcm := createSyntheticPCMBuffer(sr, ch, numSamples, func(s, c int) float64 {
			tSec := float64(s) / float64(sr)
			return dcBias + 0.3*math.Sin(2.0*math.Pi*250.0*tSec)
		})

		// Inject 3 random click spikes
		numClicks := 3
		for k := 0; k < numClicks; k++ {
			clickIdx := 100 + rng.Intn(numSamples-200)
			for c := 0; c < ch; c++ {
				pcm.Samples[c][clickIdx] = 0.98
			}
		}

		cleaner := NewAudioCleaner(DefaultDCBlockerConfig(), DefaultDeclickerConfig())
		rep, err := cleaner.Clean(pcm)
		if err != nil {
			t.Fatalf("iteration %d: Clean failed: %v", iter, err)
		}

		// Verify DC bias is <= 0.001%
		if rep.MaxResidualDCBias > 0.001 {
			t.Fatalf("iteration %d: residual DC bias %f%% exceeds 0.001%%", iter, rep.MaxResidualDCBias)
		}
		if !rep.DCCorrected {
			t.Fatalf("iteration %d: expected DCCorrected = true", iter)
		}
		if rep.ClicksDetected < numClicks {
			t.Fatalf("iteration %d: detected %d clicks, expected >= %d", iter, rep.ClicksDetected, numClicks)
		}
	}

	t.Logf("Successfully verified 100 heterogeneous audio streams with DC offset removal and de-clicking.")
}
