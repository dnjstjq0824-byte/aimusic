package audio

import (
	"math"
	"math/rand"
	"testing"
)

// TestBinaural_ITD_WoodworthModel tests Woodworth interaural time difference formula.
func TestBinaural_ITD_WoodworthModel(t *testing.T) {
	config := DefaultBinauralConfig()
	renderer := NewBinauralRenderer(config)

	// Center front (0 deg)
	dl0, dr0 := renderer.ComputeITD(0.0, 0.0)
	if dl0 != 0.0 || dr0 != 0.0 {
		t.Errorf("At 0 deg azimuth expected 0 ITD, got L=%.6fs, R=%.6fs", dl0, dr0)
	}

	// Full Right (+90 deg): Left ear should lag by ~0.656 ms (0.000656 s)
	dl90, dr90 := renderer.ComputeITD(90.0, 0.0)
	expectedITD := (config.HeadRadiusMeters / config.SoundSpeedMetersPerSec) * (1.0 + math.Pi/2.0)
	t.Logf("+90 deg ITD: Left=%.6f ms, Right=%.6f ms, Expected=%.6f ms", dl90*1000.0, dr90*1000.0, expectedITD*1000.0)

	if dr90 != 0.0 {
		t.Errorf("Expected right ear to have 0 delay at +90 deg, got %.6f", dr90)
	}
	if math.Abs(dl90-expectedITD) > 1e-6 {
		t.Errorf("Left ear ITD mismatch at +90 deg: expected %.6fs, got %.6fs", expectedITD, dl90)
	}

	// Full Left (-90 deg): Right ear should lag by same amount
	dlMinus90, drMinus90 := renderer.ComputeITD(-90.0, 0.0)
	if dlMinus90 != 0.0 {
		t.Errorf("Expected left ear to have 0 delay at -90 deg, got %.6f", dlMinus90)
	}
	if math.Abs(drMinus90-expectedITD) > 1e-6 {
		t.Errorf("Right ear ITD mismatch at -90 deg: expected %.6fs, got %.6fs", expectedITD, drMinus90)
	}
}

// TestBinaural_ILD_HeadShadowLevels tests interaural level difference calculations.
func TestBinaural_ILD_HeadShadowLevels(t *testing.T) {
	renderer := NewBinauralRenderer(DefaultBinauralConfig())

	// Center (0 deg): gains should be identical (0 dB relative diff)
	gl0, gr0 := renderer.ComputeILD(0.0, 0.0)
	if math.Abs(gl0-gr0) > 1e-6 {
		t.Errorf("At center azimuth expected equal gains, got L=%.4f, R=%.4f", gl0, gr0)
	}

	// Full Right (+90 deg): Right ear boosted (+2.5 dB), Left ear shadowed (-9.0 dB)
	gl90, gr90 := renderer.ComputeILD(90.0, 0.0)
	expectedRightDB := 2.5
	expectedLeftDB := -9.0
	actualRightDB := 20.0 * math.Log10(gr90)
	actualLeftDB := 20.0 * math.Log10(gl90)

	t.Logf("+90 deg ILD: Right=%.2f dB, Left=%.2f dB (Difference=%.2f dB)",
		actualRightDB, actualLeftDB, actualRightDB-actualLeftDB)

	if math.Abs(actualRightDB-expectedRightDB) > 0.01 {
		t.Errorf("Right ear gain mismatch at +90 deg: expected %.2f dB, got %.2f dB", expectedRightDB, actualRightDB)
	}
	if math.Abs(actualLeftDB-expectedLeftDB) > 0.01 {
		t.Errorf("Left ear gain mismatch at +90 deg: expected %.2f dB, got %.2f dB", expectedLeftDB, actualLeftDB)
	}

	// ILD difference should be 11.5 dB
	ildDB := actualRightDB - actualLeftDB
	if math.Abs(ildDB-11.5) > 0.01 {
		t.Errorf("Total ILD mismatch: expected 11.5 dB, got %.2f dB", ildDB)
	}
}

// TestBinaural_PinnaNotch_ElevationShift verifies elevation spectral notch filtering.
func TestBinaural_PinnaNotch_ElevationShift(t *testing.T) {
	sr := 48000
	numSamples := 4800

	// Test notch attenuation at 8300 Hz (0 deg elevation notch)
	notchFreq := 8300.0
	sig := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		sig[i] = 0.5 * math.Sin(2.0*math.Pi*notchFreq*float64(i)/float64(sr))
	}

	// Apply notch filter with -8 dB gain at 8300 Hz
	filtered := applyNotchFilter(sig, float64(sr), notchFreq, 4.0, -8.0)

	var origRMS, filtRMS float64
	for i := 1000; i < numSamples; i++ {
		origRMS += sig[i] * sig[i]
		filtRMS += filtered[i] * filtered[i]
	}
	origRMS = math.Sqrt(origRMS / float64(numSamples-1000))
	filtRMS = math.Sqrt(filtRMS / float64(numSamples-1000))

	measuredAttenuationDB := 20.0 * math.Log10(filtRMS/origRMS)
	t.Logf("Pinna Notch at %.0f Hz: Measured Attenuation = %.2f dB", notchFreq, measuredAttenuationDB)

	if measuredAttenuationDB > -6.0 || measuredAttenuationDB < -10.0 {
		t.Errorf("Pinna notch attenuation outside expected range [-10dB, -6dB]: got %.2f dB", measuredAttenuationDB)
	}
}

// TestBinaural_DistanceAttenuation_InverseSquareLaw verifies 1/r sound decay.
func TestBinaural_DistanceAttenuation_InverseSquareLaw(t *testing.T) {
	sr := 48000
	numSamples := 4800
	sig := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		sig[i] = 0.5 * math.Sin(2.0*math.Pi*500.0*float64(i)/float64(sr))
	}

	pcm := &PCMBuffer{
		SampleRate: sr,
		Channels:   1,
		NumSamples: numSamples,
		Samples:    [][]float64{sig},
	}

	config := DefaultBinauralConfig()
	config.EnableAirAbsorption = false // Isolate inverse square law
	config.EnablePinnaNotch = false
	renderer := NewBinauralRenderer(config)

	// Render at 1.0m, 2.0m, and 4.0m
	out1m, _ := renderer.Render(pcm, SpatialSource{AzimuthDeg: 0, ElevationDeg: 0, DistanceMeters: 1.0})
	out2m, _ := renderer.Render(pcm, SpatialSource{AzimuthDeg: 0, ElevationDeg: 0, DistanceMeters: 2.0})
	out4m, _ := renderer.Render(pcm, SpatialSource{AzimuthDeg: 0, ElevationDeg: 0, DistanceMeters: 4.0})

	rms1m := computeBufferRMS(out1m.Samples[0])
	rms2m := computeBufferRMS(out2m.Samples[0])
	rms4m := computeBufferRMS(out4m.Samples[0])

	ratio1to2 := rms2m / rms1m
	ratio1to4 := rms4m / rms1m

	t.Logf("Distance decay: 1m->2m ratio=%.4f (Expected 0.5000), 1m->4m ratio=%.4f (Expected 0.2500)",
		ratio1to2, ratio1to4)

	if math.Abs(ratio1to2-0.5) > 0.02 {
		t.Errorf("1m to 2m distance decay error: expected 0.5, got %.4f", ratio1to2)
	}
	if math.Abs(ratio1to4-0.25) > 0.02 {
		t.Errorf("1m to 4m distance decay error: expected 0.25, got %.4f", ratio1to4)
	}
}

// TestBinaural_FullSpatialRender_HeadphoneLocalization verifies full binaural render spatialization.
func TestBinaural_FullSpatialRender_HeadphoneLocalization(t *testing.T) {
	sr := 48000
	numSamples := 9600
	sig := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		sig[i] = 0.5 * math.Sin(2.0*math.Pi*2000.0*float64(i)/float64(sr))
	}

	pcm := &PCMBuffer{
		SampleRate: sr,
		Channels:   1,
		NumSamples: numSamples,
		Samples:    [][]float64{sig},
	}

	renderer := NewBinauralRenderer(DefaultBinauralConfig())

	// Source placed directly to the right (+90 deg, 1.5 meters away)
	sourceRight := SpatialSource{
		AzimuthDeg:     90.0,
		ElevationDeg:   0.0,
		DistanceMeters: 1.5,
	}

	rendered, err := renderer.Render(pcm, sourceRight)
	if err != nil {
		t.Fatalf("Binaural render failed: %v", err)
	}

	leftRMS := computeBufferRMS(rendered.Samples[0])
	rightRMS := computeBufferRMS(rendered.Samples[1])

	diffDB := 20.0 * math.Log10(rightRMS/leftRMS)
	t.Logf("Right Panned Source: Left RMS=%.4f, Right RMS=%.4f, Right/Left diff = +%.2f dB",
		leftRMS, rightRMS, diffDB)

	// Right ear should be significantly louder than left ear (> 10 dB difference due to ILD + head shadow)
	if diffDB < 10.0 {
		t.Errorf("Expected right ear to be at least 10 dB louder than left ear, got %.2f dB", diffDB)
	}
}

// TestBinaural_EdgeCases tests extreme parameters and boundary validations.
func TestBinaural_EdgeCases(t *testing.T) {
	renderer := NewBinauralRenderer(DefaultBinauralConfig())

	// Nil buffer
	if _, err := renderer.Render(nil, SpatialSource{}); err == nil {
		t.Errorf("Expected error for nil buffer")
	}

	// Empty buffer
	emptyPCM := &PCMBuffer{
		SampleRate: 48000,
		Channels:   1,
		NumSamples: 0,
		Samples:    [][]float64{},
	}
	if _, err := renderer.Render(emptyPCM, SpatialSource{}); err == nil {
		t.Errorf("Expected error for empty buffer")
	}

	// Boundary azimuth angle wraps properly
	dl1, _ := renderer.ComputeITD(450.0, 0.0) // 450 deg = 90 deg
	dl2, _ := renderer.ComputeITD(90.0, 0.0)
	if math.Abs(dl1-dl2) > 1e-9 {
		t.Errorf("Azimuth normalization failed: 450 deg (%.6f) != 90 deg (%.6f)", dl1, dl2)
	}
}

// TestBinaural_100HeterogeneousStreams verifies 100 heterogeneous 3D spatial renders.
func TestBinaural_100HeterogeneousStreams(t *testing.T) {
	rng := rand.New(rand.NewSource(12345678))
	sampleRates := []int{22050, 44100, 48000, 96000}

	renderer := NewBinauralRenderer(DefaultBinauralConfig())

	for i := 0; i < 100; i++ {
		sr := sampleRates[rng.Intn(len(sampleRates))]
		channels := rng.Intn(2) + 1
		length := rng.Intn(2400) + 2400

		pcmSamples := make([][]float64, channels)
		for ch := 0; ch < channels; ch++ {
			pcmSamples[ch] = make([]float64, length)
			freq := 150.0 + rng.Float64()*4000.0
			for s := 0; s < length; s++ {
				pcmSamples[ch][s] = 0.4 * math.Sin(2.0*math.Pi*freq*float64(s)/float64(sr))
			}
		}

		pcm := &PCMBuffer{
			SampleRate: sr,
			Channels:   channels,
			NumSamples: length,
			Samples:    pcmSamples,
		}

		source := SpatialSource{
			AzimuthDeg:     -180.0 + rng.Float64()*360.0,
			ElevationDeg:   -45.0 + rng.Float64()*135.0,
			DistanceMeters: 0.3 + rng.Float64()*20.0,
		}

		rendered, err := renderer.Render(pcm, source)
		if err != nil {
			t.Fatalf("Stream %d spatial render failed: %v", i+1, err)
		}

		if rendered.Channels != 2 {
			t.Fatalf("Stream %d expected 2 binaural channels, got %d", i+1, rendered.Channels)
		}
		if rendered.NumSamples != length {
			t.Fatalf("Stream %d length mismatch: expected %d, got %d", i+1, length, rendered.NumSamples)
		}
	}

	t.Log("Successfully verified 100 heterogeneous audio streams with 3D binaural spatial audio rendering.")
}

func computeBufferRMS(samples []float64) float64 {
	var sum float64
	for _, v := range samples {
		sum += v * v
	}
	return math.Sqrt(sum / float64(len(samples)))
}
