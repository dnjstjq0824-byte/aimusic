package audio

import (
	"fmt"
	"math"
)

// DRCProfile specifies standard presets for Dynamic Range Compression.
type DRCProfile string

const (
	DRCProfileBroadcast        DRCProfile = "BROADCAST"
	DRCProfileNightMode        DRCProfile = "NIGHT_MODE"
	DRCProfileBrickwallLimiter DRCProfile = "BRICKWALL_LIMITER"
	DRCProfileAcousticGentle   DRCProfile = "ACOUSTIC_GENTLE"
	DRCProfileCustom           DRCProfile = "CUSTOM"
)

// DRCConfig configures dynamic range compression parameters.
type DRCConfig struct {
	Profile       DRCProfile
	ThresholdDB   float64 // Threshold in dBFS (-60.0 to 0.0)
	Ratio         float64 // Compression ratio (>= 1.0)
	KneeWidthDB   float64 // Knee width in dB (0.0 for hard knee, > 0.0 for soft knee)
	AttackMs      float64 // Attack time in milliseconds (0.1 to 500.0 ms)
	ReleaseMs     float64 // Release time in milliseconds (1.0 to 5000.0 ms)
	MakeupGainDB  float64 // Manual makeup gain in dB
	AutoMakeup    bool    // Automatically calculate makeup gain
	StereoLink    bool    // Link channels for envelope detection to preserve stereo imaging
	CeilingDB     float64 // Output peak limiter ceiling in dBFS (-20.0 to 0.0)
	EnableLimiter bool    // Enable final brickwall limiter stage
}

// DRCResult holds processed audio buffer and compression telemetry metrics.
type DRCResult struct {
	ProcessedBuffer              *PCMBuffer
	MaxGainReductionDB           float64
	AverageGainReductionDB       float64
	OriginalDynamicRangeDB       float64
	CompressedDynamicRangeDB     float64
	DynamicRangeCompressionRatio float64
	PeakInputDBFS                float64
	PeakOutputDBFS               float64
}

// DefaultDRCConfig returns standard configuration based on selected profile.
func DefaultDRCConfig(profile DRCProfile) DRCConfig {
	switch profile {
	case DRCProfileBroadcast:
		return DRCConfig{
			Profile:       DRCProfileBroadcast,
			ThresholdDB:   -20.0,
			Ratio:         3.0,
			KneeWidthDB:   6.0,
			AttackMs:      10.0,
			ReleaseMs:     200.0,
			MakeupGainDB:  4.0,
			AutoMakeup:    false,
			StereoLink:    true,
			CeilingDB:     -1.0,
			EnableLimiter: true,
		}
	case DRCProfileNightMode:
		return DRCConfig{
			Profile:       DRCProfileNightMode,
			ThresholdDB:   -24.0,
			Ratio:         4.5,
			KneeWidthDB:   10.0,
			AttackMs:      5.0,
			ReleaseMs:     120.0,
			MakeupGainDB:  6.0,
			AutoMakeup:    false,
			StereoLink:    true,
			CeilingDB:     -1.5,
			EnableLimiter: true,
		}
	case DRCProfileBrickwallLimiter:
		return DRCConfig{
			Profile:       DRCProfileBrickwallLimiter,
			ThresholdDB:   -0.5,
			Ratio:         20.0,
			KneeWidthDB:   0.0,
			AttackMs:      0.5,
			ReleaseMs:     50.0,
			MakeupGainDB:  0.0,
			AutoMakeup:    false,
			StereoLink:    true,
			CeilingDB:     -0.5,
			EnableLimiter: true,
		}
	case DRCProfileAcousticGentle:
		return DRCConfig{
			Profile:       DRCProfileAcousticGentle,
			ThresholdDB:   -14.0,
			Ratio:         1.8,
			KneeWidthDB:   12.0,
			AttackMs:      20.0,
			ReleaseMs:     300.0,
			MakeupGainDB:  1.5,
			AutoMakeup:    false,
			StereoLink:    true,
			CeilingDB:     -0.5,
			EnableLimiter: true,
		}
	default:
		return DRCConfig{
			Profile:       DRCProfileCustom,
			ThresholdDB:   -18.0,
			Ratio:         2.5,
			KneeWidthDB:   4.0,
			AttackMs:      10.0,
			ReleaseMs:     150.0,
			MakeupGainDB:  0.0,
			AutoMakeup:    false,
			StereoLink:    true,
			CeilingDB:     -0.5,
			EnableLimiter: true,
		}
	}
}

// DRCEngine implements high-precision dynamic range compression and limiting.
type DRCEngine struct {
	config DRCConfig
}

// NewDRCEngine creates a dynamic range compressor engine instance.
func NewDRCEngine(config DRCConfig) *DRCEngine {
	if config.Ratio < 1.0 {
		config.Ratio = 1.0
	}
	if config.AttackMs < 0.01 {
		config.AttackMs = 0.01
	}
	if config.ReleaseMs < 0.1 {
		config.ReleaseMs = 0.1
	}
	if config.KneeWidthDB < 0.0 {
		config.KneeWidthDB = 0.0
	}
	if config.CeilingDB > 0.0 {
		config.CeilingDB = 0.0
	}
	if config.AutoMakeup {
		// Theoretical makeup gain compensation formula
		config.MakeupGainDB = -(config.ThresholdDB * (1.0 - (1.0 / config.Ratio))) * 0.6
	}

	return &DRCEngine{config: config}
}

// ComputeStaticGainReduction computes target gain reduction in dB for a given input level in dBFS.
func (e *DRCEngine) ComputeStaticGainReduction(inputDB float64) float64 {
	T := e.config.ThresholdDB
	W := e.config.KneeWidthDB
	R := e.config.Ratio

	if R <= 1.0 {
		return 0.0
	}

	var outputDB float64
	if W <= 0.0001 {
		// Hard knee characteristic
		if inputDB <= T {
			outputDB = inputDB
		} else {
			outputDB = T + (inputDB-T)/R
		}
	} else {
		// Quadratic soft knee characteristic
		halfW := W / 2.0
		if inputDB <= T-halfW {
			outputDB = inputDB
		} else if inputDB >= T+halfW {
			outputDB = T + (inputDB-T)/R
		} else {
			diff := inputDB - T + halfW
			outputDB = inputDB + ((1.0/R - 1.0) * (diff * diff)) / (2.0 * W)
		}
	}

	// Target gain reduction is output level minus input level (always <= 0 dB)
	gainReductionDB := outputDB - inputDB
	if gainReductionDB > 0.0 {
		gainReductionDB = 0.0
	}
	return gainReductionDB
}

// Process applies dynamic range compression and brickwall peak limiting to the input audio buffer.
func (e *DRCEngine) Process(pcm *PCMBuffer) (*DRCResult, error) {
	if pcm == nil {
		return nil, ErrNilPCMBuffer
	}
	if len(pcm.Samples) == 0 || pcm.Channels == 0 {
		return nil, ErrEmptyPCMBuffer
	}

	numChannels := pcm.Channels
	sampleRate := float64(pcm.SampleRate)
	numSamplesPerChannel := len(pcm.Samples[0])

	// Compute attack and release smoothing filter coefficients
	// Alpha = exp(-1.0 / (sampleRate * (timeMs / 1000.0)))
	alphaAttack := math.Exp(-1.0 / (sampleRate * (e.config.AttackMs / 1000.0)))
	alphaRelease := math.Exp(-1.0 / (sampleRate * (e.config.ReleaseMs / 1000.0)))

	makeupLinear := math.Pow(10.0, e.config.MakeupGainDB/20.0)
	ceilingLinear := math.Pow(10.0, e.config.CeilingDB/20.0)

	// Create output buffer
	outputSamples := make([][]float64, numChannels)
	for ch := 0; ch < numChannels; ch++ {
		outputSamples[ch] = make([]float64, numSamplesPerChannel)
	}

	envelopeGR := make([]float64, numChannels) // Envelope per channel in dB (initialized to 0 dB)
	linkedEnvelopeGR := 0.0

	var maxGR float64 = 0.0
	var sumGR float64 = 0.0
	var totalEvaluations int64 = 0

	var peakInputLin float64 = 0.0
	var peakOutputLin float64 = 0.0

	// Statistical tracking for original and compressed dynamic range
	origRMS := 0.0
	compRMS := 0.0

	const minLevel = 1e-6 // -120 dBFS noise floor

	for i := 0; i < numSamplesPerChannel; i++ {
		if e.config.StereoLink {
			// Find max absolute level across all channels
			maxSampleAbs := minLevel
			for ch := 0; ch < numChannels; ch++ {
				absVal := math.Abs(pcm.Samples[ch][i])
				if absVal > maxSampleAbs {
					maxSampleAbs = absVal
				}
				if absVal > peakInputLin {
					peakInputLin = absVal
				}
				origRMS += absVal * absVal
			}

			// Compute input level in dBFS
			inputDB := 20.0 * math.Log10(maxSampleAbs)
			targetGR := e.ComputeStaticGainReduction(inputDB)

			// Ballistics envelope follower
			if targetGR < linkedEnvelopeGR {
				// Attack phase: compression increases (gain drops)
				linkedEnvelopeGR = alphaAttack*linkedEnvelopeGR + (1.0-alphaAttack)*targetGR
			} else {
				// Release phase: compression decreases (gain rises towards 0 dB)
				linkedEnvelopeGR = alphaRelease*linkedEnvelopeGR + (1.0-alphaRelease)*targetGR
			}

			if linkedEnvelopeGR < maxGR {
				maxGR = linkedEnvelopeGR
			}
			sumGR += linkedEnvelopeGR
			totalEvaluations++

			currentGainLinear := math.Pow(10.0, linkedEnvelopeGR/20.0) * makeupLinear

			// Apply gain to each channel
			for ch := 0; ch < numChannels; ch++ {
				outVal := pcm.Samples[ch][i] * currentGainLinear

				// Secondary brickwall ceiling limiter if enabled
				if e.config.EnableLimiter {
					if outVal > ceilingLinear {
						outVal = ceilingLinear
					} else if outVal < -ceilingLinear {
						outVal = -ceilingLinear
					}
				}

				outputSamples[ch][i] = outVal
				absOut := math.Abs(outVal)
				if absOut > peakOutputLin {
					peakOutputLin = absOut
				}
				compRMS += absOut * absOut
			}

		} else {
			// Independent per-channel compression
			for ch := 0; ch < numChannels; ch++ {
				sampleVal := pcm.Samples[ch][i]
				absVal := math.Abs(sampleVal)
				if absVal > peakInputLin {
					peakInputLin = absVal
				}
				origRMS += absVal * absVal

				effectiveAbs := absVal
				if effectiveAbs < minLevel {
					effectiveAbs = minLevel
				}

				inputDB := 20.0 * math.Log10(effectiveAbs)
				targetGR := e.ComputeStaticGainReduction(inputDB)

				if targetGR < envelopeGR[ch] {
					envelopeGR[ch] = alphaAttack*envelopeGR[ch] + (1.0-alphaAttack)*targetGR
				} else {
					envelopeGR[ch] = alphaRelease*envelopeGR[ch] + (1.0-alphaRelease)*targetGR
				}

				if envelopeGR[ch] < maxGR {
					maxGR = envelopeGR[ch]
				}
				sumGR += envelopeGR[ch]
				totalEvaluations++

				currentGainLinear := math.Pow(10.0, envelopeGR[ch]/20.0) * makeupLinear
				outVal := sampleVal * currentGainLinear

				if e.config.EnableLimiter {
					if outVal > ceilingLinear {
						outVal = ceilingLinear
					} else if outVal < -ceilingLinear {
						outVal = -ceilingLinear
					}
				}

				outputSamples[ch][i] = outVal
				absOut := math.Abs(outVal)
				if absOut > peakOutputLin {
					peakOutputLin = absOut
				}
				compRMS += absOut * absOut
			}
		}
	}

	origLoudnessRange := computeLoudnessRange(pcm.Samples, pcm.SampleRate)
	compLoudnessRange := computeLoudnessRange(outputSamples, pcm.SampleRate)

	origPeakDBFS := -120.0
	if peakInputLin > minLevel {
		origPeakDBFS = 20.0 * math.Log10(peakInputLin)
	}

	compPeakDBFS := -120.0
	if peakOutputLin > minLevel {
		compPeakDBFS = 20.0 * math.Log10(peakOutputLin)
	}

	drcRatio := 1.0
	if compLoudnessRange > 0.001 {
		drcRatio = origLoudnessRange / compLoudnessRange
	}

	avgGR := 0.0
	if totalEvaluations > 0 {
		avgGR = sumGR / float64(totalEvaluations)
	}

	outPCM := &PCMBuffer{
		SampleRate:  pcm.SampleRate,
		Channels:    pcm.Channels,
		NumSamples:  numSamplesPerChannel,
		Samples:     outputSamples,
	}

	return &DRCResult{
		ProcessedBuffer:              outPCM,
		MaxGainReductionDB:           maxGR,
		AverageGainReductionDB:       avgGR,
		OriginalDynamicRangeDB:       origLoudnessRange,
		CompressedDynamicRangeDB:     compLoudnessRange,
		DynamicRangeCompressionRatio: drcRatio,
		PeakInputDBFS:                origPeakDBFS,
		PeakOutputDBFS:               compPeakDBFS,
	}, nil
}

// computeLoudnessRange calculates short-term dynamic range (difference between maximum and minimum active block RMS in dB).
func computeLoudnessRange(samples [][]float64, sampleRate int) float64 {
	if len(samples) == 0 || len(samples[0]) == 0 {
		return 0.0
	}
	windowSize := sampleRate / 10 // 100ms window
	if windowSize < 16 {
		windowSize = 16
	}
	numChannels := len(samples)
	numSamples := len(samples[0])

	var blockRMSs []float64
	for start := 0; start < numSamples; start += windowSize {
		end := start + windowSize
		if end > numSamples {
			end = numSamples
		}
		count := end - start
		if count < windowSize/2 {
			break
		}

		sumSq := 0.0
		for ch := 0; ch < numChannels; ch++ {
			for i := start; i < end; i++ {
				s := samples[ch][i]
				sumSq += s * s
			}
		}
		rms := math.Sqrt(sumSq / float64(numChannels*count))
		if rms > 1e-5 { // Active sound > -100 dBFS
			blockRMSs = append(blockRMSs, 20.0*math.Log10(rms))
		}
	}

	if len(blockRMSs) <= 1 {
		return 0.0
	}

	minVal := blockRMSs[0]
	maxVal := blockRMSs[0]
	for _, v := range blockRMSs {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal - minVal
}

// CalculateEffectiveGainReduction returns the dynamic range reduction in dB between quiet and loud passages.
func (r *DRCResult) String() string {
	return fmt.Sprintf("DRC Result: MaxGR=%.2f dB, AvgGR=%.2f dB, DynamicRange=[%.2f dB -> %.2f dB], PeakOut=%.2f dBFS",
		r.MaxGainReductionDB,
		r.AverageGainReductionDB,
		r.OriginalDynamicRangeDB,
		r.CompressedDynamicRangeDB,
		r.PeakOutputDBFS,
	)
}
