package audio

import (
	"math"
	"testing"
)

func TestVocalRatioAnalyzer_EmptyBuffer(t *testing.T) {
	analyzer := NewVocalRatioAnalyzer(DefaultVocalRatioConfig(44100))
	_, err := analyzer.Analyze([][]float64{})
	if err == nil {
		t.Fatalf("expected error for empty audio buffer, got nil")
	}
}

func TestVocalRatioAnalyzer_PureSineVocal(t *testing.T) {
	sampleRate := 44100
	duration := 1.0 // 1 sec
	numSamples := int(float64(sampleRate) * duration)

	// Create 1 kHz sine wave panned dead center (simulating lead vocal)
	freq := 1000.0
	chL := make([]float64, numSamples)
	chR := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		sample := 0.6 * math.Sin(2.0*math.Pi*freq*float64(i)/float64(sampleRate))
		chL[i] = sample
		chR[i] = sample
	}

	analyzer := NewVocalRatioAnalyzer(DefaultVocalRatioConfig(sampleRate))
	res, err := analyzer.Analyze([][]float64{chL, chR})
	if err != nil {
		t.Fatalf("failed to analyze pure vocal: %v", err)
	}

	if !res.HasLeadVocal {
		t.Errorf("expected HasLeadVocal=true, got false")
	}
	if res.Dominance != DominanceAcappella && res.Dominance != DominanceVocalForward {
		t.Errorf("expected Acappella or VocalForward dominance, got %s", res.Dominance)
	}
	if res.CenterPannedRatio < 0.9 {
		t.Errorf("expected high center panned ratio for mono/center vocal, got %f", res.CenterPannedRatio)
	}
	if res.IntelligibilityScore < 0.7 {
		t.Errorf("expected high intelligibility score (>0.7), got %f", res.IntelligibilityScore)
	}
}

func TestVocalRatioAnalyzer_PureInstrumentalLowSub(t *testing.T) {
	sampleRate := 44100
	numSamples := 44100 // 1 sec

	// 60 Hz Sub-Bass sine wave (outside vocal band)
	freq := 60.0
	chL := make([]float64, numSamples)
	chR := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		sample := 0.8 * math.Sin(2.0*math.Pi*freq*float64(i)/float64(sampleRate))
		chL[i] = sample
		chR[i] = sample
	}

	analyzer := NewVocalRatioAnalyzer(DefaultVocalRatioConfig(sampleRate))
	res, err := analyzer.Analyze([][]float64{chL, chR})
	if err != nil {
		t.Fatalf("failed to analyze sub-bass: %v", err)
	}

	if res.HasLeadVocal {
		t.Errorf("expected HasLeadVocal=false for 60Hz sub-bass, got true")
	}
	if res.Dominance != DominancePureInstrumental {
		t.Errorf("expected PureInstrumental, got %s", res.Dominance)
	}
	if res.IntelligibilityScore != 0.0 {
		t.Errorf("expected IntelligibilityScore=0.0, got %f", res.IntelligibilityScore)
	}
}

func TestVocalRatioAnalyzer_StereoPannedHighInstruments(t *testing.T) {
	sampleRate := 44100
	numSamples := 44100

	// 10 kHz Cymbals panned hard left and right (out of phase)
	freq := 10000.0
	chL := make([]float64, numSamples)
	chR := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		sample := 0.5 * math.Sin(2.0*math.Pi*freq*float64(i)/float64(sampleRate))
		chL[i] = sample
		chR[i] = -sample // Anti-phase, pure side channel
	}

	analyzer := NewVocalRatioAnalyzer(DefaultVocalRatioConfig(sampleRate))
	res, err := analyzer.Analyze([][]float64{chL, chR})
	if err != nil {
		t.Fatalf("failed to analyze hard-panned cymbals: %v", err)
	}

	if res.HasLeadVocal {
		t.Errorf("expected HasLeadVocal=false for stereo cymbals, got true")
	}
	if res.Bands.HighEndAirDb < -30.0 {
		t.Errorf("expected significant high end air energy, got %f", res.Bands.HighEndAirDb)
	}
	if res.Bands.SideStereoDb < -30.0 {
		t.Errorf("expected significant side stereo energy, got %f", res.Bands.SideStereoDb)
	}
}

func TestVocalRatioResult_ToJSON(t *testing.T) {
	res := &VocalRatioResult{
		SampleRate:        44100,
		Channels:          2,
		DurationSec:       3.0,
		VIRDb:             1.25,
		Dominance:         DominanceBalancedMix,
		HasLeadVocal:      true,
		CenterPannedRatio: 0.85,
	}
	jsonStr, err := res.ToJSON()
	if err != nil {
		t.Fatalf("failed to serialize ToJSON: %v", err)
	}
	if len(jsonStr) == 0 {
		t.Fatalf("expected non-empty JSON string")
	}
}
