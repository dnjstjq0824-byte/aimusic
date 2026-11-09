package audio

import (
	"bytes"
	"math/rand"
	"testing"
)

// TestADMBWF_StandardAtmos714_Parsing validates extraction of 7.1.4 Bed and 4 dynamic objects.
func TestADMBWF_StandardAtmos714_Parsing(t *testing.T) {
	channels := 16
	sampleRate := 48000
	bitDepth := 24
	bedChannels := 12 // 7.1.4
	numObjects := 4

	data, err := BuildCanonicalADMBWF(channels, sampleRate, bitDepth, bedChannels, numObjects)
	if err != nil {
		t.Fatalf("Failed to build canonical ADM BWF: %v", err)
	}

	parser := NewADMBWFParser()
	meta, err := parser.Parse(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Parse ADM BWF failed: %v", err)
	}

	t.Logf("Parsed ADM Metadata: TotalTracks=%d, BedLayout=%s, BedCount=%d, ObjectCount=%d",
		meta.TotalTracks, meta.BedLayout, meta.BedChannelCount, meta.DynamicObjectCount)

	if meta.TotalTracks != channels {
		t.Errorf("Expected total tracks %d, got %d", channels, meta.TotalTracks)
	}
	if meta.SampleRate != sampleRate {
		t.Errorf("Expected sample rate %d, got %d", sampleRate, meta.SampleRate)
	}
	if meta.BitDepth != bitDepth {
		t.Errorf("Expected bit depth %d, got %d", bitDepth, meta.BitDepth)
	}
	if meta.BedLayout != "7.1.4" {
		t.Errorf("Expected BedLayout '7.1.4', got '%s'", meta.BedLayout)
	}
	if meta.BedChannelCount != bedChannels {
		t.Errorf("Expected bed channels %d, got %d", bedChannels, meta.BedChannelCount)
	}
	if meta.DynamicObjectCount != numObjects {
		t.Errorf("Expected dynamic objects %d, got %d", numObjects, meta.DynamicObjectCount)
	}
	if !meta.SpatialAtmosCompliant {
		t.Errorf("Expected SpatialAtmosCompliant=true, got false")
	}
	if !meta.IsBinauralRenderable {
		t.Errorf("Expected IsBinauralRenderable=true, got false")
	}

	// Verify dynamic object 3D position coordinates
	obj1 := meta.Objects[bedChannels] // First dynamic object
	if obj1.Position.X != 0.25 || obj1.Position.Y != 0.5 || obj1.Position.Z != 0.1 {
		t.Errorf("Object 1 3D position coordinates mismatch: got (%.2f, %.2f, %.2f)",
			obj1.Position.X, obj1.Position.Y, obj1.Position.Z)
	}
}

// TestADMBWF_Atmos512_Parsing validates extraction of 5.1 bed and dynamic objects.
func TestADMBWF_Atmos512_Parsing(t *testing.T) {
	channels := 8
	sampleRate := 48000
	bitDepth := 24
	bedChannels := 6 // 5.1
	numObjects := 2

	data, err := BuildCanonicalADMBWF(channels, sampleRate, bitDepth, bedChannels, numObjects)
	if err != nil {
		t.Fatalf("Failed to build canonical ADM BWF: %v", err)
	}

	parser := NewADMBWFParser()
	meta, err := parser.Parse(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Parse 5.1 ADM failed: %v", err)
	}

	if meta.BedLayout != "5.1" {
		t.Errorf("Expected BedLayout '5.1', got '%s'", meta.BedLayout)
	}
	if meta.DynamicObjectCount != numObjects {
		t.Errorf("Expected %d objects, got %d", numObjects, meta.DynamicObjectCount)
	}
}

// TestADMBWF_CorruptedAndNonWave_Rejection verifies proper error handling.
func TestADMBWF_CorruptedAndNonWave_Rejection(t *testing.T) {
	parser := NewADMBWFParser()

	// Nil reader
	if _, err := parser.Parse(nil); err == nil {
		t.Errorf("Expected error for nil reader")
	}

	// Non-RIFF
	badHeader := []byte("FLAC\x00\x00\x00\x00WAVE")
	if _, err := parser.Parse(bytes.NewReader(badHeader)); err != ErrNotRIFFOrRF64 {
		t.Errorf("Expected ErrNotRIFFOrRF64, got %v", err)
	}

	// Standard stereo WAV without ADM chunks
	var stdWAV bytes.Buffer
	stdWAV.WriteString("RIFF")
	stdWAV.Write([]byte{36, 0, 0, 0})
	stdWAV.WriteString("WAVE")
	stdWAV.WriteString("fmt ")
	stdWAV.Write([]byte{16, 0, 0, 0})
	stdWAV.Write(make([]byte, 16)) // dummy fmt
	stdWAV.WriteString("data")
	stdWAV.Write([]byte{0, 0, 0, 0})

	if _, err := parser.Parse(bytes.NewReader(stdWAV.Bytes())); err != ErrMissingAxmlOrChna {
		t.Errorf("Expected ErrMissingAxmlOrChna, got %v", err)
	}
}

// TestADMBWF_100HeterogeneousStreams tests parsing of 100 heterogeneous ADM BWF files.
func TestADMBWF_100HeterogeneousStreams(t *testing.T) {
	rng := rand.New(rand.NewSource(33445566))

	sampleRates := []int{44100, 48000, 96000}
	bitDepths := []int{16, 24, 32}
	bedOptions := []int{2, 6, 8, 10, 12, 16} // 2.0, 5.1, 7.1, 5.1.4, 7.1.4, 9.1.6

	parser := NewADMBWFParser()

	for i := 0; i < 100; i++ {
		sr := sampleRates[rng.Intn(len(sampleRates))]
		bd := bitDepths[rng.Intn(len(bitDepths))]
		bed := bedOptions[rng.Intn(len(bedOptions))]
		objects := rng.Intn(8) + 1
		totalChannels := bed + objects

		data, err := BuildCanonicalADMBWF(totalChannels, sr, bd, bed, objects)
		if err != nil {
			t.Fatalf("Stream %d build failed: %v", i+1, err)
		}

		meta, err := parser.Parse(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("Stream %d parse failed: %v", i+1, err)
		}

		if meta.TotalTracks != totalChannels {
			t.Fatalf("Stream %d track mismatch: expected %d, got %d", i+1, totalChannels, meta.TotalTracks)
		}
		if meta.BedChannelCount != bed {
			t.Fatalf("Stream %d bed mismatch: expected %d, got %d", i+1, bed, meta.BedChannelCount)
		}
		if meta.DynamicObjectCount != objects {
			t.Fatalf("Stream %d object count mismatch: expected %d, got %d", i+1, objects, meta.DynamicObjectCount)
		}
		if !meta.SpatialAtmosCompliant {
			t.Fatalf("Stream %d expected SpatialAtmosCompliant=true", i+1)
		}
	}

	t.Log("Successfully verified 100 heterogeneous Dolby Atmos ADM BWF streams with 0 errors.")
}
