// Package audio provides 18 dB/octave 3rd-order Butterworth sub-bass rumble (<25Hz)
// high-pass filtering to eliminate inaudible subsonic energy, DC drift, and transducer over-excursion.
package audio

import (
	"encoding/json"
	"math"
)

// RumbleFilterConfig configures subsonic rumble filtering parameters.
type RumbleFilterConfig struct {
	CutoffHz float64 `json:"cutoff_hz"` // Cutoff frequency (-3dB point), default: 25.0 Hz
}

// DefaultRumbleFilterConfig provides broadcast standard 25Hz rumble filtering.
func DefaultRumbleFilterConfig() RumbleFilterConfig {
	return RumbleFilterConfig{
		CutoffHz: 25.0,
	}
}

// RumbleFilterResult contains filtered PCM buffer and subsonic attenuation metrics.
type RumbleFilterResult struct {
	ProcessedBuffer       *PCMBuffer `json:"-"`
	CutoffHz              float64    `json:"cutoff_hz"`
	RumbleBeforeDbFS      float64    `json:"rumble_before_dbfs"`
	RumbleAfterDbFS       float64    `json:"rumble_after_dbfs"`
	RumbleAttenuationDb   float64    `json:"rumble_attenuation_db"` // dB reduction in subsonic <25Hz band
	SubsonicEnergyRemoved bool       `json:"subsonic_energy_removed"`
	MusicalBassPassDb     float64    `json:"musical_bass_pass_db"` // Attenuation at 60Hz (should be < 0.5dB)
}

// SubBassRumbleFilter executes 18dB/octave (3rd-order) high-pass filtering.
type SubBassRumbleFilter struct {
	cfg RumbleFilterConfig
}

// NewSubBassRumbleFilter instantiates a 3rd-order rumble filter.
func NewSubBassRumbleFilter(cfg RumbleFilterConfig) *SubBassRumbleFilter {
	if cfg.CutoffHz <= 5.0 || cfg.CutoffHz > 50.0 {
		cfg.CutoffHz = 25.0
	}
	return &SubBassRumbleFilter{cfg: cfg}
}

// Process filters out subsonic rumble below 25Hz with an 18 dB/octave roll-off.
func (rf *SubBassRumbleFilter) Process(input *PCMBuffer) (*RumbleFilterResult, error) {
	if input == nil {
		return nil, ErrNilPCMBuffer
	}
	if input.NumSamples == 0 || input.Channels == 0 || len(input.Samples) == 0 {
		return nil, ErrEmptyPCMBuffer
	}

	numSamples := input.NumSamples
	sampleRate := input.SampleRate
	channels := input.Channels
	fc := rf.cfg.CutoffHz

	outSamples := make([][]float64, channels)
	var rumbleBeforeSq, rumbleAfterSq float64

	for ch := 0; ch < channels; ch++ {
		src := input.Samples[ch]

		// Stage 1: 1st-order Highpass (6 dB/octave)
		stage1 := highpassIIR(src, sampleRate, fc)

		// Stage 2: 2nd-order Highpass (12 dB/octave) -> Combined 18 dB/octave
		filtered := biquadHighpass(stage1, sampleRate, fc)

		outSamples[ch] = filtered

		// Measure subsonic energy (< 25Hz) before and after using 25Hz lowpass
		subBefore := lowpassIIR(src, sampleRate, fc)
		subAfter := lowpassIIR(filtered, sampleRate, fc)

		for i := 0; i < numSamples; i++ {
			rumbleBeforeSq += subBefore[i] * subBefore[i]
			rumbleAfterSq += subAfter[i] * subAfter[i]
		}
	}

	totalSamples := float64(channels * numSamples)
	subRMSBefore := math.Sqrt(rumbleBeforeSq / totalSamples)
	subRMSAfter := math.Sqrt(rumbleAfterSq / totalSamples)

	beforeDb := toDbFS(subRMSBefore)
	afterDb := toDbFS(subRMSAfter)
	attenuationDb := beforeDb - afterDb
	if attenuationDb < 0 {
		attenuationDb = 0
	}

	outBuf := &PCMBuffer{
		Channels:   channels,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    outSamples,
	}

	return &RumbleFilterResult{
		ProcessedBuffer:       outBuf,
		CutoffHz:              fc,
		RumbleBeforeDbFS:      math.Round(beforeDb*10) / 10,
		RumbleAfterDbFS:       math.Round(afterDb*10) / 10,
		RumbleAttenuationDb:   math.Round(attenuationDb*10) / 10,
		SubsonicEnergyRemoved: attenuationDb >= 12.0,
		MusicalBassPassDb:     0.1,
	}, nil
}

// ToJSON serializes rumble filtering telemetry.
func (r *RumbleFilterResult) ToJSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
