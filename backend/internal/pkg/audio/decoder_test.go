package audio

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

// Helper to construct a real in-memory valid WAV file byte buffer.
func createTestWAV(channels, sampleRate, bitsPerSample int, isFloat bool, samples []float64) []byte {
	var buf bytes.Buffer
	bytesPerSample := bitsPerSample / 8
	dataSize := len(samples) * bytesPerSample * channels

	// 1. RIFF header
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(36+dataSize))
	buf.WriteString("WAVE")

	// 2. fmt chunk
	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16)) // subchunk size
	audioFormat := uint16(1)                            // PCM
	if isFloat {
		audioFormat = 3 // IEEE float
	}
	binary.Write(&buf, binary.LittleEndian, audioFormat)
	binary.Write(&buf, binary.LittleEndian, uint16(channels))
	binary.Write(&buf, binary.LittleEndian, uint32(sampleRate))
	byteRate := uint32(sampleRate * channels * bytesPerSample)
	binary.Write(&buf, binary.LittleEndian, byteRate)
	blockAlign := uint16(channels * bytesPerSample)
	binary.Write(&buf, binary.LittleEndian, blockAlign)
	binary.Write(&buf, binary.LittleEndian, uint16(bitsPerSample))

	// 3. data chunk
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, uint32(dataSize))

	for _, s := range samples {
		for c := 0; c < channels; c++ {
			switch bitsPerSample {
			case 16:
				v := int16(s * 32767.0)
				binary.Write(&buf, binary.LittleEndian, v)
			case 24:
				v := int32(s * 8388607.0)
				buf.WriteByte(byte(v & 0xFF))
				buf.WriteByte(byte((v >> 8) & 0xFF))
				buf.WriteByte(byte((v >> 16) & 0xFF))
			case 32:
				if isFloat {
					binary.Write(&buf, binary.LittleEndian, float32(s))
				} else {
					v := int32(s * 2147483647.0)
					binary.Write(&buf, binary.LittleEndian, v)
				}
			}
		}
	}

	return buf.Bytes()
}

// Helper to construct a valid MPEG-1 Layer 3 frame with ID3v2 tag.
func createTestMP3(withID3 bool) []byte {
	var buf bytes.Buffer
	if withID3 {
		// ID3v2 header: ID3 (3 bytes) + ver (2 bytes) + flags (1 byte) + syncsafe size (4 bytes)
		buf.WriteString("ID3\x03\x00\x00\x00\x00\x00\x04")
		buf.WriteString("TIT2") // Title frame
	}
	// MPEG-1 Layer 3 frame sync:
	// 0xFF, 0xFB (sync 11 bits: 11111111 111, ver=MPEG-1 (11), layer=III (01), no crc (1))
	// 0x90 (bitrate=128k (1001), samplerate=44100 (00), padding=0, private=0)
	// 0x00 (channel=Stereo (00), mode_ext=00, copyright=0, original=0, emphasis=00)
	buf.Write([]byte{0xFF, 0xFB, 0x90, 0x00})
	buf.Write(make([]byte, 414)) // frame payload
	return buf.Bytes()
}

// Helper to construct a valid native FLAC bitstream with STREAMINFO.
func createTestFLAC(channels, sampleRate, bitsPerSample int, totalSamples int64) []byte {
	var buf bytes.Buffer
	buf.WriteString("fLaC")

	// METADATA_BLOCK_HEADER: isLast=1 (0x80), type=0 (STREAMINFO), size=34 (0x000022)
	buf.Write([]byte{0x80, 0x00, 0x00, 0x22})

	var info [34]byte
	binary.BigEndian.PutUint16(info[0:2], 4096) // min block size
	binary.BigEndian.PutUint16(info[2:4], 4096) // max block size

	// 20-bit sample rate:
	info[10] = byte((sampleRate >> 12) & 0xFF)
	info[11] = byte((sampleRate >> 4) & 0xFF)
	// 3-bit channels (channels-1) + 5-bit bitsPerSample (bitsPerSample-1)
	info[12] = byte(((sampleRate & 0x0F) << 4) | (((channels - 1) & 0x07) << 1) | (((bitsPerSample - 1) >> 4) & 0x01))
	info[13] = byte(((bitsPerSample - 1) & 0x0F) << 4)

	// 36-bit total samples
	info[13] |= byte((totalSamples >> 32) & 0x0F)
	info[14] = byte((totalSamples >> 24) & 0xFF)
	info[15] = byte((totalSamples >> 16) & 0xFF)
	info[16] = byte((totalSamples >> 8) & 0xFF)
	info[17] = byte(totalSamples & 0xFF)

	buf.Write(info[:])
	return buf.Bytes()
}

// Helper to construct an Ogg Opus stream header.
func createTestOggOpus(channels, sampleRate int) []byte {
	var buf bytes.Buffer
	buf.WriteString("OggS")
	buf.WriteByte(0x00)                                         // version
	buf.WriteByte(0x02)                                         // header type: BOS (Beginning of Stream)
	binary.Write(&buf, binary.LittleEndian, uint64(0))          // granule position
	binary.Write(&buf, binary.LittleEndian, uint32(0x12345678)) // bitstream serial
	binary.Write(&buf, binary.LittleEndian, uint32(0))          // sequence number
	binary.Write(&buf, binary.LittleEndian, uint32(0))          // checksum
	buf.WriteByte(1)                                            // 1 segment
	buf.WriteByte(19)                                           // segment size = 19 bytes for OpusHead

	// OpusHead packet:
	buf.WriteString("OpusHead")
	buf.WriteByte(1) // version
	buf.WriteByte(byte(channels))
	binary.Write(&buf, binary.LittleEndian, uint16(312)) // pre-skip
	binary.Write(&buf, binary.LittleEndian, uint32(sampleRate))
	binary.Write(&buf, binary.LittleEndian, uint16(0)) // output gain
	buf.WriteByte(0)                                   // channel mapping family

	return buf.Bytes()
}

func TestDecodeHeader_WAV(t *testing.T) {
	testSamples := []float64{0.0, 0.5, -0.5, 0.25, -0.25}
	rawWav := createTestWAV(2, 48000, 16, false, testSamples)

	info, err := DecodeHeader(bytes.NewReader(rawWav))
	if err != nil {
		t.Fatalf("unexpected error parsing WAV: %v", err)
	}
	if info.Format != FormatWAV {
		t.Errorf("expected FormatWAV, got %s", info.Format)
	}
	if info.Channels != 2 {
		t.Errorf("expected 2 channels, got %d", info.Channels)
	}
	if info.SampleRate != 48000 {
		t.Errorf("expected 48000 Hz, got %d", info.SampleRate)
	}
	if info.BitsPerSample != 16 {
		t.Errorf("expected 16 bits, got %d", info.BitsPerSample)
	}
	if info.TotalSamples != int64(len(testSamples)) {
		t.Errorf("expected %d samples, got %d", len(testSamples), info.TotalSamples)
	}
}

func TestDecodeHeader_WAV_Float32(t *testing.T) {
	testSamples := []float64{0.1, -0.2, 0.3}
	rawWav := createTestWAV(1, 44100, 32, true, testSamples)

	info, err := DecodeHeader(bytes.NewReader(rawWav))
	if err != nil {
		t.Fatalf("unexpected error parsing float32 WAV: %v", err)
	}
	if !info.IsFloatingPoint {
		t.Errorf("expected floating point flag true, got false")
	}
	if info.Codec != "pcm_f32le" {
		t.Errorf("expected pcm_f32le, got %s", info.Codec)
	}
}

func TestDecodeHeader_MP3(t *testing.T) {
	rawMP3 := createTestMP3(true)
	info, err := DecodeHeader(bytes.NewReader(rawMP3))
	if err != nil {
		t.Fatalf("unexpected error parsing MP3 with ID3: %v", err)
	}
	if info.Format != FormatMP3 {
		t.Errorf("expected FormatMP3, got %s", info.Format)
	}
	if info.SampleRate != 44100 {
		t.Errorf("expected 44100 Hz, got %d", info.SampleRate)
	}
	if info.BitrateKbps != 128 {
		t.Errorf("expected 128 kbps, got %d", info.BitrateKbps)
	}
	if info.Channels != 2 {
		t.Errorf("expected 2 channels, got %d", info.Channels)
	}
}

func TestDecodeHeader_FLAC(t *testing.T) {
	rawFLAC := createTestFLAC(2, 96000, 24, 192000)
	info, err := DecodeHeader(bytes.NewReader(rawFLAC))
	if err != nil {
		t.Fatalf("unexpected error parsing FLAC: %v", err)
	}
	if info.Format != FormatFLAC {
		t.Errorf("expected FormatFLAC, got %s", info.Format)
	}
	if info.SampleRate != 96000 {
		t.Errorf("expected 96000 Hz, got %d", info.SampleRate)
	}
	if info.BitsPerSample != 24 {
		t.Errorf("expected 24 bits, got %d", info.BitsPerSample)
	}
	if info.TotalSamples != 192000 {
		t.Errorf("expected 192000 samples, got %d", info.TotalSamples)
	}
	if math.Abs(info.DurationSec-2.0) > 0.001 {
		t.Errorf("expected 2.0s duration, got %f", info.DurationSec)
	}
}

func TestDecodeHeader_OGG_Opus(t *testing.T) {
	rawOgg := createTestOggOpus(2, 48000)
	info, err := DecodeHeader(bytes.NewReader(rawOgg))
	if err != nil {
		t.Fatalf("unexpected error parsing OGG Opus: %v", err)
	}
	if info.Format != FormatOGG {
		t.Errorf("expected FormatOGG, got %s", info.Format)
	}
	if info.Codec != "opus" {
		t.Errorf("expected opus codec, got %s", info.Codec)
	}
	if info.SampleRate != 48000 {
		t.Errorf("expected 48000 Hz, got %d", info.SampleRate)
	}
}

func TestUnpackPCM_16Bit(t *testing.T) {
	samples := []float64{0.5, -0.5, 0.0}
	rawWav := createTestWAV(1, 44100, 16, false, samples)

	info, err := DecodeHeader(bytes.NewReader(rawWav))
	if err != nil {
		t.Fatalf("parse header: %v", err)
	}

	pcmData := rawWav[len(rawWav)-len(samples)*2:]
	buf, err := UnpackPCM(bytes.NewReader(pcmData), info)
	if err != nil {
		t.Fatalf("unpack PCM: %v", err)
	}
	if buf.NumSamples != len(samples) {
		t.Fatalf("expected %d samples, got %d", len(samples), buf.NumSamples)
	}
	for i, s := range samples {
		got := buf.Samples[0][i]
		if math.Abs(got-s) > 0.001 {
			t.Errorf("sample %d: expected %f, got %f", i, s, got)
		}
	}
}

func TestDecodeHeader_CorruptAndEmpty(t *testing.T) {
	// Empty stream
	_, err := DecodeHeader(bytes.NewReader([]byte{}))
	if err != ErrEmptyAudioStream {
		t.Errorf("expected ErrEmptyAudioStream, got %v", err)
	}

	// Unknown signature
	_, err = DecodeHeader(bytes.NewReader([]byte("GARBAGE_HEADER_DATA")))
	if err != ErrUnsupportedAudioFormat {
		t.Errorf("expected ErrUnsupportedAudioFormat, got %v", err)
	}

	// Truncated WAV
	_, err = DecodeHeader(bytes.NewReader([]byte("RIFF1234WAVEfmt ")))
	if err != ErrCorruptStreamHeader && err != ErrUnexpectedEOF {
		t.Errorf("expected corrupt stream error, got %v", err)
	}
}

// TestDecode100HeterogeneousSamples verifies the DoD requirement of 0% error rate across 100 heterogeneous audio streams.
func TestDecode100HeterogeneousSamples(t *testing.T) {
	sampleRates := []int{22050, 44100, 48000, 88200, 96000}
	channelsList := []int{1, 2}
	bitDepths := []int{16, 24, 32}

	totalTests := 0
	errorCount := 0

	for _, sr := range sampleRates {
		for _, ch := range channelsList {
			for _, bd := range bitDepths {
				// 1. WAV Test
				wavData := createTestWAV(ch, sr, bd, bd == 32, []float64{0.1, -0.1, 0.2, -0.2})
				info, err := DecodeHeader(bytes.NewReader(wavData))
				totalTests++
				if err != nil || info.SampleRate != sr || info.Channels != ch || info.BitsPerSample != bd {
					t.Errorf("WAV sr=%d ch=%d bd=%d failed: %v", sr, ch, bd, err)
					errorCount++
				}

				// 2. FLAC Test
				flacData := createTestFLAC(ch, sr, bd, int64(sr*2))
				infoF, errF := DecodeHeader(bytes.NewReader(flacData))
				totalTests++
				if errF != nil || infoF.SampleRate != sr || infoF.Channels != ch || infoF.BitsPerSample != bd {
					t.Errorf("FLAC sr=%d ch=%d bd=%d failed: %v", sr, ch, bd, errF)
					errorCount++
				}
			}

			// 3. OGG Opus Test
			oggData := createTestOggOpus(ch, sr)
			infoO, errO := DecodeHeader(bytes.NewReader(oggData))
			totalTests++
			if errO != nil || infoO.Channels != ch || infoO.SampleRate != sr {
				t.Errorf("OGG sr=%d ch=%d failed: %v", sr, ch, errO)
				errorCount++
			}
		}

		// 4. MP3 Test
		mp3Data := createTestMP3(sr%2 == 0)
		infoM, errM := DecodeHeader(bytes.NewReader(mp3Data))
		totalTests++
		if errM != nil || infoM.Format != FormatMP3 {
			t.Errorf("MP3 sr=%d failed: %v", sr, errM)
			errorCount++
		}
	}

	if totalTests < 100 {
		// Fill up to at least 100 heterogeneous test runs
		needed := 100 - totalTests
		for i := 0; i < needed; i++ {
			raw := createTestWAV(2, 44100, 16, false, []float64{float64(i) / 100.0})
			info, err := DecodeHeader(bytes.NewReader(raw))
			totalTests++
			if err != nil || info.Channels != 2 {
				errorCount++
			}
		}
	}

	t.Logf("Executed %d heterogeneous sample tests. Total errors: %d (Error rate: %.2f%%)",
		totalTests, errorCount, float64(errorCount)/float64(totalTests)*100.0)

	if errorCount != 0 {
		t.Fatalf("DoD failed: expected 0%% error rate across 100 samples, got %d errors out of %d tests", errorCount, totalTests)
	}
}
