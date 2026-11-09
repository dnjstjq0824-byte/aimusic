// Package audio provides ITU-R BS.775 and ITU-R BS.1196 compliant multichannel
// audio downmix matrices (7.1, 5.1, Quad to Stereo Lo/Ro and Lt/Rt),
// phase cancellation detection, and clipping prevention headroom normalization.
package audio

import (
	"encoding/json"
	"fmt"
	"math"
)

// ChannelLayout defines standard multichannel spatial speaker configurations.
type ChannelLayout string

const (
	LayoutMono         ChannelLayout = "1.0_MONO"
	LayoutStereo       ChannelLayout = "2.0_STEREO"
	LayoutQuad         ChannelLayout = "4.0_QUAD"        // L, R, Ls, Rs
	Layout5Point1      ChannelLayout = "5.1_SURROUND"    // L, R, C, LFE, Ls, Rs
	Layout7Point1      ChannelLayout = "7.1_SURROUND"    // L, R, C, LFE, Ls, Rs, Rls, Rrs
	LayoutUnsupported  ChannelLayout = "UNSUPPORTED"
)

// DownmixMode defines the downmix matrix algorithm.
type DownmixMode string

const (
	// DownmixLoRo is standard Left-only / Right-only downmix per ITU-R BS.775 (no phase shifts).
	DownmixLoRo DownmixMode = "LO_RO"

	// DownmixLtRt is Left-total / Right-total matrix downmix (Surrounds are phase shifted for matrix decoders).
	DownmixLtRt DownmixMode = "LT_RT"

	// DownmixMono mixes all channels down into a single mono channel.
	DownmixMono DownmixMode = "MONO"
)

// DownmixConfig configures the downmixing parameters.
type DownmixConfig struct {
	Mode               DownmixMode `json:"mode"`
	CenterAttenuation  float64     `json:"center_attenuation"`  // Linear factor, default: 0.7071 (-3 dB)
	SurroundAttenuation float64    `json:"surround_attenuation"`// Linear factor, default: 0.7071 (-3 dB)
	IncludeLFE         bool        `json:"include_lfe"`         // Standard ITU-R omits LFE, set true to mix at -10dB
	LFEGain            float64     `json:"lfe_gain"`            // Default: 0.3162 (-10 dB)
	NormalizeHeadroom  bool        `json:"normalize_headroom"`  // True: scales output by peak sum to guarantee 0dBFS max
}

// DefaultDownmixConfig returns ITU-R BS.775 compliant Lo/Ro downmix settings.
func DefaultDownmixConfig() DownmixConfig {
	return DownmixConfig{
		Mode:                DownmixLoRo,
		CenterAttenuation:   math.Sqrt(0.5), // 0.70710678 (-3.01 dB)
		SurroundAttenuation: math.Sqrt(0.5), // 0.70710678 (-3.01 dB)
		IncludeLFE:          false,
		LFEGain:             math.Pow(10.0, -10.0/20.0), // -10 dB
		NormalizeHeadroom:   true,
	}
}

// DownmixResult contains the downmixed stereo samples and transformation telemetry.
type DownmixResult struct {
	SourceLayout      ChannelLayout `json:"source_layout"`
	TargetLayout      ChannelLayout `json:"target_layout"`
	Mode              DownmixMode   `json:"mode"`
	SampleRate        int           `json:"sample_rate"`
	NumSamples        int           `json:"num_samples"`
	PeakBeforeDbFS    float64       `json:"peak_before_dbfs"`
	PeakAfterDbFS     float64       `json:"peak_after_dbfs"`
	AppliedGainDb     float64       `json:"applied_gain_db"`
	PhaseCorrelation  float64       `json:"phase_correlation"` // -1.0 (anti-phase) to +1.0 (in-phase mono)
	StereoSamples     [][]float64   `json:"-"`                 // [2][numSamples]
}

// MultichannelDownmixer executes multichannel matrix downmixing.
type MultichannelDownmixer struct {
	cfg DownmixConfig
}

// NewMultichannelDownmixer creates a new downmixer.
func NewMultichannelDownmixer(cfg DownmixConfig) *MultichannelDownmixer {
	if cfg.CenterAttenuation <= 0 {
		cfg.CenterAttenuation = math.Sqrt(0.5)
	}
	if cfg.SurroundAttenuation <= 0 {
		cfg.SurroundAttenuation = math.Sqrt(0.5)
	}
	if cfg.LFEGain <= 0 {
		cfg.LFEGain = math.Pow(10.0, -10.0/20.0)
	}
	if cfg.Mode == "" {
		cfg.Mode = DownmixLoRo
	}
	return &MultichannelDownmixer{cfg: cfg}
}

// IdentifyLayout detects the channel configuration from channel count.
func IdentifyLayout(channels int) ChannelLayout {
	switch channels {
	case 1:
		return LayoutMono
	case 2:
		return LayoutStereo
	case 4:
		return LayoutQuad
	case 6:
		return Layout5Point1
	case 8:
		return Layout7Point1
	default:
		return LayoutUnsupported
	}
}

// Downmix converts multi-channel audio buffer to stereo (or mono) based on ITU-R standards.
func (d *MultichannelDownmixer) Downmix(input *PCMBuffer) (*DownmixResult, error) {
	if input == nil {
		return nil, ErrNilPCMBuffer
	}
	if len(input.Samples) == 0 || input.NumSamples == 0 {
		return nil, ErrEmptyPCMBuffer
	}

	layout := IdentifyLayout(input.Channels)
	if layout == LayoutUnsupported {
		return nil, fmt.Errorf("unsupported channel count %d for downmixing", input.Channels)
	}

	numSamples := input.NumSamples
	sampleRate := input.SampleRate

	// Already Stereo: pass-through or compute telemetry
	if layout == LayoutStereo && d.cfg.Mode != DownmixMono {
		outL := make([]float64, numSamples)
		outR := make([]float64, numSamples)
		copy(outL, input.Samples[0])
		copy(outR, input.Samples[1])

		corr := computeCorrelation(outL, outR)
		peak := math.Max(calcPeak(outL), calcPeak(outR))

		return &DownmixResult{
			SourceLayout:     LayoutStereo,
			TargetLayout:     LayoutStereo,
			Mode:             d.cfg.Mode,
			SampleRate:       sampleRate,
			NumSamples:       numSamples,
			PeakBeforeDbFS:   toDbFS(peak),
			PeakAfterDbFS:    toDbFS(peak),
			AppliedGainDb:    0.0,
			PhaseCorrelation: corr,
			StereoSamples:    [][]float64{outL, outR},
		}, nil
	}

	// Prepare output buffers
	left := make([]float64, numSamples)
	right := make([]float64, numSamples)

	cAtt := d.cfg.CenterAttenuation
	sAtt := d.cfg.SurroundAttenuation
	lfeGain := d.cfg.LFEGain
	incLFE := d.cfg.IncludeLFE

	var peakInput float64
	for ch := 0; ch < input.Channels; ch++ {
		p := calcPeak(input.Samples[ch])
		if p > peakInput {
			peakInput = p
		}
	}

	switch layout {
	case LayoutMono:
		// 1.0 -> Duplicate to Left and Right
		copy(left, input.Samples[0])
		copy(right, input.Samples[0])

	case LayoutQuad:
		// 4.0: L=0, R=1, Ls=2, Rs=3
		sL, sR, sLs, sRs := input.Samples[0], input.Samples[1], input.Samples[2], input.Samples[3]
		for i := 0; i < numSamples; i++ {
			if d.cfg.Mode == DownmixLtRt {
				// Surround 90 degree phase approximation / quadrature
				left[i] = sL[i] - sAtt*sLs[i] - sAtt*sRs[i]
				right[i] = sR[i] + sAtt*sLs[i] + sAtt*sRs[i]
			} else {
				// Lo/Ro
				left[i] = sL[i] + sAtt*sLs[i]
				right[i] = sR[i] + sAtt*sRs[i]
			}
		}

	case Layout5Point1:
		// 5.1 ITU-R channel ordering: 0:L, 1:R, 2:C, 3:LFE, 4:Ls, 5:Rs
		sL, sR, sC, sLFE, sLs, sRs := input.Samples[0], input.Samples[1], input.Samples[2], input.Samples[3], input.Samples[4], input.Samples[5]
		for i := 0; i < numSamples; i++ {
			c := sC[i] * cAtt
			lfe := 0.0
			if incLFE {
				lfe = sLFE[i] * lfeGain
			}

			if d.cfg.Mode == DownmixLtRt {
				// Lt = L + 0.7071*C - 0.7071*(Ls + Rs)
				// Rt = R + 0.7071*C + 0.7071*(Ls + Rs)
				surrSum := sAtt * (sLs[i] + sRs[i])
				left[i] = sL[i] + c - surrSum + lfe
				right[i] = sR[i] + c + surrSum + lfe
			} else {
				// Lo/Ro standard:
				// Lo = L + 0.7071*C + 0.7071*Ls (+ LFE)
				// Ro = R + 0.7071*C + 0.7071*Rs (+ LFE)
				left[i] = sL[i] + c + sAtt*sLs[i] + lfe
				right[i] = sR[i] + c + sAtt*sRs[i] + lfe
			}
		}

	case Layout7Point1:
		// 7.1 channel ordering: 0:L, 1:R, 2:C, 3:LFE, 4:Ls, 5:Rs, 6:Rls, 7:Rrs (Back surrounds)
		sL, sR, sC, sLFE := input.Samples[0], input.Samples[1], input.Samples[2], input.Samples[3]
		sLs, sRs, sRls, sRrs := input.Samples[4], input.Samples[5], input.Samples[6], input.Samples[7]
		backAtt := sAtt * 0.7071 // -6 dB for secondary back surrounds

		for i := 0; i < numSamples; i++ {
			c := sC[i] * cAtt
			lfe := 0.0
			if incLFE {
				lfe = sLFE[i] * lfeGain
			}

			left[i] = sL[i] + c + sAtt*sLs[i] + backAtt*sRls[i] + lfe
			right[i] = sR[i] + c + sAtt*sRs[i] + backAtt*sRrs[i] + lfe
		}
	}

	// Downmix to Mono if requested
	if d.cfg.Mode == DownmixMono {
		for i := 0; i < numSamples; i++ {
			m := 0.5 * (left[i] + right[i])
			left[i] = m
			right[i] = m
		}
	}

	// Normalization & Headroom scaling to prevent clipping
	peakOutput := math.Max(calcPeak(left), calcPeak(right))
	appliedGain := 1.0

	if d.cfg.NormalizeHeadroom && peakOutput > 0.999 {
		// Target ceiling 0.98 (-0.17 dBFS)
		targetCeiling := 0.98
		appliedGain = targetCeiling / peakOutput
		for i := 0; i < numSamples; i++ {
			left[i] *= appliedGain
			right[i] *= appliedGain
		}
		peakOutput *= appliedGain
	}

	correlation := computeCorrelation(left, right)

	targetLayout := LayoutStereo
	if d.cfg.Mode == DownmixMono {
		targetLayout = LayoutMono
	}

	appliedGainDb := 0.0
	if appliedGain > 0 {
		appliedGainDb = 20.0 * math.Log10(appliedGain)
	}

	return &DownmixResult{
		SourceLayout:     layout,
		TargetLayout:     targetLayout,
		Mode:             d.cfg.Mode,
		SampleRate:       sampleRate,
		NumSamples:       numSamples,
		PeakBeforeDbFS:   toDbFS(peakInput),
		PeakAfterDbFS:    toDbFS(peakOutput),
		AppliedGainDb:    math.Round(appliedGainDb*100) / 100,
		PhaseCorrelation: math.Round(correlation*1000) / 1000,
		StereoSamples:    [][]float64{left, right},
	}, nil
}

// computeCorrelation calculates Pearson correlation between left and right channels (-1 to +1).
func computeCorrelation(l, r []float64) float64 {
	n := len(l)
	if n == 0 {
		return 1.0
	}

	var sumL, sumR, sumL2, sumR2, sumLR float64
	for i := 0; i < n; i++ {
		vl := l[i]
		vr := r[i]
		sumL += vl
		sumR += vr
		sumL2 += vl * vl
		sumR2 += vr * vr
		sumLR += vl * vr
	}

	fn := float64(n)
	varL := sumL2 - (sumL*sumL)/fn
	varR := sumR2 - (sumR*sumR)/fn
	covar := sumLR - (sumL*sumR)/fn

	denom := math.Sqrt(varL * varR)
	if denom < 1e-12 {
		return 1.0
	}
	corr := covar / denom
	if corr > 1.0 {
		return 1.0
	}
	if corr < -1.0 {
		return -1.0
	}
	return corr
}

func calcPeak(samples []float64) float64 {
	var maxVal float64
	for _, v := range samples {
		abs := math.Abs(v)
		if abs > maxVal {
			maxVal = abs
		}
	}
	return maxVal
}

func toDbFS(amp float64) float64 {
	if amp <= 1e-6 {
		return -120.0
	}
	return 20.0 * math.Log10(amp)
}

// ToJSON serializes result telemetry.
func (r *DownmixResult) ToJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
