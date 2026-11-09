package audio

import (
	"errors"
	"math"
	"testing"
)

func TestInspector_CleanAudioPass(t *testing.T) {
	sampleRate := 48000
	duration := 3.0
	numSamples := int(float64(sampleRate) * duration)
	left := make([]float64, numSamples)
	right := make([]float64, numSamples)

	for i := 0; i < numSamples; i++ {
		tSec := float64(i) / float64(sampleRate)
		v := 0.5 * math.Sin(2.0*math.Pi*440.0*tSec)
		left[i] = v
		right[i] = v
	}

	buf := &PCMBuffer{
		Channels:   2,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{left, right},
	}

	inspector := NewAudioInspector()
	report, err := inspector.Inspect(buf)
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	t.Logf("Clean Audio: Passed = %v, Defects = %d, PhaseCorr = %.3f, DC = %v",
		report.Passed, report.TotalDefects, report.PhaseCorrelation, report.DCOffset)

	if !report.Passed || report.TotalDefects != 0 {
		t.Errorf("Expected clean audio to pass with 0 defects, got passed=%v defects=%+v",
			report.Passed, report.Defects)
	}
	if report.ClippedSamples != 0 || report.ClippedEvents != 0 {
		t.Errorf("Expected 0 clipped samples, got %d", report.ClippedSamples)
	}
}

func TestInspector_DeadSilenceDefect(t *testing.T) {
	sampleRate := 48000
	numSamples := sampleRate * 2
	buf := &PCMBuffer{
		Channels:   2,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{make([]float64, numSamples), make([]float64, numSamples)},
	}

	inspector := NewAudioInspector()
	report, err := inspector.Inspect(buf)
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	t.Logf("Dead Silence: Passed = %v, Defects = %d", report.Passed, report.TotalDefects)
	if report.Passed {
		t.Errorf("Expected dead silence to fail QC")
	}

	found := false
	for _, d := range report.Defects {
		if d.Type == DefectDeadSilence {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected DefectDeadSilence in defects list")
	}
}

func TestInspector_LeadInAndLeadOutSilence(t *testing.T) {
	// Total 10s: 4s lead-in silence (limit 3s), 2s audio, 4s lead-out silence (limit 3s)
	sampleRate := 48000
	numSamples := sampleRate * 10
	left := make([]float64, numSamples)
	right := make([]float64, numSamples)

	startAudio := sampleRate * 4
	endAudio := sampleRate * 6
	for i := startAudio; i < endAudio; i++ {
		v := 0.6 * math.Sin(2.0*math.Pi*1000.0*float64(i)/float64(sampleRate))
		left[i] = v
		right[i] = v
	}

	buf := &PCMBuffer{
		Channels:   2,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{left, right},
	}

	inspector := NewAudioInspector(WithMaxLeadInSec(3.0), WithMaxLeadOutSec(3.0))
	report, err := inspector.Inspect(buf)
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	t.Logf("Lead silence: LeadIn = %.2f s, LeadOut = %.2f s, Defects = %d",
		report.LeadInSilenceSec, report.LeadOutSilenceSec, report.TotalDefects)

	if report.Passed {
		t.Errorf("Expected excessive lead silence to fail QC")
	}

	hasLeadIn := false
	hasLeadOut := false
	for _, d := range report.Defects {
		if d.Type == DefectExcessiveLeadIn {
			hasLeadIn = true
		}
		if d.Type == DefectExcessiveLeadOut {
			hasLeadOut = true
		}
	}
	if !hasLeadIn || !hasLeadOut {
		t.Errorf("Expected both lead-in and lead-out defects: leadIn=%v leadOut=%v", hasLeadIn, hasLeadOut)
	}
}

func TestInspector_MidTrackDropout(t *testing.T) {
	// 6s total: 2s audio, 2s silence gap (limit 1.5s), 2s audio
	sampleRate := 48000
	numSamples := sampleRate * 6
	left := make([]float64, numSamples)
	right := make([]float64, numSamples)

	for i := 0; i < sampleRate*2; i++ {
		v := 0.5 * math.Sin(2.0*math.Pi*440.0*float64(i)/float64(sampleRate))
		left[i] = v
		right[i] = v
	}
	for i := sampleRate * 4; i < sampleRate*6; i++ {
		v := 0.5 * math.Sin(2.0*math.Pi*880.0*float64(i)/float64(sampleRate))
		left[i] = v
		right[i] = v
	}

	buf := &PCMBuffer{
		Channels:   2,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{left, right},
	}

	inspector := NewAudioInspector(WithMaxDropoutSec(1.5))
	report, err := inspector.Inspect(buf)
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	t.Logf("Mid-track dropout: MaxDropout = %.2f s, Defects = %d", report.MaxDropoutSec, report.TotalDefects)

	if report.Passed {
		t.Errorf("Expected mid-track dropout to fail QC")
	}

	hasDropout := false
	for _, d := range report.Defects {
		if d.Type == DefectMidTrackDropout {
			hasDropout = true
			if d.StartSec < 1.9 || d.StartSec > 2.1 {
				t.Errorf("Dropout start mismatch: expected ~2.0s, got %.2fs", d.StartSec)
			}
			if d.DurationSec < 1.9 || d.DurationSec > 2.1 {
				t.Errorf("Dropout duration mismatch: expected ~2.0s, got %.2fs", d.DurationSec)
			}
		}
	}
	if !hasDropout {
		t.Errorf("Expected DefectMidTrackDropout in report")
	}
}

func TestInspector_DigitalClipping(t *testing.T) {
	// Synthesize an overdriven waveform clamped hard at +/- 1.0
	sampleRate := 48000
	numSamples := sampleRate
	left := make([]float64, numSamples)
	right := make([]float64, numSamples)

	for i := 0; i < numSamples; i++ {
		// Massive overdrive (amp 3.0) clamped to [-1.0, 1.0]
		raw := 3.0 * math.Sin(2.0*math.Pi*100.0*float64(i)/float64(sampleRate))
		if raw > 1.0 {
			raw = 1.0
		} else if raw < -1.0 {
			raw = -1.0
		}
		left[i] = raw
		right[i] = raw
	}

	buf := &PCMBuffer{
		Channels:   2,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{left, right},
	}

	inspector := NewAudioInspector()
	report, err := inspector.Inspect(buf)
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	t.Logf("Clipping test: ClippedSamples = %d, Events = %d, Pct = %.2f%%, Passed = %v",
		report.ClippedSamples, report.ClippedEvents, report.ClippedPercent, report.Passed)

	if report.Passed {
		t.Errorf("Expected severely clipped audio to fail QC")
	}
	if report.ClippedEvents == 0 || report.ClippedSamples == 0 {
		t.Errorf("Expected clipping events and samples to be detected")
	}

	hasClippingDefect := false
	for _, d := range report.Defects {
		if d.Type == DefectSevereClipping || d.Type == DefectMinorClipping {
			hasClippingDefect = true
		}
	}
	if !hasClippingDefect {
		t.Errorf("Expected clipping defect in report")
	}
}

func TestInspector_DCOffsetAndRemoval(t *testing.T) {
	// Audio with a +0.05 DC bias (exceeds threshold 0.002)
	sampleRate := 48000
	numSamples := sampleRate * 2
	left := make([]float64, numSamples)
	right := make([]float64, numSamples)

	dcBias := 0.05
	for i := 0; i < numSamples; i++ {
		v := 0.4 * math.Sin(2.0*math.Pi*440.0*float64(i)/float64(sampleRate))
		left[i] = v + dcBias
		right[i] = v + dcBias
	}

	buf := &PCMBuffer{
		Channels:   2,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{left, right},
	}

	inspector := NewAudioInspector(WithDCOffsetThreshold(0.002))
	report1, err := inspector.Inspect(buf)
	if err != nil {
		t.Fatalf("Inspect 1 failed: %v", err)
	}

	t.Logf("Before DC removal: DC = %v, Passed = %v, Defects = %d",
		report1.DCOffset, report1.Passed, report1.TotalDefects)

	if report1.Passed {
		t.Errorf("Expected audio with DC bias to fail QC")
	}

	// Apply DC Offset correction
	cleanedBuf := RemoveDCOffset(buf)
	report2, err := inspector.Inspect(cleanedBuf)
	if err != nil {
		t.Fatalf("Inspect 2 failed: %v", err)
	}

	t.Logf("After DC removal: DC = %v, Passed = %v, Defects = %d",
		report2.DCOffset, report2.Passed, report2.TotalDefects)

	if !report2.Passed {
		t.Errorf("Expected audio after DC removal to pass QC, got defects: %+v", report2.Defects)
	}
	for ch, dc := range report2.DCOffset {
		if math.Abs(dc) > 1e-12 {
			t.Errorf("Channel %d remaining DC bias too high: %e", ch, dc)
		}
	}
}

func TestInspector_PhaseInversion(t *testing.T) {
	// Stereo signal with Left and Right 180 degrees out of phase (R = -L)
	sampleRate := 48000
	numSamples := sampleRate * 2
	left := make([]float64, numSamples)
	right := make([]float64, numSamples)

	for i := 0; i < numSamples; i++ {
		v := 0.5 * math.Sin(2.0*math.Pi*440.0*float64(i)/float64(sampleRate))
		left[i] = v
		right[i] = -v // 180 degrees inverted
	}

	buf := &PCMBuffer{
		Channels:   2,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{left, right},
	}

	inspector := NewAudioInspector()
	report, err := inspector.Inspect(buf)
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	t.Logf("Phase Inversion: PhaseCorr = %.4f, Passed = %v", report.PhaseCorrelation, report.Passed)

	if report.Passed {
		t.Errorf("Expected inverted phase stereo to fail QC")
	}
	if report.PhaseCorrelation > -0.99 {
		t.Errorf("Expected phase correlation close to -1.0, got %.4f", report.PhaseCorrelation)
	}

	hasInversion := false
	for _, d := range report.Defects {
		if d.Type == DefectPhaseInversion {
			hasInversion = true
		}
	}
	if !hasInversion {
		t.Errorf("Expected DefectPhaseInversion in report")
	}
}

func TestInspector_TrimSilence(t *testing.T) {
	// 5s audio: 1.5s silence, 2s music, 1.5s silence
	sampleRate := 48000
	numSamples := sampleRate * 5
	left := make([]float64, numSamples)
	right := make([]float64, numSamples)

	startMusic := int(float64(sampleRate) * 1.5)
	endMusic := int(float64(sampleRate) * 3.5)
	for i := startMusic; i < endMusic; i++ {
		v := 0.6 * math.Sin(2.0*math.Pi*1000.0*float64(i)/float64(sampleRate))
		left[i] = v
		right[i] = v
	}

	buf := &PCMBuffer{
		Channels:   2,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    [][]float64{left, right},
	}

	trimmed := TrimSilence(buf, -50.0)
	trimmedDurationSec := float64(trimmed.NumSamples) / float64(sampleRate)

	t.Logf("Trim silence: Original = %.2fs, Trimmed = %.2fs", 5.0, trimmedDurationSec)

	// Trimmed duration should be approximately 2.0s (+/- 0.05s)
	if math.Abs(trimmedDurationSec-2.0) > 0.05 {
		t.Errorf("Expected trimmed duration ~2.0s, got %.2fs", trimmedDurationSec)
	}
}

func TestInspector_EdgeCases(t *testing.T) {
	inspector := NewAudioInspector()

	// 1. Nil buffer
	_, err := inspector.Inspect(nil)
	if !errors.Is(err, ErrNilFingerprintBuffer) {
		t.Errorf("Expected ErrNilFingerprintBuffer, got %v", err)
	}

	// 2. Empty buffer
	emptyBuf := &PCMBuffer{Channels: 0, SampleRate: 48000, NumSamples: 0, Samples: nil}
	_, err = inspector.Inspect(emptyBuf)
	if !errors.Is(err, ErrEmptyAudioBuffer) {
		t.Errorf("Expected ErrEmptyAudioBuffer, got %v", err)
	}

	// 3. RemoveDCOffset on nil
	if res := RemoveDCOffset(nil); res != nil {
		t.Errorf("Expected nil when removing DC from nil buffer")
	}

	// 4. TrimSilence on nil
	if res := TrimSilence(nil, -50.0); res != nil {
		t.Errorf("Expected nil when trimming nil buffer")
	}
}

func TestInspector100HeterogeneousStreams(t *testing.T) {
	inspector := NewAudioInspector()

	count := 0
	for i := 0; i < 100; i++ {
		count++
		sampleRate := 48000
		channels := (i % 2) + 1
		duration := 0.2 + float64(i%5)*0.1 // 0.2s ~ 0.6s
		numSamples := int(float64(sampleRate) * duration)

		samples := make([][]float64, channels)
		for ch := 0; ch < channels; ch++ {
			samples[ch] = make([]float64, numSamples)
			freq := 200.0 + float64(i)*20.0
			amp := 0.1 + float64(i%8)*0.1
			for n := 0; n < numSamples; n++ {
				samples[ch][n] = amp * math.Sin(2.0*math.Pi*freq*float64(n)/float64(sampleRate))
			}
		}

		buf := &PCMBuffer{
			Channels:   channels,
			SampleRate: sampleRate,
			NumSamples: numSamples,
			Samples:    samples,
		}

		report, err := inspector.Inspect(buf)
		if err != nil {
			t.Fatalf("Stream %d failed inspection: %v", count, err)
		}

		if math.IsNaN(report.PhaseCorrelation) || math.IsInf(report.PhaseCorrelation, 0) {
			t.Fatalf("Stream %d: NaN or Inf phase correlation", count)
		}
		for ch, dc := range report.DCOffset {
			if math.IsNaN(dc) || math.IsInf(dc, 0) {
				t.Fatalf("Stream %d: NaN or Inf DC offset on ch %d", count, ch)
			}
		}
	}

	t.Logf("Successfully inspected %d heterogeneous audio streams with 0 errors.", count)
}
