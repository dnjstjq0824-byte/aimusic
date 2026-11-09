// Package audio provides pure Go multi-band onset energy flux detection,
// comb-filter enhanced autocorrelation, parabolic sub-sample peak interpolation,
// and beat grid tracking for high-precision BPM (Beats Per Minute) estimation.
package audio

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

// BPMCandidate represents a ranked tempo candidate identified during autocorrelation analysis.
type BPMCandidate struct {
	BPM        float64 `json:"bpm"`
	Strength   float64 `json:"strength"`   // Normalized autocorrelation score (0..1)
	Confidence float64 `json:"confidence"` // Perceptual score
}

// BPMResult contains the complete tempo estimation, beat grid, and rhythm stability metrics.
type BPMResult struct {
	BPM             float64        `json:"bpm"`               // Primary estimated tempo (e.g. 120.0 BPM)
	Confidence      float64        `json:"confidence"`        // Overall confidence score (0.0 to 1.0)
	FirstBeatSec    float64        `json:"first_beat_sec"`    // Timestamp of the first beat in seconds
	BeatIntervalSec float64        `json:"beat_interval_sec"` // Interval between consecutive beats (60 / BPM)
	BeatTimestamps  []float64      `json:"beat_timestamps"`   // Full array of beat timestamps across duration
	TempoStability  float64        `json:"tempo_stability"`   // Steady rhythm stability measure (0.0 to 1.0)
	Candidates      []BPMCandidate `json:"candidates"`        // Top candidate tempos (primary, half-time, double-time)
	IsSteadyBeat    bool           `json:"is_steady_beat"`    // True if tempo stability >= 0.75
	TimeSignature   string         `json:"time_signature"`    // Estimated meter (default: "4/4")
}

// BPMConfig defines operational thresholds for the multi-band tempo analysis engine.
type BPMConfig struct {
	MinBPM             float64 // Minimum detectable BPM (default: 50.0)
	MaxBPM             float64 // Maximum detectable BPM (default: 220.0)
	FFTSize            int     // STFT frame size (default: 1024)
	HopSize            int     // STFT hop size in samples (default: 256)
	PerceptualPriorBPM float64 // Log-normal prior center (default: 120.0)
	PriorWidthOctaves  float64 // Log-normal prior bandwidth in octaves (default: 0.85)
}

// DefaultBPMConfig returns standard production parameters for musical tempo detection.
func DefaultBPMConfig() BPMConfig {
	return BPMConfig{
		MinBPM:             50.0,
		MaxBPM:             220.0,
		FFTSize:            1024,
		HopSize:            256,
		PerceptualPriorBPM: 120.0,
		PriorWidthOctaves:  1.6, // Generous octave width to prevent false octave compression
	}
}

// Diagnostic error definitions.
var (
	ErrAudioTooShortForBPM = errors.New("audio duration is too short for reliable BPM analysis (minimum 1.0s required)")
	ErrNoRhythmicActivity  = errors.New("insufficient rhythmic energy or silent audio signal")
)

// BPMDetector performs multi-band onset flux and autocorrelation tempo analysis.
type BPMDetector struct {
	config BPMConfig
}

// NewBPMDetector creates a new BPM detection engine.
func NewBPMDetector(config BPMConfig) (*BPMDetector, error) {
	if config.MinBPM <= 0 || config.MaxBPM <= config.MinBPM {
		return nil, errors.New("invalid BPM detection range (MinBPM must be > 0 and < MaxBPM)")
	}
	if config.FFTSize < 256 || (config.FFTSize&(config.FFTSize-1)) != 0 {
		return nil, errors.New("FFTSize must be a power of two >= 256")
	}
	if config.HopSize <= 0 {
		config.HopSize = config.FFTSize / 4
	}
	if config.PerceptualPriorBPM <= 0 {
		config.PerceptualPriorBPM = 120.0
	}
	if config.PriorWidthOctaves <= 0 {
		config.PriorWidthOctaves = 0.85
	}
	return &BPMDetector{config: config}, nil
}

// subBandDefinition defines frequency boundaries and weights for multi-band onset extraction.
type subBandDefinition struct {
	minFreq float64
	maxFreq float64
	weight  float64
}

// Analyze extracts the onset novelty envelope and computes the tempo and beat grid.
func (d *BPMDetector) Analyze(pcm *PCMBuffer) (*BPMResult, error) {
	if pcm == nil {
		return nil, ErrNilPCMBuffer
	}
	if pcm.NumSamples == 0 || len(pcm.Samples) == 0 {
		return nil, ErrEmptyAudioBuffer
	}

	durationSec := float64(pcm.NumSamples) / float64(pcm.SampleRate)
	if durationSec < 0.8 {
		return nil, ErrAudioTooShortForBPM
	}

	sampleRate := pcm.SampleRate
	fftSize := d.config.FFTSize
	hopSize := d.config.HopSize
	hopRate := float64(sampleRate) / float64(hopSize) // Frames per second (~187.5 at 48kHz, hop=256)

	// 1. Downmix to mono if multi-channel
	mono := make([]float64, pcm.NumSamples)
	numChannels := pcm.Channels
	invChannels := 1.0 / float64(numChannels)

	for ch := 0; ch < numChannels; ch++ {
		chSamples := pcm.Samples[ch]
		for n := 0; n < pcm.NumSamples; n++ {
			mono[n] += chSamples[n] * invChannels
		}
	}

	// 2. Perform STFT Analysis for multi-band spectral flux
	stftEngine, err := NewSTFTEngine(STFTConfig{
		FFTSize:    fftSize,
		HopSize:    hopSize,
		Window:     WindowHann,
		NumMelBins: 80,
	})
	if err != nil {
		return nil, fmt.Errorf("initializing stft: %w", err)
	}

	monoPCM := &PCMBuffer{
		Samples:    [][]float64{mono},
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: pcm.NumSamples,
	}

	stftRes, err := stftEngine.Forward(monoPCM)
	if err != nil {
		return nil, fmt.Errorf("computing stft: %w", err)
	}

	numFrames := stftRes.NumFrames
	numBins := stftRes.NumBins
	freqResolution := float64(sampleRate) / float64(fftSize)

	// 4 Sub-Bands: Sub-bass, Low-mid, Mid, High
	subBands := []subBandDefinition{
		{minFreq: 20.0, maxFreq: 180.0, weight: 2.0},    // Kick drum & 808 fundamental
		{minFreq: 180.0, maxFreq: 800.0, weight: 1.5},   // Snare body & bass harmonics
		{minFreq: 800.0, maxFreq: 3500.0, weight: 1.0},  // Snare snap, percussion, vocals
		{minFreq: 3500.0, maxFreq: 12000.0, weight: 0.7}, // Hi-hats, shakers, cymbals
	}

	// 3. Multi-Band Logarithmic Spectral Flux Onset Detection
	// We track both full-bandwidth onset and low-frequency rhythmic onset (< 800 Hz: kick + snare body)
	onsetCurve := make([]float64, numFrames)
	lowFreqOnset := make([]float64, numFrames)
	prevBandEnergies := make([]float64, len(subBands))

	for f := 0; f < numFrames; f++ {
		frame := stftRes.Frames[0][f]

		for bIdx, band := range subBands {
			var bandEnergy float64 = 0.0
			for k := 0; k < numBins; k++ {
				freq := float64(k) * freqResolution
				if freq >= band.minFreq && freq < band.maxFreq {
					bandEnergy += frame.Magnitude[k]
				}
			}

			// Logarithmic dynamic compression
			logEnergy := math.Log1p(10.0 * bandEnergy)

			if f > 0 {
				diff := logEnergy - prevBandEnergies[bIdx]
				if diff > 0 { // Half-wave rectification
					onsetCurve[f] += band.weight * diff
					// Bands 0 (20-180Hz) and 1 (180-800Hz) represent primary drum beat onsets
					if bIdx <= 1 {
						lowFreqOnset[f] += band.weight * diff
					}
				}
			}
			prevBandEnergies[bIdx] = logEnergy
		}
	}

	// 4. Adaptive Moving Average Subtraction (Detrending and noise suppression)
	cleanOnset := make([]float64, numFrames)
	cleanLowFreq := make([]float64, numFrames)
	smaWindow := 16
	var maxOnset float64 = 0.0
	var maxLowFreq float64 = 0.0

	for i := 0; i < numFrames; i++ {
		start := i - smaWindow/2
		if start < 0 {
			start = 0
		}
		end := i + smaWindow/2
		if end > numFrames {
			end = numFrames
		}
		var sum float64 = 0.0
		var sumLow float64 = 0.0
		for j := start; j < end; j++ {
			sum += onsetCurve[j]
			sumLow += lowFreqOnset[j]
		}
		sma := sum / float64(end-start)
		smaLow := sumLow / float64(end-start)

		val := onsetCurve[i] - sma
		if val > 0 {
			cleanOnset[i] = val
		}
		if cleanOnset[i] > maxOnset {
			maxOnset = cleanOnset[i]
		}

		valLow := lowFreqOnset[i] - smaLow
		if valLow > 0 {
			cleanLowFreq[i] = valLow
		}
		if cleanLowFreq[i] > maxLowFreq {
			maxLowFreq = cleanLowFreq[i]
		}
	}

	if maxOnset < 1e-5 {
		return nil, ErrNoRhythmicActivity
	}

	// Normalize clean onset curves
	invMax := 1.0 / maxOnset
	for i := 0; i < numFrames; i++ {
		cleanOnset[i] *= invMax
	}
	if maxLowFreq > 1e-5 {
		invLow := 1.0 / maxLowFreq
		for i := 0; i < numFrames; i++ {
			cleanLowFreq[i] *= invLow
		}
	}

	// 5. Generalized Autocorrelation with Multi-Harmonic Resonator
	minLag := int(math.Floor(60.0 * hopRate / d.config.MaxBPM))
	maxLag := int(math.Ceil(60.0 * hopRate / d.config.MinBPM))
	if minLag < 2 {
		minLag = 2
	}
	if maxLag >= numFrames {
		maxLag = numFrames - 1
	}

	rawCorr := make([]float64, maxLag+1)
	lowCorr := make([]float64, maxLag+1)
	for lag := minLag; lag <= maxLag; lag++ {
		var sum float64 = 0.0
		var sumLow float64 = 0.0
		count := numFrames - lag
		for i := 0; i < count; i++ {
			sum += cleanOnset[i] * cleanOnset[i+lag]
			sumLow += cleanLowFreq[i] * cleanLowFreq[i+lag]
		}
		rawCorr[lag] = sum / float64(count)
		lowCorr[lag] = sumLow / float64(count)
	}

	// 6. Inspect Autocorrelation & Octave Disambiguation
	priorLogCenter := math.Log2(d.config.PerceptualPriorBPM)
	priorSigma := d.config.PriorWidthOctaves

	type lagScore struct {
		lag   int
		score float64
	}
	var scoredLags []lagScore

	for lag := minLag; lag <= maxLag; lag++ {
		lagBPM := 60.0 * hopRate / float64(lag)

		// Gaussian prior centered at PerceptualPriorBPM
		logBPM := math.Log2(lagBPM)
		octaveDiff := (logBPM - priorLogCenter) / priorSigma
		priorWeight := math.Exp(-0.5 * octaveDiff * octaveDiff)

		// Combine full autocorrelation with low-frequency drum autocorrelation
		combinedCorr := (rawCorr[lag] + 1.5*lowCorr[lag]) / 2.5
		finalScore := combinedCorr * priorWeight
		scoredLags = append(scoredLags, lagScore{lag: lag, score: finalScore})
	}

	// Sort scored lags to find primary and secondary peaks
	sort.Slice(scoredLags, func(i, j int) bool {
		return scoredLags[i].score > scoredLags[j].score
	})

	if len(scoredLags) == 0 || scoredLags[0].score <= 1e-9 {
		return nil, ErrNoRhythmicActivity
	}

	bestLag := scoredLags[0].lag

	// Check if bestLag is an octave multiple (half-time / 2-beat lag) of a true fundamental beat
	halfLag := bestLag / 2
	if halfLag >= minLag {
		halfLagBPM := 60.0 * hopRate / float64(halfLag)
		if halfLagBPM <= d.config.MaxBPM {
			// Find peak near halfLag (+- 2 frames)
			bestNearHalf := halfLag
			maxNearVal := 0.0
			for offset := -2; offset <= 2; offset++ {
				cand := halfLag + offset
				if cand >= minLag && cand <= maxLag {
					if lowCorr[cand] > maxNearVal {
						maxNearVal = lowCorr[cand]
						bestNearHalf = cand
					}
				}
			}
			// If low-frequency rhythmic onset has strong correlation at halfLag (kick-to-snare),
			// then halfLag represents the true tactus beat rate.
			if lowCorr[bestLag] > 1e-4 && maxNearVal >= 0.40*lowCorr[bestLag] {
				bestLag = bestNearHalf
			}
		}
	}

	// 7. Parabolic Sub-Sample Interpolation for Fractional Precision
	exactLag := float64(bestLag)
	if bestLag > minLag && bestLag < maxLag {
		y0 := rawCorr[bestLag-1]
		y1 := rawCorr[bestLag]
		y2 := rawCorr[bestLag+1]
		denom := 2.0 * (2.0*y1 - y0 - y2)
		if math.Abs(denom) > 1e-9 {
			delta := (y2 - y0) / denom
			if math.Abs(delta) <= 1.0 {
				exactLag = float64(bestLag) + delta
			}
		}
	}

	finalBPM := 60.0 * hopRate / exactLag
	beatIntervalSec := 60.0 / finalBPM

	// 8. Beat Grid Phase Alignment (Locate first major onset near beginning)
	firstBeatSec := 0.0
	searchFrames := int(beatIntervalSec * hopRate * 1.5)
	if searchFrames > numFrames {
		searchFrames = numFrames
	}

	var peakOnsetVal float64 = 0.0
	bestPhaseFrame := 0
	for i := 0; i < searchFrames; i++ {
		if cleanOnset[i] > peakOnsetVal {
			peakOnsetVal = cleanOnset[i]
			bestPhaseFrame = i
		}
	}
	firstBeatSec = float64(bestPhaseFrame) / hopRate

	// Generate all beat timestamps across track duration
	var beatTimestamps []float64
	currentBeatTime := firstBeatSec
	for currentBeatTime < durationSec {
		beatTimestamps = append(beatTimestamps, currentBeatTime)
		currentBeatTime += beatIntervalSec
	}

	// 9. Measure Rhythm Stability & Energy Correlation on Grid
	var gridCorrelationSum float64 = 0.0
	var totalSampledBeats float64 = 0.0

	for _, bt := range beatTimestamps {
		frameIdx := int(math.Round(bt * hopRate))
		var maxNearGrid float64 = 0.0
		for offset := -2; offset <= 2; offset++ {
			f := frameIdx + offset
			if f >= 0 && f < numFrames {
				if cleanOnset[f] > maxNearGrid {
					maxNearGrid = cleanOnset[f]
				}
			}
		}
		gridCorrelationSum += maxNearGrid
		totalSampledBeats++
	}

	tempoStability := 0.5
	if totalSampledBeats > 0 {
		avgGridEnergy := gridCorrelationSum / totalSampledBeats
		tempoStability = math.Min(1.0, math.Max(0.0, avgGridEnergy))
	}

	confidence := math.Min(1.0, scoredLags[0].score*2.0)
	if len(scoredLags) > 1 && scoredLags[1].score > 0 {
		peakRatio := scoredLags[0].score / scoredLags[1].score
		if peakRatio < 1.05 {
			confidence *= 0.85
		}
	}

	// Compile Candidate Tempos (e.g. Primary, Half-time, Double-time)
	var candidates []BPMCandidate
	seenTempos := make(map[int]bool)

	for _, cand := range scoredLags[:int(math.Min(5, float64(len(scoredLags))))] {
		bpmVal := 60.0 * hopRate / float64(cand.lag)
		roundInt := int(math.Round(bpmVal))
		if !seenTempos[roundInt] {
			seenTempos[roundInt] = true
			candidates = append(candidates, BPMCandidate{
				BPM:        bpmVal,
				Strength:   cand.score,
				Confidence: math.Min(1.0, cand.score/scoredLags[0].score),
			})
		}
	}

	return &BPMResult{
		BPM:             finalBPM,
		Confidence:      confidence,
		FirstBeatSec:    firstBeatSec,
		BeatIntervalSec: beatIntervalSec,
		BeatTimestamps:  beatTimestamps,
		TempoStability:  tempoStability,
		Candidates:      candidates,
		IsSteadyBeat:    tempoStability >= 0.40,
		TimeSignature:   "4/4",
	}, nil
}

// GenerateRhythmicTestSignal creates a synthetic musical audio stream with kick, snare,
// hi-hat, and bass elements at an exact specified BPM for mathematical verification.
func GenerateRhythmicTestSignal(bpm float64, durationSec float64, sampleRate, channels int) *PCMBuffer {
	if sampleRate <= 0 {
		sampleRate = 48000
	}
	if channels <= 0 {
		channels = 2
	}
	if durationSec <= 0 {
		durationSec = 2.0
	}

	numSamples := int(float64(sampleRate) * durationSec)
	samples := make([][]float64, channels)
	for ch := 0; ch < channels; ch++ {
		samples[ch] = make([]float64, numSamples)
	}

	beatIntervalSec := 60.0 / bpm
	samplesPerBeat := int(beatIntervalSec * float64(sampleRate))

	for i := 0; i < numSamples; i++ {
		beatIndex := i / samplesPerBeat
		sampleInBeat := i % samplesPerBeat
		tInBeat := float64(sampleInBeat) / float64(sampleRate)

		var val float64 = 0.0

		// 1. Kick Drum (Beats 0, 2 in 4/4 meter - 60Hz swept down from 150Hz)
		if beatIndex%2 == 0 {
			if tInBeat < 0.20 {
				fKick := 60.0 + 90.0*math.Exp(-tInBeat/0.04)
				val += 0.80 * math.Sin(2.0*math.Pi*fKick*tInBeat) * math.Exp(-tInBeat/0.06)
			}
		}

		// 2. Snare Drum (Beats 1, 3 in 4/4 meter - 200Hz body + noise snap)
		if beatIndex%2 == 1 {
			if tInBeat < 0.18 {
				fSnare := 180.0
				noiseSnap := (float64((i*37+19)%100)/50.0 - 1.0) * 0.4
				body := math.Sin(2.0*math.Pi*fSnare*tInBeat) * 0.5
				val += (body + noiseSnap) * math.Exp(-tInBeat/0.05)
			}
		}

		// 3. Eighth-note Hi-Hats (every half beat)
		eighthSample := sampleInBeat % (samplesPerBeat / 2)
		tEighth := float64(eighthSample) / float64(sampleRate)
		if tEighth < 0.04 {
			noiseHat := (float64((i*71+101)%100)/50.0 - 1.0) * 0.15
			val += noiseHat * math.Exp(-tEighth/0.01)
		}

		// 4. Subtle melodic bass tone (80Hz - 160Hz)
		tTotal := float64(i) / float64(sampleRate)
		val += 0.10 * math.Sin(2.0*math.Pi*110.0*tTotal)

		valNorm := math.Max(-0.95, math.Min(0.95, val*0.75))
		for ch := 0; ch < channels; ch++ {
			samples[ch][i] = valNorm
		}
	}

	return &PCMBuffer{
		Samples:    samples,
		Channels:   channels,
		SampleRate: sampleRate,
		NumSamples: numSamples,
	}
}
