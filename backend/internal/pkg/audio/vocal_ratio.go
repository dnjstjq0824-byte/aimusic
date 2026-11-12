// Package audio provides vocal-to-instrumental ratio (VIR) analysis,
// Mid-Side acoustic decomposition, vocal formant energy profiling, and
// lyric intelligibility scoring for commercial AI audio validation.
package audio

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

// VocalDominance categorizes the vocal presence and mix balance of an audio track.
type VocalDominance string

const (
	DominancePureInstrumental VocalDominance = "PURE_INSTRUMENTAL"  // VIR < -15 dB or no vocal activity detected
	DominanceInstrumental     VocalDominance = "INSTRUMENTAL_LEAN"  // -15 dB <= VIR < -5 dB
	DominanceBalancedMix      VocalDominance = "BALANCED_MIX"       // -5 dB <= VIR <= +3 dB (Standard pop/commercial mix)
	DominanceVocalForward     VocalDominance = "VOCAL_FORWARD"      // +3 dB < VIR <= +10 dB (Acoustic/ballad/spoken)
	DominanceAcappella        VocalDominance = "ACAPPELLA_DOMINANT" // VIR > +10 dB
)

// VocalBandAnalysis holds energy measurements within specific frequency ranges.
type VocalBandAnalysis struct {
	VocalCoreRMSDbFS float64 `json:"vocal_core_rms_dbfs"` // 300 Hz - 3400 Hz Mid channel
	SpeechPresenceDb float64 `json:"speech_presence_db"`  // 1 kHz - 3 kHz Formant band
	LowEndSubDb      float64 `json:"low_end_sub_db"`      // < 300 Hz (Bass, Kick)
	HighEndAirDb     float64 `json:"high_end_air_db"`     // > 3400 Hz (Cymbals, Air)
	SideStereoDb     float64 `json:"side_stereo_db"`      // Stereo Side channel energy
}

// VocalRatioResult contains full VIR metrics and lyric intelligibility telemetry.
type VocalRatioResult struct {
	SampleRate           int               `json:"sample_rate"`
	Channels             int               `json:"channels"`
	DurationSec          float64           `json:"duration_sec"`
	VIRDb                float64           `json:"vir_db"`                // Vocal-to-Instrumental Ratio in dB
	VocalEnergyLinear    float64           `json:"vocal_energy_linear"`   // Normalized vocal band energy
	InstEnergyLinear     float64           `json:"inst_energy_linear"`    // Normalized non-vocal + stereo side energy
	IntelligibilityScore float64           `json:"intelligibility_score"` // 0.0 (unintelligible / no vocal) to 1.0 (crystal clear)
	Dominance            VocalDominance    `json:"dominance"`
	HasLeadVocal         bool              `json:"has_lead_vocal"`
	CenterPannedRatio    float64           `json:"center_panned_ratio"` // Ratio of mid-to-total energy in vocal band
	Bands                VocalBandAnalysis `json:"bands"`
}

// VocalRatioAnalyzerConfig configures the vocal ratio analyzer.
type VocalRatioAnalyzerConfig struct {
	SampleRate       int
	FFTSize          int
	HopSize          int
	VocalMinFreqHz   float64 // default: 300.0
	VocalMaxFreqHz   float64 // default: 3400.0
	PresenceMinHz    float64 // default: 1000.0
	PresenceMaxHz    float64 // default: 3000.0
	VocalThresholdDb float64 // default: -40.0 dBFS minimum to qualify as vocal presence
}

// DefaultVocalRatioConfig returns sensible defaults for studio vocal analysis.
func DefaultVocalRatioConfig(sampleRate int) VocalRatioAnalyzerConfig {
	if sampleRate <= 0 {
		sampleRate = 44100
	}
	return VocalRatioAnalyzerConfig{
		SampleRate:       sampleRate,
		FFTSize:          2048,
		HopSize:          512,
		VocalMinFreqHz:   300.0,
		VocalMaxFreqHz:   3400.0,
		PresenceMinHz:    1000.0,
		PresenceMaxHz:    3000.0,
		VocalThresholdDb: -42.0,
	}
}

// VocalRatioAnalyzer analyzes vocal-to-instrumental ratio using Mid-Side STFT decomposition.
type VocalRatioAnalyzer struct {
	cfg VocalRatioAnalyzerConfig
}

// NewVocalRatioAnalyzer instantiates a new analyzer.
func NewVocalRatioAnalyzer(cfg VocalRatioAnalyzerConfig) *VocalRatioAnalyzer {
	if cfg.SampleRate <= 0 {
		cfg.SampleRate = 44100
	}
	if cfg.FFTSize <= 0 {
		cfg.FFTSize = 2048
	}
	if cfg.HopSize <= 0 {
		cfg.HopSize = 512
	}
	if cfg.VocalMinFreqHz <= 0 {
		cfg.VocalMinFreqHz = 300.0
	}
	if cfg.VocalMaxFreqHz <= 0 {
		cfg.VocalMaxFreqHz = 3400.0
	}
	if cfg.PresenceMinHz <= 0 {
		cfg.PresenceMinHz = 1000.0
	}
	if cfg.PresenceMaxHz <= 0 {
		cfg.PresenceMaxHz = 3000.0
	}
	if cfg.VocalThresholdDb == 0 {
		cfg.VocalThresholdDb = -42.0
	}
	return &VocalRatioAnalyzer{cfg: cfg}
}

// Analyze processes multi-channel planar PCM samples and computes vocal metrics.
// samples can be mono (1 channel) or stereo (2 channels).
func (v *VocalRatioAnalyzer) Analyze(samples [][]float64) (*VocalRatioResult, error) {
	if len(samples) == 0 || len(samples[0]) == 0 {
		return nil, errors.New("empty audio buffer provided for vocal ratio analysis")
	}

	numChannels := len(samples)
	numSamples := len(samples[0])
	duration := float64(numSamples) / float64(v.cfg.SampleRate)

	// Step 1: Compute Mid and Side signals
	mid := make([]float64, numSamples)
	side := make([]float64, numSamples)

	sqrt2 := math.Sqrt(2.0)
	if numChannels >= 2 {
		for i := 0; i < numSamples; i++ {
			l := samples[0][i]
			r := samples[1][i]
			mid[i] = (l + r) / sqrt2
			side[i] = (l - r) / sqrt2
		}
	} else {
		// Mono: Mid is identical, Side is silent
		copy(mid, samples[0])
	}

	// Step 2: Perform STFT on Mid and Side channels
	stftEngine, err := NewSTFTEngine(STFTConfig{
		FFTSize: v.cfg.FFTSize,
		HopSize: v.cfg.HopSize,
		Window:  WindowHann,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create STFT engine: %w", err)
	}

	resMid, err := stftEngine.Forward(&PCMBuffer{
		Channels:   1,
		SampleRate: v.cfg.SampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{mid},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to compute Mid spectrogram: %w", err)
	}
	resSide, err := stftEngine.Forward(&PCMBuffer{
		Channels:   1,
		SampleRate: v.cfg.SampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{side},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to compute Side spectrogram: %w", err)
	}

	freqBinWidth := float64(v.cfg.SampleRate) / float64(v.cfg.FFTSize)
	numFrames := resMid.NumFrames
	numBins := resMid.NumBins

	var totalVocalCoreEnergy float64
	var totalPresenceEnergy float64
	var totalLowEndEnergy float64
	var totalHighEndEnergy float64
	var totalSideEnergy float64
	var totalVocalBandSideEnergy float64

	vocalMinBin := int(v.cfg.VocalMinFreqHz / freqBinWidth)
	vocalMaxBin := int(v.cfg.VocalMaxFreqHz / freqBinWidth)
	presenceMinBin := int(v.cfg.PresenceMinHz / freqBinWidth)
	presenceMaxBin := int(v.cfg.PresenceMaxHz / freqBinWidth)

	if vocalMinBin < 0 {
		vocalMinBin = 0
	}
	if vocalMaxBin >= numBins {
		vocalMaxBin = numBins - 1
	}
	if presenceMinBin < 0 {
		presenceMinBin = 0
	}
	if presenceMaxBin >= numBins {
		presenceMaxBin = numBins - 1
	}

	midFrames := resMid.Frames[0]
	sideFrames := resSide.Frames[0]

	for t := 0; t < numFrames; t++ {
		midMag := midFrames[t].Magnitude
		sideMag := sideFrames[t].Magnitude

		for k := 0; k < numBins; k++ {
			f := float64(k) * freqBinWidth
			mPwr := midMag[k] * midMag[k]
			sPwr := sideMag[k] * sideMag[k]
			totPwr := mPwr + sPwr

			totalSideEnergy += sPwr

			if f < v.cfg.VocalMinFreqHz {
				totalLowEndEnergy += totPwr
			} else if f > v.cfg.VocalMaxFreqHz {
				totalHighEndEnergy += totPwr
			} else {
				// Inside vocal formant band (300Hz - 3400Hz)
				totalVocalCoreEnergy += mPwr
				totalVocalBandSideEnergy += sPwr

				if f >= v.cfg.PresenceMinHz && f <= v.cfg.PresenceMaxHz {
					totalPresenceEnergy += mPwr
				}
			}
		}
	}

	denom := float64(numFrames * numBins)
	if denom <= 0 {
		denom = 1
	}

	meanVocalCore := totalVocalCoreEnergy / denom
	meanPresence := totalPresenceEnergy / denom
	meanLow := totalLowEndEnergy / denom
	meanHigh := totalHighEndEnergy / denom
	meanSide := totalSideEnergy / denom

	toDbFS := func(energy float64) float64 {
		if energy <= 1e-12 {
			return -120.0
		}
		// 10 * log10(energy)
		return math.Max(-120.0, 10.0*math.Log10(energy))
	}

	vocalCoreDb := toDbFS(meanVocalCore)
	presenceDb := toDbFS(meanPresence)
	lowDb := toDbFS(meanLow)
	highDb := toDbFS(meanHigh)
	sideDb := toDbFS(meanSide)

	// Instrumental energy: low end + high end + stereo sides (non-centered instruments)
	instrumentalEnergy := meanLow + meanHigh + meanSide
	vocalEnergy := meanVocalCore

	// Center panned ratio in vocal band: Mid vs (Mid + Side) in 300Hz-3400Hz
	centerPannedRatio := 1.0
	vocalBandTotal := totalVocalCoreEnergy + totalVocalBandSideEnergy
	if vocalBandTotal > 1e-12 {
		centerPannedRatio = totalVocalCoreEnergy / vocalBandTotal
	}

	var virDb float64
	if instrumentalEnergy <= 1e-12 && vocalEnergy <= 1e-12 {
		virDb = -60.0
	} else if instrumentalEnergy <= 1e-12 {
		virDb = 40.0
	} else if vocalEnergy <= 1e-12 {
		virDb = -60.0
	} else {
		virDb = 10.0 * math.Log10(vocalEnergy/instrumentalEnergy)
	}

	// Lead vocal presence check:
	// Needs sufficient vocal core energy AND high center correlation (vocals are centered)
	hasLeadVocal := vocalCoreDb > v.cfg.VocalThresholdDb && centerPannedRatio > 0.55 && virDb > -14.0

	// Dominance classification
	var dominance VocalDominance
	switch {
	case !hasLeadVocal || virDb < -15.0:
		dominance = DominancePureInstrumental
	case virDb < -5.0:
		dominance = DominanceInstrumental
	case virDb <= 3.0:
		dominance = DominanceBalancedMix
	case virDb <= 10.0:
		dominance = DominanceVocalForward
	default:
		dominance = DominanceAcappella
	}

	// Intelligibility scoring (Speech Transmission Index approximation):
	// S-curve centered around -3 dB VIR with penalty for low speech presence or high side noise
	var intelligibility float64
	if !hasLeadVocal {
		intelligibility = 0.0
	} else {
		// Logistic sigmoid of VIR centered around -2dB
		sig := 1.0 / (1.0 + math.Exp(-0.35*(virDb-(-2.0))))
		// Center panning boost
		intelligibility = sig * math.Min(1.0, centerPannedRatio*1.2)
		if intelligibility > 1.0 {
			intelligibility = 1.0
		}
		if intelligibility < 0.0 {
			intelligibility = 0.0
		}
	}

	return &VocalRatioResult{
		SampleRate:           v.cfg.SampleRate,
		Channels:             numChannels,
		DurationSec:          duration,
		VIRDb:                math.Round(virDb*100) / 100,
		VocalEnergyLinear:    meanVocalCore,
		InstEnergyLinear:     instrumentalEnergy,
		IntelligibilityScore: math.Round(intelligibility*1000) / 1000,
		Dominance:            dominance,
		HasLeadVocal:         hasLeadVocal,
		CenterPannedRatio:    math.Round(centerPannedRatio*1000) / 1000,
		Bands: VocalBandAnalysis{
			VocalCoreRMSDbFS: math.Round(vocalCoreDb*100) / 100,
			SpeechPresenceDb: math.Round(presenceDb*100) / 100,
			LowEndSubDb:      math.Round(lowDb*100) / 100,
			HighEndAirDb:     math.Round(highDb*100) / 100,
			SideStereoDb:     math.Round(sideDb*100) / 100,
		},
	}, nil
}

// ToJSON serializes the result to pretty formatted JSON.
func (r *VocalRatioResult) ToJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
