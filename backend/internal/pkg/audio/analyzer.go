// Package audio provides professional music information retrieval (MIR),
// BPM / tempo detection via spectral flux onset autocorrelation, 12-TET chromagram
// Krumhansl-Schmuckler key & mode classification, and energy/danceability/valence feature extraction.
package audio

import (
	"fmt"
	"math"
)

// MusicalMode indicates whether a key is Major or Minor.
type MusicalMode string

const (
	ModeMajor MusicalMode = "MAJOR"
	ModeMinor MusicalMode = "MINOR"
)

// PitchClassNames standard 12-tone equal temperament pitch labels.
var PitchClassNames = [12]string{
	"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B",
}

// Krumhansl-Schmuckler key profiles for Major and Minor modes.
var (
	krumhanslMajorProfile = [12]float64{
		6.35, 2.23, 3.48, 2.33, 4.38, 4.09, 2.52, 5.19, 2.39, 3.66, 2.29, 2.88,
	}
	krumhanslMinorProfile = [12]float64{
		6.33, 2.68, 3.52, 5.38, 2.60, 3.53, 2.54, 4.75, 3.98, 2.69, 3.34, 3.17,
	}
)

// AudioFeatures encapsulates comprehensive MIR music analysis metrics.
type AudioFeatures struct {
	BPM              float64     `json:"bpm"`
	BPMConfidence    float64     `json:"bpm_confidence"`
	Key              string      `json:"key"`
	Mode             MusicalMode `json:"mode"`
	KeySignature     string      `json:"key_signature"`
	KeyConfidence    float64     `json:"key_confidence"`
	Energy           float64     `json:"energy"`
	Danceability     float64     `json:"danceability"`
	Valence          float64     `json:"valence"`
	Acousticness     float64     `json:"acousticness"`
	SpectralCentroid float64     `json:"spectral_centroid"`
	EstimatedGenre   string      `json:"estimated_genre"`
	Chromagram       [12]float64 `json:"chromagram"`
}

// AudioAnalyzer extracts acoustic MIR descriptors from audio buffers.
type AudioAnalyzer struct {
	sampleRate int
}

// NewAudioAnalyzer constructs a ready-to-use audio analyzer.
func NewAudioAnalyzer() *AudioAnalyzer {
	return &AudioAnalyzer{
		sampleRate: DefaultTargetSampleRate,
	}
}

// Analyze extracts tempo, key, energy, danceability, valence, and genre from a PCMBuffer.
func (a *AudioAnalyzer) Analyze(buf *PCMBuffer) (*AudioFeatures, error) {
	if buf == nil {
		return nil, ErrNilFingerprintBuffer
	}
	if buf.Channels <= 0 || buf.NumSamples <= 0 || len(buf.Samples) == 0 {
		return nil, ErrEmptyAudioBuffer
	}

	workBuf := buf
	if buf.SampleRate != a.sampleRate {
		resampled, err := ResampleTo48kHz(buf)
		if err != nil {
			return nil, fmt.Errorf("resample to 48kHz for feature extraction: %w", err)
		}
		workBuf = resampled
	}

	channels := workBuf.Channels
	numSamples := workBuf.NumSamples
	sampleRate := workBuf.SampleRate

	// 1. Downmix to mono
	mono := make([]float64, numSamples)
	invCh := 1.0 / float64(channels)
	for ch := 0; ch < channels; ch++ {
		for n := 0; n < numSamples; n++ {
			mono[n] += workBuf.Samples[ch][n] * invCh
		}
	}

	fftSize := 2048
	hopSize := 512
	if numSamples < fftSize {
		// Short buffer fallback
		return &AudioFeatures{
			BPM:            120.0,
			Key:            "C",
			Mode:           ModeMajor,
			KeySignature:   "C Major",
			Energy:         0.5,
			Danceability:   0.5,
			Valence:        0.5,
			EstimatedGenre: "Pop",
		}, nil
	}

	numFrames := (numSamples-fftSize)/hopSize + 1
	hopRate := float64(sampleRate) / float64(hopSize)

	window := make([]float64, fftSize)
	for n := 0; n < fftSize; n++ {
		window[n] = 0.5 * (1.0 - math.Cos(2.0*math.Pi*float64(n)/float64(fftSize-1)))
	}

	real := make([]float64, fftSize)
	imag := make([]float64, fftSize)

	// Pre-allocate frame magnitude storage
	prevMag := make([]float64, fftSize/2)
	currMag := make([]float64, fftSize/2)
	spectralFlux := make([]float64, numFrames)

	var rawChroma [12]float64
	var totalCentroidSum, totalWeightSum float64
	var totalRMS float64

	binHz := float64(sampleRate) / float64(fftSize)

	for m := 0; m < numFrames; m++ {
		offset := m * hopSize
		var frameRMS float64
		for n := 0; n < fftSize; n++ {
			s := mono[offset+n]
			frameRMS += s * s
			real[n] = s * window[n]
			imag[n] = 0.0
		}
		totalRMS += math.Sqrt(frameRMS / float64(fftSize))

		fft(real, imag)

		var flux float64
		for k := 1; k < fftSize/2; k++ {
			mag := math.Sqrt(real[k]*real[k] + imag[k]*imag[k])
			currMag[k] = mag

			diff := mag - prevMag[k]
			if diff > 0.0 {
				flux += diff
			}

			// Spectral centroid accumulation
			freq := float64(k) * binHz
			totalCentroidSum += freq * mag
			totalWeightSum += mag
		}

		copy(prevMag, currMag)
		spectralFlux[m] = flux
	}

	// 2. BPM & Rhythm Analysis via Spectral Flux Autocorrelation
	bpm, bpmConfidence, danceability := estimateBPM(spectralFlux, hopRate)

	// 3. 12-TET Chromagram Extraction with 4:1 decimation (12,000 Hz) for 5.86 Hz pitch resolution
	rawChroma = extractChroma12k(mono, sampleRate)
	chromaNorm := normalizeChroma(rawChroma)
	keyName, mode, keyConfidence := estimateKeyAndMode(chromaNorm)

	// 4. Perceptual Metrics
	avgRMS := totalRMS / float64(numFrames)
	spectralCentroid := 1500.0
	if totalWeightSum > 1e-12 {
		spectralCentroid = totalCentroidSum / totalWeightSum
	}

	energy := math.Min(1.0, math.Max(0.0, 0.6*(avgRMS*3.0)+0.4*(spectralCentroid/4000.0)))
	if energy < 0.05 {
		energy = 0.05
	}

	modeBonus := 0.10
	if mode == ModeMinor {
		modeBonus = -0.10
	}
	valence := math.Min(1.0, math.Max(0.0, 0.5+modeBonus+0.2*(energy-0.5)))
	acousticness := math.Min(1.0, math.Max(0.0, 1.0-(spectralCentroid/4500.0)))

	genre := estimateGenre(bpm, energy, danceability, acousticness)

	return &AudioFeatures{
		BPM:              bpm,
		BPMConfidence:    bpmConfidence,
		Key:              keyName,
		Mode:             mode,
		KeySignature:     fmt.Sprintf("%s %s", keyName, modeCapitalized(mode)),
		KeyConfidence:    keyConfidence,
		Energy:           math.Round(energy*1000.0) / 1000.0,
		Danceability:     math.Round(danceability*1000.0) / 1000.0,
		Valence:          math.Round(valence*1000.0) / 1000.0,
		Acousticness:     math.Round(acousticness*1000.0) / 1000.0,
		SpectralCentroid: math.Round(spectralCentroid*10.0) / 10.0,
		EstimatedGenre:   genre,
		Chromagram:       chromaNorm,
	}, nil
}

// estimateBPM computes autocorrelation of onset spectral flux and extracts tempo.
func estimateBPM(flux []float64, hopRate float64) (float64, float64, float64) {
	n := len(flux)
	if n < 30 {
		return 120.0, 0.0, 0.5
	}

	// Detrend flux by subtracting mean
	var sumFlux float64
	for _, f := range flux {
		sumFlux += f
	}
	meanFlux := sumFlux / float64(n)
	detrended := make([]float64, n)
	for i := range flux {
		detrended[i] = flux[i] - meanFlux
	}

	// Range corresponding to 60 BPM to 200 BPM
	minLag := int(math.Floor(60.0 * hopRate / 200.0))
	maxLag := int(math.Ceil(60.0 * hopRate / 60.0))
	if maxLag >= n {
		maxLag = n - 1
	}
	if minLag < 2 {
		minLag = 2
	}

	var bestLag int
	var maxScore float64
	var sumAutocorr float64
	var count int

	autocorrValues := make([]float64, maxLag+1)

	for lag := minLag; lag <= maxLag; lag++ {
		var sum float64
		for i := 0; i < n-lag; i++ {
			sum += detrended[i] * detrended[i+lag]
		}
		autocorrValues[lag] = sum
		sumAutocorr += sum
		count++

		// Tempo prior weighting centered around 120 BPM
		bpmAtLag := 60.0 * hopRate / float64(lag)
		ratio := math.Log2(bpmAtLag / 120.0)
		weight := math.Exp(-0.5 * (ratio * ratio) / 0.5)

		score := sum * weight
		if score > maxScore {
			maxScore = score
			bestLag = lag
		}
	}

	if bestLag <= 0 {
		return 120.0, 0.0, 0.5
	}

	// Parabolic interpolation for sub-frame accuracy
	delta := 0.0
	if bestLag > minLag && bestLag < maxLag {
		y1 := autocorrValues[bestLag-1]
		y2 := autocorrValues[bestLag]
		y3 := autocorrValues[bestLag+1]
		denom := 2.0 * (y1 - 2.0*y2 + y3)
		if math.Abs(denom) > 1e-12 {
			delta = (y1 - y3) / denom
		}
	}

	exactLag := float64(bestLag) + delta
	bpm := math.Round((60.0*hopRate/exactLag)*10.0) / 10.0

	// Confidence and Danceability calculation
	meanAutocorr := sumAutocorr / float64(count)
	peakRatio := 0.0
	if meanAutocorr > 1e-12 {
		peakRatio = autocorrValues[bestLag] / meanAutocorr
	}
	confidence := math.Min(1.0, math.Max(0.0, (peakRatio-1.0)/4.0))
	danceability := math.Min(1.0, math.Max(0.1, (peakRatio)/5.0))

	return bpm, math.Round(confidence*100.0) / 100.0, math.Round(danceability*100.0) / 100.0
}

// normalizeChroma scales the 12-TET chroma energy vector to unit length.
func normalizeChroma(raw [12]float64) [12]float64 {
	var sumSq float64
	for _, v := range raw {
		sumSq += v * v
	}
	norm := math.Sqrt(sumSq)
	var res [12]float64
	if norm > 1e-12 {
		for i := 0; i < 12; i++ {
			res[i] = math.Round((raw[i]/norm)*1000.0) / 1000.0
		}
	} else {
		for i := 0; i < 12; i++ {
			res[i] = 1.0 / 12.0
		}
	}
	return res
}

// estimateKeyAndMode performs Pearson correlation against K-S profiles across all 24 shifts.
func estimateKeyAndMode(chroma [12]float64) (string, MusicalMode, float64) {
	bestR := -2.0
	bestPitch := 0
	bestMode := ModeMajor

	for shift := 0; shift < 12; shift++ {
		rMaj := pearsonCorrelation(chroma, rotateProfile(krumhanslMajorProfile, shift))
		if rMaj > bestR {
			bestR = rMaj
			bestPitch = shift
			bestMode = ModeMajor
		}
		rMin := pearsonCorrelation(chroma, rotateProfile(krumhanslMinorProfile, shift))
		if rMin > bestR {
			bestR = rMin
			bestPitch = shift
			bestMode = ModeMinor
		}
	}

	confidence := math.Min(1.0, math.Max(0.0, (bestR+1.0)/2.0))
	return PitchClassNames[bestPitch], bestMode, math.Round(confidence*100.0) / 100.0
}

func rotateProfile(p [12]float64, shift int) [12]float64 {
	var res [12]float64
	for i := 0; i < 12; i++ {
		res[(i+shift)%12] = p[i]
	}
	return res
}

func pearsonCorrelation(x, y [12]float64) float64 {
	var sumX, sumY float64
	for i := 0; i < 12; i++ {
		sumX += x[i]
		sumY += y[i]
	}
	meanX := sumX / 12.0
	meanY := sumY / 12.0

	var num, denX, denY float64
	for i := 0; i < 12; i++ {
		dx := x[i] - meanX
		dy := y[i] - meanY
		num += dx * dy
		denX += dx * dx
		denY += dy * dy
	}
	den := math.Sqrt(denX * denY)
	if den < 1e-12 {
		return 0.0
	}
	return num / den
}

func modeCapitalized(m MusicalMode) string {
	if m == ModeMinor {
		return "Minor"
	}
	return "Major"
}

// estimateGenre infers primary musical genre from acoustic descriptors.
func estimateGenre(bpm, energy, danceability, acousticness float64) string {
	if bpm >= 120.0 && energy >= 0.65 && danceability >= 0.60 {
		return "EDM"
	}
	if bpm <= 105.0 && danceability >= 0.60 && energy >= 0.50 {
		return "Hip-Hop"
	}
	if energy >= 0.70 && acousticness <= 0.35 {
		return "Rock"
	}
	if acousticness >= 0.70 && energy <= 0.45 {
		return "Classical"
	}
	if acousticness >= 0.50 && bpm <= 115.0 {
		return "Acoustic"
	}
	if energy <= 0.30 && danceability <= 0.35 {
		return "Ambient"
	}
	return "Pop"
}

// ExtractAudioFeatures is a top-level helper to analyze a PCMBuffer.
func ExtractAudioFeatures(buf *PCMBuffer) (*AudioFeatures, error) {
	analyzer := NewAudioAnalyzer()
	return analyzer.Analyze(buf)
}

func extractChroma12k(mono []float64, sampleRate int) [12]float64 {
	decFactor := 4
	decSamples := len(mono) / decFactor
	if decSamples < 2048 {
		var uniformChroma [12]float64
		for i := range uniformChroma {
			uniformChroma[i] = 1.0 / 12.0
		}
		return uniformChroma
	}

	decRate := sampleRate / decFactor // 12,000 Hz
	mono12k := make([]float64, decSamples)
	for i := 0; i < decSamples; i++ {
		idx := i * decFactor
		mono12k[i] = 0.25 * (mono[idx] + mono[idx+1] + mono[idx+2] + mono[idx+3])
	}

	fftSize := 2048
	hopSize := 512
	numFrames := (decSamples-fftSize)/hopSize + 1
	binHz := float64(decRate) / float64(fftSize) // 5.859375 Hz

	window := make([]float64, fftSize)
	for n := 0; n < fftSize; n++ {
		window[n] = 0.5 * (1.0 - math.Cos(2.0*math.Pi*float64(n)/float64(fftSize-1)))
	}

	real := make([]float64, fftSize)
	imag := make([]float64, fftSize)
	var rawChroma [12]float64

	for m := 0; m < numFrames; m++ {
		offset := m * hopSize
		for n := 0; n < fftSize; n++ {
			real[n] = mono12k[offset+n] * window[n]
			imag[n] = 0.0
		}
		fft(real, imag)

		for k := 1; k < fftSize/2; k++ {
			freq := float64(k) * binHz
			if freq >= 65.4 && freq <= 2093.0 {
				mag := math.Sqrt(real[k]*real[k] + imag[k]*imag[k])
				midi := 12.0*math.Log2(freq/440.0) + 69.0
				c := int(math.Floor(midi)) % 12
				if c < 0 {
					c += 12
				}
				frac := midi - math.Floor(midi)
				rawChroma[c] += mag * (1.0 - frac)
				rawChroma[(c+1)%12] += mag * frac
			}
		}
	}

	return rawChroma
}
