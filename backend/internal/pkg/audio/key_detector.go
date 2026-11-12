// Package audio provides pure Go chromagram analysis, pitch class profile correlation,
// musical key (tonality) estimation, and Camelot wheel harmonic mixing mapping.
package audio

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

// ScaleType represents the musical tonality (Major or Minor).
type ScaleType string

const (
	ScaleMajor ScaleType = "Major"
	ScaleMinor ScaleType = "Minor"
)

// ProfileType specifies the cognitive tonal profile model for correlation analysis.
type ProfileType string

const (
	ProfileKrumhansl ProfileType = "KRUMHANSL"
	ProfileTemperley ProfileType = "TEMPERLEY"
)

// KeyCandidate stores rank-ordered key candidate with correlation score.
type KeyCandidate struct {
	Key         string    `json:"key"`         // e.g. "C Major", "A Minor"
	Root        string    `json:"root"`        // e.g. "C", "A"
	Scale       ScaleType `json:"scale"`       // Major or Minor
	Camelot     string    `json:"camelot"`     // e.g. "8B", "8A"
	OpenKey     string    `json:"open_key"`    // e.g. "1d", "1m"
	Correlation float64   `json:"correlation"` // Pearson r (-1.0 to 1.0)
	Confidence  float64   `json:"confidence"`  // Normalized confidence (0.0 to 1.0)
}

// KeyResult contains the complete musical key detection, Camelot wheel, and harmonic mixing information.
type KeyResult struct {
	Key                string         `json:"key"`                 // Primary detected key (e.g. "C Major")
	Root               string         `json:"root"`                // Root note (e.g. "C")
	Scale              ScaleType      `json:"scale"`               // "Major" or "Minor"
	Camelot            string         `json:"camelot"`             // Camelot wheel code (e.g. "8B")
	OpenKey            string         `json:"open_key"`            // Open Key notation (e.g. "1d")
	Confidence         float64        `json:"confidence"`          // Detection confidence (0.0 to 1.0)
	ChromaDistribution [12]float64    `json:"chroma_distribution"` // 12-semitone normalized energy histogram
	CompatibleCamelot  []string       `json:"compatible_camelot"`  // Harmonically compatible Camelot codes
	KeyRankings        []KeyCandidate `json:"key_rankings"`        // Top candidate keys
}

// KeyDetectorConfig configures frequency boundaries, STFT parameters, and profile weighting.
type KeyDetectorConfig struct {
	MinFreqHz       float64     // Minimum frequency for pitch analysis (default: 65.4 Hz - C2)
	MaxFreqHz       float64     // Maximum frequency for pitch analysis (default: 2093.0 Hz - C7)
	FFTSize         int         // STFT frame size (default: 2048)
	HopSize         int         // STFT hop size (default: 512)
	Profile         ProfileType // Profile template model (default: ProfileKrumhansl)
	TuningRefFreqHz float64     // Concert pitch reference frequency for A4 (default: 440.0 Hz)
}

// DefaultKeyDetectorConfig returns production defaults for musical tonality analysis.
func DefaultKeyDetectorConfig() KeyDetectorConfig {
	return KeyDetectorConfig{
		MinFreqHz:       65.4,   // C2
		MaxFreqHz:       2093.0, // C7
		FFTSize:         4096,
		HopSize:         1024,
		Profile:         ProfileKrumhansl,
		TuningRefFreqHz: 440.0,
	}
}

// Key detection diagnostic errors.
var (
	ErrAudioTooShortForKey = errors.New("audio duration is too short for key detection (minimum 0.5s required)")
	ErrSilentAudio         = errors.New("insufficient harmonic spectral energy or silent audio signal")
	ErrInvalidConfig       = errors.New("invalid key detector configuration parameters")
)

// KeyDetector performs chromagram analysis and key correlation estimation.
type KeyDetector struct {
	config KeyDetectorConfig
}

// NewKeyDetector creates a new musical key detection engine.
func NewKeyDetector(config KeyDetectorConfig) (*KeyDetector, error) {
	if config.MinFreqHz <= 0 || config.MaxFreqHz <= config.MinFreqHz {
		return nil, ErrInvalidConfig
	}
	if config.FFTSize < 512 || (config.FFTSize&(config.FFTSize-1)) != 0 {
		return nil, errors.New("FFTSize must be a power of two >= 512")
	}
	if config.HopSize <= 0 {
		config.HopSize = config.FFTSize / 4
	}
	if config.TuningRefFreqHz <= 0 {
		config.TuningRefFreqHz = 440.0
	}
	if config.Profile == "" {
		config.Profile = ProfileKrumhansl
	}
	return &KeyDetector{config: config}, nil
}

// Krumhansl-Schmuckler Key Profiles (1990)
var (
	krumhanslMajor = [12]float64{6.35, 2.23, 3.48, 2.33, 4.38, 4.09, 2.52, 5.19, 2.39, 3.66, 2.29, 2.88}
	krumhanslMinor = [12]float64{6.33, 2.68, 3.52, 5.38, 2.60, 3.53, 2.54, 4.75, 3.98, 2.69, 3.34, 3.17}
)

// Temperley Key Profiles (1999)
var (
	temperleyMajor = [12]float64{5.0, 2.0, 3.5, 2.0, 4.5, 4.0, 2.0, 4.5, 2.0, 3.5, 1.5, 4.0}
	temperleyMinor = [12]float64{5.0, 2.0, 3.5, 4.5, 2.0, 4.0, 2.0, 4.5, 3.5, 2.0, 1.5, 4.0}
)

// Camelot & OpenKey mapping table: [pitchClassIndex 0..11][scale: 0=Major, 1=Minor]
type keyMeta struct {
	camelot string
	openKey string
}

var keyMetadataTable = [12][2]keyMeta{
	0:  {{"8B", "1d"}, {"5A", "10m"}},  // C Major, C Minor
	1:  {{"3B", "8d"}, {"12A", "5m"}},  // C# Major, C# Minor
	2:  {{"10B", "3d"}, {"7A", "12m"}}, // D Major, D Minor
	3:  {{"5B", "10d"}, {"2A", "7m"}},  // D# Major, D# Minor
	4:  {{"12B", "5d"}, {"9A", "2m"}},  // E Major, E Minor
	5:  {{"7B", "12d"}, {"4A", "9m"}},  // F Major, F Minor
	6:  {{"2B", "7d"}, {"11A", "4m"}},  // F# Major, F# Minor
	7:  {{"9B", "2d"}, {"6A", "11m"}},  // G Major, G Minor
	8:  {{"4B", "9d"}, {"1A", "6m"}},   // G# Major, G# Minor
	9:  {{"11B", "4d"}, {"8A", "1m"}},  // A Major, A Minor
	10: {{"6B", "11d"}, {"3A", "8m"}},  // A# Major, A# Minor
	11: {{"1B", "6d"}, {"10A", "3m"}},  // B Major, B Minor
}

// GetCamelotKey returns the Camelot wheel code and OpenKey code for given root and scale.
func GetCamelotKey(root string, scale ScaleType) (string, string) {
	rootIdx := RootNameToPitchClass(root)
	if rootIdx < 0 {
		return "8B", "1d"
	}
	sIdx := 0
	if scale == ScaleMinor {
		sIdx = 1
	}
	meta := keyMetadataTable[rootIdx][sIdx]
	return meta.camelot, meta.openKey
}

// GetCompatibleCamelotKeys generates harmonically compatible Camelot codes for seamless mixing.
func GetCompatibleCamelotKeys(camelot string) []string {
	if len(camelot) < 2 {
		return []string{camelot}
	}

	letter := camelot[len(camelot)-1:]
	numStr := camelot[:len(camelot)-1]

	var num int
	_, err := fmt.Sscanf(numStr, "%d", &num)
	if err != nil || num < 1 || num > 12 {
		return []string{camelot}
	}

	oppositeLetter := "A"
	if letter == "A" {
		oppositeLetter = "B"
	}

	// Calculate adjacent hours on circle of fifths (1..12)
	dominantNum := (num % 12) + 1
	subdominantNum := ((num - 2 + 12) % 12) + 1
	energyBoostNum := ((num + 1) % 12) + 1

	return []string{
		fmt.Sprintf("%d%s", num, letter),            // Exact Match (Same Key)
		fmt.Sprintf("%d%s", num, oppositeLetter),    // Relative Major/Minor
		fmt.Sprintf("%d%s", subdominantNum, letter), // Subdominant (-1 hour)
		fmt.Sprintf("%d%s", dominantNum, letter),    // Dominant (+1 hour)
		fmt.Sprintf("%d%s", energyBoostNum, letter), // Energy Boost (+2 hours)
	}
}

// RootNameToPitchClass converts standard note strings (e.g. "C", "C#", "Db", "Bb") to pitch class 0..11.
func RootNameToPitchClass(name string) int {
	clean := strings.TrimSpace(name)
	switch strings.ToUpper(clean) {
	case "C", "B#":
		return 0
	case "C#", "DB":
		return 1
	case "D":
		return 2
	case "D#", "EB":
		return 3
	case "E", "FB":
		return 4
	case "F", "E#":
		return 5
	case "F#", "GB":
		return 6
	case "G":
		return 7
	case "G#", "AB":
		return 8
	case "A":
		return 9
	case "A#", "BB":
		return 10
	case "B", "CB":
		return 11
	default:
		return -1
	}
}

// Analyze performs chromagram extraction and tonality correlation on the PCM audio buffer.
func (d *KeyDetector) Analyze(pcm *PCMBuffer) (*KeyResult, error) {
	if pcm == nil {
		return nil, ErrNilPCMBuffer
	}
	if pcm.NumSamples == 0 || len(pcm.Samples) == 0 {
		return nil, ErrEmptyAudioBuffer
	}

	durationSec := float64(pcm.NumSamples) / float64(pcm.SampleRate)
	if durationSec < 0.5 {
		return nil, ErrAudioTooShortForKey
	}

	sampleRate := pcm.SampleRate
	fftSize := d.config.FFTSize
	hopSize := d.config.HopSize

	// 1. Downmix to mono if multi-channel
	mono := make([]float64, pcm.NumSamples)
	numChannels := pcm.Channels
	invChannels := 1.0 / float64(numChannels)

	for ch := 0; ch < numChannels; ch++ {
		chSamples := pcm.Samples[ch]
		for n := 0; n < pcm.NumSamples; n++ {
			mono[n] += chSamples[n] * invChannels
		}
	}

	// 2. Perform STFT Analysis
	stftEngine, err := NewSTFTEngine(STFTConfig{
		FFTSize:    fftSize,
		HopSize:    hopSize,
		Window:     WindowHann,
		NumMelBins: 80,
	})
	if err != nil {
		return nil, fmt.Errorf("initializing stft engine: %w", err)
	}

	monoPCM := &PCMBuffer{
		Samples:    [][]float64{mono},
		Channels:   1,
		SampleRate: sampleRate,
		NumSamples: pcm.NumSamples,
	}

	stftRes, err := stftEngine.Forward(monoPCM)
	if err != nil {
		return nil, fmt.Errorf("executing stft: %w", err)
	}

	numFrames := stftRes.NumFrames
	numBins := stftRes.NumBins
	freqResolution := float64(sampleRate) / float64(fftSize)
	refFreq := d.config.TuningRefFreqHz

	// 3. Accumulate 12-Semitone Chromagram Histogram via STFT Peak Interpolation
	var chromaSum [12]float64
	var totalChromaEnergy float64 = 0.0

	minFreq := d.config.MinFreqHz
	maxFreq := d.config.MaxFreqHz

	for f := 0; f < numFrames; f++ {
		frame := stftRes.Frames[0][f]

		for k := 2; k < numBins-2; k++ {
			mag := frame.Magnitude[k]
			if mag <= 1e-5 {
				continue
			}

			// Local spectral peak detection
			if mag > frame.Magnitude[k-1] && mag > frame.Magnitude[k+1] {
				y0 := frame.Magnitude[k-1]
				y1 := mag
				y2 := frame.Magnitude[k+1]

				// Parabolic interpolation for sub-bin frequency accuracy
				denom := 2.0 * (2.0*y1 - y0 - y2)
				delta := 0.0
				if math.Abs(denom) > 1e-12 {
					delta = (y2 - y0) / denom
				}

				trueBin := float64(k) + delta
				trueFreq := trueBin * freqResolution

				if trueFreq < minFreq || trueFreq > maxFreq {
					continue
				}

				// Map sub-bin frequency to fractional MIDI pitch number:
				// m = 12 * log2(trueFreq / refFreq) + 69 (where A4=69, refFreq=440Hz)
				midiPitch := 12.0*math.Log2(trueFreq/refFreq) + 69.0
				nearestSemitone := math.Round(midiPitch)
				semitoneDiff := midiPitch - nearestSemitone

				if math.Abs(semitoneDiff) <= 0.48 {
					// Gaussian weighting across semitone window
					weight := math.Exp(-0.5 * (semitoneDiff / 0.25) * (semitoneDiff / 0.25))
					pitchClass := int(math.Mod(math.Mod(nearestSemitone, 12.0)+12.0, 12.0))

					energy := math.Log1p(100.0*y1) * weight
					chromaSum[pitchClass] += energy
					totalChromaEnergy += energy
				}
			}
		}
	}

	if totalChromaEnergy <= 1e-5 {
		return nil, ErrSilentAudio
	}

	// Normalize Chroma Distribution (L1 norm so sum = 1.0)
	var normChroma [12]float64
	invTotal := 1.0 / totalChromaEnergy
	for i := 0; i < 12; i++ {
		normChroma[i] = chromaSum[i] * invTotal
	}

	// 4. Correlate with Musical Key Profiles (Major and Minor for all 12 roots)
	var majorProfile [12]float64
	var minorProfile [12]float64

	if d.config.Profile == ProfileTemperley {
		majorProfile = temperleyMajor
		minorProfile = temperleyMinor
	} else {
		majorProfile = krumhanslMajor
		minorProfile = krumhanslMinor
	}

	var candidates []KeyCandidate

	// Check 12 Major and 12 Minor keys
	for rootIdx := 0; rootIdx < 12; rootIdx++ {
		rootName := PitchClassNames[rootIdx]

		// Major correlation
		rotatedMaj := rotateProfile(majorProfile, rootIdx)
		rMajor := pearsonCorrelation(normChroma, rotatedMaj)
		camelotMaj, openKeyMaj := GetCamelotKey(rootName, ScaleMajor)
		candidates = append(candidates, KeyCandidate{
			Key:         fmt.Sprintf("%s Major", rootName),
			Root:        rootName,
			Scale:       ScaleMajor,
			Camelot:     camelotMaj,
			OpenKey:     openKeyMaj,
			Correlation: rMajor,
		})

		// Minor correlation
		rotatedMin := rotateProfile(minorProfile, rootIdx)
		rMinor := pearsonCorrelation(normChroma, rotatedMin)
		camelotMin, openKeyMin := GetCamelotKey(rootName, ScaleMinor)
		candidates = append(candidates, KeyCandidate{
			Key:         fmt.Sprintf("%s Minor", rootName),
			Root:        rootName,
			Scale:       ScaleMinor,
			Camelot:     camelotMin,
			OpenKey:     openKeyMin,
			Correlation: rMinor,
		})
	}

	// Sort candidates by correlation descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Correlation > candidates[j].Correlation
	})

	// Normalize candidate confidence scores
	topCorr := candidates[0].Correlation
	secondCorr := candidates[1].Correlation
	margin := topCorr - secondCorr

	confidence := math.Min(1.0, math.Max(0.0, (topCorr+1.0)/2.0*(1.0+margin*1.5)))

	for i := range candidates {
		candidates[i].Confidence = math.Min(1.0, math.Max(0.0, (candidates[i].Correlation+1.0)/2.0))
	}

	best := candidates[0]
	compatible := GetCompatibleCamelotKeys(best.Camelot)

	return &KeyResult{
		Key:                best.Key,
		Root:               best.Root,
		Scale:              best.Scale,
		Camelot:            best.Camelot,
		OpenKey:            best.OpenKey,
		Confidence:         confidence,
		ChromaDistribution: normChroma,
		CompatibleCamelot:  compatible,
		KeyRankings:        candidates,
	}, nil
}

// GenerateMusicalKeyTestSignal synthesizes a rich multi-octave harmonic test signal in the specified key.
func GenerateMusicalKeyTestSignal(rootName string, scale ScaleType, durationSec float64, sampleRate, channels int) *PCMBuffer {
	if sampleRate <= 0 {
		sampleRate = 48000
	}
	if channels <= 0 {
		channels = 2
	}
	if durationSec <= 0 {
		durationSec = 2.0
	}

	numSamples := int(float64(sampleRate) * durationSec)
	samples := make([][]float64, channels)
	for ch := 0; ch < channels; ch++ {
		samples[ch] = make([]float64, numSamples)
	}

	rootPitch := RootNameToPitchClass(rootName)
	if rootPitch < 0 {
		rootPitch = 0 // Default to C
	}

	// Diatonic musical scale intervals and perceptual weights:
	// Major scale: Root (0), M2 (2), M3 (4), P4 (5), P5 (7), M6 (9), M7 (11), Octave (12)
	// Minor scale: Root (0), M2 (2), m3 (3), P4 (5), P5 (7), m6 (8), m7 (10), Octave (12)
	var semitones []int
	var weights []float64

	if scale == ScaleMinor {
		semitones = []int{0, 2, 3, 5, 7, 8, 10, 12}
		weights = []float64{1.0, 0.20, 0.65, 0.20, 0.55, 0.20, 0.20, 0.35}
	} else {
		semitones = []int{0, 2, 4, 5, 7, 9, 11, 12}
		weights = []float64{1.0, 0.20, 0.65, 0.20, 0.55, 0.20, 0.15, 0.35}
	}

	// Fundamental base note: Octave 4 (e.g. C4 = MIDI 60, A4 = MIDI 69)
	baseMIDI := 60 + rootPitch

	for n := 0; n < numSamples; n++ {
		t := float64(n) / float64(sampleRate)
		var val float64 = 0.0

		for idx, semi := range semitones {
			midi := float64(baseMIDI + semi)
			freq := 440.0 * math.Pow(2.0, (midi-69.0)/12.0)
			w := weights[idx]

			// Fundamental note
			val += w * 0.40 * math.Sin(2.0*math.Pi*freq*t)
			// Natural instrument 2nd harmonic (octave overtone)
			val += w * 0.15 * math.Sin(2.0*math.Pi*(2.0*freq)*t)
		}

		// Gentle envelope (50ms fade-in, 50ms fade-out)
		fadeSec := 0.05
		var env float64 = 1.0
		if t < fadeSec {
			env = 0.5 * (1.0 - math.Cos(math.Pi*t/fadeSec))
		} else if t > durationSec-fadeSec {
			tOut := durationSec - t
			env = 0.5 * (1.0 - math.Cos(math.Pi*tOut/fadeSec))
		}

		// Clamp amplitude
		sampleVal := math.Max(-0.95, math.Min(0.95, val*0.25*env))
		for ch := 0; ch < channels; ch++ {
			// Subtle stereo spread if stereo
			chVal := sampleVal
			if channels == 2 && ch == 1 {
				chVal *= 0.98
			}
			samples[ch][n] = chVal
		}
	}

	return &PCMBuffer{
		Samples:    samples,
		Channels:   channels,
		SampleRate: sampleRate,
		NumSamples: numSamples,
	}
}
