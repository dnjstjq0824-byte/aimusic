// Package audio provides pure Go Xiph.Org RFC 3533 Ogg container manipulation
// and VorbisComment metadata injection and extraction for Ogg Vorbis and Ogg Opus streams.
package audio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"
)

// OggCodecType identifies the underlying codec in an Ogg container.
type OggCodecType string

const (
	OggCodecOpus   OggCodecType = "Opus"
	OggCodecVorbis OggCodecType = "Vorbis"
	OggCodecFLAC   OggCodecType = "FLAC"
	OggCodecUnknown OggCodecType = "Unknown"
)

// Standard VorbisComment field keys.
const (
	VorbisTagTitle       = "TITLE"
	VorbisTagArtist      = "ARTIST"
	VorbisTagAlbum       = "ALBUM"
	VorbisTagTrackNumber = "TRACKNUMBER"
	VorbisTagDate        = "DATE"
	VorbisTagGenre       = "GENRE"
	VorbisTagCopyright   = "COPYRIGHT"
	VorbisTagDescription = "DESCRIPTION"
	VorbisTagLicense     = "LICENSE"
	VorbisTagOrganization= "ORGANIZATION"
	VorbisTagPerformer   = "PERFORMER"

	// Article 31 AI Transparency Statutory Tags
	VorbisTagAIArticle31Disclosure = "AI_ARTICLE_31_DISCLOSURE"
	VorbisTagAIModelName           = "AI_MODEL"
	VorbisTagAIPrompt              = "AI_PROMPT"
	VorbisTagAISeed                = "AI_SEED"
	VorbisTagAIGenerator           = "AI_GENERATOR"
	VorbisTagAIIsSynthetic         = "AI_IS_SYNTHETIC"
	VorbisTagAICreationTimestamp   = "AI_TIMESTAMP"
)

// Ogg and VorbisComment error definitions.
var (
	ErrNilVorbisComment    = errors.New("vorbis comment object cannot be nil")
	ErrInvalidOggStream    = errors.New("invalid or truncated ogg bitstream")
	ErrMissingBOSPage      = errors.New("missing mandatory beginning-of-stream (bos) ogg page")
	ErrInvalidOggCRC       = errors.New("ogg page checksum verification failed")
	ErrUnsupportedCodec    = errors.New("unsupported ogg audio codec for vorbis comment injection")
	ErrTruncatedComment    = errors.New("truncated or malformed vorbis comment packet")
)

// Ogg CRC-32 table generated with polynomial 0x04C11DB7 (RFC 3533).
var oggCRCTable [256]uint32

func init() {
	for i := 0; i < 256; i++ {
		r := uint32(i) << 24
		for j := 0; j < 8; j++ {
			if (r & 0x80000000) != 0 {
				r = (r << 1) ^ 0x04C11DB7
			} else {
				r = r << 1
			}
		}
		oggCRCTable[i] = r
	}
}

// ComputeOggCRC calculates the 32-bit checksum for an Ogg page (with CRC field cleared).
func ComputeOggCRC(data []byte) uint32 {
	var crc uint32 = 0
	for _, b := range data {
		crc = (crc << 8) ^ oggCRCTable[byte(crc>>24)^b]
	}
	return crc
}

// VorbisComment encapsulates Xiph.Org VorbisComment tag metadata.
type VorbisComment struct {
	Vendor       string
	Comments     map[string][]string // Case-insensitive canonical map (stored in uppercase)

	// Structured field accessors
	Title        string
	Artist       string
	Album        string
	TrackNumber  string
	Date         string
	Genre        string
	Copyright    string
	Description  string
	License      string
	Organization string
	Performer    string

	// Article 31 AI Transparency Statutory Fields
	AIArticle31Disclosure string
	AIModelName           string
	AIPrompt              string
	AISeed                string
	AIGenerator           string
	AIIsSynthetic         bool
	AICreationTimestamp   string
}

// NewVorbisComment initializes an empty VorbisComment instance.
func NewVorbisComment() *VorbisComment {
	return &VorbisComment{
		Vendor:   "AVibe Platform VorbisComment Pure Go Engine v1.0",
		Comments: make(map[string][]string),
	}
}

// DefaultVorbisArticle31Disclosure returns the official statutory transparency text for AI music.
func DefaultVorbisArticle31Disclosure() string {
	return "본 음원은 인공지능 발전과 신뢰 기반 조성 등에 관한 기본법 제31조(인공지능 투명성 확보) 및 글로벌 AI 투명성 규범에 따라 생성형 AI 기술을 활용하여 제작되었음을 투명하게 고지합니다. (AI Generation Disclosure: This audio content was generated or synthesized using artificial intelligence in compliance with Article 31 transparency regulations.)"
}

// Get returns the first value associated with the case-insensitive tag key.
func (vc *VorbisComment) Get(key string) string {
	if vc.Comments == nil {
		return ""
	}
	vals := vc.Comments[strings.ToUpper(key)]
	if len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// GetAll returns all values associated with the case-insensitive tag key.
func (vc *VorbisComment) GetAll(key string) []string {
	if vc.Comments == nil {
		return nil
	}
	return vc.Comments[strings.ToUpper(key)]
}

// Set sets or overwrites a tag key with a single value.
func (vc *VorbisComment) Set(key, value string) {
	if vc.Comments == nil {
		vc.Comments = make(map[string][]string)
	}
	vc.Comments[strings.ToUpper(key)] = []string{value}
}

// Add appends a value to a tag key (supporting multi-value tags like multiple artists).
func (vc *VorbisComment) Add(key, value string) {
	if vc.Comments == nil {
		vc.Comments = make(map[string][]string)
	}
	canonical := strings.ToUpper(key)
	vc.Comments[canonical] = append(vc.Comments[canonical], value)
}

// syncFields synchronizes structured struct fields with the internal Comments map.
func (vc *VorbisComment) syncFields() {
	if vc.Comments == nil {
		vc.Comments = make(map[string][]string)
	}

	setIfNotEmpty := func(key, val string) {
		if val != "" {
			vc.Comments[key] = []string{val}
		}
	}

	setIfNotEmpty(VorbisTagTitle, vc.Title)
	setIfNotEmpty(VorbisTagArtist, vc.Artist)
	setIfNotEmpty(VorbisTagAlbum, vc.Album)
	setIfNotEmpty(VorbisTagTrackNumber, vc.TrackNumber)
	setIfNotEmpty(VorbisTagDate, vc.Date)
	setIfNotEmpty(VorbisTagGenre, vc.Genre)
	setIfNotEmpty(VorbisTagCopyright, vc.Copyright)
	setIfNotEmpty(VorbisTagDescription, vc.Description)
	setIfNotEmpty(VorbisTagLicense, vc.License)
	setIfNotEmpty(VorbisTagOrganization, vc.Organization)
	setIfNotEmpty(VorbisTagPerformer, vc.Performer)

	// Article 31 AI Transparency Sync
	disclosure := vc.AIArticle31Disclosure
	if disclosure == "" && (vc.AIIsSynthetic || vc.AIModelName != "") {
		disclosure = DefaultVorbisArticle31Disclosure()
		vc.AIArticle31Disclosure = disclosure
	}
	setIfNotEmpty(VorbisTagAIArticle31Disclosure, disclosure)
	setIfNotEmpty(VorbisTagAIModelName, vc.AIModelName)
	setIfNotEmpty(VorbisTagAIPrompt, vc.AIPrompt)
	setIfNotEmpty(VorbisTagAISeed, vc.AISeed)
	setIfNotEmpty(VorbisTagAIGenerator, vc.AIGenerator)

	syntheticVal := "FALSE"
	if vc.AIIsSynthetic {
		syntheticVal = "TRUE"
	}
	vc.Comments[VorbisTagAIIsSynthetic] = []string{syntheticVal}

	timestamp := vc.AICreationTimestamp
	if timestamp == "" && vc.AIIsSynthetic {
		timestamp = time.Now().UTC().Format(time.RFC3339)
		vc.AICreationTimestamp = timestamp
	}
	setIfNotEmpty(VorbisTagAICreationTimestamp, timestamp)
}

// EncodeVorbisComment encodes the VorbisComment into raw Xiph comment binary format
// (vendor length + vendor + comment count + list of [comment length + "KEY=VALUE"]).
func EncodeVorbisComment(vc *VorbisComment) ([]byte, error) {
	if vc == nil {
		return nil, ErrNilVorbisComment
	}
	vc.syncFields()

	if vc.Vendor == "" {
		vc.Vendor = "AVibe Platform VorbisComment Pure Go Engine v1.0"
	}

	buf := new(bytes.Buffer)
	vendorBytes := []byte(vc.Vendor)
	if err := binary.Write(buf, binary.LittleEndian, uint32(len(vendorBytes))); err != nil {
		return nil, err
	}
	buf.Write(vendorBytes)

	// Count total comments
	var totalComments uint32 = 0
	for _, vals := range vc.Comments {
		totalComments += uint32(len(vals))
	}

	if err := binary.Write(buf, binary.LittleEndian, totalComments); err != nil {
		return nil, err
	}

	for k, vals := range vc.Comments {
		upperK := strings.ToUpper(k)
		for _, v := range vals {
			entryStr := fmt.Sprintf("%s=%s", upperK, v)
			entryBytes := []byte(entryStr)
			if err := binary.Write(buf, binary.LittleEndian, uint32(len(entryBytes))); err != nil {
				return nil, err
			}
			buf.Write(entryBytes)
		}
	}

	return buf.Bytes(), nil
}

// DecodeVorbisComment unpacks raw VorbisComment bytes into a VorbisComment structure.
func DecodeVorbisComment(data []byte) (*VorbisComment, error) {
	if len(data) < 8 {
		return nil, ErrTruncatedComment
	}

	offset := 0
	vendorLen := binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4

	if uint64(offset)+uint64(vendorLen) > uint64(len(data)) {
		return nil, ErrTruncatedComment
	}
	vendorStr := string(data[offset : offset+int(vendorLen)])
	offset += int(vendorLen)

	if offset+4 > len(data) {
		return nil, ErrTruncatedComment
	}
	commentCount := binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4

	vc := &VorbisComment{
		Vendor:   vendorStr,
		Comments: make(map[string][]string),
	}

	for i := uint32(0); i < commentCount; i++ {
		if offset+4 > len(data) {
			return nil, ErrTruncatedComment
		}
		cLen := binary.LittleEndian.Uint32(data[offset : offset+4])
		offset += 4

		if uint64(offset)+uint64(cLen) > uint64(len(data)) {
			return nil, ErrTruncatedComment
		}
		entry := string(data[offset : offset+int(cLen)])
		offset += int(cLen)

		splitIdx := strings.IndexByte(entry, '=')
		if splitIdx > 0 {
			k := strings.ToUpper(strings.TrimSpace(entry[:splitIdx]))
			v := entry[splitIdx+1:]
			vc.Comments[k] = append(vc.Comments[k], v)

			// Populate structured fields
			switch k {
			case VorbisTagTitle:
				vc.Title = v
			case VorbisTagArtist:
				vc.Artist = v
			case VorbisTagAlbum:
				vc.Album = v
			case VorbisTagTrackNumber:
				vc.TrackNumber = v
			case VorbisTagDate:
				vc.Date = v
			case VorbisTagGenre:
				vc.Genre = v
			case VorbisTagCopyright:
				vc.Copyright = v
			case VorbisTagDescription:
				vc.Description = v
			case VorbisTagLicense:
				vc.License = v
			case VorbisTagOrganization:
				vc.Organization = v
			case VorbisTagPerformer:
				vc.Performer = v
			case VorbisTagAIArticle31Disclosure:
				vc.AIArticle31Disclosure = v
			case VorbisTagAIModelName:
				vc.AIModelName = v
			case VorbisTagAIPrompt:
				vc.AIPrompt = v
			case VorbisTagAISeed:
				vc.AISeed = v
			case VorbisTagAIGenerator:
				vc.AIGenerator = v
			case VorbisTagAIIsSynthetic:
				vc.AIIsSynthetic = strings.ToUpper(v) == "TRUE"
			case VorbisTagAICreationTimestamp:
				vc.AICreationTimestamp = v
			}
		}
	}

	return vc, nil
}

// OggParsedPage represents an individual parsed Ogg page.
type OggParsedPage struct {
	RawData  []byte
	Header   []byte
	Segments []byte
	Payload  []byte
	Flags    uint8
	Granule  uint64
	Serial   uint32
	Seq      uint32
	CRC      uint32
}

// ParseOggPages breaks an Ogg bitstream into individual validated OggParsedPage structures.
func ParseOggPages(data []byte) ([]*OggParsedPage, error) {
	var pages []*OggParsedPage
	offset := 0
	total := len(data)

	for offset < total {
		if total-offset < 27 {
			return nil, ErrInvalidOggStream
		}
		if string(data[offset:offset+4]) != "OggS" {
			return nil, ErrInvalidOggStream
		}

		flags := data[offset+5]
		granule := binary.LittleEndian.Uint64(data[offset+6 : offset+14])
		serial := binary.LittleEndian.Uint32(data[offset+14 : offset+18])
		seq := binary.LittleEndian.Uint32(data[offset+18 : offset+22])
		crc := binary.LittleEndian.Uint32(data[offset+22 : offset+26])
		numSegments := int(data[offset+26])

		headerSize := 27 + numSegments
		if total-offset < headerSize {
			return nil, ErrInvalidOggStream
		}

		segments := data[offset+27 : offset+headerSize]
		payloadSize := 0
		for _, s := range segments {
			payloadSize += int(s)
		}

		pageSize := headerSize + payloadSize
		if total-offset < pageSize {
			return nil, ErrInvalidOggStream
		}

		pageBytes := make([]byte, pageSize)
		copy(pageBytes, data[offset:offset+pageSize])

		// Verify CRC
		zeroed := make([]byte, pageSize)
		copy(zeroed, pageBytes)
		zeroed[22] = 0
		zeroed[23] = 0
		zeroed[24] = 0
		zeroed[25] = 0
		calculatedCRC := ComputeOggCRC(zeroed)
		if calculatedCRC != crc {
			return nil, ErrInvalidOggCRC
		}

		pages = append(pages, &OggParsedPage{
			RawData:  pageBytes,
			Header:   pageBytes[:headerSize],
			Segments: segments,
			Payload:  pageBytes[headerSize:],
			Flags:    flags,
			Granule:  granule,
			Serial:   serial,
			Seq:      seq,
			CRC:      crc,
		})

		offset += pageSize
	}

	return pages, nil
}

// DetectOggCodec identifies the audio codec from the BOS (Beginning of Stream) page.
func DetectOggCodec(bosPage *OggParsedPage) OggCodecType {
	if bosPage == nil || len(bosPage.Payload) < 7 {
		return OggCodecUnknown
	}
	if len(bosPage.Payload) >= 8 && string(bosPage.Payload[:8]) == "OpusHead" {
		return OggCodecOpus
	}
	if bosPage.Payload[0] == 0x01 && string(bosPage.Payload[1:7]) == "vorbis" {
		return OggCodecVorbis
	}
	if len(bosPage.Payload) >= 4 && string(bosPage.Payload[:4]) == "fLaC" {
		return OggCodecFLAC
	}
	return OggCodecUnknown
}

// CreateOggPagesFromPacket creates one or more standard RFC 3533 Ogg pages for a packet.
func CreateOggPagesFromPacket(packet []byte, serial uint32, startSeq uint32, granulePos uint64, isBOS, isEOS bool) [][]byte {
	// 1. Break packet into lacing values (max 255 per segment)
	var allSegments []byte
	remaining := packet
	for len(remaining) >= 255 {
		allSegments = append(allSegments, 255)
		remaining = remaining[255:]
	}
	allSegments = append(allSegments, byte(len(remaining)))

	// 2. Partition segments across Ogg pages (max 255 segments per page)
	var pages [][]byte
	segOffset := 0
	payloadOffset := 0
	currentSeq := startSeq

	for segOffset < len(allSegments) {
		endSeg := segOffset + 255
		if endSeg > len(allSegments) {
			endSeg = len(allSegments)
		}
		pageSegs := allSegments[segOffset:endSeg]

		pagePayloadSize := 0
		for _, s := range pageSegs {
			pagePayloadSize += int(s)
		}
		pagePayload := packet[payloadOffset : payloadOffset+pagePayloadSize]

		var flags uint8 = 0
		if isBOS && len(pages) == 0 {
			flags |= 0x02 // Beginning of Stream
		}
		if len(pages) > 0 {
			flags |= 0x01 // Continuation of packet from previous page
		}
		isLastPage := endSeg == len(allSegments)
		if isEOS && isLastPage {
			flags |= 0x04 // End of Stream
		}

		pageGranule := uint64(0)
		if isLastPage {
			pageGranule = granulePos
		}

		headerSize := 27 + len(pageSegs)
		pageBuf := make([]byte, headerSize+pagePayloadSize)
		copy(pageBuf[0:4], "OggS")
		pageBuf[4] = 0 // version 0
		pageBuf[5] = flags
		binary.LittleEndian.PutUint64(pageBuf[6:14], pageGranule)
		binary.LittleEndian.PutUint32(pageBuf[14:18], serial)
		binary.LittleEndian.PutUint32(pageBuf[18:22], currentSeq)
		// bytes 22..26: CRC initial zeroed field
		pageBuf[26] = byte(len(pageSegs))
		copy(pageBuf[27:headerSize], pageSegs)
		copy(pageBuf[headerSize:], pagePayload)

		// Calculate CRC
		crcVal := ComputeOggCRC(pageBuf)
		binary.LittleEndian.PutUint32(pageBuf[22:26], crcVal)

		pages = append(pages, pageBuf)
		segOffset = endSeg
		payloadOffset += pagePayloadSize
		currentSeq++
	}

	return pages
}

// BuildVorbisCommentPacket constructs the complete codec-specific packet
// (adding "OpusTags" prefix for Opus, or 0x03"vorbis" prefix and framing bit for Vorbis).
func BuildVorbisCommentPacket(vc *VorbisComment, codec OggCodecType) ([]byte, error) {
	rawComment, err := EncodeVorbisComment(vc)
	if err != nil {
		return nil, fmt.Errorf("encoding vorbis comment: %w", err)
	}

	switch codec {
	case OggCodecOpus:
		pkt := make([]byte, 8+len(rawComment))
		copy(pkt[0:8], "OpusTags")
		copy(pkt[8:], rawComment)
		return pkt, nil

	case OggCodecVorbis:
		// Header packet 3: 0x03 + "vorbis" (7 bytes) + rawComment + 0x01 framing bit
		pkt := make([]byte, 7+len(rawComment)+1)
		pkt[0] = 0x03
		copy(pkt[1:7], "vorbis")
		copy(pkt[7:7+len(rawComment)], rawComment)
		pkt[len(pkt)-1] = 0x01 // framing bit
		return pkt, nil

	default:
		return nil, ErrUnsupportedCodec
	}
}

// ParseVorbisCommentPacket unpacks a Vorbis or Opus comment packet into a VorbisComment struct.
func ParseVorbisCommentPacket(packet []byte) (*VorbisComment, OggCodecType, error) {
	if len(packet) < 8 {
		return nil, OggCodecUnknown, ErrTruncatedComment
	}

	if bytes.HasPrefix(packet, []byte("OpusTags")) {
		body := packet[8:]
		vc, err := DecodeVorbisComment(body)
		return vc, OggCodecOpus, err
	}

	if len(packet) >= 8 && packet[0] == 0x03 && string(packet[1:7]) == "vorbis" {
		body := packet[7 : len(packet)-1] // strip 7-byte header and 1-byte framing bit
		vc, err := DecodeVorbisComment(body)
		return vc, OggCodecVorbis, err
	}

	return nil, OggCodecUnknown, ErrUnsupportedCodec
}

// InjectOggVorbisComment injects or updates VorbisComment metadata in an Ogg stream,
// reconstructing Ogg pages, recalculating CRCs, and perfectly preserving audio payloads.
func InjectOggVorbisComment(oggData []byte, vc *VorbisComment) ([]byte, error) {
	if vc == nil {
		return nil, ErrNilVorbisComment
	}

	pages, err := ParseOggPages(oggData)
	if err != nil {
		return nil, fmt.Errorf("parsing input ogg bitstream: %w", err)
	}
	if len(pages) < 2 {
		return nil, ErrMissingBOSPage
	}

	bosPage := pages[0]
	if (bosPage.Flags & 0x02) == 0 {
		return nil, ErrMissingBOSPage
	}

	codec := DetectOggCodec(bosPage)
	if codec != OggCodecOpus && codec != OggCodecVorbis {
		return nil, ErrUnsupportedCodec
	}

	// Ingest existing tags to preserve unedited metadata
	existingVC, err := ExtractOggVorbisComment(oggData)
	if err == nil && existingVC != nil {
		if vc.Comments == nil {
			vc.Comments = make(map[string][]string)
		}
		for k, vals := range existingVC.Comments {
			if _, exists := vc.Comments[k]; !exists {
				vc.Comments[k] = vals
			}
		}
	}

	// Build codec-specific comment packet
	newPacket, err := BuildVorbisCommentPacket(vc, codec)
	if err != nil {
		return nil, fmt.Errorf("building comment packet: %w", err)
	}

	// Identify how many pages the original comment occupied
	// Page 1 is the start of the comment packet.
	// It continues across subsequent pages if they have flag 0x01 (continuation).
	origCommentPageCount := 1
	for idx := 2; idx < len(pages); idx++ {
		if (pages[idx].Flags & 0x01) != 0 {
			origCommentPageCount++
		} else {
			break
		}
	}

	// Create new Ogg page(s) for the new comment packet
	newCommentPages := CreateOggPagesFromPacket(newPacket, bosPage.Serial, 1, 0, false, false)
	newCommentPageCount := len(newCommentPages)

	// Assemble final Ogg stream:
	// 1. BOS Page (Page 0)
	finalBuf := new(bytes.Buffer)
	finalBuf.Write(bosPage.RawData)

	// 2. New Comment Page(s)
	for _, p := range newCommentPages {
		finalBuf.Write(p)
	}

	// 3. Subsequent audio pages (update sequence numbers and recompute CRC)
	seqDelta := uint32(newCommentPageCount - origCommentPageCount)
	for idx := 1 + origCommentPageCount; idx < len(pages); idx++ {
		p := pages[idx]
		newSeq := p.Seq + seqDelta

		// Copy page raw data to update sequence and CRC
		updatedPage := make([]byte, len(p.RawData))
		copy(updatedPage, p.RawData)
		binary.LittleEndian.PutUint32(updatedPage[18:22], newSeq)

		// Zero CRC field and recompute
		updatedPage[22] = 0
		updatedPage[23] = 0
		updatedPage[24] = 0
		updatedPage[25] = 0
		newCRC := ComputeOggCRC(updatedPage)
		binary.LittleEndian.PutUint32(updatedPage[22:26], newCRC)

		finalBuf.Write(updatedPage)
	}

	return finalBuf.Bytes(), nil
}

// ExtractOggVorbisComment extracts the VorbisComment metadata structure from an Ogg bitstream.
func ExtractOggVorbisComment(oggData []byte) (*VorbisComment, error) {
	pages, err := ParseOggPages(oggData)
	if err != nil {
		return nil, fmt.Errorf("parsing ogg pages: %w", err)
	}
	if len(pages) < 2 {
		return nil, ErrMissingBOSPage
	}

	// Reassemble packet from Page 1 and any continuation pages
	packetBuf := new(bytes.Buffer)
	packetBuf.Write(pages[1].Payload)

	for idx := 2; idx < len(pages); idx++ {
		if (pages[idx].Flags & 0x01) != 0 {
			packetBuf.Write(pages[idx].Payload)
		} else {
			break
		}
	}

	vc, _, err := ParseVorbisCommentPacket(packetBuf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("parsing vorbis comment packet: %w", err)
	}

	return vc, nil
}

// BuildCanonicalTestOgg creates a valid, compliant Ogg Vorbis or Ogg Opus bitstream
// with valid BOS identification headers, comment headers, and audio data packets.
func BuildCanonicalTestOgg(codec OggCodecType, durationSec float64, sampleRate, channels int) ([]byte, error) {
	if sampleRate <= 0 {
		sampleRate = 48000
	}
	if channels <= 0 {
		channels = 2
	}
	if durationSec <= 0 {
		durationSec = 1.0
	}

	var serialNum uint32 = 0x41564942 // "AVIB"

	// 1. Page 0: Identification Header (BOS)
	var idPacket []byte
	switch codec {
	case OggCodecOpus:
		// OpusHead: 19 bytes
		buf := new(bytes.Buffer)
		buf.WriteString("OpusHead")
		buf.WriteByte(1) // version 1
		buf.WriteByte(byte(channels))
		_ = binary.Write(buf, binary.LittleEndian, uint16(312)) // pre-skip
		_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
		_ = binary.Write(buf, binary.LittleEndian, int16(0)) // gain 0
		buf.WriteByte(0) // channel mapping family 0 (mono/stereo)
		idPacket = buf.Bytes()

	case OggCodecVorbis:
		// Vorbis ID header: 30 bytes
		buf := new(bytes.Buffer)
		buf.WriteByte(0x01)
		buf.WriteString("vorbis")
		_ = binary.Write(buf, binary.LittleEndian, uint32(0)) // vorbis version 0
		buf.WriteByte(byte(channels))
		_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
		_ = binary.Write(buf, binary.LittleEndian, uint32(0))      // bitrate max
		_ = binary.Write(buf, binary.LittleEndian, uint32(192000)) // bitrate nominal
		_ = binary.Write(buf, binary.LittleEndian, uint32(0))      // bitrate min
		buf.WriteByte(0xB8) // blocksize 0 = 2^8, blocksize 1 = 2^11
		buf.WriteByte(0x01) // framing flag 1
		idPacket = buf.Bytes()

	default:
		return nil, ErrUnsupportedCodec
	}

	bosPages := CreateOggPagesFromPacket(idPacket, serialNum, 0, 0, true, false)

	// 2. Page 1: Initial Comment Header
	initVC := NewVorbisComment()
	initVC.Vendor = "AVibe Canonical Ogg Generator"
	initVC.Title = "Canonical Initial Stream"
	commentPacket, err := BuildVorbisCommentPacket(initVC, codec)
	if err != nil {
		return nil, err
	}
	commentPages := CreateOggPagesFromPacket(commentPacket, serialNum, 1, 0, false, false)

	currentSeq := uint32(1 + len(commentPages))

	// 3. For Vorbis: Setup Header (Page 2)
	var setupPages [][]byte
	if codec == OggCodecVorbis {
		setupBuf := new(bytes.Buffer)
		setupBuf.WriteByte(0x05)
		setupBuf.WriteString("vorbis")
		// Canonical minimal codebook/modes payload
		setupBuf.Write(make([]byte, 48))
		setupBuf.WriteByte(0x01) // framing bit
		setupPages = CreateOggPagesFromPacket(setupBuf.Bytes(), serialNum, currentSeq, 0, false, false)
		currentSeq += uint32(len(setupPages))
	}

	// 4. Audio Pages (20ms frames, e.g. 960 samples per frame at 48kHz)
	samplesPerFrame := sampleRate / 50 // 20ms
	totalFrames := int(durationSec / 0.02)
	if totalFrames < 1 {
		totalFrames = 1
	}

	var audioPages [][]byte
	var currentGranule uint64 = 0

	for f := 0; f < totalFrames; f++ {
		currentGranule += uint64(samplesPerFrame)
		isLast := f == totalFrames-1

		// Generate synthetic audio frame
		frameSize := 80
		frameBytes := make([]byte, frameSize)
		frameBytes[0] = 0x78 // standard Opus/Vorbis frame indicator
		for b := 1; b < frameSize; b++ {
			frameBytes[b] = byte((f*17 + b*31) % 256)
		}

		p := CreateOggPagesFromPacket(frameBytes, serialNum, currentSeq, currentGranule, false, isLast)
		audioPages = append(audioPages, p...)
		currentSeq += uint32(len(p))
	}

	// Combine all pages
	finalStream := new(bytes.Buffer)
	for _, p := range bosPages {
		finalStream.Write(p)
	}
	for _, p := range commentPages {
		finalStream.Write(p)
	}
	for _, p := range setupPages {
		finalStream.Write(p)
	}
	for _, p := range audioPages {
		finalStream.Write(p)
	}

	return finalStream.Bytes(), nil
}
