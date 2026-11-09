// Package forensics provides AI audio provenance identification, Suno v3 high-frequency
// cutoff analysis, comb-filtering periodic resonance detection, and synthetic model fingerprinting.
package forensics

import (
	"encoding/json"
	"math"

	"avibe-platform/backend/internal/pkg/audio"
)

// SunoAnalysisResult encapsulates forensic indicators of Suno v3 / v3.5 audio generation.
type SunoAnalysisResult struct {
	IsSunoGenerated       bool    `json:"is_suno_generated"`        // Flagged if confidence > 0.80
	ConfidenceScore       float64 `json:"confidence_score"`         // 0.0 to 1.0
	CutoffDetected        bool    `json:"cutoff_detected"`          // True if steep ~16.5kHz shelf observed
	CutoffFrequencyHz     float64 `json:"cutoff_frequency_hz"`
	HighFreqAttenuationDb float64 `json:"high_freq_attenuation_db"` // dB loss between 15kHz and 18kHz
	CombFilterPeriodicR   float64 `json:"comb_filter_periodic_r"`   // Autocorrelation peak of spectral ripple (0..1)
	DiagnosticSummary     string  `json:"diagnostic_summary"`
}

// SunoDetector detects signature artifacts of Suno diffusion audio models.
type SunoDetector struct {
	stftEngine *audio.STFTEngine
}

// NewSunoDetector instantiates a Suno forensics analyzer.
func NewSunoDetector() (*SunoDetector, error) {
	engine, err := audio.NewSTFTEngine(audio.STFTConfig{
		FFTSize:    2048,
		HopSize:    512,
		Window:     audio.WindowHann,
		NumMelBins: 32,
	})
	if err != nil {
		return nil, err
	}
	return &SunoDetector{stftEngine: engine}, nil
}

// Analyze scans the audio buffer for Suno v3 16.5kHz cutoff and comb filter ripples.
func (sd *SunoDetector) Analyze(input *audio.PCMBuffer) (*SunoAnalysisResult, error) {
	if input == nil {
		return nil, audio.ErrNilPCMBuffer
	}
	if input.NumSamples == 0 || input.Channels == 0 || len(input.Samples) == 0 {
		return nil, audio.ErrEmptyPCMBuffer
	}

	sampleRate := input.SampleRate
	if sampleRate < 44100 {
		return &SunoAnalysisResult{
			IsSunoGenerated:   false,
			ConfidenceScore:   0.0,
			DiagnosticSummary: "Sample rate too low for ultrasonic forensic analysis",
		}, nil
	}

	stftRes, err := sd.stftEngine.Forward(input)
	if err != nil {
		return nil, err
	}

	numFrames := stftRes.NumFrames
	numBins := stftRes.NumBins
	binHz := float64(sampleRate) / 2048.0

	// Step 1: Average power spectrum across all frames
	avgSpectrum := make([]float64, numBins)
	frames := stftRes.Frames[0]
	for f := 0; f < numFrames; f++ {
		pwr := frames[f].Power
		for k := 0; k < numBins; k++ {
			avgSpectrum[k] += pwr[k]
		}
	}
	scale := 1.0 / float64(numFrames)
	for k := 0; k < numBins; k++ {
		avgSpectrum[k] *= scale
	}

	// Step 2: Measure energy before vs after 16.5 kHz
	bin14k := int(14000.0 / binHz)
	bin16k := int(16500.0 / binHz)
	bin18k := int(18500.0 / binHz)
	if bin18k >= numBins {
		bin18k = numBins - 1
	}

	var pwrPre16k, pwrPost16k float64
	for k := bin14k; k < bin16k; k++ {
		pwrPre16k += avgSpectrum[k]
	}
	for k := bin16k; k < bin18k; k++ {
		pwrPost16k += avgSpectrum[k]
	}

	pwrPre16k /= float64(bin16k - bin14k)
	pwrPost16k /= float64(bin18k - bin16k)

	attenuationDb := 0.0
	if pwrPost16k > 1e-12 && pwrPre16k > 1e-12 {
		attenuationDb = 10.0 * math.Log10(pwrPre16k / pwrPost16k)
	} else if pwrPre16k > 1e-12 {
		attenuationDb = 60.0 // Near zero energy above 16.5kHz
	}

	cutoffDetected := attenuationDb >= 25.0 // Suno models drop > 25dB above 16.5kHz

	// Step 3: Comb Filter Ripple Detection via autocorrelation of high-mid spectrum (4kHz..12kHz)
	bin4k := int(4000.0 / binHz)
	bin12k := int(12000.0 / binHz)
	sliceLen := bin12k - bin4k
	slice := make([]float64, sliceLen)
	var sliceMean float64
	for i := 0; i < sliceLen; i++ {
		v := math.Log1p(avgSpectrum[bin4k+i] * 1e6)
		slice[i] = v
		sliceMean += v
	}
	sliceMean /= float64(sliceLen)
	for i := 0; i < sliceLen; i++ {
		slice[i] -= sliceMean
	}

	// Autocorrelation to detect periodic comb notches
	var maxRipple float64
	maxLag := int(500.0 / binHz) // up to ~500 Hz ripple spacing
	minLag := int(100.0 / binHz) // down to ~100 Hz ripple spacing
	if maxLag >= sliceLen/2 {
		maxLag = sliceLen / 2 - 1
	}

	var var0 float64
	for i := 0; i < sliceLen; i++ {
		var0 += slice[i] * slice[i]
	}

	if var0 > 1e-9 {
		for lag := minLag; lag <= maxLag; lag++ {
			var covar float64
			for i := 0; i < sliceLen-lag; i++ {
				covar += slice[i] * slice[i+lag]
			}
			r := covar / var0
			if r > maxRipple {
				maxRipple = r
			}
		}
	}

	// Step 4: Confidence Score Computation
	cutoffScore := 0.0
	if attenuationDb > 35.0 {
		cutoffScore = 1.0
	} else if attenuationDb > 20.0 {
		cutoffScore = (attenuationDb - 20.0) / 15.0
	}

	combScore := 0.0
	if maxRipple > 0.40 {
		combScore = 1.0
	} else if maxRipple > 0.20 {
		combScore = (maxRipple - 0.20) / 0.20
	}

	confidence := 0.60*cutoffScore + 0.40*combScore
	if confidence > 1.0 {
		confidence = 1.0
	}

	// If no ultrasonic cutoff is detected (natural high-frequency energy above 16.5kHz exists),
	// the track cannot be Suno v3.
	if !cutoffDetected || attenuationDb < 15.0 {
		confidence = 0.0
	}

	isSuno := confidence >= 0.70

	summary := "No Suno audio generation artifacts detected; audio exhibits natural full-band spectrum."
	if isSuno {
		summary = "High-confidence Suno v3 artifact detected: steep 16.5kHz ultrasonic shelf drop and periodic comb ripple."
	}

	return &SunoAnalysisResult{
		IsSunoGenerated:       isSuno,
		ConfidenceScore:       math.Round(confidence*100) / 100,
		CutoffDetected:        cutoffDetected,
		CutoffFrequencyHz:     16500.0,
		HighFreqAttenuationDb: math.Round(attenuationDb*10) / 10,
		CombFilterPeriodicR:   math.Round(maxRipple*100) / 100,
		DiagnosticSummary:     summary,
	}, nil
}

// ToJSON serializes Suno forensic report.
func (r *SunoAnalysisResult) ToJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
