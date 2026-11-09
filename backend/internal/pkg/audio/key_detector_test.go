package audio

import (
	"fmt"
	"testing"
)

func TestKey_All24Keys_FullCoverage(t *testing.T) {
	detector, err := NewKeyDetector(DefaultKeyDetectorConfig())
	if err != nil {
		t.Fatalf("NewKeyDetector failed: %v", err)
	}

	roots := PitchClassNames
	scales := []ScaleType{ScaleMajor, ScaleMinor}

	for _, scale := range scales {
		for _, root := range roots {
			targetKey := fmt.Sprintf("%s %s", root, scale)
			expectedCamelot, expectedOpenKey := GetCamelotKey(root, scale)

			pcm := GenerateMusicalKeyTestSignal(root, scale, 2.0, 48000, 2)
			res, err := detector.Analyze(pcm)
			if err != nil {
				t.Fatalf("Analyze failed for %s: %v", targetKey, err)
			}

			if res.Key != targetKey {
				t.Errorf("Key detection mismatch: expected %s, got %s (Confidence=%.2f)",
					targetKey, res.Key, res.Confidence)
			}

			if res.Camelot != expectedCamelot {
				t.Errorf("[%s] Camelot mismatch: expected %s, got %s",
					targetKey, expectedCamelot, res.Camelot)
			}

			if res.OpenKey != expectedOpenKey {
				t.Errorf("[%s] OpenKey mismatch: expected %s, got %s",
					targetKey, expectedOpenKey, res.OpenKey)
			}

			if res.Confidence < 0.40 {
				t.Errorf("[%s] Confidence too low: %.3f (expected >= 0.40)",
					targetKey, res.Confidence)
			}

			t.Logf("Detected: %-10s -> Camelot: %-3s, OpenKey: %-3s, Confidence: %.2f",
				res.Key, res.Camelot, res.OpenKey, res.Confidence)
		}
	}
}

func TestKey_CamelotHarmonicMixing(t *testing.T) {
	// 1. C Major (8B)
	compat8B := GetCompatibleCamelotKeys("8B")
	expected8B := []string{"8B", "8A", "7B", "9B", "10B"}
	if len(compat8B) != len(expected8B) {
		t.Fatalf("expected %d compatible keys for 8B, got %d", len(expected8B), len(compat8B))
	}
	for i, exp := range expected8B {
		if compat8B[i] != exp {
			t.Errorf("8B compatible[%d]: expected %s, got %s", i, exp, compat8B[i])
		}
	}

	// 2. A Minor (8A)
	compat8A := GetCompatibleCamelotKeys("8A")
	expected8A := []string{"8A", "8B", "7A", "9A", "10A"}
	for i, exp := range expected8A {
		if compat8A[i] != exp {
			t.Errorf("8A compatible[%d]: expected %s, got %s", i, exp, compat8A[i])
		}
	}

	// 3. Circle of fifths wraparound: B Major (1B)
	// -1 hour: 12B, +1 hour: 2B, +2 hours: 3B, relative: 1A
	compat1B := GetCompatibleCamelotKeys("1B")
	expected1B := []string{"1B", "1A", "12B", "2B", "3B"}
	for i, exp := range expected1B {
		if compat1B[i] != exp {
			t.Errorf("1B wraparound[%d]: expected %s, got %s", i, exp, compat1B[i])
		}
	}

	// 4. Circle of fifths wraparound: C# Minor (12A)
	// -1 hour: 11A, +1 hour: 1A, +2 hours: 2A, relative: 12B
	compat12A := GetCompatibleCamelotKeys("12A")
	expected12A := []string{"12A", "12B", "11A", "1A", "2A"}
	for i, exp := range expected12A {
		if compat12A[i] != exp {
			t.Errorf("12A wraparound[%d]: expected %s, got %s", i, exp, compat12A[i])
		}
	}
}

func TestKey_EnharmonicEquivalence(t *testing.T) {
	detector, _ := NewKeyDetector(DefaultKeyDetectorConfig())

	testCases := []struct {
		inputRoot       string
		scale           ScaleType
		expectedKey     string
		expectedCamelot string
	}{
		{"Db", ScaleMajor, "C# Major", "3B"},
		{"Eb", ScaleMinor, "D# Minor", "2A"},
		{"Bb", ScaleMajor, "A# Major", "6B"},
		{"Ab", ScaleMinor, "G# Minor", "1A"},
		{"Gb", ScaleMajor, "F# Major", "2B"},
	}

	for _, tc := range testCases {
		pcm := GenerateMusicalKeyTestSignal(tc.inputRoot, tc.scale, 2.0, 48000, 2)
		res, err := detector.Analyze(pcm)
		if err != nil {
			t.Fatalf("Analyze failed for %s %s: %v", tc.inputRoot, tc.scale, err)
		}

		if res.Key != tc.expectedKey {
			t.Errorf("Enharmonic test failed for %s %s: expected %s, got %s",
				tc.inputRoot, tc.scale, tc.expectedKey, res.Key)
		}

		if res.Camelot != tc.expectedCamelot {
			t.Errorf("Enharmonic Camelot failed for %s %s: expected %s, got %s",
				tc.inputRoot, tc.scale, tc.expectedCamelot, res.Camelot)
		}
	}
}

func TestKey_MultiChannelInvariance(t *testing.T) {
	detector, _ := NewKeyDetector(DefaultKeyDetectorConfig())

	targetRoot := "D"
	targetScale := ScaleMajor
	expectedKey := "D Major"
	expectedCamelot := "10B"

	// Mono (1ch)
	monoPCM := GenerateMusicalKeyTestSignal(targetRoot, targetScale, 1.8, 48000, 1)
	resMono, err := detector.Analyze(monoPCM)
	if err != nil {
		t.Fatalf("Analyze mono failed: %v", err)
	}

	// Stereo (2ch)
	stereoPCM := GenerateMusicalKeyTestSignal(targetRoot, targetScale, 1.8, 48000, 2)
	resStereo, err := detector.Analyze(stereoPCM)
	if err != nil {
		t.Fatalf("Analyze stereo failed: %v", err)
	}

	// 5.1 Surround (6ch)
	surroundPCM := GenerateMusicalKeyTestSignal(targetRoot, targetScale, 1.8, 48000, 6)
	resSurround, err := detector.Analyze(surroundPCM)
	if err != nil {
		t.Fatalf("Analyze surround failed: %v", err)
	}

	if resMono.Key != expectedKey || resStereo.Key != expectedKey || resSurround.Key != expectedKey {
		t.Errorf("Channel variance detected: Mono=%s, Stereo=%s, Surround=%s",
			resMono.Key, resStereo.Key, resSurround.Key)
	}

	if resMono.Camelot != expectedCamelot || resStereo.Camelot != expectedCamelot || resSurround.Camelot != expectedCamelot {
		t.Errorf("Channel Camelot variance detected: Mono=%s, Stereo=%s, Surround=%s",
			resMono.Camelot, resStereo.Camelot, resSurround.Camelot)
	}
}

func TestKey_EdgeCases(t *testing.T) {
	detector, _ := NewKeyDetector(DefaultKeyDetectorConfig())

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

	// Too short (< 0.5s)
	shortPCM := GenerateMusicalKeyTestSignal("C", ScaleMajor, 0.3, 48000, 2)
	_, err = detector.Analyze(shortPCM)
	if err != ErrAudioTooShortForKey {
		t.Errorf("expected ErrAudioTooShortForKey, got %v", err)
	}

	// Silent buffer
	silentSamples := make([][]float64, 1)
	silentSamples[0] = make([]float64, 48000*2)
	silentPCM := &PCMBuffer{Samples: silentSamples, Channels: 1, SampleRate: 48000, NumSamples: 48000 * 2}
	_, err = detector.Analyze(silentPCM)
	if err != ErrSilentAudio {
		t.Errorf("expected ErrSilentAudio on silence, got %v", err)
	}

	// Invalid config
	_, err = NewKeyDetector(KeyDetectorConfig{MinFreqHz: 0, MaxFreqHz: 100})
	if err != ErrInvalidConfig {
		t.Errorf("expected ErrInvalidConfig, got %v", err)
	}
}

func TestKey_100HeterogeneousStreams_DoD(t *testing.T) {
	detector, err := NewKeyDetector(DefaultKeyDetectorConfig())
	if err != nil {
		t.Fatalf("NewKeyDetector failed: %v", err)
	}

	sampleRates := []int{44100, 48000, 96000}
	channelConfigs := []int{1, 2, 6}
	scales := []ScaleType{ScaleMajor, ScaleMinor}

	passedCount := 0
	var minConfidence float64 = 1.0

	for i := 1; i <= 100; i++ {
		sr := sampleRates[i%len(sampleRates)]
		ch := channelConfigs[i%len(channelConfigs)]
		dur := 1.2 + float64(i%10)*0.15 // 1.2s to 2.55s

		// Cycle across all 12 roots and 2 scales
		root := PitchClassNames[(i*7)%12]
		scale := scales[i%len(scales)]

		targetKey := fmt.Sprintf("%s %s", root, scale)
		expectedCamelot, _ := GetCamelotKey(root, scale)

		pcm := GenerateMusicalKeyTestSignal(root, scale, dur, sr, ch)

		// Inject subtle background noise to test real acoustic resilience
		for c := 0; c < ch; c++ {
			for n := 0; n < pcm.NumSamples; n++ {
				noise := (float64((n*31+i*17)%100)/50.0 - 1.0) * 0.005
				pcm.Samples[c][n] += noise
			}
		}

		res, err := detector.Analyze(pcm)
		if err != nil {
			t.Fatalf("[%d/100] Analyze failed for %s: %v", i, targetKey, err)
		}

		if res.Confidence < minConfidence {
			minConfidence = res.Confidence
		}

		if res.Key == targetKey && res.Camelot == expectedCamelot {
			passedCount++
		} else {
			t.Errorf("[%d/100] Key detection failed: Target=%s (%s), Detected=%s (%s), Confidence=%.2f",
				i, targetKey, expectedCamelot, res.Key, res.Camelot, res.Confidence)
		}
	}

	t.Logf("100 Heterogeneous Key Analysis Results:")
	t.Logf("  Pass Rate (Key & Camelot exact match): %d/100 (%.1f%%) [DoD Requirement 100%%]",
		passedCount, float64(passedCount))
	t.Logf("  Minimum Observed Confidence:          %.3f", minConfidence)

	if passedCount < 100 {
		t.Fatalf("Key detection DoD test failed: %d/100 passed", passedCount)
	}
}
