package audio

import (
	"math"
	"testing"
)

func TestBandwidthAnalyzer_TrueLossless_Detection(t *testing.T) {
	pcm := GenerateSyntheticAcousticAudio(1.0, 48000, 2)
	analyzer, err := NewBandwidthAnalyzer(DefaultBandwidthConfig())
	if err != nil {
		t.Fatalf("NewBandwidthAnalyzer failed: %v", err)
	}

	result, err := analyzer.Analyze(pcm)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if result.Verdict != VerdictTrueLossless {
		t.Errorf("expected VerdictTrueLossless, got %v", result.Verdict)
	}
	if result.IsUpscaledLossy {
		t.Errorf("expected IsUpscaledLossy=false for genuine audio, got true")
	}
	if result.RollOff99Hz < 17500.0 {
		t.Errorf("expected 99%% roll-off > 17.5 kHz, got %.1f Hz", result.RollOff99Hz)
	}
	if result.HighFrequencyEnergyRatio < 0.01 {
		t.Errorf("expected high-frequency ratio > 0.01, got %.6f", result.HighFrequencyEnergyRatio)
	}

	t.Logf("True Lossless verified: RollOff99=%.1f Hz, HF Ratio=%.4f, DropDb=%.1f dB, Verdict=%s",
		result.RollOff99Hz, result.HighFrequencyEnergyRatio, result.SpectralDropoffDb, result.Verdict)
}

func TestBandwidthAnalyzer_MP3_128k_UpscaledDetection(t *testing.T) {
	origPCM := GenerateSyntheticAcousticAudio(1.0, 48000, 2)

	// Simulate MP3 128k brickwall cutoff at 16.0 kHz
	filteredPCM, err := SimulateLossyCutoff(origPCM, 16000.0)
	if err != nil {
		t.Fatalf("SimulateLossyCutoff failed: %v", err)
	}

	analyzer, err := NewBandwidthAnalyzer(DefaultBandwidthConfig())
	if err != nil {
		t.Fatalf("NewBandwidthAnalyzer failed: %v", err)
	}

	result, err := analyzer.Analyze(filteredPCM)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if result.Verdict != VerdictUpscaledLossy {
		t.Errorf("expected VerdictUpscaledLossy, got %v", result.Verdict)
	}
	if !result.IsUpscaledLossy {
		t.Errorf("expected IsUpscaledLossy=true, got false")
	}
	if result.DetectedCutoffHz < 15500.0 || result.DetectedCutoffHz > 17000.0 {
		t.Errorf("expected cutoff between 15.5k and 17.0k, got %.1f Hz", result.DetectedCutoffHz)
	}
	if result.SpectralDropoffDb < 25.0 {
		t.Errorf("expected spectral dropoff >= 25 dB, got %.1f dB", result.SpectralDropoffDb)
	}
	if result.Confidence < 0.85 {
		t.Errorf("expected confidence >= 0.85, got %.2f", result.Confidence)
	}

	t.Logf("MP3 128k Upscaled detected: Cutoff=%.1f Hz, Drop=%.1f dB, Confidence=%.2f, Est=%s",
		result.DetectedCutoffHz, result.SpectralDropoffDb, result.Confidence, result.EstimatedOriginalBitrate)
}

func TestBandwidthAnalyzer_MP3_192k_Detection(t *testing.T) {
	origPCM := GenerateSyntheticAcousticAudio(1.0, 48000, 2)
	filteredPCM, err := SimulateLossyCutoff(origPCM, 18500.0)
	if err != nil {
		t.Fatalf("SimulateLossyCutoff failed: %v", err)
	}

	analyzer, err := NewBandwidthAnalyzer(DefaultBandwidthConfig())
	if err != nil {
		t.Fatalf("NewBandwidthAnalyzer failed: %v", err)
	}

	result, err := analyzer.Analyze(filteredPCM)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if result.Verdict != VerdictMediumLossy {
		t.Errorf("expected VerdictMediumLossy, got %v", result.Verdict)
	}
	if result.DetectedCutoffHz < 17800.0 || result.DetectedCutoffHz > 19200.0 {
		t.Errorf("expected cutoff around 18.5k, got %.1f Hz", result.DetectedCutoffHz)
	}
}

func TestBandwidthAnalyzer_MP3_64k_96k_LowBitrateDetection(t *testing.T) {
	origPCM := GenerateSyntheticAcousticAudio(1.0, 48000, 2)
	filteredPCM, err := SimulateLossyCutoff(origPCM, 14000.0)
	if err != nil {
		t.Fatalf("SimulateLossyCutoff failed: %v", err)
	}

	analyzer, err := NewBandwidthAnalyzer(DefaultBandwidthConfig())
	if err != nil {
		t.Fatalf("NewBandwidthAnalyzer failed: %v", err)
	}

	result, err := analyzer.Analyze(filteredPCM)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if result.Verdict != VerdictLowBitrate {
		t.Errorf("expected VerdictLowBitrate, got %v", result.Verdict)
	}
	if result.DetectedCutoffHz > 14500.0 {
		t.Errorf("expected cutoff <= 14.5k, got %.1f Hz", result.DetectedCutoffHz)
	}
}

func TestBandwidthAnalyzer_MinimalistAcoustic_NoFalsePositive(t *testing.T) {
	// Pure bass / cello tone at 110 Hz and 220 Hz without high-frequency harmonics
	sampleRate := 48000
	numSamples := sampleRate * 1
	samples := make([][]float64, 2)
	samples[0] = make([]float64, numSamples)
	samples[1] = make([]float64, numSamples)

	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		val := 0.5*math.Sin(2.0*math.Pi*110.0*t) + 0.3*math.Sin(2.0*math.Pi*220.0*t)
		samples[0][i] = val
		samples[1][i] = val
	}
	bassPCM := &PCMBuffer{
		Samples:    samples,
		Channels:   2,
		SampleRate: sampleRate,
		NumSamples: numSamples,
	}

	analyzer, _ := NewBandwidthAnalyzer(DefaultBandwidthConfig())
	result, err := analyzer.Analyze(bassPCM)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Must NOT trigger an upscaled lossy verdict on natural bass/minimalist audio
	if result.Verdict == VerdictUpscaledLossy || result.IsUpscaledLossy {
		t.Errorf("false positive: minimalist bass signal was misclassified as upscaled lossy!")
	}
	if result.Verdict != VerdictNaturalAcoustic && result.Verdict != VerdictTrueLossless {
		t.Errorf("expected VerdictNaturalAcoustic or VerdictTrueLossless, got %v", result.Verdict)
	}
}

func TestBandwidthAnalyzer_SampleRateEdgeCases(t *testing.T) {
	analyzer, _ := NewBandwidthAnalyzer(DefaultBandwidthConfig())

	// Sample rate too low (< 32 kHz)
	lowSrPCM := GenerateSyntheticAcousticAudio(0.5, 22050, 1)
	_, err := analyzer.Analyze(lowSrPCM)
	if err != ErrSampleRateTooLow {
		t.Errorf("expected ErrSampleRateTooLow, got %v", err)
	}

	// Nil and empty buffers
	_, err = analyzer.Analyze(nil)
	if err != ErrNilPCMBuffer {
		t.Errorf("expected ErrNilPCMBuffer, got %v", err)
	}

	_, err = analyzer.Analyze(&PCMBuffer{Channels: 2, SampleRate: 48000})
	if err != ErrEmptyAudioBuffer {
		t.Errorf("expected ErrEmptyAudioBuffer, got %v", err)
	}

	// 96 kHz Hi-Res stream analysis
	hiResPCM := GenerateSyntheticAcousticAudio(0.5, 96000, 2)
	hiResRes, err := analyzer.Analyze(hiResPCM)
	if err != nil {
		t.Fatalf("Analyze 96kHz failed: %v", err)
	}
	if hiResRes.Verdict != VerdictTrueLossless {
		t.Errorf("expected VerdictTrueLossless for 96kHz genuine audio, got %v", hiResRes.Verdict)
	}
}

func TestBandwidthAnalyzer_100HeterogeneousStreams_99PercentDoD(t *testing.T) {
	analyzer, err := NewBandwidthAnalyzer(DefaultBandwidthConfig())
	if err != nil {
		t.Fatalf("NewBandwidthAnalyzer failed: %v", err)
	}

	sampleRates := []int{44100, 48000, 96000}
	channelConfigs := []int{1, 2, 6}

	lossyTestCount := 0
	lossyDetectedCount := 0
	losslessTestCount := 0
	losslessCorrectCount := 0

	for i := 1; i <= 100; i++ {
		sr := sampleRates[i%len(sampleRates)]
		ch := channelConfigs[i%len(channelConfigs)]
		dur := 0.3 + float64(i%10)*0.08 // 0.3s to 1.02s

		pcm := GenerateSyntheticAcousticAudio(dur, sr, ch)

		isLossyTrial := (i % 2) == 0

		if isLossyTrial {
			lossyTestCount++
			// Simulate MP3 128k cutoff (between 15.8 kHz and 16.2 kHz)
			cutoffHz := 15800.0 + float64(i%5)*100.0
			filtered, err := SimulateLossyCutoff(pcm, cutoffHz)
			if err != nil {
				t.Fatalf("[%d/100] SimulateLossyCutoff failed: %v", i, err)
			}

			res, err := analyzer.Analyze(filtered)
			if err != nil {
				t.Fatalf("[%d/100] Analyze lossy failed: %v", i, err)
			}

			if res.IsUpscaledLossy && res.Verdict == VerdictUpscaledLossy {
				lossyDetectedCount++
			} else {
				t.Errorf("[%d/100] Failed to detect upscaled MP3 128k! Cutoff=%.1f, Drop=%.1f dB, Verdict=%s",
					i, res.DetectedCutoffHz, res.SpectralDropoffDb, res.Verdict)
			}
		} else {
			losslessTestCount++
			res, err := analyzer.Analyze(pcm)
			if err != nil {
				t.Fatalf("[%d/100] Analyze lossless failed: %v", i, err)
			}

			if !res.IsUpscaledLossy && res.Verdict == VerdictTrueLossless {
				losslessCorrectCount++
			} else {
				t.Errorf("[%d/100] False positive on genuine lossless audio! Verdict=%s", i, res.Verdict)
			}
		}
	}

	lossyAccuracy := float64(lossyDetectedCount) / float64(lossyTestCount) * 100.0
	losslessAccuracy := float64(losslessCorrectCount) / float64(losslessTestCount) * 100.0

	t.Logf("Results across 100 heterogeneous streams:")
	t.Logf("  MP3 128k Upscaled Detection: %d/%d (%.2f%%) [DoD Requirement >= 99%%]", lossyDetectedCount, lossyTestCount, lossyAccuracy)
	t.Logf("  Genuine Lossless Precision:  %d/%d (%.2f%%)", losslessCorrectCount, losslessTestCount, losslessAccuracy)

	if lossyAccuracy < 99.0 {
		t.Fatalf("MP3 128k detection accuracy %.2f%% failed DoD requirement (>= 99.0%%)", lossyAccuracy)
	}
	if losslessAccuracy < 99.0 {
		t.Fatalf("Lossless precision %.2f%% failed (false positives detected)", losslessAccuracy)
	}
}
