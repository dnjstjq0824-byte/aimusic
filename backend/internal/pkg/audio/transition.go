package audio

import (
	"errors"
	"fmt"
	"math"
)

// TransitionConfig configures boundary micro-fades and gapless crossfade transitions.
type TransitionConfig struct {
	DurationSeconds          float64   // Standard transition duration in seconds (default: 0.3s)
	Curve                    FadeCurve // Fade curve profile (default: FadeCurveEqualPower)
	AlignZeroCrossing        bool      // Search and align boundary to exact zero-crossing
	ZeroCrossingSearchRadius int       // Search radius in samples around boundary (default: 128)
}

// DefaultTransitionConfig returns standard 0.3s micro-fade transition configuration.
func DefaultTransitionConfig() TransitionConfig {
	return TransitionConfig{
		DurationSeconds:          0.3,
		Curve:                    FadeCurveEqualPower,
		AlignZeroCrossing:        true,
		ZeroCrossingSearchRadius: 128,
	}
}

// TransitionResult holds output buffer and telemetry metrics for track boundary transition.
type TransitionResult struct {
	ProcessedBuffer               *PCMBuffer
	HeadFadeSamples               int
	TailFadeSamples               int
	HeadZeroCrossingSample        int
	TailZeroCrossingSample        int
	InitialHeadDiscontinuityDBFS  float64
	FinalHeadEdgeAmplitude        float64
	InitialTailDiscontinuityDBFS  float64
	FinalTailEdgeAmplitude        float64
	ClickTransientSuppressed      bool
}

// CrossfadeResult holds blended gapless audio buffer and energy preservation metrics.
type CrossfadeResult struct {
	BlendedBuffer                 *PCMBuffer
	CrossfadeSamples              int
	CrossfadeDurationSeconds      float64
	MidpointEnergyVarianceDB      float64 // Deviation from 0 dB energy conservation
	EnergyConservationCompliant   bool
}

// TransitionProcessor provides 0.3s track boundary smoothing and gapless crossfades.
type TransitionProcessor struct {
	config TransitionConfig
}

// NewTransitionProcessor creates a new transition processor.
func NewTransitionProcessor(config TransitionConfig) *TransitionProcessor {
	if config.DurationSeconds <= 0.001 {
		config.DurationSeconds = 0.3
	}
	if config.ZeroCrossingSearchRadius <= 4 {
		config.ZeroCrossingSearchRadius = 128
	}
	if config.Curve == "" {
		config.Curve = FadeCurveEqualPower
	}
	return &TransitionProcessor{config: config}
}

// ProcessTrackBoundaries applies exact 0.3s zero-crossing aligned fade-in and fade-out.
func (tp *TransitionProcessor) ProcessTrackBoundaries(pcm *PCMBuffer) (*TransitionResult, error) {
	if pcm == nil {
		return nil, ErrNilPCMBuffer
	}
	if len(pcm.Samples) == 0 || pcm.Channels == 0 {
		return nil, ErrEmptyPCMBuffer
	}

	numChannels := pcm.Channels
	sr := pcm.SampleRate
	numSamples := pcm.NumSamples

	fadeSamples := int(math.Round(float64(sr) * tp.config.DurationSeconds))
	if fadeSamples < 2 {
		fadeSamples = 2
	}

	// If audio is shorter than combined head and tail fade, limit to half length
	if fadeSamples*2 > numSamples {
		fadeSamples = numSamples / 2
		if fadeSamples < 2 {
			fadeSamples = 2
		}
	}

	// Measure initial edge discontinuities before transition
	initHeadAmp := 0.0
	initTailAmp := 0.0
	for ch := 0; ch < numChannels; ch++ {
		h := math.Abs(pcm.Samples[ch][0])
		t := math.Abs(pcm.Samples[ch][numSamples-1])
		if h > initHeadAmp {
			initHeadAmp = h
		}
		if t > initTailAmp {
			initTailAmp = t
		}
	}
	headDiscDBFS := -120.0
	if initHeadAmp > 1e-6 {
		headDiscDBFS = 20.0 * math.Log10(initHeadAmp)
	}
	tailDiscDBFS := -120.0
	if initTailAmp > 1e-6 {
		tailDiscDBFS = 20.0 * math.Log10(initTailAmp)
	}

	// Clone sample buffers
	outSamples := make([][]float64, numChannels)
	for ch := 0; ch < numChannels; ch++ {
		outSamples[ch] = make([]float64, numSamples)
		copy(outSamples[ch], pcm.Samples[ch])
	}

	// Mono guide for zero-crossing detection
	guideMono := downmixToMono(outSamples)

	// 1. Head Fade-in with zero-crossing alignment
	headStart := 0
	if tp.config.AlignZeroCrossing && len(guideMono) > 10 {
		headStart = FindZeroCrossing(guideMono, 0, tp.config.ZeroCrossingSearchRadius)
		if headStart < 0 || headStart >= fadeSamples {
			headStart = 0
		}
	}

	effectiveHeadFade := fadeSamples - headStart
	if effectiveHeadFade > 1 {
		for i := 0; i < effectiveHeadFade; i++ {
			progress := float64(i) / float64(effectiveHeadFade-1)
			weight := CalculateCurveWeight(progress, tp.config.Curve, true)
			idx := headStart + i
			for ch := 0; ch < numChannels; ch++ {
				outSamples[ch][idx] *= weight
			}
		}
		// Zero out any samples before head zero-crossing
		for i := 0; i < headStart; i++ {
			for ch := 0; ch < numChannels; ch++ {
				outSamples[ch][i] = 0.0
			}
		}
	}

	// 2. Tail Fade-out with zero-crossing alignment
	tailTarget := numSamples - 1
	if tp.config.AlignZeroCrossing && len(guideMono) > 10 {
		zc := FindZeroCrossing(guideMono, numSamples-1, tp.config.ZeroCrossingSearchRadius)
		if zc > numSamples-fadeSamples && zc < numSamples {
			tailTarget = zc
		}
	}

	tailFadeStart := tailTarget - fadeSamples
	if tailFadeStart < 0 {
		tailFadeStart = 0
	}
	effectiveTailFade := tailTarget - tailFadeStart + 1
	if effectiveTailFade > 1 {
		for i := 0; i < effectiveTailFade; i++ {
			progress := float64(i) / float64(effectiveTailFade-1)
			weight := CalculateCurveWeight(progress, tp.config.Curve, false)
			idx := tailFadeStart + i
			if idx < numSamples {
				for ch := 0; ch < numChannels; ch++ {
					outSamples[ch][idx] *= weight
				}
			}
		}
		// Zero out any samples after tail zero-crossing
		for i := tailTarget + 1; i < numSamples; i++ {
			for ch := 0; ch < numChannels; ch++ {
				outSamples[ch][i] = 0.0
			}
		}
	}

	// Measure final edge amplitudes (must be ~0.0)
	finalHeadAmp := 0.0
	finalTailAmp := 0.0
	for ch := 0; ch < numChannels; ch++ {
		h := math.Abs(outSamples[ch][0])
		t := math.Abs(outSamples[ch][numSamples-1])
		if h > finalHeadAmp {
			finalHeadAmp = h
		}
		if t > finalTailAmp {
			finalTailAmp = t
		}
	}

	outPCM := &PCMBuffer{
		SampleRate: sr,
		Channels:   numChannels,
		NumSamples: numSamples,
		Samples:    outSamples,
	}

	return &TransitionResult{
		ProcessedBuffer:              outPCM,
		HeadFadeSamples:              effectiveHeadFade,
		TailFadeSamples:              effectiveTailFade,
		HeadZeroCrossingSample:       headStart,
		TailZeroCrossingSample:       tailTarget,
		InitialHeadDiscontinuityDBFS: headDiscDBFS,
		FinalHeadEdgeAmplitude:       finalHeadAmp,
		InitialTailDiscontinuityDBFS: tailDiscDBFS,
		FinalTailEdgeAmplitude:       finalTailAmp,
		ClickTransientSuppressed:     finalHeadAmp < 1e-4 && finalTailAmp < 1e-4,
	}, nil
}

// Crossfade blends trackA and trackB seamlessly with equal-power energy preservation.
func (tp *TransitionProcessor) Crossfade(trackA, trackB *PCMBuffer) (*CrossfadeResult, error) {
	if trackA == nil || trackB == nil {
		return nil, errors.New("input tracks cannot be nil")
	}
	if trackA.SampleRate != trackB.SampleRate {
		return nil, fmt.Errorf("sample rate mismatch: A=%d, B=%d", trackA.SampleRate, trackB.SampleRate)
	}
	if trackA.Channels != trackB.Channels {
		return nil, fmt.Errorf("channel count mismatch: A=%d, B=%d", trackA.Channels, trackB.Channels)
	}

	sr := trackA.SampleRate
	channels := trackA.Channels
	crossSamples := int(math.Round(float64(sr) * tp.config.DurationSeconds))

	if crossSamples > trackA.NumSamples || crossSamples > trackB.NumSamples {
		return nil, fmt.Errorf("crossfade duration %d samples exceeds track length (A=%d, B=%d)",
			crossSamples, trackA.NumSamples, trackB.NumSamples)
	}

	totalOutputSamples := trackA.NumSamples + trackB.NumSamples - crossSamples
	outSamples := make([][]float64, channels)
	for ch := 0; ch < channels; ch++ {
		outSamples[ch] = make([]float64, totalOutputSamples)
	}

	// 1. Copy Track A body (before crossfade region)
	overlapStart := trackA.NumSamples - crossSamples
	for ch := 0; ch < channels; ch++ {
		copy(outSamples[ch][:overlapStart], trackA.Samples[ch][:overlapStart])
	}

	// 2. Blend overlap region: Track A fade-out + Track B fade-in
	midpointEnergyVariance := 0.0
	for i := 0; i < crossSamples; i++ {
		progress := float64(i) / float64(crossSamples-1)
		wA := CalculateCurveWeight(progress, tp.config.Curve, false) // Fade-out weight
		wB := CalculateCurveWeight(progress, tp.config.Curve, true)  // Fade-in weight

		// Energy sum: wA^2 + wB^2
		sumEnergy := wA*wA + wB*wB
		diffDB := math.Abs(10.0 * math.Log10(sumEnergy))
		if diffDB > midpointEnergyVariance {
			midpointEnergyVariance = diffDB
		}

		outIdx := overlapStart + i
		for ch := 0; ch < channels; ch++ {
			sampleA := trackA.Samples[ch][overlapStart+i]
			sampleB := trackB.Samples[ch][i]
			outSamples[ch][outIdx] = sampleA*wA + sampleB*wB
		}
	}

	// 3. Copy Track B tail (after crossfade region)
	bRemainderStart := crossSamples
	outRemainderStart := trackA.NumSamples
	for ch := 0; ch < channels; ch++ {
		copy(outSamples[ch][outRemainderStart:], trackB.Samples[ch][bRemainderStart:])
	}

	blendedPCM := &PCMBuffer{
		SampleRate: sr,
		Channels:   channels,
		NumSamples: totalOutputSamples,
		Samples:    outSamples,
	}

	return &CrossfadeResult{
		BlendedBuffer:               blendedPCM,
		CrossfadeSamples:            crossSamples,
		CrossfadeDurationSeconds:    float64(crossSamples) / float64(sr),
		MidpointEnergyVarianceDB:    midpointEnergyVariance,
		EnergyConservationCompliant: midpointEnergyVariance < 0.1,
	}, nil
}
