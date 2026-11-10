// Package audio provides pure Go audio RMS level, True Peak, Crest Factor (CF),
// and TT Dynamic Range (DR) analysis to evaluate dynamic health and detect
// over-compressed Loudness War casualties.
package audio

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
)

// DynamicRating categorizes the overall dynamic health of an audio track.
type DynamicRating string

const (
	RatingAudiophileDynamic DynamicRating = "AUDIOPHILE_DYNAMIC" // DR >= 14, CF >= 14 dB
	RatingHealthyMaster     DynamicRating = "HEALTHY_MASTER"     // DR 10-13, CF 10-14 dB
	RatingModerateComp      DynamicRating = "MODERATE_COMP"      // DR 7-9, CF 7-10 dB
	RatingLoudnessWar       DynamicRating = "LOUDNESS_WAR_HEAVY" // DR 4-6, CF 4-7 dB
	RatingExtremeSquashed   DynamicRating = "EXTREME_SQUASHED"   // DR 0-3, CF < 4 dB
)

// ChannelDynamics holds metric measurements for an individual audio channel.
type ChannelDynamics struct {
	ChannelIndex     int     `json:"channel_index"`
	ChannelName      string  `json:"channel_name"` // "Left", "Right", "Center", etc.
	RMSDbFS          float64 `json:"rms_dbfs"`
	AES17RMSDbFS     float64 `json:"aes17_rms_dbfs"`
	PeakDbFS         float64 `json:"peak_dbfs"`
	TruePeakDbTP     float64 `json:"true_peak_dbtp"`
	CrestFactorRatio float64 `json:"crest_factor_ratio"` // Linear ratio Peak / RMS
	CrestFactorDb    float64 `json:"crest_factor_db"`    // 20*log10(ratio)
}

// CrestFactorResult encapsulates full dynamic range and Loudness War telemetry for UI metadata presentation.
type CrestFactorResult struct {
	SampleRate   int     `json:"sample_rate"`
	Channels     int     `json:"channels"`
	DurationSec  float64 `json:"duration_sec"`
	TotalSamples int     `json:"total_samples"`

	// Global Integrated Master Dynamics
	GlobalRMSDbFS      float64 `json:"global_rms_dbfs"`
	GlobalAES17RMSDbFS float64 `json:"global_aes17_rms_dbfs"`
	GlobalPeakDbFS     float64 `json:"global_peak_dbfs"`
	GlobalTruePeakDbTP float64 `json:"global_true_peak_dbtp"`
	CrestFactorRatio   float64 `json:"crest_factor_ratio"`
	CrestFactorDb      float64 `json:"crest_factor_db"`

	// Dynamic Range (TT DR Meter Standard)
	DRScore     int           `json:"dr_score"` // Integer 1-20+ (e.g. 11 for DR11)
	Rating      DynamicRating `json:"rating"`
	Description string        `json:"description"`

	// Loudness War Telemetry
	IsOverCompressed bool `json:"is_over_compressed"` // True if CF < 8.0 dB

	// Short-term sliding dynamics (400ms window, 100ms hop)
	MinShortTermCrestDb float64 `json:"min_short_term_crest_db"`
	MaxShortTermCrestDb float64 `json:"max_short_term_crest_db"`
	DynamicSpreadDb     float64 `json:"dynamic_spread_db"` // Max - Min

	// Per-channel breakdown
	ChannelBreakdown []ChannelDynamics `json:"channel_breakdown"`

	// Formatted UI Badge string
	UIBadge string `json:"ui_badge"`
}

// ToJSON serializes the CrestFactorResult into a clean JSON string for API and UI presentation.
func (r *CrestFactorResult) ToJSON() (string, error) {
	bytes, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CrestFactorConfig configures the dynamics and crest factor analyzer.
type CrestFactorConfig struct {
	EnableTruePeak         bool    // Use 4x polyphase inter-sample peak detection
	WindowSizeSec          float64 // Sliding window size for short-term analysis (default: 0.4s)
	WindowHopSec           float64 // Hop size (default: 0.1s)
	OverCompressedCutoffDb float64 // Threshold below which track is flagged as Loudness War casualty (default: 8.0 dB)
}

// DefaultCrestFactorConfig returns standard studio monitoring parameters.
func DefaultCrestFactorConfig() CrestFactorConfig {
	return CrestFactorConfig{
		EnableTruePeak:         true,
		WindowSizeSec:          0.4,
		WindowHopSec:           0.1,
		OverCompressedCutoffDb: 8.0,
	}
}

// CrestFactorAnalyzer computes RMS levels, peak amplitudes, and crest factor dynamics.
type CrestFactorAnalyzer struct {
	config CrestFactorConfig
}

// NewCrestFactorAnalyzer initializes a new crest factor analysis engine.
func NewCrestFactorAnalyzer(config CrestFactorConfig) *CrestFactorAnalyzer {
	if config.WindowSizeSec <= 0.0 {
		config.WindowSizeSec = 0.4
	}
	if config.WindowHopSec <= 0.0 {
		config.WindowHopSec = 0.1
	}
	if config.OverCompressedCutoffDb <= 0.0 {
		config.OverCompressedCutoffDb = 8.0
	}
	return &CrestFactorAnalyzer{config: config}
}

// standardChannelNames provides canonical names for standard multichannel setups.
var standardChannelNames = []string{
	"Left", "Right", "Center", "LFE", "Left Surround", "Right Surround",
	"Left Rear Surround", "Right Rear Surround",
}

func getChannelName(index int) string {
	if index >= 0 && index < len(standardChannelNames) {
		return standardChannelNames[index]
	}
	return fmt.Sprintf("Channel %d", index+1)
}

// Analyze performs comprehensive dynamics, RMS, and Crest Factor calculation on a PCM buffer.
func (a *CrestFactorAnalyzer) Analyze(pcm *PCMBuffer) (*CrestFactorResult, error) {
	if pcm == nil {
		return nil, ErrNilPCMBuffer
	}
	if pcm.NumSamples == 0 || len(pcm.Samples) == 0 {
		return nil, ErrEmptyAudioBuffer
	}
	if pcm.SampleRate <= 0 {
		return nil, errors.New("invalid sample rate")
	}

	channels := pcm.Channels
	numSamples := pcm.NumSamples
	durationSec := float64(numSamples) / float64(pcm.SampleRate)

	// 1. Channel-wise RMS and Peak Analysis
	channelBreakdown := make([]ChannelDynamics, channels)
	var globalSumSquares float64 = 0.0
	var globalPeak float64 = 0.0
	var globalTruePeak float64 = 0.0

	for ch := 0; ch < channels; ch++ {
		chSamples := pcm.Samples[ch]
		var sumSquares float64 = 0.0
		var peak float64 = 0.0

		for n := 0; n < numSamples; n++ {
			s := chSamples[n]
			sumSquares += s * s
			absS := math.Abs(s)
			if absS > peak {
				peak = absS
			}
		}

		globalSumSquares += sumSquares
		if peak > globalPeak {
			globalPeak = peak
		}

		chRMS := math.Sqrt(sumSquares / float64(numSamples))
		chRMSDb := -120.0
		if chRMS > 1e-6 {
			chRMSDb = 20.0 * math.Log10(chRMS)
		}
		chAES17Db := chRMSDb + 3.0103

		chPeakDb := -120.0
		if peak > 1e-6 {
			chPeakDb = 20.0 * math.Log10(peak)
		}

		// True Peak Calculation with 4x Sinc Oversampling
		chTruePeak := peak
		chTruePeakDb := chPeakDb
		if a.config.EnableTruePeak {
			chBuf := &PCMBuffer{
				Channels:   1,
				SampleRate: pcm.SampleRate,
				NumSamples: numSamples,
				Samples:    [][]float64{chSamples},
			}
			tpDb, err := measureTruePeak4x(chBuf)
			if err == nil {
				if tpDb < chPeakDb {
					tpDb = chPeakDb
				}
				chTruePeakDb = tpDb
				chTruePeak = math.Pow(10.0, tpDb/20.0)
			}
		}
		if chTruePeak < peak {
			chTruePeak = peak
			chTruePeakDb = chPeakDb
		}
		if chTruePeak > globalTruePeak {
			globalTruePeak = chTruePeak
		}

		// Channel Crest Factor
		chCFRatio := 1.0
		chCFDb := 0.0
		if chRMS > 1e-6 {
			chCFRatio = peak / chRMS
			chCFDb = 20.0 * math.Log10(chCFRatio)
		}

		channelBreakdown[ch] = ChannelDynamics{
			ChannelIndex:     ch,
			ChannelName:      getChannelName(ch),
			RMSDbFS:          chRMSDb,
			AES17RMSDbFS:     chAES17Db,
			PeakDbFS:         chPeakDb,
			TruePeakDbTP:     chTruePeakDb,
			CrestFactorRatio: chCFRatio,
			CrestFactorDb:    chCFDb,
		}
	}

	// 2. Global Integrated Dynamics
	totalEvaluatedSamples := float64(channels * numSamples)
	globalRMS := math.Sqrt(globalSumSquares / totalEvaluatedSamples)

	globalRMSDb := -120.0
	if globalRMS > 1e-6 {
		globalRMSDb = 20.0 * math.Log10(globalRMS)
	}
	globalAES17Db := globalRMSDb + 3.0103

	globalPeakDb := -120.0
	if globalPeak > 1e-6 {
		globalPeakDb = 20.0 * math.Log10(globalPeak)
	}

	if globalTruePeak < globalPeak {
		globalTruePeak = globalPeak
	}
	globalTruePeakDb := -120.0
	if globalTruePeak > 1e-6 {
		globalTruePeakDb = 20.0 * math.Log10(globalTruePeak)
	}

	globalCFRatio := 1.0
	globalCFDb := 0.0
	if globalRMS > 1e-6 {
		globalCFRatio = globalPeak / globalRMS
		globalCFDb = 20.0 * math.Log10(globalCFRatio)
	}

	// 3. Short-Term Sliding Crest Factor Dynamics (400ms window, 100ms hop)
	windowSamples := int(a.config.WindowSizeSec * float64(pcm.SampleRate))
	hopSamples := int(a.config.WindowHopSec * float64(pcm.SampleRate))
	if windowSamples > numSamples {
		windowSamples = numSamples
	}
	if hopSamples < 1 {
		hopSamples = 1
	}

	minShortTermCFDb := 120.0
	maxShortTermCFDb := 0.0

	// Also collect block RMS values for official TT Dynamic Range (DR) score calculation
	var blockRMSList []float64

	for start := 0; start+windowSamples <= numSamples; start += hopSamples {
		end := start + windowSamples
		var blockSumSq float64 = 0.0
		var blockPeak float64 = 0.0

		for ch := 0; ch < channels; ch++ {
			chSamples := pcm.Samples[ch]
			for n := start; n < end; n++ {
				s := chSamples[n]
				blockSumSq += s * s
				absS := math.Abs(s)
				if absS > blockPeak {
					blockPeak = absS
				}
			}
		}

		blockCount := float64(channels * windowSamples)
		blockRMS := math.Sqrt(blockSumSq / blockCount)

		if blockRMS > 1e-5 {
			blockRMSList = append(blockRMSList, blockRMS)
			cf := blockPeak / blockRMS
			cfDb := 20.0 * math.Log10(cf)
			if cfDb < minShortTermCFDb {
				minShortTermCFDb = cfDb
			}
			if cfDb > maxShortTermCFDb {
				maxShortTermCFDb = cfDb
			}
		}
	}

	if minShortTermCFDb > 100.0 {
		minShortTermCFDb = globalCFDb
	}
	if maxShortTermCFDb < 1e-5 {
		maxShortTermCFDb = globalCFDb
	}
	dynamicSpreadDb := math.Max(0.0, maxShortTermCFDb-minShortTermCFDb)

	// 4. TT Dynamic Range (DR) Score (Plea for Sanity Standard)
	// Evaluates the difference between track peak and RMS of the loudest 20% passages
	drScore := int(math.Round(globalCFDb))
	if len(blockRMSList) > 0 {
		sort.Float64s(blockRMSList)
		// Top 20% loudest blocks
		top20Count := int(math.Ceil(float64(len(blockRMSList)) * 0.20))
		if top20Count < 1 {
			top20Count = 1
		}
		top20Start := len(blockRMSList) - top20Count
		var top20Sum float64 = 0.0
		for i := top20Start; i < len(blockRMSList); i++ {
			top20Sum += blockRMSList[i] * blockRMSList[i]
		}
		top20RMS := math.Sqrt(top20Sum / float64(top20Count))
		if top20RMS > 1e-6 && globalPeak > 1e-6 {
			drRaw := 20.0 * math.Log10(globalPeak/top20RMS)
			drScore = int(math.Round(drRaw))
		}
	}
	if drScore < 1 {
		drScore = 1
	}

	// 5. Categorization and Loudness War Flagging
	isOverCompressed := globalCFDb < a.config.OverCompressedCutoffDb || drScore < 7

	var rating DynamicRating
	var description string

	switch {
	case drScore >= 14 && globalCFDb >= 14.0:
		rating = RatingAudiophileDynamic
		description = "Audiophile Dynamic Master (High Fidelity, Uncompressed Transients)"
	case drScore >= 10 && globalCFDb >= 10.0:
		rating = RatingHealthyMaster
		description = "Healthy Dynamic Master (Balanced Punch and Dynamics)"
	case drScore >= 7 && globalCFDb >= 7.0:
		rating = RatingModerateComp
		description = "Moderate Compression (Standard Commercial Radio Master)"
	case drScore >= 4 && globalCFDb >= 4.0:
		rating = RatingLoudnessWar
		description = "Heavy Compression / Loudness War Casualty (Squashed Dynamics)"
	default:
		rating = RatingExtremeSquashed
		description = "Extreme Compression / Brickwall Clipped (Zero Dynamic Contrast)"
	}

	// 6. UI Badge Formatting
	uiBadge := fmt.Sprintf("DR%d · CF %.1f dB · %s", drScore, globalCFDb, description)

	return &CrestFactorResult{
		SampleRate:          pcm.SampleRate,
		Channels:            channels,
		DurationSec:         durationSec,
		TotalSamples:        numSamples,
		GlobalRMSDbFS:       globalRMSDb,
		GlobalAES17RMSDbFS:  globalAES17Db,
		GlobalPeakDbFS:      globalPeakDb,
		GlobalTruePeakDbTP:  globalTruePeakDb,
		CrestFactorRatio:    globalCFRatio,
		CrestFactorDb:       globalCFDb,
		DRScore:             drScore,
		Rating:              rating,
		Description:         description,
		IsOverCompressed:    isOverCompressed,
		MinShortTermCrestDb: minShortTermCFDb,
		MaxShortTermCrestDb: maxShortTermCFDb,
		DynamicSpreadDb:     dynamicSpreadDb,
		ChannelBreakdown:    channelBreakdown,
		UIBadge:             uiBadge,
	}, nil
}
