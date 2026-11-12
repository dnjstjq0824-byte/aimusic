// Package audio provides Spectral Flatness Measure (SFM / Wiener Entropy) computation,
// tonality vs noise discrimination, and instrument texture richness scoring.
package audio

import (
	"encoding/json"
	"math"
)

// SpectralFlatnessConfig configures spectral flatness analysis.
type SpectralFlatnessConfig struct {
	FFTSize int `json:"fft_size"` // FFT window size (default: 1024)
	HopSize int `json:"hop_size"` // Hop size for time-averaging (default: 512)
}

// DefaultSpectralFlatnessConfig returns standard 1024-point spectral analysis settings.
func DefaultSpectralFlatnessConfig() SpectralFlatnessConfig {
	return SpectralFlatnessConfig{
		FFTSize: 1024,
		HopSize: 512,
	}
}

// SpectralFlatnessResult contains overall and band-split spectral flatness metrics.
type SpectralFlatnessResult struct {
	GlobalSFM          float64 `json:"global_sfm"`          // 0.0 (pure tone) to 1.0 (white noise)
	GlobalSFMDb        float64 `json:"global_sfm_db"`       // 10 * log10(SFM)
	TonalityScore      float64 `json:"tonality_score"`      // 0..100 (100 = pure harmonic tone)
	NoisinessScore     float64 `json:"noisiness_score"`     // 0..100 (100 = pure noise)
	InstrumentRichness float64 `json:"instrument_richness"` // 0..100 (rich harmonic overtone distribution)
	BassSFM            float64 `json:"bass_sfm"`            // 20Hz - 250Hz
	MidSFM             float64 `json:"mid_sfm"`             // 250Hz - 4kHz
	HighSFM            float64 `json:"high_sfm"`            // 4kHz - 20kHz
	FrameCount         int     `json:"frame_count"`
}

// SpectralFlatnessAnalyzer measures harmonic entropy and spectral flatness.
type SpectralFlatnessAnalyzer struct {
	cfg SpectralFlatnessConfig
}

// NewSpectralFlatnessAnalyzer instantiates an SFM analyzer.
func NewSpectralFlatnessAnalyzer(cfg SpectralFlatnessConfig) *SpectralFlatnessAnalyzer {
	if cfg.FFTSize <= 0 || (cfg.FFTSize&(cfg.FFTSize-1)) != 0 {
		cfg.FFTSize = 1024
	}
	if cfg.HopSize <= 0 {
		cfg.HopSize = cfg.FFTSize / 2
	}
	return &SpectralFlatnessAnalyzer{cfg: cfg}
}

// Analyze computes spectral flatness and texture complexity across the audio buffer.
func (sfa *SpectralFlatnessAnalyzer) Analyze(input *PCMBuffer) (*SpectralFlatnessResult, error) {
	if input == nil {
		return nil, ErrNilPCMBuffer
	}
	if input.NumSamples == 0 || input.Channels == 0 || len(input.Samples) == 0 {
		return nil, ErrEmptyPCMBuffer
	}

	numSamples := input.NumSamples
	sampleRate := input.SampleRate
	fftSize := sfa.cfg.FFTSize
	hopSize := sfa.cfg.HopSize

	if numSamples < fftSize {
		// Zero-pad or adjust if short
		fftSize = 512
		hopSize = 256
		if numSamples < fftSize {
			fftSize = 256
			hopSize = 128
		}
	}

	// Mix to mono
	mono := make([]float64, numSamples)
	channels := input.Channels
	scale := 1.0 / float64(channels)
	for i := 0; i < numSamples; i++ {
		var sum float64
		for ch := 0; ch < channels; ch++ {
			sum += input.Samples[ch][i]
		}
		mono[i] = sum * scale
	}

	// Hann window
	window := make([]float64, fftSize)
	for i := 0; i < fftSize; i++ {
		window[i] = 0.5 * (1.0 - math.Cos(2.0*math.Pi*float64(i)/float64(fftSize-1)))
	}

	numBins := fftSize / 2
	binHz := float64(sampleRate) / float64(fftSize)

	bassBinEnd := int(250.0 / binHz)
	midBinEnd := int(4000.0 / binHz)
	highBinEnd := int(20000.0 / binHz)
	if highBinEnd > numBins {
		highBinEnd = numBins
	}

	var sumGlobalSFM, sumBassSFM, sumMidSFM, sumHighSFM float64
	frameCount := 0

	frame := make([]float64, fftSize)

	for start := 0; start+fftSize <= numSamples; start += hopSize {
		for i := 0; i < fftSize; i++ {
			frame[i] = mono[start+i] * window[i]
		}

		// Compute power spectrum using discrete Fourier transform for bins
		pwr := computePowerSpectrum(frame, fftSize)

		sfmGlobal := computeSFM(pwr, 1, numBins)
		sfmBass := computeSFM(pwr, 1, bassBinEnd)
		sfmMid := computeSFM(pwr, bassBinEnd, midBinEnd)
		sfmHigh := computeSFM(pwr, midBinEnd, highBinEnd)

		sumGlobalSFM += sfmGlobal
		sumBassSFM += sfmBass
		sumMidSFM += sfmMid
		sumHighSFM += sfmHigh
		frameCount++
	}

	if frameCount == 0 {
		frameCount = 1
	}

	avgGlobal := sumGlobalSFM / float64(frameCount)
	avgBass := sumBassSFM / float64(frameCount)
	avgMid := sumMidSFM / float64(frameCount)
	avgHigh := sumHighSFM / float64(frameCount)

	globalDb := -100.0
	if avgGlobal > 1e-10 {
		globalDb = 10.0 * math.Log10(avgGlobal)
	}

	tonality := (1.0 - avgGlobal) * 100.0
	if tonality < 0 {
		tonality = 0
	} else if tonality > 100 {
		tonality = 100
	}

	noisiness := avgGlobal * 100.0
	if noisiness > 100 {
		noisiness = 100
	}

	// Instrument Richness: peak when sound is musical but complex (SFM between 0.05 and 0.40)
	// A bell curve centered at SFM = 0.20
	richness := 100.0 * math.Exp(-math.Pow((avgGlobal-0.20)/0.18, 2))
	if richness > 100 {
		richness = 100
	}

	return &SpectralFlatnessResult{
		GlobalSFM:          math.Round(avgGlobal*10000) / 10000,
		GlobalSFMDb:        math.Round(globalDb*10) / 10,
		TonalityScore:      math.Round(tonality*10) / 10,
		NoisinessScore:     math.Round(noisiness*10) / 10,
		InstrumentRichness: math.Round(richness*10) / 10,
		BassSFM:            math.Round(avgBass*10000) / 10000,
		MidSFM:             math.Round(avgMid*10000) / 10000,
		HighSFM:            math.Round(avgHigh*10000) / 10000,
		FrameCount:         frameCount,
	}, nil
}

// computePowerSpectrum returns |X(k)|^2 for positive frequencies.
func computePowerSpectrum(frame []float64, n int) []float64 {
	numBins := n / 2
	pwr := make([]float64, numBins)

	for k := 0; k < numBins; k++ {
		var realPart, imagPart float64
		theta := -2.0 * math.Pi * float64(k) / float64(n)
		for t := 0; t < n; t++ {
			angle := theta * float64(t)
			realPart += frame[t] * math.Cos(angle)
			imagPart += frame[t] * math.Sin(angle)
		}
		pwr[k] = realPart*realPart + imagPart*imagPart + 1e-12
	}
	return pwr
}

// computeSFM calculates geometric mean / arithmetic mean over a slice of spectrum bins.
func computeSFM(pwr []float64, startBin, endBin int) float64 {
	if startBin < 0 {
		startBin = 0
	}
	if endBin > len(pwr) {
		endBin = len(pwr)
	}
	count := endBin - startBin
	if count <= 0 {
		return 0.0
	}

	var sumLog, sumPwr float64
	for k := startBin; k < endBin; k++ {
		v := pwr[k]
		sumLog += math.Log(v)
		sumPwr += v
	}

	geomMean := math.Exp(sumLog / float64(count))
	arithMean := sumPwr / float64(count)

	if arithMean <= 1e-12 {
		return 0.0
	}

	sfm := geomMean / arithMean
	if sfm > 1.0 {
		sfm = 1.0
	}
	return sfm
}

// ToJSON serializes spectral flatness result.
func (r *SpectralFlatnessResult) ToJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
