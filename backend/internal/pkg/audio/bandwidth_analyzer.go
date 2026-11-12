// Package audio provides pure Go psychoacoustic spectral bandwidth analysis,
// cutoff frequency detection, and detection of upscaled low-bitrate lossy audio
// (such as MP3 128k or AAC upsampled into 24-bit 48kHz/96kHz Hi-Res containers).
package audio

import (
	"errors"
	"fmt"
	"math"
)

// QualityVerdict classifies the spectral authenticity and effective bandwidth of the audio stream.
type QualityVerdict string

const (
	VerdictTrueLossless    QualityVerdict = "TRUE_LOSSLESS"        // Full acoustic bandwidth (> 20 kHz)
	VerdictUpscaledLossy   QualityVerdict = "UPSCALED_LOSSY_128K"  // Low-pass brickwall under 16.5 kHz packaged as Hi-Res
	VerdictMediumLossy     QualityVerdict = "MEDIUM_LOSSY_192K"    // Bandwidth limited to ~18-19 kHz
	VerdictLowBitrate      QualityVerdict = "LOW_BITRATE_96K_64K"  // Bandwidth limited to <= 14.5 kHz
	VerdictNaturalAcoustic QualityVerdict = "NATURAL_ACOUSTIC_LOW" // Minimalist sound (e.g. bass/speech) without brickwall shelf
)

// BandwidthAnalysisConfig defines parameters for high-resolution spectral analysis.
type BandwidthAnalysisConfig struct {
	FFTSize                int     // FFT window size (default: 2048)
	HopSize                int     // Hop size (default: 512)
	RollOffPercentile      float64 // Percentile for spectral roll-off (default: 0.99)
	CutoffThresholdDb      float64 // Minimum dB drop between pre-cutoff and post-cutoff (default: 25.0 dB)
	LossyCutoffFreqHz      float64 // MP3 128k brickwall boundary (default: 16500.0 Hz)
	MinAnalyzedDurationSec float64 // Minimum required duration (default: 0.2s)
}

// DefaultBandwidthConfig returns standard configuration for lossy compression detection.
func DefaultBandwidthConfig() BandwidthAnalysisConfig {
	return BandwidthAnalysisConfig{
		FFTSize:                2048,
		HopSize:                512,
		RollOffPercentile:      0.99,
		CutoffThresholdDb:      25.0,
		LossyCutoffFreqHz:      16500.0,
		MinAnalyzedDurationSec: 0.2,
	}
}

// Diagnostic error definitions.
var (
	ErrBufferTooShort   = errors.New("audio duration is shorter than minimum analysis window")
	ErrSampleRateTooLow = errors.New("sample rate must be at least 32000 Hz to evaluate 16 kHz cutoff")
)

// BandwidthAnalysisResult contains comprehensive spectral bandwidth diagnostics.
type BandwidthAnalysisResult struct {
	SampleRate               int                // Audio sample rate in Hz
	Channels                 int                // Number of channels
	DurationSec              float64            // Duration analyzed in seconds
	DetectedCutoffHz         float64            // Identified brickwall cutoff frequency in Hz
	RollOff99Hz              float64            // Frequency containing 99% of spectral energy
	SpectralDropoffDb        float64            // Attenuation step across the detected cutoff
	HighFrequencyEnergyRatio float64            // Ratio of energy above 16.5kHz to energy in 1kHz-10kHz
	AverageBandEnergiesDb    map[string]float64 // Average energy levels per diagnostic sub-band in dBFS
	Verdict                  QualityVerdict     // Classification verdict
	EstimatedOriginalBitrate string             // Human-readable estimation of original bitrate
	Confidence               float64            // Confidence score (0.0 to 1.0)
	IsUpscaledLossy          bool               // True if audio exhibits low-pass brickwall under 16.5 kHz
}

// BandwidthAnalyzer inspects frequency spectra to detect upscaled lossy compression.
type BandwidthAnalyzer struct {
	config BandwidthAnalysisConfig
}

// NewBandwidthAnalyzer creates a new spectral bandwidth analyzer.
func NewBandwidthAnalyzer(config BandwidthAnalysisConfig) (*BandwidthAnalyzer, error) {
	if config.FFTSize < 512 || (config.FFTSize&(config.FFTSize-1)) != 0 {
		return nil, fmt.Errorf("FFTSize must be a power of two >= 512, got %d", config.FFTSize)
	}
	if config.HopSize <= 0 {
		config.HopSize = config.FFTSize / 4
	}
	if config.RollOffPercentile <= 0.0 || config.RollOffPercentile >= 1.0 {
		config.RollOffPercentile = 0.99
	}
	if config.CutoffThresholdDb <= 0.0 {
		config.CutoffThresholdDb = 25.0
	}
	if config.LossyCutoffFreqHz <= 0.0 {
		config.LossyCutoffFreqHz = 16500.0
	}
	return &BandwidthAnalyzer{config: config}, nil
}

// Analyze performs deep spectral inspection on the PCM buffer to detect brickwall filtering.
func (ba *BandwidthAnalyzer) Analyze(pcm *PCMBuffer) (*BandwidthAnalysisResult, error) {
	if pcm == nil {
		return nil, ErrNilPCMBuffer
	}
	if pcm.NumSamples == 0 || len(pcm.Samples) == 0 {
		return nil, ErrEmptyAudioBuffer
	}
	if pcm.SampleRate < 32000 {
		return nil, ErrSampleRateTooLow
	}

	fftSize := ba.config.FFTSize
	if pcm.NumSamples < fftSize {
		return nil, ErrBufferTooShort
	}

	stftEngine, err := NewSTFTEngine(STFTConfig{
		FFTSize:    fftSize,
		HopSize:    ba.config.HopSize,
		Window:     WindowHann,
		NumMelBins: 80,
	})
	if err != nil {
		return nil, fmt.Errorf("initializing stft engine: %w", err)
	}

	stftRes, err := stftEngine.Forward(pcm)
	if err != nil {
		return nil, fmt.Errorf("computing stft: %w", err)
	}

	numBins := stftRes.NumBins
	numFrames := stftRes.NumFrames
	channels := pcm.Channels
	freqResolution := float64(pcm.SampleRate) / float64(fftSize)
	nyquist := float64(pcm.SampleRate) / 2.0

	// 1. Compute average power spectral density across all channels and active frames
	avgPSD := make([]float64, numBins)
	activeFrames := 0

	for ch := 0; ch < channels; ch++ {
		for f := 0; f < numFrames; f++ {
			frame := stftRes.Frames[ch][f]
			frameEnergy := 0.0
			for k := 0; k < numBins; k++ {
				frameEnergy += frame.Power[k]
			}
			if frameEnergy > 1e-8 {
				activeFrames++
				for k := 0; k < numBins; k++ {
					avgPSD[k] += frame.Power[k]
				}
			}
		}
	}

	if activeFrames > 0 {
		scale := 1.0 / float64(activeFrames)
		for k := 0; k < numBins; k++ {
			avgPSD[k] *= scale
		}
	}

	// 2. Measure energy across diagnostic sub-bands
	bandLowMidEnergy := 0.0   // 1 kHz to 10 kHz
	bandHighMidEnergy := 0.0  // 10 kHz to 15 kHz
	bandCriticalEnergy := 0.0 // 15 kHz to 16.5 kHz
	bandHighEnergy := 0.0     // 16.5 kHz to 20 kHz
	bandUltraEnergy := 0.0    // 20 kHz to Nyquist
	totalSpectralEnergy := 0.0

	lowMidCount := 0
	highMidCount := 0
	criticalCount := 0
	highCount := 0
	ultraCount := 0

	for k := 0; k < numBins; k++ {
		freq := float64(k) * freqResolution
		p := avgPSD[k]
		totalSpectralEnergy += p

		if freq >= 1000.0 && freq < 10000.0 {
			bandLowMidEnergy += p
			lowMidCount++
		} else if freq >= 10000.0 && freq < 15000.0 {
			bandHighMidEnergy += p
			highMidCount++
		} else if freq >= 15000.0 && freq < 16500.0 {
			bandCriticalEnergy += p
			criticalCount++
		} else if freq >= 16500.0 && freq < 20000.0 {
			bandHighEnergy += p
			highCount++
		} else if freq >= 20000.0 && freq <= nyquist {
			bandUltraEnergy += p
			ultraCount++
		}
	}

	// 3. Calculate 99% Spectral Roll-off Frequency
	accumEnergy := 0.0
	targetEnergy := ba.config.RollOffPercentile * totalSpectralEnergy
	rollOff99Hz := nyquist

	for k := 0; k < numBins; k++ {
		accumEnergy += avgPSD[k]
		if accumEnergy >= targetEnergy {
			rollOff99Hz = float64(k) * freqResolution
			break
		}
	}

	// 4. Brickwall Cutoff Sweep Analysis via Local Spectral Slope
	// Sweeps candidate frequencies from 11 kHz to min(21.5 kHz, Nyquist - 800 Hz)
	searchStartHz := 11000.0
	searchEndHz := math.Min(21500.0, nyquist-800.0)
	stepHz := 100.0
	halfWindowHz := 600.0

	maxDropDb := 0.0
	detectedCutoffHz := nyquist

	hasAcousticHighMids := bandHighMidEnergy > 1e-8 && (bandHighMidEnergy/(bandLowMidEnergy+1e-12)) > 0.0005

	if hasAcousticHighMids {
		for fc := searchStartHz; fc <= searchEndHz; fc += stepHz {
			leftStart := fc - halfWindowHz
			leftEnd := fc
			rightStart := fc
			rightEnd := fc + halfWindowHz

			leftEnergy := 0.0
			leftBins := 0

			rightEnergy := 0.0
			rightBins := 0

			postStopbandEnergy := 0.0
			postStopbandBins := 0

			for k := 0; k < numBins; k++ {
				freq := float64(k) * freqResolution
				if freq >= leftStart && freq < leftEnd {
					leftEnergy += avgPSD[k]
					leftBins++
				} else if freq >= rightStart && freq < rightEnd {
					rightEnergy += avgPSD[k]
					rightBins++
				} else if freq >= fc+halfWindowHz && freq <= nyquist {
					postStopbandEnergy += avgPSD[k]
					postStopbandBins++
				}
			}

			if leftBins > 0 && rightBins > 0 && leftEnergy > 1e-9 {
				leftDensity := leftEnergy / float64(leftBins)
				rightDensity := rightEnergy / float64(rightBins)

				// Check that right window and post-stopband are significantly attenuated
				slopeDb := 10.0 * math.Log10(leftDensity/(rightDensity+1e-14))

				// Verify that stopband does not revive into loud musical content
				postDensity := 0.0
				if postStopbandBins > 0 {
					postDensity = postStopbandEnergy / float64(postStopbandBins)
				}
				stopbandDropDb := 10.0 * math.Log10(leftDensity/(postDensity+1e-14))

				if slopeDb > maxDropDb && stopbandDropDb >= ba.config.CutoffThresholdDb {
					maxDropDb = slopeDb
					detectedCutoffHz = fc
				}
			}
		}
	}

	// 5. Ratio of high-frequency energy (> 16.5 kHz) to low-mid energy (1k - 10k)
	hfRatio := (bandHighEnergy + bandUltraEnergy) / (bandLowMidEnergy + 1e-12)

	// 6. Sub-band dBFS levels
	toDbFS := func(energy float64, count int) float64 {
		if count <= 0 || energy <= 1e-13 {
			return -120.0
		}
		return 10.0 * math.Log10(energy/float64(count))
	}

	bandEnergiesDb := map[string]float64{
		"1k_10k_LowMid":   toDbFS(bandLowMidEnergy, lowMidCount),
		"10k_15k_HighMid": toDbFS(bandHighMidEnergy, highMidCount),
		"15k_16.5k_Cut":   toDbFS(bandCriticalEnergy, criticalCount),
		"16.5k_20k_High":  toDbFS(bandHighEnergy, highCount),
		"20k_Nyquist":     toDbFS(bandUltraEnergy, ultraCount),
	}

	// 7. Determine Quality Verdict
	durationSec := float64(pcm.NumSamples) / float64(pcm.SampleRate)
	verdict := VerdictTrueLossless
	isUpscaled := false
	estimatedBitrate := "Lossless Studio Master / Hi-Res (> 320 kbps)"
	confidence := 0.98

	if !hasAcousticHighMids && maxDropDb < ba.config.CutoffThresholdDb {
		verdict = VerdictNaturalAcoustic
		isUpscaled = false
		estimatedBitrate = "Acoustic Minimalist Signal (No High-Frequency Cutoff Shelf)"
		confidence = 0.95
	} else if maxDropDb >= ba.config.CutoffThresholdDb && detectedCutoffHz <= 14600.0 {
		verdict = VerdictLowBitrate
		isUpscaled = true
		estimatedBitrate = "MP3 64-96 kbps (Severe Low-Pass Brickwall)"
		confidence = math.Min(1.0, 0.85+0.15*(maxDropDb-25.0)/25.0)
	} else if maxDropDb >= ba.config.CutoffThresholdDb && detectedCutoffHz <= 16800.0 && hfRatio < 0.005 {
		verdict = VerdictUpscaledLossy
		isUpscaled = true
		estimatedBitrate = "MP3 128 kbps (Upscaled Low-Bitrate Lossy)"
		confidence = math.Min(1.0, 0.88+0.12*(maxDropDb-25.0)/25.0)
	} else if maxDropDb >= ba.config.CutoffThresholdDb && detectedCutoffHz <= 19500.0 && rollOff99Hz < 19500.0 {
		verdict = VerdictMediumLossy
		isUpscaled = false
		estimatedBitrate = "MP3 192 kbps (Medium Bitrate Lossy)"
		confidence = 0.92
	} else {
		verdict = VerdictTrueLossless
		isUpscaled = false
		estimatedBitrate = "Lossless Studio Master / Hi-Res (> 320 kbps)"
		confidence = 0.98
	}

	return &BandwidthAnalysisResult{
		SampleRate:               pcm.SampleRate,
		Channels:                 pcm.Channels,
		DurationSec:              durationSec,
		DetectedCutoffHz:         detectedCutoffHz,
		RollOff99Hz:              rollOff99Hz,
		SpectralDropoffDb:        maxDropDb,
		HighFrequencyEnergyRatio: hfRatio,
		AverageBandEnergiesDb:    bandEnergiesDb,
		Verdict:                  verdict,
		EstimatedOriginalBitrate: estimatedBitrate,
		Confidence:               confidence,
		IsUpscaledLossy:          isUpscaled,
	}, nil
}

// GenerateSyntheticAcousticAudio creates a rich multi-harmonic audio stream
// with spectral content extending seamlessly up to 22 kHz (authentic lossless profile).
func GenerateSyntheticAcousticAudio(durationSec float64, sampleRate, channels int) *PCMBuffer {
	if sampleRate <= 0 {
		sampleRate = 48000
	}
	if channels <= 0 {
		channels = 2
	}
	if durationSec <= 0 {
		durationSec = 1.0
	}

	numSamples := int(float64(sampleRate) * durationSec)
	samples := make([][]float64, channels)
	for ch := 0; ch < channels; ch++ {
		samples[ch] = make([]float64, numSamples)
	}

	// Fundamental and harmonic frequencies across full acoustic spectrum up to 21.5 kHz
	freqs := []float64{110.0, 220.0, 440.0, 880.0, 1760.0, 3520.0, 7040.0, 11000.0, 14000.0, 17500.0, 19500.0, 21200.0}
	amps := []float64{0.25, 0.22, 0.20, 0.16, 0.14, 0.10, 0.08, 0.06, 0.05, 0.04, 0.035, 0.03}

	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		sampleVal := 0.0

		for idx, f := range freqs {
			sampleVal += amps[idx] * math.Sin(2.0*math.Pi*f*t)
		}

		// Broadband pink-noise acoustic texture (1/f) to simulate studio acoustic resonance
		noise := (float64((i*41+73)%200)/100.0 - 1.0) * 0.025
		sampleVal += noise

		normalizedVal := sampleVal * 0.7
		for ch := 0; ch < channels; ch++ {
			samples[ch][i] = normalizedVal
		}
	}

	return &PCMBuffer{
		Samples:    samples,
		Channels:   channels,
		SampleRate: sampleRate,
		NumSamples: numSamples,
	}
}

// SimulateLossyCutoff applies a steep brickwall low-pass filter to a PCMBuffer,
// simulating the exact acoustic spectrum output of lossy MP3/AAC encoders upsampled to Hi-Res.
func SimulateLossyCutoff(pcm *PCMBuffer, cutoffHz float64) (*PCMBuffer, error) {
	if pcm == nil {
		return nil, ErrNilPCMBuffer
	}

	fftSize := 2048
	hopSize := 512
	stftEngine, err := NewSTFTEngine(STFTConfig{
		FFTSize:    fftSize,
		HopSize:    hopSize,
		Window:     WindowHann,
		NumMelBins: 80,
	})
	if err != nil {
		return nil, err
	}

	res, err := stftEngine.Forward(pcm)
	if err != nil {
		return nil, err
	}

	freqRes := float64(pcm.SampleRate) / float64(fftSize)
	transitionBandHz := 300.0

	// Apply steep low-pass filter in frequency domain
	for ch := 0; ch < pcm.Channels; ch++ {
		for f := 0; f < res.NumFrames; f++ {
			frame := res.Frames[ch][f]
			for k := 0; k < res.NumBins; k++ {
				freq := float64(k) * freqRes
				attenuation := 1.0

				if freq >= cutoffHz+transitionBandHz {
					attenuation = 1e-5 // -100 dB rejection floor
				} else if freq > cutoffHz {
					// Steep cosine roll-off
					norm := (freq - cutoffHz) / transitionBandHz
					attenuation = 0.5 * (1.0 + math.Cos(math.Pi*norm))
					if attenuation < 1e-5 {
						attenuation = 1e-5
					}
				}

				frame.Real[k] *= attenuation
				frame.Imag[k] *= attenuation
				frame.Magnitude[k] *= attenuation
				frame.Power[k] = (frame.Magnitude[k] * frame.Magnitude[k]) / float64(fftSize)
			}
		}
	}

	return stftEngine.Inverse(res)
}
