package audio

import (
	"math"
	"testing"
)

func TestBPM_StandardMusicalTempos(t *testing.T) {
	detector, err := NewBPMDetector(DefaultBPMConfig())
	if err != nil {
		t.Fatalf("NewBPMDetector failed: %v", err)
	}

	testTempos := []float64{90.0, 120.0, 128.0, 140.0, 174.0}

	for _, targetBPM := range testTempos {
		pcm := GenerateRhythmicTestSignal(targetBPM, 3.0, 48000, 2)
		res, err := detector.Analyze(pcm)
		if err != nil {
			t.Fatalf("Analyze failed for %.1f BPM: %v", targetBPM, err)
		}

		diff := math.Abs(res.BPM - targetBPM)
		if diff > 1.2 {
			t.Errorf("Tempo estimation failed for %.1f BPM: detected %.2f BPM (diff=%.2f > 1.2 BPM)",
				targetBPM, res.BPM, diff)
		}
		if !res.IsSteadyBeat {
			t.Errorf("expected IsSteadyBeat=true for %.1f BPM, got false (stability=%.2f)",
				targetBPM, res.TempoStability)
		}

		t.Logf("Target: %.1f BPM -> Detected: %.2f BPM (Diff: %.2f BPM, Stability: %.2f, Confidence: %.2f)",
			targetBPM, res.BPM, diff, res.TempoStability, res.Confidence)
	}
}

func TestBPM_FractionalTempos(t *testing.T) {
	detector, err := NewBPMDetector(DefaultBPMConfig())
	if err != nil {
		t.Fatalf("NewBPMDetector failed: %v", err)
	}

	fractionalTempos := []float64{95.5, 112.3, 123.7, 131.4, 155.8}

	for _, targetBPM := range fractionalTempos {
		pcm := GenerateRhythmicTestSignal(targetBPM, 3.5, 48000, 2)
		res, err := detector.Analyze(pcm)
		if err != nil {
			t.Fatalf("Analyze failed for %.2f BPM: %v", targetBPM, err)
		}

		diff := math.Abs(res.BPM - targetBPM)
		if diff > 1.5 {
			t.Errorf("Fractional tempo failed for %.2f BPM: detected %.2f BPM (diff=%.2f > 1.5 BPM DoD)",
				targetBPM, res.BPM, diff)
		}

		t.Logf("Fractional Target: %.2f BPM -> Detected: %.2f BPM (Diff: %.2f BPM)", targetBPM, res.BPM, diff)
	}
}

func TestBPM_BeatGridAndPhase(t *testing.T) {
	targetBPM := 120.0
	pcm := GenerateRhythmicTestSignal(targetBPM, 4.0, 48000, 2)
	detector, _ := NewBPMDetector(DefaultBPMConfig())

	res, err := detector.Analyze(pcm)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	expectedInterval := 60.0 / targetBPM // 0.5s
	if math.Abs(res.BeatIntervalSec-expectedInterval) > 0.01 {
		t.Errorf("expected beat interval ~0.50s, got %.4fs", res.BeatIntervalSec)
	}

	if len(res.BeatTimestamps) < 7 || len(res.BeatTimestamps) > 9 {
		t.Errorf("expected ~8 beats in 4.0s at 120 BPM, got %d beats", len(res.BeatTimestamps))
	}

	// Verify consecutive beat intervals
	for i := 1; i < len(res.BeatTimestamps); i++ {
		interval := res.BeatTimestamps[i] - res.BeatTimestamps[i-1]
		if math.Abs(interval-expectedInterval) > 0.015 {
			t.Errorf("beat %d to %d interval jitter: expected %.3fs, got %.3fs", i-1, i, expectedInterval, interval)
		}
	}

	t.Logf("Beat grid verified: Total beats=%d, FirstBeat=%.3fs, Interval=%.4fs",
		len(res.BeatTimestamps), res.FirstBeatSec, res.BeatIntervalSec)
}

func TestBPM_MultiChannelInvariance(t *testing.T) {
	targetBPM := 125.0
	detector, _ := NewBPMDetector(DefaultBPMConfig())

	// Mono
	monoPCM := GenerateRhythmicTestSignal(targetBPM, 2.5, 48000, 1)
	resMono, err := detector.Analyze(monoPCM)
	if err != nil {
		t.Fatalf("Analyze mono failed: %v", err)
	}

	// Stereo
	stereoPCM := GenerateRhythmicTestSignal(targetBPM, 2.5, 48000, 2)
	resStereo, err := detector.Analyze(stereoPCM)
	if err != nil {
		t.Fatalf("Analyze stereo failed: %v", err)
	}

	// 5.1 Surround (6ch)
	surroundPCM := GenerateRhythmicTestSignal(targetBPM, 2.5, 48000, 6)
	resSurround, err := detector.Analyze(surroundPCM)
	if err != nil {
		t.Fatalf("Analyze surround failed: %v", err)
	}

	diffMonoStereo := math.Abs(resMono.BPM - resStereo.BPM)
	diffStereoSurround := math.Abs(resStereo.BPM - resSurround.BPM)

	if diffMonoStereo > 0.1 || diffStereoSurround > 0.1 {
		t.Errorf("Channel variance detected: Mono=%.2f, Stereo=%.2f, Surround=%.2f",
			resMono.BPM, resStereo.BPM, resSurround.BPM)
	}
}

func TestBPM_EdgeCases(t *testing.T) {
	detector, _ := NewBPMDetector(DefaultBPMConfig())

	// Nil buffer
	_, err := detector.Analyze(nil)
	if err != ErrNilPCMBuffer {
		t.Errorf("expected ErrNilPCMBuffer, got %v", err)
	}

	// Empty buffer
	_, err = detector.Analyze(&PCMBuffer{Channels: 2, SampleRate: 48000})
	if err != ErrEmptyAudioBuffer {
		t.Errorf("expected ErrEmptyAudioBuffer, got %v", err)
	}

	// Too short (< 0.8s)
	shortPCM := GenerateRhythmicTestSignal(120.0, 0.4, 48000, 2)
	_, err = detector.Analyze(shortPCM)
	if err != ErrAudioTooShortForBPM {
		t.Errorf("expected ErrAudioTooShortForBPM, got %v", err)
	}

	// Silent buffer (no rhythmic activity)
	silentSamples := make([][]float64, 1)
	silentSamples[0] = make([]float64, 48000*2)
	silentPCM := &PCMBuffer{Samples: silentSamples, Channels: 1, SampleRate: 48000, NumSamples: 48000 * 2}
	_, err = detector.Analyze(silentPCM)
	if err != ErrNoRhythmicActivity {
		t.Errorf("expected ErrNoRhythmicActivity on silent audio, got %v", err)
	}
}

func TestBPM_100HeterogeneousStreams_DoD(t *testing.T) {
	detector, err := NewBPMDetector(DefaultBPMConfig())
	if err != nil {
		t.Fatalf("NewBPMDetector failed: %v", err)
	}

	sampleRates := []int{44100, 48000, 96000}
	channelConfigs := []int{1, 2, 6}

	maxObservedError := 0.0
	passedCount := 0

	for i := 1; i <= 100; i++ {
		sr := sampleRates[i%len(sampleRates)]
		ch := channelConfigs[i%len(channelConfigs)]
		dur := 2.2 + float64(i%10)*0.1 // 2.2s to 3.1s

		// Continuous pseudo-random tempo between 75.0 and 165.0 BPM
		targetBPM := 75.0 + float64(i%90)*1.0 + float64((i*37)%10)*0.1

		pcm := GenerateRhythmicTestSignal(targetBPM, dur, sr, ch)

		res, err := detector.Analyze(pcm)
		if err != nil {
			t.Fatalf("[%d/100] Analyze failed for target %.2f BPM: %v", i, targetBPM, err)
		}

		diff := math.Abs(res.BPM - targetBPM)
		if diff > maxObservedError {
			maxObservedError = diff
		}

		// DoD Requirement: 실제 템포 오차 +-1.5 BPM 이내 정밀 검출
		if diff <= 1.5 {
			passedCount++
		} else {
			t.Errorf("[%d/100] BPM error exceeded DoD threshold (+-1.5): Target=%.2f BPM, Detected=%.2f BPM (Diff=%.2f BPM)",
				i, targetBPM, res.BPM, diff)
		}
	}

	t.Logf("100 Heterogeneous BPM Analysis Results:")
	t.Logf("  Pass Rate (<= 1.5 BPM error): %d/100 (%.1f%%) [DoD Requirement 100%%]", passedCount, float64(passedCount))
	t.Logf("  Maximum Observed Error:      %.3f BPM", maxObservedError)

	if passedCount < 100 {
		t.Fatalf("BPM test suite failed DoD requirement: %d/100 passed", passedCount)
	}
}
