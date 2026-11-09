package audio

import (
	"fmt"
	"math"
)

// WindowType identifies the analysis/synthesis tapering window.
type WindowType string

const (
	WindowHann     WindowType = "HANN"
	WindowHamming  WindowType = "HAMMING"
	WindowBlackman WindowType = "BLACKMAN"
)

// STFTConfig configures parameters for STFT analysis, inversion, and Mel filterbank.
type STFTConfig struct {
	FFTSize     int        // Length of FFT (must be power of two, e.g. 512, 1024, 2048)
	HopSize     int        // Analysis hop in samples (e.g. FFTSize / 4 or FFTSize / 2)
	Window      WindowType // Window function (default: WindowHann)
	NumMelBins  int        // Number of Mel frequency bands (e.g. 80 or 128)
	MinFreqHz   float64    // Minimum frequency for Mel filterbank (default: 0.0 Hz)
	MaxFreqHz   float64    // Maximum frequency for Mel filterbank (0.0 means Nyquist)
}

// DefaultSTFTConfig returns standard 1024-point STFT with 80 Mel bins.
func DefaultSTFTConfig() STFTConfig {
	return STFTConfig{
		FFTSize:    1024,
		HopSize:    256, // 75% overlap for smooth spectrograms and perfect reconstruction
		Window:     WindowHann,
		NumMelBins: 80,
		MinFreqHz:  0.0,
		MaxFreqHz:  0.0,
	}
}

// STFTFrame stores complex spectrum and derived metrics for a single time frame.
type STFTFrame struct {
	Real      []float64 // Single-sided real components (len = FFTSize/2 + 1)
	Imag      []float64 // Single-sided imaginary components
	Magnitude []float64 // |X[k]|
	Phase     []float64 // atan2(Imag, Real)
	Power     []float64 // |X[k]|^2 / N
}

// STFTResult stores the complete multi-channel time-frequency 2D representation.
type STFTResult struct {
	SampleRate int
	Channels   int
	FFTSize    int
	HopSize    int
	NumFrames  int
	NumBins    int            // FFTSize/2 + 1
	Frames     [][]*STFTFrame // [channel][frameIndex]
}

// MelSpectrogram stores filterbank acoustic energies and log-dB representations.
type MelSpectrogram struct {
	SampleRate int
	Channels   int
	NumFrames  int
	NumMelBins int
	Energies   [][][]float64 // [channel][frameIndex][melBin]
	LogMelDB   [][][]float64 // [channel][frameIndex][melBin] in dB
}

// STFTEngine provides forward STFT, exact inverse STFT, and Mel spectrogram generation.
type STFTEngine struct {
	config     STFTConfig
	window     []float64
	numBins    int
	melFilters [][]float64 // [melBin][fftBin]
}

// NewSTFTEngine creates and initializes an STFT engine.
func NewSTFTEngine(config STFTConfig) (*STFTEngine, error) {
	if config.FFTSize < 16 || (config.FFTSize&(config.FFTSize-1)) != 0 {
		return nil, fmt.Errorf("FFTSize must be a power of two, got %d", config.FFTSize)
	}
	if config.HopSize <= 0 || config.HopSize > config.FFTSize {
		config.HopSize = config.FFTSize / 4
	}
	if config.NumMelBins <= 0 {
		config.NumMelBins = 80
	}

	win := make([]float64, config.FFTSize)
	N := float64(config.FFTSize - 1)
	switch config.Window {
	case WindowHamming:
		for i := 0; i < config.FFTSize; i++ {
			win[i] = 0.54 - 0.46*math.Cos(2.0*math.Pi*float64(i)/N)
		}
	case WindowBlackman:
		for i := 0; i < config.FFTSize; i++ {
			phi := 2.0 * math.Pi * float64(i) / N
			win[i] = 0.42 - 0.5*math.Cos(phi) + 0.08*math.Cos(2.0*phi)
		}
	default: // WindowHann
		for i := 0; i < config.FFTSize; i++ {
			win[i] = 0.5 * (1.0 - math.Cos(2.0*math.Pi*float64(i)/N))
		}
	}

	numBins := config.FFTSize/2 + 1

	return &STFTEngine{
		config:  config,
		window:  win,
		numBins: numBins,
	}, nil
}

// Forward computes the 2D Short-Time Fourier Transform of the input audio buffer.
func (e *STFTEngine) Forward(pcm *PCMBuffer) (*STFTResult, error) {
	if pcm == nil {
		return nil, ErrNilPCMBuffer
	}
	if len(pcm.Samples) == 0 || pcm.Channels == 0 {
		return nil, ErrEmptyPCMBuffer
	}

	numSamples := pcm.NumSamples
	fftSize := e.config.FFTSize
	hopSize := e.config.HopSize

	if numSamples < fftSize {
		return nil, fmt.Errorf("audio length %d is shorter than FFTSize %d", numSamples, fftSize)
	}

	numFrames := (numSamples-fftSize)/hopSize + 1
	numChannels := pcm.Channels

	channelFrames := make([][]*STFTFrame, numChannels)

	for ch := 0; ch < numChannels; ch++ {
		channelFrames[ch] = make([]*STFTFrame, numFrames)
		samples := pcm.Samples[ch]

		for f := 0; f < numFrames; f++ {
			startIdx := f * hopSize

			// Window input frame
			realBuf := make([]float64, fftSize)
			imagBuf := make([]float64, fftSize)
			for i := 0; i < fftSize; i++ {
				realBuf[i] = samples[startIdx+i] * e.window[i]
			}

			// In-place Cooley-Tukey Radix-2 FFT
			fftInPlace(realBuf, imagBuf, false)

			// Extract single-sided spectrum (k = 0 to fftSize/2)
			fReal := make([]float64, e.numBins)
			fImag := make([]float64, e.numBins)
			fMag := make([]float64, e.numBins)
			fPhase := make([]float64, e.numBins)
			fPower := make([]float64, e.numBins)

			invN := 1.0 / float64(fftSize)

			for k := 0; k < e.numBins; k++ {
				r := realBuf[k]
				im := imagBuf[k]
				mag := math.Hypot(r, im)

				fReal[k] = r
				fImag[k] = im
				fMag[k] = mag
				fPhase[k] = math.Atan2(im, r)
				fPower[k] = (mag * mag) * invN
			}

			channelFrames[ch][f] = &STFTFrame{
				Real:      fReal,
				Imag:      fImag,
				Magnitude: fMag,
				Phase:     fPhase,
				Power:     fPower,
			}
		}
	}

	return &STFTResult{
		SampleRate: pcm.SampleRate,
		Channels:   numChannels,
		FFTSize:    fftSize,
		HopSize:    hopSize,
		NumFrames:  numFrames,
		NumBins:    e.numBins,
		Frames:     channelFrames,
	}, nil
}

// Inverse reconstructs time-domain PCM samples from complex STFT representation with perfect WOLA.
func (e *STFTEngine) Inverse(stft *STFTResult) (*PCMBuffer, error) {
	if stft == nil || len(stft.Frames) == 0 {
		return nil, fmt.Errorf("invalid STFT result for inverse reconstruction")
	}

	numChannels := stft.Channels
	numFrames := stft.NumFrames
	fftSize := stft.FFTSize
	hopSize := stft.HopSize

	totalSamples := (numFrames-1)*hopSize + fftSize

	outputSamples := make([][]float64, numChannels)
	windowWeights := make([]float64, totalSamples)

	for ch := 0; ch < numChannels; ch++ {
		outputSamples[ch] = make([]float64, totalSamples)
	}

	// Accumulate synthesis window weights across frames
	for f := 0; f < numFrames; f++ {
		startIdx := f * hopSize
		for i := 0; i < fftSize; i++ {
			windowWeights[startIdx+i] += e.window[i] * e.window[i]
		}
	}

	for ch := 0; ch < numChannels; ch++ {
		for f := 0; f < numFrames; f++ {
			frame := stft.Frames[ch][f]
			startIdx := f * hopSize

			// Reconstruct full conjugate-symmetric two-sided spectrum
			realBuf := make([]float64, fftSize)
			imagBuf := make([]float64, fftSize)

			for k := 0; k < e.numBins; k++ {
				realBuf[k] = frame.Real[k]
				imagBuf[k] = frame.Imag[k]
			}
			for k := 1; k < fftSize/2; k++ {
				realBuf[fftSize-k] = frame.Real[k]
				imagBuf[fftSize-k] = -frame.Imag[k] // Complex conjugate
			}

			// In-place iFFT
			fftInPlace(realBuf, imagBuf, true)

			// Overlap-add multiplied by synthesis window
			for i := 0; i < fftSize; i++ {
				outputSamples[ch][startIdx+i] += realBuf[i] * e.window[i]
			}
		}

		// Normalize by accumulated squared window weights
		for i := 0; i < totalSamples; i++ {
			wSum := windowWeights[i]
			if wSum > 1e-8 {
				outputSamples[ch][i] /= wSum
			}
		}
	}

	return &PCMBuffer{
		SampleRate: stft.SampleRate,
		Channels:   numChannels,
		NumSamples: totalSamples,
		Samples:    outputSamples,
	}, nil
}

// ComputeMelSpectrogram projects linear frequency STFT power spectra onto psychoacoustic Mel bands.
func (e *STFTEngine) ComputeMelSpectrogram(stft *STFTResult) (*MelSpectrogram, error) {
	if stft == nil {
		return nil, fmt.Errorf("stft result is nil")
	}

	sr := float64(stft.SampleRate)
	numMel := e.config.NumMelBins
	minFreq := e.config.MinFreqHz
	maxFreq := e.config.MaxFreqHz
	if maxFreq <= 0.0 || maxFreq > sr/2.0 {
		maxFreq = sr / 2.0
	}

	// Build triangular Mel filterbank if not precomputed or parameters changed
	if len(e.melFilters) != numMel {
		e.melFilters = buildMelFilterbank(numMel, e.numBins, sr, minFreq, maxFreq)
	}

	numChannels := stft.Channels
	numFrames := stft.NumFrames

	energies := make([][][]float64, numChannels)
	logMelDB := make([][][]float64, numChannels)

	const minPower = 1e-10 // -100 dB floor

	for ch := 0; ch < numChannels; ch++ {
		energies[ch] = make([][]float64, numFrames)
		logMelDB[ch] = make([][]float64, numFrames)

		for f := 0; f < numFrames; f++ {
			frame := stft.Frames[ch][f]
			melFrame := make([]float64, numMel)
			logFrame := make([]float64, numMel)

			for m := 0; m < numMel; m++ {
				var sum float64
				filter := e.melFilters[m]
				for k := 0; k < e.numBins; k++ {
					sum += filter[k] * frame.Power[k]
				}
				melFrame[m] = sum

				val := sum
				if val < minPower {
					val = minPower
				}
				logFrame[m] = 10.0 * math.Log10(val)
			}

			energies[ch][f] = melFrame
			logMelDB[ch][f] = logFrame
		}
	}

	return &MelSpectrogram{
		SampleRate: stft.SampleRate,
		Channels:   numChannels,
		NumFrames:  numFrames,
		NumMelBins: numMel,
		Energies:   energies,
		LogMelDB:   logMelDB,
	}, nil
}

// fftInPlace performs radix-2 Cooley-Tukey FFT (inverse=false) or iFFT (inverse=true).
func fftInPlace(real, imag []float64, inverse bool) {
	n := len(real)

	// Bit-reversal permutation
	j := 0
	for i := 0; i < n-1; i++ {
		if i < j {
			real[i], real[j] = real[j], real[i]
			imag[i], imag[j] = imag[j], imag[i]
		}
		k := n / 2
		for k <= j {
			j -= k
			k /= 2
		}
		j += k
	}

	// Butterfly computation loops
	sign := -1.0
	if inverse {
		sign = 1.0
	}

	for length := 2; length <= n; length <<= 1 {
		halfLen := length / 2
		angle := sign * 2.0 * math.Pi / float64(length)
		wStepR := math.Cos(angle)
		wStepI := math.Sin(angle)

		for i := 0; i < n; i += length {
			wR := 1.0
			wI := 0.0

			for k := 0; k < halfLen; k++ {
				evenIdx := i + k
				oddIdx := i + k + halfLen

				uR := real[evenIdx]
				uI := imag[evenIdx]

				vR := real[oddIdx]*wR - imag[oddIdx]*wI
				vI := real[oddIdx]*wI + imag[oddIdx]*wR

				real[evenIdx] = uR + vR
				imag[evenIdx] = uI + vI
				real[oddIdx] = uR - vR
				imag[oddIdx] = uI - vI

				nextWR := wR*wStepR - wI*wStepI
				nextWI := wR*wStepI + wI*wStepR
				wR = nextWR
				wI = nextWI
			}
		}
	}

	// Scale by 1/N for iFFT
	if inverse {
		invN := 1.0 / float64(n)
		for i := 0; i < n; i++ {
			real[i] *= invN
			imag[i] *= invN
		}
	}
}

// buildMelFilterbank creates Slaney-normalized triangular Mel filters.
func buildMelFilterbank(numMel, numBins int, sampleRate, minFreq, maxFreq float64) [][]float64 {
	hzToMel := func(hz float64) float64 {
		return 2595.0 * math.Log10(1.0+hz/700.0)
	}
	melToHz := func(mel float64) float64 {
		return 700.0 * (math.Pow(10.0, mel/2595.0) - 1.0)
	}

	minMel := hzToMel(minFreq)
	maxMel := hzToMel(maxFreq)

	// Points in Mel space: numMel + 2 points for centers and boundary shoulders
	melPoints := make([]float64, numMel+2)
	step := (maxMel - minMel) / float64(numMel+1)
	for i := 0; i < numMel+2; i++ {
		melPoints[i] = minMel + float64(i)*step
	}

	// Map Mel points to Hz and then to FFT bin indices
	binPoints := make([]float64, numMel+2)
	fftMaxIdx := float64(numBins - 1)
	for i := 0; i < numMel+2; i++ {
		hz := melToHz(melPoints[i])
		binIdx := hz * (2.0 * fftMaxIdx) / sampleRate
		if binIdx > fftMaxIdx {
			binIdx = fftMaxIdx
		}
		binPoints[i] = binIdx
	}

	filters := make([][]float64, numMel)
	for m := 0; m < numMel; m++ {
		filters[m] = make([]float64, numBins)
		left := binPoints[m]
		center := binPoints[m+1]
		right := binPoints[m+2]

		for k := 0; k < numBins; k++ {
			fk := float64(k)
			if fk >= left && fk <= center && center > left {
				filters[m][k] = (fk - left) / (center - left)
			} else if fk > center && fk <= right && right > center {
				filters[m][k] = (right - fk) / (right - center)
			}
		}

		// Slaney area normalization (2.0 / (right_hz - left_hz))
		hzWidth := melToHz(melPoints[m+2]) - melToHz(melPoints[m])
		if hzWidth > 1e-6 {
			norm := 2.0 / hzWidth
			for k := 0; k < numBins; k++ {
				filters[m][k] *= norm
			}
		}
	}

	return filters
}
