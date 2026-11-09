package audio

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math"
	"math/rand"
	"strings"
	"testing"
)

// Helper to generate synthetic PCMBuffer with specified waveform envelope.
func createSyntheticPCMBuffer(sampleRate, channels, numSamples int, genFunc func(sampleIdx, ch int) float64) *PCMBuffer {
	samples := make([][]float64, channels)
	for c := 0; c < channels; c++ {
		samples[c] = make([]float64, numSamples)
		for s := 0; s < numSamples; s++ {
			samples[c][s] = genFunc(s, c)
		}
	}
	return &PCMBuffer{
		Channels:   channels,
		SampleRate: sampleRate,
		NumSamples: numSamples,
		Samples:    samples,
	}
}

// Helper to synthesize a valid canonical 16-bit PCM WAV stream in memory.
func createTestWAVStream(sampleRate, channels int, samples []float64) []byte {
	numSamples := len(samples) / channels
	dataSize := uint32(numSamples * channels * 2)
	fileSize := uint32(36 + dataSize)

	buf := new(bytes.Buffer)
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, fileSize)
	buf.WriteString("WAVEfmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16)) // subchunk1 size
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))  // PCM format
	_ = binary.Write(buf, binary.LittleEndian, uint16(channels))
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	byteRate := uint32(sampleRate * channels * 2)
	_ = binary.Write(buf, binary.LittleEndian, byteRate)
	blockAlign := uint16(channels * 2)
	_ = binary.Write(buf, binary.LittleEndian, blockAlign)
	_ = binary.Write(buf, binary.LittleEndian, uint16(16)) // bits per sample

	buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, dataSize)

	for _, s := range samples {
		clamped := math.Max(-1.0, math.Min(1.0, s))
		sampleInt16 := int16(clamped * 32767.0)
		_ = binary.Write(buf, binary.LittleEndian, sampleInt16)
	}

	return buf.Bytes()
}

func TestWaveform_Extract100Buckets_StandardSine(t *testing.T) {
	sr := 48000
	ch := 2
	durationSec := 2.0
	numSamples := int(float64(sr) * durationSec)

	// Exponentially decaying sine wave envelope
	pcm := createSyntheticPCMBuffer(sr, ch, numSamples, func(s, c int) float64 {
		timeSec := float64(s) / float64(sr)
		env := math.Exp(-2.0 * timeSec) // decaying from 1.0 to ~0.018
		freq := 440.0
		return env * math.Sin(2.0*math.Pi*freq*timeSec)
	})

	targetBuckets := 100
	wd, err := ExtractWaveformFromPCM(pcm, targetBuckets)
	if err != nil {
		t.Fatalf("ExtractWaveformFromPCM failed: %v", err)
	}

	if wd.NumBuckets != 100 {
		t.Errorf("expected 100 buckets, got %d", wd.NumBuckets)
	}
	if len(wd.Buckets) != 100 || len(wd.Peaks) != 100 {
		t.Fatalf("expected 100 buckets/peaks slice length, got %d and %d", len(wd.Buckets), len(wd.Peaks))
	}

	// First bucket should be near maximum (1.0), and last bucket significantly lower
	if wd.Peaks[0] < 0.95 {
		t.Errorf("expected initial peak near 1.0, got %f", wd.Peaks[0])
	}
	if wd.Peaks[99] >= wd.Peaks[0] {
		t.Errorf("expected decaying waveform peak at end, got start=%f end=%f", wd.Peaks[0], wd.Peaks[99])
	}

	// Verify all peaks are normalized within [0.0, 1.0]
	for i, p := range wd.Peaks {
		if p < 0.0 || p > 1.0 || math.IsNaN(p) || math.IsInf(p, 0) {
			t.Errorf("peak[%d] invalid value: %f", i, p)
		}
	}

	// Verify JSON marshaling
	jsonStr, err := WaveformDataJSON(wd)
	if err != nil {
		t.Fatalf("WaveformDataJSON failed: %v", err)
	}
	if !strings.Contains(jsonStr, `"num_buckets":100`) {
		t.Errorf("expected num_buckets in JSON: %s", jsonStr[:100])
	}
}

func TestWaveform_DualEnvelope_Symmetry(t *testing.T) {
	sr := 48000
	numSamples := 48000
	pcm := createSyntheticPCMBuffer(sr, 1, numSamples, func(s, c int) float64 {
		// Asymmetric half-wave rectified pulse:
		// Positive cycle = +0.8, Negative cycle = -0.4
		phase := math.Sin(2.0 * math.Pi * 100.0 * float64(s) / float64(sr))
		if phase >= 0 {
			return 0.8
		}
		return -0.4
	})

	wd, err := ExtractWaveformFromPCM(pcm, 100)
	if err != nil {
		t.Fatalf("ExtractWaveformFromPCM failed: %v", err)
	}

	// Positive max should normalize to 1.0, negative min should normalize to -0.5 (-0.4 / 0.8)
	for i, b := range wd.Buckets {
		if math.Abs(b.Max-1.0) > 0.05 {
			t.Errorf("bucket[%d] Max expected ~1.0, got %f", i, b.Max)
			break
		}
		if math.Abs(b.Min-(-0.5)) > 0.05 {
			t.Errorf("bucket[%d] Min expected ~ -0.5, got %f", i, b.Min)
			break
		}
		if math.Abs(b.AbsMax-1.0) > 0.05 {
			t.Errorf("bucket[%d] AbsMax expected ~1.0, got %f", i, b.AbsMax)
			break
		}
	}
}

func TestWaveform_MultiResolutionZoom(t *testing.T) {
	sr := 44100
	numSamples := 44100 * 3 // 3 seconds
	pcm := createSyntheticPCMBuffer(sr, 2, numSamples, func(s, c int) float64 {
		tSec := float64(s) / float64(sr)
		return math.Sin(2.0*math.Pi*220.0*tSec) * (0.5 + 0.5*math.Sin(2.0*math.Pi*2.0*tSec))
	})

	resolutions := []int{50, 100, 200, 500, 1000}
	for _, res := range resolutions {
		wd, err := ExtractWaveformFromPCM(pcm, res)
		if err != nil {
			t.Fatalf("resolution %d failed: %v", res, err)
		}
		if wd.NumBuckets != res || len(wd.Peaks) != res {
			t.Errorf("expected %d buckets, got %d", res, wd.NumBuckets)
		}
	}

	// Test ResampleWaveform function explicitly
	wd100, _ := ExtractWaveformFromPCM(pcm, 100)
	downsampled := ResampleWaveform(wd100.Peaks, 50)
	if len(downsampled) != 50 {
		t.Errorf("expected downsampled length 50, got %d", len(downsampled))
	}
	upsampled := ResampleWaveform(wd100.Peaks, 200)
	if len(upsampled) != 200 {
		t.Errorf("expected upsampled length 200, got %d", len(upsampled))
	}
}

func TestWaveform_DecibelScaleCompression(t *testing.T) {
	// Quiet signal at -40 dBFS (amplitude 0.01)
	samples := make([]float64, 4800)
	for i := range samples {
		samples[i] = 0.01 * math.Sin(float64(i)*0.1)
	}

	peaks, err := ExtractWaveformFromSamples(samples, 100)
	if err != nil {
		t.Fatalf("ExtractWaveformFromSamples failed: %v", err)
	}

	// Normalized linear peak is 1.0 (since maxAll is 0.01)
	// Now test ScaleWaveformToDecibels directly on raw amplitude peaks:
	rawPeaks := []float64{0.001, 0.01, 0.1, 1.0} // -60dB, -40dB, -20dB, 0dB
	scaled := ScaleWaveformToDecibels(rawPeaks, -60.0)

	expected := []float64{0.0, 20.0 / 60.0, 40.0 / 60.0, 1.0}
	for i := range scaled {
		if math.Abs(scaled[i]-expected[i]) > 0.01 {
			t.Errorf("scaled[%d] expected %f, got %f", i, expected[i], scaled[i])
		}
	}
	_ = peaks
}

func TestWaveform_BinaryPackingRoundTrip(t *testing.T) {
	peaks := []float64{0.0, 0.25, 0.50, 0.75, 1.0, 0.1234, 0.9876}
	packed := PackWaveformBinary(peaks)
	if len(packed) != len(peaks) {
		t.Fatalf("expected packed length %d, got %d", len(peaks), len(packed))
	}

	unpacked, err := UnpackWaveformBinary(packed)
	if err != nil {
		t.Fatalf("UnpackWaveformBinary failed: %v", err)
	}
	if len(unpacked) != len(peaks) {
		t.Fatalf("expected unpacked length %d, got %d", len(peaks), len(unpacked))
	}

	// Maximum 8-bit quantization error is 1/255 = 0.00392
	for i := range peaks {
		diff := math.Abs(peaks[i] - unpacked[i])
		if diff > (1.0 / 255.0 + 1e-6) {
			t.Errorf("peaks[%d] quantization error too high: orig=%f unpacked=%f diff=%f", i, peaks[i], unpacked[i], diff)
		}
	}

	// Dual envelope buckets binary packing test
	buckets := []WaveformBucket{
		{Min: -0.8, Max: 0.9, AbsMax: 0.9, RMS: 0.5},
		{Min: -0.3, Max: 0.4, AbsMax: 0.4, RMS: 0.2},
		{Min: 0.0, Max: 0.0, AbsMax: 0.0, RMS: 0.0},
	}
	packedBuckets := PackWaveformBucketsBinary(buckets)
	unpackedBuckets, err := UnpackWaveformBucketsBinary(packedBuckets)
	if err != nil {
		t.Fatalf("UnpackWaveformBucketsBinary failed: %v", err)
	}
	if len(unpackedBuckets) != len(buckets) {
		t.Fatalf("expected %d unpacked buckets, got %d", len(buckets), len(unpackedBuckets))
	}
	for i := range buckets {
		if math.Abs(buckets[i].Min-unpackedBuckets[i].Min) > 0.02 {
			t.Errorf("bucket[%d] Min diff too high: %f vs %f", i, buckets[i].Min, unpackedBuckets[i].Min)
		}
		if math.Abs(buckets[i].Max-unpackedBuckets[i].Max) > 0.02 {
			t.Errorf("bucket[%d] Max diff too high: %f vs %f", i, buckets[i].Max, unpackedBuckets[i].Max)
		}
	}
}

func TestWaveform_SVGPathAndBarsGeneration(t *testing.T) {
	peaks := []float64{0.1, 0.4, 0.8, 1.0, 0.7, 0.3, 0.05}
	svgPath := GenerateWaveformSVGPath(peaks, 500, 100)

	if !strings.HasPrefix(svgPath, "M0,50.00") {
		t.Errorf("expected SVG path starting at centerline, got: %s", svgPath[:20])
	}
	if !strings.HasSuffix(svgPath, "Z") {
		t.Errorf("expected SVG path ending with Z, got: %s", svgPath[len(svgPath)-5:])
	}
	if strings.Contains(svgPath, "NaN") || strings.Contains(svgPath, "Inf") {
		t.Errorf("SVG path contains invalid float strings: %s", svgPath)
	}

	// Standalone Bars SVG document
	barsSVG := GenerateWaveformBarsSVG(peaks, 400, 64, 3.0, 1.0)
	if !strings.HasPrefix(barsSVG, `<svg xmlns="http://www.w3.org/2000/svg"`) {
		t.Errorf("invalid SVG wrapper: %s", barsSVG[:50])
	}
	if !strings.Contains(barsSVG, "<rect") {
		t.Errorf("expected rect elements in waveform bars SVG")
	}
}

func TestWaveform_EdgeCases(t *testing.T) {
	// Nil buffer
	_, err := ExtractWaveformFromPCM(nil, 100)
	if err != ErrNilPCMBuffer {
		t.Errorf("expected ErrNilPCMBuffer, got %v", err)
	}

	// Empty buffer
	emptyPCM := &PCMBuffer{Channels: 0, SampleRate: 48000, NumSamples: 0, Samples: nil}
	_, err = ExtractWaveformFromPCM(emptyPCM, 100)
	if err != ErrEmptyPCMBuffer {
		t.Errorf("expected ErrEmptyPCMBuffer, got %v", err)
	}

	// Pure silence (all zeros)
	silentPCM := createSyntheticPCMBuffer(48000, 1, 4800, func(s, c int) float64 { return 0.0 })
	wd, err := ExtractWaveformFromPCM(silentPCM, 100)
	if err != nil {
		t.Fatalf("silence extraction failed: %v", err)
	}
	if wd.MaxPeak != 0.0 {
		t.Errorf("expected max peak 0.0 for silence, got %f", wd.MaxPeak)
	}
	for i, p := range wd.Peaks {
		if p != 0.0 {
			t.Errorf("expected peak[%d] = 0.0, got %f", i, p)
		}
	}

	// Very short buffer (1 sample) with 100 buckets
	shortPCM := createSyntheticPCMBuffer(48000, 1, 1, func(s, c int) float64 { return 0.75 })
	wdShort, err := ExtractWaveformFromPCM(shortPCM, 100)
	if err != nil {
		t.Fatalf("short buffer extraction failed: %v", err)
	}
	if wdShort.NumBuckets != 100 {
		t.Errorf("expected 100 buckets, got %d", wdShort.NumBuckets)
	}

	// Corrupt binary data
	_, err = UnpackWaveformBucketsBinary([]byte{1, 2})
	if err != ErrCorruptBinaryData {
		t.Errorf("expected ErrCorruptBinaryData, got %v", err)
	}
}

func TestWaveform_WaveformFromReaderAndWAV(t *testing.T) {
	sr := 44100
	ch := 2
	numSamples := 44100 // 1 second
	rawSamples := make([]float64, numSamples*ch)
	for i := range rawSamples {
		rawSamples[i] = 0.8 * math.Sin(2.0*math.Pi*440.0*float64(i)/(float64(sr*ch)))
	}

	wavBytes := createTestWAVStream(sr, ch, rawSamples)
	reader := bytes.NewReader(wavBytes)

	wd, err := ExtractWaveformFromReader(reader, 100)
	if err != nil {
		t.Fatalf("ExtractWaveformFromReader failed: %v", err)
	}
	if wd.NumBuckets != 100 {
		t.Errorf("expected 100 buckets, got %d", wd.NumBuckets)
	}
	if wd.Channels != 2 {
		t.Errorf("expected 2 channels, got %d", wd.Channels)
	}
	if wd.SampleRate != 44100 {
		t.Errorf("expected 44100 sample rate, got %d", wd.SampleRate)
	}
	if wd.MaxPeak < 0.75 || wd.MaxPeak > 0.85 {
		t.Errorf("expected max peak around 0.8, got %f", wd.MaxPeak)
	}
}

func TestWaveform100HeterogeneousStreams(t *testing.T) {
	sampleRates := []int{8000, 11025, 16000, 22050, 32000, 44100, 48000, 88200, 96000}
	channelsList := []int{1, 2, 4, 6}
	bucketCounts := []int{10, 50, 100, 128, 200, 500}

	rng := rand.New(rand.NewSource(20261109))
	totalTests := 100

	for iter := 0; iter < totalTests; iter++ {
		sr := sampleRates[rng.Intn(len(sampleRates))]
		channels := channelsList[rng.Intn(len(channelsList))]
		buckets := bucketCounts[rng.Intn(len(bucketCounts))]
		numSamples := sr / 10 + rng.Intn(sr*2) // 0.1s to 2.1s
		freq := 100.0 + rng.Float64()*3000.0
		amp := 0.05 + rng.Float64()*0.90
		waveformType := rng.Intn(4)

		pcm := createSyntheticPCMBuffer(sr, channels, numSamples, func(s, c int) float64 {
			tSec := float64(s) / float64(sr)
			var base float64
			switch waveformType {
			case 0: // Sine
				base = math.Sin(2.0 * math.Pi * freq * tSec)
			case 1: // Square
				if math.Sin(2.0*math.Pi*freq*tSec) >= 0 {
					base = 1.0
				} else {
					base = -1.0
				}
			case 2: // Triangle
				base = 2.0*math.Abs(2.0*(tSec*freq-math.Floor(tSec*freq+0.5))) - 1.0
			case 3: // Transient impulses
				if s%1000 == 0 {
					base = 1.0
				} else {
					base = 0.05 * (rng.Float64()*2.0 - 1.0)
				}
			}
			return amp * base
		})

		wd, err := ExtractWaveformFromPCM(pcm, buckets)
		if err != nil {
			t.Fatalf("iteration %d failed: %v", iter, err)
		}

		if wd.NumBuckets != buckets || len(wd.Peaks) != buckets {
			t.Fatalf("iteration %d: expected %d buckets, got %d", iter, buckets, wd.NumBuckets)
		}

		// Verify binary packing round-trip
		packed := PackWaveformBinary(wd.Peaks)
		unpacked, err := UnpackWaveformBinary(packed)
		if err != nil {
			t.Fatalf("iteration %d binary unpack failed: %v", iter, err)
		}
		if len(unpacked) != buckets {
			t.Fatalf("iteration %d binary unpacked length mismatch", iter)
		}

		// Verify JSON serialization round-trip
		jsonStr, err := WaveformDataJSON(wd)
		if err != nil {
			t.Fatalf("iteration %d json marshal failed: %v", iter, err)
		}
		var parsed WaveformData
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
			t.Fatalf("iteration %d json unmarshal failed: %v", iter, err)
		}
		if parsed.NumBuckets != buckets {
			t.Fatalf("iteration %d parsed json mismatch", iter)
		}

		// Verify SVG path valid string
		svgPath := GenerateWaveformSVGPath(wd.Peaks, 300, 50)
		if len(svgPath) == 0 || strings.Contains(svgPath, "NaN") {
			t.Fatalf("iteration %d invalid svg path", iter)
		}
	}

	t.Logf("Successfully verified 100 heterogeneous waveform extractions across multi-channel, multi-rate streams.")
}
