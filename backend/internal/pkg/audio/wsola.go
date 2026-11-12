package audio

import (
	"fmt"
	"math"
)

// WSOLAConfig holds configuration parameters for the WSOLA time-stretching engine.
type WSOLAConfig struct {
	WindowSizeSamples int // Overlap-add window size in samples (e.g. 1024 or 2048)
	SynthesisHop      int // Synthesis hop size (typically WindowSizeSamples / 2)
	SearchWindow      int // Maximum search range for cross-correlation peak (e.g. SynthesisHop / 2)
	ResampleQuality   int // Sinc interpolation lobe count for pitch shifting (e.g. 8 or 16)
}

// DefaultWSOLAConfig returns production-standard configuration.
func DefaultWSOLAConfig(sampleRate int) WSOLAConfig {
	// ~25ms window at current sample rate
	winSize := 1024
	if sampleRate >= 88200 {
		winSize = 2048
	} else if sampleRate <= 24000 {
		winSize = 512
	}
	hop := winSize / 2
	search := hop / 2

	return WSOLAConfig{
		WindowSizeSamples: winSize,
		SynthesisHop:      hop,
		SearchWindow:      search,
		ResampleQuality:   8,
	}
}

// WSOLAEngine provides time-scale modification (time-stretching) and pitch shifting.
type WSOLAEngine struct {
	config WSOLAConfig
	window []float64 // Pre-computed symmetric Hann window
}

// NewWSOLAEngine initializes a new WSOLA engine.
func NewWSOLAEngine(config WSOLAConfig) *WSOLAEngine {
	if config.WindowSizeSamples < 64 {
		config.WindowSizeSamples = 64
	}
	if config.SynthesisHop <= 0 {
		config.SynthesisHop = config.WindowSizeSamples / 2
	}
	if config.SearchWindow <= 0 {
		config.SearchWindow = config.SynthesisHop / 2
	}
	if config.ResampleQuality < 4 {
		config.ResampleQuality = 4
	}

	// Compute Hann window: w[n] = 0.5 * (1 - cos(2*pi*n / (W-1)))
	w := make([]float64, config.WindowSizeSamples)
	N := float64(config.WindowSizeSamples - 1)
	for i := 0; i < config.WindowSizeSamples; i++ {
		w[i] = 0.5 * (1.0 - math.Cos(2.0*math.Pi*float64(i)/N))
	}

	return &WSOLAEngine{
		config: config,
		window: w,
	}
}

// TimeStretch scales audio playback speed by speedRatio without altering pitch.
// speedRatio = 0.5: half speed (2.0x duration)
// speedRatio = 1.0: original speed (1.0x duration)
// speedRatio = 2.0: double speed (0.5x duration)
func (e *WSOLAEngine) TimeStretch(pcm *PCMBuffer, speedRatio float64) (*PCMBuffer, error) {
	if pcm == nil {
		return nil, ErrNilPCMBuffer
	}
	if len(pcm.Samples) == 0 || pcm.Channels == 0 {
		return nil, ErrEmptyPCMBuffer
	}
	if speedRatio < 0.25 || speedRatio > 4.0 {
		return nil, fmt.Errorf("speedRatio %.2f out of supported range [0.25, 4.0]", speedRatio)
	}

	// If speedRatio is virtually 1.0, return identical copy
	if math.Abs(speedRatio-1.0) < 1e-4 {
		outSamples := make([][]float64, pcm.Channels)
		for ch := 0; ch < pcm.Channels; ch++ {
			outSamples[ch] = make([]float64, pcm.NumSamples)
			copy(outSamples[ch], pcm.Samples[ch])
		}
		return &PCMBuffer{
			SampleRate: pcm.SampleRate,
			Channels:   pcm.Channels,
			NumSamples: pcm.NumSamples,
			Samples:    outSamples,
		}, nil
	}

	winSize := e.config.WindowSizeSamples
	hs := e.config.SynthesisHop
	deltaMax := e.config.SearchWindow

	numChannels := pcm.Channels
	numInSamples := pcm.NumSamples

	if numInSamples < winSize+deltaMax {
		// Signal too short for full WSOLA, return copy
		outSamples := make([][]float64, numChannels)
		for ch := 0; ch < numChannels; ch++ {
			outSamples[ch] = make([]float64, numInSamples)
			copy(outSamples[ch], pcm.Samples[ch])
		}
		return &PCMBuffer{
			SampleRate: pcm.SampleRate,
			Channels:   numChannels,
			NumSamples: numInSamples,
			Samples:    outSamples,
		}, nil
	}

	// Mono guide signal for phase-locked multi-channel correlation
	guideMono := downmixToMono(pcm.Samples)

	// Target output samples: Duration_out = Duration_in / speedRatio
	targetOutSamples := int(math.Round(float64(numInSamples) / speedRatio))
	if targetOutSamples < winSize {
		targetOutSamples = winSize
	}

	// In WSOLA: Ha = Hs * speedRatio
	// If speedRatio = 0.5 (slow down, 2x duration), Ha = Hs * 0.5 (advances half as fast in input)
	// If speedRatio = 2.0 (speed up, 0.5x duration), Ha = Hs * 2.0 (advances twice as fast in input)
	ha := float64(hs) * speedRatio

	allocatedLen := targetOutSamples + winSize*2
	outputBuffers := make([][]float64, numChannels)
	windowWeights := make([]float64, allocatedLen)
	for ch := 0; ch < numChannels; ch++ {
		outputBuffers[ch] = make([]float64, allocatedLen)
	}

	// Step 0: Initial frame at tau_0 = 0, synthesis pos S_0 = 0
	tauPrev := 0
	for i := 0; i < winSize; i++ {
		wVal := e.window[i]
		windowWeights[i] += wVal
		for ch := 0; ch < numChannels; ch++ {
			outputBuffers[ch][i] += wVal * pcm.Samples[ch][i]
		}
	}

	overlapLen := winSize - hs
	synthPos := hs
	frameIdx := 1

	for {
		if synthPos >= targetOutSamples {
			break
		}

		// Nominal analysis center for current synthesis frame
		nominalTau := float64(frameIdx) * ha
		nominalTauInt := int(math.Round(nominalTau))

		if nominalTauInt+winSize > numInSamples {
			break
		}

		// Find delta in [-deltaMax, deltaMax] that maximizes waveform similarity
		bestDelta := 0
		maxCorr := -math.MaxFloat64

		refStart := tauPrev + hs

		for d := -deltaMax; d <= deltaMax; d++ {
			candStart := nominalTauInt + d
			if candStart < 0 || candStart+overlapLen >= numInSamples || refStart+overlapLen >= numInSamples {
				continue
			}

			var sumProd, sumCandSq float64
			for k := 0; k < overlapLen; k++ {
				r := guideMono[refStart+k]
				c := guideMono[candStart+k]
				sumProd += r * c
				sumCandSq += c * c
			}

			var metric float64
			if sumCandSq > 1e-12 {
				metric = sumProd / math.Sqrt(sumCandSq)
			} else {
				metric = sumProd
			}

			if metric > maxCorr {
				maxCorr = metric
				bestDelta = d
			}
		}

		tauCurrent := nominalTauInt + bestDelta
		if tauCurrent < 0 {
			tauCurrent = 0
		}
		if tauCurrent+winSize > numInSamples {
			tauCurrent = numInSamples - winSize
		}

		// Overlap-add windowed segment across all channels
		for i := 0; i < winSize; i++ {
			wVal := e.window[i]
			outIdx := synthPos + i
			if outIdx < allocatedLen {
				windowWeights[outIdx] += wVal
				for ch := 0; ch < numChannels; ch++ {
					outputBuffers[ch][outIdx] += wVal * pcm.Samples[ch][tauCurrent+i]
				}
			}
		}

		tauPrev = tauCurrent
		synthPos += hs
		frameIdx++
	}

	finalLen := targetOutSamples
	if synthPos < targetOutSamples {
		finalLen = synthPos + overlapLen
		if finalLen > targetOutSamples {
			finalLen = targetOutSamples
		}
	}

	// Normalize overlap-add amplitude using accumulated window weights
	finalSamples := make([][]float64, numChannels)
	for ch := 0; ch < numChannels; ch++ {
		finalSamples[ch] = make([]float64, finalLen)
		for i := 0; i < finalLen; i++ {
			wTotal := windowWeights[i]
			if wTotal > 1e-4 {
				finalSamples[ch][i] = outputBuffers[ch][i] / wTotal
			} else {
				finalSamples[ch][i] = outputBuffers[ch][i]
			}
		}
	}

	return &PCMBuffer{
		SampleRate: pcm.SampleRate,
		Channels:   numChannels,
		NumSamples: finalLen,
		Samples:    finalSamples,
	}, nil
}

// PitchShift shifts audio pitch by semitones without changing tempo or duration.
// semitones = +12.0: 1 octave up
// semitones = -12.0: 1 octave down
// semitones = +2.0: 1 whole tone up
func (e *WSOLAEngine) PitchShift(pcm *PCMBuffer, semitones float64) (*PCMBuffer, error) {
	if pcm == nil {
		return nil, ErrNilPCMBuffer
	}
	if len(pcm.Samples) == 0 || pcm.Channels == 0 {
		return nil, ErrEmptyPCMBuffer
	}
	if semitones < -24.0 || semitones > 24.0 {
		return nil, fmt.Errorf("semitones %.2f out of supported range [-24.0, 24.0]", semitones)
	}

	if math.Abs(semitones) < 1e-4 {
		// 0 semitone shift, return clone
		outSamples := make([][]float64, pcm.Channels)
		for ch := 0; ch < pcm.Channels; ch++ {
			outSamples[ch] = make([]float64, pcm.NumSamples)
			copy(outSamples[ch], pcm.Samples[ch])
		}
		return &PCMBuffer{
			SampleRate: pcm.SampleRate,
			Channels:   pcm.Channels,
			NumSamples: pcm.NumSamples,
			Samples:    outSamples,
		}, nil
	}

	// Pitch ratio: P = 2^(semitones / 12)
	pitchRatio := math.Pow(2.0, semitones/12.0)

	// Step 1: Resample audio by factor pitchRatio
	// This alters pitch by P and changes length by 1/P
	resampled, err := e.fractionalSincResample(pcm, pitchRatio)
	if err != nil {
		return nil, fmt.Errorf("resampling for pitch shift failed: %w", err)
	}

	// Step 2: Time-stretch to restore duration
	// Resampled length is inLen / pitchRatio. To restore to inLen:
	// speedRatio must be 1.0 / pitchRatio
	stretched, err := e.TimeStretch(resampled, 1.0/pitchRatio)
	if err != nil {
		return nil, fmt.Errorf("time-stretch for pitch shift failed: %w", err)
	}

	// Trim or pad slightly to match exact original length
	origLen := pcm.NumSamples
	adjustedSamples := make([][]float64, pcm.Channels)
	for ch := 0; ch < pcm.Channels; ch++ {
		adjustedSamples[ch] = make([]float64, origLen)
		copyLen := origLen
		if len(stretched.Samples[ch]) < copyLen {
			copyLen = len(stretched.Samples[ch])
		}
		copy(adjustedSamples[ch], stretched.Samples[ch][:copyLen])
	}

	return &PCMBuffer{
		SampleRate: pcm.SampleRate,
		Channels:   pcm.Channels,
		NumSamples: origLen,
		Samples:    adjustedSamples,
	}, nil
}

// Process simultaneously modifies both playback speed (time-stretch) and pitch.
func (e *WSOLAEngine) Process(pcm *PCMBuffer, speedRatio float64, semitones float64) (*PCMBuffer, error) {
	current := pcm
	var err error

	if math.Abs(semitones) >= 1e-4 {
		current, err = e.PitchShift(current, semitones)
		if err != nil {
			return nil, err
		}
	}

	if math.Abs(speedRatio-1.0) >= 1e-4 {
		current, err = e.TimeStretch(current, speedRatio)
		if err != nil {
			return nil, err
		}
	}

	return current, nil
}

// fractionalSincResample resamples a PCM buffer by continuous factor ratio using Blackman-Harris windowed Sinc.
func (e *WSOLAEngine) fractionalSincResample(pcm *PCMBuffer, ratio float64) (*PCMBuffer, error) {
	if ratio <= 0.0 {
		return nil, fmt.Errorf("invalid resample ratio: %.4f", ratio)
	}

	inLen := pcm.NumSamples
	outLen := int(math.Round(float64(inLen) / ratio))
	if outLen < 1 {
		outLen = 1
	}

	lobes := e.config.ResampleQuality
	outSamples := make([][]float64, pcm.Channels)

	for ch := 0; ch < pcm.Channels; ch++ {
		inCh := pcm.Samples[ch]
		outCh := make([]float64, outLen)

		for m := 0; m < outLen; m++ {
			// Continuous position in input
			t := float64(m) * ratio
			centerIdx := int(math.Floor(t))
			frac := t - float64(centerIdx)

			var sum float64
			var weightSum float64

			for k := -lobes; k <= lobes; k++ {
				srcIdx := centerIdx + k
				if srcIdx >= 0 && srcIdx < inLen {
					dist := float64(k) - frac
					// Sinc function
					var s float64
					if math.Abs(dist) < 1e-9 {
						s = 1.0
					} else {
						piD := math.Pi * dist
						s = math.Sin(piD) / piD
					}

					// Blackman-Harris window
					normDist := dist / float64(lobes)
					if math.Abs(normDist) <= 1.0 {
						phi := math.Pi * (normDist + 1.0)
						w := 0.35875 - 0.48829*math.Cos(phi) + 0.14128*math.Cos(2.0*phi) - 0.01168*math.Cos(3.0*phi)
						weight := s * w
						sum += inCh[srcIdx] * weight
						weightSum += weight
					}
				}
			}

			if math.Abs(weightSum) > 1e-12 {
				outCh[m] = sum / weightSum
			} else {
				outCh[m] = sum
			}
		}
		outSamples[ch] = outCh
	}

	return &PCMBuffer{
		SampleRate: pcm.SampleRate,
		Channels:   pcm.Channels,
		NumSamples: outLen,
		Samples:    outSamples,
	}, nil
}
