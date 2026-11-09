// Package audio provides professional audio quality assurance, defect detection,
// silence analysis, digital clipping detection, DC offset correction, and phase correlation inspection.
package audio

import (
	"fmt"
	"math"
)

// DefectType enumerates standard audio flaws.
type DefectType string

const (
	DefectDeadSilence      DefectType = "DEAD_SILENCE"
	DefectExcessiveLeadIn  DefectType = "EXCESSIVE_LEAD_IN_SILENCE"
	DefectExcessiveLeadOut DefectType = "EXCESSIVE_LEAD_OUT_SILENCE"
	DefectMidTrackDropout  DefectType = "MID_TRACK_DROPOUT"
	DefectSevereClipping   DefectType = "SEVERE_CLIPPING"
	DefectMinorClipping    DefectType = "MINOR_CLIPPING"
	DefectDCOffset         DefectType = "DC_OFFSET"
	DefectPhaseInversion   DefectType = "PHASE_INVERSION"
)

// DefectSeverity indicates the impact on audio quality.
type DefectSeverity string

const (
	SeverityPass    DefectSeverity = "PASS"
	SeverityWarning DefectSeverity = "WARNING"
	SeverityError   DefectSeverity = "ERROR"
)

// DefectItem describes a single detected defect with location and measurement.
type DefectItem struct {
	Type        DefectType     `json:"type"`
	Severity    DefectSeverity `json:"severity"`
	Channel     int            `json:"channel"`
	StartSec    float64        `json:"start_sec"`
	DurationSec float64        `json:"duration_sec"`
	Value       float64        `json:"value"`
	Description string         `json:"description"`
}

// AudioQCReport summarizes the quality assurance inspection results.
type AudioQCReport struct {
	Passed            bool         `json:"passed"`
	TotalDefects      int          `json:"total_defects"`
	LeadInSilenceSec  float64      `json:"lead_in_silence_sec"`
	LeadOutSilenceSec float64      `json:"lead_out_silence_sec"`
	MaxDropoutSec     float64      `json:"max_dropout_sec"`
	ClippedSamples    int64        `json:"clipped_samples"`
	ClippedEvents     int          `json:"clipped_events"`
	ClippedPercent    float64      `json:"clipped_percent"`
	DCOffset          []float64    `json:"dc_offset"`
	PhaseCorrelation  float64      `json:"phase_correlation"`
	Defects           []DefectItem `json:"defects"`
}

// InspectorConfig contains configurable thresholds for audio defect inspection.
type InspectorConfig struct {
	SilenceThresholdDBFS float64 // RMS threshold below which audio is silent (default -60.0 dBFS)
	MaxLeadInSec         float64 // Maximum allowable silence at start (default 3.0 sec)
	MaxLeadOutSec        float64 // Maximum allowable silence at end (default 3.0 sec)
	MaxDropoutSec        float64 // Maximum allowable mid-track silence gap (default 1.5 sec)
	DCOffsetThreshold    float64 // Maximum allowable DC bias (default 0.002, ~-54 dBFS)
	ClipConsecutiveLimit int     // Consecutive full-scale samples to count as clipped peak (default 3)
	MaxClipEventsAllowed int     // Allowed minor clipping events before warning/error (default 0)
}

// InspectorOption configures an AudioInspector.
type InspectorOption func(*InspectorConfig)

// WithSilenceThresholdDBFS sets the RMS silence boundary.
func WithSilenceThresholdDBFS(dbfs float64) InspectorOption {
	return func(c *InspectorConfig) {
		c.SilenceThresholdDBFS = dbfs
	}
}

// WithMaxLeadInSec sets maximum allowed lead-in silence.
func WithMaxLeadInSec(sec float64) InspectorOption {
	return func(c *InspectorConfig) {
		c.MaxLeadInSec = sec
	}
}

// WithMaxLeadOutSec sets maximum allowed lead-out silence.
func WithMaxLeadOutSec(sec float64) InspectorOption {
	return func(c *InspectorConfig) {
		c.MaxLeadOutSec = sec
	}
}

// WithMaxDropoutSec sets maximum allowed mid-track dropout.
func WithMaxDropoutSec(sec float64) InspectorOption {
	return func(c *InspectorConfig) {
		c.MaxDropoutSec = sec
	}
}

// WithDCOffsetThreshold sets the DC bias detection threshold.
func WithDCOffsetThreshold(th float64) InspectorOption {
	return func(c *InspectorConfig) {
		c.DCOffsetThreshold = th
	}
}

// WithClipConsecutiveLimit sets consecutive sample limit for clipping.
func WithClipConsecutiveLimit(limit int) InspectorOption {
	return func(c *InspectorConfig) {
		if limit > 0 {
			c.ClipConsecutiveLimit = limit
		}
	}
}

// AudioInspector inspects PCM audio buffers for defects.
type AudioInspector struct {
	cfg InspectorConfig
}

// NewAudioInspector creates an inspector with default or custom options.
func NewAudioInspector(opts ...InspectorOption) *AudioInspector {
	cfg := InspectorConfig{
		SilenceThresholdDBFS: -60.0,
		MaxLeadInSec:         3.0,
		MaxLeadOutSec:        3.0,
		MaxDropoutSec:        1.5,
		DCOffsetThreshold:    0.002,
		ClipConsecutiveLimit: 3,
		MaxClipEventsAllowed: 0,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &AudioInspector{cfg: cfg}
}

// Inspect performs comprehensive defect detection on a PCMBuffer.
func (ai *AudioInspector) Inspect(buf *PCMBuffer) (*AudioQCReport, error) {
	if buf == nil {
		return nil, ErrNilFingerprintBuffer
	}
	if buf.Channels <= 0 || buf.NumSamples <= 0 || len(buf.Samples) == 0 {
		return nil, ErrEmptyAudioBuffer
	}

	channels := buf.Channels
	numSamples := buf.NumSamples
	sampleRate := buf.SampleRate

	var defects []DefectItem
	passed := true

	// 1. DC Offset Analysis
	dcOffsets := make([]float64, channels)
	for ch := 0; ch < channels; ch++ {
		var sum float64
		for n := 0; n < numSamples; n++ {
			sum += buf.Samples[ch][n]
		}
		dc := sum / float64(numSamples)
		dcOffsets[ch] = dc

		if math.Abs(dc) > ai.cfg.DCOffsetThreshold {
			passed = false
			sev := SeverityWarning
			if math.Abs(dc) > ai.cfg.DCOffsetThreshold*5.0 {
				sev = SeverityError
			}
			defects = append(defects, DefectItem{
				Type:        DefectDCOffset,
				Severity:    sev,
				Channel:     ch,
				StartSec:    0.0,
				DurationSec: float64(numSamples) / float64(sampleRate),
				Value:       dc,
				Description: fmt.Sprintf("Channel %d DC offset bias of %.5f exceeds threshold %.5f",
					ch, dc, ai.cfg.DCOffsetThreshold),
			})
		}
	}

	// 2. Stereo Phase Correlation Analysis
	phaseCorrelation := 1.0
	if channels == 2 {
		var sumL2, sumR2, sumLR float64
		for n := 0; n < numSamples; n++ {
			l := buf.Samples[0][n]
			r := buf.Samples[1][n]
			sumL2 += l * l
			sumR2 += r * r
			sumLR += l * r
		}
		denom := math.Sqrt(sumL2 * sumR2)
		if denom > 1e-12 {
			phaseCorrelation = sumLR / denom
		}
		if phaseCorrelation < -0.85 {
			passed = false
			defects = append(defects, DefectItem{
				Type:        DefectPhaseInversion,
				Severity:    SeverityError,
				Channel:     0,
				StartSec:    0.0,
				DurationSec: float64(numSamples) / float64(sampleRate),
				Value:       phaseCorrelation,
				Description: fmt.Sprintf("Stereo phase correlation of %.3f indicates inverted polarity (mono cancellation)", phaseCorrelation),
			})
		}
	}

	// 3. Digital Clipping Analysis
	var totalClippedSamples int64
	var totalClipEvents int
	clipThreshold := 0.9999

	for ch := 0; ch < channels; ch++ {
		consecutive := 0
		for n := 0; n < numSamples; n++ {
			if math.Abs(buf.Samples[ch][n]) >= clipThreshold {
				consecutive++
				totalClippedSamples++
			} else {
				if consecutive >= ai.cfg.ClipConsecutiveLimit {
					totalClipEvents++
				}
				consecutive = 0
			}
		}
		if consecutive >= ai.cfg.ClipConsecutiveLimit {
			totalClipEvents++
		}
	}

	totalSampleSlots := int64(numSamples) * int64(channels)
	clipPercent := (float64(totalClippedSamples) / float64(totalSampleSlots)) * 100.0

	if totalClipEvents > ai.cfg.MaxClipEventsAllowed {
		sev := SeverityWarning
		defType := DefectMinorClipping
		if totalClipEvents > 10 || clipPercent > 0.05 {
			sev = SeverityError
			defType = DefectSevereClipping
			passed = false
		}
		defects = append(defects, DefectItem{
			Type:        defType,
			Severity:    sev,
			Channel:     -1,
			StartSec:    0.0,
			DurationSec: float64(numSamples) / float64(sampleRate),
			Value:       clipPercent,
			Description: fmt.Sprintf("%d clipping events (%d clipped samples, %.3f%% of audio)",
				totalClipEvents, totalClippedSamples, clipPercent),
		})
	}

	// 4. Silence & Dropout Analysis (50ms block window)
	blockSize := int(math.Round(0.050 * float64(sampleRate)))
	if blockSize < 1 {
		blockSize = 1
	}
	numBlocks := numSamples / blockSize
	silenceLinThresh := math.Pow(10.0, ai.cfg.SilenceThresholdDBFS/20.0)

	isBlockSilent := make([]bool, numBlocks)
	for b := 0; b < numBlocks; b++ {
		start := b * blockSize
		var sumSq float64
		for ch := 0; ch < channels; ch++ {
			for k := 0; k < blockSize; k++ {
				v := buf.Samples[ch][start+k]
				sumSq += v * v
			}
		}
		rms := math.Sqrt(sumSq / float64(blockSize*channels))
		isBlockSilent[b] = (rms < silenceLinThresh)
	}

	// Check for total dead silence
	allSilent := true
	for _, s := range isBlockSilent {
		if !s {
			allSilent = false
			break
		}
	}

	var leadInSec, leadOutSec, maxDropoutSec float64

	if allSilent {
		passed = false
		durSec := float64(numSamples) / float64(sampleRate)
		leadInSec = durSec
		leadOutSec = durSec
		defects = append(defects, DefectItem{
			Type:        DefectDeadSilence,
			Severity:    SeverityError,
			Channel:     -1,
			StartSec:    0.0,
			DurationSec: durSec,
			Value:       durSec,
			Description: "Audio stream is entirely silent",
		})
	} else {
		// Lead-in silence
		firstActive := 0
		for firstActive < numBlocks && isBlockSilent[firstActive] {
			firstActive++
		}
		leadInSec = float64(firstActive*blockSize) / float64(sampleRate)

		if leadInSec > ai.cfg.MaxLeadInSec {
			passed = false
			defects = append(defects, DefectItem{
				Type:        DefectExcessiveLeadIn,
				Severity:    SeverityWarning,
				Channel:     -1,
				StartSec:    0.0,
				DurationSec: leadInSec,
				Value:       leadInSec,
				Description: fmt.Sprintf("Excessive lead-in silence of %.2f sec (exceeds %.2f sec limit)",
					leadInSec, ai.cfg.MaxLeadInSec),
			})
		}

		// Lead-out silence
		lastActive := numBlocks - 1
		for lastActive >= 0 && isBlockSilent[lastActive] {
			lastActive--
		}
		leadOutSec = float64((numBlocks-1-lastActive)*blockSize) / float64(sampleRate)

		if leadOutSec > ai.cfg.MaxLeadOutSec {
			passed = false
			startSec := float64((lastActive+1)*blockSize) / float64(sampleRate)
			defects = append(defects, DefectItem{
				Type:        DefectExcessiveLeadOut,
				Severity:    SeverityWarning,
				Channel:     -1,
				StartSec:    startSec,
				DurationSec: leadOutSec,
				Value:       leadOutSec,
				Description: fmt.Sprintf("Excessive lead-out silence of %.2f sec (exceeds %.2f sec limit)",
					leadOutSec, ai.cfg.MaxLeadOutSec),
			})
		}

		// Mid-track dropouts between firstActive and lastActive
		currSilentCount := 0
		currSilentStart := 0
		for b := firstActive; b <= lastActive; b++ {
			if isBlockSilent[b] {
				if currSilentCount == 0 {
					currSilentStart = b
				}
				currSilentCount++
			} else {
				if currSilentCount > 0 {
					gapSec := float64(currSilentCount*blockSize) / float64(sampleRate)
					if gapSec > maxDropoutSec {
						maxDropoutSec = gapSec
					}
					if gapSec > ai.cfg.MaxDropoutSec {
						passed = false
						gapStartSec := float64(currSilentStart*blockSize) / float64(sampleRate)
						defects = append(defects, DefectItem{
							Type:        DefectMidTrackDropout,
							Severity:    SeverityError,
							Channel:     -1,
							StartSec:    gapStartSec,
							DurationSec: gapSec,
							Value:       gapSec,
							Description: fmt.Sprintf("Mid-track audio dropout of %.2f sec at %.2f sec (exceeds %.2f sec limit)",
								gapSec, gapStartSec, ai.cfg.MaxDropoutSec),
						})
					}
					currSilentCount = 0
				}
			}
		}
	}

	return &AudioQCReport{
		Passed:            passed,
		TotalDefects:      len(defects),
		LeadInSilenceSec:  leadInSec,
		LeadOutSilenceSec: leadOutSec,
		MaxDropoutSec:     maxDropoutSec,
		ClippedSamples:    totalClippedSamples,
		ClippedEvents:     totalClipEvents,
		ClippedPercent:    clipPercent,
		DCOffset:          dcOffsets,
		PhaseCorrelation:  phaseCorrelation,
		Defects:           defects,
	}, nil
}

// RemoveDCOffset removes DC bias from all channels, returning a clean normalized buffer.
func RemoveDCOffset(buf *PCMBuffer) *PCMBuffer {
	if buf == nil {
		return nil
	}
	channels := buf.Channels
	numSamples := buf.NumSamples
	outSamples := make([][]float64, channels)

	for ch := 0; ch < channels; ch++ {
		var sum float64
		for n := 0; n < numSamples; n++ {
			sum += buf.Samples[ch][n]
		}
		dc := sum / float64(numSamples)

		outSamples[ch] = make([]float64, numSamples)
		for n := 0; n < numSamples; n++ {
			outSamples[ch][n] = buf.Samples[ch][n] - dc
		}
	}

	return &PCMBuffer{
		Channels:   channels,
		SampleRate: buf.SampleRate,
		NumSamples: numSamples,
		Samples:    outSamples,
	}
}

// TrimSilence removes leading and trailing silence exceeding thresholdDBFS.
func TrimSilence(buf *PCMBuffer, thresholdDBFS float64) *PCMBuffer {
	if buf == nil || buf.NumSamples == 0 {
		return buf
	}

	sampleRate := buf.SampleRate
	channels := buf.Channels
	numSamples := buf.NumSamples

	blockSize := int(math.Round(0.010 * float64(sampleRate))) // 10ms blocks
	if blockSize < 1 {
		blockSize = 1
	}
	numBlocks := numSamples / blockSize
	silenceLinThresh := math.Pow(10.0, thresholdDBFS/20.0)

	firstActive := 0
	for firstActive < numBlocks {
		start := firstActive * blockSize
		var sumSq float64
		for ch := 0; ch < channels; ch++ {
			for k := 0; k < blockSize; k++ {
				v := buf.Samples[ch][start+k]
				sumSq += v * v
			}
		}
		if math.Sqrt(sumSq/float64(blockSize*channels)) >= silenceLinThresh {
			break
		}
		firstActive++
	}

	if firstActive >= numBlocks {
		// Entire track is silent: keep minimum 1 block
		firstActive = 0
	}

	lastActive := numBlocks - 1
	for lastActive > firstActive {
		start := lastActive * blockSize
		var sumSq float64
		for ch := 0; ch < channels; ch++ {
			for k := 0; k < blockSize; k++ {
				v := buf.Samples[ch][start+k]
				sumSq += v * v
			}
		}
		if math.Sqrt(sumSq/float64(blockSize*channels)) >= silenceLinThresh {
			break
		}
		lastActive--
	}

	startSample := firstActive * blockSize
	endSample := (lastActive + 1) * blockSize
	if endSample > numSamples {
		endSample = numSamples
	}
	if startSample >= endSample {
		startSample = 0
		endSample = numSamples
	}

	newLen := endSample - startSample
	outSamples := make([][]float64, channels)
	for ch := 0; ch < channels; ch++ {
		outSamples[ch] = make([]float64, newLen)
		copy(outSamples[ch], buf.Samples[ch][startSample:endSample])
	}

	return &PCMBuffer{
		Channels:   channels,
		SampleRate: sampleRate,
		NumSamples: newLen,
		Samples:    outSamples,
	}
}
