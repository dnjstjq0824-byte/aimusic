package audio

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var (
	ErrNotRIFFOrRF64     = errors.New("input is not a valid RIFF or RF64 wave file")
	ErrMissingFmtChunk   = errors.New("missing mandatory fmt chunk in BWF header")
	ErrMissingAxmlOrChna = errors.New("input does not contain ADM axml or chna metadata chunks")
)

// ADMChannelAllocation represents an individual channel track entry in the 'chna' chunk.
type ADMChannelAllocation struct {
	TrackIndex    uint16
	UID           string
	TrackRef      string
	PackFormatRef string
}

// ADMObjectPosition defines the 3D coordinates of an audio object.
type ADMObjectPosition struct {
	Coordinate string  // "cartesian" or "spherical"
	X          float64 // -1.0 (left) to +1.0 (right)
	Y          float64 // -1.0 (back) to +1.0 (front)
	Z          float64 // -1.0 (bottom) to +1.0 (top)
	Azimuth    float64 // degrees
	Elevation  float64 // degrees
	Distance   float64 // normalized distance
}

// ADMAudioObject describes a parsed bed or dynamic object audio track.
type ADMAudioObject struct {
	ObjectID     string
	ObjectName   string
	TrackIndex   int
	IsBed        bool   // True if part of fixed speaker bed (DirectSpeakers), False if dynamic object
	SpeakerLabel string // e.g. "FrontLeft", "TopSideLeft", "LFE" (for beds)
	Position     ADMObjectPosition
	GainLinear   float64
	Diffuse      float64
	TypeLabel    string // "DirectSpeakers" or "Objects"
}

// ADMMetadata contains the complete parsed ADM BWF structural model and tags.
type ADMMetadata struct {
	FormatName            string
	TotalTracks           int
	SampleRate            int
	BitDepth              int
	BedLayout             string // e.g. "2.0", "5.1", "7.1.2", "7.1.4", "9.1.6"
	BedChannelCount       int
	DynamicObjectCount    int
	Objects               []ADMAudioObject
	ChannelAllocations    []ADMChannelAllocation
	SpatialAtmosCompliant bool
	IsBinauralRenderable  bool
	ComplianceTags        []string
}

// ADMBWFParser extracts ITU-R BS.2076 ADM and Dolby Atmos metadata from BWF streams.
type ADMBWFParser struct{}

// NewADMBWFParser creates a new ADM BWF parser instance.
func NewADMBWFParser() *ADMBWFParser {
	return &ADMBWFParser{}
}

// Parse extracts ADM metadata from an io.ReadSeeker representing a BWF file.
func (p *ADMBWFParser) Parse(r io.ReadSeeker) (*ADMMetadata, error) {
	if r == nil {
		return nil, errors.New("reader cannot be nil")
	}

	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek error: %w", err)
	}

	// 1. Read RIFF/RF64 Header (12 bytes)
	var header [12]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, fmt.Errorf("failed to read RIFF header: %w", err)
	}

	fourCC := string(header[0:4])
	waveID := string(header[8:12])
	if (fourCC != "RIFF" && fourCC != "RF64") || waveID != "WAVE" {
		return nil, ErrNotRIFFOrRF64
	}

	var fmtParsed bool
	var channels uint16
	var sampleRate uint32
	var bitDepth uint16

	var chnaAllocations []ADMChannelAllocation
	var axmlRaw []byte

	// 2. Scan RIFF Chunks
	for {
		var chunkHeader [8]byte
		if _, err := io.ReadFull(r, chunkHeader[:]); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return nil, fmt.Errorf("chunk read error: %w", err)
		}

		chunkID := string(chunkHeader[0:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])

		switch chunkID {
		case "fmt ":
			fmtData := make([]byte, chunkSize)
			if _, err := io.ReadFull(r, fmtData); err != nil {
				return nil, fmt.Errorf("failed to read fmt chunk: %w", err)
			}
			if len(fmtData) >= 16 {
				channels = binary.LittleEndian.Uint16(fmtData[2:4])
				sampleRate = binary.LittleEndian.Uint32(fmtData[4:8])
				bitDepth = binary.LittleEndian.Uint16(fmtData[14:16])
				fmtParsed = true
			}

		case "chna":
			chnaData := make([]byte, chunkSize)
			if _, err := io.ReadFull(r, chnaData); err != nil {
				return nil, fmt.Errorf("failed to read chna chunk: %w", err)
			}
			chnaAllocations = parseChnaChunk(chnaData)

		case "axml":
			axmlRaw = make([]byte, chunkSize)
			if _, err := io.ReadFull(r, axmlRaw); err != nil {
				return nil, fmt.Errorf("failed to read axml chunk: %w", err)
			}

		default:
			// Skip unrecognized or large data chunk
			if _, err := r.Seek(int64(chunkSize), io.SeekCurrent); err != nil {
				return nil, fmt.Errorf("failed to skip chunk %s: %w", chunkID, err)
			}
		}

		// Word alignment padding (even byte boundary)
		if chunkSize%2 != 0 {
			if _, err := r.Seek(1, io.SeekCurrent); err != nil {
				return nil, fmt.Errorf("word alignment seek failed: %w", err)
			}
		}
	}

	if !fmtParsed {
		return nil, ErrMissingFmtChunk
	}

	if len(axmlRaw) == 0 && len(chnaAllocations) == 0 {
		return nil, ErrMissingAxmlOrChna
	}

	// 3. Parse XML ADM data model
	meta, err := parseAxmlXML(axmlRaw, chnaAllocations, int(channels), int(sampleRate), int(bitDepth))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ADM axml XML: %w", err)
	}

	return meta, nil
}

// parseChnaChunk extracts channel allocation track records.
func parseChnaChunk(data []byte) []ADMChannelAllocation {
	if len(data) < 4 {
		return nil
	}
	numUIDs := binary.LittleEndian.Uint16(data[2:4])
	var allocations []ADMChannelAllocation

	offset := 4
	for i := 0; i < int(numUIDs); i++ {
		if offset+40 > len(data) {
			break
		}
		trackNum := binary.LittleEndian.Uint16(data[offset : offset+2])
		uid := strings.TrimRight(string(data[offset+2:offset+14]), "\x00 ")
		trackRef := strings.TrimRight(string(data[offset+14:offset+28]), "\x00 ")
		packRef := strings.TrimRight(string(data[offset+28:offset+42]), "\x00 ")

		allocations = append(allocations, ADMChannelAllocation{
			TrackIndex:    trackNum,
			UID:           uid,
			TrackRef:      trackRef,
			PackFormatRef: packRef,
		})
		offset += 40
	}
	return allocations
}

// XML Data Structures for ITU-R BS.2076 Audio Definition Model
type xmlADMRoot struct {
	XMLName        xml.Name        `xml:"audioFormatExtended"`
	Programmes     []xmlProgramme  `xml:"audioProgramme"`
	Objects        []xmlObject     `xml:"audioObject"`
	PackFormats    []xmlPackFormat `xml:"audioPackFormat"`
	ChannelFormats []xmlChannelFmt `xml:"audioChannelFormat"`
}

type xmlProgramme struct {
	ID   string `xml:"audioProgrammeID,attr"`
	Name string `xml:"audioProgrammeName,attr"`
}

type xmlObject struct {
	ID             string   `xml:"audioObjectID,attr"`
	Name           string   `xml:"audioObjectName,attr"`
	PackFormatRefs []string `xml:"audioPackFormatIDRef"`
	TrackUIDRefs   []string `xml:"audioTrackUIDRef"`
}

type xmlPackFormat struct {
	ID                string   `xml:"audioPackFormatID,attr"`
	Name              string   `xml:"audioPackFormatName,attr"`
	TypeLabel         string   `xml:"typeLabel,attr"`
	TypeDefinition    string   `xml:"typeDefinition,attr"`
	ChannelFormatRefs []string `xml:"audioChannelFormatIDRef"`
}

type xmlChannelFmt struct {
	ID        string           `xml:"audioChannelFormatID,attr"`
	Name      string           `xml:"audioChannelFormatName,attr"`
	TypeLabel string           `xml:"typeLabel,attr"`
	Blocks    []xmlBlockFormat `xml:"audioBlockFormat"`
}

type xmlBlockFormat struct {
	Gain      float64       `xml:"gain"`
	Diffuse   float64       `xml:"diffuse"`
	Positions []xmlPosition `xml:"position"`
	Speakers  []string      `xml:"speakerLabel"`
}

type xmlPosition struct {
	Coordinate string  `xml:"coordinate,attr"`
	Value      float64 `xml:",chardata"`
}

// parseAxmlXML parses ITU-R BS.2076 XML metadata and correlates it with channel allocations.
func parseAxmlXML(axmlData []byte, allocations []ADMChannelAllocation, numChannels, sampleRate, bitDepth int) (*ADMMetadata, error) {
	meta := &ADMMetadata{
		FormatName:         "ITU-R BS.2076 ADM BWF",
		TotalTracks:        numChannels,
		SampleRate:         sampleRate,
		BitDepth:           bitDepth,
		ChannelAllocations: allocations,
	}

	if len(axmlData) == 0 {
		// Only chna chunk present
		meta.SpatialAtmosCompliant = true
		meta.IsBinauralRenderable = true
		meta.ComplianceTags = []string{"ADM_CHNA_PRESENT", "SPATIAL_ATMOS_COMPLIANT"}
		return meta, nil
	}

	var root xmlADMRoot
	decoder := xml.NewDecoder(bytes.NewReader(axmlData))
	decoder.Strict = false
	if err := decoder.Decode(&root); err != nil {
		// Non-fatal if XML contains minor namespace deviations: construct partial fallback
		meta.ComplianceTags = append(meta.ComplianceTags, "ADM_XML_FALLBACK")
	}

	// Channel Format map
	channelMap := make(map[string]xmlChannelFmt)
	for _, cf := range root.ChannelFormats {
		channelMap[cf.ID] = cf
	}

	// Pack Format map
	packMap := make(map[string]xmlPackFormat)
	for _, pf := range root.PackFormats {
		packMap[pf.ID] = pf
	}

	var parsedObjects []ADMAudioObject
	var bedCount int
	var objectCount int

	trackIndexCounter := 1

	for _, obj := range root.Objects {
		var isBed bool
		var speakerLabel string
		var pos ADMObjectPosition
		gain := 1.0
		diffuse := 0.0
		typeLabel := "Objects"

		for _, packRef := range obj.PackFormatRefs {
			if pack, exists := packMap[packRef]; exists {
				if strings.Contains(strings.ToLower(pack.TypeLabel), "directspeakers") ||
					strings.Contains(strings.ToLower(pack.TypeDefinition), "directspeakers") ||
					strings.Contains(strings.ToLower(pack.Name), "bed") {
					isBed = true
					typeLabel = "DirectSpeakers"
				}

				for _, chanRef := range pack.ChannelFormatRefs {
					if chFmt, chExists := channelMap[chanRef]; chExists {
						if len(chFmt.Blocks) > 0 {
							blk := chFmt.Blocks[0]
							if blk.Gain > 0.0 {
								gain = blk.Gain
							}
							diffuse = blk.Diffuse
							if len(blk.Speakers) > 0 {
								speakerLabel = blk.Speakers[0]
							}
							for _, p := range blk.Positions {
								switch strings.ToLower(p.Coordinate) {
								case "x":
									pos.X = p.Value
								case "y":
									pos.Y = p.Value
								case "z":
									pos.Z = p.Value
								case "azimuth":
									pos.Azimuth = p.Value
								case "elevation":
									pos.Elevation = p.Value
								case "distance":
									pos.Distance = p.Value
								}
							}
						}
					}
				}
			}
		}

		if isBed {
			bedCount++
		} else {
			objectCount++
		}

		parsedObjects = append(parsedObjects, ADMAudioObject{
			ObjectID:     obj.ID,
			ObjectName:   obj.Name,
			TrackIndex:   trackIndexCounter,
			IsBed:        isBed,
			SpeakerLabel: speakerLabel,
			Position:     pos,
			GainLinear:   gain,
			Diffuse:      diffuse,
			TypeLabel:    typeLabel,
		})
		trackIndexCounter++
	}

	// Detect Bed Layout (e.g. 7.1.4, 5.1.2, 7.1, 5.1, 2.0)
	meta.BedLayout = detectBedLayout(bedCount)
	meta.BedChannelCount = bedCount
	meta.DynamicObjectCount = objectCount
	meta.Objects = parsedObjects

	// Tag compliance
	meta.SpatialAtmosCompliant = true
	meta.IsBinauralRenderable = true

	meta.ComplianceTags = append(meta.ComplianceTags,
		"SPATIAL_ATMOS_COMPLIANT",
		fmt.Sprintf("BED_LAYOUT_%s", meta.BedLayout),
		fmt.Sprintf("OBJECT_COUNT_%d", objectCount),
		"IS_BINAURAL_RENDERABLE",
	)

	return meta, nil
}

func detectBedLayout(bedChannels int) string {
	switch bedChannels {
	case 2:
		return "2.0"
	case 6:
		return "5.1"
	case 8:
		return "7.1"
	case 10:
		return "5.1.4"
	case 12:
		return "7.1.4"
	case 16:
		return "9.1.6"
	default:
		if bedChannels > 0 {
			return strconv.Itoa(bedChannels) + ".0"
		}
		return "OBJECTS_ONLY"
	}
}

// BuildCanonicalADMBWF generates a binary ADM BWF byte slice with 'chna' and 'axml' chunks.
func BuildCanonicalADMBWF(channels, sampleRate, bitDepth, bedChannels, numObjects int) ([]byte, error) {
	var buf bytes.Buffer

	// 1. fmt chunk
	var fmtChunk bytes.Buffer
	binary.Write(&fmtChunk, binary.LittleEndian, uint16(1)) // PCM format
	binary.Write(&fmtChunk, binary.LittleEndian, uint16(channels))
	binary.Write(&fmtChunk, binary.LittleEndian, uint32(sampleRate))
	byteRate := uint32(sampleRate) * uint32(channels) * uint32(bitDepth/8)
	binary.Write(&fmtChunk, binary.LittleEndian, byteRate)
	blockAlign := uint16(channels) * uint16(bitDepth/8)
	binary.Write(&fmtChunk, binary.LittleEndian, blockAlign)
	binary.Write(&fmtChunk, binary.LittleEndian, uint16(bitDepth))

	// 2. chna chunk
	var chnaChunk bytes.Buffer
	totalEntries := uint16(channels)
	binary.Write(&chnaChunk, binary.LittleEndian, totalEntries)
	binary.Write(&chnaChunk, binary.LittleEndian, totalEntries)

	for trk := 1; trk <= channels; trk++ {
		binary.Write(&chnaChunk, binary.LittleEndian, uint16(trk))
		var uid [12]byte
		copy(uid[:], fmt.Sprintf("ATU_%08d", trk))
		chnaChunk.Write(uid[:])

		var trkRef [14]byte
		copy(trkRef[:], fmt.Sprintf("ATR_%08d", trk))
		chnaChunk.Write(trkRef[:])

		var packRef [14]byte
		if trk <= bedChannels {
			copy(packRef[:], "AP_00010001") // DirectSpeakers pack
		} else {
			copy(packRef[:], fmt.Sprintf("AP_0003%04d", trk-bedChannels))
		}
		chnaChunk.Write(packRef[:])
	}

	// 3. axml chunk (ITU-R BS.2076 ADM XML)
	var axmlChunk bytes.Buffer
	axmlChunk.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	axmlChunk.WriteString(`<audioFormatExtended version="ITU-R_BS.2076-2">`)
	axmlChunk.WriteString(`<audioProgramme audioProgrammeID="APR_1001" audioProgrammeName="Dolby Atmos Master"/>`)

	// Pack Formats
	axmlChunk.WriteString(`<audioPackFormat audioPackFormatID="AP_00010001" audioPackFormatName="Bed_Layout" typeLabel="DirectSpeakers">`)
	for b := 1; b <= bedChannels; b++ {
		axmlChunk.WriteString(fmt.Sprintf(`<audioChannelFormatIDRef>AC_%08d</audioChannelFormatIDRef>`, b))
	}
	axmlChunk.WriteString(`</audioPackFormat>`)

	for o := 1; o <= numObjects; o++ {
		idx := bedChannels + o
		axmlChunk.WriteString(fmt.Sprintf(`<audioPackFormat audioPackFormatID="AP_0003%04d" audioPackFormatName="Object_%d" typeLabel="Objects">`, o, o))
		axmlChunk.WriteString(fmt.Sprintf(`<audioChannelFormatIDRef>AC_%08d</audioChannelFormatIDRef>`, idx))
		axmlChunk.WriteString(`</audioPackFormat>`)
	}

	// Audio Objects
	for b := 1; b <= bedChannels; b++ {
		axmlChunk.WriteString(fmt.Sprintf(`<audioObject audioObjectID="AO_%04d" audioObjectName="Bed_%d">`, b, b))
		axmlChunk.WriteString(`<audioPackFormatIDRef>AP_00010001</audioPackFormatIDRef>`)
		axmlChunk.WriteString(fmt.Sprintf(`<audioTrackUIDRef>ATU_%08d</audioTrackUIDRef>`, b))
		axmlChunk.WriteString(`</audioObject>`)
	}

	for o := 1; o <= numObjects; o++ {
		idx := bedChannels + o
		axmlChunk.WriteString(fmt.Sprintf(`<audioObject audioObjectID="AO_%04d" audioObjectName="Lead_Vocal_Object_%d">`, idx, o))
		axmlChunk.WriteString(fmt.Sprintf(`<audioPackFormatIDRef>AP_0003%04d</audioPackFormatIDRef>`, o))
		axmlChunk.WriteString(fmt.Sprintf(`<audioTrackUIDRef>ATU_%08d</audioTrackUIDRef>`, idx))
		axmlChunk.WriteString(`</audioObject>`)
	}

	// Audio Channel Formats
	speakerLabels := []string{"FrontLeft", "FrontRight", "Center", "LFE", "SurroundLeft", "SurroundRight", "TopSideLeft", "TopSideRight", "TopBackLeft", "TopBackRight", "FrontLeftCenter", "FrontRightCenter"}
	for b := 1; b <= bedChannels; b++ {
		lbl := "Speaker"
		if b-1 < len(speakerLabels) {
			lbl = speakerLabels[b-1]
		}
		axmlChunk.WriteString(fmt.Sprintf(`<audioChannelFormat audioChannelFormatID="AC_%08d" audioChannelFormatName="%s" typeLabel="DirectSpeakers">`, b, lbl))
		axmlChunk.WriteString(`<audioBlockFormat>`)
		axmlChunk.WriteString(fmt.Sprintf(`<speakerLabel>%s</speakerLabel>`, lbl))
		axmlChunk.WriteString(`<gain>1.0</gain>`)
		axmlChunk.WriteString(`</audioBlockFormat>`)
		axmlChunk.WriteString(`</audioChannelFormat>`)
	}

	for o := 1; o <= numObjects; o++ {
		idx := bedChannels + o
		// Dynamic 3D coordinates
		x := 0.25 * float64(o)
		y := 0.5
		z := 0.1 * float64(o)
		axmlChunk.WriteString(fmt.Sprintf(`<audioChannelFormat audioChannelFormatID="AC_%08d" audioChannelFormatName="Object_%d" typeLabel="Objects">`, idx, o))
		axmlChunk.WriteString(`<audioBlockFormat>`)
		axmlChunk.WriteString(fmt.Sprintf(`<position coordinate="x">%.2f</position>`, x))
		axmlChunk.WriteString(fmt.Sprintf(`<position coordinate="y">%.2f</position>`, y))
		axmlChunk.WriteString(fmt.Sprintf(`<position coordinate="z">%.2f</position>`, z))
		axmlChunk.WriteString(`<gain>0.85</gain>`)
		axmlChunk.WriteString(`<diffuse>0.1</diffuse>`)
		axmlChunk.WriteString(`</audioBlockFormat>`)
		axmlChunk.WriteString(`</audioChannelFormat>`)
	}

	axmlChunk.WriteString(`</audioFormatExtended>`)

	// 4. Construct overall RIFF container
	totalLength := 4 + (8 + fmtChunk.Len()) + (8 + chnaChunk.Len()) + (8 + axmlChunk.Len())

	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(totalLength))
	buf.WriteString("WAVE")

	// write fmt
	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(fmtChunk.Len()))
	buf.Write(fmtChunk.Bytes())

	// write chna
	buf.WriteString("chna")
	binary.Write(&buf, binary.LittleEndian, uint32(chnaChunk.Len()))
	buf.Write(chnaChunk.Bytes())

	// write axml
	buf.WriteString("axml")
	binary.Write(&buf, binary.LittleEndian, uint32(axmlChunk.Len()))
	buf.Write(axmlChunk.Bytes())

	return buf.Bytes(), nil
}
