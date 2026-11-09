package audio

import (
	"math"
)

// SpatialSource defines the 3D spherical coordinates of an audio source relative to the listener.
type SpatialSource struct {
	AzimuthDeg     float64 // Horizontal angle: -180.0 to +180.0 (0=front, +90=right, -90=left, +/-180=behind)
	ElevationDeg   float64 // Vertical angle: -45.0 to +90.0 (0=horizon, +90=overhead, -45=below)
	DistanceMeters float64 // Distance in meters (0.1 to 100.0)
}

// BinauralConfig holds physical acoustic parameters for 3D HRTF rendering.
type BinauralConfig struct {
	HeadRadiusMeters          float64 // Average human head radius (default: 0.0875 m)
	SoundSpeedMetersPerSec    float64 // Speed of sound in standard air (default: 343.0 m/s)
	EnablePinnaNotch          bool    // Enable elevation-dependent pinna spectral notch filter
	EnableAirAbsorption       bool    // Enable high-frequency air dampening over distance
	EnableDistanceAttenuation bool    // Enable 1/r inverse distance sound decay
	MinDistanceMeters         float64 // Near-field distance clamp to prevent infinite gain (default: 0.2 m)
	ReferenceDistanceMeters   float64 // Reference distance where gain is 0 dB (default: 1.0 m)
}

// DefaultBinauralConfig returns production-standard acoustic physics parameters.
func DefaultBinauralConfig() BinauralConfig {
	return BinauralConfig{
		HeadRadiusMeters:          0.0875,
		SoundSpeedMetersPerSec:    343.0,
		EnablePinnaNotch:          true,
		EnableAirAbsorption:       true,
		EnableDistanceAttenuation: true,
		MinDistanceMeters:         0.2,
		ReferenceDistanceMeters:   1.0,
	}
}

// BinauralRenderer converts mono/stereo audio into realistic 3D binaural headphone audio.
type BinauralRenderer struct {
	config BinauralConfig
}

// NewBinauralRenderer creates a new 3D spatial binaural audio renderer.
func NewBinauralRenderer(config BinauralConfig) *BinauralRenderer {
	if config.HeadRadiusMeters <= 0.01 {
		config.HeadRadiusMeters = 0.0875
	}
	if config.SoundSpeedMetersPerSec <= 10.0 {
		config.SoundSpeedMetersPerSec = 343.0
	}
	if config.MinDistanceMeters <= 0.01 {
		config.MinDistanceMeters = 0.2
	}
	if config.ReferenceDistanceMeters <= 0.01 {
		config.ReferenceDistanceMeters = 1.0
	}
	return &BinauralRenderer{config: config}
}

// ComputeITD calculates Interaural Time Difference (in seconds) for Left and Right ears using Woodworth's model.
// Positive delay means the sound reaches that ear later.
func (r *BinauralRenderer) ComputeITD(azimuthDeg, elevationDeg float64) (delayLeftSec, delayRightSec float64) {
	// Normalize angles
	az := normalizeAzimuth(azimuthDeg)
	el := clampElevation(elevationDeg)

	azRad := az * (math.Pi / 180.0)
	elRad := el * (math.Pi / 180.0)

	// Effective azimuth accounting for elevation angle projection
	sinEff := math.Sin(azRad) * math.Cos(elRad)
	thetaEff := math.Asin(math.Max(-1.0, math.Min(1.0, sinEff)))

	a := r.config.HeadRadiusMeters
	c := r.config.SoundSpeedMetersPerSec

	absTheta := math.Abs(thetaEff)
	// Woodworth formula: deltaT = (a / c) * (sin|theta| + |theta|)
	itdSec := (a / c) * (math.Sin(absTheta) + absTheta)

	if thetaEff > 0 {
		// Sound is to the right: Left ear is delayed (contralateral)
		return itdSec, 0.0
	} else if thetaEff < 0 {
		// Sound is to the left: Right ear is delayed (contralateral)
		return 0.0, itdSec
	}
	return 0.0, 0.0
}

// ComputeILD calculates Interaural Level Difference linear gain factors for Left and Right ears.
func (r *BinauralRenderer) ComputeILD(azimuthDeg, elevationDeg float64) (gainLeft, gainRight float64) {
	az := normalizeAzimuth(azimuthDeg)
	el := clampElevation(elevationDeg)

	azRad := az * (math.Pi / 180.0)
	elRad := el * (math.Pi / 180.0)

	sinVal := math.Sin(azRad) * math.Cos(elRad)

	// Head shadowing model:
	// Near ear experiences a gentle acoustic boost (+2.5 dB)
	// Far ear experiences acoustic shadowing attenuation (up to -9.0 dB at 90 degrees)
	// sinVal = +1.0 (Right side): Left is shadowed (-9dB), Right is near (+2.5dB)
	// sinVal = -1.0 (Left side): Right is shadowed (-9dB), Left is near (+2.5dB)

	if sinVal >= 0 {
		// Right is ipsilateral, Left is contralateral
		gNearDB := 2.5 * sinVal
		gFarDB := -9.0 * sinVal
		gainRight = math.Pow(10.0, gNearDB/20.0)
		gainLeft = math.Pow(10.0, gFarDB/20.0)
	} else {
		// Left is ipsilateral, Right is contralateral
		s := -sinVal
		gNearDB := 2.5 * s
		gFarDB := -9.0 * s
		gainLeft = math.Pow(10.0, gNearDB/20.0)
		gainRight = math.Pow(10.0, gFarDB/20.0)
	}
	return gainLeft, gainRight
}

// Render spatialize a mono or stereo audio buffer to binaural 3D stereo (Left=0, Right=1).
func (r *BinauralRenderer) Render(pcm *PCMBuffer, source SpatialSource) (*PCMBuffer, error) {
	if pcm == nil {
		return nil, ErrNilPCMBuffer
	}
	if len(pcm.Samples) == 0 || pcm.Channels == 0 {
		return nil, ErrEmptyPCMBuffer
	}

	sr := float64(pcm.SampleRate)
	numSamples := pcm.NumSamples

	// 1. Downmix input to mono sound source
	sourceMono := downmixToMono(pcm.Samples)

	// 2. Distance Attenuation (Inverse Square Law)
	dist := source.DistanceMeters
	if dist < r.config.MinDistanceMeters {
		dist = r.config.MinDistanceMeters
	}

	distanceGain := 1.0
	if r.config.EnableDistanceAttenuation {
		distanceGain = r.config.ReferenceDistanceMeters / dist
		if distanceGain > 5.0 {
			distanceGain = 5.0
		}
	}

	// 3. Air Absorption (atmospheric damping: 1st-order IIR lowpass)
	attenuatedSource := make([]float64, numSamples)
	if r.config.EnableAirAbsorption && dist > 1.0 {
		// Cutoff drops with distance: 20kHz down to 2.5kHz at 100m
		cutoff := 20000.0 / (1.0 + 0.07*dist)
		if cutoff < 2500.0 {
			cutoff = 2500.0
		}
		// 1st-order lowpass filter: y[n] = (1-alpha)*y[n-1] + alpha*x[n]
		alpha := 2.0 * math.Pi * cutoff / (2.0*math.Pi*cutoff + sr)
		var prev float64
		for i := 0; i < numSamples; i++ {
			prev = (1.0-alpha)*prev + alpha*sourceMono[i]*distanceGain
			attenuatedSource[i] = prev
		}
	} else {
		for i := 0; i < numSamples; i++ {
			attenuatedSource[i] = sourceMono[i] * distanceGain
		}
	}

	// 4. ITD (Interaural Time Difference)
	delayLeftSec, delayRightSec := r.ComputeITD(source.AzimuthDeg, source.ElevationDeg)
	delayLeftSamples := delayLeftSec * sr
	delayRightSamples := delayRightSec * sr

	leftDelayed := applyFractionalDelay4thOrder(attenuatedSource, delayLeftSamples)
	rightDelayed := applyFractionalDelay4thOrder(attenuatedSource, delayRightSamples)

	// 5. ILD (Interaural Level Difference)
	gainLeft, gainRight := r.ComputeILD(source.AzimuthDeg, source.ElevationDeg)

	for i := 0; i < numSamples; i++ {
		leftDelayed[i] *= gainLeft
		rightDelayed[i] *= gainRight
	}

	// 6. Head Shadow Filtering (contralateral ear frequency-dependent dampening)
	// Apply head shadow filter to whichever ear is shadowed
	az := normalizeAzimuth(source.AzimuthDeg)
	if az > 5.0 {
		// Sound is on the right, shadow left ear
		shadowRatio := math.Sin(az * math.Pi / 180.0)
		leftDelayed = applyHeadShadowFilter(leftDelayed, sr, shadowRatio)
	} else if az < -5.0 {
		// Sound is on the left, shadow right ear
		shadowRatio := math.Sin(-az * math.Pi / 180.0)
		rightDelayed = applyHeadShadowFilter(rightDelayed, sr, shadowRatio)
	}

	// 7. Pinna Elevation Notch Filter (6kHz to 10kHz notch moving with elevation)
	if r.config.EnablePinnaNotch {
		el := clampElevation(source.ElevationDeg)
		// Elevation notch: 6500 Hz at -45 deg, 8300 Hz at 0 deg, 10500 Hz at +90 deg
		notchFreq := 8300.0 + 35.0*el
		leftDelayed = applyNotchFilter(leftDelayed, sr, notchFreq, 4.0, -8.0)
		rightDelayed = applyNotchFilter(rightDelayed, sr, notchFreq, 4.0, -8.0)
	}

	outSamples := [][]float64{leftDelayed, rightDelayed}

	return &PCMBuffer{
		SampleRate: pcm.SampleRate,
		Channels:   2,
		NumSamples: numSamples,
		Samples:    outSamples,
	}, nil
}

// applyFractionalDelay4thOrder applies high-accuracy Lagrange fractional delay interpolation.
func applyFractionalDelay4thOrder(input []float64, delay float64) []float64 {
	n := len(input)
	output := make([]float64, n)

	intDelay := int(math.Floor(delay))
	frac := delay - float64(intDelay) // [0.0, 1.0)

	// 4th order Lagrange interpolation polynomial coefficients
	c0 := -frac * (frac - 1.0) * (frac - 2.0) / 6.0
	c1 := (frac + 1.0) * (frac - 1.0) * (frac - 2.0) / 2.0
	c2 := -(frac + 1.0) * frac * (frac - 2.0) / 2.0
	c3 := (frac + 1.0) * frac * (frac - 1.0) / 6.0

	for i := 0; i < n; i++ {
		idx := i - intDelay
		var val float64
		if idx-1 >= 0 && idx-1 < n {
			val += c0 * input[idx-1]
		}
		if idx >= 0 && idx < n {
			val += c1 * input[idx]
		}
		if idx+1 >= 0 && idx+1 < n {
			val += c2 * input[idx+1]
		}
		if idx+2 >= 0 && idx+2 < n {
			val += c3 * input[idx+2]
		}
		output[i] = val
	}
	return output
}

// applyHeadShadowFilter applies 1st-order IIR lowpass shadowing to the contralateral ear.
func applyHeadShadowFilter(sig []float64, sampleRate, shadowAmount float64) []float64 {
	n := len(sig)
	out := make([]float64, n)

	// Cutoff frequency transitions from 18kHz down to 2.5kHz based on shadowAmount
	cutoff := 18000.0 - shadowAmount*(18000.0-2500.0)
	alpha := 2.0 * math.Pi * cutoff / (2.0*math.Pi*cutoff + sampleRate)

	var prev float64
	for i := 0; i < n; i++ {
		prev = (1.0-alpha)*prev + alpha*sig[i]
		out[i] = prev
	}
	return out
}

// applyNotchFilter applies a 2nd-order IIR peaking/notch filter to simulate pinna reflections.
func applyNotchFilter(sig []float64, sampleRate, centerFreq, q, gainDB float64) []float64 {
	n := len(sig)
	out := make([]float64, n)

	A := math.Pow(10.0, gainDB/40.0)
	w0 := 2.0 * math.Pi * centerFreq / sampleRate
	alpha := math.Sin(w0) / (2.0 * q)

	// Peaking / notch biquad filter coefficients (Audio EQ Cookbook)
	b0 := 1.0 + alpha*A
	b1 := -2.0 * math.Cos(w0)
	b2 := 1.0 - alpha*A
	a0 := 1.0 + alpha/A
	a1 := -2.0 * math.Cos(w0)
	a2 := 1.0 - alpha/A

	b0 /= a0
	b1 /= a0
	b2 /= a0
	a1 /= a0
	a2 /= a0

	var x1, x2, y1, y2 float64
	for i := 0; i < n; i++ {
		x0 := sig[i]
		y0 := b0*x0 + b1*x1 + b2*x2 - a1*y1 - a2*y2
		out[i] = y0
		x2 = x1
		x1 = x0
		y2 = y1
		y1 = y0
	}
	return out
}

func normalizeAzimuth(az float64) float64 {
	for az > 180.0 {
		az -= 360.0
	}
	for az <= -180.0 {
		az += 360.0
	}
	return az
}

func clampElevation(el float64) float64 {
	if el > 90.0 {
		return 90.0
	}
	if el < -45.0 {
		return -45.0
	}
	return el
}
