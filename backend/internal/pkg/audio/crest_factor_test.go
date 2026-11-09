package audio

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestCrestFactor_MathematicalSignals_SineSquareTriangle(t *testing.T) {
	sampleRate := 48000
	numSamples := sampleRate * 1 // 1.0 second
	analyzer := NewCrestFactorAnalyzer(DefaultCrestFactorConfig())

	// 1. Sine wave: Theoretical CF = sqrt(2) = 1.41421356, CF_dB = 3.0103 dB, AES17 RMS = 0.00 dBFS
	sineSamples := make([][]float64, 1)
	sineSamples[0] = make([]float64, numSamples)
	freq := 1000.0 // 1 kHz
	for i := 0; i < numSamples; i++ {
		sineSamples[0][i] = math.Sin(2.0 * math.Pi * freq * float64(i) / float64(sampleRate))
	}
	sinePCM := &PCMBuffer{Samples: sineSamples, Channels: 1, SampleRate: sampleRate, NumSamples: numSamples}

	sineRes, err := analyzer.Analyze(sinePCM)
	if err != nil {
		t.Fatalf("Analyze sine failed: %v", err)
	}

	expectedSineCF := math.Sqrt(2.0)
	expectedSineCFDb := 20.0 * math.Log10(expectedSineCF) // 3.0103 dB
	if math.Abs(sineRes.CrestFactorRatio-expectedSineCF) > 0.01 {
		t.Errorf("Sine CrestFactorRatio: expected %.4f, got %.4f", expectedSineCF, sineRes.CrestFactorRatio)
	}
	if math.Abs(sineRes.CrestFactorDb-expectedSineCFDb) > 0.05 {
		t.Errorf("Sine CrestFactorDb: expected %.4f dB, got %.4f dB", expectedSineCFDb, sineRes.CrestFactorDb)
	}
	if math.Abs(sineRes.GlobalAES17RMSDbFS-0.0) > 0.05 {
		t.Errorf("Sine AES-17 RMS: expected 0.00 dBFS, got %.4f dBFS", sineRes.GlobalAES17RMSDbFS)
	}

	// 2. Square wave: Theoretical CF = 1.0000, CF_dB = 0.0000 dB
	squareSamples := make([][]float64, 1)
	squareSamples[0] = make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		if math.Sin(2.0*math.Pi*freq*float64(i)/float64(sampleRate)) >= 0 {
			squareSamples[0][i] = 1.0
		} else {
			squareSamples[0][i] = -1.0
		}
	}
	squarePCM := &PCMBuffer{Samples: squareSamples, Channels: 1, SampleRate: sampleRate, NumSamples: numSamples}

	squareRes, err := analyzer.Analyze(squarePCM)
	if err != nil {
		t.Fatalf("Analyze square failed: %v", err)
	}

	if math.Abs(squareRes.CrestFactorRatio-1.0) > 0.01 {
		t.Errorf("Square CrestFactorRatio: expected 1.0000, got %.4f", squareRes.CrestFactorRatio)
	}
	if math.Abs(squareRes.CrestFactorDb-0.0) > 0.05 {
		t.Errorf("Square CrestFactorDb: expected 0.00 dB, got %.4f dB", squareRes.CrestFactorDb)
	}

	// 3. Triangle wave: Theoretical RMS = 1/sqrt(3) = 0.57735, CF = sqrt(3) = 1.73205, CF_dB = 4.7712 dB
	triSamples := make([][]float64, 1)
	triSamples[0] = make([]float64, numSamples)
	period := float64(sampleRate) / freq
	for i := 0; i < numSamples; i++ {
		phase := math.Mod(float64(i), period) / period // 0 to 1
		triSamples[0][i] = 2.0*math.Abs(2.0*(phase-math.Floor(phase+0.5))) - 1.0
	}
	triPCM := &PCMBuffer{Samples: triSamples, Channels: 1, SampleRate: sampleRate, NumSamples: numSamples}

	triRes, err := analyzer.Analyze(triPCM)
	if err != nil {
		t.Fatalf("Analyze triangle failed: %v", err)
	}

	expectedTriCF := math.Sqrt(3.0)
	expectedTriCFDb := 20.0 * math.Log10(expectedTriCF) // 4.7712 dB
	if math.Abs(triRes.CrestFactorRatio-expectedTriCF) > 0.02 {
		t.Errorf("Triangle CrestFactorRatio: expected %.4f, got %.4f", expectedTriCF, triRes.CrestFactorRatio)
	}
	if math.Abs(triRes.CrestFactorDb-expectedTriCFDb) > 0.10 {
		t.Errorf("Triangle CrestFactorDb: expected %.4f dB, got %.4f dB", expectedTriCFDb, triRes.CrestFactorDb)
	}

	t.Logf("Theoretical verification passed: Sine CF=%.2f dB, Square CF=%.2f dB, Triangle CF=%.2f dB",
		sineRes.CrestFactorDb, squareRes.CrestFactorDb, triRes.CrestFactorDb)
}

func TestCrestFactor_LoudnessWar_OverCompressedDetection(t *testing.T) {
	sampleRate := 48000
	numSamples := sampleRate * 2
	samples := make([][]float64, 2)
	samples[0] = make([]float64, numSamples)
	samples[1] = make([]float64, numSamples)

	// Simulate brickwall-limited commercial master with severe compression (CF ~ 5.5 dB)
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		raw := math.Sin(2*math.Pi*120.0*tSec) + 0.8*math.Sin(2*math.Pi*350.0*tSec) + 0.5*math.Sin(2*math.Pi*1200.0*tSec)
		// Hard limiter / clipping compression at threshold 0.85
		amplified := raw * 2.5
		clipped := math.Max(-0.98, math.Min(0.98, amplified))
		samples[0][i] = clipped
		samples[1][i] = clipped
	}

	pcm := &PCMBuffer{Samples: samples, Channels: 2, SampleRate: sampleRate, NumSamples: numSamples}
	analyzer := NewCrestFactorAnalyzer(DefaultCrestFactorConfig())

	res, err := analyzer.Analyze(pcm)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if !res.IsOverCompressed {
		t.Errorf("expected IsOverCompressed=true for squashed master, got false (CF=%.2f dB)", res.CrestFactorDb)
	}
	if res.CrestFactorDb >= 8.0 {
		t.Errorf("expected CrestFactorDb < 8.0 dB for Loudness War casualty, got %.2f dB", res.CrestFactorDb)
	}
	if res.Rating != RatingLoudnessWar && res.Rating != RatingExtremeSquashed {
		t.Errorf("expected RatingLoudnessWar or RatingExtremeSquashed, got %v", res.Rating)
	}

	t.Logf("Loudness War detected: CF=%.2f dB, DR=%d, Rating=%s, OverCompressed=%v",
		res.CrestFactorDb, res.DRScore, res.Rating, res.IsOverCompressed)
}

func TestCrestFactor_AudiophileDynamicMaster(t *testing.T) {
	sampleRate := 48000
	numSamples := sampleRate * 2
	samples := make([][]float64, 2)
	samples[0] = make([]float64, numSamples)
	samples[1] = make([]float64, numSamples)

	// Simulate high dynamic range acoustic master (orchestra / jazz)
	// Low background ambient bed with high-transient percussive peaks (CF ~ 18 dB)
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		ambientBed := 0.05 * math.Sin(2.0*math.Pi*220.0*tSec) // -26 dBFS bed

		// Sharp transient percussion hit every 0.5s
		transient := 0.0
		if i%(sampleRate/2) < 48 {
			transient = 0.90 * math.Exp(-float64(i%(sampleRate/2))/12.0)
		}

		val := ambientBed + transient
		samples[0][i] = val
		samples[1][i] = val
	}

	pcm := &PCMBuffer{Samples: samples, Channels: 2, SampleRate: sampleRate, NumSamples: numSamples}
	analyzer := NewCrestFactorAnalyzer(DefaultCrestFactorConfig())

	res, err := analyzer.Analyze(pcm)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if res.IsOverCompressed {
		t.Errorf("expected IsOverCompressed=false for dynamic master, got true (CF=%.2f dB)", res.CrestFactorDb)
	}
	if res.CrestFactorDb < 14.0 {
		t.Errorf("expected CrestFactorDb >= 14.0 dB for audiophile master, got %.2f dB", res.CrestFactorDb)
	}
	if res.DRScore < 14 {
		t.Errorf("expected DRScore >= 14, got %d", res.DRScore)
	}
	if res.Rating != RatingAudiophileDynamic {
		t.Errorf("expected RatingAudiophileDynamic, got %v", res.Rating)
	}

	t.Logf("Audiophile Master verified: CF=%.2f dB, DR=%d, Rating=%s, UIBadge=%s",
		res.CrestFactorDb, res.DRScore, res.Rating, res.UIBadge)
}

func TestCrestFactor_TruePeak_InterSampleOvershoot(t *testing.T) {
	// Synthesize a 12 kHz sine wave at 48 kHz sample rate
	// 48000 / 12000 = exactly 4 samples per cycle!
	// With phase offset = pi/4 (45 degrees):
	// Sample points occur at angles: 45°, 135°, 225°, 315°
	// sin(45°) = sin(135°) = 1/sqrt(2) = 0.707106 (-3.01 dBFS sample peak)!
	// BUT the true continuous analog sine peak is 1.0000 (0.00 dBTP)!
	// This generates an inter-sample overshoot of +3.01 dB!
	sampleRate := 48000
	numSamples := 4800
	samples := make([][]float64, 1)
	samples[0] = make([]float64, numSamples)

	phaseOffset := math.Pi / 4.0 // 45 degrees
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		samples[0][i] = math.Sin(2.0*math.Pi*12000.0*tSec + phaseOffset)
	}

	pcm := &PCMBuffer{Samples: samples, Channels: 1, SampleRate: sampleRate, NumSamples: numSamples}
	analyzer := NewCrestFactorAnalyzer(DefaultCrestFactorConfig())

	res, err := analyzer.Analyze(pcm)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	expectedSamplePeakDb := -3.0103
	actualSamplePeakDb := res.GlobalPeakDbFS
	actualTruePeakDb := res.GlobalTruePeakDbTP

	if math.Abs(actualSamplePeakDb-expectedSamplePeakDb) > 0.05 {
		t.Errorf("expected sample peak ~ -3.01 dBFS, got %.2f dBFS", actualSamplePeakDb)
	}
	overshoot := actualTruePeakDb - actualSamplePeakDb
	if overshoot < 2.5 || overshoot > 4.5 {
		t.Errorf("expected inter-sample overshoot between +2.5 dB and +4.5 dB, got +%.2f dB", overshoot)
	}

	t.Logf("True Peak Overshoot verified: SamplePeak=%.2f dBFS, TruePeak=%.2f dBTP (Inter-sample boost: +%.2f dB)",
		actualSamplePeakDb, actualTruePeakDb, actualTruePeakDb-actualSamplePeakDb)
}

func TestCrestFactor_JSONAndUIBadgeSerialization(t *testing.T) {
	pcm := GenerateSyntheticAcousticAudio(1.0, 48000, 2)
	analyzer := NewCrestFactorAnalyzer(DefaultCrestFactorConfig())

	res, err := analyzer.Analyze(pcm)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	jsonStr, err := res.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Verify valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Unmarshal exported JSON failed: %v", err)
	}

	if _, exists := parsed["crest_factor_db"]; !exists {
		t.Errorf("missing crest_factor_db in JSON")
	}
	if _, exists := parsed["dr_score"]; !exists {
		t.Errorf("missing dr_score in JSON")
	}
	if _, exists := parsed["ui_badge"]; !exists {
		t.Errorf("missing ui_badge in JSON")
	}

	// Verify UI Badge format
	if !strings.HasPrefix(res.UIBadge, "DR") {
		t.Errorf("UIBadge should start with DR, got %q", res.UIBadge)
	}
	t.Logf("Exported UI Badge: %s", res.UIBadge)
}

func TestCrestFactor_EdgeCases(t *testing.T) {
	analyzer := NewCrestFactorAnalyzer(DefaultCrestFactorConfig())

	// Nil buffer
	_, err := analyzer.Analyze(nil)
	if err != ErrNilPCMBuffer {
		t.Errorf("expected ErrNilPCMBuffer, got %v", err)
	}

	// Empty buffer
	_, err = analyzer.Analyze(&PCMBuffer{Channels: 2, SampleRate: 48000})
	if err != ErrEmptyAudioBuffer {
		t.Errorf("expected ErrEmptyAudioBuffer, got %v", err)
	}

	// Invalid sample rate
	invalidPCM := &PCMBuffer{
		Samples:    [][]float64{{0.1, 0.2}},
		Channels:   1,
		SampleRate: 0,
		NumSamples: 2,
	}
	_, err = analyzer.Analyze(invalidPCM)
	if err == nil {
		t.Errorf("expected error on zero sample rate")
	}
}

func TestCrestFactor_100HeterogeneousAudioStreams(t *testing.T) {
	analyzer := NewCrestFactorAnalyzer(DefaultCrestFactorConfig())
	sampleRates := []int{44100, 48000, 96000}
	channelConfigs := []int{1, 2, 6, 8}

	for i := 1; i <= 100; i++ {
		sr := sampleRates[i%len(sampleRates)]
		ch := channelConfigs[i%len(channelConfigs)]
		dur := 0.2 + float64(i%15)*0.1 // 0.2s to 1.6s

		pcm := GenerateSyntheticAcousticAudio(dur, sr, ch)

		res, err := analyzer.Analyze(pcm)
		if err != nil {
			t.Fatalf("[%d/100] Analyze failed (dur=%.2f, sr=%d, ch=%d): %v", i, dur, sr, ch, err)
		}

		// Fundamental invariants of Crest Factor & Peak analysis
		if res.CrestFactorRatio < 1.0 {
			t.Fatalf("[%d/100] CrestFactorRatio < 1.0 (CF=%.4f)", i, res.CrestFactorRatio)
		}
		if res.CrestFactorDb < -0.01 {
			t.Fatalf("[%d/100] CrestFactorDb < 0 dB (CF_dB=%.2f)", i, res.CrestFactorDb)
		}
		if res.GlobalPeakDbFS < res.GlobalRMSDbFS-0.01 {
			t.Fatalf("[%d/100] PeakDbFS < RMSDbFS (Peak=%.2f, RMS=%.2f)", i, res.GlobalPeakDbFS, res.GlobalRMSDbFS)
		}
		if res.GlobalTruePeakDbTP < res.GlobalPeakDbFS-0.01 {
			t.Fatalf("[%d/100] TruePeak < SamplePeak (TruePeak=%.2f, SamplePeak=%.2f)", i, res.GlobalTruePeakDbTP, res.GlobalPeakDbFS)
		}
		if res.DRScore < 1 {
			t.Fatalf("[%d/100] invalid DRScore %d", i, res.DRScore)
		}
		if len(res.ChannelBreakdown) != ch {
			t.Fatalf("[%d/100] channel breakdown length mismatch: expected %d, got %d", i, ch, len(res.ChannelBreakdown))
		}
		if res.UIBadge == "" {
			t.Fatalf("[%d/100] empty UIBadge", i)
		}
	}

	t.Log("Successfully verified 100 heterogeneous audio streams with Crest Factor, True Peak, and DR dynamics.")
}
