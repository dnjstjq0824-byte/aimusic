package audio

import (
	"math"
	"math/rand"
	"testing"
)

// TestSTFT_CooleyTukeyRadix2_FFT_iFFT_Roundtrip verifies FFT and iFFT roundtrip precision.
func TestSTFT_CooleyTukeyRadix2_FFT_iFFT_Roundtrip(t *testing.T) {
	n := 1024
	origReal := make([]float64, n)

	// Construct test waveform
	for i := 0; i < n; i++ {
		origReal[i] = 0.6*math.Sin(2.0*math.Pi*5.0*float64(i)/float64(n)) +
			0.3*math.Cos(2.0*math.Pi*25.0*float64(i)/float64(n))
	}

	testReal := make([]float64, n)
	testImag := make([]float64, n)
	copy(testReal, origReal)

	// Forward FFT
	fftInPlace(testReal, testImag, false)

	// Inverse FFT
	fftInPlace(testReal, testImag, true)

	var maxDiff float64
	for i := 0; i < n; i++ {
		diffR := math.Abs(testReal[i] - origReal[i])
		diffI := math.Abs(testImag[i])
		if diffR > maxDiff {
			maxDiff = diffR
		}
		if diffI > maxDiff {
			maxDiff = diffI
		}
	}

	t.Logf("Radix-2 FFT/iFFT max roundtrip diff: %.2e", maxDiff)
	if maxDiff > 1e-12 {
		t.Errorf("FFT/iFFT roundtrip precision error: max diff=%.2e", maxDiff)
	}
}

// TestSTFT_iSTFT_PerfectReconstruction_SNR tests full STFT forward and iSTFT inverse reconstruction SNR.
func TestSTFT_iSTFT_PerfectReconstruction_SNR(t *testing.T) {
	sr := 48000
	numSamples := 48000 // 1.0 second

	sig := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		tVal := float64(i) / float64(sr)
		sig[i] = 0.4*math.Sin(2.0*math.Pi*150.0*tVal) +
			0.3*math.Sin(2.0*math.Pi*800.0*tVal) +
			0.2*math.Sin(2.0*math.Pi*3200.0*tVal)
	}

	pcm := &PCMBuffer{
		SampleRate: sr,
		Channels:   1,
		NumSamples: numSamples,
		Samples:    [][]float64{sig},
	}

	config := DefaultSTFTConfig()
	config.FFTSize = 1024
	config.HopSize = 256 // 75% overlap ensures perfect WOLA reconstruction

	engine, err := NewSTFTEngine(config)
	if err != nil {
		t.Fatalf("Failed to create STFT engine: %v", err)
	}

	stftRes, err := engine.Forward(pcm)
	if err != nil {
		t.Fatalf("STFT Forward failed: %v", err)
	}

	recPCM, err := engine.Inverse(stftRes)
	if err != nil {
		t.Fatalf("STFT Inverse failed: %v", err)
	}

	// Evaluate SNR over valid non-edge samples (exclude initial and final FFTSize edges where window tapers off)
	margin := config.FFTSize
	evalStart := margin
	evalEnd := recPCM.NumSamples - margin
	if evalEnd > numSamples {
		evalEnd = numSamples
	}

	var sigEnergy, noiseEnergy float64
	for i := evalStart; i < evalEnd; i++ {
		orig := sig[i]
		rec := recPCM.Samples[0][i]
		noise := orig - rec
		sigEnergy += orig * orig
		noiseEnergy += noise * noise
	}

	snrDB := 10.0 * math.Log10(sigEnergy/noiseEnergy)
	t.Logf("STFT-iSTFT Reconstruction SNR: %.2f dB", snrDB)

	// Required: SNR > 90.0 dB
	if snrDB < 90.0 {
		t.Errorf("Reconstruction SNR below threshold: expected > 90 dB, got %.2f dB", snrDB)
	}
}

// TestSTFT_ParsevalEnergyConservation verifies Parseval energy conservation between time and frequency.
func TestSTFT_ParsevalEnergyConservation(t *testing.T) {
	sr := 48000
	fftSize := 1024
	numSamples := fftSize // Single frame

	sig := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		sig[i] = 0.5 * math.Sin(2.0*math.Pi*440.0*float64(i)/float64(sr))
	}

	config := STFTConfig{
		FFTSize:    fftSize,
		HopSize:    fftSize,
		Window:     WindowHann,
		NumMelBins: 80,
	}
	engine, _ := NewSTFTEngine(config)

	// Time domain energy of windowed signal
	var timeEnergy float64
	for i := 0; i < fftSize; i++ {
		wVal := sig[i] * engine.window[i]
		timeEnergy += wVal * wVal
	}

	realBuf := make([]float64, fftSize)
	imagBuf := make([]float64, fftSize)
	for i := 0; i < fftSize; i++ {
		realBuf[i] = sig[i] * engine.window[i]
	}
	fftInPlace(realBuf, imagBuf, false)

	// Frequency domain energy (Parseval: sum(|X[k]|^2) / N)
	var freqEnergy float64
	for k := 0; k < fftSize; k++ {
		freqEnergy += realBuf[k]*realBuf[k] + imagBuf[k]*imagBuf[k]
	}
	freqEnergy /= float64(fftSize)

	ratioDB := math.Abs(10.0 * math.Log10(timeEnergy/freqEnergy))
	t.Logf("Parseval Energy Diff: %.6f dB", ratioDB)

	if ratioDB > 0.05 {
		t.Errorf("Parseval energy conservation violation: diff=%.4f dB (> 0.05 dB)", ratioDB)
	}
}

// TestSTFT_FrequencyBinAccuracy validates peak bin location against analytical expectation.
func TestSTFT_FrequencyBinAccuracy(t *testing.T) {
	sr := 48000
	fftSize := 2048
	numSamples := fftSize * 4
	targetFreq := 1200.0 // 1.2 kHz

	sig := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		sig[i] = 0.7 * math.Sin(2.0*math.Pi*targetFreq*float64(i)/float64(sr))
	}

	pcm := &PCMBuffer{
		SampleRate: sr,
		Channels:   1,
		NumSamples: numSamples,
		Samples:    [][]float64{sig},
	}

	config := STFTConfig{
		FFTSize:    fftSize,
		HopSize:    fftSize / 2,
		Window:     WindowHann,
		NumMelBins: 80,
	}
	engine, _ := NewSTFTEngine(config)

	res, err := engine.Forward(pcm)
	if err != nil {
		t.Fatalf("Forward STFT failed: %v", err)
	}

	binResolution := float64(sr) / float64(fftSize) // 48000 / 2048 = 23.4375 Hz
	expectedBin := int(math.Round(targetFreq / binResolution)) // 1200 / 23.4375 = 51.2 -> 51

	// Check middle frame peak bin
	midFrame := res.Frames[0][res.NumFrames/2]
	maxBin := 0
	maxMag := -1.0
	for k := 0; k < res.NumBins; k++ {
		if midFrame.Magnitude[k] > maxMag {
			maxMag = midFrame.Magnitude[k]
			maxBin = k
		}
	}

	t.Logf("Expected Peak Bin: %d (~%.1f Hz), Detected Peak Bin: %d (~%.1f Hz)",
		expectedBin, float64(expectedBin)*binResolution, maxBin, float64(maxBin)*binResolution)

	if math.Abs(float64(maxBin-expectedBin)) > 1 {
		t.Errorf("Frequency bin peak mismatch: expected %d, got %d", expectedBin, maxBin)
	}
}

// TestSTFT_MelSpectrogram_FilterbankAndLogDB tests 80-bin Mel filterbank energy projection.
func TestSTFT_MelSpectrogram_FilterbankAndLogDB(t *testing.T) {
	sr := 48000
	numSamples := 48000

	// Bass tone (120 Hz) + High Treble tone (7000 Hz)
	bassSig := make([]float64, numSamples)
	trebleSig := make([]float64, numSamples)

	for i := 0; i < numSamples; i++ {
		tVal := float64(i) / float64(sr)
		bassSig[i] = 0.6 * math.Sin(2.0*math.Pi*120.0*tVal)
		trebleSig[i] = 0.6 * math.Sin(2.0*math.Pi*7000.0*tVal)
	}

	pcmBass := &PCMBuffer{SampleRate: sr, Channels: 1, NumSamples: numSamples, Samples: [][]float64{bassSig}}
	pcmTreble := &PCMBuffer{SampleRate: sr, Channels: 1, NumSamples: numSamples, Samples: [][]float64{trebleSig}}

	config := DefaultSTFTConfig()
	config.NumMelBins = 80
	engine, _ := NewSTFTEngine(config)

	stftBass, _ := engine.Forward(pcmBass)
	stftTreble, _ := engine.Forward(pcmTreble)

	melBass, err := engine.ComputeMelSpectrogram(stftBass)
	if err != nil {
		t.Fatalf("ComputeMelSpectrogram bass failed: %v", err)
	}
	melTreble, err := engine.ComputeMelSpectrogram(stftTreble)
	if err != nil {
		t.Fatalf("ComputeMelSpectrogram treble failed: %v", err)
	}

	// For bass, maximum Mel energy should reside in lower bins (bin < 15)
	midFrame := melBass.NumFrames / 2
	maxBassMelBin := 0
	maxBassVal := -1.0
	for m := 0; m < 80; m++ {
		if melBass.Energies[0][midFrame][m] > maxBassVal {
			maxBassVal = melBass.Energies[0][midFrame][m]
			maxBassMelBin = m
		}
	}

	// For treble (7kHz), maximum Mel energy should reside in higher bins (bin >= 50)
	maxTrebleMelBin := 0
	maxTrebleVal := -1.0
	for m := 0; m < 80; m++ {
		if melTreble.Energies[0][midFrame][m] > maxTrebleVal {
			maxTrebleVal = melTreble.Energies[0][midFrame][m]
			maxTrebleMelBin = m
		}
	}

	t.Logf("120 Hz Peak Mel Bin: %d (Expected < 15), 7 kHz Peak Mel Bin: %d (Expected >= 50)",
		maxBassMelBin, maxTrebleMelBin)

	if maxBassMelBin >= 15 {
		t.Errorf("Bass peak Mel bin too high: expected < 15, got %d", maxBassMelBin)
	}
	if maxTrebleMelBin < 50 {
		t.Errorf("Treble peak Mel bin too low: expected >= 50, got %d", maxTrebleMelBin)
	}
}

// TestSTFT_EdgeCases tests invalid parameters and boundary error handling.
func TestSTFT_EdgeCases(t *testing.T) {
	// Non-power-of-two FFT size
	badConfig := STFTConfig{FFTSize: 1000}
	if _, err := NewSTFTEngine(badConfig); err == nil {
		t.Errorf("Expected error for non-power-of-two FFT size")
	}

	engine, _ := NewSTFTEngine(DefaultSTFTConfig())

	// Nil buffer
	if _, err := engine.Forward(nil); err == nil {
		t.Errorf("Expected error for nil buffer")
	}

	// Audio shorter than FFT size
	shortPCM := &PCMBuffer{
		SampleRate: 48000,
		Channels:   1,
		NumSamples: 100, // < 1024
		Samples:    [][]float64{make([]float64, 100)},
	}
	if _, err := engine.Forward(shortPCM); err == nil {
		t.Errorf("Expected error for buffer shorter than FFTSize")
	}

	// Nil inverse
	if _, err := engine.Inverse(nil); err == nil {
		t.Errorf("Expected error for nil inverse STFT")
	}
}

// TestSTFT_100HeterogeneousStreams validates STFT -> Mel -> iSTFT pipeline on 100 heterogeneous streams.
func TestSTFT_100HeterogeneousStreams(t *testing.T) {
	rng := rand.New(rand.NewSource(99112233))
	sampleRates := []int{22050, 44100, 48000}
	fftSizes := []int{512, 1024, 2048}

	for i := 0; i < 100; i++ {
		sr := sampleRates[rng.Intn(len(sampleRates))]
		channels := rng.Intn(2) + 1 // 1 or 2 channels
		fftSize := fftSizes[rng.Intn(len(fftSizes))]
		hopSize := fftSize / 4

		length := rng.Intn(4000) + fftSize*3 // Ensure at least 3 frames

		pcmSamples := make([][]float64, channels)
		for ch := 0; ch < channels; ch++ {
			pcmSamples[ch] = make([]float64, length)
			freq := 100.0 + rng.Float64()*3000.0
			for s := 0; s < length; s++ {
				pcmSamples[ch][s] = 0.5 * math.Sin(2.0*math.Pi*freq*float64(s)/float64(sr))
			}
		}

		pcm := &PCMBuffer{
			SampleRate: sr,
			Channels:   channels,
			NumSamples: length,
			Samples:    pcmSamples,
		}

		config := STFTConfig{
			FFTSize:    fftSize,
			HopSize:    hopSize,
			Window:     WindowHann,
			NumMelBins: 64,
		}
		engine, err := NewSTFTEngine(config)
		if err != nil {
			t.Fatalf("Stream %d engine creation failed: %v", i+1, err)
		}

		stftRes, err := engine.Forward(pcm)
		if err != nil {
			t.Fatalf("Stream %d STFT forward failed: %v", i+1, err)
		}

		melRes, err := engine.ComputeMelSpectrogram(stftRes)
		if err != nil {
			t.Fatalf("Stream %d Mel spectrogram failed: %v", i+1, err)
		}

		if melRes.NumMelBins != 64 || melRes.NumFrames != stftRes.NumFrames {
			t.Fatalf("Stream %d Mel dimension mismatch", i+1)
		}

		recPCM, err := engine.Inverse(stftRes)
		if err != nil {
			t.Fatalf("Stream %d STFT inverse failed: %v", i+1, err)
		}

		if recPCM.Channels != channels || recPCM.NumSamples <= 0 {
			t.Fatalf("Stream %d invalid reconstruction dimensions", i+1)
		}
	}

	t.Log("Successfully verified 100 heterogeneous audio streams with forward STFT, Mel spectrogram, and inverse iSTFT.")
}
