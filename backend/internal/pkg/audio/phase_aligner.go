package audio

import (
	"fmt"
	"math"
)

// PhaseAlignerConfig configures inter-track phase correlation and delay alignment.
type PhaseAlignerConfig struct {
	MaxDelayMs         float64 // Maximum search window for delay in milliseconds (e.g. 20.0 ms)
	SubSampleAccuracy  bool    // Enable parabolic interpolation for fractional-sample precision
	AutoPolarityInvert bool    // Detect and flip 180-degree out-of-phase polarity
	FilterOrder        int     // FIR filter tap count for fractional delay (e.g. 15 or 31)
}

// DefaultPhaseAlignerConfig returns high-fidelity default configuration.
func DefaultPhaseAlignerConfig() PhaseAlignerConfig {
	return PhaseAlignerConfig{
		MaxDelayMs:         20.0,
		SubSampleAccuracy:  true,
		AutoPolarityInvert: true,
		FilterOrder:        15,
	}
}

// PhaseAlignmentResult stores alignment parameters and metrics between two tracks.
type PhaseAlignmentResult struct {
	EstimatedDelaySamples float64    // Fractional delay in samples (+: target lags ref, -: target leads ref)
	EstimatedDelayMs      float64    // Fractional delay in milliseconds
	PolarityInverted      bool       // True if target track had 180-degree polarity inverted
	InitialCorrelation    float64    // Phase correlation coefficient before alignment (-1.0 to +1.0)
	AlignedCorrelation    float64    // Phase correlation coefficient after alignment
	CoherenceGainDB       float64    // Coherent summation gain improvement over unaligned summation in dB
	AlignedTrack          *PCMBuffer // Time-shifted and polarity-corrected target buffer
}

// MultiTrackAlignmentResult contains alignment metrics for multi-track stems.
type MultiTrackAlignmentResult struct {
	ReferenceTrackIndex  int
	TrackResults         []*PhaseAlignmentResult
	SummedUnaligned      *PCMBuffer
	SummedAligned        *PCMBuffer
	TotalCoherenceGainDB float64
}

// PhaseAligner provides sub-sample precision cross-correlation and phase alignment.
type PhaseAligner struct {
	config PhaseAlignerConfig
}

// NewPhaseAligner creates a phase aligner with given configuration.
func NewPhaseAligner(config PhaseAlignerConfig) *PhaseAligner {
	if config.MaxDelayMs <= 0.1 {
		config.MaxDelayMs = 0.1
	}
	if config.FilterOrder < 3 {
		config.FilterOrder = 3
	}
	if config.FilterOrder%2 == 0 {
		config.FilterOrder++ // Ensure odd filter order for symmetric linear phase
	}
	return &PhaseAligner{config: config}
}

// ComputePhaseCorrelation computes Pearson correlation coefficient between two mono signals.
// Result lies in range [-1.0, +1.0].
func ComputePhaseCorrelation(sigA, sigB []float64) float64 {
	n := len(sigA)
	if n == 0 || len(sigB) == 0 {
		return 0.0
	}
	if len(sigB) < n {
		n = len(sigB)
	}

	var sumA, sumB, sumAA, sumBB, sumAB float64
	for i := 0; i < n; i++ {
		a := sigA[i]
		b := sigB[i]
		sumA += a
		sumB += b
		sumAA += a * a
		sumBB += b * b
		sumAB += a * b
	}

	fn := float64(n)
	varA := sumAA - (sumA * sumA / fn)
	varB := sumBB - (sumB * sumB / fn)
	covar := sumAB - (sumA * sumB / fn)

	if varA <= 1e-12 || varB <= 1e-12 {
		return 0.0
	}

	corr := covar / math.Sqrt(varA*varB)
	if corr > 1.0 {
		return 1.0
	}
	if corr < -1.0 {
		return -1.0
	}
	return corr
}

// computeOverlappingCorrelation calculates phase correlation strictly over the non-padded, overlapping active audio window.
func computeOverlappingCorrelation(ref, aligned []float64, delay float64) float64 {
	n := len(ref)
	if len(aligned) < n {
		n = len(aligned)
	}
	intLag := int(math.Round(delay))
	start := 0
	end := n

	if intLag > 0 {
		end = n - intLag
	} else if intLag < 0 {
		start = -intLag
	}

	if start >= end || (end-start) < 64 {
		return ComputePhaseCorrelation(ref, aligned)
	}

	return ComputePhaseCorrelation(ref[start:end], aligned[start:end])
}

// AlignTracks aligns targetTrack to refTrack by estimating time delay and polarity.
func (pa *PhaseAligner) AlignTracks(refTrack, targetTrack *PCMBuffer) (*PhaseAlignmentResult, error) {
	if refTrack == nil || targetTrack == nil {
		return nil, ErrNilPCMBuffer
	}
	if len(refTrack.Samples) == 0 || len(targetTrack.Samples) == 0 {
		return nil, ErrEmptyPCMBuffer
	}
	if refTrack.SampleRate != targetTrack.SampleRate {
		return nil, fmt.Errorf("sample rate mismatch: ref=%d, target=%d", refTrack.SampleRate, targetTrack.SampleRate)
	}

	sr := float64(refTrack.SampleRate)
	maxLag := int(math.Ceil(pa.config.MaxDelayMs * sr / 1000.0))

	// Downmix channels to mono for delay estimation
	refMono := downmixToMono(refTrack.Samples)
	targetMono := downmixToMono(targetTrack.Samples)

	initialCorrelation := ComputePhaseCorrelation(refMono, targetMono)

	// Cross-correlation over [-maxLag, +maxLag]
	bestLag, peakCorrVal := pa.findCrossCorrelationPeak(refMono, targetMono, maxLag)

	// Sub-sample parabolic interpolation around peak
	fractionalLag := float64(bestLag)
	if pa.config.SubSampleAccuracy && bestLag > -maxLag && bestLag < maxLag {
		cMinus := pa.computeLagCorrelation(refMono, targetMono, bestLag-1)
		cCenter := peakCorrVal
		cPlus := pa.computeLagCorrelation(refMono, targetMono, bestLag+1)

		denom := 2.0 * (cMinus - 2.0*cCenter + cPlus)
		if math.Abs(denom) > 1e-12 {
			delta := (cMinus - cPlus) / denom
			if math.Abs(delta) < 1.0 {
				fractionalLag = float64(bestLag) + delta
			}
		}
	}

	// Polarity check: if correlation at peak is negative, invert polarity
	invertPolarity := false
	if pa.config.AutoPolarityInvert && peakCorrVal < 0.0 {
		invertPolarity = true
	}

	// Shift target track by -fractionalLag to align with reference
	alignedSamples := make([][]float64, targetTrack.Channels)
	for ch := 0; ch < targetTrack.Channels; ch++ {
		shifted := pa.applyFractionalDelay(targetTrack.Samples[ch], -fractionalLag)
		if invertPolarity {
			for i := range shifted {
				shifted[i] = -shifted[i]
			}
		}
		alignedSamples[ch] = shifted
	}

	alignedMono := downmixToMono(alignedSamples)
	alignedCorrelation := computeOverlappingCorrelation(refMono, alignedMono, fractionalLag)

	// Calculate coherence summation gain (cancellation prevention)
	unalignedSumEnergy := computeSumEnergy(refMono, targetMono)
	alignedSumEnergy := computeSumEnergy(refMono, alignedMono)

	coherenceGainDB := 0.0
	if unalignedSumEnergy > 1e-12 && alignedSumEnergy > 1e-12 {
		coherenceGainDB = 10.0 * math.Log10(alignedSumEnergy/unalignedSumEnergy)
	}

	alignedPCM := &PCMBuffer{
		SampleRate: targetTrack.SampleRate,
		Channels:   targetTrack.Channels,
		NumSamples: targetTrack.NumSamples,
		Samples:    alignedSamples,
	}

	delayMs := (fractionalLag / sr) * 1000.0

	return &PhaseAlignmentResult{
		EstimatedDelaySamples: fractionalLag,
		EstimatedDelayMs:      delayMs,
		PolarityInverted:      invertPolarity,
		InitialCorrelation:    initialCorrelation,
		AlignedCorrelation:    alignedCorrelation,
		CoherenceGainDB:       coherenceGainDB,
		AlignedTrack:          alignedPCM,
	}, nil
}

// AlignMultiTrack aligns multiple stems against a selected reference track (e.g. Kick or Lead Vocal).
func (pa *PhaseAligner) AlignMultiTrack(tracks []*PCMBuffer, refIndex int) (*MultiTrackAlignmentResult, error) {
	if len(tracks) < 2 {
		return nil, fmt.Errorf("at least two tracks required for multi-track alignment")
	}
	if refIndex < 0 || refIndex >= len(tracks) {
		return nil, fmt.Errorf("reference index out of bounds: %d", refIndex)
	}

	refTrack := tracks[refIndex]
	sr := refTrack.SampleRate
	numChannels := refTrack.Channels
	numSamples := refTrack.NumSamples

	alignedTracks := make([]*PCMBuffer, len(tracks))
	results := make([]*PhaseAlignmentResult, len(tracks))

	for i, trk := range tracks {
		if trk.SampleRate != sr || trk.Channels != numChannels || trk.NumSamples != numSamples {
			return nil, fmt.Errorf("track %d dimension mismatch with reference", i)
		}
		if i == refIndex {
			alignedTracks[i] = trk
			results[i] = &PhaseAlignmentResult{
				EstimatedDelaySamples: 0.0,
				EstimatedDelayMs:      0.0,
				PolarityInverted:      false,
				InitialCorrelation:    1.0,
				AlignedCorrelation:    1.0,
				CoherenceGainDB:       0.0,
				AlignedTrack:          trk,
			}
			continue
		}

		res, err := pa.AlignTracks(refTrack, trk)
		if err != nil {
			return nil, fmt.Errorf("failed to align track %d: %w", i, err)
		}
		results[i] = res
		alignedTracks[i] = res.AlignedTrack
	}

	// Mix unaligned sum vs aligned sum
	summedUnaligned := sumTracks(tracks)
	summedAligned := sumTracks(alignedTracks)

	monoUnaligned := downmixToMono(summedUnaligned.Samples)
	monoAligned := downmixToMono(summedAligned.Samples)

	energyUnaligned := computeEnergy(monoUnaligned)
	energyAligned := computeEnergy(monoAligned)

	totalGainDB := 0.0
	if energyUnaligned > 1e-12 && energyAligned > 1e-12 {
		totalGainDB = 10.0 * math.Log10(energyAligned/energyUnaligned)
	}

	return &MultiTrackAlignmentResult{
		ReferenceTrackIndex:  refIndex,
		TrackResults:         results,
		SummedUnaligned:      summedUnaligned,
		SummedAligned:        summedAligned,
		TotalCoherenceGainDB: totalGainDB,
	}, nil
}

// findCrossCorrelationPeak finds lag k in [-maxLag, maxLag] that maximizes absolute correlation.
func (pa *PhaseAligner) findCrossCorrelationPeak(ref, target []float64, maxLag int) (int, float64) {
	bestLag := 0
	maxAbsVal := -1.0
	bestVal := 0.0

	for lag := -maxLag; lag <= maxLag; lag++ {
		c := pa.computeLagCorrelation(ref, target, lag)
		absC := math.Abs(c)
		if absC > maxAbsVal {
			maxAbsVal = absC
			bestLag = lag
			bestVal = c
		}
	}
	return bestLag, bestVal
}

// computeLagCorrelation computes normalized cross-correlation for a specific integer lag.
func (pa *PhaseAligner) computeLagCorrelation(ref, target []float64, lag int) float64 {
	n := len(ref)
	if len(target) < n {
		n = len(target)
	}

	var sumAB, sumA2, sumB2 float64
	count := 0

	for i := 0; i < n; i++ {
		targetIdx := i + lag
		if targetIdx >= 0 && targetIdx < len(target) {
			a := ref[i]
			b := target[targetIdx]
			sumAB += a * b
			sumA2 += a * a
			sumB2 += b * b
			count++
		}
	}

	if count < 10 || sumA2 <= 1e-12 || sumB2 <= 1e-12 {
		return 0.0
	}
	return sumAB / math.Sqrt(sumA2*sumB2)
}

// applyFractionalDelay shifts an input signal by continuous delay (in samples) using windowed Sinc FIR interpolation.
func (pa *PhaseAligner) applyFractionalDelay(input []float64, delay float64) []float64 {
	n := len(input)
	output := make([]float64, n)

	intDelay := int(math.Floor(delay))
	fracDelay := delay - float64(intDelay) // [0.0, 1.0)

	order := pa.config.FilterOrder
	halfOrder := order / 2

	// Pre-compute windowed sinc filter weights for fractional part
	filter := make([]float64, order)
	var filterSum float64
	for m := 0; m < order; m++ {
		t := float64(m-halfOrder) - fracDelay
		// Sinc function
		var s float64
		if math.Abs(t) < 1e-9 {
			s = 1.0
		} else {
			piT := math.Pi * t
			s = math.Sin(piT) / piT
		}
		// Blackman-Harris window for > 90 dB stopband suppression
		alpha0 := 0.35875
		alpha1 := 0.48829
		alpha2 := 0.14128
		alpha3 := 0.01168
		phi := 2.0 * math.Pi * float64(m) / float64(order-1)
		w := alpha0 - alpha1*math.Cos(phi) + alpha2*math.Cos(2.0*phi) - alpha3*math.Cos(3.0*phi)

		weight := s * w
		filter[m] = weight
		filterSum += weight
	}

	// Normalize filter gain to 0 dB at DC
	if math.Abs(filterSum) > 1e-12 {
		for m := range filter {
			filter[m] /= filterSum
		}
	}

	// Convolve input with fractional delay filter
	for i := 0; i < n; i++ {
		var sum float64
		for m := 0; m < order; m++ {
			inIdx := i - intDelay - (m - halfOrder)
			if inIdx >= 0 && inIdx < n {
				sum += input[inIdx] * filter[m]
			}
		}
		output[i] = sum
	}

	return output
}

// Helper utilities for mono downmixing and energy calculations

func downmixToMono(samples [][]float64) []float64 {
	if len(samples) == 0 {
		return nil
	}
	numChannels := len(samples)
	numSamples := len(samples[0])
	mono := make([]float64, numSamples)

	invCh := 1.0 / float64(numChannels)
	for s := 0; s < numSamples; s++ {
		var sum float64
		for ch := 0; ch < numChannels; ch++ {
			sum += samples[ch][s]
		}
		mono[s] = sum * invCh
	}
	return mono
}

func computeEnergy(sig []float64) float64 {
	var e float64
	for _, v := range sig {
		e += v * v
	}
	return e
}

func computeSumEnergy(sigA, sigB []float64) float64 {
	n := len(sigA)
	if len(sigB) < n {
		n = len(sigB)
	}
	var e float64
	for i := 0; i < n; i++ {
		sum := sigA[i] + sigB[i]
		e += sum * sum
	}
	return e
}

func sumTracks(tracks []*PCMBuffer) *PCMBuffer {
	if len(tracks) == 0 {
		return nil
	}
	ref := tracks[0]
	outSamples := make([][]float64, ref.Channels)
	for ch := 0; ch < ref.Channels; ch++ {
		outSamples[ch] = make([]float64, ref.NumSamples)
	}

	for _, trk := range tracks {
		for ch := 0; ch < ref.Channels; ch++ {
			for s := 0; s < ref.NumSamples; s++ {
				outSamples[ch][s] += trk.Samples[ch][s]
			}
		}
	}

	return &PCMBuffer{
		SampleRate: ref.SampleRate,
		Channels:   ref.Channels,
		NumSamples: ref.NumSamples,
		Samples:    outSamples,
	}
}

// String provides a summary representation of alignment result.
func (r *PhaseAlignmentResult) String() string {
	return fmt.Sprintf("Phase Alignment: Delay=%.3f samples (%.3f ms), Inverted=%v, Corr=[%.3f -> %.3f], Gain=+%.2f dB",
		r.EstimatedDelaySamples,
		r.EstimatedDelayMs,
		r.PolarityInverted,
		r.InitialCorrelation,
		r.AlignedCorrelation,
		r.CoherenceGainDB,
	)
}
