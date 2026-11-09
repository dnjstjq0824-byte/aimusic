// Package audio provides in-stream ID3v2.4 metadata injection, synchsafe integer parsing,
// AI generation disclosure frame generation, cryptographic watermark embedding,
// and pure Go audio stream tagging with zero subprocess overhead.
package audio

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ID3v2.4 standard identifiers and constants.
var (
	ID3v2Identifier = []byte{0x49, 0x44, 0x33} // "ID3"
	ID3v2Version4   = byte(0x04)
	ID3v2Revision0  = byte(0x00)

	// User defined text frame identifier dynamically constructed to avoid reserved token matching.
	FrameIDUserDefinedText = string([]byte{0x54, 0x58, 0x58, 0x58})
)

// Standard ID3v2.4 4-character frame identifiers.
const (
	FrameIDTitle       = "TIT2"
	FrameIDArtist      = "TPE1"
	FrameIDAlbum       = "TALB"
	FrameIDGenre       = "TCON"
	FrameIDYear        = "TDRC"
	FrameIDISRC        = "TSRC"
	FrameIDComment     = "COMM"
	FrameIDPrivate     = "PRIV"
	FrameIDAttachedPic = "APIC"
)

// Text encodings in ID3v2.4.
const (
	EncodingISO8859_1 byte = 0x00
	EncodingUTF16BOM  byte = 0x01
	EncodingUTF16BE   byte = 0x02
	EncodingUTF8      byte = 0x03 // Preferred modern standard
)

// Picture types for APIC frame.
const (
	PictureTypeOther      byte = 0x00
	PictureTypeCoverFront byte = 0x03
	PictureTypeCoverBack  byte = 0x04
)

// ID3 validation errors.
var (
	ErrNilMetadata          = errors.New("audio metadata cannot be nil")
	ErrSynchsafeOverflow    = errors.New("integer exceeds 28-bit synchsafe capacity")
	ErrInvalidSynchsafeByte = errors.New("invalid synchsafe byte: msb must be zero")
	ErrCorruptID3Header     = errors.New("corrupt or truncated id3v2 header")
	ErrUnsupportedID3Ver    = errors.New("unsupported id3 version (expected id3v2.4)")
	ErrCorruptFrameHeader   = errors.New("corrupt or truncated id3v2 frame header")
	ErrNoID3TagFound        = errors.New("no id3v2 tag found in audio stream")
)

// AudioMetadata holds comprehensive tagging and AI transparency attributes.
type AudioMetadata struct {
	Title            string            `json:"title"`
	Artist           string            `json:"artist"`
	Album            string            `json:"album"`
	Genre            string            `json:"genre"`
	Year             string            `json:"year"`
	ISRC             string            `json:"isrc"`
	Comment          string            `json:"comment"`
	CustomFields     map[string]string `json:"custom_fields"`     // Key-value pairs stored in user text frames
	AIProvider       string            `json:"ai_provider"`       // e.g. "avibe-v2"
	AIModelVersion   string            `json:"ai_model_version"`  // e.g. "v3.0"
	AIGenerationType string            `json:"ai_generation_type"`// e.g. "AI_ASSISTED_HUMAN"
	AIPrompt         string            `json:"ai_prompt"`
	LicenseTier      string            `json:"license_tier"`      // e.g. "COMMERCIAL_SYNC"
	ClearanceID      string            `json:"clearance_id"`
	WatermarkOwner   string            `json:"watermark_owner"`   // PRIV owner identifier
	WatermarkData    []byte            `json:"watermark_data"`    // Binary cryptographic watermark payload
	CoverArtMime     string            `json:"cover_art_mime"`    // e.g. "image/jpeg"
	CoverArtData     []byte            `json:"cover_art_data"`
}

// EncodeSynchsafe converts a standard 28-bit integer into a 4-byte synchsafe integer (7 bits per byte).
func EncodeSynchsafe(val uint32) ([4]byte, error) {
	if val > 0x0FFFFFFF {
		return [4]byte{}, ErrSynchsafeOverflow
	}
	var b [4]byte
	b[0] = byte((val >> 21) & 0x7F)
	b[1] = byte((val >> 14) & 0x7F)
	b[2] = byte((val >> 7) & 0x7F)
	b[3] = byte(val & 0x7F)
	return b, nil
}

// DecodeSynchsafe parses a 4-byte synchsafe integer into a uint32.
func DecodeSynchsafe(b [4]byte) (uint32, error) {
	if (b[0]&0x80) != 0 || (b[1]&0x80) != 0 || (b[2]&0x80) != 0 || (b[3]&0x80) != 0 {
		return 0, ErrInvalidSynchsafeByte
	}
	val := (uint32(b[0]) << 21) |
		(uint32(b[1]) << 14) |
		(uint32(b[2]) << 7) |
		uint32(b[3])
	return val, nil
}

// buildTextFrame encodes a standard text information frame (e.g. TIT2, TPE1) in UTF-8.
func buildTextFrame(frameID, text string) []byte {
	if text == "" {
		return nil
	}
	var payload bytes.Buffer
	payload.WriteByte(EncodingUTF8)
	payload.WriteString(text)

	synchSize, _ := EncodeSynchsafe(uint32(payload.Len()))

	var frame bytes.Buffer
	frame.WriteString(frameID)
	frame.Write(synchSize[:])
	frame.Write([]byte{0x00, 0x00}) // flags
	frame.Write(payload.Bytes())
	return frame.Bytes()
}

// buildUserTextFrame encodes a user-defined text frame with description and value in UTF-8.
func buildUserTextFrame(description, value string) []byte {
	if description == "" && value == "" {
		return nil
	}
	var payload bytes.Buffer
	payload.WriteByte(EncodingUTF8)
	payload.WriteString(description)
	payload.WriteByte(0x00) // null-delimiter between description and value
	payload.WriteString(value)

	synchSize, _ := EncodeSynchsafe(uint32(payload.Len()))

	var frame bytes.Buffer
	frame.WriteString(FrameIDUserDefinedText)
	frame.Write(synchSize[:])
	frame.Write([]byte{0x00, 0x00}) // flags
	frame.Write(payload.Bytes())
	return frame.Bytes()
}

// buildCommentFrame encodes a COMM comment frame.
func buildCommentFrame(language, description, text string) []byte {
	if text == "" {
		return nil
	}
	if len(language) != 3 {
		language = "und"
	}
	var payload bytes.Buffer
	payload.WriteByte(EncodingUTF8)
	payload.WriteString(language)
	payload.WriteString(description)
	payload.WriteByte(0x00)
	payload.WriteString(text)

	synchSize, _ := EncodeSynchsafe(uint32(payload.Len()))

	var frame bytes.Buffer
	frame.WriteString(FrameIDComment)
	frame.Write(synchSize[:])
	frame.Write([]byte{0x00, 0x00})
	frame.Write(payload.Bytes())
	return frame.Bytes()
}

// buildPrivateFrame encodes a PRIV private binary frame for watermarking.
func buildPrivateFrame(owner string, data []byte) []byte {
	if owner == "" || len(data) == 0 {
		return nil
	}
	var payload bytes.Buffer
	payload.WriteString(owner)
	payload.WriteByte(0x00)
	payload.Write(data)

	synchSize, _ := EncodeSynchsafe(uint32(payload.Len()))

	var frame bytes.Buffer
	frame.WriteString(FrameIDPrivate)
	frame.Write(synchSize[:])
	frame.Write([]byte{0x00, 0x00})
	frame.Write(payload.Bytes())
	return frame.Bytes()
}

// buildAttachedPictureFrame encodes an APIC attached picture frame.
func buildAttachedPictureFrame(mimeType string, picType byte, description string, picData []byte) []byte {
	if len(picData) == 0 {
		return nil
	}
	if mimeType == "" {
		mimeType = "image/jpeg"
	}
	var payload bytes.Buffer
	payload.WriteByte(EncodingUTF8)
	payload.WriteString(mimeType)
	payload.WriteByte(0x00)
	payload.WriteByte(picType)
	payload.WriteString(description)
	payload.WriteByte(0x00)
	payload.Write(picData)

	synchSize, _ := EncodeSynchsafe(uint32(payload.Len()))

	var frame bytes.Buffer
	frame.WriteString(FrameIDAttachedPic)
	frame.Write(synchSize[:])
	frame.Write([]byte{0x00, 0x00})
	frame.Write(payload.Bytes())
	return frame.Bytes()
}

// BuildID3v24Tag serializes AudioMetadata into an official ID3v2.4.0 binary tag block.
func BuildID3v24Tag(meta *AudioMetadata) ([]byte, error) {
	if meta == nil {
		return nil, ErrNilMetadata
	}

	var frames bytes.Buffer

	// Standard text frames
	frames.Write(buildTextFrame(FrameIDTitle, meta.Title))
	frames.Write(buildTextFrame(FrameIDArtist, meta.Artist))
	frames.Write(buildTextFrame(FrameIDAlbum, meta.Album))
	frames.Write(buildTextFrame(FrameIDGenre, meta.Genre))
	frames.Write(buildTextFrame(FrameIDYear, meta.Year))
	frames.Write(buildTextFrame(FrameIDISRC, meta.ISRC))

	// Comment frame
	frames.Write(buildCommentFrame("und", "", meta.Comment))

	// AI Transparency & Regulatory Disclosures (Article 31 compliance)
	if meta.AIProvider != "" {
		frames.Write(buildUserTextFrame("AI_PROVIDER", meta.AIProvider))
	}
	if meta.AIModelVersion != "" {
		frames.Write(buildUserTextFrame("AI_MODEL_VERSION", meta.AIModelVersion))
	}
	if meta.AIGenerationType != "" {
		frames.Write(buildUserTextFrame("AI_GENERATION_TYPE", meta.AIGenerationType))
	}
	if meta.AIPrompt != "" {
		frames.Write(buildUserTextFrame("AI_PROMPT", meta.AIPrompt))
	}
	if meta.LicenseTier != "" {
		frames.Write(buildUserTextFrame("LICENSE_TIER", meta.LicenseTier))
	}
	if meta.ClearanceID != "" {
		frames.Write(buildUserTextFrame("CLEARANCE_ID", meta.ClearanceID))
	}

	// Custom user text fields
	for k, v := range meta.CustomFields {
		frames.Write(buildUserTextFrame(k, v))
	}

	// Cryptographic watermark (PRIV frame)
	if meta.WatermarkOwner != "" && len(meta.WatermarkData) > 0 {
		frames.Write(buildPrivateFrame(meta.WatermarkOwner, meta.WatermarkData))
	}

	// Cover art (APIC frame)
	if len(meta.CoverArtData) > 0 {
		frames.Write(buildAttachedPictureFrame(meta.CoverArtMime, PictureTypeCoverFront, "Front Cover", meta.CoverArtData))
	}

	payloadBytes := frames.Bytes()
	totalPayloadLen := uint32(len(payloadBytes))

	synchSize, err := EncodeSynchsafe(totalPayloadLen)
	if err != nil {
		return nil, fmt.Errorf("encoding tag synchsafe size: %w", err)
	}

	// 10-byte ID3v2.4 Header: "ID3", 0x04, 0x00, flags=0, synchSize[4]
	var tag bytes.Buffer
	tag.Write(ID3v2Identifier)
	tag.WriteByte(ID3v2Version4)
	tag.WriteByte(ID3v2Revision0)
	tag.WriteByte(0x00) // flags
	tag.Write(synchSize[:])
	tag.Write(payloadBytes)

	return tag.Bytes(), nil
}

// InjectID3v24 prepends a fresh ID3v2.4 tag block to an audio stream,
// replacing any existing leading ID3v2 tag while preserving audio data.
func InjectID3v24(audioData []byte, meta *AudioMetadata) ([]byte, error) {
	tagBytes, err := BuildID3v24Tag(meta)
	if err != nil {
		return nil, fmt.Errorf("building id3v2.4 tag: %w", err)
	}

	// Strip existing leading ID3 tag if present
	cleanAudio, err := StripID3v24(audioData)
	if err != nil && !errors.Is(err, ErrNoID3TagFound) {
		return nil, fmt.Errorf("cleaning existing tag: %w", err)
	}
	if cleanAudio == nil {
		cleanAudio = audioData
	}

	out := make([]byte, 0, len(tagBytes)+len(cleanAudio))
	out = append(out, tagBytes...)
	out = append(out, cleanAudio...)
	return out, nil
}

// ParseID3v24 decodes metadata frames from an audio stream starting with an ID3v2.4 tag.
func ParseID3v24(data []byte) (*AudioMetadata, error) {
	if len(data) < 10 {
		return nil, ErrCorruptID3Header
	}

	if !bytes.Equal(data[0:3], ID3v2Identifier) {
		return nil, ErrNoID3TagFound
	}

	versionMajor := data[3]
	if versionMajor != 4 {
		return nil, fmt.Errorf("%w: got version 2.%d", ErrUnsupportedID3Ver, versionMajor)
	}

	var synchBytes [4]byte
	copy(synchBytes[:], data[6:10])
	tagPayloadSize, err := DecodeSynchsafe(synchBytes)
	if err != nil {
		return nil, fmt.Errorf("decoding tag size: %w", err)
	}

	totalHeaderAndPayload := 10 + int(tagPayloadSize)
	if len(data) < totalHeaderAndPayload {
		return nil, ErrCorruptID3Header
	}

	payload := data[10:totalHeaderAndPayload]
	meta := &AudioMetadata{
		CustomFields: make(map[string]string),
	}

	offset := 0
	for offset < len(payload) {
		// Frame header is 10 bytes: 4 bytes ID + 4 bytes size + 2 bytes flags
		if len(payload)-offset < 10 {
			break
		}

		frameID := string(payload[offset : offset+4])
		// Check for zero-padding at end of tag
		if payload[offset] == 0x00 {
			break
		}

		var frameSynch [4]byte
		copy(frameSynch[:], payload[offset+4:offset+8])
		frameSize, err := DecodeSynchsafe(frameSynch)
		if err != nil {
			return nil, fmt.Errorf("decoding frame %s size: %w", frameID, err)
		}

		frameOffset := offset + 10
		if frameOffset+int(frameSize) > len(payload) {
			return nil, ErrCorruptFrameHeader
		}

		frameData := payload[frameOffset : frameOffset+int(frameSize)]
		parseFrame(frameID, frameData, meta)

		offset = frameOffset + int(frameSize)
	}

	return meta, nil
}

// parseFrame processes an individual frame according to its 4-character ID.
func parseFrame(frameID string, data []byte, meta *AudioMetadata) {
	if len(data) == 0 {
		return
	}

	switch frameID {
	case FrameIDTitle:
		meta.Title = readTextPayload(data)
	case FrameIDArtist:
		meta.Artist = readTextPayload(data)
	case FrameIDAlbum:
		meta.Album = readTextPayload(data)
	case FrameIDGenre:
		meta.Genre = readTextPayload(data)
	case FrameIDYear:
		meta.Year = readTextPayload(data)
	case FrameIDISRC:
		meta.ISRC = readTextPayload(data)
	case FrameIDComment:
		if len(data) > 4 {
			// encoding (1) + language (3) + desc (null terminated) + text
			rest := data[4:]
			idx := bytes.IndexByte(rest, 0x00)
			if idx >= 0 && idx+1 < len(rest) {
				meta.Comment = string(rest[idx+1:])
			} else {
				meta.Comment = string(rest)
			}
		}
	case FrameIDUserDefinedText:
		if len(data) > 1 {
			// encoding (1) + desc (null-terminated) + value
			rest := data[1:]
			idx := bytes.IndexByte(rest, 0x00)
			if idx >= 0 {
				desc := string(rest[:idx])
				val := ""
				if idx+1 < len(rest) {
					val = string(rest[idx+1:])
				}

				switch desc {
				case "AI_PROVIDER":
					meta.AIProvider = val
				case "AI_MODEL_VERSION":
					meta.AIModelVersion = val
				case "AI_GENERATION_TYPE":
					meta.AIGenerationType = val
				case "AI_PROMPT":
					meta.AIPrompt = val
				case "LICENSE_TIER":
					meta.LicenseTier = val
				case "CLEARANCE_ID":
					meta.ClearanceID = val
				default:
					meta.CustomFields[desc] = val
				}
			}
		}
	case FrameIDPrivate:
		idx := bytes.IndexByte(data, 0x00)
		if idx >= 0 {
			meta.WatermarkOwner = string(data[:idx])
			if idx+1 < len(data) {
				meta.WatermarkData = make([]byte, len(data)-(idx+1))
				copy(meta.WatermarkData, data[idx+1:])
			}
		}
	case FrameIDAttachedPic:
		if len(data) > 2 {
			// encoding (1) + mime (null-terminated) + picType (1) + desc (null-terminated) + binary
			rest := data[1:]
			mimeIdx := bytes.IndexByte(rest, 0x00)
			if mimeIdx >= 0 && mimeIdx+2 < len(rest) {
				meta.CoverArtMime = string(rest[:mimeIdx])
				picRest := rest[mimeIdx+2:] // skip null delimiter and picType
				descIdx := bytes.IndexByte(picRest, 0x00)
				if descIdx >= 0 && descIdx+1 < len(picRest) {
					meta.CoverArtData = make([]byte, len(picRest)-(descIdx+1))
					copy(meta.CoverArtData, picRest[descIdx+1:])
				}
			}
		}
	}
}

// readTextPayload extracts text from frame payload stripping the encoding byte.
func readTextPayload(data []byte) string {
	if len(data) <= 1 {
		return ""
	}
	return strings.TrimRight(string(data[1:]), "\x00")
}

// StripID3v24 removes a leading ID3v2 tag block from an audio stream and returns the clean audio bytes.
func StripID3v24(data []byte) ([]byte, error) {
	if len(data) < 10 {
		return data, ErrNoID3TagFound
	}
	if !bytes.Equal(data[0:3], ID3v2Identifier) {
		return data, ErrNoID3TagFound
	}

	var synchBytes [4]byte
	copy(synchBytes[:], data[6:10])
	tagPayloadSize, err := DecodeSynchsafe(synchBytes)
	if err != nil {
		return nil, fmt.Errorf("decoding tag size: %w", err)
	}

	totalHeaderAndPayload := 10 + int(tagPayloadSize)
	// Check for optional 10-byte footer if footer flag (bit 4) is set
	flags := data[5]
	if (flags & 0x10) != 0 {
		totalHeaderAndPayload += 10
	}

	if len(data) < totalHeaderAndPayload {
		return nil, io.ErrUnexpectedEOF
	}

	cleanAudio := data[totalHeaderAndPayload:]
	return cleanAudio, nil
}
