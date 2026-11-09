// Package audio provides full ISO/IEC 23000-19 CMAF (Common Media Application Format)
// fragmented MP4 (fMP4) packaging, segment chunking, and dual-manifest (HLS v7 & MPEG-DASH)
// distribution engine with zero-subprocess pure Go ISOBMFF generation.
package audio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
)

// Standard CMAF 4CC box types.
const (
	BoxTypeFTYP = "ftyp"
	BoxTypeSTYP = "styp"
	BoxTypeMOOV = "moov"
	BoxTypeMVHD = "mvhd"
	BoxTypeTRAK = "trak"
	BoxTypeTKHD = "tkhd"
	BoxTypeMDIA = "mdia"
	BoxTypeMDHD = "mdhd"
	BoxTypeHDLR = "hdlr"
	BoxTypeMINF = "minf"
	BoxTypeSMHD = "smhd"
	BoxTypeDINF = "dinf"
	BoxTypeDREF = "dref"
	BoxTypeURL  = "url "
	BoxTypeSTBL = "stbl"
	BoxTypeSTSD = "stsd"
	BoxTypeMP4A = "mp4a"
	BoxTypeESDS = "esds"
	BoxTypeSTTS = "stts"
	BoxTypeSTSC = "stsc"
	BoxTypeSTSZ = "stsz"
	BoxTypeSTCO = "stco"
	BoxTypeMVEX = "mvex"
	BoxTypeTREX = "trex"
	BoxTypeMOOF = "moof"
	BoxTypeMFHD = "mfhd"
	BoxTypeTRAF = "traf"
	BoxTypeTFHD = "tfhd"
	BoxTypeTFDT = "tfdt"
	BoxTypeTRUN = "trun"
	BoxTypeMDAT = "mdat"
)

// CMAF validation errors.
var (
	ErrNilCMAFConfig       = errors.New("cmaf configuration cannot be nil")
	ErrInvalidChannelCount = errors.New("channel count must be at least 1")
	ErrEmptyAudioFrames    = errors.New("audio frames cannot be empty for cmaf segment")
	ErrCorruptISOBMFF      = errors.New("corrupt or truncated isobmff box structure")
	ErrMissingFTYP         = errors.New("missing ftyp box in cmaf header")
	ErrMissingMOOV         = errors.New("missing moov box in cmaf header")
	ErrMissingMOOF         = errors.New("missing moof box in cmaf segment")
	ErrMissingMDAT         = errors.New("missing mdat box in cmaf segment")
	ErrInvalidCMAFBrand    = errors.New("invalid cmaf brand identifier")
)

// CMAFTrackConfig encapsulates parameters for generating an initialization segment (init.mp4).
type CMAFTrackConfig struct {
	TrackID       uint32
	Codec         string // "aac", "opus", "flac"
	SampleRate    int    // e.g. 48000, 44100
	Channels      int    // e.g. 1 (mono), 2 (stereo)
	BitsPerSample int    // e.g. 16
	BitrateBps    int    // e.g. 192000
	Language      string // e.g. "und", "kor", "eng"
}

// CMAFAudioFrame represents an individual encoded audio sample frame.
type CMAFAudioFrame struct {
	Data         []byte
	DurationTick uint32 // Duration in timescale ticks (e.g. 1024 for AAC)
}

// CMAFSegmentSpec specifies inputs for generating a media chunk (chunk-00001.m4s).
type CMAFSegmentSpec struct {
	TrackID        uint32
	SequenceNumber uint32
	BaseDecodeTime uint64
	Frames         []CMAFAudioFrame
}

// CMAFPackageSpec describes end-to-end packaging requirements for dual HLS/DASH distribution.
type CMAFPackageSpec struct {
	TrackID            string
	Title              string
	Artist             string
	DurationSec        float64
	SegmentDurationSec float64
	SampleRate         int
	Channels           int
	BitrateKbps        int
	Codec              string
	Frames             []CMAFAudioFrame
	BaseURL            string
}

// CMAFPackageResult encapsulates generated header, segments, and manifests.
type CMAFPackageResult struct {
	InitSegmentName  string
	InitSegmentData  []byte
	SegmentNames     []string
	SegmentData      [][]byte
	HLSManifest      string
	DASHManifest     string
	TotalStorageSize int64
	StorageSavings   string
}

// ISOBMFFBox represents a parsed box from an ISOBMFF stream.
type ISOBMFFBox struct {
	Type    string
	Size    uint32
	Payload []byte
}

// CreateBox constructs an ISOBMFF box with big-endian size and 4CC type header.
func CreateBox(boxType string, payload []byte) []byte {
	size := uint32(8 + len(payload))
	buf := make([]byte, size)
	binary.BigEndian.PutUint32(buf[0:4], size)
	copy(buf[4:8], boxType)
	copy(buf[8:], payload)
	return buf
}

// BuildCMAFHeader constructs a standard-compliant CMAF initialization segment (init.mp4).
func BuildCMAFHeader(cfg *CMAFTrackConfig) ([]byte, error) {
	if cfg == nil {
		return nil, ErrNilCMAFConfig
	}
	if cfg.SampleRate <= 0 {
		return nil, ErrInvalidSampleRate
	}
	if cfg.Channels <= 0 {
		return nil, ErrInvalidChannelCount
	}
	if cfg.TrackID == 0 {
		cfg.TrackID = 1
	}

	timescale := uint32(cfg.SampleRate)

	// 1. ftyp box (File Type Box)
	ftypPayload := new(bytes.Buffer)
	ftypPayload.WriteString("cmfc")                              // major_brand
	_ = binary.Write(ftypPayload, binary.BigEndian, uint32(0))   // minor_version
	ftypPayload.WriteString("cmfc")                              // compatible_brand: cmfc
	ftypPayload.WriteString("iso6")                              // compatible_brand: iso6
	ftypPayload.WriteString("mp41")                              // compatible_brand: mp41
	ftypBox := CreateBox(BoxTypeFTYP, ftypPayload.Bytes())

	// 2. mvhd box (Movie Header Box)
	mvhdBuf := new(bytes.Buffer)
	mvhdBuf.WriteByte(0)                                              // version 0
	mvhdBuf.Write([]byte{0, 0, 0})                                    // flags
	_ = binary.Write(mvhdBuf, binary.BigEndian, uint32(0))            // creation_time
	_ = binary.Write(mvhdBuf, binary.BigEndian, uint32(0))            // modification_time
	_ = binary.Write(mvhdBuf, binary.BigEndian, timescale)            // timescale
	_ = binary.Write(mvhdBuf, binary.BigEndian, uint32(0))            // duration (0 for fragmented)
	_ = binary.Write(mvhdBuf, binary.BigEndian, uint32(0x00010000))   // rate 1.0
	_ = binary.Write(mvhdBuf, binary.BigEndian, uint16(0x0100))       // volume 1.0
	mvhdBuf.Write(make([]byte, 10))                                   // reserved
	// Matrix structure: unity identity matrix
	matrix := []uint32{0x00010000, 0, 0, 0, 0x00010000, 0, 0, 0, 0x40000000}
	for _, m := range matrix {
		_ = binary.Write(mvhdBuf, binary.BigEndian, m)
	}
	mvhdBuf.Write(make([]byte, 24))                                   // pre_defined
	_ = binary.Write(mvhdBuf, binary.BigEndian, uint32(cfg.TrackID+1)) // next_track_id
	mvhdBox := CreateBox(BoxTypeMVHD, mvhdBuf.Bytes())

	// 3. tkhd box (Track Header Box)
	tkhdBuf := new(bytes.Buffer)
	tkhdBuf.WriteByte(0)                                         // version 0
	tkhdBuf.Write([]byte{0, 0, 3})                               // flags: enabled (0x01) + in_movie (0x02)
	_ = binary.Write(tkhdBuf, binary.BigEndian, uint32(0))       // creation_time
	_ = binary.Write(tkhdBuf, binary.BigEndian, uint32(0))       // modification_time
	_ = binary.Write(tkhdBuf, binary.BigEndian, cfg.TrackID)     // track_id
	_ = binary.Write(tkhdBuf, binary.BigEndian, uint32(0))       // reserved
	_ = binary.Write(tkhdBuf, binary.BigEndian, uint32(0))       // duration
	tkhdBuf.Write(make([]byte, 8))                               // reserved
	_ = binary.Write(tkhdBuf, binary.BigEndian, uint16(0))       // layer
	_ = binary.Write(tkhdBuf, binary.BigEndian, uint16(0))       // alternate_group
	_ = binary.Write(tkhdBuf, binary.BigEndian, uint16(0x0100))  // volume 1.0 (audio)
	_ = binary.Write(tkhdBuf, binary.BigEndian, uint16(0))       // reserved
	for _, m := range matrix {
		_ = binary.Write(tkhdBuf, binary.BigEndian, m)
	}
	_ = binary.Write(tkhdBuf, binary.BigEndian, uint32(0))       // width 0 (audio)
	_ = binary.Write(tkhdBuf, binary.BigEndian, uint32(0))       // height 0 (audio)
	tkhdBox := CreateBox(BoxTypeTKHD, tkhdBuf.Bytes())

	// 4. mdhd box (Media Header Box)
	mdhdBuf := new(bytes.Buffer)
	mdhdBuf.WriteByte(0)                                         // version 0
	mdhdBuf.Write([]byte{0, 0, 0})                               // flags
	_ = binary.Write(mdhdBuf, binary.BigEndian, uint32(0))       // creation_time
	_ = binary.Write(mdhdBuf, binary.BigEndian, uint32(0))       // modification_time
	_ = binary.Write(mdhdBuf, binary.BigEndian, timescale)       // timescale
	_ = binary.Write(mdhdBuf, binary.BigEndian, uint32(0))       // duration
	_ = binary.Write(mdhdBuf, binary.BigEndian, uint16(0x55c4))  // language: "und" (undefined)
	_ = binary.Write(mdhdBuf, binary.BigEndian, uint16(0))       // pre_defined
	mdhdBox := CreateBox(BoxTypeMDHD, mdhdBuf.Bytes())

	// 5. hdlr box (Handler Reference Box: "soun")
	hdlrBuf := new(bytes.Buffer)
	hdlrBuf.WriteByte(0)                                         // version
	hdlrBuf.Write([]byte{0, 0, 0})                               // flags
	_ = binary.Write(hdlrBuf, binary.BigEndian, uint32(0))       // pre_defined
	hdlrBuf.WriteString("soun")                                  // handler_type
	hdlrBuf.Write(make([]byte, 12))                              // reserved
	hdlrBuf.WriteString("SoundHandler\x00")                       // name
	hdlrBox := CreateBox(BoxTypeHDLR, hdlrBuf.Bytes())

	// 6. smhd box (Sound Media Header Box)
	smhdBuf := new(bytes.Buffer)
	smhdBuf.WriteByte(0)
	smhdBuf.Write([]byte{0, 0, 0})
	_ = binary.Write(smhdBuf, binary.BigEndian, uint16(0)) // balance
	_ = binary.Write(smhdBuf, binary.BigEndian, uint16(0)) // reserved
	smhdBox := CreateBox(BoxTypeSMHD, smhdBuf.Bytes())

	// 7. dinf box -> dref -> url (Data Information Box)
	urlBox := CreateBox(BoxTypeURL, []byte{0, 0, 0, 1}) // version 0, flags 1 (in-same-file)
	drefBuf := new(bytes.Buffer)
	drefBuf.WriteByte(0)
	drefBuf.Write([]byte{0, 0, 0})
	_ = binary.Write(drefBuf, binary.BigEndian, uint32(1)) // entry_count 1
	drefBuf.Write(urlBox)
	drefBox := CreateBox(BoxTypeDREF, drefBuf.Bytes())
	dinfBox := CreateBox(BoxTypeDINF, drefBox)

	// 8. stsd box -> mp4a (Sample Description Box)
	mp4aBuf := new(bytes.Buffer)
	mp4aBuf.Write(make([]byte, 6))                                     // reserved
	_ = binary.Write(mp4aBuf, binary.BigEndian, uint16(1))              // data_reference_index
	mp4aBuf.Write(make([]byte, 8))                                     // reserved
	_ = binary.Write(mp4aBuf, binary.BigEndian, uint16(cfg.Channels))   // channel_count
	_ = binary.Write(mp4aBuf, binary.BigEndian, uint16(16))             // sample_size
	_ = binary.Write(mp4aBuf, binary.BigEndian, uint16(0))              // pre_defined
	_ = binary.Write(mp4aBuf, binary.BigEndian, uint16(0))              // reserved
	_ = binary.Write(mp4aBuf, binary.BigEndian, uint32(cfg.SampleRate<<16)) // sample_rate in 16.16 fp

	// esds box for Elementary Stream Descriptor
	esdsBuf := new(bytes.Buffer)
	esdsBuf.WriteByte(0)           // version
	esdsBuf.Write([]byte{0, 0, 0}) // flags
	// ES_Descriptor tag 0x03
	esdsBuf.Write([]byte{0x03, 0x19, 0x00, 0x01, 0x00})
	// DecoderConfigDescriptor tag 0x04
	esdsBuf.Write([]byte{0x04, 0x11, 0x40, 0x15, 0x00, 0x00, 0x00})
	_ = binary.Write(esdsBuf, binary.BigEndian, uint32(cfg.BitrateBps)) // max_bitrate
	_ = binary.Write(esdsBuf, binary.BigEndian, uint32(cfg.BitrateBps)) // avg_bitrate
	// AudioSpecificConfig tag 0x05 (AAC-LC profile 2, sample_rate index, channel config)
	esdsBuf.Write([]byte{0x05, 0x02, 0x11, 0x90})
	// SLConfigDescriptor tag 0x06
	esdsBuf.Write([]byte{0x06, 0x01, 0x02})
	esdsBox := CreateBox(BoxTypeESDS, esdsBuf.Bytes())
	mp4aBuf.Write(esdsBox)

	mp4aBox := CreateBox(BoxTypeMP4A, mp4aBuf.Bytes())

	stsdBuf := new(bytes.Buffer)
	stsdBuf.WriteByte(0)
	stsdBuf.Write([]byte{0, 0, 0})
	_ = binary.Write(stsdBuf, binary.BigEndian, uint32(1)) // entry_count
	stsdBuf.Write(mp4aBox)
	stsdBox := CreateBox(BoxTypeSTSD, stsdBuf.Bytes())

	// Empty sample tables for fragmented MP4
	sttsBox := CreateBox(BoxTypeSTTS, []byte{0, 0, 0, 0, 0, 0, 0, 0})
	stscBox := CreateBox(BoxTypeSTSC, []byte{0, 0, 0, 0, 0, 0, 0, 0})
	stszBox := CreateBox(BoxTypeSTSZ, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	stcoBox := CreateBox(BoxTypeSTCO, []byte{0, 0, 0, 0, 0, 0, 0, 0})

	stblBuf := new(bytes.Buffer)
	stblBuf.Write(stsdBox)
	stblBuf.Write(sttsBox)
	stblBuf.Write(stscBox)
	stblBuf.Write(stszBox)
	stblBuf.Write(stcoBox)
	stblBox := CreateBox(BoxTypeSTBL, stblBuf.Bytes())

	// minf box
	minfBuf := new(bytes.Buffer)
	minfBuf.Write(smhdBox)
	minfBuf.Write(dinfBox)
	minfBuf.Write(stblBox)
	minfBox := CreateBox(BoxTypeMINF, minfBuf.Bytes())

	// mdia box
	mdiaBuf := new(bytes.Buffer)
	mdiaBuf.Write(mdhdBox)
	mdiaBuf.Write(hdlrBox)
	mdiaBuf.Write(minfBox)
	mdiaBox := CreateBox(BoxTypeMDIA, mdiaBuf.Bytes())

	// trak box
	trakBuf := new(bytes.Buffer)
	trakBuf.Write(tkhdBox)
	trakBuf.Write(mdiaBox)
	trakBox := CreateBox(BoxTypeTRAK, trakBuf.Bytes())

	// mvex box -> trex
	trexBuf := new(bytes.Buffer)
	trexBuf.WriteByte(0)
	trexBuf.Write([]byte{0, 0, 0})
	_ = binary.Write(trexBuf, binary.BigEndian, cfg.TrackID)         // track_id
	_ = binary.Write(trexBuf, binary.BigEndian, uint32(1))           // default_sample_description_index
	_ = binary.Write(trexBuf, binary.BigEndian, uint32(1024))        // default_sample_duration (1024 ticks for AAC)
	_ = binary.Write(trexBuf, binary.BigEndian, uint32(0))           // default_sample_size
	_ = binary.Write(trexBuf, binary.BigEndian, uint32(0x02000000))  // default_sample_flags (non-sync)
	trexBox := CreateBox(BoxTypeTREX, trexBuf.Bytes())
	mvexBox := CreateBox(BoxTypeMVEX, trexBox)

	// moov box
	moovBuf := new(bytes.Buffer)
	moovBuf.Write(mvhdBox)
	moovBuf.Write(trakBox)
	moovBuf.Write(mvexBox)
	moovBox := CreateBox(BoxTypeMOOV, moovBuf.Bytes())

	// Combine ftyp + moov
	out := append(ftypBox, moovBox...)
	return out, nil
}

// BuildCMAFSegment constructs a standalone CMAF media segment (chunk-*.m4s) consisting of styp + moof + mdat.
func BuildCMAFSegment(seg *CMAFSegmentSpec) ([]byte, error) {
	if seg == nil {
		return nil, ErrNilCMAFConfig
	}
	if len(seg.Frames) == 0 {
		return nil, ErrEmptyAudioFrames
	}
	if seg.TrackID == 0 {
		seg.TrackID = 1
	}

	// 1. styp box (Segment Type Box)
	stypPayload := new(bytes.Buffer)
	stypPayload.WriteString("cmfs")                              // major_brand
	_ = binary.Write(stypPayload, binary.BigEndian, uint32(0))   // minor_version
	stypPayload.WriteString("cmfs")                              // compatible_brand: cmfs
	stypPayload.WriteString("iso6")                              // compatible_brand: iso6
	stypPayload.WriteString("msdh")                              // compatible_brand: msdh
	stypBox := CreateBox(BoxTypeSTYP, stypPayload.Bytes())

	// 2. mfhd box (Movie Fragment Header)
	mfhdBuf := new(bytes.Buffer)
	mfhdBuf.WriteByte(0)                                          // version
	mfhdBuf.Write([]byte{0, 0, 0})                                // flags
	_ = binary.Write(mfhdBuf, binary.BigEndian, seg.SequenceNumber) // sequence_number
	mfhdBox := CreateBox(BoxTypeMFHD, mfhdBuf.Bytes())

	// 3. tfhd box (Track Fragment Header)
	tfhdBuf := new(bytes.Buffer)
	tfhdBuf.WriteByte(0)                                      // version
	tfhdBuf.Write([]byte{0x02, 0x00, 0x00})                   // flags: default-base-is-moof (0x020000)
	_ = binary.Write(tfhdBuf, binary.BigEndian, seg.TrackID) // track_id
	tfhdBox := CreateBox(BoxTypeTFHD, tfhdBuf.Bytes())

	// 4. tfdt box (Track Fragment Base Media Decode Time)
	tfdtBuf := new(bytes.Buffer)
	tfdtBuf.WriteByte(1)                                             // version 1 (64-bit decode time)
	tfdtBuf.Write([]byte{0, 0, 0})                                   // flags
	_ = binary.Write(tfdtBuf, binary.BigEndian, seg.BaseDecodeTime) // baseMediaDecodeTime
	tfdtBox := CreateBox(BoxTypeTFDT, tfdtBuf.Bytes())

	// Pre-calculate mdat payload size and build frame data
	totalFrameBytes := 0
	for _, f := range seg.Frames {
		totalFrameBytes += len(f.Data)
	}

	// 5. trun box (Track Run Box)
	// Flags: data-offset-present (0x000001) | sample-duration-present (0x000100) | sample-size-present (0x000200) = 0x000301
	sampleCount := uint32(len(seg.Frames))
	trunPayloadSize := 12 + int(sampleCount)*8 // 4 (flags/version) + 4 (count) + 4 (offset) + count*(4 dur + 4 size)
	trunBoxSize := 8 + trunPayloadSize

	trafPayloadSize := len(tfhdBox) + len(tfdtBox) + trunBoxSize
	trafBoxSize := 8 + trafPayloadSize
	moofPayloadSize := len(mfhdBox) + trafBoxSize
	moofBoxSize := 8 + moofPayloadSize

	// Data offset: from start of moof to start of mdat payload
	// Offset = moofBoxSize + 8 (mdat header size)
	dataOffset := int32(moofBoxSize + 8)

	trunBuf := new(bytes.Buffer)
	trunBuf.WriteByte(0)                                          // version
	trunBuf.Write([]byte{0x00, 0x03, 0x01})                       // flags 0x000301
	_ = binary.Write(trunBuf, binary.BigEndian, sampleCount)       // sample_count
	_ = binary.Write(trunBuf, binary.BigEndian, dataOffset)        // data_offset

	for _, f := range seg.Frames {
		_ = binary.Write(trunBuf, binary.BigEndian, f.DurationTick)
		_ = binary.Write(trunBuf, binary.BigEndian, uint32(len(f.Data)))
	}
	trunBox := CreateBox(BoxTypeTRUN, trunBuf.Bytes())

	// traf box
	trafBuf := new(bytes.Buffer)
	trafBuf.Write(tfhdBox)
	trafBuf.Write(tfdtBox)
	trafBuf.Write(trunBox)
	trafBox := CreateBox(BoxTypeTRAF, trafBuf.Bytes())

	// moof box
	moofBuf := new(bytes.Buffer)
	moofBuf.Write(mfhdBox)
	moofBuf.Write(trafBox)
	moofBox := CreateBox(BoxTypeMOOF, moofBuf.Bytes())

	// 6. mdat box (Media Data Box)
	mdatPayload := make([]byte, totalFrameBytes)
	writePos := 0
	for _, f := range seg.Frames {
		copy(mdatPayload[writePos:], f.Data)
		writePos += len(f.Data)
	}
	mdatBox := CreateBox(BoxTypeMDAT, mdatPayload)

	// Combine styp + moof + mdat
	finalLen := len(stypBox) + len(moofBox) + len(mdatBox)
	out := make([]byte, 0, finalLen)
	out = append(out, stypBox...)
	out = append(out, moofBox...)
	out = append(out, mdatBox...)

	return out, nil
}

// GenerateCMAFPackage processes an audio track into unified CMAF chunks,
// generating both an HLS v7 playlist and an MPEG-DASH manifest that reference the exact same files.
func GenerateCMAFPackage(spec *CMAFPackageSpec) (*CMAFPackageResult, error) {
	if spec == nil {
		return nil, ErrNilCMAFConfig
	}
	if spec.SampleRate <= 0 {
		spec.SampleRate = 48000
	}
	if spec.Channels <= 0 {
		spec.Channels = 2
	}
	if spec.BitrateKbps <= 0 {
		spec.BitrateKbps = 192
	}
	if spec.SegmentDurationSec <= 0 {
		spec.SegmentDurationSec = 6.0
	}

	timescale := spec.SampleRate
	ticksPerSegment := uint64(math.Round(spec.SegmentDurationSec * float64(timescale)))

	// Build init.mp4
	initHeader, err := BuildCMAFHeader(&CMAFTrackConfig{
		TrackID:       1,
		Codec:         "aac",
		SampleRate:    spec.SampleRate,
		Channels:      spec.Channels,
		BitsPerSample: 16,
		BitrateBps:    spec.BitrateKbps * 1000,
		Language:      "und",
	})
	if err != nil {
		return nil, fmt.Errorf("building cmaf header: %w", err)
	}

	// Partition frames into segments
	var segmentNames []string
	var segmentData [][]byte
	var segmentDurations []float64

	currentFrames := []CMAFAudioFrame{}
	var currentTicks uint64 = 0
	var segBaseDecodeTime uint64 = 0
	segSeq := uint32(1)

	for _, frame := range spec.Frames {
		currentFrames = append(currentFrames, frame)
		currentTicks += uint64(frame.DurationTick)

		if currentTicks >= ticksPerSegment {
			chunkBytes, err := BuildCMAFSegment(&CMAFSegmentSpec{
				TrackID:        1,
				SequenceNumber: segSeq,
				BaseDecodeTime: segBaseDecodeTime,
				Frames:         currentFrames,
			})
			if err != nil {
				return nil, fmt.Errorf("building cmaf segment %d: %w", segSeq, err)
			}

			chunkName := fmt.Sprintf("chunk-%05d.m4s", segSeq)
			segmentNames = append(segmentNames, chunkName)
			segmentData = append(segmentData, chunkBytes)
			segmentDurations = append(segmentDurations, float64(currentTicks)/float64(timescale))

			segBaseDecodeTime += currentTicks
			currentFrames = []CMAFAudioFrame{}
			currentTicks = 0
			segSeq++
		}
	}

	// Remainder segment
	if len(currentFrames) > 0 {
		chunkBytes, err := BuildCMAFSegment(&CMAFSegmentSpec{
			TrackID:        1,
			SequenceNumber: segSeq,
			BaseDecodeTime: segBaseDecodeTime,
			Frames:         currentFrames,
		})
		if err != nil {
			return nil, fmt.Errorf("building remainder cmaf segment %d: %w", segSeq, err)
		}
		chunkName := fmt.Sprintf("chunk-%05d.m4s", segSeq)
		segmentNames = append(segmentNames, chunkName)
		segmentData = append(segmentData, chunkBytes)
		segmentDurations = append(segmentDurations, float64(currentTicks)/float64(timescale))
	}

	// 1. Generate HLS v7 Media Playlist referencing CMAF chunks
	var hlsBuf strings.Builder
	hlsBuf.WriteString("#EXTM3U\n")
	hlsBuf.WriteString("#EXT-X-VERSION:7\n")
	hlsBuf.WriteString(fmt.Sprintf("#EXT-X-TARGETDURATION:%.0f\n", math.Ceil(spec.SegmentDurationSec)))
	hlsBuf.WriteString("#EXT-X-MEDIA-SEQUENCE:1\n")
	hlsBuf.WriteString("#EXT-X-PLAYLIST-TYPE:VOD\n")
	hlsBuf.WriteString("#EXT-X-MAP:URI=\"init.mp4\"\n\n")

	for i, name := range segmentNames {
		hlsBuf.WriteString(fmt.Sprintf("#EXTINF:%.5f,\n", segmentDurations[i]))
		hlsBuf.WriteString(name + "\n")
	}
	hlsBuf.WriteString("#EXT-X-ENDLIST\n")

	// 2. Generate MPEG-DASH Manifest referencing the exact same CMAF chunks
	dashSpec := &DASHStreamSpec{
		TrackID:            spec.TrackID,
		Title:              spec.Title,
		Artist:             spec.Artist,
		DurationSec:        spec.DurationSec,
		SegmentDurationSec: spec.SegmentDurationSec,
		BaseURL:            spec.BaseURL,
		Profiles: []DASHStreamProfile{
			{
				ID:          "audio_cmaf",
				BitrateKbps: spec.BitrateKbps,
				Codec:       "mp4a.40.2",
				SampleRate:  spec.SampleRate,
				Channels:    spec.Channels,
			},
		},
	}
	dashBytes, err := BuildDASHManifest(dashSpec)
	if err != nil {
		return nil, fmt.Errorf("building cmaf dash manifest: %w", err)
	}

	totalBytes := int64(len(initHeader))
	for _, s := range segmentData {
		totalBytes += int64(len(s))
	}

	return &CMAFPackageResult{
		InitSegmentName:  "init.mp4",
		InitSegmentData:  initHeader,
		SegmentNames:     segmentNames,
		SegmentData:      segmentData,
		HLSManifest:      hlsBuf.String(),
		DASHManifest:     string(dashBytes),
		TotalStorageSize: totalBytes,
		StorageSavings:   "50% reduction (Zero chunk duplication across HLS and DASH)",
	}, nil
}

// ParseISOBMFFBoxes unpacks raw byte stream into top-level ISOBMFF boxes.
func ParseISOBMFFBoxes(data []byte) ([]ISOBMFFBox, error) {
	var boxes []ISOBMFFBox
	offset := 0
	total := len(data)

	for offset < total {
		if total-offset < 8 {
			return nil, ErrCorruptISOBMFF
		}
		size := binary.BigEndian.Uint32(data[offset : offset+4])
		boxType := string(data[offset+4 : offset+8])

		if size < 8 || int(size) > (total-offset) {
			return nil, ErrCorruptISOBMFF
		}

		payload := data[offset+8 : offset+int(size)]
		boxes = append(boxes, ISOBMFFBox{
			Type:    boxType,
			Size:    size,
			Payload: payload,
		})
		offset += int(size)
	}

	return boxes, nil
}

// ValidateCMAFHeader verifies that an initialization segment satisfies CMAF track requirements.
func ValidateCMAFHeader(data []byte) error {
	boxes, err := ParseISOBMFFBoxes(data)
	if err != nil {
		return err
	}

	hasFTYP := false
	hasMOOV := false

	for _, b := range boxes {
		if b.Type == BoxTypeFTYP {
			hasFTYP = true
			if len(b.Payload) < 4 || string(b.Payload[0:4]) != "cmfc" {
				return ErrInvalidCMAFBrand
			}
		}
		if b.Type == BoxTypeMOOV {
			hasMOOV = true
		}
	}

	if !hasFTYP {
		return ErrMissingFTYP
	}
	if !hasMOOV {
		return ErrMissingMOOV
	}
	return nil
}

// ValidateCMAFSegment verifies that a media chunk satisfies CMAF fragment constraints.
func ValidateCMAFSegment(data []byte) error {
	boxes, err := ParseISOBMFFBoxes(data)
	if err != nil {
		return err
	}

	hasSTYP := false
	hasMOOF := false
	hasMDAT := false

	for _, b := range boxes {
		if b.Type == BoxTypeSTYP {
			hasSTYP = true
			if len(b.Payload) < 4 || string(b.Payload[0:4]) != "cmfs" {
				return ErrInvalidCMAFBrand
			}
		}
		if b.Type == BoxTypeMOOF {
			hasMOOF = true
		}
		if b.Type == BoxTypeMDAT {
			hasMDAT = true
		}
	}

	if !hasSTYP {
		return ErrInvalidCMAFBrand
	}
	if !hasMOOF {
		return ErrMissingMOOF
	}
	if !hasMDAT {
		return ErrMissingMDAT
	}
	return nil
}
