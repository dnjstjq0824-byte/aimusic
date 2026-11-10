// Package audio provides pure Go ISO/IEC 14496-12 (ISOBMFF) and Apple QuickTime
// metadata atom injection, parsing, and chunk offset updating for AI transparency disclosure.
package audio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

// Standard Apple iTunes/QuickTime ilst metadata 4CC atom types.
const (
	AtomTypeTitle       = "\xa9nam"
	AtomTypeArtist      = "\xa9ART"
	AtomTypeAlbum       = "\xa9alb"
	AtomTypeComposer    = "\xa9wrt"
	AtomTypeYear        = "\xa9day"
	AtomTypeComment     = "\xa9cmt"
	AtomTypeGenre       = "\xa9gen"
	AtomTypeCopyright   = "cprt"
	AtomTypeDescription = "desc"
	AtomTypeTrackNumber = "trkn"
	AtomTypeDiskNumber  = "disk"
	AtomTypeCoverArt    = "covr"
	AtomTypeFreeform    = "----"

	AtomTypeUDTA = "udta"
	AtomTypeMETA = "meta"
	AtomTypeHDLR = "hdlr"
	AtomTypeILST = "ilst"
	AtomTypeDATA = "data"
	AtomTypeMEAN = "mean"
	AtomTypeNAME = "name"
)

// Freeform iTunes metadata keys for AI Transparency.
const (
	AppleMeanDomain             = "com.apple.iTunes"
	TagAIArticle31Disclosure    = "AI_ARTICLE_31_DISCLOSURE"
	TagAIModelName              = "AI_MODEL"
	TagAIPrompt                 = "AI_PROMPT"
	TagAISeed                   = "AI_SEED"
	TagAIGenerator              = "AI_GENERATOR"
	TagAIIsSynthetic            = "AI_IS_SYNTHETIC"
	TagAICreationTimestamp      = "AI_TIMESTAMP"
)

// Error definitions for MP4 metadata processing.
var (
	ErrNilMP4Metadata   = errors.New("mp4 metadata object cannot be nil")
	ErrInvalidMP4Data   = errors.New("invalid or truncated mp4 byte stream")
	ErrMissingMoovBox   = errors.New("missing mandatory moov container box in mp4 stream")
	ErrMissingIlstBox   = errors.New("missing ilst metadata box in mp4 stream")
	ErrUnsupportedAtom  = errors.New("unsupported or malformed ilst item atom")
)

// MP4Metadata holds comprehensive metadata for QuickTime / Apple Music playback,
// including regulatory AI transparency disclosures.
type MP4Metadata struct {
	Title         string
	Artist        string
	Album         string
	Composer      string
	Year          string
	Copyright     string
	Comment       string
	Genre         string
	Description   string
	TrackNumber   uint16
	TotalTracks   uint16
	DiscNumber    uint16
	TotalDiscs    uint16
	CoverArt      []byte
	CoverMimeType string // "image/jpeg" or "image/png"

	// Article 31 AI Transparency Fields (in compliance with Korean Framework Act on AI & EU AI Act)
	AIArticle31Disclosure string
	AIModelName           string
	AIPrompt              string
	AISeed                string
	AIGenerator           string
	AIIsSynthetic         bool
	AICreationTimestamp   string

	// Custom user/system freeform tags
	CustomFields map[string]string
}

// DefaultArticle31Disclosure returns the official statutory transparency text for AI-generated music.
func DefaultArticle31Disclosure() string {
	return "본 음원은 인공지능 발전과 신뢰 기반 조성 등에 관한 기본법 제31조(인공지능 투명성 확보) 및 글로벌 AI 투명성 규범에 따라 생성형 AI 기술을 활용하여 제작되었음을 투명하게 고지합니다. (AI Generation Disclosure: This audio content was generated or synthesized using artificial intelligence in compliance with Article 31 transparency regulations.)"
}

// MP4BoxNode represents an arbitrary node in an ISOBMFF box hierarchy.
type MP4BoxNode struct {
	Type     string
	Size     uint32
	Payload  []byte
	Children []*MP4BoxNode
	IsFullBox bool
	Version  uint8
	Flags    uint32
}

// isContainerBox returns true if the box type represents a container of child boxes.
func isContainerBox(boxType string) bool {
	switch boxType {
	case "moov", "trak", "mdia", "minf", "dinf", "stbl", AtomTypeUDTA, AtomTypeILST,
		AtomTypeTitle, AtomTypeArtist, AtomTypeAlbum, AtomTypeComposer, AtomTypeYear,
		AtomTypeComment, AtomTypeGenre, AtomTypeCopyright, AtomTypeDescription,
		AtomTypeTrackNumber, AtomTypeDiskNumber, AtomTypeCoverArt, AtomTypeFreeform:
		return true
	default:
		return false
	}
}

// parseBoxTree recursively unpacks ISOBMFF boxes from a byte slice into MP4BoxNode structures.
func parseBoxTree(data []byte) ([]*MP4BoxNode, error) {
	var nodes []*MP4BoxNode
	offset := 0
	total := len(data)

	for offset < total {
		if total-offset < 8 {
			return nil, ErrInvalidMP4Data
		}
		boxSize := binary.BigEndian.Uint32(data[offset : offset+4])
		boxType := string(data[offset+4 : offset+8])

		if boxSize < 8 || int(boxSize) > (total-offset) {
			return nil, ErrInvalidMP4Data
		}

		rawPayload := data[offset+8 : offset+int(boxSize)]
		node := &MP4BoxNode{
			Type: boxType,
			Size: boxSize,
		}

		if boxType == AtomTypeMETA {
			// ISO/IEC 14496-12 FullBox meta has 4 bytes version/flags
			if len(rawPayload) >= 4 {
				node.IsFullBox = true
				node.Version = rawPayload[0]
				node.Flags = binary.BigEndian.Uint32([]byte{0, rawPayload[1], rawPayload[2], rawPayload[3]})
				childData := rawPayload[4:]
				children, err := parseBoxTree(childData)
				if err == nil {
					node.Children = children
				} else {
					node.Payload = rawPayload
				}
			} else {
				node.Payload = rawPayload
			}
		} else if isContainerBox(boxType) {
			children, err := parseBoxTree(rawPayload)
			if err == nil && len(children) > 0 {
				node.Children = children
			} else {
				node.Payload = rawPayload
			}
		} else {
			node.Payload = rawPayload
		}

		nodes = append(nodes, node)
		offset += int(boxSize)
	}

	return nodes, nil
}

// Serialize converts the MP4BoxNode tree back to standard-compliant ISOBMFF bytes.
func (n *MP4BoxNode) Serialize() []byte {
	if n.Type == AtomTypeMETA {
		buf := new(bytes.Buffer)
		if n.IsFullBox {
			flagsBytes := []byte{n.Version, byte(n.Flags >> 16), byte(n.Flags >> 8), byte(n.Flags)}
			buf.Write(flagsBytes)
		}
		for _, ch := range n.Children {
			buf.Write(ch.Serialize())
		}
		return CreateBox(AtomTypeMETA, buf.Bytes())
	}

	if len(n.Children) > 0 {
		buf := new(bytes.Buffer)
		for _, ch := range n.Children {
			buf.Write(ch.Serialize())
		}
		return CreateBox(n.Type, buf.Bytes())
	}

	return CreateBox(n.Type, n.Payload)
}

// shiftChunkOffsets recursively traverses the box tree under moov and increments
// all stco and co64 chunk offsets by delta if their original offset >= moovStartOffset.
func shiftChunkOffsets(node *MP4BoxNode, moovStartOffset int64, delta int64) {
	if node.Type == "stco" {
		if len(node.Payload) >= 8 {
			entryCount := binary.BigEndian.Uint32(node.Payload[4:8])
			for i := uint32(0); i < entryCount; i++ {
				idx := 8 + i*4
				if idx+4 <= uint32(len(node.Payload)) {
					curOffset := binary.BigEndian.Uint32(node.Payload[idx : idx+4])
					if int64(curOffset) >= moovStartOffset {
						newOffset := int64(curOffset) + delta
						if newOffset >= 0 && newOffset <= math.MaxUint32 {
							binary.BigEndian.PutUint32(node.Payload[idx:idx+4], uint32(newOffset))
						}
					}
				}
			}
		}
		return
	}

	if node.Type == "co64" {
		if len(node.Payload) >= 8 {
			entryCount := binary.BigEndian.Uint32(node.Payload[4:8])
			for i := uint32(0); i < entryCount; i++ {
				idx := 8 + i*8
				if idx+8 <= uint32(len(node.Payload)) {
					curOffset := binary.BigEndian.Uint64(node.Payload[idx : idx+8])
					if int64(curOffset) >= moovStartOffset {
						newOffset := int64(curOffset) + delta
						if newOffset >= 0 {
							binary.BigEndian.PutUint64(node.Payload[idx:idx+8], uint64(newOffset))
						}
					}
				}
			}
		}
		return
	}

	for _, child := range node.Children {
		shiftChunkOffsets(child, moovStartOffset, delta)
	}
}

// createTextDataAtom builds a QuickTime metadata atom containing a single UTF-8 data box.
func createTextDataAtom(atomType, text string) *MP4BoxNode {
	if text == "" {
		return nil
	}
	dataPayload := make([]byte, 8+len(text))
	binary.BigEndian.PutUint32(dataPayload[0:4], 1) // Type 1 = UTF-8 text
	binary.BigEndian.PutUint32(dataPayload[4:8], 0) // Locale 0
	copy(dataPayload[8:], []byte(text))

	dataNode := &MP4BoxNode{
		Type:    AtomTypeDATA,
		Payload: dataPayload,
	}

	return &MP4BoxNode{
		Type:     atomType,
		Children: []*MP4BoxNode{dataNode},
	}
}

// createTrackNumberAtom builds a trkn metadata atom with track and total numbers.
func createTrackNumberAtom(track, total uint16) *MP4BoxNode {
	if track == 0 && total == 0 {
		return nil
	}
	dataPayload := make([]byte, 16)
	binary.BigEndian.PutUint32(dataPayload[0:4], 0) // Implicit type
	binary.BigEndian.PutUint32(dataPayload[4:8], 0)
	binary.BigEndian.PutUint16(dataPayload[8:10], 0)
	binary.BigEndian.PutUint16(dataPayload[10:12], track)
	binary.BigEndian.PutUint16(dataPayload[12:14], total)
	binary.BigEndian.PutUint16(dataPayload[14:16], 0)

	dataNode := &MP4BoxNode{
		Type:    AtomTypeDATA,
		Payload: dataPayload,
	}
	return &MP4BoxNode{
		Type:     AtomTypeTrackNumber,
		Children: []*MP4BoxNode{dataNode},
	}
}

// createDiskNumberAtom builds a disk metadata atom with disc and total disc numbers.
func createDiskNumberAtom(disc, total uint16) *MP4BoxNode {
	if disc == 0 && total == 0 {
		return nil
	}
	dataPayload := make([]byte, 14)
	binary.BigEndian.PutUint32(dataPayload[0:4], 0)
	binary.BigEndian.PutUint32(dataPayload[4:8], 0)
	binary.BigEndian.PutUint16(dataPayload[8:10], 0)
	binary.BigEndian.PutUint16(dataPayload[10:12], disc)
	binary.BigEndian.PutUint16(dataPayload[12:14], total)

	dataNode := &MP4BoxNode{
		Type:    AtomTypeDATA,
		Payload: dataPayload,
	}
	return &MP4BoxNode{
		Type:     AtomTypeDiskNumber,
		Children: []*MP4BoxNode{dataNode},
	}
}

// createCoverArtAtom builds a covr metadata atom with embedded JPEG or PNG cover image.
func createCoverArtAtom(imgBytes []byte, mimeType string) *MP4BoxNode {
	if len(imgBytes) == 0 {
		return nil
	}
	typeCode := uint32(13) // JPEG default
	if mimeType == "image/png" || (len(imgBytes) >= 8 && bytes.Equal(imgBytes[:8], []byte("\x89PNG\r\n\x1a\n"))) {
		typeCode = 14 // PNG
	}

	dataPayload := make([]byte, 8+len(imgBytes))
	binary.BigEndian.PutUint32(dataPayload[0:4], typeCode)
	binary.BigEndian.PutUint32(dataPayload[4:8], 0)
	copy(dataPayload[8:], imgBytes)

	dataNode := &MP4BoxNode{
		Type:    AtomTypeDATA,
		Payload: dataPayload,
	}
	return &MP4BoxNode{
		Type:     AtomTypeCoverArt,
		Children: []*MP4BoxNode{dataNode},
	}
}

// createFreeformAtom builds an Apple reverse-DNS '----' atom with mean, name, and data children.
func createFreeformAtom(domain, name, value string) *MP4BoxNode {
	if domain == "" {
		domain = AppleMeanDomain
	}
	if name == "" || value == "" {
		return nil
	}

	meanPayload := make([]byte, 4+len(domain))
	copy(meanPayload[4:], []byte(domain))
	meanNode := &MP4BoxNode{
		Type:    AtomTypeMEAN,
		Payload: meanPayload,
	}

	namePayload := make([]byte, 4+len(name))
	copy(namePayload[4:], []byte(name))
	nameNode := &MP4BoxNode{
		Type:    AtomTypeNAME,
		Payload: namePayload,
	}

	dataPayload := make([]byte, 8+len(value))
	binary.BigEndian.PutUint32(dataPayload[0:4], 1) // Type 1 = UTF-8 text
	binary.BigEndian.PutUint32(dataPayload[4:8], 0) // Locale 0
	copy(dataPayload[8:], []byte(value))
	dataNode := &MP4BoxNode{
		Type:    AtomTypeDATA,
		Payload: dataPayload,
	}

	return &MP4BoxNode{
		Type:     AtomTypeFreeform,
		Children: []*MP4BoxNode{meanNode, nameNode, dataNode},
	}
}

// buildIlstAtoms compiles all metadata fields and freeform AI transparency atoms into an ilst node.
func buildIlstAtoms(meta *MP4Metadata, existingIlst []*MP4BoxNode) *MP4BoxNode {
	atomMap := make(map[string]*MP4BoxNode)
	freeformMap := make(map[string]*MP4BoxNode)

	// Ingest existing atoms to preserve unmodified fields
	for _, atom := range existingIlst {
		if atom.Type == AtomTypeFreeform {
			var nameVal string
			for _, ch := range atom.Children {
				if ch.Type == AtomTypeNAME && len(ch.Payload) > 4 {
					nameVal = string(ch.Payload[4:])
				}
			}
			if nameVal != "" {
				freeformMap[nameVal] = atom
			}
		} else {
			atomMap[atom.Type] = atom
		}
	}

	// Update or add standard fields
	if atom := createTextDataAtom(AtomTypeTitle, meta.Title); atom != nil {
		atomMap[AtomTypeTitle] = atom
	}
	if atom := createTextDataAtom(AtomTypeArtist, meta.Artist); atom != nil {
		atomMap[AtomTypeArtist] = atom
	}
	if atom := createTextDataAtom(AtomTypeAlbum, meta.Album); atom != nil {
		atomMap[AtomTypeAlbum] = atom
	}
	if atom := createTextDataAtom(AtomTypeComposer, meta.Composer); atom != nil {
		atomMap[AtomTypeComposer] = atom
	}
	if atom := createTextDataAtom(AtomTypeYear, meta.Year); atom != nil {
		atomMap[AtomTypeYear] = atom
	}
	if atom := createTextDataAtom(AtomTypeCopyright, meta.Copyright); atom != nil {
		atomMap[AtomTypeCopyright] = atom
	}
	if atom := createTextDataAtom(AtomTypeComment, meta.Comment); atom != nil {
		atomMap[AtomTypeComment] = atom
	}
	if atom := createTextDataAtom(AtomTypeGenre, meta.Genre); atom != nil {
		atomMap[AtomTypeGenre] = atom
	}
	if atom := createTextDataAtom(AtomTypeDescription, meta.Description); atom != nil {
		atomMap[AtomTypeDescription] = atom
	}
	if atom := createTrackNumberAtom(meta.TrackNumber, meta.TotalTracks); atom != nil {
		atomMap[AtomTypeTrackNumber] = atom
	}
	if atom := createDiskNumberAtom(meta.DiscNumber, meta.TotalDiscs); atom != nil {
		atomMap[AtomTypeDiskNumber] = atom
	}
	if atom := createCoverArtAtom(meta.CoverArt, meta.CoverMimeType); atom != nil {
		atomMap[AtomTypeCoverArt] = atom
	}

	// AI Transparency Fields
	article31 := meta.AIArticle31Disclosure
	if article31 == "" && (meta.AIIsSynthetic || meta.AIModelName != "") {
		article31 = DefaultArticle31Disclosure()
	}
	if article31 != "" {
		freeformMap[TagAIArticle31Disclosure] = createFreeformAtom(AppleMeanDomain, TagAIArticle31Disclosure, article31)
	}
	if meta.AIModelName != "" {
		freeformMap[TagAIModelName] = createFreeformAtom(AppleMeanDomain, TagAIModelName, meta.AIModelName)
	}
	if meta.AIPrompt != "" {
		freeformMap[TagAIPrompt] = createFreeformAtom(AppleMeanDomain, TagAIPrompt, meta.AIPrompt)
	}
	if meta.AISeed != "" {
		freeformMap[TagAISeed] = createFreeformAtom(AppleMeanDomain, TagAISeed, meta.AISeed)
	}
	if meta.AIGenerator != "" {
		freeformMap[TagAIGenerator] = createFreeformAtom(AppleMeanDomain, TagAIGenerator, meta.AIGenerator)
	}
	syntheticVal := "false"
	if meta.AIIsSynthetic {
		syntheticVal = "true"
	}
	freeformMap[TagAIIsSynthetic] = createFreeformAtom(AppleMeanDomain, TagAIIsSynthetic, syntheticVal)

	timestamp := meta.AICreationTimestamp
	if timestamp == "" && meta.AIIsSynthetic {
		timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	if timestamp != "" {
		freeformMap[TagAICreationTimestamp] = createFreeformAtom(AppleMeanDomain, TagAICreationTimestamp, timestamp)
	}

	for k, v := range meta.CustomFields {
		if k != "" && v != "" {
			freeformMap[k] = createFreeformAtom(AppleMeanDomain, k, v)
		}
	}

	var children []*MP4BoxNode
	for _, atom := range atomMap {
		children = append(children, atom)
	}
	for _, atom := range freeformMap {
		children = append(children, atom)
	}

	return &MP4BoxNode{
		Type:     AtomTypeILST,
		Children: children,
	}
}

// InjectMP4IlstMetadata injects or updates QuickTime/iTunes ilst metadata atoms into an MP4/M4A stream,
// updating stco/co64 chunk offsets when the moov box expands or contracts.
func InjectMP4IlstMetadata(mp4Data []byte, meta *MP4Metadata) ([]byte, error) {
	if meta == nil {
		return nil, ErrNilMP4Metadata
	}
	if len(mp4Data) < 16 {
		return nil, ErrInvalidMP4Data
	}

	topNodes, err := parseBoxTree(mp4Data)
	if err != nil {
		return nil, fmt.Errorf("parsing top-level mp4 boxes: %w", err)
	}

	var moovNode *MP4BoxNode
	var moovStartOffset int64 = -1
	var mdatStartOffset int64 = -1
	var currentOffset int64 = 0

	for _, node := range topNodes {
		switch node.Type {
		case BoxTypeMOOV:
			moovNode = node
			moovStartOffset = currentOffset
		case BoxTypeMDAT:
			mdatStartOffset = currentOffset
		}
		currentOffset += int64(node.Size)
	}

	if moovNode == nil {
		return nil, ErrMissingMoovBox
	}

	oldMoovLen := int64(moovNode.Size)

	// Locate or construct udta box under moov
	var udtaNode *MP4BoxNode
	for _, ch := range moovNode.Children {
		if ch.Type == AtomTypeUDTA {
			udtaNode = ch
			break
		}
	}
	if udtaNode == nil {
		udtaNode = &MP4BoxNode{
			Type: AtomTypeUDTA,
		}
		moovNode.Children = append(moovNode.Children, udtaNode)
	}

	// Locate or construct meta box under udta
	var metaNode *MP4BoxNode
	for _, ch := range udtaNode.Children {
		if ch.Type == AtomTypeMETA {
			metaNode = ch
			break
		}
	}
	if metaNode == nil {
		metaNode = &MP4BoxNode{
			Type:      AtomTypeMETA,
			IsFullBox: true,
			Version:   0,
			Flags:     0,
		}
		udtaNode.Children = append(udtaNode.Children, metaNode)
	}

	// Ensure hdlr box exists under meta (required by QuickTime/Apple Music for ilst metadata)
	var hdlrNode *MP4BoxNode
	for _, ch := range metaNode.Children {
		if ch.Type == AtomTypeHDLR {
			hdlrNode = ch
			break
		}
	}
	if hdlrNode == nil {
		// hdlr payload: 4 bytes version/flags (0), 4 bytes pre_defined (0),
		// 4 bytes handler_type ("mdir"), 4 bytes manufacturer ("appl"),
		// 8 bytes reserved (0), 1 byte null terminator for name.
		hdlrBuf := new(bytes.Buffer)
		hdlrBuf.Write([]byte{0, 0, 0, 0})        // version 0, flags 0
		hdlrBuf.Write([]byte{0, 0, 0, 0})        // pre_defined
		hdlrBuf.WriteString("mdir")              // handler_type = mdir
		hdlrBuf.WriteString("appl")              // manufacturer = appl
		hdlrBuf.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0}) // reserved
		hdlrBuf.WriteByte(0)                     // name = ""
		hdlrNode = &MP4BoxNode{
			Type:    AtomTypeHDLR,
			Payload: hdlrBuf.Bytes(),
		}
		metaNode.Children = append([]*MP4BoxNode{hdlrNode}, metaNode.Children...)
	}

	// Locate existing ilst if present to preserve other tags
	var existingIlst []*MP4BoxNode
	var ilstIndex = -1
	for idx, ch := range metaNode.Children {
		if ch.Type == AtomTypeILST {
			existingIlst = ch.Children
			ilstIndex = idx
			break
		}
	}

	newIlstNode := buildIlstAtoms(meta, existingIlst)
	if ilstIndex >= 0 {
		metaNode.Children[ilstIndex] = newIlstNode
	} else {
		metaNode.Children = append(metaNode.Children, newIlstNode)
	}

	// Measure new moov size and delta
	tentativeMoovBytes := moovNode.Serialize()
	newMoovLen := int64(len(tentativeMoovBytes))
	delta := newMoovLen - oldMoovLen

	// Update stco/co64 chunk offsets if moov precedes mdat
	if moovStartOffset < mdatStartOffset && delta != 0 {
		shiftChunkOffsets(moovNode, moovStartOffset, delta)
	}

	// Final serialization of all top-level boxes
	finalBuf := new(bytes.Buffer)
	for _, node := range topNodes {
		finalBuf.Write(node.Serialize())
	}

	return finalBuf.Bytes(), nil
}

// ExtractMP4IlstMetadata parses an MP4/M4A byte stream and extracts all ilst metadata,
// including Article 31 AI transparency disclosures and custom freeform atoms.
func ExtractMP4IlstMetadata(mp4Data []byte) (*MP4Metadata, error) {
	if len(mp4Data) < 16 {
		return nil, ErrInvalidMP4Data
	}

	topNodes, err := parseBoxTree(mp4Data)
	if err != nil {
		return nil, fmt.Errorf("parsing top-level mp4 boxes: %w", err)
	}

	var moovNode *MP4BoxNode
	for _, node := range topNodes {
		if node.Type == BoxTypeMOOV {
			moovNode = node
			break
		}
	}
	if moovNode == nil {
		return nil, ErrMissingMoovBox
	}

	var udtaNode *MP4BoxNode
	for _, ch := range moovNode.Children {
		if ch.Type == AtomTypeUDTA {
			udtaNode = ch
			break
		}
	}
	if udtaNode == nil {
		return nil, ErrMissingIlstBox
	}

	var metaNode *MP4BoxNode
	for _, ch := range udtaNode.Children {
		if ch.Type == AtomTypeMETA {
			metaNode = ch
			break
		}
	}
	if metaNode == nil {
		return nil, ErrMissingIlstBox
	}

	var ilstNode *MP4BoxNode
	for _, ch := range metaNode.Children {
		if ch.Type == AtomTypeILST {
			ilstNode = ch
			break
		}
	}
	if ilstNode == nil {
		return nil, ErrMissingIlstBox
	}

	meta := &MP4Metadata{
		CustomFields: make(map[string]string),
	}

	for _, item := range ilstNode.Children {
		switch item.Type {
		case AtomTypeTitle:
			meta.Title = extractStringData(item)
		case AtomTypeArtist:
			meta.Artist = extractStringData(item)
		case AtomTypeAlbum:
			meta.Album = extractStringData(item)
		case AtomTypeComposer:
			meta.Composer = extractStringData(item)
		case AtomTypeYear:
			meta.Year = extractStringData(item)
		case AtomTypeCopyright:
			meta.Copyright = extractStringData(item)
		case AtomTypeComment:
			meta.Comment = extractStringData(item)
		case AtomTypeGenre:
			meta.Genre = extractStringData(item)
		case AtomTypeDescription:
			meta.Description = extractStringData(item)
		case AtomTypeTrackNumber:
			if dataNode := findChildByType(item, AtomTypeDATA); dataNode != nil && len(dataNode.Payload) >= 16 {
				meta.TrackNumber = binary.BigEndian.Uint16(dataNode.Payload[10:12])
				meta.TotalTracks = binary.BigEndian.Uint16(dataNode.Payload[12:14])
			}
		case AtomTypeDiskNumber:
			if dataNode := findChildByType(item, AtomTypeDATA); dataNode != nil && len(dataNode.Payload) >= 14 {
				meta.DiscNumber = binary.BigEndian.Uint16(dataNode.Payload[10:12])
				meta.TotalDiscs = binary.BigEndian.Uint16(dataNode.Payload[12:14])
			}
		case AtomTypeCoverArt:
			if dataNode := findChildByType(item, AtomTypeDATA); dataNode != nil && len(dataNode.Payload) >= 8 {
				typeCode := binary.BigEndian.Uint32(dataNode.Payload[0:4])
				switch typeCode {
				case 13:
					meta.CoverMimeType = "image/jpeg"
				case 14:
					meta.CoverMimeType = "image/png"
				}
				meta.CoverArt = make([]byte, len(dataNode.Payload)-8)
				copy(meta.CoverArt, dataNode.Payload[8:])
			}
		case AtomTypeFreeform:
			nameNode := findChildByType(item, AtomTypeNAME)
			dataNode := findChildByType(item, AtomTypeDATA)
			if nameNode != nil && dataNode != nil && len(nameNode.Payload) > 4 && len(dataNode.Payload) >= 8 {
				keyName := string(nameNode.Payload[4:])
				valText := string(dataNode.Payload[8:])
				meta.CustomFields[keyName] = valText

				switch keyName {
				case TagAIArticle31Disclosure:
					meta.AIArticle31Disclosure = valText
				case TagAIModelName:
					meta.AIModelName = valText
				case TagAIPrompt:
					meta.AIPrompt = valText
				case TagAISeed:
					meta.AISeed = valText
				case TagAIGenerator:
					meta.AIGenerator = valText
				case TagAIIsSynthetic:
					meta.AIIsSynthetic = strings.ToLower(valText) == "true"
				case TagAICreationTimestamp:
					meta.AICreationTimestamp = valText
				}
			}
		}
	}

	return meta, nil
}

// extractStringData retrieves UTF-8 text from a standard QuickTime metadata item data box.
func extractStringData(atom *MP4BoxNode) string {
	dataNode := findChildByType(atom, AtomTypeDATA)
	if dataNode != nil && len(dataNode.Payload) >= 8 {
		return string(dataNode.Payload[8:])
	}
	return ""
}

// findChildByType searches immediate children of a box for a specific 4CC type.
func findChildByType(parent *MP4BoxNode, boxType string) *MP4BoxNode {
	for _, ch := range parent.Children {
		if ch.Type == boxType {
			return ch
		}
	}
	return nil
}

// BuildCanonicalTestMP4 generates a fully compliant, unfragmented MP4 audio file
// containing ftyp, moov, audio track with stco chunk table, and real mdat payload.
func BuildCanonicalTestMP4(durationSec float64, sampleRate, channels int) ([]byte, error) {
	if sampleRate <= 0 {
		sampleRate = 48000
	}
	if channels <= 0 {
		channels = 2
	}
	if durationSec <= 0 {
		durationSec = 1.0
	}

	// 1. ftyp box
	ftypPayload := new(bytes.Buffer)
	ftypPayload.WriteString("M4A ")                 // major brand
	_ = binary.Write(ftypPayload, binary.BigEndian, uint32(0)) // minor version
	ftypPayload.WriteString("M4A ")                 // compatible brand 1
	ftypPayload.WriteString("mp42")                 // compatible brand 2
	ftypPayload.WriteString("isom")                 // compatible brand 3
	ftypBox := CreateBox(BoxTypeFTYP, ftypPayload.Bytes())

	// 2. Audio track specifications
	frameSamples := 1024
	totalSamples := int(math.Ceil(durationSec * float64(sampleRate)))
	numFrames := totalSamples / frameSamples
	if numFrames < 1 {
		numFrames = 1
	}
	frameByteSize := 256
	totalMediaBytes := numFrames * frameByteSize

	// Generate deterministic synthetic audio frames
	mediaPayload := make([]byte, totalMediaBytes)
	for i := 0; i < totalMediaBytes; i++ {
		mediaPayload[i] = byte((i*37 + 101) % 256)
	}

	// Construct moov box components
	mvhdBuf := new(bytes.Buffer)
	mvhdBuf.Write([]byte{0, 0, 0, 0}) // version 0, flags 0
	_ = binary.Write(mvhdBuf, binary.BigEndian, uint32(0)) // creation time
	_ = binary.Write(mvhdBuf, binary.BigEndian, uint32(0)) // modification time
	_ = binary.Write(mvhdBuf, binary.BigEndian, uint32(sampleRate)) // timescale
	_ = binary.Write(mvhdBuf, binary.BigEndian, uint32(numFrames*frameSamples)) // duration
	_ = binary.Write(mvhdBuf, binary.BigEndian, uint32(0x00010000)) // rate 1.0
	_ = binary.Write(mvhdBuf, binary.BigEndian, uint16(0x0100))     // volume 1.0
	mvhdBuf.Write(make([]byte, 10)) // reserved
	mvhdBuf.Write([]byte{
		0x00, 0x01, 0x00, 0x00, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0x00, 0x01, 0x00, 0x00, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0x40, 0x00, 0x00, 0x00,
	}) // unity matrix
	mvhdBuf.Write(make([]byte, 24)) // pre_defined
	_ = binary.Write(mvhdBuf, binary.BigEndian, uint32(2)) // next_track_id
	mvhdBox := CreateBox(BoxTypeMVHD, mvhdBuf.Bytes())

	tkhdBuf := new(bytes.Buffer)
	tkhdBuf.Write([]byte{0, 0, 0, 7}) // track enabled, in movie, in preview
	_ = binary.Write(tkhdBuf, binary.BigEndian, uint32(0)) // creation time
	_ = binary.Write(tkhdBuf, binary.BigEndian, uint32(0)) // modification time
	_ = binary.Write(tkhdBuf, binary.BigEndian, uint32(1)) // track ID 1
	_ = binary.Write(tkhdBuf, binary.BigEndian, uint32(0)) // reserved
	_ = binary.Write(tkhdBuf, binary.BigEndian, uint32(numFrames*frameSamples)) // duration
	tkhdBuf.Write(make([]byte, 8)) // reserved
	_ = binary.Write(tkhdBuf, binary.BigEndian, uint16(0)) // layer
	_ = binary.Write(tkhdBuf, binary.BigEndian, uint16(0)) // alternate group
	_ = binary.Write(tkhdBuf, binary.BigEndian, uint16(0x0100)) // volume 1.0
	tkhdBuf.Write(make([]byte, 2)) // reserved
	tkhdBuf.Write([]byte{
		0x00, 0x01, 0x00, 0x00, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0x00, 0x01, 0x00, 0x00, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0x40, 0x00, 0x00, 0x00,
	}) // unity matrix
	_ = binary.Write(tkhdBuf, binary.BigEndian, uint32(0)) // width
	_ = binary.Write(tkhdBuf, binary.BigEndian, uint32(0)) // height
	tkhdBox := CreateBox(BoxTypeTKHD, tkhdBuf.Bytes())

	mdhdBuf := new(bytes.Buffer)
	mdhdBuf.Write([]byte{0, 0, 0, 0})
	_ = binary.Write(mdhdBuf, binary.BigEndian, uint32(0))
	_ = binary.Write(mdhdBuf, binary.BigEndian, uint32(0))
	_ = binary.Write(mdhdBuf, binary.BigEndian, uint32(sampleRate))
	_ = binary.Write(mdhdBuf, binary.BigEndian, uint32(numFrames*frameSamples))
	_ = binary.Write(mdhdBuf, binary.BigEndian, uint16(0x15c7)) // "und" language code
	_ = binary.Write(mdhdBuf, binary.BigEndian, uint16(0))
	mdhdBox := CreateBox(BoxTypeMDHD, mdhdBuf.Bytes())

	hdlrBuf := new(bytes.Buffer)
	hdlrBuf.Write([]byte{0, 0, 0, 0})
	_ = binary.Write(hdlrBuf, binary.BigEndian, uint32(0))
	hdlrBuf.WriteString("soun") // sound track
	hdlrBuf.Write(make([]byte, 12))
	hdlrBuf.WriteString("SoundHandler\x00")
	hdlrBox := CreateBox(BoxTypeHDLR, hdlrBuf.Bytes())

	smhdBox := CreateBox(BoxTypeSMHD, []byte{0, 0, 0, 0, 0, 0, 0, 0})

	urlBox := CreateBox(BoxTypeURL, []byte{0, 0, 0, 1})
	drefBuf := new(bytes.Buffer)
	drefBuf.Write([]byte{0, 0, 0, 0})
	_ = binary.Write(drefBuf, binary.BigEndian, uint32(1))
	drefBuf.Write(urlBox)
	drefBox := CreateBox(BoxTypeDREF, drefBuf.Bytes())
	dinfBox := CreateBox(BoxTypeDINF, drefBox)

	// Sample description box with mp4a
	esdsBuf := new(bytes.Buffer)
	esdsBuf.Write([]byte{0, 0, 0, 0}) // version 0, flags 0
	esdsBuf.Write([]byte{0x03, 0x19, 0x00, 0x01, 0x00}) // ES_DescrTag
	esdsBuf.Write([]byte{0x04, 0x11, 0x40, 0x15, 0x00, 0x00, 0x00, 0x00, 0x02, 0xee, 0x00, 0x00, 0x02, 0xee, 0x00})
	esdsBuf.Write([]byte{0x05, 0x02, 0x11, 0x90}) // AudioSpecificConfig (AAC-LC, 48kHz, 2ch)
	esdsBuf.Write([]byte{0x06, 0x01, 0x02})
	esdsBox := CreateBox(BoxTypeESDS, esdsBuf.Bytes())

	mp4aBuf := new(bytes.Buffer)
	mp4aBuf.Write(make([]byte, 6)) // reserved
	_ = binary.Write(mp4aBuf, binary.BigEndian, uint16(1)) // data_reference_index
	_ = binary.Write(mp4aBuf, binary.BigEndian, uint32(0)) // reserved
	_ = binary.Write(mp4aBuf, binary.BigEndian, uint32(0)) // reserved
	_ = binary.Write(mp4aBuf, binary.BigEndian, uint16(channels))
	_ = binary.Write(mp4aBuf, binary.BigEndian, uint16(16)) // 16-bit
	_ = binary.Write(mp4aBuf, binary.BigEndian, uint16(0))
	_ = binary.Write(mp4aBuf, binary.BigEndian, uint16(0))
	_ = binary.Write(mp4aBuf, binary.BigEndian, uint32(sampleRate<<16))
	mp4aBuf.Write(esdsBox)
	mp4aBox := CreateBox(BoxTypeMP4A, mp4aBuf.Bytes())

	stsdBuf := new(bytes.Buffer)
	stsdBuf.Write([]byte{0, 0, 0, 0})
	_ = binary.Write(stsdBuf, binary.BigEndian, uint32(1))
	stsdBuf.Write(mp4aBox)
	stsdBox := CreateBox(BoxTypeSTSD, stsdBuf.Bytes())

	// Time to sample
	sttsBuf := new(bytes.Buffer)
	sttsBuf.Write([]byte{0, 0, 0, 0})
	_ = binary.Write(sttsBuf, binary.BigEndian, uint32(1))
	_ = binary.Write(sttsBuf, binary.BigEndian, uint32(numFrames))
	_ = binary.Write(sttsBuf, binary.BigEndian, uint32(frameSamples))
	sttsBox := CreateBox(BoxTypeSTTS, sttsBuf.Bytes())

	// Sample to chunk: 1 chunk per frame
	stscBuf := new(bytes.Buffer)
	stscBuf.Write([]byte{0, 0, 0, 0})
	_ = binary.Write(stscBuf, binary.BigEndian, uint32(1))
	_ = binary.Write(stscBuf, binary.BigEndian, uint32(1)) // first chunk
	_ = binary.Write(stscBuf, binary.BigEndian, uint32(1)) // samples per chunk
	_ = binary.Write(stscBuf, binary.BigEndian, uint32(1)) // sample desc index
	stscBox := CreateBox(BoxTypeSTSC, stscBuf.Bytes())

	// Sample sizes
	stszBuf := new(bytes.Buffer)
	stszBuf.Write([]byte{0, 0, 0, 0})
	_ = binary.Write(stszBuf, binary.BigEndian, uint32(frameByteSize)) // uniform size
	_ = binary.Write(stszBuf, binary.BigEndian, uint32(numFrames))
	stszBox := CreateBox(BoxTypeSTSZ, stszBuf.Bytes())

	// Preliminary layout stco box to measure exact moov length
	layoutStcoBuf := new(bytes.Buffer)
	layoutStcoBuf.Write([]byte{0, 0, 0, 0})
	_ = binary.Write(layoutStcoBuf, binary.BigEndian, uint32(numFrames))
	for i := 0; i < numFrames; i++ {
		_ = binary.Write(layoutStcoBuf, binary.BigEndian, uint32(0))
	}
	layoutStcoBox := CreateBox(BoxTypeSTCO, layoutStcoBuf.Bytes())

	stblBuf := new(bytes.Buffer)
	stblBuf.Write(stsdBox)
	stblBuf.Write(sttsBox)
	stblBuf.Write(stscBox)
	stblBuf.Write(stszBox)
	stblBuf.Write(layoutStcoBox)
	stblBox := CreateBox(BoxTypeSTBL, stblBuf.Bytes())

	minfBuf := new(bytes.Buffer)
	minfBuf.Write(smhdBox)
	minfBuf.Write(dinfBox)
	minfBuf.Write(stblBox)
	minfBox := CreateBox(BoxTypeMINF, minfBuf.Bytes())

	mdiaBuf := new(bytes.Buffer)
	mdiaBuf.Write(mdhdBox)
	mdiaBuf.Write(hdlrBox)
	mdiaBuf.Write(minfBox)
	mdiaBox := CreateBox(BoxTypeMDIA, mdiaBuf.Bytes())

	trakBuf := new(bytes.Buffer)
	trakBuf.Write(tkhdBox)
	trakBuf.Write(mdiaBox)
	trakBox := CreateBox(BoxTypeTRAK, trakBuf.Bytes())

	moovBuf := new(bytes.Buffer)
	moovBuf.Write(mvhdBox)
	moovBuf.Write(trakBox)
	prelimMoovBox := CreateBox(BoxTypeMOOV, moovBuf.Bytes())

	// Exact mdat start calculation
	mdatOffset := len(ftypBox) + len(prelimMoovBox)
	mediaDataStart := uint32(mdatOffset + 8) // +8 for mdat header

	// Now build real stco box with exact offsets
	realStcoBuf := new(bytes.Buffer)
	realStcoBuf.Write([]byte{0, 0, 0, 0})
	_ = binary.Write(realStcoBuf, binary.BigEndian, uint32(numFrames))
	for i := 0; i < numFrames; i++ {
		chunkOffset := mediaDataStart + uint32(i*frameByteSize)
		_ = binary.Write(realStcoBuf, binary.BigEndian, chunkOffset)
	}
	realStcoBox := CreateBox(BoxTypeSTCO, realStcoBuf.Bytes())

	// Reassemble stbl and moov with real stco
	stblBuf.Reset()
	stblBuf.Write(stsdBox)
	stblBuf.Write(sttsBox)
	stblBuf.Write(stscBox)
	stblBuf.Write(stszBox)
	stblBuf.Write(realStcoBox)
	stblBox = CreateBox(BoxTypeSTBL, stblBuf.Bytes())

	minfBuf.Reset()
	minfBuf.Write(smhdBox)
	minfBuf.Write(dinfBox)
	minfBuf.Write(stblBox)
	minfBox = CreateBox(BoxTypeMINF, minfBuf.Bytes())

	mdiaBuf.Reset()
	mdiaBuf.Write(mdhdBox)
	mdiaBuf.Write(hdlrBox)
	mdiaBuf.Write(minfBox)
	mdiaBox = CreateBox(BoxTypeMDIA, mdiaBuf.Bytes())

	trakBuf.Reset()
	trakBuf.Write(tkhdBox)
	trakBuf.Write(mdiaBox)
	trakBox = CreateBox(BoxTypeTRAK, trakBuf.Bytes())

	moovBuf.Reset()
	moovBuf.Write(mvhdBox)
	moovBuf.Write(trakBox)
	realMoovBox := CreateBox(BoxTypeMOOV, moovBuf.Bytes())

	mdatBox := CreateBox(BoxTypeMDAT, mediaPayload)

	finalMP4 := make([]byte, 0, len(ftypBox)+len(realMoovBox)+len(mdatBox))
	finalMP4 = append(finalMP4, ftypBox...)
	finalMP4 = append(finalMP4, realMoovBox...)
	finalMP4 = append(finalMP4, mdatBox...)

	return finalMP4, nil
}
