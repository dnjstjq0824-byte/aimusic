package audio

import (
	"errors"
	"math"
	"testing"
)

// generateRhythmicBeat generates a series of transient pulses at the specified BPM
func generateRhythmicBeat(bpm float64, sampleRate int, durationSec float64) *PCMBuffer {
	numSamples := int(float64(sampleRate) * durationSec)
	left := make([]float64, numSamples)
	right := make([]float64, numSamples)

	beatIntervalSamples := int(float64(sampleRate) * (60.0 / bpm))
	pulseLen := int(float64(sampleRate) * 0.05) // 50ms percussive kick

	for start := 0; start < numSamples; start += beatIntervalSamples {
		for i := 0; i < pulseLen && start+i < numSamples; i++ {
			tSec := float64(i) / float64(sampleRate)
			// Pitch drop kick simulation (150Hz -> 50Hz) + noise transient
			freq := 150.0 - (100.0 * float64(i) / float64(pulseLen))
			env := 1.0 - (float64(i) / float64(pulseLen))
			v := 0.8 * env * math.Sin(2.0*math.Pi*freq*tSec)
			left[start+i] = v
			right[start+i] = v
		}
	}

	return &PCMBuffer{
		Channels:   2,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{left, right},
	}
}

// generateChordProgression synthesizes multi-note chords over sequential measures
func generateChordProgression(chordTones [][]float64, chordDurationSec float64, sampleRate int) *PCMBuffer {
	numChords := len(chordTones)
	chordSamples := int(float64(sampleRate) * chordDurationSec)
	totalSamples := chordSamples * numChords

	left := make([]float64, totalSamples)
	right := make([]float64, totalSamples)

	for cIdx, freqs := range chordTones {
		start := cIdx * chordSamples
		for i := 0; i < chordSamples; i++ {
			tSec := float64(i) / float64(sampleRate)
			var sum float64
			for _, f := range freqs {
				sum += 0.25 * math.Sin(2.0*math.Pi*f*tSec)
			}
			left[start+i] = sum
			right[start+i] = sum
		}
	}

	return &PCMBuffer{
		Channels:   2,
		SampleRate: sampleRate,
		NumSamples: totalSamples,
		Samples:    [][]float64{left, right},
	}
}

func TestAnalyzer_BPMDetection_120BPM(t *testing.T) {
	sampleRate := 48000
	buf := generateRhythmicBeat(120.0, sampleRate, 6.0)

	analyzer := NewAudioAnalyzer()
	features, err := analyzer.Analyze(buf)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	t.Logf("120 BPM Beat Analysis: Detected BPM = %.1f, Confidence = %.2f, Danceability = %.2f, Genre = %s",
		features.BPM, features.BPMConfidence, features.Danceability, features.EstimatedGenre)

	if math.Abs(features.BPM-120.0) > 2.5 {
		t.Errorf("BPM detection mismatch: expected ~120.0 BPM, got %.1f BPM", features.BPM)
	}
	if features.Danceability < 0.35 {
		t.Errorf("Expected substantial danceability for rhythmic beat, got %.2f", features.Danceability)
	}
}

func TestAnalyzer_BPMDetection_140BPM(t *testing.T) {
	sampleRate := 48000
	buf := generateRhythmicBeat(140.0, sampleRate, 6.0)

	features, err := ExtractAudioFeatures(buf)
	if err != nil {
		t.Fatalf("ExtractAudioFeatures failed: %v", err)
	}

	t.Logf("140 BPM Beat Analysis: Detected BPM = %.1f, Confidence = %.2f",
		features.BPM, features.BPMConfidence)

	if math.Abs(features.BPM-140.0) > 2.5 {
		t.Errorf("BPM detection mismatch: expected ~140.0 BPM, got %.1f BPM", features.BPM)
	}
}

func TestAnalyzer_KeyDetection_CMajor(t *testing.T) {
	// C Major triad progression: C-E-G (C), F-A-C (F), G-B-D (G), C-E-G (C)
	chords := [][]float64{
		{261.63, 329.63, 392.00}, // C
		{349.23, 440.00, 523.25}, // F
		{392.00, 493.88, 587.33}, // G
		{261.63, 329.63, 392.00}, // C
	}
	buf := generateChordProgression(chords, 1.0, 48000)

	analyzer := NewAudioAnalyzer()
	features, err := analyzer.Analyze(buf)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	t.Logf("C Major Chord Progression: Key = %s, Mode = %s, Signature = %s, Confidence = %.2f",
		features.Key, features.Mode, features.KeySignature, features.KeyConfidence)

	if features.Key != "C" || features.Mode != ModeMajor {
		t.Errorf("Expected C Major, got %s", features.KeySignature)
	}
}

func TestAnalyzer_KeyDetection_AMinor(t *testing.T) {
	// A Minor natural progression: A-C-E (Am), D-F-A (Dm), E-G-B (Em), A-C-E (Am)
	chords := [][]float64{
		{220.00, 261.63, 329.63}, // Am
		{293.66, 349.23, 440.00}, // Dm
		{329.63, 392.00, 493.88}, // Em
		{220.00, 261.63, 329.63}, // Am
	}
	buf := generateChordProgression(chords, 1.0, 48000)

	features, err := ExtractAudioFeatures(buf)
	if err != nil {
		t.Fatalf("ExtractAudioFeatures failed: %v", err)
	}

	t.Logf("A Minor Chord Progression: Key = %s, Mode = %s, Signature = %s, Confidence = %.2f, Chroma = %v",
		features.Key, features.Mode, features.KeySignature, features.KeyConfidence, features.Chromagram)

	if features.Key != "A" || features.Mode != ModeMinor {
		t.Errorf("Expected A Minor, got %s", features.KeySignature)
	}
}

func TestAnalyzer_EnergyAndValence(t *testing.T) {
	sampleRate := 48000
	numSamples := sampleRate * 2

	// Track 1: High energy, high frequency chord (bright & upbeat)
	bufHigh := &PCMBuffer{
		Channels:   2,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples: [][]float64{
			make([]float64, numSamples),
			make([]float64, numSamples),
		},
	}
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		v := 0.5*math.Sin(2.0*math.Pi*1200.0*tSec) + 0.3*math.Sin(2.0*math.Pi*2400.0*tSec)
		bufHigh.Samples[0][i] = v
		bufHigh.Samples[1][i] = v
	}

	// Track 2: Low energy, low frequency tone (dark & mellow)
	bufLow := &PCMBuffer{
		Channels:   2,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples: [][]float64{
			make([]float64, numSamples),
			make([]float64, numSamples),
		},
	}
	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		v := 0.08 * math.Sin(2.0*math.Pi*110.0*tSec)
		bufLow.Samples[0][i] = v
		bufLow.Samples[1][i] = v
	}

	analyzer := NewAudioAnalyzer()
	featHigh, err1 := analyzer.Analyze(bufHigh)
	featLow, err2 := analyzer.Analyze(bufLow)
	if err1 != nil || err2 != nil {
		t.Fatalf("Analysis failed: %v, %v", err1, err2)
	}

	t.Logf("High Energy Track: Energy = %.3f, Valence = %.3f, Centroid = %.1f Hz",
		featHigh.Energy, featHigh.Valence, featHigh.SpectralCentroid)
	t.Logf("Low Energy Track: Energy = %.3f, Valence = %.3f, Centroid = %.1f Hz",
		featLow.Energy, featLow.Valence, featLow.SpectralCentroid)

	if featHigh.Energy <= featLow.Energy {
		t.Errorf("Expected featHigh.Energy (%f) > featLow.Energy (%f)", featHigh.Energy, featLow.Energy)
	}
	if featHigh.SpectralCentroid <= featLow.SpectralCentroid {
		t.Errorf("Expected featHigh.Centroid (%f) > featLow.Centroid (%f)",
			featHigh.SpectralCentroid, featLow.SpectralCentroid)
	}
}

func TestAnalyzer_EdgeCases(t *testing.T) {
	analyzer := NewAudioAnalyzer()

	// 1. Nil buffer
	_, err := analyzer.Analyze(nil)
	if !errors.Is(err, ErrNilFingerprintBuffer) {
		t.Errorf("Expected ErrNilFingerprintBuffer, got %v", err)
	}

	// 2. Empty buffer
	emptyBuf := &PCMBuffer{Channels: 0, SampleRate: 48000, NumSamples: 0, Samples: nil}
	_, err = analyzer.Analyze(emptyBuf)
	if !errors.Is(err, ErrEmptyAudioBuffer) {
		t.Errorf("Expected ErrEmptyAudioBuffer, got %v", err)
	}

	// 3. Very short buffer (< 2048 samples) should return graceful defaults
	shortBuf := &PCMBuffer{
		Channels:   1,
		SampleRate: 48000,
		NumSamples: 500,
		Samples:    [][]float64{make([]float64, 500)},
	}
	featShort, err := analyzer.Analyze(shortBuf)
	if err != nil {
		t.Fatalf("Short buffer analyze failed: %v", err)
	}
	if featShort.BPM <= 0 || featShort.KeySignature == "" {
		t.Errorf("Expected valid default features on short buffer, got %+v", featShort)
	}
}

func TestAnalyzer100HeterogeneousTracks(t *testing.T) {
	analyzer := NewAudioAnalyzer()

	count := 0
	bpms := []float64{70.0, 80.0, 90.0, 100.0, 110.0, 120.0, 128.0, 140.0, 150.0, 160.0}
	pitches := []float64{130.81, 164.81, 196.00, 220.00, 261.63, 293.66, 329.63, 392.00, 440.00, 523.25}

	for _, bpm := range bpms {
		for _, pitch := range pitches {
			count++
			sampleRate := 48000
			channels := (count % 2) + 1
			duration := 1.5
			numSamples := int(float64(sampleRate) * duration)

			samples := make([][]float64, channels)
			for ch := 0; ch < channels; ch++ {
				samples[ch] = make([]float64, numSamples)
				for n := 0; n < numSamples; n++ {
					tSec := float64(n) / float64(sampleRate)
					samples[ch][n] = 0.5 * math.Sin(2.0*math.Pi*pitch*tSec)
				}
			}

			buf := &PCMBuffer{
				Channels:   channels,
				SampleRate: sampleRate,
				NumSamples: numSamples,
				Samples:    samples,
			}

			features, err := analyzer.Analyze(buf)
			if err != nil {
				t.Fatalf("Track %d (BPM=%.0f, Pitch=%.1f) failed: %v", count, bpm, pitch, err)
			}

			if features.BPM < 50.0 || features.BPM > 220.0 {
				t.Fatalf("Track %d: BPM %.1f out of realistic musical range", count, features.BPM)
			}
			if features.Key == "" || features.KeySignature == "" {
				t.Fatalf("Track %d: missing key signature", count)
			}
			if math.IsNaN(features.Energy) || math.IsInf(features.Energy, 0) {
				t.Fatalf("Track %d: NaN or Inf energy", count)
			}
			if math.IsNaN(features.Danceability) || math.IsInf(features.Danceability, 0) {
				t.Fatalf("Track %d: NaN or Inf danceability", count)
			}
			if math.IsNaN(features.Valence) || math.IsInf(features.Valence, 0) {
				t.Fatalf("Track %d: NaN or Inf valence", count)
			}

			if count >= 100 {
				break
			}
		}
		if count >= 100 {
			break
		}
	}

	t.Logf("Successfully analyzed %d heterogeneous musical tracks with 0 errors.", count)
}
