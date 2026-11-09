// Package audio provides professional audio decoding, stream header parsing,
// and buffer unpacking for heterogeneous audio formats (WAV, MP3, FLAC, OGG).
package audio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

// AudioFormat represents supported container and codec formats.
type AudioFormat string

const (
	FormatUnknown AudioFormat = "UNKNOWN"
	FormatWAV     AudioFormat = "WAV"
	FormatMP3     AudioFormat = "MP3"
	FormatFLAC    AudioFormat = "FLAC"
	FormatOGG     AudioFormat = "OGG"
)

// Common errors for audio stream parsing and unpacking.
var (
	ErrEmptyAudioStream       = errors.New("audio stream is empty")
	ErrUnsupportedAudioFormat = errors.New("unsupported or unrecognized audio format")
	ErrCorruptStreamHeader    = errors.New("corrupt audio stream header")
	ErrUnexpectedEOF          = errors.New("unexpected end of audio stream while parsing")
	ErrUnsupportedBitDepth    = errors.New("unsupported audio bit depth for PCM unpacking")
)

// AudioStreamInfo contains canonical unpacked stream properties across all formats.
type AudioStreamInfo struct {
	Format       AudioFormat
	Codec        string
	SampleRate   int
	Channels     int
	BitsPerSample int
	BitrateKbps  int
	TotalSamples int64
	DurationSec  float64
	HeaderOffset int64
	DataSize     int64
	IsFloatingPoint bool
}

// PCMBuffer holds unpacked normalized 64-bit float audio samples in memory.
// Samples are organized by channel: Samples[channelIndex][sampleIndex].
type PCMBuffer struct {
	Channels   int
	SampleRate int
	NumSamples int
	Samples    [][]float64
}

// DetectFormat inspects the initial bytes of a stream to identify its format.
func DetectFormat(lead []byte) AudioFormat {
	if len(lead) < 4 {
		return FormatUnknown
	}
	if len(lead) >= 12 && string(lead[0:4]) == "RIFF" && string(lead[8:12]) == "WAVE" {
		return FormatWAV
	}
	if string(lead[0:4]) == "fLaC" {
		return FormatFLAC
	}
	if string(lead[0:4]) == "OggS" {
		return FormatOGG
	}
	if string(lead[0:3]) == "ID3" {
		return FormatMP3
	}
	if lead[0] == 0xFF && (lead[1]&0xE0) == 0xE0 {
		return FormatMP3
	}
	return FormatUnknown
}

// DecodeHeader automatically detects the format from r and parses the full stream header.
func DecodeHeader(r io.Reader) (*AudioStreamInfo, error) {
	buf := make([]byte, 16)
	n, err := io.ReadFull(r, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, fmt.Errorf("read header signature: %w", err)
	}
	if n < 4 {
		return nil, ErrEmptyAudioStream
	}
	format := DetectFormat(buf[:n])
	combined := io.MultiReader(bytes.NewReader(buf[:n]), r)

	switch format {
	case FormatWAV:
		return ParseWAV(combined)
	case FormatMP3:
		return ParseMP3(combined)
	case FormatFLAC:
		return ParseFLAC(combined)
	case FormatOGG:
		return ParseOGG(combined)
	default:
		return nil, ErrUnsupportedAudioFormat
	}
}

// ParseWAV decodes a RIFF/WAVE header and extracts format and data chunk metrics.
func ParseWAV(r io.Reader) (*AudioStreamInfo, error) {
	var riffHeader [12]byte
	if _, err := io.ReadFull(r, riffHeader[:]); err != nil {
		return nil, ErrCorruptStreamHeader
	}
	if string(riffHeader[0:4]) != "RIFF" || string(riffHeader[8:12]) != "WAVE" {
		return nil, ErrCorruptStreamHeader
	}

	info := &AudioStreamInfo{
		Format: FormatWAV,
		Codec:  "pcm",
	}

	foundFmt := false
	foundData := false

	for {
		var chunkHeader [8]byte
		if _, err := io.ReadFull(r, chunkHeader[:]); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return nil, err
		}
		chunkID := string(chunkHeader[0:4])
		chunkSize := int64(binary.LittleEndian.Uint32(chunkHeader[4:8]))

		if chunkID == "fmt " {
			if chunkSize < 16 {
				return nil, ErrCorruptStreamHeader
			}
			fmtData := make([]byte, chunkSize)
			if _, err := io.ReadFull(r, fmtData); err != nil {
				return nil, ErrUnexpectedEOF
			}
			audioFormat := binary.LittleEndian.Uint16(fmtData[0:2])
			info.Channels = int(binary.LittleEndian.Uint16(fmtData[2:4]))
			info.SampleRate = int(binary.LittleEndian.Uint32(fmtData[4:8]))
			byteRate := int(binary.LittleEndian.Uint32(fmtData[8:12]))
			info.BitrateKbps = (byteRate * 8) / 1000
			info.BitsPerSample = int(binary.LittleEndian.Uint16(fmtData[14:16]))

			if audioFormat == 3 {
				info.IsFloatingPoint = true
				info.Codec = "pcm_f32le"
			} else if audioFormat == 1 {
				info.Codec = fmt.Sprintf("pcm_s%dle", info.BitsPerSample)
			}
			foundFmt = true
		} else if chunkID == "data" {
			info.DataSize = chunkSize
			foundData = true
			break
		} else {
			if _, err := io.CopyN(io.Discard, r, chunkSize); err != nil {
				return nil, ErrUnexpectedEOF
			}
		}
	}

	if !foundFmt {
		return nil, ErrCorruptStreamHeader
	}

	if info.BitsPerSample > 0 && info.Channels > 0 && foundData {
		bytesPerSample := (info.BitsPerSample / 8) * info.Channels
		if bytesPerSample > 0 {
			info.TotalSamples = info.DataSize / int64(bytesPerSample)
			if info.SampleRate > 0 {
				info.DurationSec = float64(info.TotalSamples) / float64(info.SampleRate)
			}
		}
	}

	return info, nil
}

// mp3Bitrates defines standard bitrates (kbps) for MPEG-1 Audio Layer III.
var mp3Bitrates = [16]int{
	0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 0,
}

// mp3SampleRates defines standard sampling frequencies for MPEG-1.
var mp3SampleRates = [4]int{
	44100, 48000, 32000, 0,
}

// ParseMP3 scans ID3v2 tags and reads MPEG-1 Layer 3 frame sync and headers.
func ParseMP3(r io.Reader) (*AudioStreamInfo, error) {
	buf := make([]byte, 10)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, ErrCorruptStreamHeader
	}

	reader := io.MultiReader(bytes.NewReader(buf), r)
	if string(buf[0:3]) == "ID3" {
		id3Size := (int(buf[6]) << 21) | (int(buf[7]) << 14) | (int(buf[8]) << 7) | int(buf[9])
		skipTotal := int64(10 + id3Size)
		discardBuf := make([]byte, skipTotal)
		if _, err := io.ReadFull(reader, discardBuf); err != nil {
			return nil, ErrUnexpectedEOF
		}
	}

	// Scan for frame synchronization word (11 bits set: 0xFFE0 mask)
	var syncBuf [4]byte
	foundSync := false

	for i := 0; i < 4096; i++ {
		var single [1]byte
		if _, err := io.ReadFull(r, single[:]); err != nil {
			return nil, ErrCorruptStreamHeader
		}
		if single[0] == 0xFF {
			syncBuf[0] = 0xFF
			if _, err := io.ReadFull(r, syncBuf[1:4]); err != nil {
				return nil, ErrCorruptStreamHeader
			}
			if (syncBuf[1] & 0xE0) == 0xE0 {
				foundSync = true
				break
			}
		}
	}

	if !foundSync {
		return nil, errors.New("valid MP3 frame sync not found")
	}

	versionBits := (syncBuf[1] >> 3) & 0x03
	layerBits := (syncBuf[1] >> 1) & 0x03
	bitrateIdx := (syncBuf[2] >> 4) & 0x0F
	samplerateIdx := (syncBuf[2] >> 2) & 0x03
	channelMode := (syncBuf[3] >> 6) & 0x03

	if versionBits != 3 || layerBits != 1 {
		return nil, errors.New("only MPEG-1 Layer 3 is directly supported by parser")
	}

	bitrate := mp3Bitrates[bitrateIdx]
	samplerate := mp3SampleRates[samplerateIdx]
	if bitrate == 0 || samplerate == 0 {
		return nil, ErrCorruptStreamHeader
	}

	channels := 2
	if channelMode == 3 {
		channels = 1
	}

	return &AudioStreamInfo{
		Format:        FormatMP3,
		Codec:         "mp3",
		SampleRate:    samplerate,
		Channels:      channels,
		BitsPerSample: 16,
		BitrateKbps:   bitrate,
	}, nil
}

// ParseFLAC decodes a Native FLAC bitstream STREAMINFO metadata block.
func ParseFLAC(r io.Reader) (*AudioStreamInfo, error) {
	var magic [4]byte
	if _, err := io.ReadFull(r, magic[:]); err != nil {
		return nil, ErrCorruptStreamHeader
	}
	if string(magic[:]) != "fLaC" {
		return nil, ErrCorruptStreamHeader
	}

	var blockHeader [4]byte
	if _, err := io.ReadFull(r, blockHeader[:]); err != nil {
		return nil, ErrCorruptStreamHeader
	}

	blockType := blockHeader[0] & 0x7F
	blockSize := (int(blockHeader[1]) << 16) | (int(blockHeader[2]) << 8) | int(blockHeader[3])

	if blockType != 0 || blockSize < 34 {
		return nil, errors.New("FLAC stream missing STREAMINFO metadata block")
	}

	infoBlock := make([]byte, blockSize)
	if _, err := io.ReadFull(r, infoBlock); err != nil {
		return nil, ErrUnexpectedEOF
	}

	sampleRate := (int(infoBlock[10]) << 12) | (int(infoBlock[11]) << 4) | (int(infoBlock[12]) >> 4)
	channels := int((infoBlock[12]>>1)&0x07) + 1
	bitsPerSample := int(((infoBlock[12]&0x01)<<4)|(infoBlock[13]>>4)) + 1
	totalSamples := (int64(infoBlock[13]&0x0F) << 32) |
		(int64(infoBlock[14]) << 24) |
		(int64(infoBlock[15]) << 16) |
		(int64(infoBlock[16]) << 8) |
		int64(infoBlock[17])

	duration := 0.0
	if sampleRate > 0 && totalSamples > 0 {
		duration = float64(totalSamples) / float64(sampleRate)
	}

	return &AudioStreamInfo{
		Format:        FormatFLAC,
		Codec:         "flac",
		SampleRate:    sampleRate,
		Channels:      channels,
		BitsPerSample: bitsPerSample,
		TotalSamples:  totalSamples,
		DurationSec:   duration,
	}, nil
}

// ParseOGG reads the initial OggS capture pattern and Vorbis/Opus identification packet.
func ParseOGG(r io.Reader) (*AudioStreamInfo, error) {
	var oggHeader [27]byte
	if _, err := io.ReadFull(r, oggHeader[:]); err != nil {
		return nil, ErrCorruptStreamHeader
	}
	if string(oggHeader[0:4]) != "OggS" {
		return nil, ErrCorruptStreamHeader
	}

	numSegments := int(oggHeader[26])
	segmentTable := make([]byte, numSegments)
	if _, err := io.ReadFull(r, segmentTable); err != nil {
		return nil, ErrUnexpectedEOF
	}

	packetSize := 0
	for _, s := range segmentTable {
		packetSize += int(s)
	}

	packetData := make([]byte, packetSize)
	if _, err := io.ReadFull(r, packetData); err != nil {
		return nil, ErrUnexpectedEOF
	}

	info := &AudioStreamInfo{
		Format: FormatOGG,
	}

	if len(packetData) >= 8 && string(packetData[0:8]) == "OpusHead" {
		info.Codec = "opus"
		info.Channels = int(packetData[9])
		info.SampleRate = int(binary.LittleEndian.Uint32(packetData[12:16]))
		info.BitsPerSample = 16
		return info, nil
	}

	if len(packetData) >= 7 && string(packetData[1:7]) == "vorbis" {
		info.Codec = "vorbis"
		info.Channels = int(packetData[11])
		info.SampleRate = int(binary.LittleEndian.Uint32(packetData[12:16]))
		nominalBitrate := int(binary.LittleEndian.Uint32(packetData[20:24]))
		if nominalBitrate > 0 {
			info.BitrateKbps = nominalBitrate / 1000
		}
		info.BitsPerSample = 16
		return info, nil
	}

	return nil, errors.New("unrecognized Ogg payload (neither OpusHead nor Vorbis)")
}

// UnpackPCM reads raw PCM data according to info and unpacks into normalized float64 samples [-1.0, 1.0].
func UnpackPCM(r io.Reader, info *AudioStreamInfo) (*PCMBuffer, error) {
	if info == nil {
		return nil, errors.New("cannot unpack PCM with nil AudioStreamInfo")
	}
	if info.Channels <= 0 || info.SampleRate <= 0 {
		return nil, errors.New("invalid channel or sample rate configuration")
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read PCM stream: %w", err)
	}

	bytesPerSample := info.BitsPerSample / 8
	if bytesPerSample <= 0 {
		return nil, ErrUnsupportedBitDepth
	}
	bytesPerFrame := bytesPerSample * info.Channels
	if bytesPerFrame <= 0 || len(data) < bytesPerFrame {
		return nil, ErrEmptyAudioStream
	}

	numFrames := len(data) / bytesPerFrame
	buf := &PCMBuffer{
		Channels:   info.Channels,
		SampleRate: info.SampleRate,
		NumSamples: numFrames,
		Samples:    make([][]float64, info.Channels),
	}
	for c := 0; c < info.Channels; c++ {
		buf.Samples[c] = make([]float64, numFrames)
	}

	for i := 0; i < numFrames; i++ {
		frameOffset := i * bytesPerFrame
		for c := 0; c < info.Channels; c++ {
			sampleOffset := frameOffset + c*bytesPerSample
			var val float64
			switch info.BitsPerSample {
			case 16:
				raw := int16(binary.LittleEndian.Uint16(data[sampleOffset : sampleOffset+2]))
				val = float64(raw) / 32768.0
			case 24:
				b := data[sampleOffset : sampleOffset+3]
				raw := int32(b[0]) | (int32(b[1]) << 8) | (int32(b[2]) << 16)
				if raw&0x800000 != 0 {
					raw |= ^0xFFFFFF
				}
				val = float64(raw) / 8388608.0
			case 32:
				if info.IsFloatingPoint {
					bits := binary.LittleEndian.Uint32(data[sampleOffset : sampleOffset+4])
					val = float64(math.Float32frombits(bits))
				} else {
					raw := int32(binary.LittleEndian.Uint32(data[sampleOffset : sampleOffset+4]))
					val = float64(raw) / 2147483648.0
				}
			default:
				return nil, ErrUnsupportedBitDepth
			}
			buf.Samples[c][i] = val
		}
	}

	return buf, nil
}
